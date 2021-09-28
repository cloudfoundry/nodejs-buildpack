/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var HttpEntryProbe = require('./http-entry-probe').HttpEntryProbe;
var HttpExitProbe = require('./http-exit-probe').HttpExitProbe;

function HttpProbe(agent) {
  this.agent = agent;
  this.packages = ['http', 'https'];

  this.entryProbe = new HttpEntryProbe(agent);
  this.exitProbe = new HttpExitProbe(agent);

  this.init();
}
exports.HttpProbe = HttpProbe;

HttpProbe.prototype.init = function() {
  this.entryProbe.init();
  this.exitProbe.init();
};

HttpProbe.prototype.attach = function(obj, moduleName) {
  var self = this;
  var proxy = this.agent.proxy;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  self.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
      proxy.release(obj.Server.prototype.on);
      proxy.release(obj.Server.prototype.addListener);
      proxy.release(obj.request);
    }
  });

  this.entryProbe.attach(obj, moduleName);
  this.exitProbe.attach(obj, moduleName);
};
