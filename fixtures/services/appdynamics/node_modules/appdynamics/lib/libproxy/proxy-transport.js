/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var util = require('util');
var os = require('os');
var fs = require('fs');
var cluster = require('cluster');
var path = require('path');
var ProtobufModel = require('../proxy/protobuf-model').ProtobufModel;
var Schema;
var zmq;

var DEFAULT_PROXY_COMM_PORT = 10101;
var DEFAULT_PROXY_REQUEST_PORT = 10102;
var DEFAULT_PROXY_REPORTING_PORT = 10103;

// NOTE: these dependencies aren't fully baked (missing binary addons),
// so we skip loading them during unit testing.
if (process.env.NODE_ENV != 'appd_test') {
  Schema = require('appdynamics-protobuf').Schema;
  zmq = require('appdynamics-zmq');
}

function ProxyTransport(agent) {
  this.agent = agent;

  this.schema = undefined;
  this.controlSock = undefined;
  this.configSocket = undefined;
  this.reportingSocket = undefined;
  this.infoSocket = undefined;

  this.protobufModel = undefined;
}
exports.ProxyTransport = ProxyTransport;

/* istanbul ignore next -- not unit testable */
ProxyTransport.prototype.init = function() {
  var self = this;

  self.protobufModel = new ProtobufModel(self.agent);
  self.protobufModel.init();
  self.schema = new Schema(fs.readFileSync(path.join(__dirname, '../..', 'conf/protobuf.desc')));
  self.nextMessageId = 0;
  self.transactionMap = {};

  self.agent.once('proxyStarted', function(nodeIndex) {
    if (self.agent.nsolidEnabled && !self.agent.nsolidMetaCollected) {
      // wait for nsolid metadata
      self.agent.on('session', launch);
    } else {
      launch();
    }

    function launch() {
      try {
        self.control(nodeIndex);
      }
      catch(err) {
        self.agent.logger.error(err);
      }
    }
  });


  self.proxyConnected = false;

  self.agent.on('configUpdated', function() {
    self.proxyConnected = true;
  });
};

ProxyTransport.prototype.cleanupTransactionMap = function() {
  var self = this;

  var minTs = Date.now() - 10000;
  for(var requestId in self.transactionMap) {
    var transaction = self.transactionMap[requestId];
    if(transaction.ts < minTs) {
      self.agent.logger.info('cleanup transaction with requestId ' + requestId);
      delete self.transactionMap[requestId];
    }
  }
};

/* istanbul ignore next -- not unit testable */
ProxyTransport.prototype.control = function(nodeIndex) {
  var self = this;

  if(cluster.isMaster && !self.agent.opts.monitorClusterMaster) {
    // trying to identify cluster master
    var isClusterMaster = false;
    if(cluster.workers) {
      if ((Object.keys(cluster.workers)).length > 0)
        isClusterMaster = true;
    }

    if(isClusterMaster) {
      // do not monitor cluster master node
      return;
    }
  }

  self.agent.logger.debug('starting control socket..');

  self.StartNodeRequest = self.schema['appdynamics.pb.StartNodeRequest'];
  self.StartNodeResponse = self.schema['appdynamics.pb.StartNodeResponse'];

  self.controlSock = zmq.socket('req');
  self.controlSock.setsockopt(zmq.ZMQ_LINGER, 0);
  self.controlSock.connect(self.agent.isWindows
                          ? 'tcp://localhost:' + (self.agent.opts.proxyCommPort || DEFAULT_PROXY_COMM_PORT)
                          : 'ipc://' + self.agent.proxyCtrlDir + '/0');
  self.controlSock.on('message', function(envelope, message) {
    self.agent.logger.debug('control response message received from proxy');

    if(self.controlResponseTimeout) {
      self.agent.timers.clearInterval(self.controlResponseTimeout);
      self.controlResponseTimeout = undefined;
    }

    if(message) {
      self.receiveStartNodeResponse(message);
    }
  });
  self.controlSock.on('error', function(err) {
    self.agent.logger.error(err);
  });

  self.controlRetryCount = 0;
  self.controlResponseTimeout = self.agent.timers.setInterval(function() {
    self.agent.logger.error('no response on control socket');
    if(++self.controlRetryCount > 3) {
      self.agent.logger.error('unable to contact proxy');
      self.agent.timers.clearInterval(self.controlResponseTimeout);
    } else if(!self.agent.opts.proxyAutolaunchDisabled && cluster.isMaster) {
      self.agent.logger.warn('attempting to re-launch proxy...');
      self.agent.emit('launchProxy', nodeIndex, true);
    }
  }, 10000);

  self.agent.timers.setTimeout(function() {
    self.sendStartNodeRequest(nodeIndex);
  }, 2000);
};

