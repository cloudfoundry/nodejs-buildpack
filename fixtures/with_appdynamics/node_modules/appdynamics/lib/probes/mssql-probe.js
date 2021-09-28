/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

function MssqlProbe(agent) {
  this.agent = agent;

  this.packages = ['mssql'];
}

exports.MssqlProbe = MssqlProbe;

MssqlProbe.prototype.attach = function(obj) {
  var self = this;
  var proxy = self.agent.proxy;
  var profiler = self.agent.profiler;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  self.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
      proxy.release(obj.Request.prototype.query);
    }
  });

  function complete(err, locals) {
    if (!locals.exitCall) return;
    if (!locals.time.done()) return;

    profiler.addExitCall(locals.time, locals.exitCall, err);
  }

  proxy.around(obj.Request.prototype, 'query',
    function before(obj, args, locals) {
      var params = [];
      if (obj.parameters) {
        for (var param in obj.parameters) {
          params.push(obj.parameters[param].value);
        }
      }
      //When creating MsSQL query through typeorm, typeorm uses db connection obtained from pool.transaction to create the request. 
      //This request is used to create the query. Due to this, config is found in obj.parent.parent instead of obj.parent
      self.createExitCall(obj.parent.config || obj.parent.parent.config, args && args[0], params, locals);

      locals.hasCallback = proxy.callback(args, -1, function(obj, args) {
        var error = proxy.getErrorObject(args);
        complete(error, locals);
      });
    },
    function after(obj, args, ret, locals) {
      if (!locals.hasCallback) {
        complete(ret.error || undefined, locals);
      }
    }
  );
};

MssqlProbe.prototype.createExitCall = function(config, command, params, locals) {
  var agent = this.agent;
  var profiler = agent.profiler;

  var supportedProperties = {
    'HOST': config.host || config.server || 'localhost',
    'PORT': config.port && config.port.toString() || '1433',
    'DATABASE': config.database,
    'VENDOR': 'MSSQL'
  };

  var trace = profiler.stackTrace();

  locals.time = profiler.time();
  locals.exitCall = profiler.createExitCall(locals.time, {
    exitType: 'EXIT_DB',
    supportedProperties: supportedProperties,
    command: profiler.sanitize(command),
    commandArgs: profiler.sanitize(params),
    user: config.user,
    stackTrace: trace,
    isSql: true
  });
};