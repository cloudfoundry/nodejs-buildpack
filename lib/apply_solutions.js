var _ = require('lodash');

process.stdin.resume();
process.stdin.setEncoding('utf8');

var data = "";

process.stdin.on('data', function(chunk) {
    data += chunk;
});

process.stdin.on('end', function() {
    var app = JSON.parse(data);
    var solutions = JSON.parse(process.argv[2]);
    
    if (app.dependencies && _.isArray(solutions)) {
        solutions.forEach(function(solution) {
            var parts = solution.split("@");
            var name = parts[0];
            var version = parts[1];
            
            if (app.dependencies[name]) {
                app.dependencies[name] = version;
            }
        });
    }
    
    console.log(JSON.stringify(app));
});