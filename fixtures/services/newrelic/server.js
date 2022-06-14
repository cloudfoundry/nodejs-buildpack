const exec = require('child_process').exec;
const http = require('http');
const fs = require('fs');

const newrelic = require("newrelic");

const server = http.createServer((request, response) => {
  switch (request.url) {
    case '/process':
      response.end(JSON.stringify({
        version: process.version,
        env: process.env,
      }));
      break;

    default:
      response.end('Hello, World!');
  }
});

const port = process.env.PORT || 8080;
server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err);
  }

  console.log(`server is listening on ${port}`);
})
