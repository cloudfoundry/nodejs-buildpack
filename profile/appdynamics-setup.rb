#!/usr/bin/env ruby
if ENV["APPD_AGENT"].nil?
  f = $stdout
  $stdout = open("/tmp/appdynamics-setup-profile.out.log", "w")
  $stderr.reopen("/tmp/appdynamics-setup-profile.err.log", "w")
  require 'json'

  vcap = JSON.load(ENV['VCAP_SERVICES']) rescue {}
  credentials = nil

  offering_name = vcap.keys.select { |k| k =~ /app(\-)?dynamics/ }.first
  if offering_name
    credentials = vcap.dig(offering_name, 0, 'credentials')
  elsif vcap['user-provided']
    service = vcap['user-provided'].find do |service|
      service['name'] =~ /app(\-)?dynamics/
    end
    credentials = service['credentials'] if service
  end

  if credentials
    f.puts "export APPDYNAMICS_CONTROLLER_HOST_NAME=#{credentials['host-name']}" if credentials['host-name']
    f.puts "export APPDYNAMICS_CONTROLLER_PORT=#{credentials['port']}" if credentials['port']
    f.puts "export APPDYNAMICS_AGENT_ACCOUNT_NAME=#{credentials['account-name']}" if credentials['account-name']
    f.puts "export APPDYNAMICS_CONTROLLER_SSL_ENABLED=#{credentials['ssl-enabled']}" if credentials['ssl-enabled']
    f.puts "export APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY=#{credentials['account-access-key']}" if credentials['account-access-key']
  
    vcap = JSON.load(ENV['VCAP_APPLICATION']) rescue {}
    if vcap['application_name']
      if ENV["APPDYNAMICS_AGENT_APPLICATION_NAME"].nil?
        f.puts "export APPDYNAMICS_AGENT_APPLICATION_NAME=#{vcap['application_name']}"
      end  
      if ENV["APPDYNAMICS_AGENT_TIER_NAME"].nil?
        f.puts "export APPDYNAMICS_AGENT_TIER_NAME=#{vcap['application_name']}"
      end
      if ENV["APPDYNAMICS_AGENT_NODE_NAME"].nil?
        f.puts "export APPDYNAMICS_AGENT_NODE_NAME=#{vcap['application_name']}:\$CF_INSTANCE_INDEX"
      end
    end
  end
end