/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

function ExpressionEvaluator(agent) {
  this.agent = agent;
}
exports.ExpressionEvaluator = ExpressionEvaluator;


ExpressionEvaluator.prototype.init = function() {
};

ExpressionEvaluator.prototype.evaluate = function(node, ctx) {
  switch (node.type) {
  case 'EQUALS':
  case 'NOT_EQUALS':
  case 'LT':
  case 'GT':
  case 'LE':
  case 'GE':
    return this.opComparison(node, ctx);
  case 'NOT':
    return this.opNot(node.notOp, ctx);
  case 'AND':
    return this.opAnd(node.andOp, ctx);
  case 'OR':
    return this.opOr(node.orOp, ctx);
  case 'GETTER':
    return this.opGetter(node.getterOp, ctx);
  case 'STRINGMATCH':
    return this.opStringMatch(node.stringMatchOp, ctx);
  case 'MERGE':
    return this.opMerge(node.mergeOp, ctx);
  case 'SPLIT':
    return this.opSplit(node.splitOp, ctx);
  case 'REGEXCAPTURE':
    return this.opRegexCapture(node.regexCaptureOp, ctx);
  case 'ENTITY':
    return this.entityValue(node.entityValue, ctx);
  case 'STRING':
    return this.entityString(node.stringValue, ctx);
  case 'INTEGER':
    return this.entityInteger(node.integerValue, ctx);
  default:
    this.agent.logger.error('unhandled expression op: ' + node.type);
  }
};

ExpressionEvaluator.prototype.opComparison = function(node, ctx) {
  var lhs = this.evaluate(node.comparisonOp.lhs, ctx);
  var rhs = this.evaluate(node.comparisonOp.rhs, ctx);

  switch (node.type) {
  case 'EQUALS':
    return lhs === rhs;
  case 'NOT_EQUALS':
    return lhs !== rhs;
  case 'LT':
    return lhs < rhs;
  case 'GT':
    return lhs > rhs;
  case 'LE':
    return lhs <= rhs;
  case 'GE':
    return lhs >= rhs;
  }
};

ExpressionEvaluator.prototype.opNot = function(op, ctx) {
  return !this.evaluate(op.operand, ctx);
};

ExpressionEvaluator.prototype.opAnd = function(op, ctx) {
  for (var i = 0, length = op.operands.length; i < length; ++i) {
    if (!this.evaluate(op.operands[i], ctx)) {
      return false;
    }
  }

  return true;
};

ExpressionEvaluator.prototype.opOr = function(op, ctx) {
  for (var i = 0, length = op.operands.length; i < length; ++i) {
    if (this.evaluate(op.operands[i], ctx)) {
      return true;
    }
  }

  return false;
};

ExpressionEvaluator.prototype.opGetter = function(op, ctx) {
  var base = this.evaluate(op.base, ctx);
  if (!base) {
    this.agent.logger.debug('getter base is not an object');
    return undefined;
  }

  return base[op.field];
};

ExpressionEvaluator.prototype.opStringMatch = function(op, ctx) {
  var input = this.evaluate(op.input, ctx);
  return this.agent.stringMatcher.matchString(op.condition, input);
};

ExpressionEvaluator.prototype.opMerge = function(op, ctx) {
  var inputs = this.evaluate(op.inputArray, ctx);
  return inputs.join(op.delimiter);
};

ExpressionEvaluator.prototype.opSplit = function(op, ctx) {
  var input = this.evaluate(op.input, ctx);
  var result = input.split(op.delimiter);
  var segments = op.segments;

  if (segments !== undefined && segments.type !== undefined) {
    switch (segments.type) {
    case 'FIRST':
      result = result.slice(0, segments.numSegments);
      break;
    case 'LAST':
      result = result.slice(-segments.numSegments);
      break;
    case 'SELECTED':
      var selectedSegments = [];
      for (var i = 0, length = segments.selectedSegments.length; i < length; ++i) {
        // segment number is 1 based
        var index = segments.selectedSegments[i];
        if (index > 0 && index <= result.length)
          selectedSegments.push(result[index - 1]);
      }
      result = selectedSegments;
      break;
    }
  }


  return result;
};

ExpressionEvaluator.prototype.opRegexCapture = function(op, ctx) {
  var input = this.evaluate(op.input, ctx);
  var re = new RegExp(op.regex);
  var result = re.exec(input);
  if (!result) {
    return [];
  }
  if (op.regexGroups) {
    var groups = [];
    for (var i = 0, length = op.regexGroups.length; i < length; ++i) {
      //  - Check each grouping's string to avoid placing empty strings.
      //  - In the case of naming rules that merge groups using a delimiter,
      //    the delimiter will be added even if the string contains nothing.
      //  Ex)    Regex: .*/(\d+)/([a-z]+/)?(\d+)/.*
      //        String: 123/456/789
      //         Merge: .
      //        Result: 123.456
      //    Without If: 123..456
      if (result[op.regexGroups[i]]) {
        groups.push(result[op.regexGroups[i]]);
      }
    }

    result = groups;
  }
  else {
    result = [ result[0] ];
  }

  return result;
};

ExpressionEvaluator.prototype.entityValue = function(entity, ctx) {
  switch (entity.type) {
  case 'INVOKED_OBJECT':
    return ctx;
  case 'IDENTIFYING_PROPERTY':
    return ctx[entity.propertyName];
  default:
    this.agent.logger.debug('unhandled entity value type: ' + entity.type);
  }
};

ExpressionEvaluator.prototype.entityString = function(entity) {
  return entity;
};

ExpressionEvaluator.prototype.entityInteger = function(entity) {
  return entity;
};
