/*
 * Copyright (c) AppDynamics, Inc., and its affiliates
 * 2016
 * All Rights Reserved
 * THIS IS UNPUBLISHED PROPRIETARY CODE OF APPDYNAMICS, INC.
 * The copyright notice above does not evidence any actual or intended publication of such source code
 */

'use strict';
var txRules = require('./transaction-rules').TransactionRules;

function SepRules(agent) {
  this.agent = agent;
  this.sepIncludeRules = undefined;
  this.sepExcludeRules = undefined;
}

exports.SepRules = SepRules;

SepRules.prototype.init = function() {
  var self = this;

  self.sepRules = [];
  self.sepIncludeRules = [];
  self.sepExcludeRules = [];

  self.agent.on('configUpdated', function() {
    self.sepRules = self.agent.configManager.getConfigValue('sepConfig.customDefinitions');
    self.sepIncludeRules = [];
    self.sepExcludeRules = [];
    if (self.sepRules && self.sepRules.length) {
      self.sepRules.forEach(function (rule) {
        // An excluded rule has id 1, and included one has id 0
        // Is excluded sep rule
        if (rule.id === 1) {
          self.sepExcludeRules.push(rule);
        } else {
          // Is include sep rule
          self.sepIncludeRules.push(rule);
        }
      });

      self.sepIncludeRules = self.sepIncludeRules.sort(function(r1, r2) { return r2.priority - r1.priority; });
      self.sepExcludeRules = self.sepExcludeRules.sort(function(r1, r2) { return r2.priority - r1.priority; });
    }
  });
};

SepRules.prototype.isExcludeSepRuleMatch = function(req) {
  return reqRuleMatch(req, this.sepExcludeRules, this.agent.stringMatcher);
};

SepRules.prototype.isIncludedSepRuleMatch = function(req) {
  return reqRuleMatch(req, this.sepIncludeRules, this.agent.stringMatcher);
};

function reqRuleMatch(req, rules, stringMatcher) {
  var matchResult;
  if (!rules || !rules.length) { return; }
  for (var i = 0; i < rules.length; i++) {
    matchResult = txRules.matchesRule(req, rules[i].condition.http, stringMatcher);
    if (matchResult) {
      return rules[i].btName;
    }
  }
}