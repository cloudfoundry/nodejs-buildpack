/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var os = require('os');
var fs = require('fs');
var util = require('util');
var path = require('path');
var EventEmitter = require('events').EventEmitter;
var crypto = require('crypto');

// modules used by both proxy and libagent
var Context = require('./context').Context;
var Logger = require('./logger').Logger;
var Timers = require('./timers').Timers;
var System = require('./system').System;
var AppDProxy = require('./appDProxy').AppDProxy;
var Thread = require('./thread').Thread;
var AnalyticsReporter = require('../transactions/analytics-reporter').AnalyticsReporter;
var MetricsManager = require('../metrics/metrics-manager').MetricsManager;
var ProcessInfo = require('../process/process-info').ProcessInfo;
var ProcessScanner = require('../process/process-scanner').ProcessScanner;
var ProcessStats = require('../process/process-stats').ProcessStats;
var InstanceTracker = require('../process/instance-tracker').InstanceTracker;
var StringMatcher = require('../proxy/string-matcher').StringMatcher;
var ExpressionEvaluator = require('../proxy/expression-evaluator').ExpressionEvaluator;
var ConfigManager = require('../transactions/config-manager').ConfigManager;
var TransactionRegistry = require('../transactions/transaction-registry').TransactionRegistry;
var TransactionNaming = require('../transactions/transaction-naming').TransactionNaming;
var TransactionRules = require('../transactions/transaction-rules').TransactionRules;
var SepRules = require('../transactions/sep-config').SepRules;
var Correlation = require('../transactions/correlation').Correlation;
var DataCollectors = require('../transactions/data-collectors').DataCollectors;
var Eum = require('../transactions/eum').Eum;
var Profiler = require('../profiler/profiler').Profiler;
var CustomTransaction = require('../profiler/custom-transaction').CustomTransaction;
var GCStats = require('../v8/gc-stats').GCStats;
var CpuProfiler = require('../v8/cpu-profiler').CpuProfiler;
var HeapProfiler = require('../v8/heap-profiler').HeapProfiler;
var BackendConfig = require('../proxy/backend-config').BackendConfig;
var agentVersion = require('../../appdynamics_version.json');
var appDNativeLoader = require('appdynamics-native');

// modules only used by libagent
var LibagentConnector = require('../libagent/libagent-connector').LibagentConnector;
var TransactionSender = require('../libagent/transaction-sender').TransactionSender;
var ProcessSnapshotSender = require('../libagent/process-snapshot-sender').ProcessSnapshotSender;
var MetricSender = require('../libagent/metric-sender').MetricSender;
var InstanceInfoSender = require('../libagent/instance-info-sender').InstanceInfoSender;

function Agent() {
  this.isWindows = os.platform() == 'win32';
  this.rootTmpDir = '/tmp/appd';
  if (this.isWindows) {
    this.rootTmpDir = os.tmpdir();
  } else if (!fs.existsSync('/tmp')) {
    this.rootTmpDir = './tmp/appd'; // Heroku case
  }

  this.initialized = false;
  this.version = agentVersion.version;
  this.compatibilityVersion = agentVersion.compatibilityVersion;
  this.nextId = Math.round(Math.random() * Math.pow(10, 6));
  this.appdNative = undefined;
  this.meta = [];

  // predefine options
  this.precompiled = undefined;

  this.proxyCtrlDir = undefined;
  this.proxyLogsDir = undefined;
  this.proxyRuntimeDir = undefined;

  EventEmitter.call(this);

  // create modules
  this.context = new Context(this);
  this.logger = new Logger(this);
  this.timers = new Timers(this);
  this.system = new System(this);
  this.proxy = new AppDProxy(this);
  this.thread = new Thread(this);
  this.analyticsReporter = new AnalyticsReporter(this);
  this.metricsManager = new MetricsManager(this);
  this.processInfo = new ProcessInfo(this);
  this.processScanner = new ProcessScanner(this);
  this.processStats = new ProcessStats(this);
  this.instanceTracker = new InstanceTracker(this);
  this.stringMatcher = new StringMatcher(this);
  this.expressionEvaluator = new ExpressionEvaluator(this);
  this.configManager = new ConfigManager(this);
  this.transactionRegistry = new TransactionRegistry(this);
  this.transactionNaming = new TransactionNaming(this);
  this.transactionRules = new TransactionRules(this);
  this.sepRules = new SepRules(this);
  this.correlation = new Correlation(this);
  this.dataCollectors = new DataCollectors(this);
  this.eum = new Eum(this);
  this.profiler = new Profiler(this);
  this.customTransaction = new CustomTransaction(this);
  this.gcStats = new GCStats(this);
  this.cpuProfiler = new CpuProfiler(this);
  this.heapProfiler = new HeapProfiler(this);
  this.backendConfig = new BackendConfig(this);

  this.libproxy = null;
  this.libagent = null;

  // nsolid support
  if (typeof (nsolid) == 'undefined') {
    try {
      this.nsolid = require('nsolid');
    } catch (e) {
      // ignored; not available
    }
  } else {
    this.nsolid = nsolid; // eslint-disable-line no-undef
  }

  this.nsolidEnabled = typeof (this.nsolid) !== 'undefined' &&
    typeof (this.nsolid.info) === 'function';

  this.nsolidMetaCollected = false;

  // libagent
  this.libagentConnector = new LibagentConnector(this);
  this.transactionSender = new TransactionSender(this);
  this.processSnapshotSender = new ProcessSnapshotSender(this);
  this.metricSender = new MetricSender(this);
  this.instanceInfoSender = new InstanceInfoSender(this);
}

