#!/usr/bin/env bash
# bench-external.sh — process-level CLI comparison of bin/gowkhtmltopdf
# against the installed WeasyPrint and Puppeteer engines.
#
# The bench never invokes engine commands itself: each engine is run through
# its print script (scripts/weasyprint/print.sh, scripts/puppeteer/print.sh),
# so the measured command is exactly what that script runs.
#
# Usage:
#   scripts/bench-external.sh [--engines=weasyprint,puppeteer]
#                             [--sizes=2,10,50,100] [--runs=3]
#
# Requires: bin/gowkhtmltopdf (make build), /usr/bin/time, gs.
# Puppeteer additionally requires node and scripts/puppeteer/node_modules
# (npm ci --prefix scripts/puppeteer); WeasyPrint requires the weasyprint CLI.
# Engines that are not installed are skipped.
#
# Methodology (same as bench-cli-compare): for each requested page size the
# report fixture (testdata/golden/benchmarks/templates/report.html.tmpl, 20
# invoice rows per page) is generated, then each engine runs one warmup
# followed by --runs timed runs. Wall time is measured around /usr/bin/time;
# peak RSS is %M, except for Puppeteer which samples the whole process tree
# (node driver + headless Chrome children) from ps snapshots. Ghostscript
# verifies the rendered page counts. The median of the timed runs is written
# to testdata/golden/benchmarks/{weasyprint,puppeteer}-compare.md and
# -compare-results.csv.
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
GOWK_BIN="$ROOT/bin/gowkhtmltopdf"
ARTIFACT_DIR="${BENCH_ARTIFACT_DIR:-$ROOT/testdata/golden/benchmarks}"
TEMPLATE="$ROOT/testdata/golden/benchmarks/templates/report.html.tmpl"
POLL_INTERVAL=0.02
RUN_TIMEOUT=300
ROWS_PER_PAGE=20

sizes=(2 10 50 100)
runs=3
requested_engines=()
engines_specified=0

usage() {
  sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
}

die_usage() {
  echo "bench-external: $1" >&2
  usage >&2
  exit 2
}

validate_positive_int() {
  local label=$1 value=$2
  if [[ ! "$value" =~ ^[1-9][0-9]*$ ]]; then
    die_usage "$label must be a positive integer: $value"
  fi
}

validate_sizes() {
  local size
  if [ "${#sizes[@]}" -eq 0 ]; then
    die_usage "--sizes must contain at least one page count"
  fi
  for size in "${sizes[@]}"; do
    validate_positive_int "page size" "$size"
  done
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --engines)
      [ "$#" -ge 2 ] || die_usage "--engines requires a comma-separated value"
      engines_specified=1
      IFS=',' read -r -a requested_engines <<<"$2"
      shift 2
      ;;
    --engines=*)
      engines_specified=1
      IFS=',' read -r -a requested_engines <<<"${1#*=}"
      shift
      ;;
    --sizes)
      [ "$#" -ge 2 ] || die_usage "--sizes requires a comma-separated value"
      IFS=',' read -r -a sizes <<<"$2"
      shift 2
      ;;
    --sizes=*)
      IFS=',' read -r -a sizes <<<"${1#*=}"
      shift
      ;;
    --runs)
      [ "$#" -ge 2 ] || die_usage "--runs requires a value"
      runs=$2
      shift 2
      ;;
    --runs=*)
      runs=${1#*=}
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      die_usage "unknown argument: $1"
      ;;
  esac
done

validate_sizes
validate_positive_int "--runs" "$runs"

[ -x "$GOWK_BIN" ] || {
  echo "bench-external: $GOWK_BIN not found; run make build" >&2
  exit 1
}

[ -x /usr/bin/time ] || {
  echo "bench-external: /usr/bin/time not found (GNU time package)" >&2
  exit 1
}

TIMEOUT_CMD=$(command -v timeout || true)
[ -n "$TIMEOUT_CMD" ] || {
  echo "bench-external: timeout not found (GNU coreutils package)" >&2
  exit 1
}

[ -r "$TEMPLATE" ] || {
  echo "bench-external: benchmark template not readable: $TEMPLATE" >&2
  exit 1
}

if command -v gs >/dev/null 2>&1; then
  HAVE_GS=1
else
  HAVE_GS=0
  echo "bench-external: gs not found; page counts will report 0" >&2
fi

# ---------------------------------------------------------------------------
# Engine registry
# ---------------------------------------------------------------------------

engine_available() { # name -> 0/1
  local name=$1
  case "$name" in
    weasyprint)
      [ -x "$ROOT/scripts/weasyprint/print.sh" ] && command -v weasyprint >/dev/null 2>&1
      ;;
    puppeteer)
      [ -x "$ROOT/scripts/puppeteer/print.sh" ] &&
        command -v node >/dev/null 2>&1 &&
        [ -f "$ROOT/scripts/puppeteer/node_modules/puppeteer-core/package.json" ] &&
        executable_available "$(puppeteer_executable)"
      ;;
    *)
      return 1
      ;;
  esac
}

