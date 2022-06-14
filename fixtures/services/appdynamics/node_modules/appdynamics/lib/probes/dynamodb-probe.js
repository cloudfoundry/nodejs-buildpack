/*
 * Copyright (c) AppDynamics, Inc., and its affiliates
 * 2016
 * All Rights Reserved
 * THIS IS UNPUBLISHED PROPRIETARY CODE OF APPDYNAMICS, INC.
 * The copyright notice above does not evidence any actual or intended publication of such source code
 */
'use strict';

function DynamoDbProbe(agent) {
  this.agent = agent;
  this.packages = ['aws-sdk'];
}
exports.DynamoDbProbe = DynamoDbProbe;

DynamoDbProbe.prototype.attach = function(obj) {
  var self = this;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  self.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete  obj.__appdynamicsProbeAttached__;
      proxy.release(obj.DynamoDB);
    }
  });

  var proxy = self.agent.proxy;
  var profiler = self.agent.profiler;

  proxy.after(obj, 'DynamoDB', function(obj, args, ret) {
    var dynamodb = ret,
      supportedProperties = {
        'VENDOR': 'Amazon',
        'SERVICE': 'DynamoDB',
        'ENDPOINT': dynamodb.endpoint.href
      };

    proxy.around(dynamodb, 'makeRequest', function(obj, args,locals) {
      var command = args.length > 0 ? args[0] : undefined,
        params = (args.length > 1 && typeof(args) === 'object') ? args[1] : undefined;

      locals.time = profiler.time();

      locals.exitCall = profiler.createExitCall(locals.time, {
        exitType: 'EXIT_CUSTOM',
        exitSubType: 'Amazon Web Services',
        configType: 'Aws',
        supportedProperties: supportedProperties,
        command: profiler.sanitize(command),
        commandArgs: profiler.sanitize(JSON.stringify(params)),
        stackTrace: profiler.stackTrace()
      });

      if(!locals.exitCall) return;

      locals.methodHasCb = proxy.callback(args, -1, function(obj, args) {
        complete(args, locals);
      }, null, self.agent.thread.current());
    }, after, true);
  }, true);

  function complete(err, locals) {
    if (!locals.exitCall) return;
    if (!locals.time.done()) return;

    var error = self.agent.proxy.getErrorObject(err);
    profiler.addExitCall(locals.time, locals.exitCall, error);
  }

  function after(obj, args, ret, locals) {
    if (locals.methodHasCb)
      return;
    if (!ret || !ret.__appdynamicsIsPromiseResult__)
      complete(null, locals);
    else if (ret.error)
      complete(ret.error, locals);
    else {
      complete(null, locals);
    }
  }
};
