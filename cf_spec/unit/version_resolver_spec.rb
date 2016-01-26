$: << 'cf_spec'
require 'bundler/setup'
require 'machete'
require 'json'
require 'fileutils'

describe "Node version resolver" do

  # https://github.com/isaacs/node-semver

  def resolve_version(version = "")
    if `uname`.include?("Darwin")
      node_executable = "/usr/local/bin/node"
    else
      node_executable = "./bin/node"
    end

    stable_version = '0.10.27'
    versions_arr_as_json = ["0.0.1", "0.9.1", "0.10.12", "0.10.13", "0.10.14", "0.11.0"].to_json
    `#{node_executable} lib/version_resolver.js "#{version}" "#{versions_arr_as_json.inspect}" #{stable_version}`.strip
  end

  describe 'supporting ranges' do
    it 'resolves no version' do
      expect(resolve_version).to eql('0.10.27')
    end

    it 'resolves common variants' do
      expect(resolve_version('0.10.13')).to eql '0.10.13'
      expect(resolve_version('0.10.13+build2012')).to eql '0.10.13'
      expect(resolve_version('>0.10.13')).to eql '0.10.14'
      expect(resolve_version('<0.10.13')).to eql '0.10.12'
      expect(resolve_version('>=0.10.14')).to eql '0.10.14'
      expect(resolve_version('>=0.10.15')).to eql '0.11.0'
      expect(resolve_version('<=0.10.14')).to eql '0.10.14'
      expect(resolve_version('<=0.10.15')).to eql '0.10.14'
      expect(resolve_version('~0.9.0')).to eql '0.9.1'
      expect(resolve_version('^0.9')).to eql '0.9.1'
      expect(resolve_version('^0.0.1')).to eql '0.0.1'
      expect(resolve_version('0.10.x')).to eql '0.10.14'
      expect(resolve_version('0.x')).to eql '0.10.14'
      expect(resolve_version('x')).to eql '0.10.14'
      expect(resolve_version('*')).to eql '0.10.14'
    end

    specify "when there's a stable version in the range" do
      expect(resolve_version('0.10.11 - 0.10.14')).to eql '0.10.14'
    end

    specify "when there isn't a stable version in the range" do
      expect(resolve_version('0.10.30 - 0.13.0')).to eql '0.11.0'
    end

  end
end
