$: << 'cf_spec'
require 'spec_helper'

describe 'CF NodeJS Buildpack' do
  subject(:app) { Machete.deploy_app(app_name) }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  describe 'switching stacks' do
    subject(:app) { Machete.deploy_app(app_name, stack: 'lucid64') }
    let(:app_name) { 'node_web_app_no_dependencies' }

    specify do
      expect(app).to be_running(60)

      browser.visit_path('/')
      expect(browser).to have_body('Hello, World!')

      replacement_app = Machete::App.new(app_name, Machete::Host.create, stack: 'cflinuxfs2')

      app_push_command = Machete::CF::PushApp.new
      app_push_command.execute(replacement_app)

      expect(replacement_app).to be_running(60)

      browser.visit_path('/')
      expect(browser).to have_body('Hello, World!')
      expect(app).not_to have_logged('Restoring node modules from cache')
    end
  end

  describe 'with non-specific version' do
    context 'app specifies version range' do
      let(:app_name) { 'node_web_app_with_version_range' }

      specify do
        expect(app).to be_running

        browser.visit_path('/')
        expect(browser).to have_body('Hello, World!')
      end
    end
  end

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
