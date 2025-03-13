'use strict';

var util = require('util');
var EventEmitter = require('events').EventEmitter;


function JsonArrayScanner() {
  this.STATE_OBJECT = 1;
  this.STATE_ARRAY = 2;
  this.STATE_VALUE = 3;

  this.state = this.STATE_OBJECT;

  this.value = '';
  this.charIndex = 0;
  this.valueIndex = 0;
  this.escapeIndex = -2;

  EventEmitter.call(this);
}

util.inherits(JsonArrayScanner, EventEmitter);
exports.JsonArrayScanner = JsonArrayScanner;


JsonArrayScanner.prototype.scan = function(jsonChunk) {
  var self = this;

  var ch;
  for(var i = 0; i < jsonChunk.length; i++) {
    ch = jsonChunk[i];

    if(self.state !== self.STATE_VALUE) {
      switch(ch) {
      case '[':
        self.state = self.STATE_ARRAY;
        break;

      case ']':
        self.emit('value', self.valueIndex, self.value);
        self.value = '';
        self.valueIndex = 0;
        self.state = self.STATE_OBJECT;
        break;

      case '"':
        self.state = self.STATE_VALUE;
        break;

      case ',':
        if(self.state === self.STATE_ARRAY) {
          self.emit('value', self.valueIndex, self.value);
          self.value = '';
          self.valueIndex++;
        }
        break;
      }
    }
    else {
      switch(ch) {
      case '\\':
        if(self.escapeIndex !== (self.charIndex - 1)) {
          self.escapeIndex = self.charIndex;
        }

        self.value += ch;
        break;

      case '"':
        if(self.escapeIndex !== (self.charIndex - 1)) {
            // unescaped quotes, exit the string value
          self.state = self.STATE_ARRAY;
        }
        else {
          self.value += ch;
        }
        break;

      default:
        self.value += ch;
      }
    }

    self.charIndex++;
  }
};
