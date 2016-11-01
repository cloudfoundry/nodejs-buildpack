$: << 'cf_spec'
require 'bundler/setup'
require 'machete'
require 'json'
require 'fileutils'

describe "Node version resolver" do

  # https://github.com/isaacs/node-semver

  def resolve_version(version = "")
    default_version = '4.1.1'
    versions_arr_as_json = ["0.0.1", "0.9.1", "0.10.12", "0.10.13", "0.10.14", "0.11.0"].to_json
    `ruby lib/version_resolver.rb "#{version}" "#{versions_arr_as_json.inspect}" #{default_version}`.strip
  end

  describe 'supporting ranges' do
    it 'resolves default version if no version given' do
      expect(resolve_version).to eql("4.1.1")
    end

    it 'resolves common variants' do
      expect(resolve_version('0.10.13')).to eql '0.10.13'
      expect(resolve_version('0.10.13+build2012')).to eql '0.10.13'
      expect(resolve_version('>0.10.13')).to eql '0.11.0'
      expect(resolve_version('<0.10.13')).to eql '0.10.12'
      expect(resolve_version('>=0.10.14')).to eql '0.11.0'
      expect(resolve_version('>=0.10.15')).to eql '0.11.0'
      expect(resolve_version('<=0.10.14')).to eql '0.10.14'
      expect(resolve_version('<=0.10.15')).to eql '0.10.14'
      expect(resolve_version('~0.9.0')).to eql '0.9.1'
      expect(resolve_version('^0.9')).to eql '0.9.1'
      expect(resolve_version('^0.0.1')).to eql '0.0.1'
      expect(resolve_version('0.10.x')).to eql '0.10.14'
      expect(resolve_version('0.x')).to eql '0.11.0'
      expect(resolve_version('x')).to eql '0.11.0'
      expect(resolve_version('*')).to eql '0.11.0'
    end

    specify "when there's a stable version in the range" do
      expect(resolve_version('0.10.11 - 0.10.14')).to eql '0.10.14'
    end

    specify "when there isn't a stable version in the range" do
      expect(resolve_version('0.10.30 - 0.13.0')).to eql '0.11.0'
    end

  end
end
