/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

function CpuProfiler(agent) {
  this.agent = agent;

  this.active = false;
  this.classRegex = undefined;
  this.appdRegex = undefined;
  this.onStopCallback = undefined;
  this.threadsProfiled = [];
  this.graceBtCompletionTimeMS = 5000;
  this.inGracePeriod = false;
  this.attachedHandler = false;
}

exports.CpuProfiler = CpuProfiler;

CpuProfiler.prototype.init = function() {
  this.classRegex = /^(.+)\.([^\.]+)$/;
  this.appdRegex = /\/appdynamics\//;
  this.agent.once('destroy', this.onAgentDestroy.bind(this));
};

CpuProfiler.prototype.onAgentDestroy = function () {
  if(!this.active) return;

  var callback = this.onStopCallback;

  try {
    this.stopCpuProfiler(); // ignoring any output
    if (callback) {
      callback("CPU profiling was aborted because of the destroy() call");
    }
  }
  catch(err) {
    if (callback) {
      callback(err);
    }
  }
};

CpuProfiler.prototype.startCpuProfiler = function(seconds, callback, threadId) {
  var self = this;

  if(!self.agent.appdNative) {
    return callback("V8 tools are not loaded.");
  }

  if(!self.inGracePeriod) self.threadsProfiled.push(threadId);

  if(self.active) {
    return callback("CPU profiler is already active.");
  }

  self.active = true;
  self.onStopCallback = callback;

  self.agent.appdNative.startV8Profiler();
  self.agent.logger.debug("V8 CPU profiler started");

  // stop v8 profiler automatically after the specified number of seconds
  self.agent.timers.setTimeout(function() {
    if(!self.active) return;

    if(self.threadsProfiled.length > 0) {
      // Wait for graceBtCompletionTimeMS to stop the profiler
      self.inGracePeriod = true;

      self.agent.timers.setTimeout(function() {
        if(!self.active) return;

        self.safeStopCpuProfiler(callback);
      }, self.graceBtCompletionTimeMS);
    } else {
      self.safeStopCpuProfiler(callback);
    }
  }, seconds * 1000);
  if (!this.attachedHandler) {
    this.attachedHandler = true;
    self.agent.on('btDetails', function(btDetails, transaction) {
      if (!self.active) return;

      var isBtBeingProfiled = self.threadsProfiled.indexOf(transaction.threadId);
      if(isBtBeingProfiled > -1) {
        self.threadsProfiled.splice(isBtBeingProfiled, 1);
      }
    });
  }
};

function endsWith(str, suffix) {
  return str.indexOf(suffix, str.length - suffix.length) !== -1;
}

CpuProfiler.prototype.safeStopCpuProfiler = function(callback) {
  var self = this;

  self.threadsProfiled = [];
  self.inGracePeriod = false;
  try {
    callback(null, self.stopCpuProfiler());
  }
  catch(err) {
    callback(err);
  }
};

CpuProfiler.prototype.stopCpuProfiler = function() {
  var self = this;

  if(!self.agent.appdNative || !self.active) return;

  var processCallGraph = {
    numOfRootElements: 1,
    callElements: []
  };

  var excludeAgentFromCallGraph = self.agent.opts.excludeAgentFromCallGraph;

  self.agent.appdNative.stopV8Profiler(
    function(childrenCount, samplesCount, functionName, scriptResourceName) {
      if(functionName === '(program)') {
        return true;
      }
      if (endsWith(scriptResourceName, "/proxy-funcs.js")) {
        return false;
      }
      if(excludeAgentFromCallGraph && self.appdRegex.exec(scriptResourceName)) {
        return true;
      }

      return false;
    },
    function(childrenCount, samplesCount, functionName, scriptResourceName, lineNumber) {
      var classMatch = self.classRegex.exec(functionName);
      var klass, method;
      if(classMatch && classMatch.length == 3) {
        klass = classMatch[1];
        method = classMatch[2];
      }
      else {
        klass = '(global)';
        method = functionName;
      }

      var callElement = {
        klass: klass,
        method: method,
        lineNumber: lineNumber,
        fileName: scriptResourceName,
        numChildren: childrenCount,
        samplesCount: samplesCount,
        type: 'JS'
      };

      processCallGraph.callElements.push(callElement);
    });


  self.agent.logger.debug("V8 CPU profiler stopped");

  self.active = false;
  self.onStopCallback = undefined;

  return processCallGraph;
};

CpuProfiler.prototype.getHeapSnapshot = function(types, callback) {
  if (this.active) return callback('cpu-profiler-active');
  this.agent.appdNative.getV8HeapSnapshot(types, callback);
};
