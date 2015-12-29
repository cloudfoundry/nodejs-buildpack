# Cloud Foundry Node.js Buildpack
[![CF Slack](https://s3.amazonaws.com/buildpacks-assets/buildpacks-slack.svg)](http://slack.cloudfoundry.org)

A Cloud Foundry [buildpack](http://docs.cloudfoundry.org/buildpacks/) for Node based apps.

This is based on the [Heroku buildpack] (https://github.com/heroku/heroku-buildpack-nodejs).

Additional documentation can be found at the [CloudFoundry.org](http://docs.cloudfoundry.org/buildpacks/).

## Usage

This buildpack will get used if you have a `package.json` file in your project's root directory.

```bash
cf push my_app -b https://github.com/cloudfoundry/buildpack-nodejs.git
```

## Disconnected environments
To use this buildpack on Cloud Foundry, where the Cloud Foundry instance limits some or all internet activity, please read the [Disconnected Environments documentation](https://github.com/cf-buildpacks/buildpack-packager/blob/master/doc/disconnected_environments.md).

### Vendoring app dependencies
As stated in the [Disconnected Environments documentation](https://github.com/cf-buildpacks/buildpack-packager/blob/master/doc/disconnected_environments.md), your application must 'vendor' it's dependencies.

For the NodeJS buildpack, use ```npm```:

```shell 
cd <your app dir>
npm install # vendors into /node_modules
```

```cf push``` uploads your vendored dependencies.

### Additional extensions
In cached mode, [use the semver node_module](bin/compile#L30-32) (as opposed to http://semver.io) to resolve the correct node version. The semver.io service has an additional preference for stable versions not present in the node module version. We wrap the node module using [lib/version_resolver.js](lib/version_resolver.js) to add back this functionality.

## Building
1. Make sure you have fetched submodules

  ```bash
  git submodule update --init
  ```
1. Get latest buildpack dependencies

  ```shell
  BUNDLE_GEMFILE=cf.Gemfile bundle
  ```

1. Build the buildpack

  ```shell
  BUNDLE_GEMFILE=cf.Gemfile bundle exec buildpack-packager [ --cached | --uncached ]
  ```

1. Use in Cloud Foundry

  Upload the buildpack to your Cloud Foundry and optionally specify it by name
  
  ```bash
  cf create-buildpack custom_node_buildpack node_buildpack-offline-custom.zip 1
  cf push my_app -b custom_node_buildpack
  ```

## Supported binary dependencies

The buildpack only supports the stable patches for each dependency listed in the [manifest.yml](manifest.yml) and [releases page](https://github.com/cloudfoundry/nodejs-buildpack/releases).


If you try to use a binary that is not currently supported, staging your app will fail and you will see the following error message:

```
       Could not get translated url, exited with: DEPENDENCY_MISSING_IN_MANIFEST: ...
 !
 !     exit
 !
Staging failed: Buildpack compilation step failed
```

## Testing
Buildpacks use the [Machete](https://github.com/cloudfoundry/machete) framework for running integration tests.

To test a buildpack, run the following command from the buildpack's directory:

```
BUNDLE_GEMFILE=cf.Gemfile bundle exec buildpack-build
```

More options can be found on Machete's [Github page.](https://github.com/cloudfoundry/machete)


## Options

### Specify a node version

Set engines.node in package.json to the semver range
(or specific version) of node you'd like to use.
(It's a good idea to make this the same version you use during development)

```json
"engines": {
  "node": "0.11.x"
}
```

```json
"engines": {
  "node": "0.10.33"
}
```

## Contributing

Find our guidelines [here](./CONTRIBUTING.md).

## Help and Support

Join the #buildpacks channel in our [Slack community] (http://slack.cloudfoundry.org/) 


## Reporting Issues

Open an issue on this project

## Active Development

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066)
