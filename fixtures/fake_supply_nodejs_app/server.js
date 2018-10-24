var exec = require('child_process').exec;
var express = require("express");
var logfmt = require("logfmt");
var app = express();

app.use(logfmt.requestLogger());

app.get('/', function(req, res) {
  exec('dotnet --version', (error, stdout, stderr) => {
    if (error) {
      res.send('Error: ' + error);
    } else {
      res.send('dotnet: ' + stdout);
    }
  });
});

var port = Number(process.env.PORT || 5000);
app.listen(port, function() {
  console.log("Listening on " + port);
});
