$: << 'cf_spec'
require 'bundler/setup'
require 'json'
require 'fileutils'
require 'tmpdir'
require 'open3'

describe "New Relic Installer" do
  let(:buildpack_dir) { File.join(File.expand_path(File.dirname(__FILE__)), '..', '..') }
  let(:build_dir)     { Dir.mktmpdir }
  let(:app_name)      { 'unit-test-app' }
  let(:app_guid)      { 'fff-fff-fff-fff' }
  let(:profile_d_new_relic) { File.join(build_dir, '.profile.d', 'new-relic-setup.sh') }

  before do
    ENV["BP_DIR"] = buildpack_dir
      vcap_application =  {
        "application_id": app_guid,
        "application_name": app_name
      }
    ENV["VCAP_APPLICATION"] = vcap_application.to_json
  end

  after do
    FileUtils.rm_rf(build_dir) if defined? build_dir
    ENV["VCAP_APPLICATION"] = nil
  end

  subject do
    Dir.chdir(buildpack_dir) do
      @stdout, @stderr, @status = Open3.capture3("lib/vendor/new_relic/install.sh #{build_dir}")
    end
  end

  context 'vcap services' do
    before do
      ENV["VCAP_SERVICES"] = vcap_services.to_json
    end

    after do
      ENV["VCAP_SERVICES"] = nil
    end

    context 'contains newrelic' do
      let(:vcap_services) {
        {
          "newrelic":[{
            "credentials": {
              "licenseKey": "new_relic_license_key_set_by_service_binding"
            }
          }]
        }
      }

      context 'NEW_RELIC_LICENSE_KEY and NEW_RELIC_APP_NAME not set' do
        it "sets the NEW_RELIC_LICENSE_KEY variable from VCAP_SERVICES" do
          subject
          expect(@status).to be_success
          profile_d_contents = File.read(profile_d_new_relic)
          expect(profile_d_contents).to include('export NEW_RELIC_LICENSE_KEY=new_relic_license_key_set_by_service_binding')
        end

        it "sets the NEW_RELIC_APP_NAME variable" do
          subject
          expect(@status).to be_success
          profile_d_contents = File.read(profile_d_new_relic)
          expect(profile_d_contents).to include('export NEW_RELIC_APP_NAME=unit-test-app_fff-fff-fff-fff')
        end
      end

      context 'NEW_RELIC_LICENSE_KEY set' do
        before do
          ENV["NEW_RELIC_LICENSE_KEY"] = "license_key_in_env_var"
        end

        after do
          ENV["NEW_RELIC_LICENSE_KEY"] = nil
        end

        it "sets the NEW_RELIC_APP_NAME variable" do
          subject
          expect(@status).to be_success
          profile_d_contents = File.read(profile_d_new_relic)
          expect(profile_d_contents).to include('export NEW_RELIC_APP_NAME=unit-test-app_fff-fff-fff-fff')
        end

        it "does not modify NEW_RELIC_LICENSE_KEY" do
          subject
          expect(@status).to be_success
          profile_d_contents = File.read(profile_d_new_relic)
          expect(profile_d_contents).to_not include('NEW_RELIC_LICENSE_KEY')
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
          subject
          expect(@status).to be_success
          profile_d_contents = File.read(profile_d_new_relic)
          expect(profile_d_contents).to_not include('NEW_RELIC_APP_NAME')
        end

        it "sets the NEW_RELIC_LICENSE_KEY variable from VCAP_SERVICES" do
          subject
          expect(@status).to be_success
          profile_d_contents = File.read(profile_d_new_relic)
          expect(profile_d_contents).to include('export NEW_RELIC_LICENSE_KEY=new_relic_license_key_set_by_service_binding')
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

        it 'does not create .profile.d/new-relic-setup.sh file' do
          subject
          expect(@status).to be_success
          expect(File.exist?(profile_d_new_relic)).to be_falsey
        end
      end
    end

    context 'does not contain newrelic' do
      let(:vcap_services) { {} }

      it 'does not create .profile.d/new-relic-setup.sh file' do
        subject
        expect(@status).to be_success
        expect(File.exist?(profile_d_new_relic)).to be_falsey
      end
    end

    context 'VCAP_SERVICES is not present in environment' do
      let(:vcap_services) { {} }

      subject do
        Dir.chdir(buildpack_dir) do
          @stdout, @stderr, @status = Open3.capture3("unset VCAP_SERVICES; lib/vendor/new_relic/install.sh #{build_dir}")
        end
      end

      it 'does not create .profile.d/new-relic-setup.sh file' do
        subject
        expect(@status).to be_success
        expect(File.exist?(profile_d_new_relic)).to be_falsey
      end
    end
  end
end
