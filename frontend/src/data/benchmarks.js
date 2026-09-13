// Benchmark data for the Benchmarks page.
//
// Current capture: 2026-09-13 full-matrix capture on the 0.2.6 release tree
// (VERSION 0.2.6). CLI and external rows are cold full-process runs: median of
// three timed runs after one warmup. All three engine tables share the same
// gowk CLI column from the capture's make bench-cli-compare run. In-process
// rows: median of three fresh-process -benchtime=1x -count=1 rounds; B/op and
// allocs/op are the median-time sample's raw values, never averages; the
// 2-page row is cold (first conversion in its process) and later rows are warm.
// Python rows: warm median of 10 timed iterations after one warmup.
// Consolidated record: documentation/benchmarks.md. Raw results:
// plans/0.2.6/perf-review/results/2026-09-13/.
//
// Historical snapshots stay dated and are never relabeled: the 2026-09-12 full
// capture, the 2026-09-11 perf-time closure capture, the 2026-09-11 recovery
// capture, and the 2026-08-19 full 0.2.4 matrices below.

export const SNAPSHOT = {
  date: '2026-09-13',
  host: 'Linux amd64, 13th Gen Intel Core i7-13700HX (WSL2, 24 CPUs)',
  go: 'go1.26.4',
  gowk: 'gowkhtmltopdf 0.2.6 generic CLI, freshly built',
  wkhtml: 'wkhtmltopdf 0.12.6.1 (with patched qt)',
  flags: '--quiet --allow-local-files -o OUTPUT INPUT',
  method: 'median of 3 timed process runs after 1 warmup',
  fixture: 'report.html.tmpl, 20 invoice rows per requested page',
}

export const CLI_ROWS = [
  { pages: 2, gowkMs: 13, wkMs: 258, speedup: 19.68, gowkRss: 19584, wkRss: 44528, gowkBytes: 34210, wkBytes: 18486 },
  { pages: 5, gowkMs: 18, wkMs: 269, speedup: 14.68, gowkRss: 22080, wkRss: 44784, gowkBytes: 42795, wkBytes: 30584 },
  { pages: 10, gowkMs: 24, wkMs: 279, speedup: 11.65, gowkRss: 24384, wkRss: 45888, gowkBytes: 57239, wkBytes: 50994 },
  { pages: 20, gowkMs: 35, wkMs: 310, speedup: 8.75, gowkRss: 26304, wkRss: 47300, gowkBytes: 84680, wkBytes: 90742 },
  { pages: 50, gowkMs: 67, wkMs: 393, speedup: 5.84, gowkRss: 29376, wkRss: 51652, gowkBytes: 167525, wkBytes: 210678 },
  { pages: 100, gowkMs: 124, wkMs: 532, speedup: 4.30, gowkRss: 35520, wkRss: 59172, gowkBytes: 306321, wkBytes: 411260 },
  { pages: 200, gowkMs: 240, wkMs: 814, speedup: 3.39, gowkRss: 45888, wkRss: 74356, gowkBytes: 583670, wkBytes: 816285 },
  { pages: 250, gowkMs: 279, wkMs: 973, speedup: 3.49, gowkRss: 52032, wkRss: 81632, gowkBytes: 722322, wkBytes: 1019315 },
  { pages: 500, gowkMs: 573, wkMs: 1718, speedup: 3.00, gowkRss: 80448, wkRss: 123068, gowkBytes: 1420537, wkBytes: 2036776 },
]

export const WEASYPRINT_ROWS = [
  { pages: 2, gowkMs: 13, engineMs: 639, speedup: 49.18, gowkRss: 19584, engineRss: 81744, gowkBytes: 34210, engineBytes: 15584 },
  { pages: 10, gowkMs: 24, engineMs: 1435, speedup: 59.78, gowkRss: 24384, engineRss: 110976, gowkBytes: 57239, engineBytes: 45174 },
  { pages: 50, gowkMs: 67, engineMs: 5496, speedup: 82.04, gowkRss: 29376, engineRss: 251804, gowkBytes: 167525, engineBytes: 190544 },
  { pages: 100, gowkMs: 124, engineMs: 10953, speedup: 88.33, gowkRss: 35520, engineRss: 427372, gowkBytes: 306321, engineBytes: 372868 },
]

