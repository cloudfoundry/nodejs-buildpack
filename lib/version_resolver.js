var semver = require('semver');

var semverRange = process.argv[2];
var versionsAsJson = process.argv[3];
var stableVersion = process.argv[4];
var version;
var allVersions;
var stableVersions;

var allVersions = JSON.parse(versionsAsJson);
allVersions = allVersions.sort(semver.compare).reverse();

if (semverRange === "") {
  version = stableVersion;
} else {
  stableVersions = allVersions.filter(function(version) {
    return version.match(/[0-9]+\.[0-9]*[02468]\.[0-9]+/g)
  });
  version = (stableVersions.concat(allVersions)).filter(function(version) {
    return semver.satisfies(version, semverRange);
  })[0];
}

console.log(version);
