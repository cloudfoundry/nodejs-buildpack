/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
*/
'use strict';
var HttpCommon = require('./http-common');

var url = require('url');
var HTTPParser = process.binding('http_parser').HTTPParser;
var nodeVersionRegEx = /v([0-9]*).([0-9]*).([0-9]*)/;

function HttpExitProbe(agent) {
  this.agent = agent;
}

exports.HttpExitProbe = HttpExitProbe;

HttpExitProbe.prototype.init = function () { };

HttpExitProbe.prototype.attach = function (obj, moduleName) {
  var self = this;
  var profiler = self.agent.profiler;
  var proxy = self.agent.proxy;
  var nodeMajorVersion = (process.version).match(nodeVersionRegEx);
  if (parseInt(nodeMajorVersion[1], 10) >= 8) {
    var orig = obj.get;
    obj.get = function (options, cb) {
      var req = obj.request(options, cb);
      req.end();
      return req;
    };
    obj.get.__appdynamicsProxyInfo__ = {
      obj: obj,
      meth: 'get',
      orig: orig
    };
  }

  // support 0.11.x and further
  if (obj.globalAgent && obj.globalAgent.request) {
    obj = obj.globalAgent;
  }

  function clientCallback(locals) {
    if (!locals.time.done()) return;

    var exitCall = locals.exitCall;
    var error = locals.error;

    if (exitCall) {
      if (locals.res) {
        exitCall.responseHeaders = locals.res.headers;
        exitCall.statusCode = ~~locals.res.statusCode;
        if((!error) && ((exitCall.statusCode < 200) || (exitCall.statusCode >= 400))) {
          error = HttpCommon.getHttpExitCallError(exitCall.statusCode, exitCall.stack, locals);
        }
      }
      profiler.addExitCall(locals.time, exitCall, error);
    }
  }

  proxy.around(obj, 'request', function (obj, args, locals) {
    var isRepeatForHttps = false;
    if (moduleName === 'https') {
      if (typeof (args[0]) === 'string') {
        args[0] = url.parse(args[0]);
      }
      args[0].__appdIsHttps = true;
    }
    self.setHttpDefaults(locals, args[0], moduleName);

    if (locals.opts.method === 'CONNECT') {
      // ignore proxy tunnel setup; requests sent via the 
      // proxy tunnel will get instrumented as they are made
      args[0].appdIgnore = true;
      return;
    }

    locals.time = profiler.time();
    self.agent.logger.debug('HTTP exit call is initiated by the transaction for endpoint: ' +
      locals.opts.method + ' ' +
      locals.opts.hostname + ' ' +
      locals.opts.port + ' ' +
      locals.opts.path);

    if (moduleName === 'http' && args[0].__appdIsHttps) {
      isRepeatForHttps = true;
    }

    var isDynamoDBReq = args[0].headers && args[0].headers['X-Amz-Target'] && args[0].headers['X-Amz-Target'].indexOf('DynamoDB') > -1;
    if (args[0].appdIgnore || isDynamoDBReq || isRepeatForHttps) {
      // (ignore internal HTTP calls, e.g. to Analytics Agent, DynamoDB calls)
      self.agent.logger.debug('Skipping HTTP exit call for the transaction.' +
        'AppdIgnore is: ' + args[0].appdIgnore + ' ' +
        'DynamoDB call is:' + isDynamoDBReq + ' ' +
        'HTTPS call is:' + isRepeatForHttps);
    } else {
      self.agent.logger.debug('Gatheing HTTP exit call information.');
      var threadId = self.agent.thread.current();

      self.agent.metricsManager.addMetric(self.agent.metricsManager.HTTP_OUTGOING_COUNT, 1);
      var host = locals.opts.hostname;
      var port = locals.opts.port;
      var path = locals.opts.path;

      var supportedProperties = {
        'HOST': host,
        'PORT': port
      };

      // only populate the query string if it's required for naming or identity
      // TODO: needs libagent support
      if (self.agent.backendConfig.isParsedUrlRequired()) {
        var parsedUrl = url.parse(path);
        supportedProperties.URL = parsedUrl.pathname;
        if (parsedUrl.query) {
          supportedProperties['QUERY STRING'] = parsedUrl.query;
        }
      }

      var category = ((locals.opts.method === 'POST' || locals.opts.method === 'PUT') ? "write" : "read");

      locals.exitCall = profiler.createExitCall(locals.time, {
        exitType: 'EXIT_HTTP',
        supportedProperties: supportedProperties,
        stackTrace: profiler.stackTrace(),
        group: (locals.opts.method || 'GET'),
        method: locals.opts.method,
        command: host + ':' + port + path,
        requestHeaders: locals.opts.headers,
        category: category,
        protocol: moduleName
      });

      if (!locals.exitCall) return;

      Error.captureStackTrace(locals.exitCall);
      var dataConsumption = false, res;
      proxy.callback(args, -1, function (obj, args) {
        res = args[0];
        // If there is no way to consume the data here, then close up the exit call loop and
        // release the httpParser's object httpParserMethod.
        // There are 3 ways to consume data from a readable stream according to the doc:
        // https://nodejs.org/dist/latest-v4.x/docs/api/stream.html#stream_class_stream_readable
        proxy.before(res, ['on', 'addListener'], function (obj, args) {
          // workaround for end event
          if (!dataConsumption && args[0] === 'data') {
            dataConsumption = true;
          }

          proxy.callback(args, -1, null, null, threadId);
        }, false, false, threadId);

        proxy.before(res, 'pipe', function () {
          dataConsumption = true;
        }, false, false, threadId);

        proxy.before(res, 'resume', function () {
          dataConsumption = true;
        }, false, false, threadId);
      }, function () {
        if (dataConsumption) return;
        var httpParser = res.socket.parser,
          kOnHeadersComplete = HTTPParser.kOnHeadersComplete | 0,
          httpParserMethod = kOnHeadersComplete ? kOnHeadersComplete : 'onHeadersComplete';
        proxy.release(httpParser[httpParserMethod]);
        locals.res = res;
        clientCallback(locals);
        res.socket.__appdynamicsCleanup = true;
      }, threadId);
    }
  },
    function (obj, args, ret, locals) {
      var writeOnce = false, httpParser, httpParserMethod;
      var threadId = locals.time.threadId;

      if (!args[0].appdIgnore && (moduleName != 'http' || (moduleName === 'http' && !args[0].__appdIsHttps))) {
        proxy.before(ret, ['on', 'addListener'], function (obj, args) {
          proxy.callback(args, -1, null, null, threadId);
        }, false, false, threadId);

        proxy.before(ret, ['write', 'end'], function (obj) {
          if (!writeOnce) {
            writeOnce = true;
            proxy.callback(ret, -1, null, null, threadId);
          } else {
            return;
          }

          if (locals.exitCall) {
            var correlationHeaderValue = self.agent.backendConnector.getCorrelationHeader(locals.exitCall);
            if (correlationHeaderValue) obj.setHeader(self.agent.correlation.HEADER_NAME, correlationHeaderValue);
          }
        }, false, false, threadId);

        ret.on('socket', function (socket) {
          // For v0.10.0 and below the httpParser has method named onHeadersComplete.
          // For v0.12.0 and above the httpParser has a constant int value for kOnHeadersComplete. The
          // callback for onHeaderComplete is attached as integer property on the parser and the
          // property is defined by the constant value of kOnHeadersComplete.
          httpParser = socket.parser;
          var kOnHeadersComplete = HTTPParser.kOnHeadersComplete | 0;
          httpParserMethod = kOnHeadersComplete ? kOnHeadersComplete : 'onHeadersComplete';

          var socketCloseHandler = function () {
            socket.removeListener('close', socketCloseHandler);
            if (socket.__appdynamicsCleanup) {
              return;
            }
            proxy.release(httpParser[httpParserMethod]);
            clientCallback(locals);
          };
          socket.on('close', socketCloseHandler);
          proxy.after(httpParser, httpParserMethod, function (obj, args, ret) {
            var resp = httpParser.incoming;
            resp.on('end', function () {
              proxy.release(httpParser[httpParserMethod]);
              locals.res = resp;
              clientCallback(locals);
              socket.__appdynamicsCleanup = true;
              socket.removeListener('close', socketCloseHandler);
            });
            return ret;
          }, false, threadId);
        });

        ret.on('error', function (error) {
          var currentCtxt = self.agent.thread.current();
          self.agent.thread.resume(threadId);
          if (httpParser && httpParserMethod) {
            proxy.release(httpParser[httpParserMethod]);
          }
          locals.error = error;
          clientCallback(locals);
          if (ret.socket) {
            ret.socket.__appdynamicsCleanup = true;
          }
          self.agent.thread.resume(currentCtxt);
        });
      }
    });
};

HttpExitProbe.prototype.setHttpDefaults = function (locals, spec, protocol) {
  if (typeof (spec) === 'string') {
    locals.opts = url.parse(spec);
  }
  else {
    locals.opts = spec;
  }

  locals.opts.hostname = locals.opts.hostname || locals.opts.host || 'localhost';
  locals.opts.port = locals.opts.port || ((protocol === 'https') ? 443 : 80);
  locals.opts.path = locals.opts.path || '/';
};
