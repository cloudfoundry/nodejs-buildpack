const express = require("express");
const logfmt = require("logfmt");
const microtime = require('microtime');

const app = express();
app.use(logfmt.requestLogger());

app.get('/', (req, res) => {
  res.send('Hello, World!');
});

app.get('/microtime', (req, res) => {
  res.send('native time: ' + microtime.nowDouble().toString());
});

const port = Number(process.env.PORT || 5000);
app.listen(port, () => {
  console.log("Listening on " + port);
});
