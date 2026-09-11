#!/usr/bin/env bash
# bench-performance-recovery.sh - controlled-host capture for the 0.2.6
# performance recovery plan (PERF-01; the harness for VALID-02 / VALID-03).
# The mode list is closed on purpose, so generic product measurements and
# internal-only benchmark paths never share a result file.
#
# Usage: scripts/bench-performance-recovery.sh --mode=<mode> [options]
#
# Modes:
#   internal-pdf   generic in-process PDF via BenchmarkPDFPages/generic
#   public-pdf     public Document.WritePDF via BenchmarkLibraryPDF
#   public-image   public ImageDocument.WriteImage via BenchmarkLibraryImage
#   cli-rss        generic CLI process time and peak RSS via /usr/bin/time
#   external       scripts/bench-external.sh, then make bench-cli-compare
#
# Options:
#   --mode=MODE       required; one of the modes above
#   --sizes=LIST      comma-separated page counts, or tile counts for
#                     public-image. Defaults: internal-pdf/public-pdf 2,500;
#                     public-image 250,500; cli-rss 2,100,500;
#                     external 2,10,50,100
#   --benchtime=VALUE go test -benchtime for in-process modes (default 1x)
#   --count=N         go test -count for in-process modes (default 1)
#   --runs=N          timed CLI runs after one warmup (default 3)
#   --out=DIR         result directory (default
#                     plans/0.2.6/perf-review/results/<today>)
#   --dry-run         print every command and file access, then exit 0
#   -h | --help       this text
#
# Each real run writes one result file whose header records date, mode, sizes,
# Go version, CPU, memory, OS/kernel, cache state, fixture hash, source
# hashes, and (CLI modes) the binary path plus hash. Process RSS from
# /usr/bin/time %M stays separate from Go B/op; CLI mode prints process rows.
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

TEMPLATE="$ROOT/testdata/golden/benchmarks/templates/report.html.tmpl"
GOWK_BIN="$ROOT/bin/gowkhtmltopdf"

MODE= SIZES= BENCHTIME=1x COUNT=1 RUNS=3 DRY_RUN=0 OUT_DIR= WORK_DIR=

usage() {
  sed -n '2,/^set -euo/p' "$0" | sed '/^set /d; s/^# \{0,1\}//'
}

die_usage() {
  echo "bench-performance-recovery: $1" >&2
  usage >&2
  exit 2
}

validate_positive_int() {
  local label=$1 value=$2
  [[ "$value" =~ ^[1-9][0-9]*$ ]] || die_usage "$label must be a positive integer: $value"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --dry-run) DRY_RUN=1 ;;
    -h | --help)
      usage
      exit 0
      ;;
    --*=*)
      key=${1%%=*}
      value=${1#*=}
      case "$key" in
        --mode) MODE=$value ;;
        --sizes) SIZES=$value ;;
        --benchtime) BENCHTIME=$value ;;
        --count) COUNT=$value ;;
        --runs) RUNS=$value ;;
        --out) OUT_DIR=$value ;;
        *) die_usage "unknown argument: $key" ;;
      esac
      ;;
    --mode | --sizes | --benchtime | --count | --runs | --out)
      die_usage "$1 requires --flag=value syntax"
      ;;
    *) die_usage "unknown argument: $1" ;;
  esac
  shift
done

[ -n "$MODE" ] || die_usage "--mode is required"
case "$MODE" in
  internal-pdf | public-pdf | public-image | cli-rss | external) ;;
  *) die_usage "unknown mode: $MODE" ;;
esac

if [ -z "$SIZES" ]; then
  case "$MODE" in
    internal-pdf | public-pdf) SIZES="2,500" ;;
    public-image) SIZES="250,500" ;;
    cli-rss) SIZES="2,100,500" ;;
    external) SIZES="2,10,50,100" ;;
  esac
fi

IFS=',' read -r -a SIZE_LIST <<<"$SIZES"
for size in "${SIZE_LIST[@]}"; do
  validate_positive_int "size" "$size"
done
validate_positive_int "--count" "$COUNT"
validate_positive_int "--runs" "$RUNS"

if [ "$MODE" = "public-image" ]; then SIZE_UNIT=" (tiles)"; else SIZE_UNIT=" (pages)"; fi

