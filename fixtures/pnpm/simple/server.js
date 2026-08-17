const express = require('express');
const logfmt = require('logfmt');
const app = express();
const port = process.env.PORT || 8080;

app.use(logfmt.requestLogger());

app.get('/', (req, res) => {
  res.send('Hello, World!');
});

app.listen(port, () => {
  console.log('server is listening on ' + port);
});