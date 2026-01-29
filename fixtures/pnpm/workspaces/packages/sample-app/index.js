const express = require('express');
const sampleLib = require('@sample/sample-lib');
const app = express();
const port = process.env.PORT || 8080;

app.get('/', (req, res) => {
  res.send('Hello from Workspace! ' + sampleLib.message);
});

app.get('/check', (req, res) => {
  res.json({ message: sampleLib.message });
});

app.listen(port, () => {
  console.log('server is listening on ' + port);
});
