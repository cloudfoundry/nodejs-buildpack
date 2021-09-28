/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
/* global process, require, exports */
'use strict';

var Metric = require('./metric').Metric;
var ProxyLauncher = require('./proxy-launcher').ProxyLauncher;
var ProxyTransport = require('./proxy-transport').ProxyTransport;
var InstanceInfoSender = require('./instance-info-sender').InstanceInfoSender;
var TransactionReporter = require('./transaction-reporter').TransactionReporter;
var ExitCall = require('../transactions/exit-call').ExitCall;
var path = require('path');

function LibProxy(agent) {
  this.agent = agent;
  this.proxyLauncher = new ProxyLauncher(agent);
  this.proxyTransport = new ProxyTransport(agent);
  this.instanceInfoSender = new InstanceInfoSender(agent);
  this.transactionReporter = new TransactionReporter(agent);
  // For unit-testing
  this.proxyMode = true;

  agent.Metric = Metric;
}

exports.LibProxy = LibProxy;

LibProxy.prototype.init = function () {
  this.proxyTransport.init();
  this.proxyLauncher.init();
  this.proxyLauncher.start();
  this.instanceInfoSender.init();
  this.transactionReporter.init();
};

LibProxy.prototype.addNodeIndexToNodeName = function () {
  // No logic is present in this function. Proxy already
  // does this work in proxy-launcher.
  // Function is placed here to match the corresponding libagent file.
};

LibProxy.prototype.initializeLogger = function () {
  this.agent.logger.init(this.agent.opts.logging, true);
};

LibProxy.prototype.createCLRDirectories = function () {
  var proxyTmpDir = path.join(this.agent.tmpDir, 'proxy');

  this.agent.proxyCtrlDir = LibProxy.resolveProxyCtrlDir(path.join(proxyTmpDir, 'c'), this.agent.opts);
  this.agent.logger.info("agent.proxyCtrlDir = " + JSON.stringify(this.agent.proxyCtrlDir));
  this.agent.recursiveMkDir(this.agent.proxyCtrlDir);

  this.agent.proxyLogsDir = path.join(proxyTmpDir, 'l');
  this.agent.recursiveMkDir(this.agent.proxyLogsDir);

  this.agent.proxyRuntimeDir = path.join(proxyTmpDir, 'r');
  this.agent.recursiveMkDir(this.agent.proxyRuntimeDir);
};

LibProxy.prototype.intializeAgentHelpers = function () {
  // Function is placed here to match the corresponding libagent file.
};

LibProxy.prototype.startAgent = function () {
  // Function is placed here to match the corresponding libagent file.
};

LibProxy.prototype.createExitCall = function (time, exitCallInfo) {
  var self = this;
  var exitType = exitCallInfo.exitType.replace(/^EXIT_/, '');
  // if there's no backend config (OOTB or custom), then ignore this exit call;
  // if exit call has backend config and identifying properties already, we use
  // those (which allows API constructed exit calls to supply these values).
  var backendConfig = exitCallInfo.backendConfig;
  var props = exitCallInfo.identifyingProperties;
  if (!props) {
    var configType = exitCallInfo.configType || (exitType[0] + exitType.substring(1).toLowerCase());

    if (!backendConfig) {
      var configTypeMethod = 'get' + configType + 'Config';
      if (configTypeMethod in self.agent.backendConfig) {
        backendConfig = self.agent.backendConfig[configTypeMethod](exitCallInfo.supportedProperties);
      }
    }

    if (!backendConfig) {
      return;
    }

    var propertiesMethod = 'populate' + configType + 'Properties';
    if (propertiesMethod in self.agent.backendConfig) {
      props = self.agent.backendConfig['populate' + configType + 'Properties'](backendConfig, exitCallInfo.supportedProperties);
    }
  }
  if (!props) {
    return;
  }

  exitCallInfo.backendConfig = backendConfig;
  exitCallInfo.identifyingProperties = props;

  var exitCall = new ExitCall(exitCallInfo);
  exitCall.time = time;
  exitCall.id = time.id;
  exitCall.ts = time.begin;
  exitCall.threadId = time.threadId;

  var transaction = self.agent.profiler.transactions[time.threadId];

  if (transaction && !transaction.ignore) {
    exitCall.sequenceInfo = self.agent.profiler.__getNextSequenceInfo(transaction);

    if (transaction.api && transaction.api.beforeExitCall) {
      exitCall = transaction.api.beforeExitCall(exitCall);
      if (!exitCall) {
        return;
      }
    }

    if (!transaction.startedExitCalls) {
      transaction.startedExitCalls = [];
    }

    transaction.startedExitCalls.push(exitCall);
  }

  return exitCall;
};

LibProxy.prototype.getCorrelationHeader = function (exitCall) {
  if (!exitCall.backendConfig.correlationEnabled) return;
  var transaction = this.agent.profiler.getTransaction(exitCall.threadId);
  if (!transaction) return;
  var corrHeader = this.agent.correlation.newCorrelationHeader();
  corrHeader.build(transaction, exitCall);

  // to check if we are ignoring possible snapshot request from btInfoResponse
  if (!transaction.ignore && !transaction.btInfoResponse) {
    this.agent.logger.debug("btInfoResponse is not yet available");
  }
  return corrHeader.getStringHeader();
};

LibProxy.prototype.createSnapshotTrigger = function (transaction) {
  if (!transaction.btInfoResponse) return {
    attachSnapshot: false
  };
  if (transaction.btInfoResponse.isSnapshotRequired) {
    return {
      attachSnapshot: true,
      snapshotTrigger: 'REQUIRED'
    };
  } else if (transaction.btInfoResponse.currentSlowThreshold > 0 &&
    transaction.btInfoResponse.currentSlowThreshold < transaction.ms) {
    return {
      attachSnapshot: true,
      snapshotTrigger: 'SLOW'
    };
  }
  else if (transaction.btInfoResponse.sendSnapshotIfError &&
    transaction.hasErrors) {
    return {
      attachSnapshot: true,
      snapshotTrigger: 'ERROR'
    };
  }
  else if (transaction.btInfoResponse.sendSnapshotIfContinuing &&
    transaction.corrHeader &&
    (!transaction.corrHeader.crossAppCorrelation) &&
    transaction.corrHeader.getSubHeader(this.agent.correlation.SNAPSHOT_ENABLE)) {
    return {
      attachSnapshot: true,
      snapshotTrigger: 'CONTINUING'
    };
  } else {
    return {
      attachSnapshot: false,
      snapshotTrigger: undefined
    };
  }
};

LibProxy.prototype.startProcessSnapshot = function () {
};

LibProxy.resolveProxyCtrlDir = function (defaultDir, opts) {
  var dirFromOpts = opts.proxyCtrlDir;
  if (!dirFromOpts) {
    return defaultDir;
  }
  dirFromOpts = path.normalize(dirFromOpts.toString());
  if (path.dirname(dirFromOpts) == dirFromOpts) {
    // Eeeck!!!
    // Looks like someone tried to specify the root directory as
    // the proxy communication directory.   There is a really unsafe
    // rm -rf in proxy-launcher that will do a:
    // rm -rf //* if the proxy communication directory is '/'.
    return defaultDir;
  }
  return dirFromOpts;
};
