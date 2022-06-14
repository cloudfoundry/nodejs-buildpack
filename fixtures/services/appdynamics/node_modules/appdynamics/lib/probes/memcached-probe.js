/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';


var commands = [
  'get',
  'gets',
  'getMulti',
  'set',
  'replace',
  'add',
  'cas',
  'append',
  'prepend',
  'increment',
  'decrement',
  'incr',
  'decr',
  'del',
  'delete',
  'version',
  'flush',
  'samples',
  'slabs',
  'items',
  'flushAll',
  'statsSettings',
  'statsSlabs',
  'statsItems',
  'cachedump'
];


function MemcachedProbe(agent) {
  this.agent = agent;

  this.packages = ['memcached'];
}
exports.MemcachedProbe = MemcachedProbe;



MemcachedProbe.prototype.attach = function(obj) {
  var self = this;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  self.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
      commands.forEach(function(command) {
        proxy.release(obj[command]);
      });
    }
  });

  var proxy = self.agent.proxy;
  var profiler = self.agent.profiler;

  commands.forEach(function(command) {
    proxy.around(obj.prototype, command, function(obj, args, locals) {
      // ignore, getMulti will be called
      if(command === 'get' && Array.isArray(args[0])) return;
      var client = obj;
      var params = args;

      locals.time = profiler.time();
      var serverPool = self.getServerPool(client.servers);

      var supportedProperties = {
        'SERVER POOL': serverPool.join('\n'),
        'VENDOR': 'MEMCACHED'
      };

      locals.exitCall = profiler.createExitCall(locals.time, {
        exitType: 'EXIT_CACHE',
        supportedProperties: supportedProperties,
        command: command,
        commandArgs: profiler.sanitize(params),
        stackTrace: profiler.stackTrace()
      });

      if (!locals.exitCall) return;

      locals.serverPool = serverPool;
      locals.methodHasCb = proxy.callback(args, -1, function(obj, args) {
        complete(args, locals);
      }, null, self.agent.thread.current());
    }, after);
  });

  function complete(err, locals) {
    if (!locals.exitCall) return;
    if (!locals.time.done()) return;
    if (locals.methodHasCb && locals.serverPool.length === 0) return;

    var error = self.agent.proxy.getErrorObject(err);
    profiler.addExitCall(locals.time, locals.exitCall, error);
  }

  function after(obj,args, ret, locals) {
    if (locals.methodHasCb) return;
    if (!ret || !ret.__appdynamicsIsPromiseResult__)
      complete(null, locals);
    else if (ret.error)
      complete(ret.error, locals);
    else {
      complete(null, locals);
    }
  }
};

MemcachedProbe.prototype.getServerPool = function(servers) {
  var serverPool = [];

  if(typeof(servers) === 'string') {
    serverPool = [servers];
  }
  else if(typeof(servers) === 'object') {
    if(Array.isArray(servers)) {
      servers.forEach(function(server) {
        serverPool.push(server);
      });
    }
    else {
      for(var prop in servers) {
        serverPool.push(prop);
      }
    }
  }
  serverPool.sort();

  return serverPool;
};
