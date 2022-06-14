/*
 * Copyright (c) AppDynamics, Inc., and its affiliates
 * 2016
 * All Rights Reserved
 * THIS IS UNPUBLISHED PROPRIETARY CODE OF APPDYNAMICS, INC.
 * The copyright notice above does not evidence any actual or intended publication of such source code
 */

'use strict';

function NSolidProbe(agent) {
  this.agent = agent;
  this.packages = ['nsolid'];
}

exports.NSolidProbe = NSolidProbe;

NSolidProbe.prototype.attach = function(obj) {
  var self = this;
  var agent = self.agent;
  self.nsolid = obj;

  if (!agent.nsolidEnabled) return;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
    }
  });

  // setup N|Solid metrics and begin collection
  self.startMetricCollection();

  // expose async activity for process snapshot use
  agent.asyncActivity = this.fetchAsyncActivity.bind(this);
};

var METRIC_REPORTERS = {
  'NSOLID_UPTIME': function (metrics) { return metrics.uptime; },
  'NSOLID_HEAPTOTAL': function (metrics) { return metrics.heapTotal / 1024; },
  'NSOLID_ACTIVEREQUESTS': function (metrics) { return metrics.activeRequests; },
  'NSOLID_ACTIVEHANDLES': function (metrics) { return metrics.activeHandles; },
  'NSOLID_TOTALAVAILABLESIZE': function (metrics) { return metrics.totalAvailableSize / 1024; },
  'NSOLID_HEAPSIZELIMIT': function (metrics) { return metrics.heapSizeLimit / 1024; },
  'NSOLID_FREEMEM': function (metrics) { return metrics.freeMem / 1024; },
  'NSOLID_SYSTEMUPTIME': function (metrics) { return metrics.systemUptime; },
  'NSOLID_LOAD1M': function (metrics) { return metrics.load1m; },
  'NSOLID_LOAD5M': function (metrics) { return metrics.load5m; },
  'NSOLID_LOAD15M': function (metrics) { return metrics.load15m; },
  'NSOLID_IDLEPERCENT': function(metrics) { return metrics.loopIdlePercent; },
  'NSOLID_ESTIMATEDLAG': function(metrics) { return metrics.loopEstimatedLag; },
  'NSOLID_TURNRATE': function(metrics) { return metrics.loopsPerSecond; },
  'NSOLID_AVGTASKS': function(metrics) { return metrics.loopAvgTasks; },
  'NSOLID_TURNCOUNT': function(metrics) { return metrics.loopTotalCount; },
  'NSOLID_CPUSYSTEM': function(metrics) { return metrics.cpuSystemPercent; },
  'NSOLID_CPUUSER': function(metrics) { return metrics.cpuUserPercent; },
  'NSOLID_CSINVOLUNTARTY': function(metrics) { return metrics.ctxSwitchInvoluntaryCount; },
  'NSOLID_CSVOLUNTARTY': function(metrics) { return metrics.ctxSwitchVoluntaryCount; },
  'NSOLID_IPCRECEIVED': function(metrics) { return metrics.ipcReceivedCount; },
  'NSOLID_IPCSENT': function(metrics) { return metrics.ipcSentCount; },
  'NSOLID_SIGNALSRECEIVED': function(metrics) { return metrics.signalCount; },
  'NSOLID_PAGEFAULTSSOFT': function(metrics) { return metrics.pageFaultSoftCount; },
  'NSOLID_PAGEFAULTSHARD': function(metrics) { return metrics.pageFaultHardCount; },
  'NSOLID_SWAPCOUNT': function(metrics) { return metrics.swapCount; },
  'NSOLID_BLOCKINPUTS': function(metrics) { return metrics.blockInputOpCount; },
  'NSOLID_BLOCKOUTPUTS': function(metrics) { return metrics.blockOutputOpCount; },
  'NSOLID_GCTOTAL': function(metrics) { return metrics.gcCount; },
  'NSOLID_GCFULL': function(metrics) { return metrics.gcFullCount; },
  'NSOLID_GCMAJOR': function(metrics) { return metrics.gcMajorCount; },
  'NSOLID_GCFORCED': function(metrics) { return metrics.gcForcedCount; },
  'NSOLID_GCCPU': function(metrics) { return metrics.gcCpuPercent; },
  'NSOLID_GCTIME99': function(metrics) { return metrics.gcDurUs99Ptile; },
  'NSOLID_GCTIMEMEDIAN': function(metrics) { return metrics.gcDurUsMedian; }
};

NSolidProbe.prototype.startMetricCollection = function() {
  var self = this, agent = self.agent, nsolid = self.nsolid, metric;

  self.metrics = {};
  for (metric in METRIC_REPORTERS) {
    self.metrics[metric] = agent.metricsManager.createMetric(agent.metricsManager[metric]);
  }

  agent.timers.setInterval(collectMetrics, 30000);
  collectMetrics();

  var cpuSpeedRecorded = false;
  function collectMetrics() {
    nsolid.metrics(function (err, results) {
      if (err) {
        agent.logger.warn('failed to retrieve N|Solid metrics: ' + err.message);
        return;
      }

      for (metric in METRIC_REPORTERS) {
        self.metrics[metric].addValue(METRIC_REPORTERS[metric](results));
      }

      // CPU speed is reported as a metric but, since it's constant,
      // we report it as agent metadata instead
      if (!cpuSpeedRecorded) {
        cpuSpeedRecorded = true;
        agent.meta.push({ name: 'CPU Speed', value: results.cpuSpeed });
      }
    });

    self.collectMetrics = collectMetrics; // expose for integration tests
  }
};

NSolidProbe.prototype.fetchAsyncActivity = function(cb) {
  var self = this, agent = self.agent, nsolid = self.nsolid;

  if (!agent.nsolidEnabled) {
    return process.nextTick(function() {
      cb(null, null);
    });
  }

  nsolid.asyncActivity(function (err, results) {
    var data = [];

    if (err) return cb(err);

    ['handles', 'requests', 'pending'].forEach(function(category) {
      results[category].forEach(function(entry) {
        // if (entry && entry.fn && entry.location) {
        if (entry) {
          var activity = {
            category: category,
            type: entry.type,
            location: {
              name: (entry.anonymous ? '(anonymous)' : entry.name),
            },
            details: entry.details,
            metadata: entry.metadata
          };
          if (entry.location) {
            activity.location.file = entry.location.file;
            activity.location.line = entry.location.line;
            activity.location.column = entry.location.column;
          }
          data.push(activity);
        }
      });
    });

    cb(null, data);
  });
};
