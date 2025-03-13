/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';


function GlobalProbe(agent) {
  this.agent = agent;
}
exports.GlobalProbe = GlobalProbe;



GlobalProbe.prototype.attach = function(obj) {
  var self = this;
  var proxy = self.agent.proxy;
  var thread = self.agent.thread;
  var fns = ['setTimeout', 'setInterval', 'setImmediate'];
  var timers = require('timers');
  var syncGlobalTimerFns = {};
  fns.forEach(function(functionName) {
    if (obj[functionName] == timers[functionName]) {
      syncGlobalTimerFns[functionName] = true;
    }
  });

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  self.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
      fns.forEach(function(fn) {
        proxy.release(obj[fn]);
      });
    }
  });

  // we need these for thread simulation
  proxy.before(obj, fns, function(obj, args) {
    proxy.callback(args, 0, null, null, thread.current());
  }, false, false, thread.current());

  // Many global methods memory reference the timers methods in v >= 4.*
  // When attaching a probe around these methods on global object, they get
  // out of sync with timers methods.
  // To make sure these methods from both the objects (global and timers)
  // reference same memory location they are synced again.
  (Object.keys(syncGlobalTimerFns)).forEach(function(functionName) {
    timers[functionName] = obj[functionName];
  });
};