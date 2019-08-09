const bcrypt = require('bcrypt');
const express = require("express");
const logfmt = require("logfmt");

const port = process.env.PORT || 8080;
const app = express();

app.use(logfmt.requestLogger());

app.get('/', function(req, res) {
  res.send('Hello, World!');
});

app.listen(port, function() {

  bcrypt.hash('myPassword', 10, function(err, hash) {
    console.log('inside of hash method')
  });
  console.log("Listening on " + port);
});

