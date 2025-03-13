/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
'use strict';

function EumCookie(agent, transaction, request, response, isHttps) {
  this.agent = agent;
  this.eum = agent.eum;

  this.transaction = transaction;
  this.request = request;
  this.response = response;
  this.isHttps = isHttps;

  this.isAjax = undefined;
  this.isMobile = undefined;
  this.ajaxHeaderCounter = 0;

  this.cookieValue = undefined;
  this.keyForm = 'short';
  this.guid = undefined;
}
exports.EumCookie = EumCookie;

EumCookie.prototype.now = function() {
  return Date.now();
};

EumCookie.prototype.addSubCookie = function(name, value) {
  var self = this;

  self.agent.logger.debug("eum-cookie: addSubCookie: Setting value " + name + ": " + value);

  if(self.isAjax) {
    self.response.setHeader(self.eum.ADRUM_PREFIX + (self.ajaxHeaderCounter++), name + ':' + encodeURIComponent(value));
  }
  else {
    if(!self.cookieValue) {
      var referer = self.request.headers.referer;
      self.cookieValue = 'R:' + (referer ? referer.length : '0');
    }

    self.cookieValue += '|' + name + ':' + encodeURIComponent(value);
    if (self.agent.opts.alwaysAddEumMetadataInHttpHeaders) {
      self.response.setHeader(self.eum.ADRUM_PREFIX + (self.ajaxHeaderCounter++), name + ':' + encodeURIComponent(value));
    }
  }
};

EumCookie.prototype.setCookie = function() {
  var self = this;

  // headers already set
  if(self.isAjax) {
    return;
  }

  self.setCookieHeader(self.eum.ADRUM_MASTER_COOKIE_NAME, self.cookieValue);
};

EumCookie.prototype.setCookieHeader = function(name, val) {
  var self = this;

  var pairs = [name + '=' + val];

  pairs.push('Path=/');
  pairs.push('Expires=' + (new Date(self.now() + 30000)).toUTCString());
  if(self.isHttps) pairs.push('Secure');

  var rumCookie = pairs.join('; ');

  self.agent.logger.debug("eum-cookie: setCookieHeader: Setting cookie: " + rumCookie);

  var cookies = self.response.getHeader('Set-Cookie');
  if(cookies) {
    if(Array.isArray(cookies)) {
      rumCookie = cookies.concat(rumCookie);
    }
    else {
      rumCookie = [cookies, rumCookie];
    }
  }

  self.response.setHeader('Set-Cookie', rumCookie);
};

EumCookie.prototype.build = function() {
  var self = this;

  // skip if transaction is ignored
  if(self.transaction.ignore) {
    return false;
  }

  // is ajax request
  var adrumHeader = self.request.headers.adrum;
  self.isAjax = !!(adrumHeader && adrumHeader.indexOf('isAjax:true') != -1);

  // is mobile request
  var adrumHeader1 = self.request.headers.adrum_1;
  self.isMobile = !!(adrumHeader1 && adrumHeader1.indexOf('isMobile:true') != -1);

  if (self.isMobile) {
    self.keyForm = 'long';
  }
  return self.setFieldValues();
};

EumCookie.prototype.setFieldValues = function() {
  var self = this;

  // generate GUID
  self.guid = self.eum.generateGuid();

  // if a snapshot is to be sent, it will include this guid
  self.transaction.eumGuid = self.guid;

  // set values
  // BT_DURATION_KEY is not set, as we can't know the duration at this point
  self.addSubCookie(self.eum.CLIENT_REQUEST_GUID_KEY[self.keyForm], self.guid);

  if (self.transaction.registrationId) {
    self.addSubCookie(self.eum.BT_ID_KEY[self.keyForm], self.transaction.registrationId);
  }

  var btInfoResponse = self.transaction.btInfoResponse;
  if (btInfoResponse && btInfoResponse.averageResponseTimeForLastMinute) {
    self.addSubCookie(self.eum.BT_ART_KEY[self.keyForm], btInfoResponse.averageResponseTimeForLastMinute.toString());
  }

  if (self.transaction.hasSnapshotWithCallGraphData()) {
    self.addSubCookie(self.eum.SERVER_SNAPSHOT_TYPE_KEY[self.keyForm], 'f');
  }

  var hasErrors = self.transaction.error || (self.transaction.exitCalls &&
    self.transaction.exitCalls.filter(function(call) { return call.error; }).length);
  if (hasErrors) {
    self.addSubCookie(self.eum.HAS_ENTRY_POINT_ERRORS_KEY[self.keyForm], 'e');
  }

  if (self.eum.globalAccountName) {
    self.addSubCookie(self.eum.GLOBAL_ACCOUNT_NAME_KEY[self.keyForm], self.eum.globalAccountName);
  }

  // add cookie to response
  self.setCookie();

  return true;
};
