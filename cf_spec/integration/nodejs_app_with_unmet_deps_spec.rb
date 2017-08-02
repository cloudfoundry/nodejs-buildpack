$: << 'cf_spec'
require 'spec_helper'

describe 'Node.js applications with unmet dependencies' do
  subject(:app) { Machete.deploy_app(app_name) }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  context 'package manager is npm' do
    let(:app_name) { 'unmet_dep_npm' }

    it 'warns that unmet dependencies may cause issues' do
      expect(app).to be_running
      expect(app).to have_logged('Unmet dependencies don\'t fail npm install but may cause runtime issues')
    end
  end

  context 'package manager is yarn' do
    let(:app_name) { 'unmet_dep_yarn' }

    it 'warns that unmet dependencies may cause issues' do
      expect(app).to be_running
      expect(app).to have_logged('Unmet dependencies don\'t fail yarn install but may cause runtime issues')
    end
  end
end