/* istanbul ignore next -- not unit testable */
ProxyTransport.prototype.start = function(dataSocketDirPath, skipInitialConfigRequest) {
  var self = this;

  // proxy communication directory for zmq
  var opts = self.agent.opts;

  // init transport and response listeners
  self.ASyncRequest = self.schema['appdynamics.pb.ASyncRequest'];
  self.ConfigResponse = self.schema['appdynamics.pb.ConfigResponse'];
  self.BTInfoRequest = self.schema['appdynamics.pb.BTInfoRequest'];
  self.BTInfoResponse = self.schema['appdynamics.pb.BTInfoResponse'];
  self.ASyncMessage = self.schema['appdynamics.pb.ASyncMessage'];

  self.configSocket = zmq.socket('router');
  self.configSocket.setsockopt(zmq.ZMQ_LINGER, 0);
  self.configSocket.connect(self.agent.isWindows
                           ? 'tcp://localhost:' + (opts.proxyRequestPort || DEFAULT_PROXY_REQUEST_PORT)
                           : 'ipc://' + dataSocketDirPath + '/0');
  self.configSocket.on('message', function(envelope, empty, part1) {
    self.agent.logger.debug('config response message received from proxy');
    if (part1) {
      self.receiveConfigResponse(part1);
    }
    else {
      self.agent.logger.warn('empty message from proxy');
    }
  });
  self.configSocket.on('error', function(err) {
    self.agent.logger.warn(err);
  });

  self.infoSocket = zmq.socket('router');
  self.infoSocket.setsockopt(zmq.ZMQ_LINGER, 0);
  self.infoSocket.connect(self.agent.isWindows
                         ? 'tcp://localhost:' + (opts.proxyRequestPort || DEFAULT_PROXY_REQUEST_PORT)
                         : 'ipc://' + dataSocketDirPath + '/0');
  self.infoSocket.on('message', function(envelope, empty, part1) {
    self.agent.logger.debug('info response message received from proxy');
    if (part1) {
      self.receiveBTInfoResponse(part1);
    } else {
      self.agent.logger.warn('empty message from proxy');
    }
  });
  self.infoSocket.on('error', function(err) {
    self.agent.logger.warn(err);
  });

  self.reportingSocket = zmq.socket('pub');
  self.reportingSocket.setsockopt(zmq.ZMQ_LINGER, 0);
  self.reportingSocket.connect(self.agent.isWindows
                              ? 'tcp://localhost:' + (opts.proxyReportingPort || DEFAULT_PROXY_REPORTING_PORT)
                              : 'ipc://' + dataSocketDirPath + '/1');
  self.reportingSocket.on('error', function(err) {
    self.agent.logger.warn(err);
  });

  // send on connection, do not wait
  if(!skipInitialConfigRequest) {
    self.agent.timers.setTimeout(function() {
      self.sendConfigRequest();
    }, 2000);
  }

  // should we also request config after registering unregistered?
  self.agent.timers.setInterval(function() {
    self.sendConfigRequest();
  }, 30000);

  self.agent.timers.setInterval(function() {
    self.cleanupTransactionMap();
  }, 10000);
};

