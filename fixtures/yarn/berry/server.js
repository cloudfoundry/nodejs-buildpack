const http = require("http");

const port = process.env.PORT || 8080;

http.createServer((request, response) => {
  response.writeHead(200, { "Content-Type": "text/plain" });
  response.end("Hello, World!");
}).listen(port, () => {
  console.log("Listening on " + port);
});
