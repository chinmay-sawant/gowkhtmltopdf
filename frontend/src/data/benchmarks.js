// Benchmark snapshots for the Benchmarks page.
//
// Current engine snapshot: 2026-09-11 perf-improve phase-7 capture on the
// uncommitted 0.2.6 warm-path working tree (VERSION 0.2.5). In-process and
// library rows are three independent 1x samples per workload (median of the
// three raw values, B/op is never averaged). Raw evidence:
// plans/0.2.6/perf-improve/results/phase-7/final-capture.md and Snapshot L in
// testdata/golden/benchmarks/benchmark-results.txt.
//
// Current CLI comparison: the 2026-09-11 recovery capture, kept dated because
// the phase-7 capture measured gowkhtmltopdf only (cli-rss mode).
//
// Historical snapshot: 2026-08-19 full 0.2.4 matrices, kept dated so no 0.2.4
// number is relabeled as current.

export const SNAPSHOT = {
  date: '2026-09-11',
  host: 'Linux amd64, 13th Gen Intel Core i7-13700HX (WSL2, 24 CPUs)',
  go: 'go1.26.4',
  gowk: 'gowkhtmltopdf 0.2.6 recovery working tree (VERSION 0.2.5), freshly built generic CLI',
  wkhtml: 'wkhtmltopdf 0.12.6.1 (with patched qt)',
  flags: '--quiet --allow-local-files -o OUTPUT INPUT',
  method: 'median of 3 timed process runs after 1 warmup',
  fixture: 'report.html.tmpl, 20 invoice rows per requested page',
}

export const CLI_ROWS = [
  { pages: 2, gowkMs: 17, wkMs: 258, speedup: 14.95, gowkRss: 24576, wkRss: 44716, gowkBytes: 34209, wkBytes: 18486 },
  { pages: 5, gowkMs: 23, wkMs: 266, speedup: 11.44, gowkRss: 26496, wkRss: 45068, gowkBytes: 42791, wkBytes: 30584 },
  { pages: 10, gowkMs: 34, wkMs: 286, speedup: 8.29, gowkRss: 28608, wkRss: 46024, gowkBytes: 57231, wkBytes: 50994 },
  { pages: 20, gowkMs: 53, wkMs: 306, speedup: 5.79, gowkRss: 33984, wkRss: 47772, gowkBytes: 84654, wkBytes: 90742 },
  { pages: 50, gowkMs: 122, wkMs: 394, speedup: 3.23, gowkRss: 47616, wkRss: 52240, gowkBytes: 167442, wkBytes: 210678 },
  { pages: 100, gowkMs: 229, wkMs: 541, speedup: 2.36, gowkRss: 69120, wkRss: 59380, gowkBytes: 306144, wkBytes: 411260 },
  { pages: 200, gowkMs: 468, wkMs: 830, speedup: 1.77, gowkRss: 113280, wkRss: 74492, gowkBytes: 583231, wkBytes: 816285 },
  { pages: 250, gowkMs: 599, wkMs: 988, speedup: 1.65, gowkRss: 139584, wkRss: 81884, gowkBytes: 721739, wkBytes: 1019315 },
  { pages: 500, gowkMs: 1288, wkMs: 1753, speedup: 1.36, gowkRss: 240960, wkRss: 123172, gowkBytes: 1419234, wkBytes: 2036776 },
]

export const WEASYPRINT_ROWS = [
  { pages: 2, gowkMs: 21, engineMs: 653, speedup: 31.13, gowkRss: 25152, engineRss: 81932, gowkBytes: 34209, engineBytes: 15586 },
  { pages: 10, gowkMs: 36, engineMs: 1431, speedup: 39.80, gowkRss: 28416, engineRss: 111444, gowkBytes: 57231, engineBytes: 45172 },
  { pages: 50, gowkMs: 124, engineMs: 5482, speedup: 44.07, gowkRss: 48000, engineRss: 252448, gowkBytes: 167442, engineBytes: 190545 },
  { pages: 100, gowkMs: 245, engineMs: 11119, speedup: 45.29, gowkRss: 65856, engineRss: 427880, gowkBytes: 306144, engineBytes: 372867 },
]

export const PUPPETEER_ROWS = [
  { pages: 2, gowkMs: 21, engineMs: 1470, speedup: 68.85, gowkRss: 24960, engineRss: 973260, gowkBytes: 34209, engineBytes: 134319 },
  { pages: 10, gowkMs: 37, engineMs: 1550, speedup: 42.45, gowkRss: 28800, engineRss: 1021160, gowkBytes: 57231, engineBytes: 450799 },
  { pages: 50, gowkMs: 120, engineMs: 1815, speedup: 15.08, gowkRss: 48576, engineRss: 1114380, gowkBytes: 167442, engineBytes: 1981892 },
  { pages: 100, gowkMs: 249, engineMs: 2179, speedup: 8.74, gowkRss: 68928, engineRss: 1241476, gowkBytes: 306144, engineBytes: 3936067 },
]

