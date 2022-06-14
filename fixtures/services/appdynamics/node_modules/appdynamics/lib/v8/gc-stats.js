/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var BYTES_TO_MB = 1 / (1024 * 1024);

function GCStats(agent) {
  this.agent = agent;
}

exports.GCStats = GCStats;

GCStats.prototype.init = function() {
  /* istanbul ignore next */
  if (!this.agent.appdNative) return;

  var mm = this.agent.metricsManager;
  this.numFullGC = mm.createMetric(mm.GC_FULL);
  this.numIncGC = mm.createMetric(mm.GC_INC);
  this.sizeChange = mm.createMetric(mm.HEAP_SIZE_CHANGE);
  this.lastUsedHeapSize = 0;

  this.agent.appdNative.afterGC(this.onAfterGC.bind(this));
};


GCStats.prototype.onAfterGC = function(gcType, gcFlags, usedHeapSize) {
  switch (gcType) {
  case 'kGCTypeMarkSweepCompact':
    this.numFullGC.addValue(1);
    break;
  case 'kGCTypeScavenge':
    this.numIncGC.addValue(1);
    break;
  }

  this.sizeChange.addValue((usedHeapSize - this.lastUsedHeapSize) * BYTES_TO_MB);
  this.lastUsedHeapSize = usedHeapSize;
};
