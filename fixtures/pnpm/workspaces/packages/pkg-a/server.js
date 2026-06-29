const http = require('http');
const pkgB = require('pkg-b');
const port = process.env.PORT || 8080;

const requestHandler = (request, response) => {
  response.end('Hello from Workspace! ' + pkgB.hello());
}

const server = http.createServer(requestHandler);

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err);
  }
  console.log('server is listening on ' + port);
});