if [ -z "$OUT_DIR" ]; then
  OUT_DIR="$ROOT/plans/0.2.6/perf-review/results/$(date +%F)"
fi
case "$OUT_DIR" in /*) ;; *) OUT_DIR="$ROOT/$OUT_DIR" ;; esac

SIZE_SLUG=${SIZES//,/-}
RESULT="$OUT_DIR/perf-recovery-${MODE}-${SIZE_SLUG}.txt"

source_files() {
  printf '%s\n' "scripts/bench-performance-recovery.sh"
  case "$MODE" in
    internal-pdf) printf '%s\n' "internal/convert/benchmarks_test.go" ;;
    public-pdf | public-image) printf '%s\n' "document_bench_test.go" "document_bench_validate_test.go" ;;
    cli-rss) printf '%s\n' "document_bench_test.go" "document_bench_html_test.go" ;;
    external)
      printf '%s\n' "scripts/bench-external.sh" "internal/convert/wk_compare_test.go" "Makefile"
      ;;
  esac
}

sha256_of() { sha256sum "$1" 2>/dev/null | awk '{print $1}'; }

host_cpu() {
  if command -v lscpu >/dev/null 2>&1; then
    lscpu | awk -F: '/^Model name/ {sub(/^[ \t]+/, "", $2); print $2; exit}'
  else
    awk -F: '/^model name/ {sub(/^[ \t]+/, "", $2); print $2; exit}' /proc/cpuinfo
  fi
}

host_memory() { free -h 2>/dev/null | awk '/^Mem:/ {print $0; exit}'; }

host_cache_state() {
  local drop_caches="absent"
  [ -e /proc/sys/vm/drop_caches ] && drop_caches="present"
  printf '%s; /proc/sys/vm/drop_caches %s (never written, needs root)' \
    "$(host_memory)" "$drop_caches"
}

write_header() {
  local source
  {
    echo "# bench-performance-recovery result"
    echo "# date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "# mode: $MODE"
    echo "# sizes: $SIZES$SIZE_UNIT"
    echo "# go: $(go version)"
    echo "# cpu: $(host_cpu) ($(nproc 2>/dev/null || printf '?') CPUs)"
    echo "# memory: $(host_memory)"
    echo "# os: $(uname -srm)"
    echo "# cache: $(host_cache_state)"
    echo "# fixture: $TEMPLATE sha256=$(sha256_of "$TEMPLATE")"
    echo "# sources:"
    while read -r source; do
      echo "#   $ROOT/$source sha256=$(sha256_of "$ROOT/$source")"
    done < <(source_files)
    if [ "$MODE" = "cli-rss" ] || [ "$MODE" = "external" ]; then
      echo "# binary: $GOWK_BIN sha256=$(sha256_of "$GOWK_BIN")"
    else
      echo "# binary: not used (in-process mode)"
    fi
    echo "# runs: warmup + $RUNS timed (cli/external); benchtime=$BENCHTIME count=$COUNT (in-process)"
    echo "# result: $RESULT"
  } >"$RESULT"
}

execute() {
  local -a cmd=("$@")
  local display rc
  display=$(printf '%q ' "${cmd[@]}")
  if [ "$DRY_RUN" -eq 1 ]; then
    printf 'would run: %s\n' "$display"
    return 0
  fi
  printf 'running: %s\n' "$display" | tee -a "$RESULT"
  "${cmd[@]}" 2>&1 | tee -a "$RESULT"
  rc=${PIPESTATUS[0]}
  printf '# exit: %s\n' "$rc" >>"$RESULT"
  return "$rc"
}

sizes_to_alt() { # 2,500 + suffix -> 2Pages|500Pages
  local suffix=$1 out= size first=1
  for size in "${SIZE_LIST[@]}"; do
    if [ "$first" -eq 1 ]; then first=0; else out+='|'; fi
    out+="${size}${suffix}"
  done
  printf '%s' "$out"
}

run_go_bench() { # package bench-prefix size-suffix
  execute go test "$1" -run '^$' \
    -bench "$2($(sizes_to_alt "$3"))\$" \
    -benchmem "-benchtime=$BENCHTIME" "-count=$COUNT"
}

count_pdf_pages() { # same page objects the Go CLI comparison counts
  grep -a -o -E '/Type ?/Page([^s]|$)' "$1" | wc -l | tr -d ' '
}

append_cli_measurement() {
  local label=$1 pages=$2 timefile=$3 pdf=$4 elapsed rss rendered bytes
  read -r elapsed rss <"$timefile"
  rendered=$(count_pdf_pages "$pdf")
  bytes=$(wc -c <"$pdf" | tr -d ' ')
  if [ "$rendered" -ne "$pages" ]; then
    echo "bench-performance-recovery: $pdf rendered $rendered pages, want $pages" >&2
    return 1
  fi
  printf '%s: pages=%s elapsed_s=%s peak_rss_kib=%s pdf_bytes=%s rendered_pages=%s\n' \
    "$label" "$pages" "$elapsed" "$rss" "$bytes" "$rendered" | tee -a "$RESULT"
}

run_cli_rss() {
  local html pdf timefile index label pages
  if [ "$DRY_RUN" -eq 1 ]; then
    WORK_DIR="mktemp-dir"
  else
    WORK_DIR=$(mktemp -d "${TMPDIR:-/tmp}/bench-performance-recovery.XXXXXX")
    trap 'if [ -n "$WORK_DIR" ]; then rm -rf "$WORK_DIR"; fi' EXIT
  fi

  execute env "GOWKHTMLTOPDF_BENCH_HTML_DIR=$WORK_DIR" "GOWKHTMLTOPDF_BENCH_HTML_SIZES=$SIZES" \
    go test . -run '^TestWriteBenchmarkHTML$' -count=1

  for pages in "${SIZE_LIST[@]}"; do
    html="$WORK_DIR/doc_${pages}.html"
    pdf="$WORK_DIR/gowk_${pages}.pdf"
    for index in $(seq 0 "$RUNS"); do
      if [ "$index" -eq 0 ]; then label=warmup; else label="run$index"; fi
      timefile="$WORK_DIR/time_${pages}_${label}.txt"

      execute /usr/bin/time -f '%e %M' -o "$timefile" \
        "$GOWK_BIN" --quiet --allow-local-files -o "$pdf" "$html"

      if [ "$DRY_RUN" -eq 1 ]; then
        printf 'would read and check: %s, %s\n' "$timefile" "$pdf"
      else
        append_cli_measurement "$label" "$pages" "$timefile" "$pdf"
      fi
    done
  done
}

run_external() {
  execute ./scripts/bench-external.sh "--sizes=$SIZES" "--runs=$RUNS"
  execute make bench-cli-compare
}

dispatch_mode() {
  case "$MODE" in
    internal-pdf) run_go_bench ./internal/convert 'BenchmarkPDFPages/generic/' Pages ;;
    public-pdf) run_go_bench . 'BenchmarkLibraryPDF/' Pages ;;
    public-image) run_go_bench . 'BenchmarkLibraryImage/' Tiles ;;
    cli-rss) run_cli_rss ;;
    external) run_external ;;
  esac
}

if [ "$DRY_RUN" -eq 1 ]; then
  echo "dry-run: mode=$MODE sizes=$SIZES$SIZE_UNIT benchtime=$BENCHTIME count=$COUNT runs=$RUNS"
  echo "dry-run: result file would be: $RESULT"
  while read -r source; do
    echo "dry-run: would read: $ROOT/$source"
  done < <(source_files)
  echo "dry-run: would read: $TEMPLATE"
  case "$MODE" in
    cli-rss)
      echo "dry-run: would read: $GOWK_BIN (after make build)"
      echo "dry-run: would write: mktemp-dir/{doc_<size>.html,gowk_<size>.pdf,time_<size>_<run>.txt}"
      ;;
    external)
      echo "dry-run: would write: $GOWK_BIN (make build)"
      echo "dry-run: would write: testdata/golden/benchmarks/{weasyprint,puppeteer}-compare.{md,csv}"
      echo "dry-run: would write: testdata/golden/benchmarks/cli-compare.{md,csv}"
      ;;
    *) echo "dry-run: would write: $RESULT" ;;
  esac
  case "$MODE" in cli-rss | external) execute make build ;; esac
  dispatch_mode
  exit 0
fi

mkdir -p "$OUT_DIR"
if [ "$MODE" = "cli-rss" ] || [ "$MODE" = "external" ]; then
  printf 'building: make build\n'
  make build
fi
write_header
dispatch_mode
echo "wrote $RESULT"
