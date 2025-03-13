/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

function System(agent) {
  this.agent = agent;

  this.hasHrtime = true;
  this.timekit = undefined;
}
exports.System = System;


System.prototype.init = function() {
  var self = this;

  self.appdNative = self.agent.appdNative;

  // make sure hrtime is available
  self.hasHrtime = process.hasOwnProperty('hrtime');
};


System.prototype.hrtime = function() {
  if(this.appdNative) {
    return this.appdNative.time();
  } else
  /* istanbul ignore else */
  if(this.hasHrtime) { // can't unit-test final else  clause as monkey-patching away process.hrtime() causes breakage
    var ht = process.hrtime();
    return ht[0] * 1000000 + Math.round(ht[1] / 1000);
  }
  else {
    return Date.now() * 1000;
  }
};


System.prototype.micros = function() {
  return this.appdNative ? this.appdNative.time() : Date.now() * 1000;
};


System.prototype.millis = function() {
  return this.appdNative ? this.appdNative.time() / 1000 : Date.now();
};


System.prototype.cputime = function() {
  return this.appdNative ? this.appdNative.cpuTime() : undefined;
};

