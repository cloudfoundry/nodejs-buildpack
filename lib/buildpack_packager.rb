require 'base_packager'
require 'json'

class BuildpackPackager < BasePackager
  def dependencies
    run_cmd "mkdir -p #{target_path}/files"
    run_cmd "curl https://semver.io/node.json > #{target_path}/files/versions.json"

    versions = JSON.parse(File.read("#{target_path}/files/versions.json"))["versions"]

    versions.map do |version|
      "http://nodejs.org/dist/v#{version}/node-v#{version}-linux-x64.tar.gz"
    end
  end

  def excluded_files
    []
  end
end

BuildpackPackager.new("nodejs", ARGV.first.to_sym).package
