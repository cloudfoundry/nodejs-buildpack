const http = require('http');
const leftpad = require('leftpad');

const server = http.createServer((request, response) => {
  console.log(request.url);

  response.end(leftpad(5, 10));
})

const port = Number(process.env.PORT || 8080);
server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err);
  }

  console.log(`server is listening on ${port}`);
});