util.inherits(Agent, EventEmitter);

/* istanbul ignore next */
Agent.prototype.recursiveMkDir = function (dir) {
  var dirsToMake = [];
  var currDir = path.normalize(dir);
  var parentDir = path.dirname(currDir);
  var controlVar = true;
  while (controlVar) {
    if (fs.existsSync(currDir))
      break;
    dirsToMake.push(currDir);
    currDir = parentDir;
    parentDir = path.dirname(currDir);
    if (currDir == parentDir)
      break;
  }

  while (dirsToMake.length) {
    currDir = dirsToMake.pop();
    try {
      fs.mkdirSync(currDir);
    } catch (e) {
      if (e.code != 'EEXIST') throw e;
    }
  }
};

Agent.computeTmpDir = function (rootTmpDir,
  controllerHost,
  controllerPort,
  appName,
  tierName,
  nodeName) {
  var stringToHash = controllerHost.toString() + ',' +
    controllerPort.toString() + ',' +
    appName.toString() + ',' +
    tierName.toString() + ',' +
    nodeName.toString();
  // We don't need cryptographic security here,
  // just a unique, predictable fixed length directory name.
  var md5 = crypto.createHash('md5');
  md5.update(stringToHash);
  return path.join(rootTmpDir, md5.digest('hex'));
};

