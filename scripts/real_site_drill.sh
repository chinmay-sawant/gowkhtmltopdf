#!/usr/bin/env bash
# Real-site conversion drill with before/after evidence.
#
# Built for the v0.2.7 LearnCpp drill (plans/0.2.7/learncpp, Phase 8) and
# reusable for future real-site drills. It wraps one conversion plus the
# per-page forensics in scripts/pdf_page_forensics.py so every run collects
# the same numbers instead of ad-hoc measurements.
#
# What one run does:
#   1. appends a run header (date, command, binary, baseline) to <log>
#   2. runs <binary> --url <url> -o <output.pdf> with stdout+stderr appended
#      to <log>, and records the exit code and elapsed seconds
#   3. when a non-empty, freshly written PDF exists, runs
#      pdf_page_forensics.py against it, adding --compare <baseline.json>
#      when --baseline was given; the full forensics output goes to <log>
#   4. prints a compact final summary (forensics table lines are left out)
#
# Failure is a normal outcome, not a crash: a failed fetch logs the converter
# error text and its exit code, prints a short summary, and exits nonzero
# without a stack trace. Forensics is skipped when the run left no new
# non-empty PDF, so a stale file from an earlier run is never measured as if
# it were new. The log is opened in append mode, so re-running keeps prior
# runs for comparison.
#
# Local HTML inputs (a path that exists, or a file:// URL) get
# --allow-local-files added; the engine denies local reads by default.
# Network URLs get the exact command, no extra flags.
#
# Usage:
#   bash scripts/real_site_drill.sh <url> <output.pdf> <log> \
#       [--baseline <report.json>] [--binary <path>]
#   bash scripts/real_site_drill.sh --help
#
# Examples:
#   bash scripts/real_site_drill.sh https://www.learncpp.com/ learncpp.pdf \
#       learncpp_phase8.log \
#       --baseline plans/0.2.7/learncpp/evidence/2026-09-16-baseline-noimages.json
#   bash scripts/real_site_drill.sh testdata/golden/fixture-01-simple-invoice.html \
#       /tmp/opencode/drill/invoice.pdf /tmp/opencode/drill/invoice.log \
#       --binary bin/gowkhtmltopdf
#
# The default binary is <repo>/bin/gowkhtmltopdf, built with `make build`.
set -euo pipefail

usage() {
	cat <<'EOF'
Usage:
  real_site_drill.sh <url> <output.pdf> <log> [--baseline <report.json>] [--binary <path>]

Converts <url> with gowkhtmltopdf, appends date, command, converter output,
exit code, and elapsed seconds to <log>, then runs
scripts/pdf_page_forensics.py against the produced PDF (with --compare when
--baseline is given). Prints a compact final summary.

Arguments:
  <url>              page URL, local HTML path, or file:// URL
  <output.pdf>       PDF path to write (overwritten by the converter)
  <log>              log file, opened in append mode

Options:
  --baseline <json>  previous forensics report JSON; adds the delta summary
  --binary <path>    converter binary (default: <repo>/bin/gowkhtmltopdf)
  -h, --help         show this help
EOF
}

die() {
	echo "real_site_drill: $*" >&2
	exit 2
}

file_mtime() {
	stat -c %Y -- "$1" 2>/dev/null || stat -f %m -- "$1" 2>/dev/null || echo ''
}

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
forensics="$script_dir/pdf_page_forensics.py"

url=''
output=''
log=''
baseline=''
binary="$repo_root/bin/gowkhtmltopdf"

while [ $# -gt 0 ]; do
	case $1 in
		-h | --help)
			usage
			exit 0
			;;
		--baseline)
			[ $# -ge 2 ] || die "--baseline needs a path"
			baseline=$2
			shift 2
			;;
		--binary)
			[ $# -ge 2 ] || die "--binary needs a path"
			binary=$2
			shift 2
			;;
		--*)
			die "unknown option: $1 (see --help)"
			;;
		*)
			if [ -z "$url" ]; then
				url=$1
			elif [ -z "$output" ]; then
				output=$1
			elif [ -z "$log" ]; then
				log=$1
			else
				die "unexpected extra argument: $1 (see --help)"
			fi
			shift
			;;
	esac
