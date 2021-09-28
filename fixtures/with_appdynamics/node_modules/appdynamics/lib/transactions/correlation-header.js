/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';


function CorrelationHeader(agent) {
  this.agent = agent;
  this.correlation = agent.correlation;

  this.subHeaders = {};
  this.selfResolutionBackendId = undefined;
  this.crossAppCorrelation = false;
  this.crossAppCorrelationBackendId = undefined;
  this.txDetect = true;
}
exports.CorrelationHeader = CorrelationHeader;


CorrelationHeader.prototype.addSubHeader = function(name, value) {
  var self = this;

  self.subHeaders[name] = value;
};

CorrelationHeader.prototype.getSubHeader = function(name, defaultValue) {
  var self = this;

  var value = self.subHeaders[name];

  if(value === undefined && defaultValue !== undefined) {
    return defaultValue;
  }

  return value;
};

CorrelationHeader.prototype.getStringSubHeader = function(name) {
  var self = this;

  var value = self.subHeaders[name];
  if(value) {
    if(Array.isArray(value)) {
      return value.join(',');
    }
    if(typeof(value) === 'boolean') {
      return value.toString();
    }
    else {
      return value;
    }
  }
  else {
    return undefined;
  }
};

CorrelationHeader.prototype.getStringHeader = function() {
  var self = this;

  var pairs = [];

  for(var name in self.subHeaders) {
    pairs.push(name + '=' + self.getStringSubHeader(name));
  }

  return pairs.join('*');
};

CorrelationHeader.prototype.parseHeaderString = function(headerString) {
  var self = this;
  // sanitize header based on CORE-20346
  var headerStringParts = headerString.split(', ');
  headerString = headerStringParts[headerStringParts.length - 1];

  var pairsMap = {};
  var pairs = headerString.split('*');
  pairs.forEach(function(pairString) {
    var pair = pairString.split('=');

    if(pair.length == 2 && pair[1] !== undefined) {
      pairsMap[pair[0]] = pair[1];
    }
  });


  // value lists
  [self.correlation.COMPONENT_ID_FROM,
   self.correlation.COMPONENT_ID_TO,
   self.correlation.EXIT_CALL_TYPE_ORDER,
   self.correlation.EXIT_CALL_SUBTYPE_KEY].forEach(function(name) {
     var value = pairsMap[name];
     if(value !== undefined) {
       self.addSubHeader(name, value.split(','));
       delete pairsMap[name];
     }
   });


  // boolean values
  [self.correlation.DONOTRESOLVE,
   self.correlation.SNAPSHOT_ENABLE,
   self.correlation.DISABLE_TRANSACTION_DETECTION,
   self.correlation.DEBUG_ENABLED].forEach(function(name) {
     var value = pairsMap[name];
     if(value !== undefined) {
       self.addSubHeader(name, value.toLowerCase() === 'true');
       delete pairsMap[name];
     }
   });


  // string values
  for(var name in pairsMap) {
    self.addSubHeader(name, pairsMap[name]);
  }
};

