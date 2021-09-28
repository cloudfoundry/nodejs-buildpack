/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var uuid = require('uuid');
var CorrelationHeader = require('../transactions/correlation-header').CorrelationHeader;


function TransactionReporter(agent) {
  this.agent = agent;
  this.enabled = undefined;
  this.uuidInstance = undefined;
  this.nextRequestId = undefined;
  this.markTransactionAsError = undefined;
}
exports.TransactionReporter = TransactionReporter;


TransactionReporter.prototype.init = function() {
  var self = this;

  self.enabled = false;

  self.uuidInstance = uuid.v4();
  self.nextRequestId = 1;

  var configManager = self.agent.configManager;
  var registry = self.agent.transactionRegistry;
  var naming = self.agent.transactionNaming;
  var rules = self.agent.transactionRules;


  self.agent.on('configUpdated', function() {
    var txConfig = configManager.getConfigValue('txConfig');
    var enabled = configManager.getConfigValue('txConfig.nodejsWeb.enabled');
    if(txConfig && enabled !== undefined) {
      self.enabled = enabled;
    }
    self.markTransactionAsError = self.agent.configManager.getConfigValue('errorConfig.errorDetection.markTransactionAsError');
  });


  self.agent.on('transactionStarted', function(transaction, req) {
    var corrHeader, isApiTransaction,
      sepRules = self.agent.sepRules;

    if (!self.enabled) {
      transaction.ignore = true;
      transaction.emit('transactionIgnored');
      return;
    }

    transaction.guid = self.uuidInstance + (self.nextRequestId++);
    transaction.skewAdjustedStartWallTime =
      Date.now() + (configManager.getConfigValue('timestampSkew') | 0);

    if (transaction.entryType == 'NODEJS_API') {
      transaction.entryType = 'NODEJS_WEB';
      transaction.isAutoDiscovered = false;
      isApiTransaction = true;
    }

    if (typeof(req) === 'string') {
      transaction.name = req;
      req = undefined;
    } else if (req instanceof CorrelationHeader) {
      corrHeader = req;
      req = null;
    } else if (req && req.headers && req.headers[self.agent.correlation.HEADER_NAME]) {
      corrHeader = self.agent.correlation.newCorrelationHeader();
      if (!corrHeader.parse(req.headers[self.agent.correlation.HEADER_NAME])) {

        if (corrHeader.crossAppCorrelation) {
          var incomingCrossAppGUID = corrHeader.getSubHeader(self.agent.correlation.REQUEST_GUID);
          if (incomingCrossAppGUID) {
            transaction.incomingCrossAppSnapshotEnabled =
              corrHeader.getSubHeader(self.agent.correlation.SNAPSHOT_ENABLE);
            transaction.incomingCrossAppGUID = incomingCrossAppGUID;
          }
          if (corrHeader.crossAppCorrelationBackendId !== undefined) {
            self.agent.logger.debug("attaching correlation header to transaction for cross app correlation!");
            transaction.corrHeader = corrHeader;
          }
        }

        if (corrHeader.txDetect === false) {
          transaction.ignore = true;
          // even though it's an ignored tx, still put the corrHeader on the tx
          // so that exit calls also pass on the notxdetect
          transaction.corrHeader = corrHeader;
          transaction.emit('transactionIgnored');
          return;
        }

        corrHeader = null;
      }
    }

    if (corrHeader) {
      transaction.corrHeader = corrHeader;
      corrHeader.makeContinuingTransaction(transaction);
    } else {
      if (req && !isApiTransaction && !rules.accept(req, transaction)) {
        transaction.ignore = true;
        transaction.emit('transactionIgnored');
        return;
      }

      if (!transaction.name && req) {
        transaction.name = naming.createHttpTransactionName(req, transaction);
      }

      self.agent.logger.debug('transaction name: ' + transaction.name);

      if (!transaction.name) {
        transaction.ignore = true;
        self.agent.logger.debug("cannot create transaction name");
        transaction.emit('transactionIgnored');
        return;
      }

      registry.matchTransaction(transaction);
    }

    if (registry.isExcludedTransaction(transaction)) {
      transaction.ignore = true;
      transaction.emit('transactionIgnored');
      return;
    }

    if (req) {
      self.agent.dataCollectors.collectHttpData(transaction, req);
    }

    var sepExcludeRuleName = sepRules.isExcludeSepRuleMatch(req);
    if (sepExcludeRuleName) {
      transaction.sepRuleName = {
        ruleName: sepExcludeRuleName,
        isExcluded: true
      };
    } else {
      var sepIncludeRuleName = sepRules.isIncludedSepRuleMatch(req);
      if (sepIncludeRuleName) {
        transaction.sepRuleName = {
          ruleName: sepIncludeRuleName,
          isExcluded: false
        };
      }
    }
    self.agent.backendConnector.proxyTransport.sendBTInfoRequest(transaction);
  });


  self.agent.on('transaction', function(transaction) {
    if(!self.enabled || transaction.ignore) return;

    if(transaction.exitCalls) {
      transaction.exitCalls.forEach(function(exitCall) {
        registry.matchBackendCall(exitCall);
      });
    }

    self.agent.backendConnector.proxyTransport.sendTransactionDetails(transaction);
  });
};