// Writing the start node request to a json file helps
// support debug controller communication problems.
/* istanbul ignore next */
ProxyTransport.prototype.writeStartNodeRequestJSON = function (startNodeRequestObj) {
  var self = this;
  try {
    var fileNameRegEx = /[\/\?<>\\:\*\|":\x00-\x1f\x80-\x9f]/g;
    var fileName = startNodeRequestObj.nodeName.replace(fileNameRegEx, "_") + ".json";
    var controllerInfoJSONFile = fs.openSync(path.join(self.agent.tmpDir, fileName), 'w');
    try {
      fs.writeSync(controllerInfoJSONFile, JSON.stringify(startNodeRequestObj, null, " ") + "\n");
    }
    finally {
      fs.closeSync(controllerInfoJSONFile);
    }
  }
  catch (err) {
    self.agent.logger.warn(err);
  }
};

ProxyTransport.prototype.isControllerSSLEnabled = function(controllerSslEnabled) {
  return (controllerSslEnabled === 'true' || controllerSslEnabled === '1' || controllerSslEnabled === true) || false;
};

ProxyTransport.prototype.sendStartNodeRequest = function(nodeIndex) {
  var self = this;
  var opts = self.agent.opts;

  var computedNodeName = (opts.nodeName || os.hostname());
  if (!opts.noNodeNameSuffix)
    computedNodeName += '-' + nodeIndex;

  var proxyDir = require('appdynamics-proxy').dir,
    proxySha = '';
  if (fs.existsSync(proxyDir + '/sha')) {
    proxySha = fs.readFileSync(proxyDir + '/sha');
  }
  var agentVersion = self.agent.version.split('.'),
    proxyVersion = self.agent.compatibilityVersion ? self.agent.compatibilityVersion : '';

  agentVersion.pop();
  agentVersion = agentVersion.join('.');
  var startNodeRequestObj = {
    appName: opts.applicationName,
    tierName: opts.tierName,
    nodeName: computedNodeName,
    controllerHost: opts.controllerHostName,
    controllerPort: opts.controllerPort,
    sslEnabled: self.isControllerSSLEnabled(opts.controllerSslEnabled),
    accountName: opts.accountName,
    accountAccessKey: opts.accountAccessKey,
    httpProxyHost: opts.proxyHost,
    httpProxyPort: opts.proxyPort,
    httpProxyUser: opts.proxyUser,
    httpProxyPasswordFile: opts.proxyPasswordFile,
    agentVersion: 'Proxy v' + agentVersion + ' compatible with ' + proxyVersion + ' SHA ' + proxySha,
    metadata: self.agent.meta
  };

  if (opts.reuseNode)
    startNodeRequestObj.nodeReuse = opts.reuseNode;
  if (opts.reuseNodePrefix)
    startNodeRequestObj.nodeReusePrefix = opts.reuseNodePrefix;

  if (self.agent.isWindows) {
    startNodeRequestObj.requestPort = opts.proxyRequestPort || DEFAULT_PROXY_REQUEST_PORT;
    startNodeRequestObj.reportingPort = opts.proxyReportingPort || DEFAULT_PROXY_REPORTING_PORT;
  }

  self.writeStartNodeRequestJSON(startNodeRequestObj);

  try {
    if (self.agent.logger.isDebugEnabled()) {
      self.agent.logger.debug(util.inspect(startNodeRequestObj, {depth: 20}));
    }
    var startNodeRequestBytes = self.StartNodeRequest.serialize(startNodeRequestObj);
    self.controlSock.send(["", startNodeRequestBytes]);
  }
  catch(err) {
    self.agent.logger.warn(err);
  }
};

ProxyTransport.prototype.receiveStartNodeResponse = function(payload) {
  var self = this;

  var startNodeResponseObj = self.StartNodeResponse.parse(payload);

  if(startNodeResponseObj) {
    if(startNodeResponseObj.dataSocketDirPath) {
      self.start(startNodeResponseObj.dataSocketDirPath, !!startNodeResponseObj.configResponse);
    }

    if(startNodeResponseObj.configResponse) {
      var configResponseObj = startNodeResponseObj.configResponse;
      if (self.agent.logger.isDebugEnabled()) {
        self.agent.logger.debug(util.inspect(configResponseObj, {depth: 20}));
      }

      self.agent.configManager.updateConfig(configResponseObj);

      if(configResponseObj.processCallGraphReq) {
        self.agent.processScanner.startManualSnapshot(
          configResponseObj.processCallGraphReq,
          self.sendProcessSnapshot.bind(self));
      }
    }
  }
};

ProxyTransport.prototype.sendConfigRequest = function() {
  var self = this;

  var lastConfigVersion = self.agent.configManager.getConfigVersion();
  var processMetrics = self.agent.metricsManager.getProcessMetrics();

  var asyncRequestObj = {
    type: 'CONFIG',
    configReq: {
      lastVersion: lastConfigVersion,
      nodejsProcessMetrics: processMetrics
    }
  };

  try {
    if (self.agent.logger.isDebugEnabled()) {
      self.agent.logger.debug(util.inspect(asyncRequestObj, {depth: 20}));
    }
    var asyncRequestBytes = self.ASyncRequest.serialize(asyncRequestObj);

    self.configSocket.send([
      "AsyncReqRouter",
      "",
      asyncRequestBytes
    ]);
  }
  catch(err) {
    self.agent.logger.warn(err);
  }
};

ProxyTransport.prototype.receiveConfigResponse = function(payload) {
  var self = this;

  var configResponseObj = self.ConfigResponse.parse(payload);
  if ((process.env.NODE_ENV == 'test') && configResponseObj.agentIdentity && configResponseObj.agentIdentity.appID) {
    self.agent.emit('configResponse');
  }
  if (self.agent.logger.isDebugEnabled()) {
    self.agent.logger.debug(util.inspect(configResponseObj, {depth: 20}));
  }

  self.agent.configManager.updateConfig(configResponseObj);

  if(configResponseObj.processCallGraphReq) {
    self.agent.processScanner.startManualSnapshot(
      configResponseObj.processCallGraphReq,
      self.sendProcessSnapshot.bind(self));
  }
};

ProxyTransport.prototype.sendBTInfoRequest = function(transaction) {
  var self = this;

  var btInfoRequest = {
    requestID: transaction.id.toString(),
    messageID: 0,
    btIdentifier: self.protobufModel.createBTIdentifier(transaction),
    correlation: self.protobufModel.createCorrelation(transaction),
  };
  var crossAppCorrelation =
    transaction.corrHeader && transaction.corrHeader.crossAppCorrelation;
  if(crossAppCorrelation) {
    btInfoRequest.crossAppCorrelationBackendId =
      transaction.corrHeader.crossAppCorrelationBackendId;
  }

  if (transaction.incomingCrossAppSnapshotEnabled) {
    btInfoRequest.incomingCrossAppSnapshotEnabled = true;
  }

  // assign transaction to message id in order to find it when info response is received
  transaction.btInfoRequest = btInfoRequest;
  self.transactionMap[btInfoRequest.requestID] = transaction;

  // send to proxy
  var asyncRequestObj = {
    type: 'BTINFO',
    btInfoReq: btInfoRequest
  };

  try {
    var asyncRequestBytes = self.ASyncRequest.serialize(asyncRequestObj);

    self.infoSocket.send([
      "AsyncReqRouter",
      "",
      asyncRequestBytes
    ]);
  }
  catch(err) {
    delete self.transactionMap[transaction.id];
    self.agent.logger.warn(err);
  }
};

ProxyTransport.prototype.receiveBTInfoResponse = function(payload) {
  var self = this;

  var btInfoResponse = self.BTInfoResponse.parse(payload);
  if (self.agent.logger.isDebugEnabled()) {
    self.agent.logger.debug("btInfoResponse");
    self.agent.logger.debug(btInfoResponse);
  }

  if(btInfoResponse.processCallGraphReq) {
    self.agent.processScanner.startManualSnapshot(
      btInfoResponse.processCallGraphReq,
      self.sendProcessSnapshot.bind(self));
  }

  var transaction = self.transactionMap[btInfoResponse.requestID];
  if(transaction) {
    delete self.transactionMap[btInfoResponse.requestID];

    if(transaction.isSent) {
      return;
    }

    transaction.btInfoResponse = btInfoResponse;

    // If snapshot is required, try to start a process snapshot.
    if (btInfoResponse.isSnapshotRequired) {
      self.agent.processScanner.startAutoSnapshotIfPossible(
        self.sendProcessSnapshot.bind(self));
    }
    else if (btInfoResponse.sendSnapshotIfContinuing &&
             transaction.corrHeader &&
             (!transaction.corrHeader.crossAppCorrelation) &&
             transaction.corrHeader.getSubHeader(self.agent.correlation.SNAPSHOT_ENABLE)) {
      self.agent.processScanner.startAutoSnapshotIfPossible(
        self.sendProcessSnapshot.bind(self));
    }

    transaction.emit('btInfoResponse');

    // if transaction has finished
    if(transaction.isFinished) {
      self.sendTransactionDetails(transaction);
      return;
    }
  }
};

ProxyTransport.prototype.sendTransactionDetails = function(transaction) {
  var self = this;

  if(transaction.isSent) {
    self.agent.logger.debug('Transaction has already been sent. Name: ' + transaction.name);
    return;
  }
  transaction.isSent = true;

  var messageObj = {
    type: 'BTDETAILS',
    btDetails: self.protobufModel.createBTDetails(transaction)
  };

  try {
    var asyncMessageBytes = self.ASyncMessage.serialize(messageObj);
    if (self.agent.logger.isDebugEnabled()) {
      self.agent.logger.debug('BTDetails payload');
      self.agent.logger.debug(util.inspect(self.ASyncMessage.parse(asyncMessageBytes), {depth: 20}));
    }

    self.reportingSocket.send(asyncMessageBytes);
  }
  catch(err) {
    self.agent.logger.warn(err);
  }

  self.agent.emit('btDetails', messageObj.btDetails, transaction);
};

ProxyTransport.prototype.sendSelfReResolution = function(selfReResolution) {
  var self = this;

  var messageObj = {
    type: 'SELFRERESOLUTION',
    selfReResolution: selfReResolution
  };

  try {
    var asyncMessageBytes = self.ASyncMessage.serialize(messageObj);
    if (self.agent.logger.isDebugEnabled()) {
      self.agent.logger.debug('SelfReResolution payload');
      self.agent.logger.debug(util.inspect(self.ASyncMessage.parse(asyncMessageBytes), {depth: 20}));
    }

    self.reportingSocket.send(asyncMessageBytes);
  }
  catch(err) {
    self.agent.logger.warn(err);
  }
};

ProxyTransport.prototype.sendInstanceData = function(instanceData) {
  var self = this;

  var messageObj = {
    type: 'INSTANCEDATA',
    instanceData: {
      instances:
      instanceData
    }
  };

  try {
    var asyncMessageBytes = self.ASyncMessage.serialize(messageObj);
    if (self.agent.logger.isDebugEnabled()) {
      self.agent.logger.debug(util.inspect(self.ASyncMessage.parse(asyncMessageBytes), {depth: 20}));
    }
    self.reportingSocket.send(asyncMessageBytes);
  } catch(err) {
    self.agent.logger.warn(err);
  }
};

ProxyTransport.prototype.sendProcessSnapshot = function(err, processSnapshot) {
  var self = this;

  if (err) {
    self.agent.logger.error(err);
    return;
  }

  var messageObj = {
    type: 'PROCESSSNAPSHOT',
    processSnapshot: processSnapshot
  };

  try {
    var asyncMessageBytes = self.ASyncMessage.serialize(messageObj);
    if (self.agent.logger.isDebugEnabled()) {
      self.agent.logger.debug('ProcessSnapshot payload');
      self.agent.logger.debug(util.inspect(self.ASyncMessage.parse(asyncMessageBytes), {depth: 20}));
    }
    self.reportingSocket.send(asyncMessageBytes);
  }
  catch(err) {
    self.agent.logger.warn(err);
  }
};

ProxyTransport.prototype.sendCustomMetricData = function(metric) {
  var self = this,
    customMetric = {
      name: metric.metricName,
      value: metric._value,
      clusterRollup: metric.clusterRollup,
      timeRollup: metric.op,
      holeHandling: metric.holeHandling
    },
    messageObj = {
      type: 'CUSTOMMETRIC',
      customMetric: customMetric
    };

  try {
    var asyncMessageBytes = self.ASyncMessage.serialize(messageObj);
    self.agent.logger.debug('Custom Metric payload');
    self.agent.logger.debug(util.inspect(self.ASyncMessage.parse(asyncMessageBytes), {depth: 20}));
    self.reportingSocket.send(asyncMessageBytes);
  }
  catch(err) {
    self.agent.logger.error(err);
  }
};

ProxyTransport.prototype.sendAppException = function(error) {
  var self = this;

  if(!self.proxyConnected) return;

  var appException = this.protobufModel.createAppException(error);
  if (!appException) return;

  var messageObj = {
    type: 'APPEXCEPTION',
    appException: appException
  };

  try {
    var asyncMessageBytes = self.ASyncMessage.serialize(messageObj);
    if (self.agent.logger.isDebugEnabled()) {
      self.agent.logger.debug('AppException payload');
      self.agent.logger.debug(util.inspect(self.ASyncMessage.parse(asyncMessageBytes), {depth: 20}));
    }

    self.reportingSocket.send(asyncMessageBytes);
  }
  catch (err) {
    self.agent.logger.warn(err);
  }
};

// selection sort algorithm