/* istanbul ignore next -- requires too much stubbing and mocking to unit test */
Agent.prototype.init = function (opts) {
  var self = this;

  try {
    if (self.initialized)
      return;

    self.initialized = true;

    self.initializeOpts(opts);

    self.opts.clsDisabled = !!self.opts.clsDisabled;

    self.backendConnector = self.getAgentCustomFns(self.opts.proxy);

    self.precompiled = opts.precompiled === undefined || opts.precompiled;

    if (self.opts.excludeAgentFromCallGraph === undefined) {
      self.opts.excludeAgentFromCallGraph = true;
    }

    if (self.opts.rootTmpDir) {
      self.rootTmpDir = self.opts.rootTmpDir;
    }

    // Temp directory
    if (self.opts.tmpDir) {
      self.tmpDir = self.opts.tmpDir;
    }
    else {
      if (!self.opts.controllerHostName || self.opts.controllerHostName.length <= 0) {
        self.logger.error('AppDynamics agent cannot be started: controller host name is either missing or empty');
        return;
      }
      if (!self.opts.controllerPort) {
        self.logger.error('AppDynamics agent cannot be started: controller port is missing');
        return;
      }
      if (!self.opts.applicationName || self.opts.applicationName.length <= 0) {
        self.logger.error('AppDynamics agent cannot be started: application name is either missing or empty');
        return;
      }
      if (!self.opts.tierName || self.opts.tierName.length <= 0) {
        self.logger.error('AppDynamics agent cannot be started: application tier name is either missing or empty');
        return;
      }
      if (!self.opts.nodeName || self.opts.nodeName.length <= 0) {
        self.logger.error('AppDynamics agent cannot be started: node name is either missing or empty');
        return;
      }
      self.tmpDir = Agent.computeTmpDir(self.rootTmpDir,
        self.opts.controllerHostName,
        self.opts.controllerPort,
        self.opts.applicationName,
        self.opts.tierName,
        self.opts.nodeName);
    }

    self.backendConnector.addNodeIndexToNodeName();

    // Initialize logger first.
    self.backendConnector.initializeLogger();

    self.dumpRuntimeProperties();

    self.backendConnector.createCLRDirectories();

    self.setMaxListeners(15);

    // Load native extention
    self.loadNativeExtention();

    // Initialize core modules first.
    self.context.init();
    self.timers.init();
    self.system.init();
    self.proxy.init();
    self.thread.init();

    // Initialize other modules.
    self.backendConnector.init();
    self.stringMatcher.init();
    self.expressionEvaluator.init();
    self.configManager.init();
    self.transactionRegistry.init();
    self.transactionNaming.init();
    self.transactionRules.init();
    self.sepRules.init();
    self.correlation.init();
    self.dataCollectors.init();
    self.eum.init();

    // Metrics aggregator should be initialize before
    // metric senders.
    self.metricsManager.init();

    // Initialize the rest.
    self.analyticsReporter.init();
    self.processInfo.init();
    self.processScanner.init();
    self.processStats.init();
    self.instanceTracker.init();
    self.profiler.init();
    self.customTransaction.init();
    self.gcStats.init();
    self.cpuProfiler.init();
    self.heapProfiler.init();
    self.backendConfig.init();

    // Initialize libagent
    self.backendConnector.intializeAgentHelpers();

    // Prepare probes.
    self.loadProbes();

    self.fetchMetadata(function (err, meta) {
      if (err) {
        self.logger.error(err);
      }

      try {
        self.emit('session');
      }
      catch (err) {
        self.logger.error(err);
      }

      var filters = {
        dataFilters: opts.dataFilters || [],
        urlFilters: opts.urlFilters || [],
        messageFilters: opts.messageFilters || []
      };

      self.backendConnector.startAgent(meta, filters);
    });
  } catch (err) {
    self.logger.error('Appdynamics agent cannot be initialized due to ' + err + '\n' + err.stack);
  }
};

Agent.prototype.initializeOpts = function (opts) {
  var self = this;
  opts = opts || {};

  self.opts = opts;
  if (self.opts.controllerHostName === undefined) {
    self.opts.controllerHostName = process.env.APPDYNAMICS_CONTROLLER_HOST_NAME;
  }
  if (self.opts.controllerPort === undefined) {
    self.opts.controllerPort = process.env.APPDYNAMICS_CONTROLLER_PORT;
  }
  if (self.opts.controllerSslEnabled === undefined) {
    self.opts.controllerSslEnabled = process.env.APPDYNAMICS_CONTROLLER_SSL_ENABLED;
  }
  if (self.opts.accountName === undefined) {
    self.opts.accountName = process.env.APPDYNAMICS_AGENT_ACCOUNT_NAME;
  }
  if (self.opts.accountAccessKey === undefined) {
    self.opts.accountAccessKey = process.env.APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY;
  }
  if (self.opts.applicationName === undefined) {
    self.opts.applicationName = process.env.APPDYNAMICS_AGENT_APPLICATION_NAME;
  }
  if (self.opts.tierName === undefined) {
    self.opts.tierName = process.env.APPDYNAMICS_AGENT_TIER_NAME;
  }
  if (self.opts.nodeName === undefined) {
    self.opts.nodeName = process.env.APPDYNAMICS_AGENT_NODE_NAME;
  }
};


