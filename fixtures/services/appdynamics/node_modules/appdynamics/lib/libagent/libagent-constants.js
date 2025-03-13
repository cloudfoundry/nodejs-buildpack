/*
* Copyright (c) AppDynamics, Inc., and its affiliates
* 2016
* All Rights Reserved
*/
'use strict';

function define_constant(name, value) {
  Object.defineProperty(module.exports, name, {value: value, writable: false});
}

// these constants must also be in sync with the respective enums
// defined in agent.h
define_constant('TIME_ROLLUP_AVERAGE', 0);
define_constant('TIME_ROLLUP_SUM', 1);
define_constant('TIME_ROLLUP_CURRENT', 2);

define_constant('CLUSTER_ROLLUP_INDIVIDUAL', 0);
define_constant('CLUSTER_ROLLUP_COLLECTIVE', 1);

define_constant('HOLE_HANDLING_RATE_COUNTER', 0);
define_constant('HOLE_HANDLING_REGULAR_COUNTER', 1);

define_constant('AGGREGATOR_AVERAGE', 0);
define_constant('AGGREGATOR_ADVANCED_AVERAGE', 1);
define_constant('AGGREGATOR_SUM', 2);
// TODO: observation/percentile types when defined in libagent
