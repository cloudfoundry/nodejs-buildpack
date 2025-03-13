/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';



function StringMatcher(agent) {
  this.agent = agent;
}
exports.StringMatcher = StringMatcher;


StringMatcher.prototype.init = function() {
};

StringMatcher.prototype.matchString = function(matchCondition, input) {
  var match = false;
  if(!input) {
    match = false;
  }
  else {
    if(matchCondition.type === 'IS_NOT_EMPTY') {
      if(input && input.length > 0) {
        match = true;
      }
    }
    else if(matchCondition.matchStrings.length == 1) {
      var matchString = matchCondition.matchStrings[0];

      if(!matchString || (input.length < matchString.length) && matchCondition.type != 'MATCHES_REGEX') {
        match = false;
      }
      else {
        switch(matchCondition.type) {
        case 'EQUALS':
          match = (matchString === input);
          break;
        case 'STARTS_WITH':
          match = (input.substr(0, matchString.length) === matchString);
          break;
        case 'ENDS_WITH':
          match = (input.substr(input.length - matchString.length, matchString.length) === matchString);
          break;
        case 'CONTAINS':
          match = (input.indexOf(matchString) != -1);
          break;
        case 'MATCHES_REGEX':
          if(!matchCondition._matchStringRegex && !matchCondition._matchStringRegexFailed) {
            try {
              matchCondition._matchStringRegex = new RegExp(matchString);
            }
            catch(err) {
              matchCondition._matchStringRegexFailed = true;
              this.agent.logger.warn('MatchResult has invalid regex: ' + matchString);
            }
          }

          if(!matchCondition._matchStringRegexFailed) {
            match = !!matchCondition._matchStringRegex.exec(input);
          }

          break;
        }
      }
    }
    else if(matchCondition.matchStrings.length > 1) {
      match = false;

      if(matchCondition.type === 'IS_IN_LIST') {
        var matchStrings = matchCondition.matchStrings;
        if(matchStrings) {
          for(var i = 0; i < matchStrings.length; i++) {
            if(matchStrings[i] === input) {
              match = true;
              break;
            }
          }
        }
      }
    }
  }

  return !!(match ^ matchCondition.isNot);
};

StringMatcher.prototype.matchKeyValue = function(match, keyValuePairs) {
  var type  = match.type,
    key   = match.key,
    value = match.value;

  if (key && key.matchStrings && key.matchStrings.length === 0) return true;

  // If keyValueMatch key or value are repeated, not a valid key/value mapping.
  if (key && key.matchStrings && key.matchStrings.length > 1) return false;
  if (value && value.matchStrings && value.matchStrings.length > 1) return false;

  // find a keyValuePairs entries whose key name matches...
  for (var item in keyValuePairs) {
    if (this.matchString(key, item)) {
      // ...which is enough on check exists...
      if (type == 'CHECK_FOR_EXISTENCE') return true;

      // ...otherwise see if its value matches:
      if (this.matchString(value, keyValuePairs[item])) {
        return true;
      }
    }
  }

  return false; // invalid type
};

StringMatcher.prototype.matchRepeatedKeyValue = function(rules, keyValuePairs) {
  if (!rules) return true;

  for (var i = 0; i < rules.length; i++) {
    if (!this.matchKeyValue(rules[i], keyValuePairs)) {
      return false;
    }
  }

  return true;
};
