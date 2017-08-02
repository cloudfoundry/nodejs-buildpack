# Cloud Foundry Node.js Buildpack

[![CF Slack](https://www.google.com/s2/favicons?domain=www.slack.com) Join us on Slack](https://cloudfoundry.slack.com/messages/buildpacks/)

A Cloud Foundry [buildpack](http://docs.cloudfoundry.org/buildpacks/) for Node based apps.

This is based on the [Heroku buildpack](https://github.com/heroku/heroku-buildpack-nodejs).

Additional documentation can be found at the [CloudFoundry.org](http://docs.cloudfoundry.org/buildpacks/node/index.html).

### Buildpack User Documentation

Official buildpack documentation can be found at [node buildpack docs](http://docs.cloudfoundry.org/buildpacks/node/index.html).

### Building the Buildpack

1. Install buildpack-packager

  ```shell
  (cd src/staticfile/vendor/github.com/cloudfoundry/libbuildpack/packager/buildpack-packager && go install)
  ```

1. Build the buildpack

  ```shell
  buildpack-packager [ --cached | --uncached ]
  ```

1. Use in Cloud Foundry

  Upload the buildpack to your Cloud Foundry and optionally specify it by name

  ```bash
  cf create-buildpack [BUILDPACK_NAME] [BUILDPACK_ZIP_FILE_PATH] 1
  cf push my_app -b [BUILDPACK_NAME]
  ```

### Testing
Buildpacks use the [Cutlass](https://github.com/cloudfoundry/libbuildpack/cutlass) framework for running integration tests.

To test this buildpack, run the following command from the buildpack's directory:

1. Install ginkgo

  ```bash
  (cd src/nodejs/vendor/github.com/onsi/ginkgo/ginkgo && go install)
  ```

1. Run unit tests

  ```bash
  ./scripts/unit.sh
  ```

1. Run integration tests

  ```bash
  ./scripts/integration.sh
  ```

More information can be found on github [cutlass](https://github.com/cloudfoundry/libbuildpack/cutlass).

### Contributing

Find our guidelines [here](./CONTRIBUTING.md).

### Help and Support

Join the #buildpacks channel in our [Slack community](http://slack.cloudfoundry.org/)

### Reporting Issues

Open an issue on this project

### Active Development

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066)
