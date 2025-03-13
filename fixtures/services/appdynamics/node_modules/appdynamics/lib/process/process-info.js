/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

/* istanbul ignore next -- OS interface */
(function() {

  var os = require('os');

/*
 * Sends process information to the server every minute
 */

  function ProcessInfo(agent) {
    this.agent = agent;
    this.isv0_8 = process.version.match(/^v0\.8\./);
  }

  exports.ProcessInfo = ProcessInfo;

  ProcessInfo.prototype.init = function() {
  };

  ProcessInfo.prototype.fetchInfo = function() {
    var self = this, key, subkey, info = {};

    info['Application name'] = self.agent.appName;
    info['Node Version'] = process.version;
    info['Exec Path'] = process.execPath;
    info['NODE_ENV'] = process.env.NODE_ENV;

    try {
      info.Hostname = os.hostname();
    } catch(err) {
      self.logError(err);
    }

    try {
      info['OS type'] = os.type();
    }
    catch(err) {
      self.logError(err);
    }

    try {
      info.Platform = os.platform();
    }
    catch(err) {
      self.logError(err);
    }

    try {
      info.Architecture = os.arch();
    }
    catch(err) {
      self.logError(err);
    }

    try {
      var cpus = os.cpus();
      info['CPU Model'] = cpus[0].model;
      info['CPU Speed'] = cpus[0].speed;
      info['CPU Cores'] = cpus.length;
    }
    catch(err) {
      self.logError(err);
    }

    try {
      info['Node arguments'] = process.argv.join(' ');
    }
    catch(err) {
      self.logError(err);
    }

    try {
      info['Node PID'] = process.pid;
    }
    catch(err) {
      self.logError(err);
    }

    try {
      for (key in process.versions) {
        info['Module versions: ' + key] = process.versions[key];
      }
    }
    catch(err) {
      self.logError(err);
    }

    info['AppDynamics version'] = self.agent.version;

    for (key in self.agent.opts) {
      if (key == 'accountAccessKey' || key == 'logging') {
        continue;
      } else if (key == 'analytics') {
        for (subkey in self.agent.opts[key]) {
          info['AppDynamics options: analytics: ' + key] = self.agent.opts[key][subkey];
        }
      } else {
        info['AppDynamics options: ' + key] = self.agent.opts[key];
      }
    }

    return info;
  };

  ProcessInfo.prototype.logError = function(err) {
    this.agent.logger.error(err);
  };

})();