puppeteer_executable() {
  printf '%s' "${PUPPETEER_EXECUTABLE_PATH:-/usr/bin/google-chrome}"
}

executable_available() { # path or PATH command -> 0/1
  local executable=$1
  if [[ "$executable" == */* ]]; then
    [ -x "$executable" ]
  else
    command -v "$executable" >/dev/null 2>&1
  fi
}

engine_display() { # name
  case "$1" in
    weasyprint) printf 'WeasyPrint' ;;
    puppeteer) printf 'Puppeteer' ;;
  esac
}

engine_script() { # name
  case "$1" in
    weasyprint) printf '%s' "$ROOT/scripts/weasyprint/print.sh" ;;
    puppeteer) printf '%s' "$ROOT/scripts/puppeteer/print.sh" ;;
  esac
}

engine_version() { # name
  case "$1" in
    weasyprint)
      if command_version=$(weasyprint --version 2>/dev/null); then
        printf '%s' "$command_version" | sed -n '1s/^[[:space:]]*//p'
      else
        printf 'unknown'
      fi
      ;;
    puppeteer)
      local ver
      ver=$(awk -F'"' '/"version"/ { print $4; exit }' \
        "$ROOT/scripts/puppeteer/node_modules/puppeteer-core/package.json")
      local chrome
      chrome=$(puppeteer_executable)
      if chrome_version=$("$chrome" --version 2>/dev/null); then
        printf 'puppeteer-core %s + %s' "$ver" "$(printf '%s' "$chrome_version" | sed 's/[[:space:]]*$//')"
      else
        printf 'puppeteer-core %s' "$ver"
      fi
      ;;
  esac
}

engine_flags_note() { # name
  case "$1" in
    weasyprint)
      printf 'weasyprint used `-q` (quiet)'
      ;;
    puppeteer)
      printf 'Puppeteer printed via headless Chrome (`%s`) with `format: A4`, `printBackground: true`, `preferCSSPageSize: true`' "$(puppeteer_executable)"
      ;;
  esac
}

engine_rss_note() { # name
  case "$1" in
    weasyprint)
      printf 'weasyprint RSS is the peak of the weasyprint CLI process from `%%M`'
      ;;
    puppeteer)
      printf 'Puppeteer RSS is the peak process-tree RSS (node driver + headless Chrome children) sampled from a `ps` snapshot every %g s' "$POLL_INTERVAL"
      ;;
  esac
}

engine_tree_rss() { # name -> 0/1
  case "$1" in
    puppeteer) printf '1' ;;
    *) printf '0' ;;
  esac
}

# ---------------------------------------------------------------------------
# Fixture generation from testdata/golden/benchmarks/templates/report.html.tmpl
# ---------------------------------------------------------------------------

