/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

var cookies = require('./cookie-util');

function TransactionNaming(agent) {
  this.agent = agent;

  this.namingProps = undefined;
}
exports.TransactionNaming = TransactionNaming;


TransactionNaming.prototype.init = function() {
  var self = this;

  self.agent.on('configUpdated', function() {
    var props = self.agent.configManager.getConfigValue('txConfig.nodejsWeb.discoveryConfig.namingScheme.properties');
    self.namingProps = self.formatNamingProps(props);
  });
};

TransactionNaming.prototype.formatNamingProps = function(props) {
  props = this.agent.configManager.convertListToMap(props);

  // Headers are lowercased in Node.js
  if(props['uri-suffix-scheme'] === 'header-value' && props['suffix-key']) {
    props['suffix-key'] = props['suffix-key'].toLowerCase();
  }

  return props;
};

TransactionNaming.prototype.createHttpTransactionName = function(req, transaction) {
  var matchName = transaction.customMatch && transaction.customMatch.btName
    ,   namingProps = transaction.customNaming || this.namingProps;

  return this.createHttpTransactionNameFromNamingScheme(req, namingProps, matchName);
};

TransactionNaming.prototype.createHttpTransactionNameFromNamingScheme = function(req, namingProps, btName) {
  var self = this, suffix;

  var baseName = btName || self.createWithUriSegments(req, namingProps);

  var uriSuffixScheme = namingProps['uri-suffix-scheme'];
  var suffixKey = namingProps['suffix-key'];
  var delimiter = namingProps['delimiter'];
  delimiter = delimiter ? delimiter : "";

  if (!uriSuffixScheme)
    return baseName;
  if (uriSuffixScheme !== 'method' && !suffixKey)
    return baseName;

  switch(uriSuffixScheme) {
  case 'first-n-segments':
  case 'last-n-segments':
    suffix = this.getUriSegments(req.url, uriSuffixScheme, suffixKey);
    baseName = btName ? btName + '.' + suffix : suffix;
    break;
  case 'uri-segment-number':
    suffix = self.transformUriSegments(req, baseName, suffixKey, delimiter);
    baseName = btName ? btName + '.' + suffix : suffix;
    break;
  case 'param-value':
    baseName = self.appendParamValues(req, baseName, suffixKey);
    break;
  case 'header-value':
    baseName = self.appendHeaderValues(req, baseName, suffixKey);
    break;
  case 'cookie-value':
    baseName = self.appendCookieValues(req, baseName, suffixKey);
    break;
  case 'method':
    baseName = self.appendMethod(req, baseName);
    break;
  }

  return baseName;
};



TransactionNaming.prototype.createWithUriSegments = function(req, namingProps) {
  var self = this;

  var segmentLength = namingProps['segment-length'];
  var uriLength = namingProps['uri-length'] || 'first-n-segments';

  return self.getUriSegments(req.url, uriLength, segmentLength);
};

TransactionNaming.prototype.getUriSegments = function(url, uriLength, segmentLength) {
  var parts = require('url').parse(url || '/');
  var segments = parts.pathname.split('/').splice(1);
  var i;

  // make sure last slash doesn't cause another empty segment
  if(segments.length > 1 && segments[segments.length - 1] === "") {
    segments.pop();
  }

  var name = "";
  segmentLength = segmentLength ? Math.min(segmentLength, segments.length) : segments.length;
  if(uriLength === 'first-n-segments') {
    for(i = 0; i < segmentLength; i++) {
      name += "/" + segments[i];
    }
  }
  else if(uriLength === 'last-n-segments') {
    for(i = segments.length - segmentLength; i < segments.length; i++) {
      name += "/" + segments[i];
    }
  }

  return name;
};


TransactionNaming.prototype.transformUriSegments = function(req, baseName, suffixKey, delimiter) {
  var parts = require('url').parse(req.url || '/');
  var segments = parts.pathname.split('/').splice(1);

  // make sure last slash doesn't cause another empty segment
  if(segments.length > 1 && segments[segments.length - 1] === "") {
    segments.pop();
  }

  var setmentNumberArr = suffixKey.replace(/ /g,'').split(',');
  var setmentNumberMap = {};
  setmentNumberArr.forEach(function(segmentNumberStr) {
    var segmentNumber = parseInt(segmentNumberStr);
    if(segmentNumber > 0) {
      setmentNumberMap[segmentNumber] = true;
    }
  });

  var newSegments = [];
  for(var i = 0; i < segments.length; i++) {
    if(setmentNumberMap[i + 1]) {
      newSegments.push(segments[i]);
    }
  }

  if(newSegments.length == 0) {
    return baseName;
  }

  return newSegments.join(delimiter);
};


TransactionNaming.prototype.appendParamValues = function(req, baseName, suffixKey) {
  var paramNames = suffixKey.replace(/ /g,'').split(',');

  // GET params
  var parts = require('url').parse(req.url || '/', true);
  var paramValues = [];
  paramNames.forEach(function(paramName) {
    var paramValue = parts.query[paramName];
    if(paramValue) {
      paramValues.push(paramValue);
    }
  });

  // POST params only available if Express is used
  if(req.body) {
    paramNames.forEach(function(paramName) {
      var paramValue = req.body[paramName];
      if(paramValue) {
        paramValues.push(paramValue);
      }
    });
  }

  if(paramValues.length == 0) {
    return baseName;
  }

  return baseName + '.' + paramValues.join('.');
};


TransactionNaming.prototype.appendHeaderValues = function(req, baseName, suffixKey) {
  var headerNames = suffixKey.replace(/ /g,'').split(',');

  var headerValues = [];
  headerNames.forEach(function(headerName) {
    var headerValue = req.headers[headerName.toLowerCase()];
    if(headerValue) {
      headerValues.push(headerValue);
    }
  });

  if(headerValues.length == 0) {
    return baseName;
  }

  return baseName + '.' + headerValues.join('.');
};


TransactionNaming.prototype.appendCookieValues = function(req, baseName, suffixKey) {
  var cookieNames = suffixKey.replace(/ /g,'').split(',');

  var cookieValues = [];
  var cookieMap = cookies.parseCookies(req);
  cookieNames.forEach(function(cookieName) {
    var cookieValue = cookieMap[cookieName];
    if(cookieValue) {
      cookieValues.push(cookieValue);
    }
  });

  if(cookieValues.length == 0) {
    return baseName;
  }

  return baseName + '.' + cookieValues.join('.');
};


TransactionNaming.prototype.appendMethod = function(req, baseName) {
  return baseName + '.' + req.method;
};