Agent.prototype.fetchMetadata = function (cb) {
  var self = this, key;

  function add(key, value) {
    self.meta.push({ name: key, value: value });
  }

  var processInfo = self.processInfo.fetchInfo();
  for (key in processInfo) {
    add(key, processInfo[key]);
  }

  if (!this.nsolidEnabled) {
    return cb(null, self.meta);
  }

  this.nsolid.info(function (err, results) {
    var label;

    self.nsolidMetaCollected = true;
    if (err) return cb(err);

    var labels = {
      app: 'App Name',
      appVersion: 'App Version',
      processStart: 'Process Start'
    };

    for (key in labels) if (labels.hasOwnProperty(key)) {
      add('N|Solid ' + labels[key], results[key]);
    }
    for (key in results.versions.nsolid_lib) if (results.versions.nsolid_lib.hasOwnProperty(key)) {
      label = 'N|Solid Module Versions: ' + key;
      add(label, results.versions.nsolid_lib[key]);
    }
    add('N|Solid Tags', results.tags.join(', '));

    cb(null, self.meta);
  });
};


Agent.prototype.profile = Agent.prototype.init;


/* istanbul ignore next -- not unit testable */
Agent.prototype.loadProbes = function () {
  var self = this;

  // Dynamic probes.
  var probeCons = [];
  probeCons.push(require('../probes/cluster-probe').ClusterProbe);
  probeCons.push(require('../probes/disk-probe').DiskProbe);
  probeCons.push(require('../probes/http-probe').HttpProbe);
  probeCons.push(require('../probes/http2-probe').Http2Probe);
  probeCons.push(require('../probes/memcached-probe').MemcachedProbe);
  probeCons.push(require('../probes/mongodb-probe').MongodbProbe);
  probeCons.push(require('../probes/mssql-probe').MssqlProbe);
  probeCons.push(require('../probes/mysql-probe').MysqlProbe);
  probeCons.push(require('../probes/net-probe').NetProbe);
  probeCons.push(require('../probes/nsolid-probe').NSolidProbe);
  probeCons.push(require('../probes/pg-probe').PgProbe);
  probeCons.push(require('../probes/redis-probe').RedisProbe);
  probeCons.push(require('../probes/ioredis-probe').IoredisProbe);
  probeCons.push(require('../probes/socket.io-probe').SocketioProbe);
  probeCons.push(require('../probes/dynamodb-probe').DynamoDbProbe);
  probeCons.push(require('../probes/couchbase-probe').CouchBaseProbe);
  probeCons.push(require('../probes/cassandra-probe').CassandraProbe);
  probeCons.push(require('../probes/express-probe').ExpressProbe);

  var packageProbes = {};
  probeCons.forEach(function (ProbeCon) {
    var probe = new ProbeCon(self);
    probe.packages.forEach(function (pkg) {
      packageProbes[pkg] = probe;
    });
  });

  // on demand probe attaching
  self.proxy.after(module.__proto__, 'require', function (obj, args, ret) {
    var probe = packageProbes[args[0]];
    if (probe) {
      return probe.attach(ret, args[0]);
    }
  });

  // Trying to preattaching probes.
  for (var name in packageProbes) {
    try {
      if (require.main) {
        require.main.require(name);
        self.logger.debug('found ' + name + ' module');
      }
    }
    catch (err) {
      // ignore exceptions
    }
  }


  // Explicit probes.
  var ProcessProbe = require('../probes/process-probe').ProcessProbe;
  new ProcessProbe(self).attach(process);
  var GlobalProbe = require('../probes/global-probe').GlobalProbe;
  new GlobalProbe(self).attach(global);
};


/* istanbul ignore next */
Agent.prototype.loadNativeExtention = function () {
  var self = this;

  if (!self.appdNative)
    self.appdNative = appDNativeLoader.load(this);
};



Agent.prototype.getNextId = function () {
  return this.nextId++;
};


Agent.prototype.destroy = function () {
  try {
    this.emit('destroy');
  }
  catch (err) {
    this.logger.error(err);
  }

  this.removeAllListeners();
};

Agent.prototype.getTransaction = function (req) {
  var threadId, txn;

  if (!this.initialized) return;

  threadId = req && req.__appdThreadId;
  txn = threadId && this.profiler.getTransaction(threadId);

  return txn && !txn.ignore && this.customTransaction.join(req, txn);
};

