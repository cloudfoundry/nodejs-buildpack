$LOAD_PATH.push "#{File.dirname(__FILE__)}/../vendor/semantic_range-1.0.0/lib"

require 'json'
require 'semantic_range'

requested_range = ARGV[0]
manifest_versions_json = ARGV[1]
default_version = ARGV[2]

manifest_versions = JSON.parse(manifest_versions_json)

if requested_range == ""
  puts default_version
else
  sorted_manifest = manifest_versions.sort do |v1, v2|
    if SemanticRange.lt(v1,v2)
      -1
    elsif SemanticRange.gt(v1,v2)
      1
    else
      0
    end
  end.reverse

  resolved_version = sorted_manifest.select do |ver|
    SemanticRange.satisfies(ver, requested_range)
  end.first

  puts resolved_version
end

