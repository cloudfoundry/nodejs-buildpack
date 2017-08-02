var bson = require('bson-ext');
var http = require('http');

http.createServer(function (req, res) {
     res.writeHead(200, {'Content-Type': 'text/plain'});
     for (var env in process.env) {
         res.write(env + '=' + process.env[env] + '\n');
     }
     res.end();
}).listen(process.env.PORT || 3000);