generate_report() { # pages outfile
  local pages=$1 out=$2
  awk -v target_pages="$pages" -v rows_per_page="$ROWS_PER_PAGE" '
    function render_page(page, i, line, in_rows, row, sku, quantity, amount, row_line) {
      in_rows = 0
      for (i = 1; i <= page_line_count; i++) {
        line = page_line[i]
        if (line ~ /\{\{range \.Rows\}\}/) {
          in_rows = 1
          continue
        }
        if (in_rows) {
          if (line ~ /^[[:space:]]*\{\{end\}\}[[:space:]]*$/) {
            in_rows = 0
            continue
          }
          for (row = 1; row <= rows_per_page; row++) {
            row_line = line
            sku = sprintf("SKU-%03d-%03d", page, row)
            quantity = (row + page - 1) % 7 + 1
            amount = sprintf("%d.%02d", page * row, (page - 1 + row) % 100)
            gsub(/\{\{\.Number\}\}/, row, row_line)
            gsub(/\{\{\.SKU\}\}/, sku, row_line)
            gsub(/\{\{\.Description\}\}/, "Platform operations and support service " row, row_line)
            gsub(/\{\{\.Quantity\}\}/, quantity, row_line)
            gsub(/\{\{\.Amount\}\}/, amount, row_line)
            print row_line
          }
          continue
        }
        if (page == 1) {
          gsub(/\{\{if \.First\}\} first\{\{end\}\}/, " first", line)
        } else {
          gsub(/\{\{if \.First\}\} first\{\{end\}\}/, "", line)
        }
        gsub(/\{\{\.Number\}\}/, page, line)
        print line
      }
    }
    {
      if (!in_pages) {
        if ($0 ~ /\{\{range \.Pages\}\}/) {
          in_pages = 1
          page_depth = 1
          next
        }
        print
        next
      }
      if ($0 ~ /\{\{range \.Rows\}\}/) {
        page_depth++
        page_line[++page_line_count] = $0
        next
      }
      if ($0 ~ /^[[:space:]]*\{\{end\}\}[[:space:]]*$/ && page_depth == 1) {
        for (page = 1; page <= target_pages; page++) {
          render_page(page)
        }
        in_pages = 0
        next
      }
      if ($0 ~ /^[[:space:]]*\{\{end\}\}[[:space:]]*$/) {
        page_depth--
      }
      page_line[++page_line_count] = $0
    }
  ' "$TEMPLATE" >"$out"

  if grep -F '{{' "$out" >/dev/null 2>&1; then
    echo "bench-external: template actions remained after rendering: $TEMPLATE" >&2
    return 1
  fi
}

# ---------------------------------------------------------------------------
# Process measurement
# ---------------------------------------------------------------------------

pdf_page_count() { # pdf -> pages (0 when gs is unavailable)
  if [ "$HAVE_GS" -ne 1 ]; then
    printf '0'
    return 0
  fi
  local out
  out=$(gs -q -dNODISPLAY -c "($1) (r) file runpdfbegin pdfpagecount = quit" 2>/dev/null | tr -d ' \n')
  printf '%s' "${out:-0}"
}

# tree_rss <pid> -> KiB summed over pid and all descendants. Uses one ps
# snapshot per sample (procfs `children` files are gone on newer kernels).
tree_rss() {
  ps -e -o pid=,ppid=,rss= | awk -v root="$1" '
    function walk(p,   i) {
      total += rss[p]
      for (i = 0; i < n[p]; i++) walk(child[p, i])
    }
    {
      pid = $1
      ppid = $2
      rss[pid] = $3
      child[ppid, n[ppid]++] = pid
    }
    END {
      total = 0
      walk(root)
      print total + 0
    }'
}

poll_tree_rss() { # pid -> peak KiB of the whole process tree until exit
  local pid=$1 peak=0 current
  while kill -0 "$pid" 2>/dev/null; do
    current=$(tree_rss "$pid")
    if [ "$current" -gt "$peak" ]; then
      peak=$current
    fi
    sleep "$POLL_INTERVAL"
  done
  printf '%d' "$peak"
}

