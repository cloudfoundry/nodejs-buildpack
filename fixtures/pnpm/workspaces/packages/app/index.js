var express = require('express');
var utils = require('@pnpm-sample/utils');

var app = express();

app.get('/', function(req, res) {
  res.send('Hello from pnpm workspaces! ' + utils.getVersion());
});

var port = process.env.PORT || 3000;
app.listen(port, function() {
  console.log("Listening on " + port);
});
