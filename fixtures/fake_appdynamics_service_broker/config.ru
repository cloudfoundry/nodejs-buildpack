require 'roda'
require 'erb'

class App < Roda
  plugin :default_headers, 'Content-Type'=>'application/json'

  route do |r|
    r.on 'v2/catalog' do
      ERB.new(open('catalog.json.erb').read).result
    end

    r.on 'v2/service_instances', String do |name|
      r.on 'service_bindings', String  do |sbid|
        '{"credentials": {"host-name":"test-sb-host","port":"1234","account-name":"test-account","ssl-enabled":"true","account-access-key":"test-key"}}'
      end

      r.on do
        '{"dashboard_url": "http://example.com"}'
      end
    end

  end
end

run App.freeze.app
