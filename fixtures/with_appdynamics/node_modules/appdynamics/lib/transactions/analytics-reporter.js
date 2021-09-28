var http = require('http');
var https = require('https');

var PERIOD = 10 * 1000; // every 10 seconds

function AnalyticsReporter(agent) {
  this.agent = agent;
  this.enabled = false;
  this.analyticsHost = process.env.APPDYNAMICS_ANALYTICS_HOST_NAME || 'localhost';
  this.analyticsPort = parseInt(process.env.APPDYNAMICS_ANALYTICS_PORT, 10) || 9090;
  this.analyticsSSL = (process.env.APPDYNAMICS_ANALYTICS_SSL_ENABLED === 'true') || false;
}

exports.AnalyticsReporter = AnalyticsReporter;

AnalyticsReporter.prototype.init = function() {
  var config = this.agent.opts && this.agent.opts.analytics;
  this.analyticsHost = config && config.host || this.analyticsHost;
  this.analyticsPort = config && config.port || this.analyticsPort;
  this.analyticsSSL = (config && config.ssl) ? config.ssl : this.analyticsSSL;

  this.enabled = false;
  this.initialized = false;
  this.agent.on('configUpdated', this.configUpdated.bind(this));
  this.configUpdated();
};

AnalyticsReporter.prototype.initNative = function() {
  var self = this;
  var agentIdentity = this.agent.configManager.getConfig().agentIdentity;
  if (agentIdentity) {
    this.agent.logger.debug('Initializing Analytics');
    this.initialized = true;
    this.agent.appdNative.initAnalytics({
      'applicationName': this.agent.opts.applicationName,
      'tierName': this.agent.opts.tierName,
      'nodeName': this.agent.opts.nodeName,
      'applicationId': agentIdentity.appID,
      'tierId': agentIdentity.tierID,
      'nodeId': agentIdentity.nodeID
    }, function httpCallback(json) {
      var options = {
        appdIgnore: true,
        host: self.analyticsHost,
        port: self.analyticsPort,
        path: '/v1/sinks/bt',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      };
      var client = (self.analyticsSSL) ? https : http;
      var req = client.request(options, function (response) {
        self.agent.logger.debug('analytics response: '+response.statusCode);
      });

      req.on('error', function(err) {
        self.agent.logger.warn('analytics reporting error: ' + err.message);
      });

      if (self.agent.logger.isDebugEnabled()) {
        self.agent.logger.debug('sending analytics events:\n' + json);
      }
      req.write(json);
      req.end();
    });
  }
};

AnalyticsReporter.prototype.configUpdated = function() {
  this.agent.logger.debug('updating analytics config');
  var config = this.agent.configManager.getConfig();
  var analyticsConfig = config && config.analyticsConfig || {};

  var wasEnabled = this.enabled;
  this.enabled = !!analyticsConfig.isEnabled;
  this.enabledBTIDs = analyticsConfig.analyticsBTIDs || [];

  if (this.enabled) {
    if (!this.initialized) {
      this.initNative();
    }

    if (!wasEnabled) {
      // changed state from disabled to enabled; install monitors
      this.agent.appdNative.enableAnalytics(this.enabled);

      this._transactionHandler = this.addTransaction.bind(this);
      this.agent.on('transaction', this._transactionHandler);
      this._reportInterval = this.agent.timers.setInterval(
        this.reportAnalytics.bind(this), PERIOD);
    }
  } else if (wasEnabled) {
    // changed state from enabled to disabled; remove monitors
    this.agent.appdNative.enableAnalytics(this.enabled);

    if (this._transactionHandler) {
      this.agent.removeListener('transaction', this._transactionHandler);
    }
    this.agent.timers.clearInterval(this._reportInterval);
    this._transactionHandler = null;
    this._reportInterval = null;
  }
};

AnalyticsReporter.prototype.getTimestamp = function() {
  return (new Date()).toISOString();
};

AnalyticsReporter.prototype.addTransaction = function(txn) {
  if (!this.enabled) return;
  if (txn.ignore) return;
  if (!txn.registrationId) return;
  if (this.enabledBTIDs.indexOf(''+txn.registrationId) < 0) {
    return;
  }

  var httpData = txn.httpRequestAnalyticsData;
  this.agent.logger.debug('analytics transaction added: ' + txn.name);
  this.agent.appdNative.recordTransaction({
    "eventTimestamp": this.getTimestamp(),

    "requestGUID": txn.guid,
    "transactionId": txn.registrationId,
    "transactionName": txn.name,
    "transactionTime": txn.ms,
    "clientRequestGUID": txn.eumGuid,

    "requestExperience": getUserExperience(txn),
    "entryPoint": !txn.corrHeader,

    exitCalls: (txn.exitCalls || []).map(function(call) {
      var componentID = call.componentID;
      var backendID = call.backendIdentifier &&
           call.backendIdentifier.registeredBackend &&
           call.backendIdentifier.registeredBackend.backendID;
      return {
        exitCallType: call.exitType,
        avgResponseTimeMillis: call.ms,
        numberOfErrors: call.error ? 1 : 0,
        numberOfCalls: 1,
        toEntityId: componentID || backendID || null,
        toEntityType:
          componentID && (call.componentIsForeignAppID ? 'APPLICATION' : 'APPLICATION_COMPONENT') ||
          backendID && 'BACKEND' || null
      };
    }),

    "httpData": {
      "cookies": httpData && httpData.cookies,
      "headers": httpData && httpData.headers,
      "parameters": httpData && httpData.httpParams,
      "url": txn.url,

      // Not supported for Node:
      //
      // "principal": "No User Principal",
      // "sessionId": null,
      // "sessionObjects": {},
      // "uriPathSegments": {}
    },

    "userData": txn.api && txn.api.analyticsData
  });
};

AnalyticsReporter.prototype.reportAnalytics = function() {
  if (!this.enabled) return;
  this.agent.logger.debug('polling analytics reporter');
  this.agent.appdNative.reportTransactions();
};

AnalyticsReporter.prototype._getUserExperience = getUserExperience; // expose to unit tests
function getUserExperience(txn) {
  if (txn.hasErrors) {
    return "ERROR";
  }
  if (txn.btInfoResponse && txn.btInfoResponse.currentVerySlowThreshold > 0 &&
      txn.btInfoResponse.currentVerySlowThreshold < txn.ms) {
    return "VERY_SLOW";
  }
  if (txn.btInfoResponse && txn.btInfoResponse.currentSlowThreshold > 0 &&
      txn.btInfoResponse.currentSlowThreshold < txn.ms) {
    return "SLOW";
  }
  return "NORMAL";
}
