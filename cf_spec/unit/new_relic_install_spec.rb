$: << 'cf_spec'
require 'bundler/setup'
require 'json'
require 'fileutils'

def new_relic_license_key
  `source lib/vendor/new_relic/install.sh && echo $NEW_RELIC_LICENSE_KEY`.strip
end

def new_relic_app_name
  `source lib/vendor/new_relic/install.sh && echo $NEW_RELIC_APP_NAME`.strip
end

describe "New Relic Installer" do
  let(:buildpack_dir) { File.join(File.expand_path(File.dirname(__FILE__)), '..', '..') }
  let(:app_name)      { 'unit-test-app' }
  let(:app_guid)      { 'fff-fff-fff-fff' }

  before do
    ENV["BP_DIR"] = buildpack_dir
      vcap_application =  {
        "application_id": app_guid,
        "application_name": app_name
      }
    ENV["VCAP_APPLICATION"] = vcap_application.to_json
  end

  after do
    ENV["VCAP_APPLICATION"] = nil
  end

  context 'vcap services contains newrelic' do
    before do
      vcap_services = {"newrelic":[{
        "credentials": {
          "licenseKey": "new_relic_license_key_set_by_service_binding"
        }}]
      }
      ENV["VCAP_SERVICES"] = vcap_services.to_json
    end

    after do
      ENV["VCAP_SERVICES"] = nil
    end

    context 'NEW_RELIC_LICENSE_KEY not set' do
      it "sets the NEW_RELIC_LICENSE_KEY variable from VCAP_SERVICES" do
        buildpack_dir = File.join(File.expand_path(File.dirname(__FILE__)), '..', '..')
        Dir.chdir(buildpack_dir) do
          expect(new_relic_license_key).to eq("new_relic_license_key_set_by_service_binding")
        end
      end
    end

    context 'NEW_RELIC_LICENSE_KEY set' do
      before do
        ENV["NEW_RELIC_LICENSE_KEY"] = "license_key_in_env_var"
      end

      after do
        ENV["NEW_RELIC_LICENSE_KEY"] = nil
      end

      it "does not modify NEW_RELIC_LICENSE_KEY" do
        buildpack_dir = File.join(File.expand_path(File.dirname(__FILE__)), '..', '..')
        Dir.chdir(buildpack_dir) do
          expect(new_relic_license_key).to eq("license_key_in_env_var")
        end
      end
    end

    context 'NEW_RELIC_APP_NAME not set' do
      it "sets the NEW_RELIC_APP_NAME variable" do
        buildpack_dir = File.join(File.expand_path(File.dirname(__FILE__)), '..', '..')
        Dir.chdir(buildpack_dir) do
          expect(new_relic_app_name).to eq(app_name + '_' + app_guid)
        end
      end
    end

    context 'NEW_RELIC_APP_NAME set' do
      before do
        ENV["NEW_RELIC_APP_NAME"] = "new_relic_app_name"
      end

      after do
        ENV["NEW_RELIC_APP_NAME"] = nil
      end

      it "does not modify NEW_RELIC_APP_NAME" do
        buildpack_dir = File.join(File.expand_path(File.dirname(__FILE__)), '..', '..')
        Dir.chdir(buildpack_dir) do
          expect(new_relic_app_name).to eq("new_relic_app_name")
        end
      end
    end

    context 'NEW_RELIC_APP_NAME and NEW_RELIC_LICENSE_KEY set' do
      before do
        ENV["NEW_RELIC_LICENSE_KEY"] = "license_key_in_env_var"
        ENV["NEW_RELIC_APP_NAME"] = "new_relic_app_name"
      end

      after do
        ENV["NEW_RELIC_LICENSE_KEY"] = nil
        ENV["NEW_RELIC_APP_NAME"] = nil
      end

      it "does not modify NEW_RELIC_LICENSE_KEY" do
        buildpack_dir = File.join(File.expand_path(File.dirname(__FILE__)), '..', '..')
        Dir.chdir(buildpack_dir) do
          expect(new_relic_app_name).to eq("new_relic_app_name")
        end
      end

      it "does not modify NEW_RELIC_APP_NAME" do
        buildpack_dir = File.join(File.expand_path(File.dirname(__FILE__)), '..', '..')
        Dir.chdir(buildpack_dir) do
          expect(new_relic_app_name).to eq("new_relic_app_name")
        end
      end
    end
  end

  context 'vcap services does not contain newrelic' do
    it "does not set the NEW_RELIC_LICENSE_KEY variable" do
      Dir.chdir(buildpack_dir) do
        expect(new_relic_license_key).to eq("")
      end
    end

    it "does not set the NEW_RELIC_APP_NAME variable" do
      Dir.chdir(buildpack_dir) do
        expect(new_relic_app_name).to eq("")
      end
    end
  end
end
