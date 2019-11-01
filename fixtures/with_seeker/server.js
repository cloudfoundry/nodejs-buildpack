var fs = require('fs')
var express = require("express");
var logfmt = require("logfmt");
var app = express();

app.use(logfmt.requestLogger());

app.get('/', function(req, res) {
  res.send('Hello, World!');
});

app.get('/config', function(req, res) {
  res.send('SEEKER_SERVER_URL: ' + process.env.SEEKER_SERVER_URL);
});

var port = Number(process.env.PORT || 5000);
app.listen(port, function() {
  console.log("Listening on " + port);
});
