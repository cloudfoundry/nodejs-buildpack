/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var backendInfoToString = require('../transactions/exit-call').backendInfoToString;

function TransactionRegistry(agent) {
  this.agent = agent;

  this.registeredBTIndex = undefined;
  this.excludedBTIndex = undefined;
  this.registeredBackendIndex = undefined;
  this.resolvedBackendIds = undefined;
}
exports.TransactionRegistry = TransactionRegistry;


TransactionRegistry.prototype.init = function() {
  var self = this;

  self.registeredBTIndex = {};
  self.excludedBTIndex = {};
  self.registeredBackendIndex = {};
  self.resolvedBackendIds = {};

  self.agent.on('configUpdated', function() {
    self.updateRegisteredBTIndex();
    self.updateExcludedBTIndex();
    self.updateRegisteredBackendIndex();
  });
};

TransactionRegistry.prototype.updateRegisteredBTIndex = function() {
  var self = this;

  self.registeredBTIndex = {};

  var config = self.agent.configManager.getConfig();

  if(!config.txInfo || !config.txInfo.registeredBTs) return;
  var registeredBTs = config.txInfo.registeredBTs;

  registeredBTs.forEach(function(registeredBT) {
    if(!registeredBT.btInfo || !registeredBT.btInfo.internalName || !registeredBT.id) return;

    self.registeredBTIndex[registeredBT.btInfo.internalName.toString()] = registeredBT;
  });
};


TransactionRegistry.prototype.updateExcludedBTIndex = function() {
  var self = this;

  self.excludedBTIndex = {};

  var config = self.agent.configManager.getConfig();

  if(!config.txInfo || !config.txInfo.blackListedAndExcludedBTs) return;
  var blackListedAndExcludedBTs = config.txInfo.blackListedAndExcludedBTs;

  blackListedAndExcludedBTs.forEach(function(excludedBT) {
    if(!excludedBT.internalName) return;

    self.excludedBTIndex[excludedBT.internalName.toString()] = excludedBT;
  });
};

TransactionRegistry.prototype.updateRegisteredBackendIndex = function() {
  var self = this;

  self.registeredBackendIndex = {};

  var config = self.agent.configManager.getConfig();

  if(!config.bckInfo || !config.bckInfo.registeredBackends) return;
  var registeredBackends = config.bckInfo.registeredBackends;

  registeredBackends.forEach(function(entry) {
    if(
        !entry.registeredBackend.exitPointType ||
        !entry.registeredBackend.backendID ||
        !entry.exitCallInfo.exitPointType ||
        !entry.exitCallInfo.identifyingProperties)
    {
      return;
    }

    var identifyingPropertiesMap = {};
    entry.exitCallInfo.identifyingProperties.forEach(function(prop) {
      if(prop.value == null) return;
      identifyingPropertiesMap[prop.name] = prop.value;
    });

    var key =
      backendInfoToString(entry.exitCallInfo.exitPointType, entry.exitCallInfo.exitPointSubtype, identifyingPropertiesMap);
    self.agent.logger.debug("registered backendInfo: " + key);
    self.registeredBackendIndex[key] = entry;

    if(entry.registeredBackend.componentID) {
      self.resolvedBackendIds[entry.registeredBackend.backendID] = entry.registeredBackend.componentID;
    }
  });
};


TransactionRegistry.prototype.isExcludedTransaction = function(transaction) {
  var self = this;

  if(self.excludedBTIndex[transaction.name]) {
    return true;
  }

  return false;
};


TransactionRegistry.prototype.matchTransaction = function(transaction) {
  var self = this;

  var registeredBT = self.registeredBTIndex[transaction.name];
  if(registeredBT) {
    transaction.registrationId = registeredBT.id;
  }
  else {
    transaction.registrationId = undefined;
  }

  if (transaction.isAutoDiscovered === undefined) {
    transaction.isAutoDiscovered = true;
  }
};



TransactionRegistry.prototype.matchBackendCall = function(exitCall) {
  var self = this;

  var exitCallId =
    backendInfoToString(exitCall.exitType, exitCall.exitSubType, exitCall.identifyingProperties);
  self.agent.logger.debug("detected backendInfo: " + exitCallId);
  var registeredBackendEntry = self.registeredBackendIndex[exitCallId];
  if(registeredBackendEntry) {
    var registeredBackend = registeredBackendEntry.registeredBackend;
    exitCall.registrationId = registeredBackend.backendID;
    exitCall.componentId = registeredBackend.componentID;
    exitCall.componentIsForeignAppId = !!registeredBackend.componentIsForeignAppID;

  }
  else {
    exitCall.registrationId = undefined;
    exitCall.componentId = undefined;
  }
};
