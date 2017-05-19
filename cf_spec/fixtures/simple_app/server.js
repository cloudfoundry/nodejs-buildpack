const http = require('http')
const port = process.env.PORT || 8080

const requestHandler = (request, response) => {
  response.end(`MaxOldSpace: ${process.env.npm_config_max_old_space_size}`)
}

const server = http.createServer(requestHandler)

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err)
  }

  console.log(`server is listening on ${port}`)
})
