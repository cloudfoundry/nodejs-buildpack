/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var BYTES_TO_KB = 1 / 1024;

function NetProbe(agent) {
  this.agent = agent;

  this.packages = ['net'];
}
exports.NetProbe = NetProbe;



NetProbe.prototype.attach = function(obj) {
  var self = this;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  self.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
      proxy.release(obj.connect);
      proxy.release(obj.createConnection);
    }
  });

  var proxy = self.agent.proxy;

  var bytesWritten = 0;
  var bytesRead = 0;

  self.writes = self.agent.metricsManager.createMetric(self.agent.metricsManager.NETWORK_IO_WRITE);
  self.reads = self.agent.metricsManager.createMetric(self.agent.metricsManager.NETWORK_IO_READ);

  proxy.after(obj, ['connect', 'createConnection'], function(obj, args, ret) {
    var socket = ret;
    var lastBytesWritten = 0;
    var lastBytesRead = 0;
    var threadId = self.agent.thread.current();

    proxy.before(ret, ['write', 'end'], function() {
      var currentBytesWritten = socket.bytesWritten || 0;
      bytesWritten = currentBytesWritten - lastBytesWritten;
      lastBytesWritten = currentBytesWritten;
      self.writes.addValue(bytesWritten * BYTES_TO_KB);
    }, false, false, threadId);

    proxy.before(ret, 'on', function(obj, args) {
      if(args.length < 1 || args[0] !== 'data') return;

      proxy.callback(args, -1, function() {
        var currentBytesRead = socket.bytesRead || 0;
        bytesRead = currentBytesRead - lastBytesRead;
        lastBytesRead = currentBytesRead;
        self.reads.addValue(bytesRead * BYTES_TO_KB);
      }, null, threadId);
    }, false, false, threadId);
  }, false, self.agent.thread.current());
};

