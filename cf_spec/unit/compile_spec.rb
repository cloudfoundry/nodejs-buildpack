$: << 'cf_spec'
require 'spec_helper'

describe 'Compile step of the buildpack' do
  context 'on an unsupported stack' do
    it 'displays a helpful error message' do
      app_dir = Dir.mktmpdir
      cache_dir = Dir.mktmpdir

      output = `env CF_STACK='unsupported' ./bin/compile #{app_dir} #{cache_dir} 2>&1`
      expect(output).to include 'not supported by this buildpack'
    end
  end
end
