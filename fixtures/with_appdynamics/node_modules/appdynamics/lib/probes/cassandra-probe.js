/*
 * Copyright (c) AppDynamics, Inc., and its affiliates
 * 2016
 * All Rights Reserved
 * THIS IS UNPUBLISHED PROPRIETARY CODE OF APPDYNAMICS, INC.
 * The copyright notice above does not evidence any actual or intended publication of such source code
 */

'use strict';

function CassandraProbe(agent) {
  this.agent = agent;
  this.packages = ['cassandra-driver'];
}

exports.CassandraProbe = CassandraProbe;

CassandraProbe.prototype.attach = function(obj) {
  var self = this;
  var proxy = self.agent.proxy;
  var profiler = self.agent.profiler;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  self.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete  obj.__appdynamicsProbeAttached__;
      proxy.release(obj.Client);
      proxy.release(obj.Client.prototype.batch);
      proxy.release(obj.Client.prototype.eachRow);
      proxy.release(obj.Client.prototype.execute);
    }
  });

  // only way to determine cluster name is to query the database,
  // so we need to hook the (undocumented) 'connected' event once
  // the driver is instantiated. NOTE: Client is a constructor
  // so we must fixup the prototype chain after hooking it.
  var Client = obj.Client;
  proxy.after(obj, 'Client', function(obj) {
    obj.once('connected', onconnected);
    function onconnected(err) {
      if (err) {
        // connection was unsuccessful; try again on next connect
        return obj.once('connected', onconnected);
      }
      // we have a connection; get the cluster name
      obj.execute("SELECT cluster_name FROM system.local WHERE key = 'local'",
        function (err, results) {
          if (!err && results && results.rows && results.rows[0]) {
            obj.cluster = results.rows[0].cluster_name;
          }
          if (!obj.cluster) {
            self.agent.logger.warn(
              'unable to determine Cassandra cluster name; ' +
              'Cassandra exit call discovery will be disabled');
          }
        }
      );
    }
  });
  obj.Client.prototype = Client.prototype;

  // execute() and batch() return a promise if they don't
  // receive a callback so we need both before and after
  // advice to handle both cases
  proxy.around(obj.Client.prototype, ['execute', 'batch'], before, after);

  // eachRow() doesn't return a promise if no callback is given,
  // so we don't need after advice to handle that; but we also
  // have to be careful about detecting the optional completion
  // callback as it's preceded by another (non-optional)
  // function argument
  proxy.before(obj.Client.prototype, 'eachRow',
    function(obj, args, locals) {
      var missingCallback = false;
      var cb1 = args[args.length - 2];
      var cb2 = args[args.length - 1];

      if (!(typeof(cb1) == 'function' && typeof(cb2) == 'function')) {
        missingCallback = true;
      }

      // handle the exit call
      before.call(this, obj, args, locals, missingCallback);
    }
  );

  // capture exit call start and, if an operation completion
  // callback is available, hook it to end the exit call
  // when the operation completes. for operations that can
  // return a promise, see after()
  function before(obj, args, locals, noCallback) {
    var client = obj;
    var host = client.hosts && client.hosts.values()[0];
    var command = args[0];

    if (!(host && client.cluster)) return;

    var supportedProperties = {
      'CLUSTER NAME': client.cluster,
      'KEYSPACE': client.keyspace,
      'HOST': host.address.split(':')[0],
      'PORT': host.address.split(':')[1],
      'DATA CENTER': host.datacenter,
      'RACK': host.rack
    };

    locals = locals || {};
    locals.time = profiler.time();
    locals.exitCall = createExitCall(client, profiler, locals.time, command, supportedProperties);

    if (noCallback) {
      // special case for beforeEach() without callback;
      // no promise is returned and there is no event we
      // can hook or other means to get called back when
      // the query completes, so we just complete the exit
      // call right away
      complete(args[0], locals);
    } else {
      locals.methodHasCb = self.agent.proxy.callback(args, -1, function(obj, args) {
        // the operation completed via a callback; end the exit call
        complete(args[0], locals);
      }, null, self.agent.thread.current());
    }
  }

  // end the exit call via promise resolution for operations that
  // can return a promise; ret is assumed to be a promise if it is
  // 'promise-like', i.e. it has a `then` property that is callable.
  function after(obj, args, ret, locals) {
    if (locals.methodHasCb) return;
    if (!ret || !ret.__appdynamicsIsPromiseResult__)
      return;
    if (ret.error)
      complete(ret.error, locals);
    else {
      complete(null, locals);
    }
  }

  function complete(err, locals) {
    if (!locals.exitCall) return;
    if (!locals.time.done()) return;

    var error = self.agent.proxy.getErrorObject(err);
    profiler.addExitCall(locals.time, locals.exitCall, error);
  }
};

function createExitCall(client, profiler, time, command, props) {
  var stringifiedCommand = command;
  if (typeof command != 'string' && command.length) {
    stringifiedCommand = [];
    for(var i = 0; i < command.length; i++) {
      stringifiedCommand[i] = command[i].query;
      if (typeof command[i] === 'string') {
        stringifiedCommand[i] = command[i];
      }
    }
  }
  return profiler.createExitCall(time, {
    exitType: 'EXIT_CUSTOM',
    exitSubType: 'Cassandra CQL',
    configType: 'Cassandra',
    supportedProperties: props,
    command: profiler.sanitize(stringifiedCommand),
    user: client.options.authProvider && client.options.authProvider.username,
    stackTrace: profiler.stackTrace(),
    isSql: false
  });
}