CorrelationHeader.prototype.parse = function(headerString) {
  var self = this;

  // parse string header to subheader pairs
  self.parseHeaderString(headerString);

  // TODO: debug == true -> enable logging for only this transaction

  // disable transaction detection if subheader is set
  if(self.getSubHeader(self.correlation.DISABLE_TRANSACTION_DETECTION)) {
    self.txDetect = false;
    self.agent.logger.debug("CorrelationHeader.parse: transaction disabled from the originating tier, not processing");
    return false;
  }

  var accountGuid = self.getSubHeader(self.correlation.ACCOUNT_GUID);
  var controllerGuid = self.getSubHeader(self.correlation.CONTROLLER_GUID);
  var appId = self.getSubHeader(self.correlation.APP_ID);
  if(accountGuid && accountGuid != self.correlation.accountGuid) {
    self.agent.logger.debug("CorrelationHeader.parse: Remote account GUID [" + accountGuid + "] and local [" + self.correlation.accountGuid + "] do not match, not processing");
    return false;
  }

  if(controllerGuid && controllerGuid != self.correlation.controllerGuid) {
    self.agent.logger.debug("CorrelationHeader.parse: Remote controller GUID [" + controllerGuid + "] and local [" + self.correlation.controllerGuid + "] do not match, not processing");
    return false;
  }

  var crossAppCorrelation = false;
  if(appId && appId !== self.correlation.appId) {
    if(!(accountGuid && controllerGuid)) {
      self.agent.logger.debug("CorrelationHeader.parse: Remote app ID [" + appId + "] and local app ID [" + self.correlation.appId + "] do not match, not processing");
      return false;
    }
    else {
      crossAppCorrelation = true;
    }
  }

  // parse components
  var componentLinks = []; // needed for size
  var componentLink;
  var m;

  var cidFrom = self.getSubHeader(self.correlation.COMPONENT_ID_FROM) || [];
  var cidTo = self.getSubHeader(self.correlation.COMPONENT_ID_TO) || [];
  var eTypeOrder = self.getSubHeader(self.correlation.EXIT_CALL_TYPE_ORDER) || [];
  var eSubType = self.getSubHeader(self.correlation.EXIT_CALL_SUBTYPE_KEY) || [];

  if(cidFrom.length != cidTo.length || cidFrom.length != eTypeOrder.length) {
    self.agent.logger.warn("CorrelationHeader.parse: malformed caller chain");
    return false;
  }

  for(var i = 0; i < cidFrom.length; i++) {
    componentLink = {};
    componentLink[self.correlation.COMPONENT_ID_FROM] = cidFrom[i];
    componentLink[self.correlation.COMPONENT_ID_TO] = cidTo[i];
    componentLink[self.correlation.EXIT_CALL_TYPE_ORDER] = eTypeOrder[i];
    componentLink[self.correlation.EXIT_CALL_SUBTYPE_KEY] = eSubType[i];
    componentLinks.push(componentLink);
  }

  var lastComponent = componentLinks[componentLinks.length - 1];


  // report cross app correlation backend ID
  var crossAppCorrelationBackendId;
  if(crossAppCorrelation) {
    var lastComponentToId = lastComponent[self.correlation.COMPONENT_ID_TO];
    m = self.correlation.cidRegex.exec(lastComponentToId);
    if(m && m.length == 2) {
      crossAppCorrelationBackendId = parseInt(m[1]) | 0;
      if (crossAppCorrelationBackendId <= 0) {
        self.agent.logger.debug("CorrelationHeader.parse: unresolved backend ID is invalid for cross app correlation: "+ lastComponentToId);
        return false;
      }
      self.agent.logger.debug("CorrelationHeader.parse: crossAppCorrelationBackendId = " + crossAppCorrelationBackendId);
      self.crossAppCorrelationBackendId = crossAppCorrelationBackendId;
      self.crossAppCorrelation = true;
      return false;
    }
    m = self.correlation.cidResolvedCrossAppRegEx.exec(lastComponentToId);

    // If the incoming cidto is not understood by us, then just try to
    // resolve the incoming backend id to this app.  The legimate case
    // this approach handles is the case when lastComponentToId is a component
    // id from the upstream app.
    var crossAppCorrelationToAppId =
      (m && (m.length == 2)) ? (parseInt(m[1]) | 0) : 0;

    if (crossAppCorrelationToAppId == self.correlation.appId) {
      self.agent.logger.debug("CorrelationHeader.parse: foreign backend is already resolved to this application: " + lastComponentToId);
      self.crossAppCorrelation = true;
      return false;
    }

    crossAppCorrelationBackendId = self.getSubHeader(self.correlation.UNRESOLVED_EXIT_ID) | 0;
    if (crossAppCorrelationBackendId <= 0) {
      self.agent.logger.debug("CorrelationHeader.parse: cross app correlation header is missing valid " +
                            self.correlation.UNRESOLVED_EXIT_ID + " subheader:" + headerString);
      return false;
    }
    self.agent.logger.debug("CorrelationHeader.parse: re-resolving foreign backend " +
                          crossAppCorrelationBackendId + " to this application.");
    self.crossAppCorrelationBackendId = crossAppCorrelationBackendId;
    self.crossAppCorrelation = true;
    return false;
  }
  self.crossAppCorrelation = false;

  // add own component link
  if(self.getSubHeader(self.correlation.DONOTRESOLVE)) {
    cidFrom.push(lastComponent[self.correlation.COMPONENT_ID_TO]);
    cidTo.push(self.correlation.tierId.toString());
    eTypeOrder.push(lastComponent[self.correlation.EXIT_CALL_TYPE_ORDER]);

    componentLink = {};
    componentLink[self.correlation.COMPONENT_ID_FROM] = lastComponent[self.correlation.COMPONENT_ID_TO];
    componentLink[self.correlation.COMPONENT_ID_TO] = self.correlation.tierId.toString();
    componentLink[self.correlation.EXIT_CALL_TYPE_ORDER] = lastComponent[self.correlation.EXIT_CALL_TYPE_ORDER];
    componentLink[self.correlation.EXIT_CALL_SUBTYPE_KEY] = lastComponent[self.correlation.EXIT_CALL_SUBTYPE_KEY];
    componentLinks.push(componentLink);
  }


  // extract backend ID
  if(!self.getSubHeader(self.correlation.DONOTRESOLVE)) {
    m = self.correlation.cidRegex.exec(lastComponent[self.correlation.COMPONENT_ID_TO]);
    if(m && m.length == 2) {
      self.selfResolutionBackendId = parseInt(m[1]);
    }
  }

  // backend ID resolution
  if(self.getSubHeader(self.correlation.UNRESOLVED_EXIT_ID) !== undefined) {
    var unresolvedExitId = parseInt(self.getSubHeader(self.correlation.UNRESOLVED_EXIT_ID));
    if(unresolvedExitId > 0) {
      self.selfResolutionBackendId = unresolvedExitId;

      var correlatedComponentId = self.agent.transactionRegistry.resolvedBackendIds[unresolvedExitId];
      if(correlatedComponentId !== undefined && correlatedComponentId !== self.correlation.tierId) {
        self.agent.backendConnector.proxyTransport.sendSelfReResolution({
          backendId: unresolvedExitId
        });

        return false;
      }
      m = self.correlation.cidResolvedCrossAppRegEx.exec(lastComponent[self.correlation.COMPONENT_ID_TO]);
      if (m && (m.length == 2)) {
        self.agent.logger.debug("Re-resolving backend " + unresolvedExitId + " to this tier, it was resolve to an application.");
        self.agent.backendConnector.proxyTransport.sendSelfReResolution({
          backendId: unresolvedExitId
        });

        return false;
      }
    }
  }


  // apply header size limitations
  var size = 0;
  componentLinks.forEach(function(componentLink) {
    size += 30;
    size += componentLink[self.correlation.COMPONENT_ID_TO].length;
    size += componentLink[self.correlation.COMPONENT_ID_FROM].length;
    size += componentLink[self.correlation.EXIT_CALL_TYPE_ORDER].length;
  });

  if(size > 750 + 16 + 15 + 20) {
    return false;
  }

  return true;
};

