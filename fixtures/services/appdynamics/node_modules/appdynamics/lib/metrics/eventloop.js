/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

function EventLoopMetrics(agent) {
  this.agent = agent;
}

exports.EventLoopMetrics = EventLoopMetrics;

EventLoopMetrics.prototype.init = function() {
  var self = this;

  var metricsManager = this.agent.metricsManager;

  var eventLoopStats;

  /* istanbul ignore else */
  if (process.env.NODE_ENV == 'appd_test') {
    eventLoopStats = function() {
      return [];
    };
  }
  else {
    eventLoopStats = function() {
      return self.agent.appdNative.getEventLoopInfo();
    };
  }

  metricsManager.addMetric(metricsManager.EVENT_LOOP_TICK_COUNT, function() {
    return eventLoopStats()[0];
  });
  metricsManager.addMetric(metricsManager.EVENT_LOOP_TICK_LENGTH_MIN, function() {
    return eventLoopStats()[1];
  });
  metricsManager.addMetric(metricsManager.EVENT_LOOP_TICK_LENGTH_MAX, function() {
    return eventLoopStats()[2];
  });
  metricsManager.addMetric(metricsManager.EVENT_LOOP_TICK_LENGTH_AVG, function() {
    return eventLoopStats()[3] / eventLoopStats()[0];
  });
  metricsManager.addMetric(metricsManager.EVENT_LOOP_IO_TIME_AVG, function() {
    return eventLoopStats()[4] / eventLoopStats()[0];
  });
};

EventLoopMetrics.prototype.reset = function() {
  this.agent.appdNative.resetEventLoopInfo();
};
