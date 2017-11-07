#!/usr/bin/env bash
set -euo pipefail

export ROOT=$(dirname $(readlink -f ${BASH_SOURCE%/*}))
if [ ! -f "$ROOT/.bin/ginkgo" ]; then
  echo "Installing ginkgo"
  (cd "$ROOT/src/nodejs/vendor/github.com/onsi/ginkgo/ginkgo/" && go install)
fi

GINKGO_NODES=${GINKGO_NODES:-3}
GINKGO_ATTEMPTS=${GINKGO_ATTEMPTS:-2}

cd $ROOT/src/nodejs/brats
ginkgo -r --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES
