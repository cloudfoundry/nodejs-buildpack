/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';


function RedisProbe(agent) {
  this.agent = agent;

  this.packages = ['redis'];
}
exports.RedisProbe = RedisProbe;



RedisProbe.prototype.attach = function(obj) {
  var self = this;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  self.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
      proxy.release(obj.createClient);
    }
  });

  var proxy = self.agent.proxy;
  var profiler = self.agent.profiler;

  function proxyCommand(client) {
    var method = client.internal_send_command
      ? "internal_send_command"
      : "send_command";

    proxy.around(client, method, before, after);

    function before(obj, args, locals) {
      locals.time = profiler.time();
      var address = client.address || (client.host + ':' + client.port);
      var command, params;

      if (typeof(args[0]) == 'string') {
        command = args[0];
        params = args[1];
      } else {
        command = args[0].command;
        params = args[0].args;
      }

      var supportedProperties = {
        'SERVER POOL': address,
        'VENDOR': 'REDIS'
      };

      locals.exitCall = profiler.createExitCall(locals.time, {
        exitType: 'EXIT_CACHE',
        supportedProperties: supportedProperties,
        command: command,
        commandArgs: profiler.sanitize(params),
        stackTrace: profiler.stackTrace()
      });

      if (typeof(args[0]) === 'object' && typeof(args[0].callback) === 'function') {
        locals.methodHasCb = true;
        proxy.before(args[0], 'callback', function(obj, args) {
          complete(args, locals);
        }, false, false, self.agent.thread.current());
      } else if (typeof(args[2]) === 'function') {
        locals.methodHasCb = true;
        proxy.before(args, 2, function(obj, args) {
          complete(args, locals);
        }, false, false, self.agent.thread.current());
      } else {
        locals.methodHasCb = proxy.callback(args[1], -1, function(obj, args) {
          complete(args, locals);
        }, undefined, self.agent.thread.current());
      }
    }

    function after(obj, args, ret, locals) {
      if (locals.methodHasCb) return;
      if (!ret || !ret.__appdynamicsIsPromiseResult__)
        complete(null, locals);
      else if (ret.error)
        complete(ret.error, locals);
      else
        complete(null, locals);
    }

    function complete(err, locals) {
      if (!locals.exitCall) return;
      if (!locals.time.done()) return;

      var error = proxy.getErrorObject(err);
      profiler.addExitCall(locals.time, locals.exitCall, error);
    }
  }


  proxy.after(obj, 'createClient', function(obj, args, ret) {
    var client = ret;

    proxyCommand(client);
  });
};
