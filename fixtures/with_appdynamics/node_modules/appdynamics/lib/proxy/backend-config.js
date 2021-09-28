/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var ExpressionEvaluator = require('./expression-evaluator').ExpressionEvaluator;

function BackendConfig(agent) {
  this.agent = agent;
  this.dbBackendConfig = undefined;
  this.cacheBackendConfig = undefined;
  this.httpBackendConfig = undefined;
  this.mongodbBackendConfig = undefined;
  this.awsBackendConfig = undefined;
  this.couchBaseConfig = undefined;
  this.cassandraConfig = undefined;
  this.orderedDbProperties = ['HOST', 'PORT', 'DATABASE', 'VENDOR'];
  this.orderedCacheProperties = ['SERVER POOL', 'VENDOR'];
  this.orderedMongodbProperties = ['HOST', 'PORT', 'DATABASE'];
  this.orderedHttpProperties = ['HOST', 'PORT', 'URL', 'QUERY STRING'];
  this.expressionEvaluator = new ExpressionEvaluator(agent);
}
// The Java agent trims the host names of http/db hosts to 50 characters and URL
// paths to 100, so this behavior should be matched
exports.BackendConfig = BackendConfig;

BackendConfig.MAX_PROPERTY_LENGTHS = {
  DATABASE: 50,
  HOST: 50,
  URL: 100,
  VENDOR: 50
};

BackendConfig.CONFIG_NAME = 'Configuration Name';

BackendConfig.prototype.init = function() {
  var self = this;

  self.expressionEvaluator.init();

  self.agent.on('configUpdated', function() {
    self.dbBackendConfig = self.agent.configManager.getConfigValue('bckConfig.nodejsDefinition.dbBackendConfig');
    self.cacheBackendConfig = self.agent.configManager.getConfigValue('bckConfig.nodejsDefinition.cacheBackendConfig');
    self.httpBackendConfig = self.agent.configManager.getConfigValue('bckConfig.nodejsDefinition.httpBackendConfig');
    self.mongodbBackendConfig = self.agent.configManager.getConfigValue('bckConfig.nodejsDefinition.mongodbBackendConfig');
    self.awsBackendConfig = self.agent.configManager.getConfigValue('bckConfig.nodejsDefinition.awsBackendConfig');
    self.couchBaseConfig = self.agent.configManager.getConfigValue('bckConfig.nodejsDefinition.couchBaseBackendConfig');
    self.cassandraConfig = self.agent.configManager.getConfigValue('bckConfig.nodejsDefinition.cassandraBackendConfig');

    if (self.dbBackendConfig) {
      self.dbBackendConfig.forEach(function(config) {
        setCustomConfigFlag(config, 'dbMatchRule');
      });
    }

    if (self.cacheBackendConfig) {
      self.cacheBackendConfig.forEach(function(config) {
        setCustomConfigFlag(config, 'cacheMatchRule');
      });
    }

    if (self.mongodbBackendConfig) {
      self.mongodbBackendConfig.forEach(function(config) {
        setCustomConfigFlag(config, 'mongdbMatchRule');
      });
    }

    if (self.awsBackendConfig) {
      self.awsBackendConfig.forEach(function(config) {
        setCustomConfigFlag(config, 'amazonwebserviceMatchRule');
      });
    }

    if (self.couchBaseConfig) {
      self.couchBaseConfig.forEach(function(config) {
        setCustomConfigFlag(config, 'couchbasedbMatchRule');
      });
    }

    if (self.cassandraConfig) {
      self.cassandraConfig.forEach(function(config) {
        setCustomConfigFlag(config, 'cassandradbMatchRule');
      });
    }

    self.parsedUrlRequired = false;
    if (self.httpBackendConfig) {
      self.httpBackendConfig.forEach(function(config) {
        setCustomConfigFlag(config, 'httpMatchRule');

        if (!self.parsedUrlRequired) {
          if (config.matchRule && config.matchRule.httpMatchRule) {
            var matchRule = config.matchRule.httpMatchRule;
            if (matchRule.url || matchRule.queryString) {
              self.parsedUrlRequired = true;
            }
          }
          if (config.namingRule && config.namingRule.httpNamingRule) {
            var namingRule = config.namingRule.httpNamingRule;
            if (namingRule.url || namingRule.queryString) {
              self.parsedUrlRequired = true;
            }
          }
        }
      });
    }
  });
};

