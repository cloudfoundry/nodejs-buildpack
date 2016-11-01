# coding: utf-8
lib = File.expand_path('../lib', __FILE__)
$LOAD_PATH.unshift(lib) unless $LOAD_PATH.include?(lib)
require 'semantic_range/version'

Gem::Specification.new do |spec|
  spec.name          = "semantic_range"
  spec.version       = SemanticRange::VERSION
  spec.authors       = ["Andrew Nesbitt"]
  spec.email         = ["andrewnez@gmail.com"]

  spec.summary       = %q{node-semver rewritten in ruby, for comparison and inclusion of semantic versions and rangee}
  spec.homepage      = "https://libraries.io/github/librariesio/semantic_range"
  spec.license       = "MIT"

  spec.files         = `git ls-files -z`.split("\x0").reject { |f| f.match(%r{^(test|spec|features)/}) }
  spec.bindir        = "exe"
  spec.executables   = spec.files.grep(%r{^exe/}) { |f| File.basename(f) }
  spec.require_paths = ["lib"]

  spec.add_development_dependency "bundler", "~> 1.10"
  spec.add_development_dependency "rake", "~> 10.4"
  spec.add_development_dependency "rspec", "~> 3.4"
end
