$: << 'cf_spec'
require 'spec_helper'

describe 'deploying a nodejs app' do
  it "makes the homepage available" do
    Machete.deploy_app("node_web_app") do |app|
      expect(app).to be_staged
      expect(app.homepage_html).to include "Hello, World!"
    end
  end

  it "deploys apps without vendored dependencies", if: Machete::BuildpackMode.online? do
    app_name = "node_web_app_no_dependencies"

    expect(Dir.exists?("cf_spec/fixtures/#{app_name}/node_modules")).to eql false

    Machete.deploy_app(app_name) do |app|
      expect(app).to be_staged
      expect(app.homepage_html).to include "Hello, World!"
    end
  end
end
