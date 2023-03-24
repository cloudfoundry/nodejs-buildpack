const fs = require('fs');
const express = require("express");
const logfmt = require("logfmt");
const features = require('cpu-features')();

const app = express();
app.use(logfmt.requestLogger());

app.get('/prepost', (req, res) => {
  const text_1 = fs.readFileSync("text_1.txt");
  const text_2 = fs.readFileSync("text_2.txt");
  res.end(`Text: ${text_1}\nText: ${text_2}\n`);
});

app.get('/mysql', (req, res) => {
  const Client = require('mysql').Client;

  var client = new Client();
  client.user = 'root';
  client.password = 'root';

  res.end('Successfully created mysql client');
});

app.get('/', (req, res) => {
  res.send('Hello, World!');
});

const port = Number(process.env.PORT || 5000);
app.listen(port, () => {
  console.log("Listening on " + port);
});
