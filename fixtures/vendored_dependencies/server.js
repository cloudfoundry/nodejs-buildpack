const http = require('http')
const port = Number(process.env.PORT || 8080);
const leftpad = require('leftpad')

const requestHandler = (request, response) => {
  console.log(request.url)
  response.end(leftpad(5, 10))
}

const server = http.createServer(requestHandler)

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err)
  }

  console.log(`server is listening on ${port}`)
})