CorrelationHeader.prototype.makeContinuingTransaction = function(transaction) {
  var self = this;

  var btId = self.getSubHeader(self.correlation.BT_ID);
  var btName = self.getSubHeader(self.correlation.BT_NAME);

  if(btId) {
    transaction.registrationId = btId;
  }
  else if(btName) {
    var btType = self.getSubHeader(self.correlation.ENTRY_POINT_TYPE);
    var btComp = self.getSubHeader(self.correlation.BT_COMPONENT_MAPPING);
    var mcType = self.getSubHeader(self.correlation.MATCH_CRITERIA_TYPE);
    var mcValue = self.getSubHeader(self.correlation.MATCH_CRITERIA_VALUE);

    if(mcType === self.correlation.MATCH_CRITERIA_TYPE_DISCOVERED) {
      transaction.isAutoDiscovered = true;
      transaction.namingSchemeType = mcValue;
    }
    else {
      transaction.isAutoDiscovered = false;
    }

    transaction.name = btName;
    transaction.entryType = btType;
    transaction.componentId = btComp;
  }
  else {
    self.agent.logger.warn("CorrelationHeader.makeContinuingTransaction: invalid correlation header, did not find BT id or name");
    return false;
  }

  transaction.guid = self.getSubHeader(self.correlation.REQUEST_GUID);
  transaction.skewAdjustedStartWallTime = self.getSubHeader(self.correlation.TIMESTAMP);

  return true;
};

