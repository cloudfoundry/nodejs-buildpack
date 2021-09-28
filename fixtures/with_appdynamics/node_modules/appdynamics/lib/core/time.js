/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

/*
 * The Time class is used to calculate execution time and CPU time
 * of a call. It also emits call-start and call-done events.
 */

function Time(agent, isTransaction) {
  this.agent = agent;

  this.isTransaction = isTransaction;

  this.id = agent.getNextId();

  this._begin = undefined;
  this.begin = undefined;
  this.end = undefined;
  this.ms = undefined;

  this.threadId = undefined;
}
exports.Time = Time;


Time.prototype.start = function() {
  var self = this;

  var system = self.agent.system;
  var thread = self.agent.thread;

  self.begin = system.millis();
  self._begin = system.hrtime();

  // threads
  if(self.isTransaction) {
    self.threadId = thread.enter();
  }
  else {
    self.threadId = thread.current();
  }
};


Time.prototype.done = function() {
  var self = this;

  if(self.ms !== undefined) return false;

  var system = self.agent.system;
  var thread = self.agent.thread;

  self.ms = (system.hrtime() - self._begin) / 1000;
  self.end = self.begin + self.ms;

  // threads
  if(self.isTransaction) {
    thread.exit();
  }
  else {
    thread.resume(self.threadId);
  }

  return true;
};


