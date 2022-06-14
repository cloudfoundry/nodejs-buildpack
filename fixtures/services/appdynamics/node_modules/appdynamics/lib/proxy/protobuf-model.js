/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

function ProtobufModel(agent) {
  this.agent = agent;

  this.detectErrors = undefined;
  this.errorThreshold = undefined;
  this.ignoredMessagesConfig = undefined;
  this.ignoredExceptionConfig = undefined;
  this.callGraphConfig = undefined;

  this.traceRegex = undefined;
  this.userCodeErrRegEx = undefined;
  this.classRegex = undefined;
  this.agentRegex = undefined;
}
exports.ProtobufModel = ProtobufModel;


ProtobufModel.prototype.init = function() {
  var self = this;

  self.currentProcessSnapshot = undefined;

  self.btCallsStarted = null;
  self.btCallsCompleted = null;
  self.traceRegex = /at\s([^\(]+)\s\(([a-zA-Z]\:)?([^\:]+)\:(\d+)\:\d+\)$/;
  self.userCodeErrRegEx = /at\s([a-zA-Z]\:\\)?([^\:]+)\:(\d+)\:\d+$/;
  self.classRegex = /^(.+)\.([^\.]+)$/;
  self.agentRegex = /node_modules[\\\/]appdynamics/;

  self.agent.on('configUpdated', function() {
    self.detectErrors = self.agent.configManager.getConfigValue('errorConfig.errorDetection.detectErrors');
    self.errorThreshold = self.agent.configManager.getConfigValue('errorConfig.errorDetection.phpErrorThreshold');
    self.ignoredMessagesConfig = self.agent.configManager.getConfigValue('errorConfig.ignoredMessages');
    self.ignoredExceptionsConfig = self.agent.configManager.getConfigValue('errorConfig.ignoredExceptions');
    self.callGraphConfig = self.agent.configManager.getConfigValue('callgraphConfig');
  });
};

ProtobufModel.prototype.createBTIdentifier = function(transaction) {
  var btIdentifier;
  var inAppCorreation = transaction.corrHeader && (!transaction.corrHeader.crossAppCorrelation);
  if(transaction.registrationId) {
    btIdentifier = {
      type: inAppCorreation ? 'REMOTE_REGISTERED' : 'REGISTERED',
      btID: transaction.registrationId
    };
  }
  else {
    if(inAppCorreation) {
      btIdentifier = {
        type: 'REMOTE_UNREGISTERED',
        unregisteredRemoteBT: {
          btName: transaction.name,
          entryPointType: transaction.entryType
        }
      };

      if(transaction.isAutoDiscovered) {
        btIdentifier.unregisteredRemoteBT.matchCriteriaType = 'DISCOVERED';
        btIdentifier.unregisteredRemoteBT.namingSchemeType = transaction.namingSchemeType;
      }
      else {
        btIdentifier.unregisteredRemoteBT.matchCriteriaType = 'CUSTOM';
      }
    }
    else {
      btIdentifier = {
        type: 'UNREGISTERED',
        unregisteredBT: {
          btInfo: {
            internalName: transaction.name,
            entryPointType: transaction.entryType
          },
          isAutoDiscovered: transaction.isAutoDiscovered
        }
      };
    }
  }

  return btIdentifier;
};



ProtobufModel.prototype.createCorrelation = function(transaction) {
  var self = this;

  var corrHeader = transaction.corrHeader;
  if(!corrHeader || corrHeader.crossAppCorrelation) {
    return undefined;
  }

  var correlation = self.agent.correlation;

  var corrObj = {
    incomingBackendId: corrHeader.selfResolutionBackendId,
    incomingSnapshotEnabled: corrHeader.getSubHeader(correlation.SNAPSHOT_ENABLE) || false,
    doNotSelfResolve: corrHeader.getSubHeader(correlation.DONOTRESOLVE) || false,
    exitCallSequence: corrHeader.getSubHeader(correlation.EXIT_POINT_GUID),
    componentLinks: [],
  };

  var compFrom = corrHeader.getSubHeader(correlation.COMPONENT_ID_FROM);
  var compTo = corrHeader.getSubHeader(correlation.COMPONENT_ID_TO);
  var exitOrder = corrHeader.getSubHeader(correlation.EXIT_CALL_TYPE_ORDER);

  if(compFrom) {
    for(var i = 0; i < compFrom.length; i++) {
      corrObj.componentLinks.push({
        fromComponentID: compFrom[i],
        toComponentID: compTo[i],
        exitPointType: exitOrder[i]
      });
    }
  }

  return corrObj;
};



ProtobufModel.prototype.createBTDetails = function(transaction) {
  var self = this;

  var errorInfo = self.createErrorInfo(transaction);
  var exceptionInfo = self.createExceptionInfo(transaction);
  // This is awful, but the backend metrics must be created
  // before the snapshot info because creating the background metrics
  // attaches the backendIdentifier to the exit call objects.
  var backendMetrics = self.createBackendMetrics(transaction);
  var snapshotInfo = self.createSnapshotInfo(
    transaction, errorInfo, exceptionInfo);

  var btInfoState =
    transaction.btInfoResponse ? 'RESPONSE_RECEIVED' : 'MISSING_RESPONSE';

  var btDetails = {
    btInfoRequest: transaction.btInfoRequest,
    btMetrics: {
      timeTaken: transaction.ms,
      isError: !!transaction.hasErrors,
      backendMetrics: backendMetrics
    },
    btInfoState: btInfoState,
    snapshotInfo: snapshotInfo,
    errors: {
      exceptionInfo: exceptionInfo,
      errorInfo: errorInfo
    },
    sepInfo: transaction.sepRuleName
  };

  return btDetails;
};



ProtobufModel.prototype.createSnapshotInfo = function(transaction, errorInfo, exceptionInfo) {
  var self = this;

  //console.log('btInfoResponse', transaction.btInfoResponse)

  var snapshotInfo = undefined;
  var snapshotTriggerObject = self.agent.backendConnector.createSnapshotTrigger(transaction);
  if (snapshotTriggerObject.attachSnapshot) {
    var processSnapshots = transaction.processSnapshots;
    var processSnapshotGUIDs = undefined;
    var callGraph = undefined;
    var totalTimeMS = undefined;
    if (processSnapshots) {
      processSnapshotGUIDs = Object.keys(processSnapshots);
      if (processSnapshotGUIDs.length) {
        callGraph = self.createCallGraph(transaction);
      }
    }

    if (!callGraph) {
      totalTimeMS = transaction.ms;
    }

    snapshotInfo = {
      trigger: snapshotTriggerObject.snapshotTrigger,
      snapshot: {
        snapshotGUID: transaction.guid,
        timestamp: transaction.ts,
        callGraph: callGraph,
        errorInfo: errorInfo,
        exceptionInfo: exceptionInfo,
        processID: process.pid,
        exitCalls: self.createSnapshotExitCalls(transaction),
        totalTimeMS: totalTimeMS,
        upstreamCrossAppSnapshotGUID: transaction.incomingCrossAppGUID
      }
    };

    if (transaction.httpRequestSnapshotData) {
      snapshotInfo.snapshot.httpRequestData = transaction.httpRequestSnapshotData;
      snapshotInfo.snapshot.httpRequestData.requestMethod = transaction.method;
      snapshotInfo.snapshot.httpRequestData.responseCode = transaction.statusCode;
    }

    if(transaction.eumGuid) {
      snapshotInfo.snapshot.eumGUID = transaction.eumGuid;
    }

    processSnapshots = transaction.processSnapshots;
    if (processSnapshots) {
      snapshotInfo.snapshot.processSnapshotGUIDs = Object.keys(processSnapshots);
    }

    if(transaction.api) {
      if(transaction.api.onSnapshotCaptured) {
        transaction.api.onSnapshotCaptured(transaction.api);
      }

      if(transaction.api.snapshotData && transaction.api.snapshotData.length) {
        snapshotInfo.snapshot.methodInvocationData = (snapshotInfo.snapshot.methodInvocationData || [])
          .concat(transaction.api.snapshotData);
      }
    }
  }

  return snapshotInfo;
};

ProtobufModel.prototype.createSnapshot = function(transaction) {
  var self = this;
  var errorInfo = self.createErrorInfo(transaction);
  var exceptionInfo = self.createExceptionInfo(transaction);

  var processSnapshots = transaction.processSnapshots;
  var processSnapshotGUIDs = undefined;
  var snapshotInfo = self.createSnapshotInfo(transaction, errorInfo, exceptionInfo);
  if (processSnapshots) {
    processSnapshotGUIDs = Object.keys(processSnapshots);
    if (processSnapshotGUIDs.length) {
      //callGraph = self.createCallGraph(transaction);
    }
  }

  if (transaction.httpRequestData) {
    snapshotInfo.snapshot.httpRequestData = transaction.httpRequestData;
  }

  if(transaction.eumGuid) {
    snapshotInfo.snapshot.eumGUID = transaction.eumGuid;
  }

  processSnapshots = transaction.processSnapshots;
  if (processSnapshots) {
    snapshotInfo.snapshot.processSnapshotGUIDs = Object.keys(processSnapshots);
  }

  return snapshotInfo;
};



ProtobufModel.prototype.createAppException = function(error) {
  return this.createExceptionInfo({ error: error }, true);
};

ProtobufModel.prototype.createBackendMetrics = function(transaction) {
  var self = this;

  var backendMetricsMap = {};
  var backendMetrics = [];
  if(transaction.exitCalls) {
    transaction.exitCalls.forEach(function(exitCall) {
      var exitCallId = exitCall.getBackendInfoString();
      var backendMetric = backendMetricsMap[exitCallId];
      if(backendMetric) {
        backendMetric.numOfCalls++;
        if(exitCall.error) {
          backendMetric.numOfErrors++;
        }

        if(exitCall.ms < backendMetric.minCallTime) {
          backendMetric.minCallTime = exitCall.ms;
        }

        if(exitCall.ms > backendMetric.maxCallTime) {
          backendMetric.maxCallTime = exitCall.ms;
        }

        // this will be needed by SnapshotDbCalls
        exitCall.backendIdentifier = backendMetric.backendIdentifier;

        return;
      }

      var backendIdentifier;
      if(exitCall.registrationId) {
        var registeredBackend = {
          exitPointType: exitCall.exitType,
          exitPointSubtype: exitCall.exitSubType,
          backendID: exitCall.registrationId
        };
        var componentId = exitCall.componentId;
        if (componentId) {
          registeredBackend.componentID = componentId;
          registeredBackend.componentIsForeignAppID = !!exitCall.componentIsForeignAppId;
        }
        backendIdentifier = {
          type: 'REGISTERED',
          registeredBackend: registeredBackend
        };
      }
      else {
        var identifyingProperties = [];
        var detectedIdentifyingProperties = exitCall.identifyingProperties;
        for (var propName in detectedIdentifyingProperties) {
          if (!detectedIdentifyingProperties.hasOwnProperty(propName))
            continue;
          var prop =
              {name: propName, value: detectedIdentifyingProperties[propName]};
          identifyingProperties.push(prop);
        }

        var displayName = self.agent.backendConfig.generateDisplayName(exitCall);
        backendIdentifier = {
          type: 'UNREGISTERED',
          unregisteredBackend: {
            exitCallInfo: {
              exitPointType: exitCall.exitType,
              exitPointSubtype: exitCall.exitSubType,
              displayName: displayName,
              identifyingProperties: identifyingProperties
            }
          }
        };
      }

      backendMetric = {
        category: exitCall.category,
        timeTaken: exitCall.ms,
        numOfCalls: 1,
        numOfErrors: (exitCall.error ? 1 : 0),
        minCallTime: exitCall.ms,
        maxCallTime: exitCall.ms,
        backendIdentifier: backendIdentifier
      };

      backendMetrics.push(backendMetric);
      backendMetricsMap[exitCallId] = backendMetric;

      // this will be needed by SnapshotDbCalls
      exitCall.backendIdentifier = backendIdentifier;
    });
  }

  return backendMetrics;
};



ProtobufModel.prototype.createErrorInfo = function() {
  // will be reused for console.log and console.error messages

  var self = this;

  if(!self.detectErrors) {
    return undefined;
  }

  var errorInfo = {
    errors: []
  };

  // iterate over console.log and console.error messages here instead

  /*
if(transaction.exitCalls) {
transaction.exitCalls.forEach(function(exitCall) {
if(!exitCall.error) return;
if(exitCall.error.stack) return; // filter out exceptions
// don't care about errorThreshold for now, because we only have ERRORs

var errorMessage = self.extractErrorMessage(error);
if(!errorMessage) return;

if(self.isErrorIgnored(errorMessage)) return;

var error = errorsMap[errorMessage];
if(error) {
error.count++;
return;
}

error = {
errorThreshold: 'ERROR',
errorMessage: exitCall.error.message,
displayName: "Node.js Error",
count: 1
}

errorsMap[errorMessage] = error;
errorInfo.errors.push(error)
});
}
   */

  if(errorInfo.errors.length > 0) {
    return errorInfo;
  }

  return undefined;
};


ProtobufModel.prototype.createAppException = function(error) {
  return this.createExceptionInfo({ error: error }, true);
};

ProtobufModel.prototype.constructStackTrace = function(stackTraceStr) {
  var self = this;
  var stackTrace = {
    elements: []
  };

  if(stackTraceStr && typeof(stackTraceStr) === 'string') {
    stackTraceStr = stackTraceStr.replace(/\(anonymous\sfunction\)/, '<anonymous>');
    var lines = stackTraceStr.split("\n");
    lines.shift();
    lines.forEach(function(line) {
      var traceMatch = self.traceRegex.exec(line);
      var userCodeErrMatch = self.userCodeErrRegEx.exec(line);
      if(traceMatch && traceMatch.length == 5) {
        var klass, method, classMatch, drivePrefix;
        classMatch = self.classRegex.exec(traceMatch[1]);
        if(classMatch && classMatch.length == 3) {
          klass = classMatch[1];
          method = classMatch[2];
        }
        else {
          klass = '';
          method = traceMatch[1];
        }

        if (!self.agentRegex.exec(traceMatch[3])) {
          drivePrefix = traceMatch[2] || '';
          stackTrace.elements.push({
            klass: klass,
            method: method,
            fileName: drivePrefix + traceMatch[3],
            lineNumber: parseInt(traceMatch[4])
          });
        }
      } else if (userCodeErrMatch && userCodeErrMatch.length == 4) {
        if (!self.agentRegex.exec(userCodeErrMatch[2])) {
          drivePrefix = userCodeErrMatch[1] || '';
          stackTrace.elements.push({
            klass: '',
            method: '',
            fileName: drivePrefix + userCodeErrMatch[2],
            lineNumber: parseInt(userCodeErrMatch[3])
          });
        }
      }
    });
  }
  return stackTrace;
};

ProtobufModel.prototype.createExceptionInfo = function(transaction, skipIgnored) {
  var self = this;

  var exceptionsMap = {};
  var exceptionInfo = {
    exceptions: [],
    stackTraces: []
  };

  var errorSources = [];
  if (transaction.error) errorSources.push(transaction);
  if (transaction.exitCalls) errorSources = errorSources.concat(transaction.exitCalls);
  errorSources.forEach(function(errorSource) {
    if(!errorSource.error) return;

    var message = self.extractErrorMessage(errorSource.error);
    var rootException = exceptionsMap[message];

    if(rootException) {
      rootException.count++;
      return;
    }

    var ignored = self.isExceptionIgnored(errorSource.error.name, message);
    if (!ignored) transaction.hasErrors = true;
    if (ignored && skipIgnored) {
      return;
    }

    var stackTrace = self.constructStackTrace(errorSource.error.stack);

    // exception
    rootException = {
      root: {
        klass: (errorSource.backendName && errorSource.backendName + ' Error') ||
          errorSource.error.name ||
          'Unknown Error',
        message: message,
        stackTraceID: exceptionInfo.stackTraces.length
      },
      count: 1
    };

    exceptionInfo.stackTraces.push(stackTrace);
    exceptionInfo.exceptions.push(rootException);
    exceptionsMap[message] = rootException;
  });

  if(exceptionInfo.exceptions.length > 0) {
    return exceptionInfo;
  }

  return undefined;
};


ProtobufModel.prototype.isErrorIgnored = function(message) {
  var self = this;

  if(!self.ignoredMessagesConfig) {
    return false;
  }

  var ignore = false;
  self.ignoredMessagesConfig.forEach(function(ignoredMessageConfig) {
    if(self.agent.stringMatcher.matchString(ignoredMessageConfig, message)) {
      ignore = true;
    }
  });

  return ignore;
};


ProtobufModel.prototype.isExceptionIgnored = function(name, message) {
  var self = this;

  if(!self.ignoredExceptionsConfig) {
    return false;
  }

  var ignore = false;
  self.ignoredExceptionsConfig.forEach(function(ignoredExceptionConfig) {
    var match = ignoredExceptionConfig.classNames[0];
    if ((match == name || match == '*') && self.agent.stringMatcher.matchString(
      ignoredExceptionConfig.matchCondition, message)) {
      ignore = true;
    }
  });

  return ignore;
};

/*
If there is one or more process snapshots linked to this transaction
then make a fake call graph such that the UI will indicate the
BT snapshot for this transaction has call graph data ( aka deep dive data ).
 */
ProtobufModel.prototype.createCallGraph = function(transaction) {
  var callGraph = {
    callElements: [ {
      timeTaken: transaction.ms,
      numOfChildren: 0,
      name: '{name}',
      type: 'JS',
      method: '{request}'
    } ]
  };
  return callGraph;
};

ProtobufModel.prototype.createSnapshotExitCalls = function(transaction) {
  var self = this;

  var exitCalls = [];
  var exitCallsMap = {};
  if(transaction.exitCalls) {
    transaction.exitCalls.forEach(function(exitCall) {
      var snapshotExitCallId;
      var snapshotExitCall;
      if (exitCall.isSql) {
        snapshotExitCallId = exitCall.backendName + ':' + exitCall.command;
        snapshotExitCall = exitCallsMap[snapshotExitCallId];
        if(snapshotExitCall) {
          snapshotExitCall.count++;
          snapshotExitCall.timeTaken += exitCall.ms;
          return;
        }
      }

      snapshotExitCall = {
        backendIdentifier: exitCall.backendIdentifier,
        sequenceInfo: exitCall.sequenceInfo,
        timeTaken: exitCall.ms,
        count: 1,

        errorDetails: self.extractErrorMessage(exitCall.error),
        detailString: exitCall.command,
        boundParameters: undefined
      };

      if(exitCall.isSql && exitCall.commandArgs) {
        snapshotExitCall.boundParameters = {
          type: 'POSITIONAL',
          posParameters: exitCall.commandArgs
        };
      }

      exitCalls.push(snapshotExitCall);

      if((!exitCall.commandArgs) && (exitCall.isSql)) {
        exitCallsMap[snapshotExitCallId] = snapshotExitCall;
      }
    });
  }

  if(exitCalls.length > 0) {
    return exitCalls;
  }

  return undefined;
};

ProtobufModel.prototype.extractErrorData = function(error, field) {
  if(typeof(error) == 'string') {
    return error;
  } else if(typeof(error) == 'object' && error !== null) {
    return error[field];
  }

  return undefined;
};

ProtobufModel.prototype.extractErrorMessage = function(error) {
  var self = this;
  return self.extractErrorData(error, 'message');
};

ProtobufModel.prototype.extractErrorName = function(error) {
  var self = this;
  return self.extractErrorData(error, 'name');
};