done

if [ -z "$url" ] || [ -z "$output" ] || [ -z "$log" ]; then
	usage >&2
	exit 2
fi

[ -x "$binary" ] || die "binary not executable: $binary"
[ -f "$forensics" ] || die "missing forensics script: $forensics"
command -v python3 >/dev/null 2>&1 || die "python3 not on PATH"
if [ ! -d "$(dirname -- "$log")" ]; then
	die "log directory does not exist: $(dirname -- "$log")"
fi
if [ -n "$baseline" ] && [ ! -f "$baseline" ]; then
	die "baseline not found: $baseline"
fi

extra=()
case $url in
	file://*) extra=(--allow-local-files) ;;
	*://*) ;;
	*)
		if [ -e "$url" ]; then
			extra=(--allow-local-files)
		fi
		;;
esac

cmd=("$binary" "${extra[@]}" --url "$url" -o "$output")

output_mtime_before=''
if [ -e "$output" ]; then
	output_mtime_before=$(file_mtime "$output")
fi

{
	echo "===== real-site drill $(date '+%Y-%m-%dT%H:%M:%S%z') ====="
	echo "url: $url"
	echo "output: $output"
	echo "binary: $binary"
	echo "baseline: ${baseline:-(none)}"
	echo "cmd: ${cmd[*]}"
} >>"$log"

SECONDS=0
if "${cmd[@]}" >>"$log" 2>&1; then
	convert_rc=0
else
	convert_rc=$?
fi
elapsed=$SECONDS

{
	echo "exit: $convert_rc"
	echo "elapsed: ${elapsed}s"
} >>"$log"

if [ "$convert_rc" -eq 0 ] && [ ! -s "$output" ]; then
	echo "real_site_drill: converter reported success but $output is missing or empty" >>"$log"
	convert_rc=1
fi

analyze=0
if [ -s "$output" ]; then
	if [ "$convert_rc" -eq 0 ]; then
		analyze=1
	elif [ "$output_mtime_before" != "$(file_mtime "$output")" ]; then
		analyze=1
	fi
fi

forensics_rc=0
forensics_out=''
if [ "$analyze" -eq 1 ]; then
	fargs=("$output")
	if [ -n "$baseline" ]; then
		fargs+=(--compare "$baseline")
	fi
	if forensics_out=$(python3 "$forensics" "${fargs[@]}" 2>&1); then
		forensics_rc=0
	else
		forensics_rc=$?
	fi
	printf '%s\n' "$forensics_out" >>"$log"
elif [ -s "$output" ]; then
	echo "forensics: skipped; conversion failed and $output was not rewritten (stale from an earlier run)" >>"$log"
else
	echo "forensics: skipped; no non-empty PDF at $output" >>"$log"
fi

rc=$convert_rc
if [ "$rc" -eq 0 ] && [ "$forensics_rc" -ne 0 ]; then
	rc=$forensics_rc
fi

if [ -n "$forensics_out" ]; then
	printf '%s\n' "$forensics_out" | sed '/^|/d'
fi
bytes=0
if [ -s "$output" ]; then
	bytes=$(wc -c <"$output")
fi
if [ "$convert_rc" -ne 0 ]; then
	echo "real_site_drill: conversion failed (exit $convert_rc); error text is in $log"
fi
echo "real_site_drill: pdf $output (${bytes} bytes)"
echo "real_site_drill: log $log"
echo "real_site_drill: conversion exit $convert_rc, forensics exit $forensics_rc, elapsed ${elapsed}s"
echo "real_site_drill: result exit $rc"

exit "$rc"
