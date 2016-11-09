$: << 'cf_spec'
require 'spec_helper'

describe 'CF NodeJS Buildpack' do
  subject(:app)           { Machete.deploy_app(app_name) }
  let(:browser)           { Machete::Browser.new(app) }
  let(:buildpack_dir)     { File.join(File.dirname(__FILE__), '..', '..') }
  let(:version_file)      { File.join(buildpack_dir, 'VERSION') }
  let(:buildpack_version) { File.read(version_file).strip }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  context 'when specifying a range for the nodeJS version in the package.json' do
    let(:app_name) { 'node_web_app_with_version_range' }

    it 'resolves to a nodeJS version successfully' do
      expect(app).to be_running
      expect(app).to have_logged /Downloading and installing node 4\.\d+\.\d+/

      browser.visit_path('/')
      expect(browser).to have_body('Hello, World!')
    end
  end

  context 'when specifying a version 5 for the nodeJS version in the package.json' do
    let(:app_name) { 'node_web_app_with_version_5' }

    it 'resolves to a nodeJS version successfully' do
      expect(app).to be_running
      expect(app).to have_logged /Downloading and installing node 5\.\d+\.\d+/

      browser.visit_path('/')
      expect(browser).to have_body('Hello, World!')
    end
  end

  context 'when specifying a version 6 for the nodeJS version in the package.json' do
    let(:app_name) { 'node_web_app_with_version_6' }

    it 'resolves to a nodeJS version successfully' do
      expect(app).to be_running
      expect(app).to have_logged /Downloading and installing node 6\.\d+\.\d+/

      browser.visit_path('/')
      expect(browser).to have_body('Hello, World!')
    end
  end

  context 'when not specifying a nodeJS version in the package.json' do
    let(:app_name) { 'node_web_app_without_version' }

    it 'resolves to the stable nodeJS version successfully' do
      expect(app).to be_running
      expect(app).to have_logged /Downloading and installing node 4\.\d+\.\d+/

      browser.visit_path('/')
      expect(browser).to have_body('Hello, World!')
    end

    it 'correctly displays the buildpack version' do
      expect(app).to have_logged "node.js #{buildpack_version}"
    end
  end

  context 'with an unreleased nodejs version' do
    let(:app_name) { 'node_web_app_with_unreleased_version' }

    it 'displays a nice error messages and gracefully fails' do
      expect(app).to_not be_running
      expect(app).to have_logged 'Downloading and installing node 9000.0.0'
      expect(app).to_not have_logged 'Downloaded ['
      expect(app).to have_logged /DEPENDENCY MISSING IN MANIFEST: node 9000\.0\.0.*-----> Build failed/m
    end
  end

  context 'with an unsupported, but released, nodejs version' do
    let(:app_name) { 'node_web_app_with_unsupported_version' }

    it 'displays a nice error messages and gracefully fails' do
      expect(app).to_not be_running
      expect(app).to have_logged 'Downloading and installing node 4.1.1'
      expect(app).to_not have_logged 'Downloaded ['
      expect(app).to have_logged /DEPENDENCY MISSING IN MANIFEST: node 4\.1\.1.*-----> Build failed/m
    end
  end

  context 'with an app that has vendored dependencies' do
    let(:app_name) { 'node_web_app_with_vendored_dependencies' }

    it 'does not output protip that recommends user vendors dependencies' do
      expect(app).not_to have_logged("PRO TIP: It is recommended to vendor the application's Node.js dependencies")
    end

    context 'with an uncached buildpack', :uncached do
      it 'successfully deploys and includes the dependencies' do
        expect(app).to be_running

        browser.visit_path('/')
        expect(browser).to have_body('Hello, World!')
        expect(app).to have_logged(/Downloaded \[https:\/\/.*\]/)
      end
    end

    context 'with a cached buildpack', :cached do
      it 'deploys without hitting the internet' do
        expect(app).to be_running

        browser.visit_path('/')
        expect(browser).to have_body('Hello, World!')

        expect(app).not_to have_internet_traffic
        expect(app).to have_logged(/Downloaded \[file:\/\/.*\]/)
      end
    end
  end

  context 'with an app with no vendored dependencies' do
    let(:app_name) { 'node_web_app_no_vendored_dependencies' }

    it 'successfully deploys and vendors the dependencies' do
      expect(app).to be_running
      expect(Dir).to_not exist("cf_spec/fixtures/#{app_name}/node_modules")
      expect(app).to have_file '/app/node_modules'

      browser.visit_path('/')
      expect(browser).to have_body('Hello, World!')
    end

    it "uses a proxy during staging if present", :uncached do
      expect(app).to use_proxy_during_staging
    end

    it 'outputs protip that recommends user vendors dependencies' do
      expect(app).to have_logged("PRO TIP: It is recommended to vendor the application's Node.js dependencies")
    end
  end

  context 'with an incomplete node_modules directory' do
    let (:app_name) { 'node_web_app_with_incomplete_node_modules' }

    it 'downloads missing dependencies from package.json' do
      expect(app).to be_running
      expect(Dir).to_not exist("cf_spec/fixtures/node_web_app_with_incomplete_node_modules/node_modules/hashish")
      expect(app).to have_file("/app/node_modules/hashish")
      expect(app).to have_file("/app/node_modules/express")
    end
  end

  context 'with an incomplete package.json' do
    let (:app_name) { 'node_web_app_with_incomplete_package_json' }

    it 'does not overwrite the vendored modules not listed in package.json' do
      expect(app).to be_running

      replacement_app = Machete::App.new(app_name)
      app_push_command = Machete::CF::PushApp.new
      app_push_command.execute(replacement_app)
      expect(replacement_app).to be_running

      expect(app).to have_file("/app/node_modules/logfmt")
      expect(app).to have_file("/app/node_modules/express")
      expect(app).to have_file("/app/node_modules/hashish")
    end
  end

  context 'with a cached buildpack in an air gapped environment', :cached do
    before(:each) do
      `cf unbind-staging-security-group public_networks`
      `cf unbind-staging-security-group dns`
    end

    after(:each) do
      `cf bind-staging-security-group public_networks`
      `cf bind-staging-security-group dns`
    end

    context 'with no npm version specified' do
      let (:app_name) { 'node_web_app_airgapped_no_npm_version' }

      subject(:app) do
        Machete.deploy_app(app_name, env: {'BP_DEBUG' => '1'})
      end

      it 'is running with the default version of npm' do
        expect(app).to be_running
        expect(app).not_to have_internet_traffic
        expect(app).to have_logged("Using default npm version")
        expect(app).to have_logged('DEBUG: default_version_for node is')
      end
    end

    context 'with invalid npm version specified' do
      let (:app_name) { 'node_web_app_airgapped_invalid_npm_version' }

      it 'is not running and prints an error message' do
        expect(app).not_to be_running
        expect(app).to have_logged("We're unable to download the version of npm")
      end
    end
  end
end
