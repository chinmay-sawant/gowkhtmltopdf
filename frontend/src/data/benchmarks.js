// Benchmark data for the Benchmarks page.
//
// Current capture: 2026-09-12 full-matrix capture on the 0.2.6 working tree
// (VERSION 0.2.5). CLI and external rows: median of three timed runs after one
// warmup. In-process rows: median of three fresh-process -benchtime=1x
// -count=1 rounds; B/op and allocs/op are the median-time sample's raw values,
// never averages. Consolidated record: documentation/benchmarks.md. Raw
// results: plans/0.2.6/perf-review/results/2026-09-12/.
//
// Historical snapshots stay dated and are never relabeled: the 2026-09-11
// perf-time closure capture, the 2026-09-11 recovery capture, and the
// 2026-08-19 full 0.2.4 matrices below.

export const SNAPSHOT = {
  date: '2026-09-12',
  host: 'Linux amd64, 13th Gen Intel Core i7-13700HX (WSL2, 24 CPUs)',
  go: 'go1.26.4',
  gowk: 'gowkhtmltopdf 0.2.5 generic CLI (0.2.6 working tree), freshly built',
  wkhtml: 'wkhtmltopdf 0.12.6.1 (with patched qt)',
  flags: '--quiet --allow-local-files -o OUTPUT INPUT',
  method: 'median of 3 timed process runs after 1 warmup',
  fixture: 'report.html.tmpl, 20 invoice rows per requested page',
}

export const CLI_ROWS = [
  { pages: 2, gowkMs: 14, wkMs: 260, speedup: 18.5, gowkRss: 19200, wkRss: 44720, gowkBytes: 34210, wkBytes: 18486 },
  { pages: 5, gowkMs: 19, wkMs: 269, speedup: 14.37, gowkRss: 21696, wkRss: 45168, gowkBytes: 42795, wkBytes: 30584 },
  { pages: 10, gowkMs: 26, wkMs: 286, speedup: 11.18, gowkRss: 25152, wkRss: 46016, gowkBytes: 57239, wkBytes: 50994 },
  { pages: 20, gowkMs: 36, wkMs: 315, speedup: 8.74, gowkRss: 25920, wkRss: 47620, gowkBytes: 84680, wkBytes: 90742 },
  { pages: 50, gowkMs: 69, wkMs: 403, speedup: 5.86, gowkRss: 29760, wkRss: 52128, gowkBytes: 167525, wkBytes: 210678 },
  { pages: 100, gowkMs: 126, wkMs: 546, speedup: 4.35, gowkRss: 35904, wkRss: 59460, gowkBytes: 306321, wkBytes: 411260 },
  { pages: 200, gowkMs: 234, wkMs: 852, speedup: 3.64, gowkRss: 44928, wkRss: 74308, gowkBytes: 583670, wkBytes: 816285 },
  { pages: 250, gowkMs: 288, wkMs: 1008, speedup: 3.5, gowkRss: 51264, wkRss: 81820, gowkBytes: 722322, wkBytes: 1019315 },
  { pages: 500, gowkMs: 562, wkMs: 1760, speedup: 3.13, gowkRss: 79296, wkRss: 123076, gowkBytes: 1420537, wkBytes: 2036776 },
]

export const WEASYPRINT_ROWS = [
  { pages: 2, gowkMs: 16, engineMs: 634, speedup: 40.86, gowkRss: 19584, engineRss: 81648, gowkBytes: 34210, engineBytes: 15584 },
  { pages: 10, gowkMs: 27, engineMs: 1434, speedup: 53.62, gowkRss: 24576, engineRss: 111104, gowkBytes: 57239, engineBytes: 45174 },
  { pages: 50, gowkMs: 71, engineMs: 5441, speedup: 76.88, gowkRss: 29760, engineRss: 252412, gowkBytes: 167525, engineBytes: 190544 },
  { pages: 100, gowkMs: 124, engineMs: 11072, speedup: 89.43, gowkRss: 34752, engineRss: 427468, gowkBytes: 306321, engineBytes: 372867 },
]