function setCustomConfigFlag(config, ruleType) {
  var customConfig = false;
  if (config.matchRule && config.matchRule[ruleType]) {
    customConfig = true;
  }
  config.customConfig = customConfig;
}

BackendConfig.prototype.isParsedUrlRequired = function() {
  return this.parsedUrlRequired;
};

BackendConfig.prototype.getDbConfig = function(supportedProperties) {
  if (!this.dbBackendConfig) {
    return undefined;
  }

  var stringMatcher = this.agent.stringMatcher;
  return getConfig(stringMatcher, this.dbBackendConfig,
                   'dbMatchRule', supportedProperties);
};

BackendConfig.prototype.getCacheConfig = function(supportedProperties) {
  if (!this.cacheBackendConfig) {
    return undefined;
  }

  var stringMatcher = this.agent.stringMatcher;
  return getConfig(stringMatcher, this.cacheBackendConfig,
                   'cacheMatchRule', supportedProperties);
};

BackendConfig.prototype.getMongodbConfig = function(supportedProperties) {
  if (!this.mongodbBackendConfig) {
    return undefined;
  }

  var stringMatcher = this.agent.stringMatcher;
  return getConfig(stringMatcher, this.mongodbBackendConfig,
                   'mongodbMatchRule', supportedProperties);
};

BackendConfig.prototype.getAwsConfig = function(supportedProperties) {
  if (!this.awsBackendConfig) {
    return undefined;
  }

  var stringMatcher = this.agent.stringMatcher;
  return getConfig(stringMatcher, this.awsBackendConfig,
                   'amazonwebserviceMatchRule', supportedProperties);
};

BackendConfig.prototype.getCouchBaseConfig = function(supportedProperties) {
  if (!this.couchBaseConfig) return undefined;

  var stringMatcher = this.agent.stringMatcher;
  return getConfig(stringMatcher, this.couchBaseConfig,
                   'couchbasedbMatchRule', supportedProperties);
};

BackendConfig.prototype.getCassandraConfig = function(supportedProperties) {
  if (!this.cassandraConfig) return undefined;

  var stringMatcher = this.agent.stringMatcher;
  return getConfig(stringMatcher, this.cassandraConfig,
                   'cassandradbMatchRule', supportedProperties);
};

BackendConfig.prototype.getHttpConfig = function(supportedProperties) {
  if (!this.httpBackendConfig) {
    return undefined;
  }

  var stringMatcher = this.agent.stringMatcher;
  return getConfig(stringMatcher, this.httpBackendConfig,
                   'httpMatchRule', supportedProperties);
};

BackendConfig.prototype.populateDbProperties = function(backendConfig,
                                                        supportedProperties) {
  return getIdentifyingProperties(this.expressionEvaluator,
                                  backendConfig.namingRule, 'dbNamingRule',
                                  supportedProperties,
                                  backendConfig.name);
};

BackendConfig.prototype.populateCacheProperties = function(backendConfig,
                                                           supportedProperties) {
  return getIdentifyingProperties(this.expressionEvaluator,
                                  backendConfig.namingRule, 'cacheNamingRule',
                                  supportedProperties,
                                  backendConfig.name);
};

BackendConfig.prototype.populateMongodbProperties = function(backendConfig,
                                                             supportedProperties) {
  return getIdentifyingProperties(this.expressionEvaluator,
                                  backendConfig.namingRule, 'mongodbNamingRule',
                                  supportedProperties,
                                  backendConfig.name);
};

