Testing for Cloud Foundry
=========================

This build pack is designed to work in Cloud Foundry 'Offline' mode


To run the tests:

    $ BUNDLE_GEMFILE=cf.Gemfile bundle
    $ BUNDLE_GEMFILE=cf.Gemfile rspec cf_spec/
    $ BUNDLE_GEMFILE=cf.Gemfile BUILDPACK_MODE=offline rspec cf_spec/




Notes for buildpack developers
==============================

* We bundle a version of node as ./bin/node for offline 'semantic versioning'. This node might be different from the one
in ./vendor.
* To run the heroku tests, use a linux machine. Even with a Darwin version of JQ, tests fail on MacOSX

