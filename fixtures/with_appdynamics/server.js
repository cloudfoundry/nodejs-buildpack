var fs = require('fs')
var child_process = require('child_process');
var appdynamics = require("appdynamics").profile({
  controllerHostName: process.env.APPDYNAMICS_CONTROLLER_HOST_NAME,
  controllerPort: process.env.APPDYNAMICS_CONTROLLER_PORT,
  controllerSslEnabled: process.env.APPDYNAMICS_CONTROLLER_SSL_ENABLED,
  accountName: process.env.APPDYNAMICS_AGENT_ACCOUNT_NAME,
  accountAccessKey: process.env.APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY,
  applicationName: process.env.APPDYNAMICS_AGENT_APPLICATION_NAME,
  tierName: process.env.APPDYNAMICS_AGENT_TIER_NAME,
  nodeName: process.env.APPDYNAMICS_AGENT_NODE_NAME,
});
var express = require("express");
var logfmt = require("logfmt");
var app = express();

app.use(logfmt.requestLogger());

app.get('/', function(req, res) {
  res.send('Hello, World!');
});

app.get('/name', function(req, res) {
 res.send(String(process.env.APPDYNAMICS_AGENT_APPLICATION_NAME));
});

app.get('/logs', function(req, res) {
  child_process.exec('cat /tmp/appd/*/*.log', {}, function(err, stdout, stderr) {
    console.log(err, stdout, stderr);
    if (err) {
      res.send(err);
    }
    res.send(stdout);
  });
});

var port = Number(process.env.PORT || 5000);
app.listen(port, function() {
  console.log("Listening on " + port);
});
