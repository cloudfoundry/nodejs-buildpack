/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';


function ProcessProbe(agent) {
  this.agent = agent;
}
exports.ProcessProbe = ProcessProbe;



ProcessProbe.prototype.attach = function(obj) {
  var proxy = this.agent.proxy;
  var thread = this.agent.thread;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  this.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
      proxy.release(obj.nextTick);
    }
  });

  // we need nextThick for thread simulation
  proxy.before(obj, ['nextTick'], function(obj, args) {
    proxy.callback(args, 0, null, null, thread.current());
  }, false, false, thread.current());
};
