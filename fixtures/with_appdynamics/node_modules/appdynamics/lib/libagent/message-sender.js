/*
* Copyright (c) AppDynamics, Inc., and its affiliates
* 2016
* All Rights Reserved
*/

// MessageSender is a simple wrapper around setInterval, just to
// handle a variable initial delay before starting a periodic job.

function MessageSender(agent, initialDelayInMs, timeoutInMs, body) {
  this.agent = agent;
  this.timeoutInMs = timeoutInMs;
  this.body = body;
  this.timerObj = undefined;

  var self = this;
  this.startupTimer = agent.timers.setTimeout(function() {
    self.initialize();
  }, initialDelayInMs);
}

exports.MessageSender = MessageSender;

MessageSender.prototype.initialize = function() {
  var self = this;

  self.body();
  self.timerObj = self.agent.timers.setInterval(function() {
    self.body();
  }, self.timeoutInMs);
};