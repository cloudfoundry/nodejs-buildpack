# Contributing

## Run the tests

See the [Machete](https://github.com/cf-buildpacks/machete) CF buildpack test framework for more information.

## Pull Requests

1. Fork the project
1. Submit a pull request

**NOTE:** When submitting a pull request, *please make sure to target the `develop` branch*, so that your changes are up-to-date and easy to integrate with the most recent work on the buildpack. Thanks!
## Updating nodejs versions

The buildpacks supports disconnected environments, which means `semver.io` cannot be reached for node version resolution. If a new version of node is added to the `manifest.yml` it must be added to `files/versions.json`.

*NOTE:* If it happens to be the latest stable release, update the `stable` key in `files/versions.json`.

