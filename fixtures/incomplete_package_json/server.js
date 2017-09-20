const http = require('http')
const port = Number(process.env.PORT || 8080);

const requestHandler = (request, response) => {
  response.end('Hello, World!');
}

const server = http.createServer(requestHandler)

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err)
  }

  console.log(`server is listening on ${port}`)
})
