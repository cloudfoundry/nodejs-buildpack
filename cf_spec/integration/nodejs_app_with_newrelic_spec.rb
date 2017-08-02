$: << 'cf_spec'
require 'spec_helper'

describe 'CF NodeJS Buildpack' do
  subject(:app) { Machete.deploy_app(app_name) }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  context 'deploying a NodeJS app with NewRelic' do
    let(:app_name) { 'with_newrelic' }

    context 'when New Relic environment variables are set' do
      subject(:app) do
        Machete.deploy_app(app_name)
      end

      it 'tries to talk to NewRelic with the license key from the env vars' do
        expect(app).to be_running
        expect(app).to have_logged('&license_key=fake_new_relic_key2')
        expect(app).to_not have_logged('&license_key=fake_new_relic_key1')
      end
    end

    context 'when newrelic.js sets license_key' do
      let(:app_name) { 'with_newrelic_js' }

      it 'tries to talk to NewRelic with the license key from newrelic.js' do
        expect(app).to be_running
        expect(app).to have_logged('&license_key=fake_new_relic_key1')
      end
    end
  end
end
