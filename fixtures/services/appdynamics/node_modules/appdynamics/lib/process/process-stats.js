/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

/* istanbul ignore next -- needs integratoin tests */
(function() {

/*
 * Sending process related data as metrics.
 */

  function ProcessStats(agent) {
    this.agent = agent;

    this.lastCpuTime = undefined;
  }
  exports.ProcessStats = ProcessStats;


  ProcessStats.prototype.init = function() {
    var self = this;

    self.agent.metricsManager.addMetric(self.agent.metricsManager.NODE_RSS,
                                      function() { return process.memoryUsage().rss / 1048576; });
    self.agent.metricsManager.addMetric(self.agent.metricsManager.HEAP_USAGE,
                                      function () { return process.memoryUsage().heapUsed / 1048576; });
    self.agent.metricsManager.addMetric(self.agent.metricsManager.HEAP_TOTAL,
                                      function () { return process.memoryUsage().heapTotal / 1048576; });

    self.lastTimestamp = Date.now();
    self.lastCpuTime = self.agent.system.cputime() || 0;
    self.agent.metricsManager.addMetric(self.agent.metricsManager.CPU_PERCENT_BUSY, function() {
      var timeNow = Date.now();
      var cpuTime = self.agent.system.cputime() || 0;
      var result = ((cpuTime - self.lastCpuTime) / 1000) / (timeNow - self.lastTimestamp) * 100;

      self.lastTimestamp = timeNow;
      self.lastCpuTime = cpuTime;
      return result;
    });
  };

})();
