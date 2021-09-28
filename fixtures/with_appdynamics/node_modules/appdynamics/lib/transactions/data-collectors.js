/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
var parseCookies = require('./cookie-util').parseCookies,
  parseUrl = require('url').parse;

function DataCollectors(agent) {
  this.agent = agent;
  this.gatherersByID = [];
  this.btIDtoGathererIDs = [];
}

module.exports.DataCollectors = DataCollectors;

DataCollectors.prototype.init = function() {
  var self = this;

  self.agent.on('configUpdated', function() {
    var dataGatherers = self.agent.configManager.getConfigValue('dataGatherers');
    self.gatherersByID = !dataGatherers ? [] : dataGatherers.reduce(function(mapping, config) {
      mapping[config.gathererID] = config;
      return mapping;
    }, {});

    var btConfig = self.agent.configManager.getConfigValue('dataGathererBTConfig.btConfig');
    self.btIDtoGathererIDs = !btConfig ? [] : btConfig.reduce(function(mapping, config) {
      mapping[config.btID] = config.gathererIDs;
      return mapping;
    }, {});
  });
};

DataCollectors.prototype.collectHttpData = function(transaction, req) {
  var btID = transaction.registrationId,
    collectors = this.getDataCollectorsFor(btID),
    analyticsData = transaction.httpRequestAnalyticsData = { url: req.url && parseUrl(req.url).pathname },
    snapshotData = transaction.httpRequestSnapshotData = { url: req.url && parseUrl(req.url).pathname };

  function addData(config, type, name, value) {
    if (config.enabledForAnalytics) {
      analyticsData[type] = analyticsData[type] || [];
      analyticsData[type].push({ name: name, value: value });
    }
    if (config.enabledForApm) {
      snapshotData[type] = snapshotData[type] || [];
      snapshotData[type].push({ name: name, value: value });
    }
  }

  if (collectors) {
    collectors.forEach(function(collector) {
      var config;

      if (!collector) return;
      if (collector.type !== 'HTTP') return;

      config = collector.httpDataGathererConfig;
      if (!(config.enabledForApm || config.enabledForAnalytics)) return;

      if (config.cookieNames) {
        var cookies = parseCookies(req);
        config.cookieNames.forEach(function(name) {
          var value = cookies[name];
          if (value) addData(config, 'cookies', name, value);
        }, this);
      }

      if (config.headers) {
        config.headers.forEach(function(name) {
          var value = req.headers[name.toLowerCase()];
          if (value) addData(config, 'headers', name, value);
        }, this);
      }

      if (config.requestParams) {
        var url = parseUrl(req.url, true),
          params = url.query;
        config.requestParams.forEach(function(param) {
          var value = params[param.name];
          if (value) addData(config, 'httpParams', param.name, value);
        }, this);
      }
    }, this);
  }
};

DataCollectors.prototype.getDataCollectorsFor = function(btID) {
  var gathererIDs = this.btIDtoGathererIDs[btID];

  return gathererIDs && gathererIDs.map(function(id) {
    return this.gatherersByID[id];
  }, this);
};
