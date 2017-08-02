$: << 'cf_spec'
require 'spec_helper'

describe 'CF NodeJS Buildpack' do
  subject(:app) { Machete.deploy_app(app_name) }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  context 'deploying a NodeJS app with AppDynamics' do
    let(:app_name) { 'with_appdynamics' }

    context 'when App Dynamics environment variables are set' do
      subject(:app) do
        Machete.deploy_app(app_name)
      end

      it 'tries to talk to AppDynamics with host-name from the env vars' do
        expect(app).to be_running
        expect(app).to have_logged('appdynamics v')
        expect(app).to have_logged('starting control socket')
        expect(app).to have_logged('controllerHost: \'test-host\'')
      end
    end
  end
end
