'use strict';

var JsonArrayScanner = require('./json-array-scanner').JsonArrayScanner;
var AllocationGraphBuilder = require('./allocation-graph-builder.js').AllocationGraphBuilder;


function HeapProfiler(agent) {
  this.agent = agent;

  this.MAX_TREE_DEPTH = 250;

  this.classRegex = undefined;
}
exports.HeapProfiler = HeapProfiler;


HeapProfiler.prototype.init = function() {
  var self = this;

  this.active = false;
  self.classRegex = /^(.+)\.([^\.]+)$/;
};


HeapProfiler.prototype.isObjectTrackingSupported = function() {
  var self = this;

  return self.agent.appdNative.isObjectTrackingSupported();
};


HeapProfiler.prototype.trackAllocations = function(duration, callback) {
  var self = this;

  if(!self.agent.appdNative) {
    return callback("V8 tools are not loaded.");
  }

  if(self.active) {
    return callback("CPU profiler is already active.");
  }

  self.active = true;

  self.agent.appdNative.startTrackingHeapObjects();

  self.agent.timers.setTimeout(function() {
    if(!self.active) return;

    var builder = self.getBuilder();
    var scanner = self.getScanner();
    var strings = {};

    scanner.on('value', function(index, value) {
      if(builder.needsString(index)) {
        strings[index] = value;
      }
    });

    var profileBuf = [''];

    var stringsPart = false;
    var stringsPropRegex = /,\s*\"strings\"\s*:\s*\[/;

    self.agent.appdNative.takeHeapSnapshot(
      /* istanbul ignore next -- requires integration testing */
      function(chunk) {
        if(!stringsPart) {
          var lastChunk = profileBuf[profileBuf.length - 1];
          var stringsPropMatch = stringsPropRegex.exec(lastChunk + chunk);
          if(stringsPropMatch) {
            stringsPart = true;

            if(stringsPropMatch.index >= lastChunk.length) {
              profileBuf.push(chunk.substring(0, stringsPropMatch.index - lastChunk.length) + '}');
            }
            else {
              profileBuf[profileBuf.length - 1] = lastChunk.substring(0, stringsPropMatch.index) + '}';
            }

            builder.build(JSON.parse(profileBuf.join('')));

            scanner.scan('[' + chunk.substring(stringsPropMatch.index + stringsPropMatch[0].length - lastChunk.length));
          }
          else {
            profileBuf.push(chunk);
          }
        }
        else {
          if(!builder.needsMoreStrings()) {
            return false;
          }

          scanner.scan(chunk);
        }

        return true;
      }
    );

    self.agent.appdNative.stopTrackingHeapObjects();

    builder.setStrings(strings);

    self.active = false;
    self.agent.logger.debug("V8 stopped tracking object allocations");

    callback(null, self.convertToProtobufFormat(builder.getGraph()));
  }, duration * 1000);

};

/* istanbul ignore next -- here for unit test support */
HeapProfiler.prototype.getBuilder = function() {
  return new AllocationGraphBuilder();
};

/* istanbul ignore next -- here for unit test support */
HeapProfiler.prototype.getScanner = function() {
  return new JsonArrayScanner();
};

HeapProfiler.prototype.convertToProtobufFormat = function(allocationGraph) {
  var self = this;

  function convertNode(origNode) {
    var protoNode = {};

    var classMatch = self.classRegex.exec(origNode.functionInfo.functionName);
    var klass, method;
    if(classMatch && classMatch.length == 3) {
      klass = classMatch[1];
      method = classMatch[2];
    }
    else {
      klass = '(global)';
      method = origNode.functionInfo.functionName;
    }

    protoNode.numChildren = origNode.children.length;
    protoNode.size = origNode.totalLiveSize;
    protoNode.count = origNode.totalLiveCount;
    protoNode.klass = klass;
    protoNode.method = method;
    protoNode.fileName = origNode.functionInfo.scriptName;
    protoNode.lineNumber = origNode.functionInfo.line;
    protoNode.type = 'JS';

    return protoNode;
  }

  var elements = [];
  var nodeQueue = [];

  allocationGraph.level = 1;
  nodeQueue.push(allocationGraph);

  while(nodeQueue.length > 0) {
    var node = nodeQueue.shift();

    if(node.level <= self.MAX_TREE_DEPTH) {
      node.children.forEach(function(childNode) {
        childNode.level = node.level + 1;
        nodeQueue.push(childNode);
      });
    }

    var protoNode = convertNode(node);

    if(node.level > self.MAX_TREE_DEPTH) {
      protoNode.numChildren = 0;
    }

    elements.push(protoNode);
  }


  var processAllocationGraph = {
    allocationElements: elements,
    numOfRootElements: 1
  };

  return processAllocationGraph;
};

