/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var util = require('util');
var EventEmitter = require('events').EventEmitter;


function Transaction() {
  this.connection = undefined;
  this.command = undefined;
  this.commandArgs = undefined;
  this.stackTrace = undefined;
  this.error = undefined;
  this.url = undefined;
  this.method = undefined;
  this.requestHeaders = undefined;
  this.responseHeaders = undefined;
  this.statusCode = undefined;
  this.name = undefined;
  this.label = undefined;
  this.id = undefined;
  this.ms = undefined;
  this.ts = undefined;
  this.threadId = undefined;
  this.entryType = undefined;
  this.host = undefined;
  this.port = undefined;
  this.guid = undefined;
  this.startedExitCalls = undefined;
  this.exitCalls = undefined;
  this.btInfoRequest = undefined;
  this.btInfoResponse = undefined;
  this.registrationId = undefined;
  this.isAutoDiscovered = undefined;
  this.hasErrors = undefined;
  this.ignore = undefined;
  this.exitCallCounter = 0;
  this.corrHeader = undefined;
  this.namingSchemeType = undefined;
  this.eumGuid = undefined;
  this.processSnapshots = undefined;
  this.incomingCrossAppSnapshotEnabled = undefined;
  this.incomingCrossAppGUID = undefined;
  this.skewAdjustedStartWallTime = undefined;
  this.isResponseSent = undefined;
  this.isFinished = undefined;

  EventEmitter.call(this);
}

util.inherits(Transaction, EventEmitter);
exports.Transaction = Transaction;

Transaction.prototype.addProcessSnapshotGUID = function (processSnapshotGUID) {
  var self = this;
  if (!self.processSnapshots) {
    self.processSnapshots = { };
  }
  self.processSnapshots[processSnapshotGUID] = true;
};

Transaction.prototype.hasSnapshotWithCallGraphData = function () {
  var self = this;

  if (!self.btInfoResponse) {
    return false;
  }

  if (!self.btInfoResponse.isSnapshotRequired) {
    return false;
  }

  if (!self.processSnapshots) {
    return false;
  }

  return Object.keys(self.processSnapshots).length > 0;
};