export const RECOVERY_DATE = '2026-09-11'
export const HISTORY_DATE = '2026-08-19'
export const PERF_IMPROVE_CAPTURE = {
  date: '2026-09-11',
  label: 'perf-improve phase-7 capture',
  raw: 'plans/0.2.6/perf-improve/results/phase-7/final-capture.md',
}

// Current 2026-09-11 perf-improve capture: fresh-process 1x samples, median
// of three. The 2-page PDF B/op rows carry the one-time default-font cost,
// now about 2.7 MB, charged to the single operation, so they are not
// like-for-like with the 2026-08-19 multi-iteration rows below. The warm
// 500-page B/op now meets the 240 MB acceptance (235.50 MB standalone median;
// 234.92 MB warm matrix); the 500-page warm time (1,297.98 ms standalone
// median; 1,228.72 ms warm matrix) is still above the 1.010 s Snapshot I row.
export const INPROC_PDF_GENERIC = [
  { n: 2, ms: 6.11, mb: 2.67, allocs: '6.2K' },
  { n: 500, ms: 1297.98, mb: 235.5, allocs: '1.23M' },
]

export const LIBRARY_PDF = [
  { n: 2, ms: 7.14, mb: 2.68, allocs: '6.2K' },
  { n: 500, ms: 1240.82, mb: 236.91, allocs: '1.23M' },
]

export const LIBRARY_IMAGE = [
  { n: 250, ms: 50.72, mb: 14.45, allocs: '9.4K' },
  { n: 500, ms: 98.47, mb: 26.73, allocs: '18.0K' },
]

// Historical 2026-08-19 full 0.2.4 matrices. Kept dated; not current claims.
export const INPROC_PDF_GENERIC_HISTORY = [
  { n: 2, ms: 3.58, mb: 2.21, allocs: '5.8K', multiplier: 72.43 },
  { n: 5, ms: 7.29, mb: 3.61, allocs: '12.7K', multiplier: 36.74 },
  { n: 10, ms: 14.86, mb: 6.08, allocs: '24.1K', multiplier: 18.58 },
  { n: 20, ms: 28.64, mb: 10.73, allocs: '46.9K', multiplier: 11.07 },
  { n: 50, ms: 84.17, mb: 24.98, allocs: '115.8K', multiplier: 4.82 },
  { n: 100, ms: 157.69, mb: 48.73, allocs: '230.5K', multiplier: 3.34 },
  { n: 200, ms: 384.45, mb: 96.32, allocs: '460.3K', multiplier: 2.11 },
  { n: 250, ms: 449.51, mb: 119.36, allocs: '575.3K', multiplier: 2.14 },
  { n: 500, ms: 1009.8, mb: 237.76, allocs: '1.15M', multiplier: 1.65 },
]

export const INPROC_TEMPLATE_GENERIC_HISTORY = [
  { n: 2, ms: 3.34, mb: 2.23, allocs: '6.0K' },
  { n: 5, ms: 8.27, mb: 3.68, allocs: '13.2K' },
  { n: 10, ms: 14.93, mb: 6.23, allocs: '25.1K' },
  { n: 20, ms: 33.8, mb: 11.02, allocs: '49.0K' },
  { n: 50, ms: 80.62, mb: 25.56, allocs: '120.9K' },
  { n: 100, ms: 182.06, mb: 49.95, allocs: '240.8K' },
  { n: 200, ms: 374.28, mb: 98.79, allocs: '480.8K' },
  { n: 250, ms: 497.18, mb: 121.91, allocs: '600.8K' },
  { n: 500, ms: 1033.27, mb: 242.68, allocs: '1.20M' },
]

export const INPROC_WEB_FETCH_HISTORY = [
  { n: 2, ms: 11.43, mb: 7.46, allocs: '2.0K' },
  { n: 5, ms: 11.97, mb: 7.75, allocs: '2.2K' },
  { n: 10, ms: 13.43, mb: 8.35, allocs: '2.7K' },
  { n: 20, ms: 16.92, mb: 10.07, allocs: '3.1K' },
  { n: 50, ms: 26.91, mb: 15.2, allocs: '4.5K' },
  { n: 100, ms: 43.67, mb: 23.8, allocs: '6.9K' },
  { n: 200, ms: 78.72, mb: 41.01, allocs: '11.5K' },
  { n: 250, ms: 90.66, mb: 49.44, allocs: '13.9K' },
  { n: 500, ms: 171.25, mb: 92.48, allocs: '25.5K' },
]

export const INPROC_INLINE_HISTORY = [
  { n: 2, ms: 16.01, mb: 13.96, allocs: '2.1K' },
  { n: 5, ms: 10.2, mb: 7.26, allocs: '1.7K' },
  { n: 10, ms: 11.07, mb: 7.87, allocs: '2.1K' },
  { n: 20, ms: 13.56, mb: 9.61, allocs: '2.5K' },
  { n: 50, ms: 25.47, mb: 14.75, allocs: '3.9K' },
  { n: 100, ms: 42.97, mb: 23.4, allocs: '6.3K' },
  { n: 200, ms: 71.34, mb: 40.71, allocs: '11.0K' },
  { n: 250, ms: 91.51, mb: 49.21, allocs: '13.3K' },
  { n: 500, ms: 174.12, mb: 92.45, allocs: '24.9K' },
]

