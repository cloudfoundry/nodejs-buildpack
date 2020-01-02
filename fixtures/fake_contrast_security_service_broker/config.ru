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
        '{"credentials": {
              "api_key": "sample_api_key",
              "org_uuid": "sample_org_uuid",
              "service_key": "sample_service_key",
              "teamserver_url": "sample_teamserver_url",
              "username": "sample_username"
             }
         }'
      end

      r.on do
        '{"dashboard_url": "http://example.com"}'
      end
    end

  end
end

run App.freeze.app
