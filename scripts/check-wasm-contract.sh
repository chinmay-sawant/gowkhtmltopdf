#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
artifact_dir=$(mktemp -d)
trap 'rm -rf "$artifact_dir"' EXIT

cd "$repo_root"

go test ./bindings/wasm
GOOS=js GOARCH=wasm go build -o "$artifact_dir/gowkhtmltopdf.wasm" ./bindings/wasm
test -s "$artifact_dir/gowkhtmltopdf.wasm"

echo "WASM contract tests and JS/WASM build passed."
