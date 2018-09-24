#!/bin/bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source .envrc

if [ ! -f .bin/ginkgo ]; then
  (cd src/*/vendor/github.com/onsi/ginkgo/ginkgo/ && go install)
fi
if [ ! -f .bin/buildpack-packager ]; then
  (cd src/*/vendor/github.com/cloudfoundry/libbuildpack/packager/buildpack-packager && go install)
fi
if [ ! -f .bin/pack ]; then
  host=$([ $(uname -s) == 'Darwin' ] &&  printf "macos" || printf "linux")
  version=$(curl --silent "https://api.github.com/repos/buildpack/pack/releases/latest" | jq -r .tag_name)
  wget --quiet "https://github.com/buildpack/pack/releases/download/$version/pack-$host" -O .bin/pack && chmod +x .bin/pack
fi