var exitPointTypeToString = {
  'EXIT_HTTP': 'HTTP',
  'EXIT_CACHE': 'CACHE',
  'EXIT_DB': 'DB'
};

CorrelationHeader.prototype.build = function(transaction, exitCall, doNotResolve, isApiCall) {
  var self = this;

  // if the upstream tier has sent a notxdetect=true, then we should
  // also pass it on to all downstream tiers
  if (transaction.corrHeader && transaction.corrHeader.txDetect === false) {
    self.disableTransactionDetection();
  }

  if (transaction.ignore) {
    return;
  }

  // assign backendID and componentID to backend call if available
  self.agent.transactionRegistry.matchBackendCall(exitCall);


  var incomingHeader;
  if (transaction.corrHeader) {
    if (!transaction.corrHeader.crossAppCorrelation) {
      incomingHeader = transaction.corrHeader;
    }
  }

  // if backend call is not registered
  if(!exitCall.registrationId) {
    self.disableTransactionDetection();

    if(incomingHeader && incomingHeader.getSubHeader(self.correlation.DEBUG_ENABLED)) {
      self.addSubHeader(self.correlation.DEBUG_ENABLED, true);
    }

    self.agent.logger.debug("CorrelationHeader.build: disabling correlation header generated: " + self.getStringHeader());
    return;
  }


  // add app id subheader
  self.addSubHeader(self.correlation.ACCOUNT_GUID, self.correlation.accountGuid);
  self.addSubHeader(self.correlation.CONTROLLER_GUID, self.correlation.controllerGuid);
  self.addSubHeader(self.correlation.APP_ID, self.correlation.appId);


  // add BT related subheaders
  if(transaction.registrationId) {
    self.addSubHeader(self.correlation.BT_ID, transaction.registrationId);
  }
  else {
    self.addSubHeader(self.correlation.BT_NAME, transaction.name);
    self.addSubHeader(self.correlation.ENTRY_POINT_TYPE, transaction.entryType);
    self.addSubHeader(self.correlation.BT_COMPONENT_MAPPING, self.correlation.tierId);

    if(incomingHeader) {
      if(transaction.isAutoDiscovered) {
        self.addSubHeader(self.correlation.MATCH_CRITERIA_TYPE, self.correlation.MATCH_CRITERIA_TYPE_DISCOVERED);
        self.addSubHeader(self.correlation.MATCH_CRITERIA_VALUE, transaction.namingSchemeType);
      }
      else {
        self.addSubHeader(self.correlation.MATCH_CRITERIA_TYPE, self.correlation.MATCH_CRITERIA_TYPE_DISCOVERED);
        self.addSubHeader(self.correlation.MATCH_CRITERIA_VALUE, transaction.name);
      }
    }
    else {
      if(transaction.isAutoDiscovered) {
        self.addSubHeader(self.correlation.MATCH_CRITERIA_TYPE, self.correlation.MATCH_CRITERIA_TYPE_DISCOVERED);
        self.addSubHeader(self.correlation.MATCH_CRITERIA_VALUE, self.correlation.namingSchemeType);
      }
      else {
        self.addSubHeader(self.correlation.MATCH_CRITERIA_TYPE, self.correlation.MATCH_CRITERIA_TYPE_DISCOVERED);
        if (isApiCall) {
          self.addSubHeader(self.correlation.MATCH_CRITERIA_VALUE, transaction.name);
        } else {
          self.addSubHeader(self.correlation.MATCH_CRITERIA_VALUE, transaction.customMatch.btName);
        }
      }
    }
  }


  // add request guid subheader
  self.addSubHeader(self.correlation.REQUEST_GUID, transaction.guid);
  if (transaction.skewAdjustedStartWallTime) {
    self.addSubHeader(self.correlation.TIMESTAMP, transaction.skewAdjustedStartWallTime);
  }

  // add debug subheader
  if(incomingHeader && incomingHeader.getSubHeader(self.correlation.DEBUG_ENABLED)) {
    self.addSubHeader(self.correlation.DEBUG_ENABLED, true);
  }


  // add snapshot enable subheader
  var btInfoResponse = transaction.btInfoResponse;
  if((incomingHeader && incomingHeader.getSubHeader(self.correlation.SNAPSHOT_ENABLE)) ||
     (btInfoResponse && btInfoResponse.isSnapshotRequired)) {
    self.addSubHeader(self.correlation.SNAPSHOT_ENABLE, true);
  }

  // add unresoved exit id subheader
  var componentId;
  if(exitCall.componentId !== undefined) {
    var crossAppPrefixIfNeeded =
      exitCall.componentIsForeignAppId ? self.correlation.CID_IS_APPID_PREFIX : "";
    componentId = crossAppPrefixIfNeeded + exitCall.componentId;
  }
  else {
    componentId = "{[UNRESOLVED][" + exitCall.registrationId + "]}";
  }

  self.addSubHeader(self.correlation.UNRESOLVED_EXIT_ID, exitCall.registrationId);

  // add exit guid subheader
  self.addSubHeader(self.correlation.EXIT_POINT_GUID, exitCall.sequenceInfo);


  // add component link subheaders
  var compFrom;
  var compTo;
  var exitOrder;
  var exitSubType;

  if(incomingHeader) {
    compFrom = incomingHeader.getSubHeader(self.correlation.COMPONENT_ID_FROM) || [];
    compTo = incomingHeader.getSubHeader(self.correlation.COMPONENT_ID_TO) || [];
    exitOrder = incomingHeader.getSubHeader(self.correlation.EXIT_CALL_TYPE_ORDER) || [];
    exitSubType = incomingHeader.getSubHeader(self.correlation.EXIT_CALL_SUBTYPE_KEY) || [];
  }
  else {
    compFrom = [];
    compTo = [];
    exitOrder = [];
    exitSubType = [];
  }

  compFrom.push(self.correlation.tierId);
  compTo.push(componentId);
  exitOrder.push(exitPointTypeToString[exitCall.exitType]);
  exitSubType.push(exitCall.exitSubType);

  self.addSubHeader(self.correlation.COMPONENT_ID_FROM, compFrom);
  self.addSubHeader(self.correlation.COMPONENT_ID_TO, compTo);
  self.addSubHeader(self.correlation.EXIT_CALL_TYPE_ORDER, exitOrder);
  self.addSubHeader(self.correlation.EXIT_CALL_SUBTYPE_KEY, exitSubType);

  if(doNotResolve) {
    self.addSubHeader(self.correlation.DONOTRESOLVE, true);
  }

  self.agent.logger.debug("CorrelationHeader.build: correlation header generated: " + self.getStringHeader());
};

CorrelationHeader.prototype.disableTransactionDetection = function() {
  this.addSubHeader(this.correlation.DISABLE_TRANSACTION_DETECTION, true);
};
