#!/usr/bin/env bash
# bench-external.sh — process-level CLI comparison of bin/gowkhtmltopdf
# against the installed WeasyPrint and Puppeteer engines.
#
# The bench never invokes engine commands itself: each engine is run through
# its print script (scripts/weasyprint/print.sh, scripts/puppeteer/print.sh),
# so the measured command is exactly what that script runs.
#
# Usage:
#   scripts/bench-external.sh [--engines weasyprint,puppeteer]
#                             [--sizes 2,10,50,100] [--runs 3]
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
ARTIFACT_DIR="$ROOT/testdata/golden/benchmarks"
TEMPLATE="$ROOT/testdata/golden/benchmarks/templates/report.html.tmpl"
POLL_INTERVAL=0.02
RUN_TIMEOUT=300
ROWS_PER_PAGE=20

sizes=(2 10 50 100)
runs=3
requested_engines=()

usage() {
  sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --engines)
      IFS=',' read -r -a requested_engines <<<"$2"
      shift 2
      ;;
    --sizes)
      IFS=',' read -r -a sizes <<<"$2"
      shift 2
      ;;
    --runs)
      runs=$2
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "bench-external: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

[ -x "$GOWK_BIN" ] || {
  echo "bench-external: $GOWK_BIN not found; run make build" >&2
  exit 1
}

[ -x /usr/bin/time ] || {
  echo "bench-external: /usr/bin/time not found (GNU time package)" >&2
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
        [ -f "$ROOT/scripts/puppeteer/node_modules/puppeteer-core/package.json" ]
      ;;
    *)
      return 1
      ;;
  esac
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
      weasyprint --version | sed -n '1s/^[[:space:]]*//p' | sed -n '1p'
      ;;
    puppeteer)
      local ver
      ver=$(awk -F'"' '/"version"/ { print $4; exit }' \
        "$ROOT/scripts/puppeteer/node_modules/puppeteer-core/package.json")
      if command -v google-chrome >/dev/null 2>&1; then
        printf 'puppeteer-core %s + %s' "$ver" "$(google-chrome --version | sed 's/[[:space:]]*$//')"
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
      printf 'Puppeteer printed via headless Chrome with `format: A4`, `printBackground: true`, `preferCSSPageSize: true`'
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
# Fixture generation (mirrors testdata/golden/benchmarks/templates/report.html.tmpl)
# ---------------------------------------------------------------------------

