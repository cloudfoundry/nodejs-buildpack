/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
/* global process, require, exports */
'use strict';

var fs = require('fs');
var path = require('path');
var cp = require('child_process');
var cluster = require('cluster');
var jre = require('appdynamics-jre');
var proxy = require('appdynamics-proxy');
var ExceptionHandlers = require('../core/exception-handlers.js').ExceptionHandlers;

/*
 * Autolaunching:
 * - The proxy is not launched if "proxyAutolaunchDisabled" option is set
 *   to false.
 * - Only master node autolaunches proxy, i.e. in case of cluster only
 *   one proxy instance is autolaunched.
 *
 * StartNodeRequest:
 * - In case of cluster, only workers send StartNodeRequest
 * - For the cases where cluster master cannot be identified there
 *   is a "monitorClusterMaster" option, which is false by default
 * - nodeIndex parameter is calculated by master node, which writes
 *   files for each worker to pick up their indexes from
 */

function ProxyLauncher(agent) {
  this.agent = agent;

  this.indexDir = undefined;

  this.processProcess = undefined;
  this.exceptionHandlers = undefined;
}
exports.ProxyLauncher = ProxyLauncher;

ProxyLauncher.prototype.init = function() {
  var self = this;
  self.indexDir = self.agent.tmpDir + '/index';

  self.agent.once('nodeIndex', function(nodeIndex) {
    self.exceptionHandlers = new ExceptionHandlers();
    self.exceptionHandlers.init(
      self.agent,
      function(){
        if (!cluster.isMaster) return;
        self.agent.logger.info("Terminating proxy");
        if (self.proxyProcess) {
          self.proxyProcess.kill('SIGTERM');
        }
        self.clearProxyControlDirectory();
        var proxyRuntimeDir = self.agent.proxyRuntimeDir;
        var proxyPidFile = path.join(proxyRuntimeDir, 'proxy.pid');
        self.clearProxyPID(proxyPidFile);
        self.agent.emit('proxyTerminated');
      },
      function(e){
        self.agent.logger.error('uncaughtException:');
        self.agent.logger.error(e);

        self.agent.backendConnector.proxyTransport.sendAppException(e);
      });

    if (!self.copyCaCertKeystore()) {
      self.agent.emit('launchProxy', nodeIndex);
    }
  });

  self.agent.on('launchProxy', function(nodeIndex, force) {
    if(!self.agent.opts.proxyAutolaunchDisabled) {
      if(cluster.isMaster) {
        self.agent.logger.info('launching proxy from master node ' + nodeIndex);
        self.startProxy(nodeIndex, force);
      }

      // wait for proxy to launch
      self.agent.timers.setTimeout(function() {
        self.agent.emit('proxyStarted', nodeIndex);
      }, 5000);
    }
    else {
      // need some time to be able to identify cluster master
      // by checking if any workers have been forked
      self.agent.timers.setTimeout(function() {
        self.agent.emit('proxyStarted', nodeIndex);
      }, 1000);
    }
  });
};

ProxyLauncher.prototype.start = function() {
  var self = this;
  var nodeIndex;

  if(cluster.isMaster) {
    nodeIndex = self.agent.opts.nodeIndex || 0;
    self.agent.emit('nodeIndex', nodeIndex);
  }
  else if ('pm_id' in process.env && !isNaN(process.env.pm_id)) {
    nodeIndex = Number(process.env.pm_id);
    self.agent.emit('nodeIndex', nodeIndex);
  }
  else {
    self.agent.timers.setTimeout(function() {
      self.readNodeIndex(function(nodeIndex) {
        if(nodeIndex !== null) {
          self.agent.emit('nodeIndex', nodeIndex);
        }
        else {
          self.agent.timers.setTimeout(function() {
            self.readNodeIndex(function(nodeIndex) {
              if(nodeIndex !== null) {
                self.agent.emit('nodeIndex', nodeIndex);
              }
              else {
                // return pid instead of index if indexing is not available,
                // e.g. this process is forked from a worker
                self.agent.emit('nodeIndex', process.pid);
              }
            });
          }, 4000);
        }
      });
    }, 1000);
  }
};

/* istanbul ignore next */
ProxyLauncher.prototype.readNodeIndex = function(callback) {
  var self = this;

  var callbackCalled = false;
  function callbackOnce(ret) {
    if(!callbackCalled) {
      callbackCalled = true;
      callback(ret);
    }
  }

  fs.exists(self.indexDir, function(exists) {
    if(!exists) return;

    fs.readdir(self.indexDir, function(err, indexFiles) {
      if(err) return self.agent.logger.error(err);

      indexFiles.forEach(function(indexFile) {
        var nodeIndex = parseInt(indexFile.split('.')[0]);
        if(!isNaN(nodeIndex)) {
          fs.readFile(self.indexDir + '/' + indexFile, function(err, pid) {
            if(err) return self.agent.logger.error(err);

            if(pid == process.pid) {
              callbackOnce(nodeIndex);
            }
          });
        }
      });
    });
  });

  self.agent.timers.setTimeout(function() {
    callbackOnce(null);
  }, 2000);
};

