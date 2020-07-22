const fs = require("fs")
const express = require("express");
const logfmt = require("logfmt");
const app = express();

app.use(logfmt.requestLogger());

app.get('/', function(req, res) {
  res.send('Hello, World!');
});

app.get('/config', function(req, res) {
  res.send('SEEKER_SERVER_URL: ' + process.env.SEEKER_SERVER_URL);
});

app.get('/self', function(req, res) {
  fs.readFile(__filename, 'utf-8', (err, data) => {
    if (err) throw err;
    res.send(data);
  });
});

var port = Number(process.env.PORT || 5000);
app.listen(port, function() {
  console.log("Listening on " + port);
});
