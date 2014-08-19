$: << 'cf_spec'
require 'spec_helper'

describe 'CF NodeJS Buildpack' do
  subject(:app) { Machete.deploy_app(app_name) }
  let(:browser) { Machete::Browser.new(app) }

  context 'with cached buildpack dependencies' do
    context 'in an offline environment', if: Machete::BuildpackMode.offline? do
      let(:app_name) { 'node_web_app' }

      specify do
        expect(app).to be_running

        browser.visit_path('/')
        expect(browser).to have_body('Hello, World!')

        expect(app.host).not_to have_internet_traffic
      end
    end
  end

  context 'without cached buildpack dependencies' do
    context 'in an online environment', if: Machete::BuildpackMode.online? do

      context 'app has dependencies' do
        let(:app_name) { 'node_web_app' }

        specify do
          expect(app).to be_running

          browser.visit_path('/')
          expect(browser).to have_body('Hello, World!')
        end
      end

      context 'app has no dependencies' do
        let(:app_name) { 'node_web_app_no_dependencies' }

        specify do
          expect(Dir.exists?("cf_spec/fixtures/#{app_name}/node_modules")).to eql false
          expect(app).to be_running

          browser.visit_path('/')
          expect(browser).to have_body('Hello, World!')
        end
      end
    end
  end
end
