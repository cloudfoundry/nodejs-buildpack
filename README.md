# CloudFoundry build pack: Node.js

<<<<<<< HEAD
A Cloud Foundry [buildpack](http://docs.cloudfoundry.org/buildpacks/) for Node based apps.
=======
![heroku-buildpack-featuerd](https://cloud.githubusercontent.com/assets/51578/6953435/52e1af5c-d897-11e4-8712-35fbd4d471b1.png)

This is the official [Heroku buildpack](http://devcenter.heroku.com/articles/buildpacks) for Node.js apps. If you fork this repository, please **update this README** to explain what your fork does and why it's special.
>>>>>>> upstream/master

This is based on the [Heroku buildpack] (https://github.com/heroku/heroku-buildpack-nodejs).

Additional documentation can be found at the [CloundFoundry.org](http://docs.cloudfoundry.org/buildpacks/).

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

<<<<<<< HEAD
  ```shell
  BUNDLE_GEMFILE=cf.Gemfile bundle
  ```

1. Build the buildpack

  ```shell
  BUNDLE_GEMFILE=cf.Gemfile bundle exec buildpack-packager [ cached | uncached ]
  ```

1. Use in Cloud Foundry

  Upload the buildpack to your Cloud Foundry and optionally specify it by name
  ```bash
  cf create-buildpack custom_node_buildpack node_buildpack-offline-custom.zip 1
  cf push my_app -b custom_node_buildpack
  ```

## Supported binary dependencies

The NodeJS buildpack only supports the two most recent stable patches for each dependency in the [manifest.yml](manifest.yml).
=======
```
heroku buildpack:set BUILDPACK_URL=https://github.com/heroku/heroku-buildpack-nodejs#v63 -a my-app
git commit -am "empty" --allow-empty
git push heroku master
```
>>>>>>> upstream/master

If you want to use previously supported dependency versions, provide the `--use-custom-manifest=manifest-including-unsupported.yml` option to `buildpack-packager`.

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

## Reporting Issues

Open an issue on this project

<<<<<<< HEAD
## Active Development
=======
### Configure npm with .npmrc

Sometimes, a project needs custom npm behavior to set up proxies,
use a different registry, etc. For such behavior,
just include an `.npmrc` file in the root of your project:

```
# .npmrc
registry = 'https://custom-registry.com/'
```

### Reasonable defaults for concurrency

This buildpack adds two environment variables: `WEB_MEMORY` and `WEB_CONCURRENCY`.
You can set either of them, but if unset the buildpack will fill them with reasonable defaults.

- `WEB_MEMORY`: expected memory use by each node process (in MB, default: 512)
- `WEB_CONCURRENCY`: recommended number of processes to Cluster based on the current environment

Clustering is not done automatically; concurrency should be part of the app,
usually via a library like [throng](https://github.com/hunterloftis/throng).
Apps without any clustering mechanism will remain unaffected by these variables.

This behavior allows your app to automatically take advantage of larger containers.
The default settings will cluster
1 process on a 1X dyno, 2 processes on a 2X dyno, and 12 processes on a PX dyno.

For example, when your app starts:

```
app[web.1]: Detected 1024 MB available memory, 512 MB limit per process (WEB_MEMORY)
app[web.1]: Recommending WEB_CONCURRENCY=2
app[web.1]:
app[web.1]: > example-concurrency@1.0.0 start /app
app[web.1]: > node server.js
app[web.1]: Listening on 51118
app[web.1]: Listening on 51118
```

Notice that on a 2X dyno, the
[example concurrency app](https://github.com/heroku-examples/node-concurrency)
listens on two processes concurrently.

### Chain Node with multiple buildpacks

This buildpack automatically exports node, npm, and any node_modules binaries
into the `$PATH` for easy use in subsequent buildpacks.

## Feedback

Having trouble? Dig it? Feature request?

- [help.heroku.com](https://help.heroku.com/)
- [@hunterloftis](http://twitter.com/hunterloftis)
- [github issues](https://github.com/heroku/heroku-buildpack-nodejs/issues)

## Hacking

To make changes to this buildpack, fork it on Github. Push up changes to your fork, then create a new Heroku app to test it, or configure an existing app to use your buildpack:

```
# Create a new Heroku app that uses your buildpack
heroku create --buildpack <your-github-url>

# Configure an existing Heroku app to use your buildpack
heroku buildpacks:set <your-github-url>

# You can also use a git branch!
heroku buildpacks:set <your-github-url>#your-branch
```

## Testing

The buildpack tests use [Docker](https://www.docker.com/) to simulate
Heroku's Cedar and Cedar-14 containers.

To run the test suite:

```
make test
```

Or to just test in cedar or cedar-14:

```
make test-cedar-10
make test-cedar-14
```
>>>>>>> upstream/master

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066)
