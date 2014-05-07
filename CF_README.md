Testing for Cloud Foundry
=========================

This build pack is designed to work in Cloud Foundry 'Offline' mode


To run the tests:

    $ BUNDLE_GEMFILE=cf.Gemfile bundle
    $ BUNDLE_GEMFILE=cf.Gemfile rspec cf_spec/
    $ BUNDLE_GEMFILE=cf.Gemfile BUILDPACK_MODE=offline rspec cf_spec/