export const PUPPETEER_ROWS = [
  { pages: 2, gowkMs: 13, engineMs: 1452, speedup: 111.73, gowkRss: 19584, engineRss: 942964, gowkBytes: 34210, engineBytes: 134319 },
  { pages: 10, gowkMs: 24, engineMs: 1479, speedup: 61.63, gowkRss: 24384, engineRss: 1022844, gowkBytes: 57239, engineBytes: 450799 },
  { pages: 50, gowkMs: 67, engineMs: 1785, speedup: 26.65, gowkRss: 29376, engineRss: 1114080, gowkBytes: 167525, engineBytes: 1981892 },
  { pages: 100, gowkMs: 124, engineMs: 2158, speedup: 17.4, gowkRss: 35520, engineRss: 1241028, gowkBytes: 306321, engineBytes: 3936067 },
]

export const HISTORY_DATE = '2026-08-19'

export const CURRENT_CAPTURE = {
  date: '2026-09-13',
  label: 'full-matrix capture',
  raw: 'plans/0.2.6/perf-review/results/2026-09-13/',
}

// Current 2026-09-13 in-process internal matrix: full ascending warm matrix
// in one process per round, median of three rounds. The 2-page row is the
// first conversion in its process and includes the one-time font load.
export const INPROC_PDF_GENERIC = [
  { n: 2, ms: 5.2, mb: 4.1, allocs: '4.1K' },
  { n: 5, ms: 7.89, mb: 6.84, allocs: '8.2K' },
  { n: 10, ms: 11.75, mb: 4.63, allocs: '14.9K' },
  { n: 20, ms: 21.44, mb: 5.2, allocs: '28.4K' },
  { n: 50, ms: 54.76, mb: 11.56, allocs: '69.4K' },
  { n: 100, ms: 103.51, mb: 23.19, allocs: '137.9K' },
  { n: 200, ms: 207.91, mb: 44.73, allocs: '275.2K' },
  { n: 250, ms: 262.21, mb: 56.49, allocs: '344.0K' },
  { n: 500, ms: 539.33, mb: 111.43, allocs: '687.4K' },
]

// Current 2026-09-13 make bench-engine workloads (2 / 10 / 100 / 500), median
// of three fresh processes. PDF pages run after the image workloads in the
// same process, so those rows are warm and read lower than the standalone
// INPROC_PDF_GENERIC matrix above.
export const INPROC_TEMPLATE_GENERIC = [
  { n: 2, ms: 3.34, mb: 2.12, allocs: '4.2K' },
  { n: 10, ms: 11.04, mb: 3.13, allocs: '15.9K' },
  { n: 100, ms: 104.47, mb: 24.36, allocs: '148.1K' },
  { n: 500, ms: 539.47, mb: 116.27, allocs: '738.6K' },
]

export const INPROC_WEB_FETCH = [
  { n: 2, ms: 10.51, mb: 2.82, allocs: '1.7K' },
  { n: 10, ms: 11.44, mb: 3.1, allocs: '2.1K' },
  { n: 100, ms: 38.03, mb: 6.93, allocs: '5.3K' },
  { n: 500, ms: 38.44, mb: 11.54, allocs: '19.9K' },
]

export const INPROC_INLINE = [
  { n: 2, ms: 10.96, mb: 5.85, allocs: '1.2K' },
  { n: 10, ms: 9.1, mb: 2.93, allocs: '1.6K' },
  { n: 100, ms: 40.2, mb: 20.92, allocs: '4.7K' },
  { n: 500, ms: 35.79, mb: 11.82, allocs: '19.4K' },
]