# time_command <tree_rss:0|1> <out_pdf> <name> <cmd...>
# Runs one timed invocation; appends "elapsed rss pages bytes" to $RUN_LOG.
RUN_LOG=
time_command() {
  local tree_rss=$1 out_pdf=$2 name=$3
  shift 3
  local start end timefile rc elapsed rss pages bytes
  timefile=$(mktemp "$tmp/time.XXXXXX")
  start=$(date +%s.%N)
  if [ "$tree_rss" -eq 1 ]; then
    "$TIMEOUT_CMD" "$RUN_TIMEOUT" /usr/bin/time -f '%e %M' -o "$timefile" -- "$@" &
    local pid=$!
    rss=$(poll_tree_rss "$pid")
    set +e
    wait "$pid"
    rc=$?
    set -e
  else
    set +e
    "$TIMEOUT_CMD" "$RUN_TIMEOUT" /usr/bin/time -f '%e %M' -o "$timefile" -- "$@"
    rc=$?
    set -e
  fi
  if [ "$rc" -eq 124 ]; then
    echo "bench-external: $name timed out after ${RUN_TIMEOUT}s" >&2
    rm -f "$timefile"
    return 1
  fi
  if [ "$rc" -ne 0 ]; then
    echo "bench-external: $name failed (exit $rc)" >&2
    rm -f "$timefile"
    return 1
  fi
  if [ ! -s "$out_pdf" ]; then
    echo "bench-external: $name produced no PDF: $out_pdf" >&2
    rm -f "$timefile"
    return 1
  fi
  end=$(date +%s.%N)
  if [ "$tree_rss" -ne 1 ]; then
    rss=$(awk 'NR == 1 { print $2 }' "$timefile")
  fi
  elapsed=$(awk -v a="$start" -v b="$end" 'BEGIN { printf "%.6f", b - a }')
  bytes=$(wc -c <"$out_pdf" | tr -d ' ')
  pages=$(pdf_page_count "$out_pdf")
  printf '%s %s %s %s\n' "$elapsed" "$rss" "$pages" "$bytes" >>"$RUN_LOG"
  rm -f "$timefile"
}

median_value() { # file -> middle value of the sorted lines
  local n
  n=$(wc -l <"$1" | tr -d ' ')
  sort -n "$1" | awk -v n="$n" 'NR == int((n + 1) / 2) { print; exit }'
}

format_ms() { # seconds float
  awk -v s="$1" 'BEGIN { ms = int(s * 1000 + 0.5); if (ms < 1000) printf "%d ms", ms; else printf "%.3f s", ms / 1000 }'
}

