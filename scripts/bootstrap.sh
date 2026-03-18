#!/usr/bin/env sh
set -eu

echo "[bootstrap] installing pinned tools"
make init

echo "[bootstrap] generating code"
make generate

echo "[bootstrap] running tests"
make test
