/*
 * Copyright (c) AppDynamics, Inc., and its affiliates
 * 2017
 * All Rights Reserved
 * THIS IS UNPUBLISHED PROPRIETARY CODE OF APPDYNAMICS, INC.
 * The copyright notice above does not evidence any actual or intended publication of such source code
 */

'use strict';
function ExpressProbe(agent) {
  this.agent = agent;
  this.packages = ['express'];
}

function copyPropsToWrapper(src, destination) {
  Object.getOwnPropertyNames(src).forEach(function forEachOwnPropertyName(name) {
    // Copy descriptor
    var descriptor = Object.getOwnPropertyDescriptor(src, name);
    Object.defineProperty(destination, name, descriptor);
  });
}

var wrappedExpressFunction, routesArray = [];

exports.ExpressProbe = ExpressProbe;

ExpressProbe.prototype.attach = function(obj) {
  var self = this,
    proxy = self.agent.proxy;

  if(obj.__appdynamicsProbeAttached__) {
    return obj.__appdynamicsWrapperFn;
  }
  obj.__appdynamicsProbeAttached__ = true;

  var attachRouterHandler = function (router) {
    if (!router.__appdynamicsProbeAttached__) {
      // Now for this routerObj, monitor its stack property.
      // For every push into the stack, intercept it and get hold
      // of the Layer object.
      self.interceptLayersHandleErrorMethod(router);
      proxy.after(router, 'route', function(routerObj, routFnArgs, routeObj) {
        self.interceptLayersHandleErrorMethod(routeObj);
      });
      router.__appdynamicsProbeAttached__ = true;
    }
  };

  // Create a wrapper function
  wrappedExpressFunction = function() {
    var appObj = obj(arguments), vInstrumentDone = false;

    // For v3.x
    if (!vInstrumentDone && appObj._router) {
      vInstrumentDone = true;
      self.interceptLayersHandleErrorMethod(appObj);
      proxy.before(appObj._router, 'route', function(routerObj, routeArgs) {
        if (routeArgs && routeArgs.length >= 2) {
          for (var i = 2; i < routeArgs.length; i++) {
            proxy.before(routeArgs, i, function(callbacksObj, fnArgs) {
              var expressReqError = fnArgs && fnArgs.length === 4 ? fnArgs[0] : undefined;
              expressReqError && self.attachErrorToTransaction(expressReqError);
            });
          }
        }
      });
    }

    // For v5.X
    if (!vInstrumentDone && !appObj.lazyrouter && appObj.router) {
      vInstrumentDone = true;
      attachRouterHandler(appObj.router);
      routesArray.push(appObj.router);
    }

    // For v4.x
    if (!vInstrumentDone) {
      proxy.after(appObj, ['lazyrouter'], function() {
        routesArray.push(appObj._router);
        attachRouterHandler(appObj._router);
      });
    }

    return appObj;
  };
  // For integration testing
  wrappedExpressFunction.testName = 'wrapperFunction';

  // Copy all the properties of the object to this wrapper function
  copyPropsToWrapper(obj, wrappedExpressFunction);

  self.agent.on('destroy', function() {
    if(wrappedExpressFunction.__appdynamicsProbeAttached__) {
      delete wrappedExpressFunction.__appdynamicsProbeAttached__;
      proxy.release(wrappedExpressFunction.Router);
      proxy.release(wrappedExpressFunction.Route);
    }
    if(obj.__appdynamicsProbeAttached__) {
      delete obj.__appdynamicsProbeAttached__;
      delete obj.__appdynamicsWrapperFn;
    }
    for(var i = 0; i < routesArray.length; i++)
      proxy.release(routesArray[i]);
  });

  // Instrument Router and Route methods of express.
  proxy.after(wrappedExpressFunction, ['Router', 'Route'], function(expressObj, routerArgs, routerObj) {
    // Make sure these objects are instrumented only once.
    // Multiple calls to these function objects should
    // not rewrap the instrumentation.
    if (!routerObj.__appdynamicsProbeAttached__) {
      // Now for this routerObj, monitor its stack property.
      // For every push into the stack, intercept it and get hold
      // of the Layer object
      self.interceptLayersHandleErrorMethod(routerObj);

      if (routerObj.name === 'router') {
        proxy.after(routerObj, 'route', function (routerObj, routFnArgs, routeObj) {
          self.interceptLayersHandleErrorMethod(routeObj);
        });
      }
      routerObj.__appdynamicsProbeAttached__ = true;
    }
  });
  obj.__appdynamicsWrapperFn = wrappedExpressFunction;
  return wrappedExpressFunction;
};

ExpressProbe.prototype.attachErrorToTransaction = function(err) {
  var profiler = this.agent.profiler,
    transaction = profiler.transactions[profiler.time().threadId];
  if (transaction) {
    transaction.error = err;
  }
};

ExpressProbe.prototype.interceptLayersHandleErrorMethod = function(obj) {
  var self = this, proxy = self.agent.proxy;
  proxy.before(obj.stack, 'push', function(routerObj, stackArgs) {
    var layerObj = stackArgs && stackArgs.length && stackArgs[0];
    if (layerObj) {
      // loopback manages middleware ordering with a custom sort routine
      // that uses equality comparison on the handle property; because
      // we're wrapping it, we need to preserve the original value
      // where loopback can find it
      var handler = layerObj.handle;

      // capture errors thrown by express route handlers; we have to be
      // carefull to preserve the arity of the handler here, as express
      // uses the arity to determine how to invoke the handler
      layerObj.handle = layerObj.handle.length === 4
        ? function(err, req, res, next) {
          try {
            return handler(err, req, res, next);
          } catch (e) {
            self.attachErrorToTransaction(e);
            throw e;
          }
        }
        : function(req, res, next) {
          try {
            return handler(req, res, next);
          } catch (e) {
            self.attachErrorToTransaction(e);
            throw e;
          }
        };

      // also capture errors passed to error handlers, which may or may not
      // propogate; errors thrown from route handlers have already been
      // captured; this picks up errors passed via next()
      proxy.before(layerObj, 'handle', function(layerObject, fnArgs) {
        var expressReqError = fnArgs && fnArgs.length === 4 ? fnArgs[0] : undefined;
        expressReqError && self.attachErrorToTransaction(expressReqError);
      });
      layerObj.handle.__appdOriginal = handler; // property loopback will find

      // we can't use proxy.before()'s support for preserving own properties
      // from the proxied function on the proxy function as we've already
      // replaced the original with the arity-preserving wrapper above; copy
      // own properties across here instead
      var methodProps = Object.getOwnPropertyNames(handler);
      for(var i = 0; i < methodProps.length; i++) {
        if(Object.getOwnPropertyDescriptor(handler, methodProps[i]).writable) {
          layerObj.handle[methodProps[i]] = handler[methodProps[i]];
        }
      }
    }
  });
};