Agent.prototype.startTransaction = function (transactionInfo) {
  if (!this.initialized) return;

  return this.customTransaction.start(transactionInfo);
};

Agent.prototype.parseCorrelationInfo = function (source) {
  var corrHeader = this.correlation.newCorrelationHeader();
  if (typeof (source) === 'object') {
    source = source.headers && source.headers[this.correlation.HEADER_NAME];
  }
  if (!corrHeader.parse(source)) {
    return false;
  }
  return corrHeader;
};


Agent.prototype.createCustomMetric = function (name, unit, op, customRollup, holeHandling) {
  if (!this.initialized) {
    this.logger.warn('Can\'t create a metric before agent initialization');
    return;
  }

  name = name ? 'Custom Metrics|' + name : 'Custom Metrics';
  return this.metricsManager.createMetric({
    path: name,
    unit: unit
  }, op, customRollup, holeHandling, true);
};

Agent.prototype.addAnalyticsData = function (key, value) {
  var currentTxnTp = this.getCurrentTransactionTimePromise();
  if (!currentTxnTp) {
    this.logger.warn('Business transaction associated with the request cannot be found');
    return;
  }
  currentTxnTp.addAnalyticsData(key, value);
};

Agent.prototype.addSnapshotData = function (key, value) {
  var currentTxnTp = this.getCurrentTransactionTimePromise();
  if (!currentTxnTp) {
    this.logger.warn('Business transaction associated with the request cannot be found');
    return;
  }
  currentTxnTp.addSnapshotData(key, value);
};

Agent.prototype.markError = function (err, statusCode) {
  var currentTxnTp = this.getCurrentTransactionTimePromise();
  if (!currentTxnTp) {
    this.logger.warn('Business transaction associated with the request cannot be found');
    return;
  }
  currentTxnTp.markError(err, statusCode);
};

Agent.prototype.getCurrentTransactionTimePromise = function () {
  var threadId = this.thread.current();
  var txn = threadId && this.profiler.getTransaction(threadId);
  return txn && !txn.ignore && this.customTransaction.join(null, txn);
};
Agent.prototype.dumpRuntimeProperties = function () {
  var self = this;
  self.logger.env('NodeJS ' + process.arch + ' runtime properties for PID ' + process.pid);
  self.logger.env('Process command line [' + process.argv + ']');
  self.logger.env('Full node executable path: ' + process.execPath);
  var key;
  for (key in process.versions) {
    self.logger.env('version: ' + key + ' = ' + process.versions[key]);
  }
  for (key in process.features) {
    self.logger.env('feature: ' + key + ' = ' + process.features[key]);
  }
  for (key in process.release) {
    self.logger.env('release information: ' + key + ' = ' + process.release[key]);
  }
  for (key in process.config.target_defaults) {
    self.logger.env('configuration target defaults: ' + key + ' = ' + process.config.target_defaults[key]);
  }
  for (key in process.config.variables) {
    self.logger.env('configuration variables: ' + key + ' = ' + process.config.variables[key]);
  }
};

Agent.prototype.getAgentCustomFns = function (isProxy) {
  if (isProxy) {
    var LibProxy = require('../libproxy/libproxy').LibProxy;
    return new LibProxy(this);
  } else {
    var LibAgent = require('../libagent/libagent').LibAgent;
    return new LibAgent(this);
  }
};

var AppDynamics = function () {
  var self = this;

  var agent = new Agent();
  ['profile',
    'destroy',
    'getTransaction',
    'startTransaction',
    'parseCorrelationInfo',
    'createCustomMetric',
    'addAnalyticsData',
    'addSnapshotData',
    'markError'
  ].forEach(function (meth) {
    self[meth] = function () {
      return agent[meth].apply(agent, arguments);
    };
  });

  ['on',
    'addListener',
    'pause',
    'resume'
  ].forEach(function (meth) {
    self[meth] = function () {
      // deprecated
    };
  });

  // Here so jasmine tests can access
  // agent functions.
  self.__AppDynamics = AppDynamics;
  self.__agent = agent;
};

// Here so jasmine tests can access
// agent functions.
AppDynamics.Agent = Agent;

exports = module.exports = new AppDynamics();
