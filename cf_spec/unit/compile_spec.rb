$: << 'cf_spec'
require 'spec_helper'

describe 'Compile step of the buildpack' do
  def run(cmd, env: {})
    if RUBY_PLATFORM =~ /darwin/i
      env_flags = env.map{|k,v| "-e #{k}=#{v}"}.join(' ')
      `docker run --rm #{env_flags} -v #{Dir.pwd}:/buildpack:ro -w /buildpack cloudfoundry/cflinuxfs2 #{cmd}`
    else
      `env #{env.map{|k,v| "#{k}=#{v}"}.join(' ')} #{cmd}`
    end
  end

  context 'on an unsupported stack' do
    it 'displays a helpful error message' do
      app_dir = Dir.mktmpdir
      cache_dir = Dir.mktmpdir

      output = run("./bin/compile #{app_dir} #{cache_dir} 2>&1", env:{CF_STACK: 'unsupported'})
      expect(output).to include 'Stack not supported by buildpack'
    end
  end
end