BackendConfig.prototype.populateAwsProperties = function(backendConfig,
                                                             supportedProperties) {
  return getIdentifyingProperties(this.agent.expressionEvaluator,
                                  backendConfig.namingRule, 'amazonwebserviceNamingRule',
                                  supportedProperties,
                                  backendConfig.name);
};

BackendConfig.prototype.populateCouchBaseProperties = function(backendConfig,
                                                                supportedProperties) {
  return getIdentifyingProperties(this.agent.expressionEvaluator,
                                  backendConfig.namingRule, 'couchbasedbNamingRule',
                                  supportedProperties,
                                  backendConfig.name);
};

BackendConfig.prototype.populateCassandraProperties = function(backendConfig,
                                                                supportedProperties) {
  return getIdentifyingProperties(this.agent.expressionEvaluator,
                                  backendConfig.namingRule, 'cassandradbNamingRule',
                                  supportedProperties,
                                  backendConfig.name);
};

BackendConfig.prototype.populateHttpProperties = function(backendConfig,
                                                          supportedProperties) {
  return getIdentifyingProperties(this.expressionEvaluator,
                                  backendConfig.namingRule, 'httpNamingRule',
                                  supportedProperties,
                                  backendConfig.name);
};

BackendConfig.prototype.generateDisplayName = function(exitCall) {
  var config = exitCall.backendConfig;

  var defaultSeparator = ' - ';
  var displayName = '';
  var prefix = '';
  // config can be undefined when using the node API
  if (config && config.customConfig) {
    prefix = config.name;
  }

  var orderedProperties;
  switch (exitCall.exitType) {
  case 'EXIT_DB':
    orderedProperties = this.orderedDbProperties;
    break;
  case 'EXIT_CACHE':
    orderedProperties = this.orderedCacheProperties;
    break;
  case 'EXIT_HTTP':
    orderedProperties = this.orderedHttpProperties;
    break;
  case 'EXIT_MONGODB':
    orderedProperties = this.orderedMongodbProperties;
    break;
  default:
    // unknown types used by the API
    orderedProperties = [];
    for (var property in exitCall.identifyingProperties) {
      if (exitCall.identifyingProperties.hasOwnProperty(property)) {
        orderedProperties.push(property);
      }
    }
    break;
  }

  for (var i = 0, length = orderedProperties.length; i < length; ++i) {
    var propertyName = orderedProperties[i];
    var propertyValue = exitCall.identifyingProperties[propertyName];
    if (!propertyValue) {
      continue;
    }

    if (displayName.length > 0) {
      if (propertyName === 'QUERY STRING') {
        displayName += '?';
      }
      else if (propertyName === 'URL') {
        // url already has the leading /, so nothing to add
      }
      else if (propertyName === 'PORT' &&
               i > 0 && orderedProperties[i - 1] === 'HOST') {
        displayName += ':';
      }
      else {
        displayName += defaultSeparator;
      }
    }

    if (exitCall.protocol && propertyName === 'HOST') {
      displayName += exitCall.protocol + '://';
    }

    // server pool values are a \n separated list, so just us the last value
    // rather than using a potentially big list
    if (propertyName === 'SERVER POOL') {
      var servers = propertyValue.split('\n');
      propertyValue = servers[servers.length - 1];
    }

    displayName += propertyValue;
  }

  if (prefix.length > 0 && displayName.length > 0) {
    prefix += defaultSeparator;
  }

  return prefix + displayName;
};

function getConfig(stringMatcher, backendConfigs, matchType, supportedProperties) {
  for (var i = 0, length = backendConfigs.length; i < length; ++i) {
    var config = backendConfigs[i];
    if (config.matchRule && config.matchRule[matchType]) {
      var matchRule = config.matchRule[matchType];

      if (matchBackend(stringMatcher, matchRule, supportedProperties)) {
        return config;
      }
    }
    else if (!config.customConfig) {
      // OOTB config has no match rules
      return config;
    }
  }

  return undefined;
}

