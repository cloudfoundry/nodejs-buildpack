'use strict';

function AllocationGraphBuilder() {
  this.graph = undefined;
  this.needsStrings = {};
  this.needsStringsCount = 0;
}

exports.AllocationGraphBuilder = AllocationGraphBuilder;

var FunctionAllocationInfo = function(functionNameOffset, scriptNameOffset, scriptId, line, column) {
  this.functionNameOffset = functionNameOffset;
  this.scriptNameOffset = scriptNameOffset;
  this.functionName = undefined;
  this.scriptName = undefined;
  this.scriptId = scriptId;
  this.line = line;
  this.column = column;
  this.totalCount = 0;
  this.totalSize = 0;
  this.totalLiveCount = 0;
  this.totalLiveSize = 0;
  this._traceTops = [];
};

FunctionAllocationInfo.prototype.addTraceTopNode = function(node) {
  if (node.allocationCount === 0)
    return;
  this._traceTops.push(node);
  this.totalCount += node.allocationCount;
  this.totalSize += node.allocationSize;
  this.totalLiveCount += node.liveCount;
  this.totalLiveSize += node.liveSize;
};


var TopDownAllocationNode = function(id, functionInfo, count, size, liveCount, liveSize, parent)
{
  this.id = id;
  this.functionInfo = functionInfo;
  this.allocationCount = count;
  this.allocationSize = size;
  this.liveCount = liveCount;
  this.liveSize = liveSize;
  this.totalLiveCount = liveCount;
  this.totalLiveSize = liveSize;
  this.parent = parent;
  this.children = [];
};


var HeapSnapshotNode = function(profile, nodeIndex) {
  this.profile = profile;
  this.nodeIndex = nodeIndex;
};

HeapSnapshotNode.prototype.selfSize = function() {
  return this.profile.nodes[this.nodeIndex + this.profile._nodeSelfSizeOffset];
};

HeapSnapshotNode.prototype.traceNodeId = function() {
  return this.profile.nodes[this.nodeIndex + this.profile._nodeTraceNodeIdOffset];
};




AllocationGraphBuilder.prototype.build = function(profile) {
  var self = this;

  profile._rootNodeIndex = 0;
  if(profile.snapshot.root_index) {
    profile._rootNodeIndex = profile.snapshot.root_index;
  }
  profile._nodeSelfSizeOffset = profile.snapshot.meta.node_fields.indexOf("self_size");
  profile._nodeTraceNodeIdOffset = profile.snapshot.meta.node_fields.indexOf("trace_node_id");


  // build live objects
  if (!profile.snapshot.trace_function_count) {
    return;
  }

  var nodesLength = profile.nodes.length;
  var nodeFieldCount = profile.snapshot.meta.node_fields.length;
  var node = new HeapSnapshotNode(profile, profile._rootNodeIndex);
  var liveObjectStats = {};
  for (var nodeIndex = 0; nodeIndex < nodesLength; nodeIndex += nodeFieldCount) {
    node.nodeIndex = nodeIndex;
    var traceNodeId = node.traceNodeId();
    var stats = liveObjectStats[traceNodeId];
    if (!stats)
      liveObjectStats[traceNodeId] = stats = { count: 0, size: 0, ids: [] };
    stats.count++;
    stats.size += node.selfSize();
  }



  // build functions
  var functionInfoFields = profile.snapshot.meta.trace_function_info_fields;
  var functionNameOffset = functionInfoFields.indexOf("name");
  var scriptNameOffset = functionInfoFields.indexOf("script_name");
  var scriptIdOffset = functionInfoFields.indexOf("script_id");
  var lineOffset = functionInfoFields.indexOf("line");
  var columnOffset = functionInfoFields.indexOf("column");
  var functionInfoFieldCount = functionInfoFields.length;

  var rawInfos = profile.trace_function_infos;
  var infoLength = rawInfos.length;
  var functionInfos = this._functionInfos = new Array(infoLength / functionInfoFieldCount);
  var index = 0;
  for (var i = 0; i < infoLength; i += functionInfoFieldCount) {
    functionInfos[index++] = new FunctionAllocationInfo(
      rawInfos[i + functionNameOffset],
      rawInfos[i + scriptNameOffset],
      rawInfos[i + scriptIdOffset],
      rawInfos[i + lineOffset],
      rawInfos[i + columnOffset]);

    self.needsStrings[rawInfos[i + functionNameOffset]] = '';
    self.needsStrings[rawInfos[i + scriptNameOffset]] = '';
  }


  // build tree
  var traceTreeRaw = profile.trace_tree;
  var idToTopDownNode = [];


  var traceNodeFields = profile.snapshot.meta.trace_node_fields;
  var nodeIdOffset = traceNodeFields.indexOf("id");
  var functionInfoIndexOffset = traceNodeFields.indexOf("function_info_index");
  var allocationCountOffset = traceNodeFields.indexOf("count");
  var allocationSizeOffset = traceNodeFields.indexOf("size");
  var childrenOffset = traceNodeFields.indexOf("children");
  nodeFieldCount = traceNodeFields.length;


  function traverseNode(rawNodeArray, nodeOffset, parent) {
    var functionInfo = functionInfos[rawNodeArray[nodeOffset + functionInfoIndexOffset]];
    var id = rawNodeArray[nodeOffset + nodeIdOffset];
    var stats = liveObjectStats[id];
    var liveCount = stats ? stats.count : 0;
    var liveSize = stats ? stats.size : 0;
    var result = new TopDownAllocationNode(
      id,
      functionInfo,
      rawNodeArray[nodeOffset + allocationCountOffset],
      rawNodeArray[nodeOffset + allocationSizeOffset],
      liveCount,
      liveSize,
      parent);

    idToTopDownNode[id] = result;
    functionInfo.addTraceTopNode(result);

    var rawChildren = rawNodeArray[nodeOffset + childrenOffset];
    for (var i = 0; i < rawChildren.length; i += nodeFieldCount) {
      var child = traverseNode(rawChildren, i, result);

      // skip leaf nodes with zero size
      if(child.totalLiveCount === 0 && child.totalLiveSize === 0) {
        continue;
      }

      result.children.push(child);

      result.totalLiveCount += child.totalLiveCount;
      result.totalLiveSize += child.totalLiveSize;
    }

    return result;
  }

  self.graph = traverseNode(traceTreeRaw, 0, null);


  self.needsStringsCount = Object.keys(self.needsStrings).length;
};


AllocationGraphBuilder.prototype.needsString = function(offset) {
  var self = this;

  if(self.needsStrings[offset] !== undefined) {
    self.needsStringsCount--;
    return true;
  } else {
    return false;
  }
};


AllocationGraphBuilder.prototype.needsMoreStrings = function() {
  var self = this;

  return self.needsStringsCount > 0;
};


AllocationGraphBuilder.prototype.setStrings = function(strings) {
  var self = this;

  var nodeQueue = [];

  nodeQueue.push(self.graph);

  while(nodeQueue.length > 0) {
    var node = nodeQueue.shift();

    node.children.forEach(function(childNode) {
      nodeQueue.push(childNode);
    });

    node.functionInfo.functionName = strings[node.functionInfo.functionNameOffset];
    node.functionInfo.scriptName = strings[node.functionInfo.scriptNameOffset];
  }
};


AllocationGraphBuilder.prototype.getGraph = function() {
  var self = this;

  return self.graph;
};