export const PUPPETEER_ROWS = [
  { pages: 2, gowkMs: 16, engineMs: 1445, speedup: 92.85, gowkRss: 19200, engineRss: 940600, gowkBytes: 34210, engineBytes: 134319 },
  { pages: 10, gowkMs: 26, engineMs: 1488, speedup: 57.16, gowkRss: 24192, engineRss: 1019952, gowkBytes: 57239, engineBytes: 450799 },
  { pages: 50, gowkMs: 72, engineMs: 1801, speedup: 25.1, gowkRss: 29760, engineRss: 1119360, gowkBytes: 167525, engineBytes: 1981892 },
  { pages: 100, gowkMs: 127, engineMs: 2178, speedup: 17.16, gowkRss: 35136, engineRss: 1240920, gowkBytes: 306321, engineBytes: 3936067 },
]

export const HISTORY_DATE = '2026-08-19'

export const CURRENT_CAPTURE = {
  date: '2026-09-12',
  label: 'full-matrix capture',
  raw: 'plans/0.2.6/perf-review/results/2026-09-12/',
}

// Current 2026-09-12 in-process internal matrix: full ascending warm matrix
// in one process per round, median of three rounds. The 2-page row is the
// first conversion in its process and includes the one-time font load.
export const INPROC_PDF_GENERIC = [
  { n: 2, ms: 5.14, mb: 4.09, allocs: '4.1K' },
  { n: 5, ms: 7.75, mb: 6.84, allocs: '8.2K' },
  { n: 10, ms: 11.77, mb: 4.63, allocs: '14.9K' },
  { n: 20, ms: 21.24, mb: 6.02, allocs: '28.4K' },
  { n: 50, ms: 53.54, mb: 13.24, allocs: '69.5K' },
  { n: 100, ms: 106.56, mb: 23.26, allocs: '138.0K' },
  { n: 200, ms: 204.88, mb: 44.66, allocs: '275.2K' },
  { n: 250, ms: 259.06, mb: 56.5, allocs: '344.0K' },
  { n: 500, ms: 535.34, mb: 111.44, allocs: '687.4K' },
]

export const LIBRARY_PDF = [
  { n: 2, ms: 5.75, mb: 4.11, allocs: '4.1K' },
  { n: 5, ms: 8.83, mb: 6.87, allocs: '8.2K' },
  { n: 10, ms: 13.71, mb: 4.67, allocs: '14.9K' },
  { n: 20, ms: 21.13, mb: 6.09, allocs: '28.4K' },
  { n: 50, ms: 51.83, mb: 12.59, allocs: '69.5K' },
  { n: 100, ms: 107.16, mb: 23.45, allocs: '137.9K' },
  { n: 200, ms: 222.68, mb: 45.32, allocs: '275.2K' },
  { n: 250, ms: 266.55, mb: 57.32, allocs: '344.0K' },
  { n: 500, ms: 532.24, mb: 113.1, allocs: '687.4K' },
]

// Current 2026-09-12 public image rows. Lossless PNG output stays at about
// 141,917 B for 250 tiles and 282,749 B for 500 tiles.
export const LIBRARY_IMAGE = [
  { n: 2, ms: 14.12, mb: 11.91, allocs: '490' },
  { n: 5, ms: 12.25, mb: 3.45, allocs: '578' },
  { n: 10, ms: 13.75, mb: 3.68, allocs: '872' },
  { n: 20, ms: 12.96, mb: 3.82, allocs: '1.1K' },
  { n: 50, ms: 17.39, mb: 4.2, allocs: '1.9K' },
  { n: 100, ms: 30.16, mb: 20.0, allocs: '3.2K' },
  { n: 200, ms: 55.8, mb: 37.09, allocs: '5.8K' },
  { n: 250, ms: 16.78, mb: 6.38, allocs: '7.2K' },
  { n: 500, ms: 34.02, mb: 10.4, allocs: '13.7K' },
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

// Public library 2-page row from the 2026-09-12 capture, used for the
// landing-page multiplier against the same capture's wkhtmltopdf CLI baseline.
export const LIBRARY_2P_MS = 5.75

export const LIBRARY_HEADLINE = {
  pages: LIBRARY_PDF[0].n,
  ms: LIBRARY_2P_MS,
  wkMs: CLI_ROWS[0].wkMs,
  multiplier: CLI_ROWS[0].wkMs / LIBRARY_2P_MS,
  displayMultiplier: Math.round(CLI_ROWS[0].wkMs / LIBRARY_2P_MS / 10) * 10,
}
