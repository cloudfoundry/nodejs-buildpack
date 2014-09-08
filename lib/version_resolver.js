var semver = require('semver');

var fs = require('fs');
var path = require('path');
var file = path.join(__dirname, '..', 'files', 'versions.json');
var semverRange = process.argv[2];
var version;
var allVersions;
var stableVersions;

fs.readFile(file, 'utf8', function (err, data) {
  if (!err) {
    var node = JSON.parse(data);

    if (semverRange === "null") {
      version = node.stable;
    } else {

      allVersions = node.all.reverse();
      stableVersions = allVersions.filter(function(version) {
        return version.match(/[0-9]+\.[0-9]*[02468]\.[0-9]+/g)
      });
      version = (stableVersions.concat(allVersions)).filter(function(version) {
        return semver.satisfies(version, semverRange);
      })[0];
    }
  }

  console.log(version);
});
