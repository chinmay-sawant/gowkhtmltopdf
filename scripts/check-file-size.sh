#!/usr/bin/env bash
# File-size soft-limit gate (AGENTS.md "Code structure": ~2,000 lines).
#
# Scans repo .go files, test files included, and skips tool/artifact dirs.
# Every scanned file over the limit must be recorded in
# scripts/file-size-allowlist.txt with its exact line count. The gate fails
# when an over-limit file is unlisted, when a listed file is missing, no
# longer over the limit, or when its recorded count is stale.
#
# Usage:
#   bash scripts/check-file-size.sh
#   FILE_SIZE_LIMIT=1500 bash scripts/check-file-size.sh
#
# Wired as `make size-check`, which `make lint` calls.
set -euo pipefail

# macOS ships bash 3.2; this script needs bash 4+ for associative arrays and
# mapfile. Fail early with a clear message instead of a cryptic syntax error.
if [ "${BASH_VERSINFO[0]:-0}" -lt 4 ]; then
	echo "size-check: bash 4+ required; macOS default bash 3.2 is too old" >&2
	exit 1
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
cd "$repo_root"

limit=${FILE_SIZE_LIMIT:-2000}
allowlist=scripts/file-size-allowlist.txt

case $limit in
	'' | *[!0-9]*)
		echo "size-check: FILE_SIZE_LIMIT must be a positive integer, got '$limit'" >&2
		exit 1
		;;
esac

if [ ! -f "$allowlist" ]; then
	echo "size-check: missing $allowlist" >&2
	exit 1
fi

# Tool and artifact dirs excluded from the scan, as paths relative to the
# repo root. Add to this list only for dirs that never hold hand-written Go.
prune_dirs=(
	./.git
	./verapdf
	./compliance
	./.pdf-validators
	./pen
	./temps
	./bin
	./dist
	./docs
	./frontend
	./scripts/puppeteer
	./bindings/python
)

find_expr=(-path "${prune_dirs[0]}")
for dir in "${prune_dirs[@]:1}"; do
	find_expr+=(-o -path "$dir")
done

# Collect scannable .go files, normalized to repo-relative paths.
mapfile -t scanned_files < <(find . \( "${find_expr[@]}" \) -prune -o -type f -name '*.go' -print | LC_ALL=C sort)

declare -A scanned=()
declare -A actual_lines=()
actual_paths=()
for file in "${scanned_files[@]}"; do
	file=${file#./}
	scanned["$file"]=1
	lines=$(wc -l < "$file")
	if [ "$lines" -gt "$limit" ]; then
		actual_paths+=("$file")
		actual_lines["$file"]=$lines
		echo "over limit: $file ($lines lines, limit $limit)"
	fi
done

problems=()

# Parse the allowlist: one "path<TAB>lines" entry per line, # comments and
# blank lines skipped. Entries stay in file order for stable messages.
allowed_paths=()
declare -A allowed_lines=()
lineno=0
while IFS= read -r line || [ -n "$line" ]; do
	lineno=$((lineno + 1))
	line=${line%$'\r'}
	case $line in
		'' | '#'*) continue ;;
	esac
	path=${line%%$'\t'*}
	recorded=${line#*$'\t'}
	if [ "$recorded" = "$line" ]; then
		problems+=("$allowlist:$lineno: expected 'path<TAB>lines', got: $line")
		continue
	fi
	path=${path#./}
	if [ -z "$path" ]; then
		problems+=("$allowlist:$lineno: empty path before the tab")
		continue
	fi
	case $recorded in
		'' | *[!0-9]*)
			problems+=("$allowlist:$lineno: line count must be an integer, got '$recorded'")
			continue
			;;
	esac
	if [ -n "${allowed_lines[$path]+set}" ]; then
		problems+=("$allowlist:$lineno: duplicate entry for $path")
		continue
	fi
	allowed_paths+=("$path")
	allowed_lines["$path"]=$recorded
done < "$allowlist"

# 1. Every scanned over-limit file must be allowlisted with its exact count.
for path in "${actual_paths[@]}"; do
	lines=${actual_lines[$path]}
	if [ -z "${allowed_lines[$path]+set}" ]; then
		problems+=("$path is $lines lines (limit $limit) and is not in $allowlist; split the file or add an allowlist entry")
	elif [ "${allowed_lines[$path]}" -ne "$lines" ]; then
		problems+=("$path is $lines lines, recorded as ${allowed_lines[$path]}; update the recorded count if the change was deliberate")
	fi
done

# 2. Every allowlisted path must exist, be scanned, still exceed the limit,
# and agree with its recorded count.
for path in "${allowed_paths[@]}"; do
	recorded=${allowed_lines[$path]}
	if [ ! -e "$path" ]; then
		problems+=("$path is allowlisted ($recorded lines) but does not exist; remove the stale entry")
		continue
	fi
	if [ -z "${scanned[$path]+set}" ]; then
		problems+=("$path is allowlisted but was not scanned (not a .go file or under a pruned directory); remove the entry")
		continue
	fi
	if [ -z "${actual_lines[$path]+set}" ]; then
		actual=$(wc -l < "$path")
		problems+=("$path is $actual lines, no longer over the $limit limit (recorded $recorded); remove its allowlist entry")
	fi
done

if [ ${#problems[@]} -gt 0 ]; then
	for problem in "${problems[@]}"; do
		echo "size-check: $problem" >&2
	done
	exit 1
fi

case ${#allowed_paths[@]} in
	0) echo "size-check: clean (0 over-limit files)." ;;
	1) echo "size-check: clean (1 allowlisted over-limit file)." ;;
	*) echo "size-check: clean (${#allowed_paths[@]} allowlisted over-limit files)." ;;
esac