export const LIBRARY_PDF = [
  { n: 2, ms: 5.5, mb: 4.11, allocs: '4.2K' },
  { n: 5, ms: 8.99, mb: 6.87, allocs: '8.2K' },
  { n: 10, ms: 13.41, mb: 3.85, allocs: '14.9K' },
  { n: 20, ms: 21.92, mb: 6.09, allocs: '28.4K' },
  { n: 50, ms: 54.49, mb: 12.55, allocs: '69.5K' },
  { n: 100, ms: 107.2, mb: 23.52, allocs: '137.9K' },
  { n: 200, ms: 214.32, mb: 45.39, allocs: '275.2K' },
  { n: 250, ms: 267.34, mb: 57.33, allocs: '344.0K' },
  { n: 500, ms: 554.56, mb: 113.09, allocs: '687.4K' },
]

// Current 2026-09-13 public image rows. Lossless PNG output stays at about
// 141,917 B for 250 tiles and 282,749 B for 500 tiles. The 250+ tile canvases
// take the direct-raster path, so 250 and 500 tiles are faster than 200.
export const LIBRARY_IMAGE = [
  { n: 2, ms: 14.26, mb: 11.91, allocs: '490' },
  { n: 5, ms: 11.4, mb: 3.45, allocs: '578' },
  { n: 10, ms: 13.33, mb: 3.68, allocs: '873' },
  { n: 20, ms: 14.1, mb: 3.82, allocs: '1.1K' },
  { n: 50, ms: 16.67, mb: 4.2, allocs: '1.9K' },
  { n: 100, ms: 28.82, mb: 20.0, allocs: '3.2K' },
  { n: 200, ms: 59.39, mb: 37.09, allocs: '5.8K' },
  { n: 250, ms: 16.86, mb: 6.38, allocs: '7.2K' },
  { n: 500, ms: 33.51, mb: 9.92, allocs: '13.7K' },
]

// Current 2026-09-13 Python c-shared rows: warm median of 10 timed iterations
// after one warmup, same engine and fixture as the Go library. Kept separate
// from LIBRARY_PDF / LIBRARY_IMAGE because the method is warm, not 1x fresh.
export const PYTHON_LIBRARY_PDF = [
  { n: 2, ms: 3.52 },
  { n: 5, ms: 5.75 },
  { n: 10, ms: 10.76 },
  { n: 20, ms: 21.91 },
  { n: 50, ms: 49.63 },
  { n: 100, ms: 98.44 },
  { n: 200, ms: 198.74 },
  { n: 250, ms: 246.38 },
  { n: 500, ms: 504.93 },
]

export const PYTHON_LIBRARY_IMAGE = [
  { n: 2, ms: 9.67 },
  { n: 5, ms: 10.01 },
  { n: 10, ms: 11.91 },
  { n: 20, ms: 11.76 },
  { n: 50, ms: 15.06 },
  { n: 100, ms: 25.19 },
  { n: 200, ms: 47.29 },
  { n: 250, ms: 15.65 },
  { n: 500, ms: 31.66 },
]

// Historical 2026-09-11 perf-improve phase-7 engine rows. Dated, not current.
export const INPROC_PDF_GENERIC_IMPROVE_HISTORY = [
  { n: 2, ms: 6.11, mb: 2.67, allocs: '6.2K' },
  { n: 500, ms: 1297.98, mb: 235.5, allocs: '1.23M' },
]

export const LIBRARY_PDF_IMPROVE_HISTORY = [
  { n: 2, ms: 7.14, mb: 2.68, allocs: '6.2K' },
  { n: 500, ms: 1240.82, mb: 236.91, allocs: '1.23M' },
]

export const LIBRARY_IMAGE_IMPROVE_HISTORY = [
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
}

// Public library 2-page row from the 2026-09-13 capture, used for the
// landing-page multiplier against the same capture's wkhtmltopdf CLI baseline.
export const LIBRARY_2P_MS = 5.5

export const LIBRARY_HEADLINE = {
  pages: LIBRARY_PDF[0].n,
  ms: LIBRARY_2P_MS,
  wkMs: CLI_ROWS[0].wkMs,
  multiplier: CLI_ROWS[0].wkMs / LIBRARY_2P_MS,
  displayMultiplier: Math.round(CLI_ROWS[0].wkMs / LIBRARY_2P_MS / 10) * 10,
}
