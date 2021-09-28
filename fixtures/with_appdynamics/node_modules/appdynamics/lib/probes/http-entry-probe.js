/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var HttpCommon = require('./http-common');

function HttpEntryProbe(agent) {
  this.agent = agent;

  this.statusCodesConfig = undefined;
  this.delayedCallbackQueue = [];
}

exports.HttpEntryProbe = HttpEntryProbe;

HttpEntryProbe.prototype.init = function () {
  var self = this;

  self.agent.on('configUpdated', function () {
    self.statusCodesConfig = self.agent.configManager.getConfigValue('errorConfig.httpStatusCodes');
  });
};

HttpEntryProbe.prototype.attach = function (obj, moduleName) {
  var self = this;

  self.agent.timers.startTimer(100, true, function () {
    var now = Date.now();

    while (self.delayedCallbackQueue.length > 0) {
      if (self.delayedCallbackQueue[0].ts < now - 10) {
        var delayedCallbackInfo = self.delayedCallbackQueue.shift();
        delayedCallbackInfo.func.call(this);
      } else {
        break;
      }
    }
  });

  self.isHTTPs = obj.Agent && (obj.Agent.prototype.defaultPort == 443);

  // server probe
  self.agent.proxy.before(obj.Server.prototype, ['on', 'addListener'], function (obj, args) {
    if (args[0] !== 'request') return;

    if (obj.__httpProbe__) return;
    obj.__httpProbe__ = true;

    var cbIndex = args.length - 1;

    args[cbIndex] = self.__createRequestHandler(args[cbIndex], moduleName === 'https');
  });
};

HttpEntryProbe.prototype.finalizeTransaction = function (err, profiler, time, transaction, req, res) {
  if (!time.done()) return;

  transaction.error = transaction.error || res.error || err;
  transaction.statusCode = transaction.statusCode ||
    (transaction.error && transaction.error.statusCode) ||
    (res && res.statusCode) ||
    500;
  transaction.stackTrace = transaction.stackTrace || profiler.formatStackTrace(transaction.error);

  var error = HttpCommon.generateError(transaction.error, transaction.statusCode, this.statusCodesConfig);
  if (error) {
    transaction.error = error;
  }

  if (transaction.api && transaction.api.onResponseComplete) {
    transaction.api.onResponseComplete.apply(transaction.api, [req, res]);
  }
  profiler.endTransaction(time, transaction);
};

function createBTCallback(agent, profiler, time, transaction, req, res, thread, callback, self, origSelf, origArgs) {
  var didRun = false;
  var threadId = thread.current();

  // Node 0.8: need to ensure request isn't consumed
  // before delayed handler gets run
  if (agent.processInfo.isv0_8) {
    req.pause();
  }

  return self.agent.context.bind(function () {
    if (didRun) return;
    didRun = true;

    // Node 0.8: safe to resume the request now
    // we're ready to process it
    if (agent.processInfo.isv0_8) {
      req.resume();
    }

    var oldThreadId = thread.current();
    thread.resume(threadId);
    try {
      callback = agent.proxy.wrapWithThreadProxyIfEnabled(callback);
      callback.apply(origSelf, origArgs);
    } catch (e) {
      self.finalizeTransaction(e, profiler, time, transaction, req, res);
      throw e;
    } finally {
      thread.resume(oldThreadId);
    }
  });
}

HttpEntryProbe.prototype.__createRequestHandler = function (callback, isHTTPs) {
  var self = this;

  return function (req, res) {
    self.agent.context.run(requestHandler, req, res);
  };

  function requestHandler(req, res) {
    var profiler = self.agent.profiler;
    var proxy = self.agent.proxy;
    var time = profiler.time(true);

    self.agent.metricsManager.addMetric(self.agent.metricsManager.HTTP_INCOMING_COUNT, 1);

    var transaction = profiler.startTransaction(time, req, 'NODEJS_WEB');
    self.agent.context.set('threadId', transaction.threadId);
    req.__appdThreadId = transaction.threadId;

    transaction.url = req.url;
    transaction.method = req.method;
    transaction.requestHeaders = req.headers;

    var eumEnabled = (transaction.eumEnabled && !transaction.skip) || (self.agent.eum.enabled && self.agent.eum.enabledForTransaction(req));

    if (!transaction.corrHeader && eumEnabled) {
      proxy.before(res, 'writeHead', function (obj) {
        if(!transaction.isFinished) {
          var eumCookie = self.agent.eum.newEumCookie(transaction, req, obj, isHTTPs);
          eumCookie.build();
        }
      });
    }

    proxy.after(res, 'end', function () {
      self.finalizeTransaction(null, profiler, time, transaction, req, res);
    });

    if (self.agent.opts.btEntryPointDelayDisabled) {
      try {
        return callback.apply(this, arguments);
      } catch (e) {
        self.finalizeTransaction(e, profiler, time, transaction, req, res);
        throw e;
      }
    }

    var delayedCallback = createBTCallback(self.agent,
      profiler,
      time,
      transaction,
      req, res,
      self.agent.thread,
      callback,
      self,
      this,
      arguments);

    transaction.once('ignoreTransactionCbExecute', delayedCallback);
    transaction.emit('delayedCallbackReady');
    transaction.once('btInfoResponse', delayedCallback);
    self.delayedCallbackQueue.push({ ts: Date.now(), func: delayedCallback });
  }
};
