$: << 'cf_spec'
require 'bundler/setup'
require 'json'
require 'fileutils'
require 'tmpdir'
require 'open3'

describe "Appdynamics Installer" do
  let(:buildpack_dir)           { File.join(File.expand_path(File.dirname(__FILE__)), '..', '..') }
  let(:build_dir)               { Dir.mktmpdir }
  let(:app_name)                { 'unit-test-app' }
  let(:tier_name)               { 'tier-name' }
  let(:node_name)               { 'node-name' }
  let(:host_name)               { '1.1.1.1' }
  let(:port)                    { '1234' }
  let(:account_name)            { 'customer' }
  let(:account_access_key)      { 'fffff-fffff-fffff-fffff' }
  let(:profile_d_appdynamics)   { File.join(build_dir, '.profile.d', 'appdynamics-setup.sh') }

  before do
    ENV["BP_DIR"] = buildpack_dir
      vcap_application =  {
        "application_name"=> app_name
      }
    ENV["VCAP_APPLICATION"] = vcap_application.to_json
  end

  after do
    FileUtils.rm_rf(build_dir) if defined? build_dir
    ENV["VCAP_APPLICATION"] = nil
  end

  subject do
    Dir.chdir(buildpack_dir) do
      @stdout, @stderr, @status = Open3.capture3("lib/vendor/appdynamics/install.sh #{build_dir}")
    end
  end

  context 'vcap services' do
    before do
      ENV["VCAP_SERVICES"] = vcap_services.to_json
    end

    after do
      ENV["VCAP_SERVICES"] = nil
    end

    context 'contains appdynamics' do
      let(:vcap_services) {
        {
          "appdynamics"=>[{
            "credentials"=> {
              "host-name"=> "test-host-name",
              "port"=> "1234",
              "account-name"=> "test-customer",
              "account-access-key"=> "appdynamics_license_key_set_by_service_binding"
            }
          }]
        }
      }

      context 'APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY and APPDYNAMICS_AGENT_ACCOUNT_NAME not set' do
        it "sets the APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY variable from VCAP_SERVICES" do
          subject
          expect(@status).to be_success
          profile_d_contents = File.read(profile_d_appdynamics)
          expect(profile_d_contents).to include('export APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY=appdynamics_license_key_set_by_service_binding')
        end

        it "sets the APPDYNAMICS_AGENT_ACCOUNT_NAME variable" do
          subject
          expect(@status).to be_success
          profile_d_contents = File.read(profile_d_appdynamics)
          expect(profile_d_contents).to include('export APPDYNAMICS_AGENT_ACCOUNT_NAME=test-customer')
        end
      end

      context 'APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY set' do
        before do
          ENV["APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY"] = "license_key_in_env_var"
        end

        after do
          ENV["APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY"] = nil
        end

        it "sets the APPDYNAMICS_AGENT_APPLICATION_NAME variable" do
          subject
          expect(@status).to be_success
          profile_d_contents = File.read(profile_d_appdynamics)
          expect(profile_d_contents).to include('export APPDYNAMICS_AGENT_APPLICATION_NAME=unit-test-app')
        end
      end

      context 'APPDYNAMICS_AGENT_APPLICATION_NAME set' do
        before do
          ENV["APPDYNAMICS_AGENT_APPLICATION_NAME"] = "unit-test-app"
        end

        after do
          ENV["APPDYNAMICS_AGENT_APPLICATION_NAME"] = nil
        end

        it "sets the APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY variable from VCAP_SERVICES" do
          subject
          expect(@status).to be_success
          profile_d_contents = File.read(profile_d_appdynamics)
          expect(profile_d_contents).to include('export APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY=appdynamics_license_key_set_by_service_binding')
        end
      end

      context 'APPDYNAMICS_AGENT_APPLICATION_NAME and APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY set' do
        before do
          ENV["APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY"] = "appdynamics_license_key_set_by_service_binding"
          ENV["APPDYNAMICS_AGENT_APPLICATION_NAME"] = "unit-test-app"
        end

        after do
          ENV["APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY"] = nil
          ENV["APPDYNAMICS_AGENT_APPLICATION_NAME"] = nil
        end
      end
    end

    context 'VCAP_SERVICES is not present in environment' do
      let(:vcap_services) { {} }

      subject do
        Dir.chdir(buildpack_dir) do
          @stdout, @stderr, @status = Open3.capture3("unset VCAP_SERVICES; lib/vendor/appdynamics/install.sh #{build_dir}")
        end
      end

      it 'does not create .profile.d/appdynamics-setup.sh file' do
        subject
        expect(@status).to be_success
        expect(File.exist?(profile_d_appdynamics)).to be_falsey
      end
    end
  end
end
