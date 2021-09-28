/*
* Copyright (c) AppDynamics, Inc., and its affiliates
* 2016
* All Rights Reserved
*/
var url = require('url');

function TransactionSender(agent) {
  this.agent = agent;
  this.btNamingConfig = undefined;
  this.isEnabled = false;
}
exports.TransactionSender = TransactionSender;


TransactionSender.prototype.init = function() {
  var self = this;
  var libagentConnector = self.agent.libagentConnector;


  libagentConnector.on("btNamingProperties", function(btNamingProperties) {
    self.isEnabled = true;

    self.agent.transactionNaming.namingProps = btNamingProperties;
  });


  self.agent.on("transactionStarted", function(transaction, req) {
    if (!self.isEnabled) {
      self.agent.logger.warn('transactionStarted sent without enabled sender');
      return;
    }
    var apiCall = transaction.entryType === 'NODEJS_API';
    var corrHeader = '';

    var name = '';

    var isHttpRequest = false;
    if (apiCall && typeof(req) === 'string') {
      // req is the transaction name
      name = transaction.name = req;
    }
    else {
      if (typeof(req) === 'object' && req.headers) {
        isHttpRequest = true;
        if('singularityheader' in req.headers) {
          corrHeader = req.headers.singularityheader || "";
        }
      }
      if (req.businessTransactionName) {
        name = transaction.name = req.businessTransactionName;
      }
    }

    var txData = libagentConnector.startBusinessTransaction('NODEJS_WEB', name, corrHeader, self.createBtNamingWrapper(req), isHttpRequest);

    if (txData === undefined || txData.isExcluded) {
      transaction.skip = true;
      return;
    }

    transaction.skip = false;
    transaction.name = txData.name;
    transaction.btGuid = txData.btGuid;
    transaction.guid = txData.guid;
    transaction.eumEnabled = txData.eumEnabled;
    transaction.isHttpRequest = isHttpRequest;

    if (libagentConnector.isSnapshotRequired(transaction)) {
      libagentConnector.emit("autoProcessSnapshot");
    }
  });

  self.agent.on("transaction", function(transaction) {
    if (!self.isEnabled || transaction.skip) {
      return;
    }

    if (libagentConnector.isSnapshotRequired(transaction)) {
      libagentConnector.emit("autoProcessSnapshot");
      var snapshot = libagentConnector.protobufModel.createSnapshot(transaction);
      libagentConnector.sendTransactionSnapshot(transaction, snapshot);
      if (transaction.isHttpRequest) {
        libagentConnector.setHttpParamsInTransactionSnapshot(transaction);
        libagentConnector.addHttpDataToTransactionSnapshot(transaction, transaction.httpRequestData);
      }
    }

    libagentConnector.stopBusinessTransaction(transaction);
  });


  self.agent.on("exitCallStarted", function(transaction, exitCall) {
    if (!self.isEnabled || transaction.skip) {
      return;
    }

    libagentConnector.startExitCall(transaction, exitCall);
  });


  self.agent.on("exitCallStopped", function(transaction, exitCall, error) {
    if (!self.isEnabled || transaction.skip) {
      return;
    }

    libagentConnector.stopExitCall(exitCall, error);
  });
};


TransactionSender.prototype.createBtNamingWrapper = function(req) {
  // TODO: replace these with boost::regex in libagent bindings
  if (req.url) {
    var parsedUrl = url.parse(req.url);
    req.parsedPathName = parsedUrl.pathname;
    req.parsedParameterString = parsedUrl.query;
  }
  return req;
};
