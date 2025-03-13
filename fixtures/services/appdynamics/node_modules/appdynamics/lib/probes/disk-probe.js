/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var BYTES_TO_KB = 1 / 1024;

function DiskProbe(agent) {
  this.agent = agent;
  this.packages = ['fs'];
}

exports.DiskProbe = DiskProbe;


DiskProbe.prototype.attach = function(obj) {
  var self = this;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  self.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
      proxy.release(obj.open);
      proxy.release(obj.close);
      proxy.release(obj.read);
      proxy.release(obj.write);
      proxy.release(obj.openSync);
      proxy.release(obj.closeSync);
      proxy.release(obj.readSync);
      proxy.release(obj.writeSync);
    }
  });

  var proxy = self.agent.proxy;
  var fileFDs = {};
  var fs = obj;


  self.reads = self.agent.metricsManager.createMetric(self.agent.metricsManager.DISK_IO_READ);
  self.writes = self.agent.metricsManager.createMetric(self.agent.metricsManager.DISK_IO_WRITE);

  // on open, determine if the returned fd is a regular file and
  // enable read/write tracking for that fd if it is
  proxy.before(obj, 'open', function(obj, args) {
    proxy.callback(args, -1, function(obj, args) {
      var err = args[0];
      var fd = args[1];

      if (err) return; // open failed; nothing to track
      fs.fstat(fd, function(er, st) {
        if (er || st && st.rdev !== 0) return; // not a regular file
        fileFDs[fd] = true;
      });
    });
  });

  // on close, turn off read/write tracking for the closed fd
  proxy.before(obj, 'close', function(obj, args) {
    var fd = args[0];
    if (!fileFDs[fd]) return;
    delete fileFDs[fd];
  });

  // on read, capture the number of bytes actually read
  proxy.before(obj, 'read', function(obj, args) {
    var fd = args[0];
    if (!fileFDs[fd]) return;
    proxy.callback(args, -1, function(obj, args) {
      var count = args[1] || 0;
      self.reads.addValue(count * BYTES_TO_KB);
    });
  });

  // on write, capture the number of bytes actually written
  proxy.before(obj, 'write', function(obj, args) {
    var fd = args[0];
    if (!fileFDs[fd]) return;
    proxy.callback(args, -1, function(obj, args) {
      var count = args[1] || 0;
      self.writes.addValue(count * BYTES_TO_KB);
    });
  });


  proxy.after(obj, 'openSync', function(obj, args, ret) {
    var fd = ret;
    var stats = fs.fstatSync(fd);
    if (stats.rdev !== 0) return;
    fileFDs[fd] = true;
  });

  proxy.after(obj, 'closeSync', function(obj, args, ret) {
    var fd = ret;
    if (!fileFDs[fd]) return;
    delete fileFDs[fd];
  });

  proxy.after(obj, 'readSync', function(obj, args, ret) {
    var fd = args[0];
    if (!fileFDs[fd]) return;
    var count = ret;
    self.reads.addValue(count * BYTES_TO_KB);
  });

  proxy.after(obj, 'writeSync', function(obj, args, ret) {
    var fd = args[0];
    if (!fileFDs[fd]) return;
    var count = ret;
    self.writes.addValue(count * BYTES_TO_KB);
  });
};
