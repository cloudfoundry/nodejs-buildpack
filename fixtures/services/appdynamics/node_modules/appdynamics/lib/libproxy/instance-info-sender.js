/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
/* global process, require, exports */
'use strict';

function InstanceInfoSender(agent) {
  this.agent = agent;
}

exports.InstanceInfoSender = InstanceInfoSender;

InstanceInfoSender.prototype.init = function() {
  var self = this;

  self.agent.on('configUpdated', function() {
    var configManager = self.agent.configManager;
    var config = configManager.getConfigValue('instanceTrackingConfig');
    var instanceTracker = self.agent.instanceTracker;
    var isRunning;

    if (config) {
      isRunning = instanceTracker.enabled;
      instanceTracker.enabled = config.enabled;
      instanceTracker.customTypes = config.customTypes || instanceTracker.customTypes;

      if (config.enabled && !isRunning) {
        instanceTracker.startInstanceTracking(function(err, instanceCounts) {
          if (err) {
            return;
          }
          self.agent.backendConnector.proxyTransport.sendInstanceData(instanceCounts);
        });
      } else if (!config.enabled && isRunning) {
        instanceTracker.stopInstanceTracking();
      }
    }
  });
};
