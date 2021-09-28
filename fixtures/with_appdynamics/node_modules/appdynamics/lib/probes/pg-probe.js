/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';


function PgProbe(agent) {
  this.agent = agent;

  this.packages = ['pg'];
}
exports.PgProbe = PgProbe;



PgProbe.prototype.attach = function(obj) {
  var self = this;

  if(obj.__appdynamicsProbeAttached__) return;
  obj.__appdynamicsProbeAttached__ = true;
  self.agent.on('destroy', function() {
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
      proxy.release(obj.Client.prototype.connect);
    }
  });

  var proxy = self.agent.proxy;
  var profiler = self.agent.profiler;

  function probe(obj) {
    if(obj.__appdynamicsProbeAttached__) return;
    obj.__appdynamicsProbeAttached__ = true;
    self.agent.on('destroy', function() {
      if(obj.__appdynamicsProbeAttached__) {
        delete obj.__appdynamicsProbeAttached__;
        proxy.release(obj.connect);
        proxy.release(obj.query);
      }
    });

    proxy.after(obj, 'connect', function(obj) {
      var supportedProperties = {
        'HOST': obj.host,
        'PORT': obj.port,
        'DATABASE': obj.database,
        'VENDOR': 'POSTGRESQL'
      };

      // Callback API
      proxy.around(obj, 'query', function(obj, args, locals) {
        locals.time = profiler.time();
        locals.exitCall = createExitCall(self.agent, obj, locals.time, args, supportedProperties);
        if (!locals.exitCall) return;

        locals.methodHasCb = proxy.callback(args, -1, function(obj, args) {
          complete(args, locals);
        }, null, self.agent.thread.current());
      }, function(obj, args, ret, locals) {
        // If has a callback, ignore
        if (locals.methodHasCb) return;
        if (!locals.exitCall) return;
        // NodeJS support Promises OOTB from Node v0.12 and above.
        // Rely on event API for versions below Node v0.12.
        // Fallback on event API when the return value is not a Promise.
        if (!proxy.isPromiseSupported() || (!ret || !ret.__appdynamicsIsPromiseResult__)) { // Evented API
          proxy.before(ret, 'on', function(obj, args) {
            var event = args[0];

            if(event !== 'end' && event !== 'error') return;

            var error;
            proxy.callback(args, -1, function(obj, args) {
              if(event === 'error')
                error = proxy.getErrorObject(args);

              complete(error, locals);
            }, null, self.agent.thread.current());
          }, false, false, self.agent.thread.current());
        }
        else if (ret.error) {
          complete(ret.error, locals);
        }
        else {
          complete(null, locals);
        }
      }, false, self.agent.thread.current());

      function complete(err, locals) {
        if (!locals.exitCall) return;
        if (!locals.time.done()) return;

        var error = proxy.getErrorObject(err);
        profiler.addExitCall(locals.time, locals.exitCall, error);
      }
    });
  }

  // Native, reinitialize probe
  proxy.getter(obj, 'native', function(obj, ret) {
    proxy.after(ret, 'Client', function(obj, args, ret) {
      probe(ret.__proto__);
    });
  });

  probe(obj.Client.prototype);
};

function createExitCall(agent, client, time, args, props) {
  var command = args.length > 0 ? args[0] : undefined;
  var params = args.length > 1 && Array.isArray(args[1]) ? args[1] : undefined;

  return agent.profiler.createExitCall(time, {
    exitType: 'EXIT_DB',
    supportedProperties: props,
    command: truncate(agent.profiler, command),
    commandArgs: agent.profiler.sanitize(params),
    user: client.user,
    stackTrace: agent.profiler.stackTrace(),
    isSql: true
  });
}

function truncate(profiler, str) {
  if(str && typeof(str) === 'object') {
    str = str.text;
  }

  return profiler.sanitize(str);
}
