var _ = require('lodash');

process.stdin.resume();
process.stdin.setEncoding('utf8');

var data = "";

process.stdin.on('data', function(chunk) {
    data += chunk;
});

process.stdin.on('end', function() {
    var app = JSON.parse(data);
    
    var result = {
        name: app.name,
        version: app.version
    };
    
    if (app.dependencies) {
        result.dependencies = clean_deps(app.dependencies); 
    }
    
    var info = process.env.VCAP_APPLICATION
    
    console.log(typeof info, info);
    
    console.log(JSON.stringify(result));
});

function clean_deps(deps) {
    return _.mapValues(deps, function(dep) {
        var result = {
            version: dep.version,
            from: dep.from
        };
        
        if (dep.dependencies) {
            result.dependencies = clean_deps(dep.dependencies);
        }
        
        return result;
    });
}