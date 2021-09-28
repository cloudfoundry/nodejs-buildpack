/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

/*
 * Metric object used for aggregating values and transferring
 * the aggregated metric to data sender. It can have three states:
 * - created/resetted
 * - initialized, i.e. accumulating values
 * - aggregated
 */

function convertOp(op) {
  if (op == 'sum' || op == 'inc')
    return 'TIME_SUM';
  else if (op == 'avg')
    return 'TIME_AVERAGE';
  else if (op == 'set')
    return 'CURRENT';
  else
    return op;
}

function Metric(agent, definition, isCustom) {
  this.agent = agent;
  this.metricName = definition.path;
  this.unit = definition.unit;
  this.op = convertOp(definition.op) || 'TIME_AVERAGE';
  this.clusterRollup = definition.clusterRollUp || 'INDIVIDUAL';
  this.holeHandling = definition.holeHandling || 'REGULAR_COUNTER';
  this.isCustom = isCustom;
  // used for avg aggregation only
  this._count = undefined;

  this.reset();

  // dynamic value property to support live metrics
  Object.defineProperty(this, 'value', {
    enumerable: true,
    get: function() {
      return (typeof(this._value) == 'function'
             ? this._value()
             : this._value);
    }
  });
}

exports.Metric = Metric;

Metric.prototype.reset = function() {
  var self = this;

  if (typeof(self._value) != 'function') {
    self._value = 0;
  }

  if(self.op === 'TIME_AVERAGE') {
    self._count = 0;
  } else {
    self._count = undefined;
  }
};

Metric.prototype.addValue = function(value) {
  var self = this;

  if(typeof(value) != 'number' && typeof(value) == 'function' && self.op != 'CURRENT') return;

  switch(self.op) {
  case 'TIME_AVERAGE':
    self._value = (value + self._value * self._count) / ++self._count;
    break;
  case 'TIME_SUM':
    self._value += value;
    break;
  case 'CURRENT':
    self._value = value;
    break;
  }

  if(self.isCustom) {
    self.agent.backendConnector.proxyTransport.sendCustomMetricData(self);
  }
};

Metric.prototype.clone = function() {
  var cln = new Metric(this.agent, {
    path: this.metricName,
    unit: this.unit,
    op: this.op,
    clusterRollup: this.clusterRollup,
    holeHandling: this.holeHandling
  });

  cln._value = this._value;

  if(this.op === 'TIME_AVERAGE') {
    cln._count = this._count;
  }

  return cln;
};




