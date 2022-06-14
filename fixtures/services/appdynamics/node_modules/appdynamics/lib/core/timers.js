/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

/* istanbul ignore next -- native JS interface */
(function() {

  function Timers(agent) {
    this.agent = agent;

    this.timeoutIds = [];
    this.intervalIds = [];

    this.Timer = undefined;
    this.timers = [];
  }
  exports.Timers = Timers;


  Timers.prototype.init = function() {
    var self = this;

    try {
      if (parseInt(process.versions.node.split('.')[0], 10) < 10)
        self.Timer = process.binding('timer_wrap').Timer;
      else
        self.agent.logger.info('Timers: Current node version is >= 10. Falling back to setTimeout');
    }
    catch(err) {
      self.agent.logger.warn("Timers: Timer wrap is not available, falling back to setTimeout");
    }

    self.agent.on('destroy', function() {
      self.timeoutIds.forEach(function(timeoutId) {
        clearTimeout(timeoutId);
      });
      self.timeoutIds = [];

      self.intervalIds.forEach(function(intervalId) {
        clearInterval(intervalId);
      });
      self.intervalIds = [];

      self.timers.forEach(function(timer) {
        timer.stop();
      });
      self.timers = [];
    });
  };


  Timers.prototype.setTimeout = function(func, ms) {
    var self = this;

    var context = {timeoutId: undefined};

    var funcWrap = function() {
      try {
        func.apply(this, arguments);
      }
      catch(err) {
        self.agent.logger.warn(err);
      }

      self.deleteTimeout(context.timeoutId);
    };

    context.timeoutId = setTimeout(funcWrap, ms);
    this.timeoutIds.push(context.timeoutId);

    return context.timeoutId;
  };


  Timers.prototype.setInterval = function(func, ms) {
    var self = this;

    var funcWrap = function() {
      try {
        func();
      }
    catch(err) {
      self.agent.logger.warn(err);
    }
    };

    var intervalId = setInterval(funcWrap, ms);
    self.intervalIds.push(intervalId);
    return intervalId;
  };


  Timers.prototype.clearTimeout = function(timeoutId) {
    this.deleteTimeout(timeoutId);
    clearTimeout(timeoutId);
  };


  Timers.prototype.clearInterval = function(intervalId) {
    this.deleteInterval(intervalId);
    clearTimeout(intervalId);
  };


  Timers.prototype.deleteTimeout = function(timeoutId) {
    for (var i = 0; i < this.timeoutIds.length; i++) {
      if(this.timeoutIds[i] === timeoutId) {
        this.timeoutIds.splice(i, 1);
        break;
      }
    }
  };


  Timers.prototype.deleteInterval = function(intervalId) {
    for (var i = 0; i < this.intervalIds.length; i++) {
      if(this.intervalIds[i] === intervalId) {
        this.intervalIds.splice(i, 1);
        break;
      }
    }
  };


  Timers.prototype.startTimer = function(ms, repeat, func) {
    var self = this;

    if(self.Timer) {
      var timer = new self.Timer();
      timer.start(ms, 0);

      var wrapper = function() {
        func();
        timer.start(ms, 0);
      };

      if(self.Timer.hasOwnProperty('kOnTimeout')) {
        timer[self.Timer.kOnTimeout] = repeat ? wrapper : func;
      }
      else {
        timer.ontimeout = repeat ? wrapper : func;
      }

      self.timers.push(timer);
    }
    else {
      if(repeat) {
        self.setInterval(func, ms);
      }
      else {
        self.setTimeout(func, ms);
      }
    }

    return null;
  };

})();
