require 'zip'
require 'tmpdir'
require 'json'

module BuildpackPackager
  EXCLUDE_FROM_BUILDPACK = [
      /\.git/,
      /\.gitignore/,
      /\.{1,2}$/
  ]

  class << self
    def package(mode)
      Dir.mktmpdir do |temp_dir|
        copy_buildpack_contents(temp_dir)
        download_dependencies(temp_dir) if mode == :offline
        compress_buildpack(temp_dir)
      end
    end

    private

    def copy_buildpack_contents(target_path)
      run_cmd "cp -r * #{target_path}"
    end

    def download_dependencies(target_path)
      dependency_path = File.join(target_path, 'dependencies')

      run_cmd "mkdir -p #{dependency_path}"

      run_cmd "mkdir -p #{target_path}/tmp; curl https://semver.io/node.json > #{target_path}/tmp/versions.json"

      dependencies(target_path).each do |version|
        run_cmd "cd #{dependency_path}; curl http://nodejs.org/dist/v#{version}/node-v#{version}-linux-x64.tar.gz -O"
      end
    end

    def dependencies(target_path)
      JSON.parse(File.read("#{target_path}/tmp/versions.json"))["versions"]
    end

    def in_pack?(file)
      !EXCLUDE_FROM_BUILDPACK.any? { |re| file =~ re }
    end

    def compress_buildpack(target_path)
      Zip::File.open('nodejs_buildpack.zip', Zip::File::CREATE) do |zipfile|
        Dir.glob(File.join(target_path, "**", "**"), File::FNM_DOTMATCH).each do |file|
          zipfile.add(file.sub(target_path + '/', ''), file) if (in_pack?(file))
        end
      end
    end

    def run_cmd(cmd)
      puts "$ #{cmd}"
      `#{cmd}`
    end
  end
end

BuildpackPackager.package(ARGV.first.to_sym)