function matchBackend(stringMatcher, matchRules, supportedProperties) {
  for (var property in supportedProperties) {
    var input = supportedProperties[property];
    var matchRule;
    switch (property) {
    case 'DATABASE':
      matchRule = matchRules.database;
      input = normalizeProperty(input, BackendConfig.MAX_PROPERTY_LENGTHS[property]);
      break;
    case 'VENDOR':
      matchRule = matchRules.vendor;
      input = normalizeProperty(input, BackendConfig.MAX_PROPERTY_LENGTHS[property]);
      break;
    case 'SERVER POOL':
      matchRule = matchRules.serverPool;
      break;
    case 'HOST':
      matchRule = matchRules.host;
      input = normalizeProperty(input, BackendConfig.MAX_PROPERTY_LENGTHS[property]);
      break;
    case 'PORT':
      matchRule = matchRules.port;
      input = input.toString();
      break;
    case 'URL':
      matchRule = matchRules.url;
      input = normalizeProperty(input, BackendConfig.MAX_PROPERTY_LENGTHS[property]);
      break;
    case 'QUERY STRING':
      matchRule = matchRules.queryString;
      break;
    }

    if (!matchRule) {
      continue;
    }

    if (!stringMatcher.matchString(matchRule, input)) {
      return false;
    }
  }

  // safe to always return true, as it's not possible to have a custom backend
  // config with no match rules
  return true;
}

function getIdentifyingProperties(exprEval, namingRules, ruleType, supportedProperties, configName) {
  var properties = {};
  if (!(namingRules && namingRules[ruleType])) {
    fillInDefaultConfigName(properties, configName);
    return properties;
  }

  var namingRule = namingRules[ruleType];
  for (var property in supportedProperties) {
    var rule;
    switch (property) {
    case 'DATABASE':
      rule = namingRule.database;
      break;
    case 'VENDOR':
      rule = namingRule.vendor;
      break;
    case 'SERVER POOL':
      rule = namingRule.serverPool;
      break;
    case 'HOST':
      rule = namingRule.host;
      break;
    case 'PORT':
      rule = namingRule.port;
      break;
    case 'URL':
      rule = namingRule.url;
      break;
    case 'QUERY STRING':
      rule = namingRule.queryString;
      break;
    case 'SERVICE':
      rule = namingRule.service;
      break;
    case 'ENDPOINT':
      rule = namingRule.endpoint;
      break;
    case 'BUCKET NAME':
      rule = namingRule.bucket;
      break;
    case 'KEYSPACE':
      rule = namingRule.keyspace;
      break;
    case 'CLUSTER NAME':
      rule = namingRule.cluster;
      break;
    case 'DATA CENTER':
      rule = namingRule.dataCenter;
      break;
    case 'RACK':
      rule = namingRule.rack;
      break;
    }

    if (!rule) {
      continue;
    }

    var value = exprEval.evaluate(rule, supportedProperties);
    if (!value) {
      continue;
    }

    switch (property) {
    case 'DATABASE':
    case 'HOST':
    case 'URL':
    case 'VENDOR':
      value = normalizeProperty(value, BackendConfig.MAX_PROPERTY_LENGTHS[property]);
      break;
    case 'PORT':
      value = value.toString();
      break;
    }

    properties[property] = value && value.replace(';', '_');
  }

  if (Object.keys(properties).length === 0) {
    fillInDefaultConfigName(properties, configName);
  }

  return properties;
}

function fillInDefaultConfigName(properties, configName) {
  properties[BackendConfig.CONFIG_NAME] = configName;
}

function normalizeProperty(value, length) {
  if (!value || value.length <= length) {
    return value;
  }

  return value.substring(0, length);
}
