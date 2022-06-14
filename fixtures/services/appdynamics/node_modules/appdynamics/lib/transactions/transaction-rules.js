/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

// ------------------------------------------------------------------------

var cs = require('./cookie-util')
  ,   url = require('url');

// ------------------------------------------------------------------------

function TransactionRules(agent) {
  this.agent = agent;
  this.customMatches = undefined;
  this.customExcludes = undefined;
}

exports.TransactionRules = TransactionRules;

TransactionRules.prototype.init = function() {
  var self = this;

  self.customMatches = [];
  self.customExcludes = [];

  self.agent.on('configUpdated', function() {
    var matches = self.agent.configManager.getConfigValue('txConfig.nodejsWeb.customDefinitions')
      ,   excludes = self.agent.configManager.getConfigValue('txConfig.nodejsWeb.discoveryConfig.excludes')
      ,   discovery = self.agent.configManager.getConfigValue('txConfig.nodejsWeb.discoveryConfig.enabled');

    matches = matches || [];
    excludes = excludes || [];

    // keep matches in priority order
    matches.sort(function(r1, r2) { return r2.priority - r1.priority; });

    // format custom naming rules
    matches.map(function(customMatch) {
      var http = customMatch.condition.http;
      if (http && Array.isArray(http.properties)) {
        http.properties = self.agent.configManager.convertListToMap(http.properties);
      }
      return http;
    });

    // map any header names to lower case
    matches.forEach(headerRuleToLowerCase);
    excludes.forEach(headerRuleToLowerCase);

    // update matches / excludes
    self.customMatches = matches;
    self.customExcludes = excludes;
    self.discoveryEnabled = !!discovery;
  });
};

TransactionRules.prototype.accept = function(req, transaction) {
  if (this.match(req, transaction)) return true;
  if (this.discoveryEnabled) return !this.exclude(req);
  return false;
};

TransactionRules.prototype.match = function(req, transaction) {
  var self = this;

  if (self.customMatches && self.customMatches.length) {
    for (var i = 0; i < self.customMatches.length; i++) {
      if (TransactionRules.matchesRule(req, self.customMatches[i].condition.http, self.agent.stringMatcher)) {
        self.agent.logger.debug('transaction matched: ' + self.customMatches[i].btName);
        transaction.customNaming = self.customMatches[i].condition.http.properties;
        transaction.customMatch = self.customMatches[i];
        transaction.isAutoDiscovered = false;
        return true;
      }
    }
  }

  return false;
};

TransactionRules.prototype.exclude = function(req) {
  var self = this;

  for (var i = 0; i < self.customExcludes.length; i++) {
    if (TransactionRules.matchesRule(req, self.customExcludes[i].http, self.agent.stringMatcher)) {
      return true;
    }
  }

  return false;
};

TransactionRules.matchesRule = function(req, rule, stringMatcher) {
  var matches = true, uri, host, port, pair;

  uri = url.parse(req.url).pathname;
  pair = req && req.headers && req.headers.host && req.headers.host.split(':');
  port = pair && pair[1] || (req.connection.localPort && req.connection.localPort.toString());
  host = pair && pair[0] || 'localhost';

  if ('method' in rule) matches = matches && rule.method == req.method;
  if ('uri'    in rule) matches = matches && stringMatcher.matchString(rule.uri,  uri);
  if ('host'   in rule) matches = matches && stringMatcher.matchString(rule.host, host);
  if (port && 'port' in rule) matches = matches && stringMatcher.matchString(rule.port, port);

  matches = matches && matchGetPostParams(req, rule, stringMatcher);
  matches = matches && matchCookies(req, rule, stringMatcher);
  matches = matches && matchHeaders(req, rule, stringMatcher);

  return matches;
};

// ------------------------------------------------------------------------

function headerRuleToLowerCase(rule) {
  var headers = rule && (rule.headers || (rule.condition && rule.condition.http && rule.condition.http.headers));
  if (headers) headers.forEach(function(headerRule) {
    if (headerRule.key && headerRule.key.matchStrings) {
      headerRule.key.matchStrings = headerRule.key.matchStrings.map(function(header) {
        return header.toLowerCase();
      });
    }
  });
}

function matchGetPostParams(req, rule, stringMatcher) {
  var getParams = url.parse(req.url || '/', true).query,
    postParams = req.body || {}, // NOTE: requires Express
    paramRules = rule.params || [],
    paramRule;

  for (var i = 0; i < paramRules.length; i++) {
    paramRule = paramRules[i];
    if (!stringMatcher.matchKeyValue(paramRule, getParams) && !stringMatcher.matchKeyValue(paramRule, postParams)) {
      return false;
    }
  }

  return true;
}

function matchCookies(req, rule, stringMatcher) {
  var cookies = cs.parseCookies(req),
    cookieRules = rule.cookies || [],
    cookieRule;

  for (var i = 0; i < cookieRules.length; i++) {
    cookieRule = cookieRules[i];
    if (!stringMatcher.matchKeyValue(cookieRule, cookies)) {
      return false;
    }
  }

  return true;
}

function matchHeaders(req, rule, stringMatcher) {
  var headers = req.headers,
    headerRules = rule.headers || [],
    headerRule;

  for (var i = 0; i < headerRules.length; i++) {
    headerRule = headerRules[i];
    if (!stringMatcher.matchKeyValue(headerRule, headers)) {
      return false;
    }
  }

  return true;
}
