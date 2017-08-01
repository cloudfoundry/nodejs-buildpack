const fs = require('fs')
const http = require('http')
const port = process.env.PORT || 8080

const requestHandler = (request, response) => {
  const text_1 = fs.readFileSync("text_1.txt")
  const text_2 = fs.readFileSync("text_2.txt")
  response.end(`Text: ${text_1}\nText: ${text_2}\n`)
}

const server = http.createServer(requestHandler)

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err)
  }

  console.log(`server is listening on ${port}`)
})
