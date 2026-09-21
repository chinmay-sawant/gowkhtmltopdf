#!/usr/bin/env bash
# Wall-time smoke for `make run` (skills/PR/PR_TEMPLATE.md).
# Converts one HTML fixture through the built CLI and fails if wall time
# is at or above RUN_MAX_MS (default 400). Soft ±50ms of a stored
# reference is host noise on this WSL2 box and is not a fail gate.
set -euo pipefail

html=${1:?html path}
max_ms=${2:?max milliseconds}
bin=${3:?gowkhtmltopdf binary}

if [[ ! -f $html ]]; then
	echo "run-walltime: missing html $html" >&2
	exit 2
fi
if [[ ! -x $bin ]]; then
	echo "run-walltime: missing binary $bin (run make build)" >&2
	exit 2
fi

out=$(mktemp /tmp/gowk-run-XXXXXX.pdf)
trap 'rm -f "$out"' EXIT

start=$(date +%s%N)
"$bin" --quiet --allow-local-files -o "$out" "$html"
end=$(date +%s%N)
ms=$(((end - start) / 1000000))
bytes=$(wc -c <"$out")

echo "make run: ${ms}ms  bytes=${bytes}  html=${html}  hard<${max_ms}ms"
if ((ms >= max_ms)); then
	echo "make run: FAIL wall time ${ms}ms >= ${max_ms}ms" >&2
	exit 1
fi
