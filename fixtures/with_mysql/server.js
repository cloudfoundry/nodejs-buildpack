var http = require('http')
var port = process.env.PORT || 3306

var requestHandler = (request, response) => {
  response.end(`Hey`)
}

var server = http.createServer(requestHandler)

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err)
  }
  console.log(`server is listening on ${port}`)
})

var Client = require('mysql').Client,
    client = new Client();

client.user = 'root';
client.password = 'root';

