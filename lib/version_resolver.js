var semver = require('semver');

var requestedRange = process.argv[2];
var manifestVersionsJson = process.argv[3];
var defaultVersion = process.argv[4];
var resolvedVersion;

var manifestVersions = JSON.parse(manifestVersionsJson);
manifestVersions = manifestVersions.sort(semver.compare).reverse();

if (requestedRange === "") {
  console.log(defaultVersion);
}
else {
  resolvedVersion = manifestVersions.filter(function(version) {
      return semver.satisfies(version, requestedRange);
  })[0];

  console.log(resolvedVersion);
}
