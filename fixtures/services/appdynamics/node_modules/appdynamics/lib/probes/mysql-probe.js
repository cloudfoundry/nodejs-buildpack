/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';
// Interceptor(Capture threadID) ---> Async Handler(Execute in the threadID context of the interceptor)
// In MySql probe. Query method is the main interceptor
// In Pool mode:
//  Query method interceptor on poolClient ---> aync query handling on mysql connection
//    ---> async handling of the poolClient query cb/promise
// In Non-pool mode:
//  Query method interceptor on mysqlConnection ---> async handling of the query's cb/promise


function MysqlProbe(agent) {
  this.agent = agent;

  this.packages = ['mysql', 'mysql2'];
}
exports.MysqlProbe = MysqlProbe;

function prepareExitCall(config, client, agent) {
  var context = agent.context;
  var profiler = agent.profiler;
  var proxy = agent.proxy;
  var thread = agent.thread;

  var supportedProperties = {
    'HOST': config.host,
    'PORT': config.port,
    'DATABASE': config.database,
    'VENDOR': 'MYSQL'
  };
  if (client.query.__appdynamicsProbeAttached__) return;

  proxy.around(client, 'query', function(obj, args, locals) {
    var command, params;
    var trace = profiler.stackTrace();

    function processQueryCb(obj, args) {
      complete(args, locals);
    }

    function createExitCall() {
      locals.exitCall = profiler.createExitCall(locals.time, {
        exitType: 'EXIT_DB',
        supportedProperties: supportedProperties,
        command: profiler.sanitize(command),
        commandArgs: profiler.sanitize(params),
        user: config.user,
        stackTrace: trace,
        isSql: true
      });
    }

    if (args[0] && args[0].__appdCbFnHolder) {
      thread.resume(args[0].__appdCbFnHolder.queryThreadId);
      locals.time = profiler.time();
      command = args[0].sql;
      params = args[0].values;
      locals.methodHasCb = true;
      createExitCall();
      proxy.before(args[0].__appdCbFnHolder, 'triggerOnCb', processQueryCb, false, false, args[0].__appdCbFnHolder.queryThreadId);
    } else {
      locals.time = profiler.time();
      command = args.length > 0 ? args[0] : undefined;
      params = args.length > 1 && Array.isArray(args[1]) ? args[1] : undefined;
      createExitCall();
      // mysql driver breaks CLS continuity so we must explicitly bind callback:
      args[args.length - 1] = context.bind(args[args.length - 1]);
      locals.methodHasCb = proxy.callback(args, -1, processQueryCb, null, thread.current());
    }
  }, after, false);

  client.query.__appdynamicsProbeAttached__ = true;

  function after(obj, args, ret, locals) {
    if (locals.methodHasCb) return;
    if (!ret || !ret.__appdynamicsIsPromiseResult__)
      complete(null, locals);
    else if (ret.error)
      complete(ret.error, locals);
    else {
      complete(null, locals);
    }
  }

  function complete(err, locals) {
    if (!locals.exitCall) return;
    if (!locals.time.done()) return;

    var error = proxy.getErrorObject(err);
    profiler.addExitCall(locals.time, locals.exitCall, error);
  }
}

MysqlProbe.prototype.attach = function(obj, name) {
  var self = this;
  var proxy = self.agent.proxy;
  var profiler = self.agent.profiler;
  var cmds = ['createClient', 'createConnection', 'createPool'];
  var mysqlConnection;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  self.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
      cmds.forEach(function(createCmd) {
        if (obj[createCmd])
          proxy.release(obj[createCmd]);
      });
      if (mysqlConnection)
        delete mysqlConnection.query.__appdynamicsProbeAttached__;
    }
  });

  if (name === 'mysql2' && obj.Connection && obj.Connection.createQuery) {
    proxy.after(obj.Connection, 'createQuery', function(obj, args, ret) {
      if (typeof (ret.then) === 'function') {
        // Disable mysql2's error logging when trying to treat
        // non-promise query as promise
        ret.then = ret.catch = null;
      }
    });
  }

  function queryCbFnThreadCtxtSwitch(queryArgs, cbIndex, queryLocals) {
    // mysql driver breaks CLS continuity so we must explicitly bind callback:
    queryArgs[cbIndex] = self.agent.context.bind(queryArgs[cbIndex]);
    proxy.before(queryArgs, cbIndex, function() {
      queryLocals.triggerOnCb();
    }, false, false);
  }

  cmds.forEach(function(createCmd) {
    proxy.after(obj, createCmd, function(obj, args, ret) {
      if (createCmd === 'createPool') {
        proxy.around(ret, 'getConnection', function(obj, args, locals) {
          locals.methodHasCb = proxy.callback(args, -1, function(obj, args) {
            if (args[0]) return;
            if (args[1]) {
              mysqlConnection = args[1];
              var config = mysqlConnection.config;
              prepareExitCall(config, mysqlConnection, self.agent);
            }
          });
        }, function(obj, args, ret, locals) {
          if (locals.methodHasCb) return;
          if (!ret || !ret.__appdynamicsIsPromiseResult__) return;
          if (ret.error) return;
          else {
            mysqlConnection = ret.data;
            var config = mysqlConnection.config;
            prepareExitCall(config, mysqlConnection, self.agent);
          }
        });

        proxy.around(ret, 'query', function(obj, args, locals) {
          locals.triggerOnCb = function() {
            self.agent.logger.debug('Placeholder function');
          };
          locals.queryThreadId = profiler.time().threadId;
          if (typeof args[0] === 'function') {
            queryCbFnThreadCtxtSwitch(args, '0', locals);
          } else if (typeof args[1] === 'function') {
            queryCbFnThreadCtxtSwitch(args, '1', locals);
          } else {
            queryCbFnThreadCtxtSwitch(args, '2', locals);
          }
        }, function(obj, args, query, locals) {
          query.__appdCbFnHolder = locals;
        });
      } else {
        mysqlConnection = ret;
        var config = (createCmd === 'createClient' ? mysqlConnection : mysqlConnection.config);
        if (!config) {
          return;
        }
        prepareExitCall(config, mysqlConnection, self.agent);
      }
    });
  });
};