export const LIBRARY_PDF_HISTORY = [
  { n: 2, ms: 3.77, mb: 1.48, allocs: '5.7K', multiplier: 68.73 },
  { n: 5, ms: 8.33, mb: 3.06, allocs: '12.6K', multiplier: 32.19 },
  { n: 10, ms: 16.03, mb: 5.38, allocs: '24.1K', multiplier: 17.22 },
  { n: 20, ms: 31.19, mb: 10.31, allocs: '46.9K', multiplier: 10.16 },
  { n: 50, ms: 74.83, mb: 24.83, allocs: '115.8K', multiplier: 5.43 },
  { n: 100, ms: 160.94, mb: 48.46, allocs: '230.5K', multiplier: 3.27 },
  { n: 200, ms: 337.38, mb: 95.63, allocs: '460.3K', multiplier: 2.4 },
  { n: 250, ms: 441.4, mb: 118.89, allocs: '575.3K', multiplier: 2.18 },
  { n: 500, ms: 1104.51, mb: 236.85, allocs: '1.15M', multiplier: 1.51 },
]

export const LIBRARY_IMAGE_HISTORY = [
  { n: 2, ms: 11.07, mb: 4.17, allocs: '444' },
  { n: 5, ms: 13.43, mb: 6.13, allocs: '647' },
  { n: 10, ms: 14.61, mb: 6.81, allocs: '1.0K' },
  { n: 20, ms: 14.67, mb: 6.1, allocs: '1.4K' },
  { n: 50, ms: 17.42, mb: 7.32, allocs: '2.4K' },
  { n: 100, ms: 30.11, mb: 8.26, allocs: '4.1K' },
  { n: 200, ms: 63.03, mb: 19.35, allocs: '7.5K' },
  { n: 250, ms: 73.06, mb: 20.66, allocs: '9.2K' },
  { n: 500, ms: 142.59, mb: 52.0, allocs: '17.6K' },
]

export const INPROC_SNAPSHOT_DATE = HISTORY_DATE

export const CHART_PAGES = [2, 10, 50, 100, 500]

export function speedup(row) {
  return row.speedup ?? row.wkMs / row.gowkMs
}

export function externalSpeedup(row) {
  return row.speedup ?? row.engineMs / row.gowkMs
}

export function relativeMultiplier(row, baselineRows = CLI_ROWS) {
  const baseline = baselineRows.find((item) => item.pages === row.n)
  return row.multiplier ?? (baseline ? baseline.wkMs / row.ms : null)
}

export function rssDelta(row) {
  return (row.wkRss - row.gowkRss) / row.wkRss
}

export function formatMs(ms) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(3)} s`
  if (Number.isInteger(ms)) return `${ms} ms`
  return `${ms} ms`
}

export function formatKiB(kib) {
  return `${kib.toLocaleString('en-US')} KiB`
}

export function formatBytes(n) {
  return n.toLocaleString('en-US')
}

export function formatSpeedup(n) {
  return `${n.toFixed(2)}x`
}

export function formatRssDelta(row) {
  const delta = rssDelta(row)
  if (Math.abs(delta) < 0.02) return 'about even'
  if (delta > 0) return `${Math.round(delta * 100)}% less RSS`
  return `${Math.round(-delta * 100)}% more RSS`
}

export const HEADLINE = {
  smallPages: 2,
  smallSpeedup: speedup(CLI_ROWS[0]),
  smallGowk: CLI_ROWS[0].gowkMs,
  smallWk: CLI_ROWS[0].wkMs,
  largePages: 500,
  largeSpeedup: speedup(CLI_ROWS[CLI_ROWS.length - 1]),
  largeGowk: CLI_ROWS[CLI_ROWS.length - 1].gowkMs,
  largeWk: CLI_ROWS[CLI_ROWS.length - 1].wkMs,
  rssCrossover: 50,
}

// Dated landing-claim anchor: the 2026-09-11 recovery capture public library
// 2-page row (10.88 ms) against that capture's wkhtmltopdf CLI baseline
// (258 ms). The phase-7 capture measured 7.14 ms on the same boundary; the
// landing copy keeps the dated anchor until that claim is re-approved.
export const RECOVERY_LIBRARY_2P_MS = 10.88

export const LIBRARY_HEADLINE = {
  pages: LIBRARY_PDF[0].n,
  ms: RECOVERY_LIBRARY_2P_MS,
  wkMs: CLI_ROWS[0].wkMs,
  multiplier: CLI_ROWS[0].wkMs / RECOVERY_LIBRARY_2P_MS,
  displayMultiplier: Math.round(CLI_ROWS[0].wkMs / RECOVERY_LIBRARY_2P_MS / 10) * 10,
}
