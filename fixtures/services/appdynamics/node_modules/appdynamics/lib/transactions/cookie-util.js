/*
Copyright (c) AppDynamics, Inc., and its affiliates
2015
All Rights Reserved
 */
function trim(str) {
  return str.replace(/^\s+|\s+$/g, '');
}

function parseCookies(req) {
  var cookieHeader = req.headers['cookie'];
  var cookieMap;

  // if app is using Express with the cookieParser middleware,
  // use the pre-parsed cookies
  if (req.cookies) {
    return req.cookies;
  }

  // otherwise, try to parse them directly
  cookieMap = {};
  if(cookieHeader) {
    cookieHeader.split(';').forEach(function(cookie) {
      var parts = cookie.split('=');
      if(parts.length == 2 && parts[0] && parts[1]) {
        cookieMap[trim(parts[0])] = decodeURIComponent(trim(parts[1]));
      }
    });
  }

  return cookieMap;
}

exports.parseCookies = parseCookies;