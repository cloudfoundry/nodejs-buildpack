/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var uuid = require('uuid');
var txRules = require('./transaction-rules').TransactionRules;


function Eum(agent) {
  this.agent = agent;

  this.enabled = false;
  this.globalAccountName = null;
  this.excludeRules = null;
  this.includeRules = null;

  // GUID generation
  this.uuidBase = uuid.v4();
  this.nextEumCookieId = 1;

  // constants
  this.ADRUM_MASTER_COOKIE_NAME = 'ADRUM_BT';
  this.ADRUM_PREFIX = "ADRUM_";
  this.CLIENT_REQUEST_GUID_KEY = { long: 'clientRequestGUID', short: 'g' };
  this.BT_ID_KEY = { long: 'btId', short: 'i' };
  this.BT_ART_KEY = { long: 'btERT', short: 'e' };
  this.BT_DURATION_KEY = { long: 'btDuration', short: 'd' };
  this.SERVER_SNAPSHOT_TYPE_KEY = { long: 'serverSnapshotType', short: 's' };
  this.HAS_ENTRY_POINT_ERRORS_KEY = { long: 'hasEntryPointErrors', short: 'h' };
  this.GLOBAL_ACCOUNT_NAME_KEY = { long: 'globalAccountName', short: 'n' };

  this.eumCookie = null;
}
exports.Eum = Eum;

Eum.prototype.init = function() {
  var self = this;

  self.registerEumCookieType();
  self.agent.on('configUpdated', function() {
    self.enabled = self.agent.configManager.getConfigValue("eumConfig.enabled");
    self.globalAccountName = self.agent.configManager.getConfigValue("eumConfig.globalAccountName");
    self.excludeRules = self.agent.configManager.getConfigValue("eumConfig.excludeRules");
    self.includeRules = self.agent.configManager.getConfigValue("eumConfig.includeRules");
  });
};

Eum.prototype.registerEumCookieType = function() {
  var self = this;
  self.eumCookie = require('./eum-cookie').EumCookie;
};

Eum.prototype.generateGuid = function() {
  var self = this;

  return (self.uuidBase + (self.nextEumCookieId++));
};

Eum.prototype.newEumCookie = function(transaction, request, response, isHttps) {
  var self = this;

  return new self.eumCookie(self.agent, transaction, request, response, isHttps);
};

Eum.prototype.enabledForTransaction = function(req) {
  var self = this,
    rule, matchResult, counter;

  if(self.excludeRules && self.excludeRules.length) {
    for(counter = 0; counter < self.excludeRules.length; counter++) {
      rule = self.excludeRules[counter];
      matchResult = txRules.matchesRule(req, rule, self.agent.stringMatcher);
      if (matchResult) return false;
    }
  }

  if(self.includeRules && self.includeRules.length) {
    for(counter = 0; counter < self.includeRules.length; counter++) {
      rule = self.includeRules[counter];
      matchResult = txRules.matchesRule(req, rule, self.agent.stringMatcher);
      if (matchResult) return true;
    }
    return false;
  }

  return self.enabled;
};
