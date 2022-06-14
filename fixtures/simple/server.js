const exec = require('child_process').exec;
const http = require('http');
const fs = require('fs');

const server = http.createServer((request, response) => {
  switch (request.url) {
    case '/dotnet':
      exec('dotnet --version', (error, stdout, stderr) => {
        if (error) {
          response.end('Error: ' + error);
        } else {
          response.end('dotnet: ' + stdout);
        }
      });
      break;

    case '/fs/Procfile':
      const procfile = fs.readFileSync('Procfile');
      response.end(procfile);
      break;

    case '/fs/package.json':
      const packagejson = fs.readFileSync('package.json');
      response.end(packagejson);
      break;

    case '/fs/server.js':
      const serverjs = fs.readFileSync('server.js');
      response.end(serverjs);
      break;

    case '/process':
      response.end(JSON.stringify({
        version: process.version,
        env: process.env,
      }));
      break;

    default:
      response.end('Hello world!');
  }
});

const port = process.env.PORT || 8080;
server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err);
  }

  console.log(`server is listening on ${port}`);
})