generate_report() { # pages outfile
  local pages=$1 out=$2 p line sku qty amount
  : >"$out"
  cat >>"$out" <<'EOF'
<!DOCTYPE html>
<!-- generated for bench-external.sh; content mirrors report.html.tmpl -->
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Benchmark report</title>
  <style>
    body { color: #172033; font-family: sans-serif; font-size: 9pt; margin: 0; }
    .benchmark-page { page-break-before: always; page-break-inside: avoid; padding: 2mm 0; }
    .benchmark-page.first { page-break-before: auto; }
    h1 { color: #174a7c; font-size: 16pt; margin: 0 0 3mm; }
    p { margin: 0 0 3mm; }
    table { border-collapse: collapse; width: 100%; }
    tr { page-break-inside: avoid; break-inside: avoid; }
    th, td { border: 1px solid #a8b5c5; padding: 1.5mm 2mm; white-space: nowrap; }
    th { background: #e6eef7; text-align: left; }
    td.amount { text-align: right; }
  </style>
</head>
<body>
EOF
  for ((p = 1; p <= pages; p++)); do
    {
      if [ "$p" -eq 1 ]; then
        printf '  <section class="benchmark-page first">\n'
      else
        printf '  <section class="benchmark-page">\n'
      fi
      printf '    <h1>Benchmark report — page %d</h1>\n' "$p"
      printf '    <p>Representative invoice and operations data for the full HTML-to-PDF pipeline.</p>\n'
      printf '    <table>\n      <thead>\n        <tr><th>Line</th><th>SKU</th><th>Description</th><th>Quantity</th><th>Amount</th></tr>\n      </thead>\n      <tbody>\n'
      for ((line = 1; line <= ROWS_PER_PAGE; line++)); do
        sku=$(printf 'SKU-%03d-%03d' "$p" "$line")
        qty=$(((line + p - 1) % 7 + 1))
        amount=$(printf '%d.%02d' $((p * line)) $(((p - 1 + line) % 100)))
        printf '        <tr><td>%d</td><td>%s</td><td>Platform operations and support service %d</td><td>%d</td><td class="amount">%s</td></tr>\n' \
          "$line" "$sku" "$line" "$qty" "$amount"
      done
      printf '      </tbody>\n    </table>\n  </section>\n'
    } >>"$out"
  done
  printf '</body>\n</html>\n' >>"$out"
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
  timefile=$(mktemp)
  start=$(date +%s.%N)
  if [ "$tree_rss" -eq 1 ]; then
    timeout "$RUN_TIMEOUT" /usr/bin/time -f '%e %M' -o "$timefile" -- "$@" &
    local pid=$!
    rss=$(poll_tree_rss "$pid")
    set +e
    wait "$pid"
    rc=$?
    set -e
  else
    set +e
    timeout "$RUN_TIMEOUT" /usr/bin/time -f '%e %M' -o "$timefile" -- "$@"
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
if [ ${#requested_engines[@]} -eq 0 ]; then
  requested_engines=(weasyprint puppeteer)
fi

for engine in "${requested_engines[@]}"; do
  if engine_available "$engine"; then
    engines+=("$engine")
  else
    echo "bench-external: $engine not installed; skipping" >&2
  fi
done

if [ ${#engines[@]} -eq 0 ]; then
  echo "bench-external: no external engine selected (--engines) or installed" >&2
  exit 0
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
  echo "flags:   gowkhtmltopdf used \`--quiet --enable-local-file-access\`; $(engine_flags_note "$engine")."
  echo "rss:     $(engine_rss_note "$engine"); gowkhtmltopdf RSS is \`%M\`."
  echo "runs:    warmup + $runs timed (median)"
  echo "================================================================================================"
  printf '%-8s | %-12s | %-16s | %-8s | %-14s | %-20s | %-8s\n' \
    "Pages" "gowk time" "$display time" "Speedup" "gowk RSS" "$display RSS" "gowk PDF"
  echo "------------------------------------------------------------------------------------------------"

  for pages in "${sizes[@]}"; do
    html="$tmp/doc_${pages}.html"
    generate_report "$pages" "$html"
    gowk_out="$tmp/gowk_${pages}.pdf"
    engine_out="$tmp/${engine}_${pages}.pdf"

    gowk_log="$tmp/gowk_${pages}.times"
    engine_log="$tmp/${engine}_${pages}.times"
    : >"$gowk_log"
    : >"$engine_log"

    echo "page size $pages: timing gowk..."
    RUN_LOG="$gowk_log"
    time_command 0 "$gowk_out" "gowkhtmltopdf" "$GOWK_BIN" --quiet --enable-local-file-access "$html" "$gowk_out"
    echo "  warmup: $(format_ms "$(awk 'NR == 1 {print $1}' "$gowk_log")") ($(awk 'NR == 1 {print $3}' "$gowk_log") pages)"
    for ((i = 1; i <= runs; i++)); do
      time_command 0 "$gowk_out" "gowkhtmltopdf" "$GOWK_BIN" --quiet --enable-local-file-access "$html" "$gowk_out"
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
    if [ "$gowk_pages" -ne "$pages" ]; then
      echo "  warning: gowk rendered $gowk_pages pages, requested $pages" >&2
    fi
    if [ "$engine_pages" -ne "$pages" ]; then
      echo "  warning: $display rendered $engine_pages pages, requested $pages" >&2
    fi

    gowk_elapsed=$(awk '{print $1}' "$gowk_log" >"$gowk_log.elapsed" && median_value "$gowk_log.elapsed")
    engine_elapsed=$(awk '{print $1}' "$engine_log" >"$engine_log.elapsed" && median_value "$engine_log.elapsed")
    gowk_rss=$(awk '{print $2}' "$gowk_log" >"$gowk_log.rss" && median_value "$gowk_log.rss")
    engine_rss=$(awk '{print $2}' "$engine_log" >"$engine_log.rss" && median_value "$engine_log.rss")
    gowk_bytes=$(awk 'NR == 1 {print $4}' "$gowk_log")
    engine_bytes=$(awk 'NR == 1 {print $4}' "$engine_log")
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
    printf 'gowkhtmltopdf used `--quiet --enable-local-file-access`; %s.\n' "$(engine_flags_note "$engine")"
    printf '%s; gowkhtmltopdf RSS is `%%M`.\n\n' "$(engine_rss_note "$engine")"
    printf -- '- gowkhtmltopdf: `%s` (generic CLI)\n' "$GOWK_BIN"
    printf -- '- %s: `%s` (%s)\n' "$display" "$script" "$version"
    printf -- '- Reproduce: `./scripts/bench-external.sh --engines=%s` (or `make bench-external`)\n\n' "$engine"
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
