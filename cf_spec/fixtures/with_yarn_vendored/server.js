// web.js
var express = require("express");
var logfmt = require("logfmt");
var microtime = require('microtime');
var app = express();

app.use(logfmt.requestLogger());

app.get('/', function(req, res) {
  res.send('Hello, World!');
});
app.get('/microtime', function(req, res) {
  res.send('native time: ' + microtime.nowDouble().toString());
});

var port = Number(process.env.PORT || 5000);
app.listen(port, function() {
  console.log("Listening on " + port);
});
