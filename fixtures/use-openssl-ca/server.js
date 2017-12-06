const fs = require('fs')
const http = require('http')
const https = require('https')
const port = process.env.PORT || 8080

const listenErrHandler = (port) => (err) => {
  if (err) {
    return console.log('something bad happened', err)
  }
  console.log(`server is listening on ${port}`)
}

// Generate certificates:
// openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -subj '/CN=localhost' -keyout key.pem -out cert.pem

const key = fs.readFileSync('key.pem')
const cert = fs.readFileSync('cert.pem')
const httpsServer = https.createServer({key, cert, ca: cert}, (request, response) => {
  console.log('Received https request')
  response.writeHead(200, {'X-MyHeader': 'MyData'})
  response.end('Response over self signed https\n')
}).listen(8888, listenErrHandler(8888))

const options = {
    hostname: 'localhost',
    port: 8888,
    path: '/',
    agent: false
}

const httpServer = http.createServer((request, response) => {
  console.log('Received http request')
  https.get(options, (res) => {
    console.log('statusCode:', res.statusCode)
    response.writeHead(res.statusCode, res.headers)
    var data = ''
    res.on('data', (d) => data += d)
    res.on('end', (d) => {
      data += d
      response.end(data)
    })
  }).on('error', (e) => {
    console.log(e)
    response.end(e)
  })
}).listen(port, listenErrHandler(port))
