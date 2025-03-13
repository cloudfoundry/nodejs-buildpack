/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var PERIOD = 60e3; // 60 seconds

function InstanceTracker(agent) {
  this.agent = agent;
  this.enabled = false;
  this.customTypes = [];
  this.intervalHandle = null;
  this.lastSnapshotTimestamp = 0;
}

exports.InstanceTracker = InstanceTracker;

InstanceTracker.prototype.init = function() {};

InstanceTracker.prototype.startInstanceTracking = function(callback) {
  this.agent.logger.info('Object instance tracking enabled.');
  this.intervalHandle = this.agent.timers.setInterval(
    this.collectInstanceCounts.bind(this, callback),
    PERIOD);
  this.collectInstanceCounts(callback); // don't wait on setInterval for first snapshot
};

InstanceTracker.prototype.stopInstanceTracking = function() {
  this.agent.logger.debug('Object instance tracking disabled.');
  this.agent.timers.clearInterval(this.intervalHandle);
};

InstanceTracker.prototype.collectInstanceCounts = function(callback) {
  var self = this;

  if (Date.now() - self.lastSnapshotTimestamp < PERIOD/2) {
    self.agent.logger.debug('Skipping instance tracking snapshot (throttled)');
    return;
  }

  self.agent.logger.debug('Taking instance tracking snapshot');
  self.agent.cpuProfiler.getHeapSnapshot(self.customTypes, function(err, json) {
    if (err) {
      self.agent.logger.debug('Skipping instance tracking snapshot: ' + err);
      return;
    }

    try {
      var instanceCounts = JSON.parse(json);

      if (callback) {
        callback(null, instanceCounts);
      }
      else {
        self.agent.logger.debug('Missing callback for instance tracking!');
      }

      self.agent.logger.debug('Completed instance tracking snapshot');
    } catch (e) {
      self.agent.logger.warn('Failed to collect instance tracking data: ' +
                              e + '\n' + json);
    }

    self.lastSnapshotTimestamp = Date.now();
  });
};