/* istanbul ignore next */
ProxyLauncher.prototype.isProxyRunning = function(proxyPidFile) {
  if (fs.existsSync(proxyPidFile)) {
    try {
      var proxyPid = fs.readFileSync(proxyPidFile, 'utf-8');
      process.kill(proxyPid, 0);
      return true;
    } catch (e) {
      // proxy process went away; clean up stale pid file
      fs.unlinkSync(proxyPidFile);
    }
  }

  return false;
};

/* istanbul ignore next */
ProxyLauncher.prototype.storeProxyPID = function(proxyPidFile, pid) {
  fs.writeFileSync(proxyPidFile, pid);
};

/* istanbul ignore next */
ProxyLauncher.prototype.clearProxyPID = function(proxyPidFile) {
  try {
    fs.unlinkSync(proxyPidFile);
  } catch (e) {
    this.agent.logger.warn('unable to delete proxy PID file: ' + e.message);
  }
};

ProxyLauncher.prototype.quote = function(path) {
  var self = this;
  if (!self.agent.isWindows || path.indexOf(' ') === -1) {
    return path;
  }

  return  '"' + path + '"';
};

ProxyLauncher.prototype.startProxy = function(nodeIndex, force) {
  var self = this;

  var opts = self.agent.opts;

  var proxyDir = proxy.dir;
  var proxyCommDir = self.agent.proxyCtrlDir;
  var proxyLogsDir = self.agent.proxyLogsDir;
  var proxyRuntimeDir = self.agent.proxyRuntimeDir;
  var proxyPidFile = path.join(proxyRuntimeDir, 'proxy.pid');

  if (!force && self.isProxyRunning(proxyPidFile)) {
    self.agent.logger.info('reusing existing proxy process');
    return;
  }

  self.clearProxyDirectories(function() {
    var proxyArgs = [
      '-d', self.quote(proxyDir),
      '-r', self.quote(proxyRuntimeDir),
      '-j', self.quote(jre.dir),
      '--'
    ];
    if (self.agent.isWindows) {
      proxyArgs.push(
        self.quote(proxyLogsDir),
        opts.proxyCommPort || 10101
      );
    } else {
      proxyArgs.push(
        self.quote(proxyCommDir),
        self.quote(proxyLogsDir),
        '-Dregister=false'
      );
    }

    var proxyOutput;

    self.agent.logger.info("proxyArgs = " + JSON.stringify(proxyArgs, null, " "));

    if(self.agent.opts.proxyOutEnabled) {
      proxyOutput = fs.openSync(proxyLogsDir + "/proxy.out", 'w');
      self.proxyProcess = self.spawn(proxyDir, proxyArgs, ['ignore', proxyOutput, proxyOutput]);
    }
    else {
      self.proxyProcess = self.spawn(proxyDir, proxyArgs, 'ignore');
    }

    self.storeProxyPID(proxyPidFile, self.proxyProcess.pid);

    self.agent.logger.info("Proxy logging to: " + proxyLogsDir);
    self.agent.logger.info("Proxy spawned!");
    self.proxyProcess.unref();
  });
};

/* istanbul ignore next */
ProxyLauncher.prototype.clearProxyDirectories = function(cb) {
  var proxyCommDir = this.agent.proxyCtrlDir;
  var proxyLogsDir = this.agent.proxyLogsDir;
  cp.exec('rm -rf ' + proxyCommDir + '/* ' + proxyLogsDir + '/*', cb);
};

/* istanbul ignore next */
ProxyLauncher.prototype.clearProxyControlDirectory = function() {
  cp.exec('rm -rf ' + this.agent.proxyCtrlDir + '/*');
};

/* istanbul ignore next */
ProxyLauncher.prototype.spawn = function(proxyDir, proxyArgs, input) {
  var proxyCmd = path.join(proxyDir, "runProxy");
  if (this.agent.isWindows) {
    proxyCmd = this.quote(proxyCmd + '.cmd');
  }
  this.agent.logger.debug("cmdLine = " + proxyCmd + " " + proxyArgs.join(" "));
  return cp.spawn(proxyCmd, proxyArgs, {
    detached: false, stdio: input, shell: !!this.agent.isWindows
  });
};

ProxyLauncher.prototype.copyCaCertKeystore = function() {
  if (!this.agent.opts.certificateFile) return;
  var self = this;
  var pathToJreIs = require.resolve('appdynamics-jre');
  var destKeystoreName = pathToJreIs.substring(0, pathToJreIs.indexOf('appdynamics-jre')) + 'appdynamics-jre/jre/lib/security/cacerts';
  var sourceStream = fs.createReadStream(self.agent.opts.certificateFile);
  sourceStream.on('error', function(error) {
    self.agent.logger.error('Error in reading from the certificate file' + error);
    return 'Failed';
  });
  var destinationStream = fs.createWriteStream(destKeystoreName);
  destinationStream.on('error', function(error) {
    self.agent.logger.error('Error in creating the destination keystore ' + error);
    return 'Failed';
  });
  sourceStream.pipe(destinationStream);
  self.agent.logger.info('SSL keystore copy completed');
  return;
};
