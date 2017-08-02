$: << 'cf_spec'
require 'bundler/setup'
require 'json'
require 'fileutils'
require 'tmpdir'
require 'open3'

describe "Appdynamics Installer" do
  let(:buildpack_dir)           { File.join(File.expand_path(File.dirname(__FILE__)), '..', '..') }
  let(:vcap_application) do
    {
      "application_name" => "unit-test-app"
    }
  end

  before do
    ENV['VCAP_APPLICATION'] = vcap_application.to_json
    ENV['VCAP_SERVICES'] = vcap_services.to_json
    ENV['APPDYNAMICS_CONTROLLER_HOST_NAME'] = ""
    ENV['APPDYNAMICS_CONTROLLER_PORT'] = ""
    ENV['APPDYNAMICS_AGENT_ACCOUNT_NAME'] = ""
    ENV['APPDYNAMICS_CONTROLLER_SSL_ENABLED'] = ""
    ENV['APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY'] = ""
    ENV['APPDYNAMICS_AGENT_APPLICATION_NAME'] = ""
    ENV['APPDYNAMICS_AGENT_TIER_NAME'] = ""
    ENV['APPDYNAMICS_AGENT_NODE_NAME_PREFIX'] = ""
  end

  subject do
    Dir.chdir(buildpack_dir) do
      @stdout, @stderr, @status = Open3.capture3(" . profile/appdynamics-setup.sh;
                                                 echo $APPDYNAMICS_CONTROLLER_HOST_NAME;
                                                 echo $APPDYNAMICS_CONTROLLER_PORT;
                                                 echo $APPDYNAMICS_AGENT_ACCOUNT_NAME;
                                                 echo $APPDYNAMICS_CONTROLLER_SSL_ENABLED;
                                                 echo $APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY;
                                                 echo $APPDYNAMICS_AGENT_APPLICATION_NAME;
                                                 echo $APPDYNAMICS_AGENT_TIER_NAME;
                                                 echo $APPDYNAMICS_AGENT_NODE_NAME_PREFIX")
    end
  end

  context 'vcap_services is not present in environment' do
    let(:vcap_services) { "{}" }

    it 'does not set the AppDynamics environment variables' do
      subject
      expect(@status).to be_success
      expect(@stdout).to eq("\n\n\n\n\n\n\n\n")
    end
  end

  context 'vcap services contains appdynamics service' do
    let(:vcap_services) do
      {
        "elephantsql": [{}],
        "appdynamics": [ {
                         "credentials":{
                           "host-name":"test-host",
                           "port":"1234",
                           "account-name":"test-account",
                           "ssl-enabled":"true",
                           "account-access-key":"test-key"
                         }
                      } ]
      }
    end


    it "sets the appropriate environment variables from VCAP_SERVICES AppDynamics credentials" do
      subject
      expect(@status).to be_success
      expect(@stdout).to eq(<<~STDOUT
                            test-host
                            1234
                            test-account
                            true
                            test-key
                            unit-test-app
                            unit-test-app
                            unit-test-app
                           STDOUT
                           )
    end
  end
end
