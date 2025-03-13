/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

function ExitCall(exitCallInfo) {
  this.user = exitCallInfo.user;
  this.command = exitCallInfo.command;
  this.commandArgs = exitCallInfo.commandArgs;
  this.stackTrace = exitCallInfo.stackTrace;
  this.error = exitCallInfo.error;
  this.url = exitCallInfo.url;
  this.method = exitCallInfo.method;
  this.requestHeaders = exitCallInfo.requestHeaders;
  this.responseHeaders = exitCallInfo.responseHeaders;
  this.statusCode = exitCallInfo.statusCode;
  this.backendConfig = exitCallInfo.backendConfig;
  this.id = exitCallInfo.id;
  this.ms = exitCallInfo.ms;
  this.ts = exitCallInfo.ts;
  this.threadId = exitCallInfo.threadId;
  this.exitType = exitCallInfo.exitType;
  this.exitSubType = exitCallInfo.exitSubType;
  this.backendName = exitCallInfo.backendName;
  this.isSql = exitCallInfo.isSql;
  this.identifyingProperties = exitCallInfo.identifyingProperties;
  this.registrationId = exitCallInfo.registrationId;
  this.componentId = exitCallInfo.componentId;
  this.componentIsForeignAppId = exitCallInfo.componentIsForeignAppId;
  this.backendIdentifier = exitCallInfo.backendIdentifier;
  this.sequenceInfo = exitCallInfo.sequenceInfo;
  this.category = exitCallInfo.category;
  this.protocol = exitCallInfo.protocol;
  this.vendor = exitCallInfo.vendor;
  this.group = exitCallInfo.group;
  this.exitCallGuid = exitCallInfo.exitCallGuid;

  if (!this.exitSubType) {
    this.exitSubType = this.exitType && this.exitType.replace('EXIT_', '');
  }
}

function backendInfoToString(exitPointType, exitPointSubtype, identifyingProperties) {
  var identifyingPropertyNames = Object.keys(identifyingProperties);
  identifyingPropertyNames.sort();

  var backedInfoParts = [exitPointType.toString()];
  if (exitPointSubtype) backedInfoParts.push(exitPointSubtype.toString());

  identifyingPropertyNames.forEach(function (propertyName) {
    var propertyValue = identifyingProperties[propertyName];
    backedInfoParts.push(propertyName);
    backedInfoParts.push(propertyValue.toString());
  });

  var result = JSON.stringify(backedInfoParts);
  return result;
}

ExitCall.prototype.getBackendInfoString = function () {
  return backendInfoToString(this.exitType, this.exitSubType, this.identifyingProperties);
};

exports.ExitCall = ExitCall;
exports.backendInfoToString = backendInfoToString;
