#!/bin/sh
# Builds the c-shared Go library into the Python source tree for cibuildwheel
# runs OUTSIDE linux containers (macos and windows hosts). The linux wheels
# keep using [tool.cibuildwheel.linux] before-build in
# bindings/python/pyproject.toml (uses {project} as the repo root and
# downloads Go inside the manylinux image). This script covers hosts where
# that container path does not run.
set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)
OUT_DIR="$ROOT/bindings/python/src/gowkhtmltopdf"

test -f "$ROOT/VERSION"
test -f "$ROOT/go.mod"
test -d "$ROOT/bindings/c"

case "$(uname -s)" in
	Darwin) EXT=.dylib ;;
	MINGW*|MSYS*|CYGWIN*) EXT=.dll ;;
	*) EXT=.so ;;
esac

mkdir -p "$OUT_DIR"
cd "$ROOT"
CGO_ENABLED=1 go build -buildmode=c-shared \
	-ldflags "-X main.libVersion=$(cat VERSION) -s -w" \
	-o "$OUT_DIR/libgowkhtmltopdf$EXT" ./bindings/c

echo "built $OUT_DIR/libgowkhtmltopdf$EXT"
