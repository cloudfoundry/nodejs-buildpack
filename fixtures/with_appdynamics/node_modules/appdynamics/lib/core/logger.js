/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var cluster = require('cluster');
var log4js = require('log4js');
var path = require('path');


function isDebugEnabled(agent) {
  return agent && agent.opts && agent.opts.debug;
}

function Logger(agent) {
  this.agent = agent;

  // temporary stubs, until initialized
  /* istanbul ignore next */
  this.logger = {
    trace: function (msg) {
      if (isDebugEnabled(agent)) console.log(msg);
    },
    debug: function (msg) {
      if (isDebugEnabled(agent)) console.log(msg);
    },
    info: function (msg) {
      if (isDebugEnabled(agent)) console.info(msg);
    },
    warn: function (msg) {
      if (isDebugEnabled(agent)) console.warn(msg);
    },
    error: function (msg) {
      if (isDebugEnabled(agent)) console.error(msg);
    },
    fatal: function (msg) {
      if (isDebugEnabled(agent)) console.error(msg);
    },
    env: function (msg) {
      if (isDebugEnabled(agent)) console.info(msg);
    },
    isTraceEnabled: function () { return isDebugEnabled(agent); },
    isDebugEnabled: function () { return isDebugEnabled(agent); },
    isInfoEnabled: function () { return isDebugEnabled(agent); },
    isWarnEnabled: function () { return isDebugEnabled(agent); },
    isErrorEnabled: function () { return isDebugEnabled(agent); },
    isFatalEnabled: function () { return isDebugEnabled(agent); }
  };
}
exports.Logger = Logger;
Logger.sharedLogger = undefined; // (for unit tests)


/* istanbul ignore next -- log4j mocked in unit tests */
Logger.prototype.init = function (config, initializeBackend) {
  if (!config) {
    config = {};
  }

  if (initializeBackend && Logger.sharedLogger) {
    // work-around for unit tests to prevent too many loggers
    // being created, which kills output to stdout :-/
    this.logger = Logger.sharedLogger;
  } else {
    var dateString = new Date().toJSON();
    dateString = dateString.replace(/[-:]/g, "_").replace("T", "__").replace(/\..*$/, "");
    var logFileName = 'appd_node_agent_' + dateString + ".log";
    var rootPath = config.root_directory || this.agent.tmpDir;
    var baseName = config.filename || logFileName;
    // Libagent logFile is already created in the libagent. Since, there is no
    // way to tap on to the file name created by boost::log used by libagent
    // show the log filePath till agent tmpDir.
    var fileName = initializeBackend ? path.join(rootPath, baseName) : this.agent.tmpDir;


    var consoleOnly = (this.agent.opts.logging && this.agent.opts.logging.logfiles &&
      this.agent.opts.logging.logfiles.every(function (lf) {
        return lf.outputType == 'console';
      }));

    // If console logging env variable is set and no custom logging config is provided
    // then don't create temp path for logs
    if (process.env.APPDYNAMICS_LOGGER_OUTPUT_TYPE === "console" && (!config.logfiles || config.logfiles.length == 0)) {
      consoleOnly = true;
    }

    if (!consoleOnly) {
      this.agent.recursiveMkDir(rootPath);
    }

    if (this.agent.opts.debug) {
      // force logging level to DEBUG and output location of logs
      config.level = 'DEBUG';
      if (!consoleOnly) {
        console.log('[DEBUG] Appdynamics agent logs: ' + fileName);
      }
    }

    // libagent uses its own logging, and does not need the log4js backend to be initialized
    if (!initializeBackend) {
      this.libAgentConnector.initLogger();
      return;
    }

    if (process.env.APPDYNAMICS_LOGGER_OUTPUT_TYPE === "console") {
      // Update the outputType in logging config to console for JavaProxy
      config.outputType = config.outputType || "console";
    }

    if (cluster.isMaster) {
      log4js.configure({
        appenders: [{
          type: 'clustered',
          appenders: [{
            type: 'logLevelFilter',
            level: config.level || process.env.APPDYNAMICS_LOGGER_LEVEL || 'INFO',
            appender: {
              type: config.outputType || 'file',
              filename: fileName,
              maxLogSize: config.max_size || 5242880,
              backups: config.max_files || 10,
              category: 'default',
              mode: config.mode || 'append'
            }
          }]
        }]
      });
    }
    else {
      log4js.configure({
        appenders: [{
          type: 'clustered'
        }]
      });
    }

    Logger.sharedLogger = this.logger = log4js.getLogger('default');
  }
};

Logger.prototype.trace = function (msg) {
  if (this.libAgentConnector) {
    this.libAgentConnector.logTrace(msg);
    return;
  }
  this.logger.trace(msg);
};

Logger.prototype.debug = function (msg) {
  if (this.libAgentConnector) {
    this.libAgentConnector.logDebug(msg);
    return;
  }
  this.logger.debug(msg);
};

Logger.prototype.info = function (msg) {
  if (this.libAgentConnector) {
    this.libAgentConnector.logInfo(msg);
    return;
  }
  this.logger.info(msg);
};

Logger.prototype.warn = function (msg) {
  if (this.libAgentConnector) {
    this.libAgentConnector.logWarn(msg);
    return;
  }
  this.logger.warn(msg);
};

Logger.prototype.error = function (err) {
  if (this.libAgentConnector) {
    this.libAgentConnector.logError(err.stack ? err.stack : err);
    return;
  }
  this.logger.error(err.stack ? err.stack : err);
};

Logger.prototype.fatal = function (msg) {
  if (this.libAgentConnector) {
    this.libAgentConnector.logFatal(msg);
    return;
  }
  this.logger.fatal(msg);
};

Logger.prototype.env = function (msg) {
  if (this.libAgentConnector) {
    this.libAgentConnector.logEnv(msg);
    return;
  }
  this.logger.info(msg);
};

Logger.prototype.setLibAgentConnector = function (libAgentConnector) {
  this.libAgentConnector = libAgentConnector;
};

Logger.prototype.isTraceEnabled = function () {
  return this.logger.isTraceEnabled();
};

Logger.prototype.isDebugEnabled = function () {
  return this.logger.isDebugEnabled();
};

Logger.prototype.isInfoEnabled = function () {
  return this.logger.isInfoEnabled();
};

Logger.prototype.isWarnEnabled = function () {
  return this.logger.isWarnEnabled();
};

Logger.prototype.isErrorEnabled = function () {
  return this.logger.isErrorEnabled();
};

Logger.prototype.isFatalEnabled = function () {
  return this.logger.isFatalEnabled();
};
