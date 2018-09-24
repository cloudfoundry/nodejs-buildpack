#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source .envrc
./scripts/install_tools.sh

GINKGO_NODES=${GINKGO_NODES:-1}
GINKGO_ATTEMPTS=${GINKGO_ATTEMPTS:-1}

export CNB_BUILD_IMAGE=${CNB_BUILD_IMAGE:-cfbuildpacks/cflinuxfs3-cnb-experimental:build}

cd src/*/v3/integration

echo "Run V3 Buildpack Runtime Integration Tests"
ginkgo -r --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES
