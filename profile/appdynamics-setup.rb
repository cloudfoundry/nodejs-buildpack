#!/usr/bin/env ruby
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
  if ENV["APPD_AGENT"].nil? || ENV["APPD_AGENT"]!= "nodejs"
    f.puts " >&2 echo AppDynamics integration is now only suppoted via extension buildpack. Please refer https://docs.pivotal.io/partners/appdynamics/multibuildpack.html"
  end
end
