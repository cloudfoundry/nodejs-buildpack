#!/usr/bin/env bash

set -e
set -u
set -o pipefail

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOTDIR

function main() {
  local src
  src="$(find "${ROOTDIR}/src" -mindepth 1 -maxdepth 1 -type d )"

  for name in supply finalize; do
    GOOS=linux \
      go build \
        -mod vendor \
        -ldflags="-s -w" \
        -o "${ROOTDIR}/bin/${name}" \
          "${src}/${name}/cli"
  done
}

main "${@:-}"
