#!/usr/bin/env bash

set -euo pipefail

GOCACHE="$PWD/go-build"

cd libbuildpack
go test
