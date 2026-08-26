#!/bin/sh
# Version alignment gate (Phase 46): fails unless the repo VERSION file and
# bindings/python/pyproject.toml carry the same release number. Wired as
# `make check-versions` and run in the publish-pypi check job before
# `twine check`, so a tag whose files disagree never reaches PyPI.
set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)

ver=$(tr -d '[:space:]' < "$ROOT/VERSION")
py=$(sed -n 's/^version = "\(.*\)"$/\1/p' "$ROOT/bindings/python/pyproject.toml" \
	| head -n1 | tr -d '[:space:]')

if [ "$ver" != "$py" ]; then
	echo "version mismatch: VERSION=${ver} bindings/python/pyproject.toml=${py}" >&2
	exit 1
fi
echo "versions aligned: ${ver}"