comma() { # int -> 1,234,567
  local n=$1 out=
  while [ ${#n} -gt 3 ]; do
    out=",${n: -3}${out}"
    n=${n:0:${#n} - 3}
  done
  printf '%s%s' "$n" "$out"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

engines=()
if [ "$engines_specified" -eq 0 ]; then
  requested_engines=(weasyprint puppeteer)
fi

for engine in "${requested_engines[@]}"; do
  if engine_available "$engine"; then
    engines+=("$engine")
  else
    echo "bench-external: $engine unavailable or unsupported; skipping" >&2
  fi
done

if [ ${#engines[@]} -eq 0 ]; then
  echo "bench-external: no external engine selected (--engines) or installed" >&2
  exit 1
fi

host=$(uname -srm 2>/dev/null || printf 'unknown host')
cpu_count=$(getconf _NPROCESSORS_ONLN 2>/dev/null || printf 'unknown')
toolchain=$(go version 2>/dev/null || printf 'Go toolchain unavailable')
gowk_version=$(
  "$GOWK_BIN" --version 2>/dev/null | awk '/^Version:/ { print $2; exit }'
)
gowk_version=${gowk_version:-unknown}
if [ "$HAVE_GS" -eq 1 ]; then
  page_count_note='Ghostscript `gs` was present; rendered page counts were checked against the requested size.'
else
  page_count_note='Ghostscript `gs` was absent; rendered page counts are reported as 0 and were not validated.'
fi

for engine in "${engines[@]}"; do
  display=$(engine_display "$engine")
  script=$(engine_script "$engine")
  version=$(engine_version "$engine")
  tree_rss_flag=$(engine_tree_rss "$engine")
  rows=()

  echo
  echo "================================================================================================"
  echo "PROCESS CLI COMPARISON: gowkhtmltopdf vs $display"
  echo "gowk:    $GOWK_BIN"
  echo "engine:  $script ($version)"
  echo "flags:   gowkhtmltopdf used \`--quiet --allow-local-files -o OUTPUT INPUT\`; $(engine_flags_note "$engine")."
  echo "rss:     $(engine_rss_note "$engine"); gowkhtmltopdf RSS is \`%M\`."
  echo "runs:    warmup + $runs timed (median)"
  echo "================================================================================================"
  printf '%-8s | %-12s | %-16s | %-8s | %-14s | %-20s | %-8s\n' \
    "Pages" "gowk time" "$display time" "Speedup" "gowk RSS" "$display RSS" "gowk PDF"
  echo "------------------------------------------------------------------------------------------------"

  for pages in "${sizes[@]}"; do
    html="$tmp/doc_${pages}.html"
    generate_report "$pages" "$html" || exit 1
    gowk_out="$tmp/gowk_${pages}.pdf"
    engine_out="$tmp/${engine}_${pages}.pdf"

    gowk_log="$tmp/gowk_${pages}.times"
    engine_log="$tmp/${engine}_${pages}.times"
    : >"$gowk_log"
    : >"$engine_log"

    echo "page size $pages: timing gowk..."
    RUN_LOG="$gowk_log"
    time_command 0 "$gowk_out" "gowkhtmltopdf" "$GOWK_BIN" --quiet --allow-local-files -o "$gowk_out" "$html"
    echo "  warmup: $(format_ms "$(awk 'NR == 1 {print $1}' "$gowk_log")") ($(awk 'NR == 1 {print $3}' "$gowk_log") pages)"
    for ((i = 1; i <= runs; i++)); do
      time_command 0 "$gowk_out" "gowkhtmltopdf" "$GOWK_BIN" --quiet --allow-local-files -o "$gowk_out" "$html"
      echo "  run $i/$runs: $(format_ms "$(awk 'NR == '"$((i + 1))"' {print $1}' "$gowk_log")")"
    done

    echo "page size $pages: timing $display..."
    RUN_LOG="$engine_log"
    time_command "$tree_rss_flag" "$engine_out" "$display" "$script" "$html" "$engine_out"
    echo "  warmup: $(format_ms "$(awk 'NR == 1 {print $1}' "$engine_log")") ($(awk 'NR == 1 {print $3}' "$engine_log") pages)"
    for ((i = 1; i <= runs; i++)); do
      time_command "$tree_rss_flag" "$engine_out" "$display" "$script" "$html" "$engine_out"
      echo "  run $i/$runs: $(format_ms "$(awk 'NR == '"$((i + 1))"' {print $1}' "$engine_log")")"
    done

    gowk_pages=$(awk 'NR == 1 {print $3}' "$gowk_log")
    engine_pages=$(awk 'NR == 1 {print $3}' "$engine_log")
    if [ "$HAVE_GS" -eq 1 ]; then
      if [ "$gowk_pages" -ne "$pages" ]; then
        echo "  warning: gowk rendered $gowk_pages pages, requested $pages" >&2
      fi
      if [ "$engine_pages" -ne "$pages" ]; then
        echo "  warning: $display rendered $engine_pages pages, requested $pages" >&2
      fi
      while read -r _ _ rendered_pages _; do
        if [ "$rendered_pages" -ne "$pages" ]; then
          echo "  warning: gowk rendered $rendered_pages pages in one run, requested $pages" >&2
        fi
      done <"$gowk_log"
      while read -r _ _ rendered_pages _; do
        if [ "$rendered_pages" -ne "$pages" ]; then
          echo "  warning: $display rendered $rendered_pages pages in one run, requested $pages" >&2
        fi
      done <"$engine_log"
    fi

    gowk_elapsed=$(awk 'NR > 1 {print $1}' "$gowk_log" >"$gowk_log.elapsed" && median_value "$gowk_log.elapsed")
    engine_elapsed=$(awk 'NR > 1 {print $1}' "$engine_log" >"$engine_log.elapsed" && median_value "$engine_log.elapsed")
    gowk_rss=$(awk 'NR > 1 {print $2}' "$gowk_log" >"$gowk_log.rss" && median_value "$gowk_log.rss")
    engine_rss=$(awk 'NR > 1 {print $2}' "$engine_log" >"$engine_log.rss" && median_value "$engine_log.rss")
    gowk_bytes=$(awk -v timed_runs="$runs" 'NR == timed_runs + 1 {print $4}' "$gowk_log")
    engine_bytes=$(awk -v timed_runs="$runs" 'NR == timed_runs + 1 {print $4}' "$engine_log")
    speedup=$(awk -v e="$engine_elapsed" -v g="$gowk_elapsed" 'BEGIN { printf "%.2f", e / g }')

    printf '%-8d | %-12s | %-16s | %-8s | %-14s | %-20s | %-8d\n' \
      "$pages" \
      "$(format_ms "$gowk_elapsed")" \
      "$(format_ms "$engine_elapsed")" \
      "${speedup}x" \
      "$(comma "$gowk_rss") KiB" \
      "$(comma "$engine_rss") KiB" \
      "$gowk_bytes"

    rows+=("$pages|$gowk_elapsed|$engine_elapsed|$gowk_rss|$engine_rss|$gowk_bytes|$engine_bytes")
  done

  echo "================================================================================================"

  csv="$ARTIFACT_DIR/${engine}-compare-results.csv"
  md="$ARTIFACT_DIR/${engine}-compare.md"
  mkdir -p "$ARTIFACT_DIR"

  {
    printf 'pages,gowk_ms,%s_ms,speedup,gowk_rss_kib,%s_rss_kib,gowk_pdf_bytes,%s_pdf_bytes\n' \
      "$engine" "$engine" "$engine"
    for row in "${rows[@]}"; do
      IFS='|' read -r pages g_elapsed e_elapsed g_rss e_rss g_bytes e_bytes <<<"$row"
      g_ms=$(awk -v s="$g_elapsed" 'BEGIN { printf "%d", int(s * 1000 + 0.5) }')
      e_ms=$(awk -v s="$e_elapsed" 'BEGIN { printf "%d", int(s * 1000 + 0.5) }')
      speedup=$(awk -v e="$e_elapsed" -v g="$g_elapsed" 'BEGIN { printf "%.3f", e / g }')
      printf '%d,%d,%d,%s,%d,%d,%d,%d\n' \
        "$pages" "$g_ms" "$e_ms" "$speedup" "$g_rss" "$e_rss" "$g_bytes" "$e_bytes"
    done
  } >"$csv"

  {
    printf '# Direct CLI comparison: gowkhtmltopdf vs %s\n\n' "$display"
    echo 'Process-level measurement. Each cell is the median of' \
      "$runs" 'timed runs after one warmup.'
    echo 'Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).'
    printf 'Fixture: `%s` (20 invoice rows per requested page).\n' "$TEMPLATE"
    printf 'Host: %s (%s CPUs); toolchain: %s; gowkhtmltopdf: %s.\n' \
      "$host" "$cpu_count" "$toolchain" "$gowk_version"
    printf '%s\n' "$page_count_note"
    printf 'gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; %s.\n' "$(engine_flags_note "$engine")"
    printf '%s; gowkhtmltopdf RSS is `%%M`.\n\n' "$(engine_rss_note "$engine")"
    printf -- '- gowkhtmltopdf: `%s` (generic CLI)\n' "$GOWK_BIN"
    printf -- '- %s: `%s` (%s)\n' "$display" "$script" "$version"
    printf -- '- Reproduce: `./scripts/bench-external.sh --engines=%s` (or `make bench`)\n\n' "$engine"
    printf '| Pages | Gowk time | %s time | Speedup | Gowk RSS |' "$display"
    printf ' %s RSS | Gowk PDF bytes | %s PDF bytes |\n' "$display" "$display"
    echo '|---:|---:|---:|---:|---:|---:|---:|---:|'
    for row in "${rows[@]}"; do
      IFS='|' read -r pages g_elapsed e_elapsed g_rss e_rss g_bytes e_bytes <<<"$row"
      speedup=$(awk -v e="$e_elapsed" -v g="$g_elapsed" 'BEGIN { printf "%.2f", e / g }')
      printf '| %d | %s | %s | %sx | %s | %s | %s | %s |\n' \
        "$pages" \
        "$(format_ms "$g_elapsed")" \
        "$(format_ms "$e_elapsed")" \
        "$speedup" \
        "$(comma "$g_rss") KiB" \
        "$(comma "$e_rss") KiB" \
        "$(comma "$g_bytes")" \
        "$(comma "$e_bytes")"
    done
  } >"$md"

  echo "wrote $csv and $md"
done
