Cloud Foundry NodeJS Buildpack
==============================

This is a fork of https://github.com/heroku/heroku-buildpack-nodejs designed to support Cloud Foundry
on premises installations.

This buildpack allows the NodeJS Buildpack to work with Cloud Foundry.

Notes to buildpack developers
=============================

* We bundle a version of node as ./bin/node for offline 'semantic versioning'. This node might be different from the one
in ./vendor.
* To run the heroku tests, use a linux machine. Even with a Darwin version of JQ, tests fail on MacOSX
