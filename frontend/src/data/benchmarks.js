export const SNAPSHOT = {
  date: '2026-08-19',
  host: 'Linux amd64, 13th Gen Intel Core i7-13700HX (WSL2, 24 CPUs)',
  go: 'go1.26.4',
  gowk: 'gowkhtmltopdf 0.2.4, freshly built generic CLI',
  wkhtml: 'wkhtmltopdf 0.12.6.1 (with patched qt)',
  flags: '--quiet --allow-local-files -o OUTPUT INPUT',
  method: 'median of 3 timed process runs after 1 warmup',
  fixture: 'report.html.tmpl, 20 invoice rows per requested page',
}

export const CLI_ROWS = [
  { pages: 2, gowkMs: 17, wkMs: 259, speedup: 15.46, gowkRss: 23808, wkRss: 44192, gowkBytes: 34068, wkBytes: 18486 },
  { pages: 5, gowkMs: 22, wkMs: 268, speedup: 12.36, gowkRss: 24960, wkRss: 44716, gowkBytes: 42467, wkBytes: 30584 },
  { pages: 10, gowkMs: 30, wkMs: 276, speedup: 9.21, gowkRss: 27264, wkRss: 45992, gowkBytes: 56607, wkBytes: 50994 },
  { pages: 20, gowkMs: 45, wkMs: 317, speedup: 7.06, gowkRss: 30528, wkRss: 47464, gowkBytes: 83434, wkBytes: 90742 },
  { pages: 50, gowkMs: 112, wkMs: 406, speedup: 3.63, gowkRss: 43200, wkRss: 51856, gowkBytes: 164461, wkBytes: 210678 },
  { pages: 100, gowkMs: 184, wkMs: 526, speedup: 2.85, gowkRss: 61248, wkRss: 59048, gowkBytes: 300249, wkBytes: 411260 },
  { pages: 200, gowkMs: 376, wkMs: 811, speedup: 2.15, gowkRss: 96192, wkRss: 74192, gowkBytes: 571397, wkBytes: 816285 },
  { pages: 250, gowkMs: 480, wkMs: 964, speedup: 2.01, gowkRss: 116736, wkRss: 81740, gowkBytes: 707011, wkBytes: 1019315 },
  { pages: 500, gowkMs: 1042, wkMs: 1671, speedup: 1.60, gowkRss: 208128, wkRss: 123080, gowkBytes: 1390014, wkBytes: 2036776 },
]

export const WEASYPRINT_ROWS = [
  { pages: 2, gowkMs: 19, engineMs: 616, speedup: 32.15, gowkRss: 24576, engineRss: 77420, gowkBytes: 34068, engineBytes: 15587 },
  { pages: 10, gowkMs: 31, engineMs: 1352, speedup: 43.65, gowkRss: 26880, engineRss: 106000, gowkBytes: 56607, engineBytes: 45174 },
  { pages: 50, gowkMs: 100, engineMs: 5217, speedup: 52.01, gowkRss: 42624, engineRss: 246876, gowkBytes: 164461, engineBytes: 190546 },
  { pages: 100, gowkMs: 186, engineMs: 10528, speedup: 56.62, gowkRss: 58560, engineRss: 423004, gowkBytes: 300249, engineBytes: 372868 },
]

export const PUPPETEER_ROWS = [
  { pages: 2, gowkMs: 18, engineMs: 1411, speedup: 77.30, gowkRss: 23808, engineRss: 944056, gowkBytes: 34068, engineBytes: 102932 },
  { pages: 10, gowkMs: 32, engineMs: 1548, speedup: 47.84, gowkRss: 27264, engineRss: 1019896, gowkBytes: 56607, engineBytes: 406557 },
  { pages: 50, gowkMs: 121, engineMs: 2069, speedup: 17.06, gowkRss: 43008, engineRss: 1108580, gowkBytes: 164461, engineBytes: 1934728 },
  { pages: 100, gowkMs: 199, engineMs: 2145, speedup: 10.78, gowkRss: 62016, engineRss: 1245988, gowkBytes: 300249, engineBytes: 3884017 },
]

export const INPROC_SNAPSHOT_DATE = '2026-08-19'

export const CHART_PAGES = [2, 10, 50, 100, 500]

export const INPROC_PDF_GENERIC = [
  { n: 2, ms: 3.58, mb: 2.21, allocs: '5.8K', multiplier: 72.43 },
  { n: 5, ms: 7.29, mb: 3.61, allocs: '12.7K', multiplier: 36.74 },
  { n: 10, ms: 14.86, mb: 6.08, allocs: '24.1K', multiplier: 18.58 },
  { n: 20, ms: 28.64, mb: 10.73, allocs: '46.9K', multiplier: 11.07 },
  { n: 50, ms: 84.17, mb: 24.98, allocs: '115.8K', multiplier: 4.82 },
  { n: 100, ms: 157.69, mb: 48.73, allocs: '230.5K', multiplier: 3.34 },
  { n: 200, ms: 384.45, mb: 96.32, allocs: '460.3K', multiplier: 2.11 },
  { n: 250, ms: 449.51, mb: 119.36, allocs: '575.3K', multiplier: 2.14 },
  { n: 500, ms: 1009.80, mb: 237.76, allocs: '1.15M', multiplier: 1.65 },
]

export const INPROC_TEMPLATE_GENERIC = [
  { n: 2, ms: 3.34, mb: 2.23, allocs: '6.0K' },
  { n: 5, ms: 8.27, mb: 3.68, allocs: '13.2K' },
  { n: 10, ms: 14.93, mb: 6.23, allocs: '25.1K' },
  { n: 20, ms: 33.80, mb: 11.02, allocs: '49.0K' },
  { n: 50, ms: 80.62, mb: 25.56, allocs: '120.9K' },
  { n: 100, ms: 182.06, mb: 49.95, allocs: '240.8K' },
  { n: 200, ms: 374.28, mb: 98.79, allocs: '480.8K' },
  { n: 250, ms: 497.18, mb: 121.91, allocs: '600.8K' },
  { n: 500, ms: 1033.27, mb: 242.68, allocs: '1.20M' },
]

export const INPROC_WEB_FETCH = [
  { n: 2, ms: 11.43, mb: 7.46, allocs: '2.0K' },
  { n: 5, ms: 11.97, mb: 7.75, allocs: '2.2K' },
  { n: 10, ms: 13.43, mb: 8.35, allocs: '2.7K' },
  { n: 20, ms: 16.92, mb: 10.07, allocs: '3.1K' },
  { n: 50, ms: 26.91, mb: 15.20, allocs: '4.5K' },
  { n: 100, ms: 43.67, mb: 23.80, allocs: '6.9K' },
  { n: 200, ms: 78.72, mb: 41.01, allocs: '11.5K' },
  { n: 250, ms: 90.66, mb: 49.44, allocs: '13.9K' },
  { n: 500, ms: 171.25, mb: 92.48, allocs: '25.5K' },
]

export const INPROC_INLINE = [
  { n: 2, ms: 16.01, mb: 13.96, allocs: '2.1K' },
  { n: 5, ms: 10.20, mb: 7.26, allocs: '1.7K' },
  { n: 10, ms: 11.07, mb: 7.87, allocs: '2.1K' },
  { n: 20, ms: 13.56, mb: 9.61, allocs: '2.5K' },
  { n: 50, ms: 25.47, mb: 14.75, allocs: '3.9K' },
  { n: 100, ms: 42.97, mb: 23.40, allocs: '6.3K' },
  { n: 200, ms: 71.34, mb: 40.71, allocs: '11.0K' },
  { n: 250, ms: 91.51, mb: 49.21, allocs: '13.3K' },
  { n: 500, ms: 174.12, mb: 92.45, allocs: '24.9K' },
]

export const LIBRARY_PDF = [
  { n: 2, ms: 3.77, mb: 1.48, allocs: '5.7K', multiplier: 68.73 },
  { n: 5, ms: 8.33, mb: 3.06, allocs: '12.6K', multiplier: 32.19 },
  { n: 10, ms: 16.03, mb: 5.38, allocs: '24.1K', multiplier: 17.22 },
  { n: 20, ms: 31.19, mb: 10.31, allocs: '46.9K', multiplier: 10.16 },
  { n: 50, ms: 74.83, mb: 24.83, allocs: '115.8K', multiplier: 5.43 },
  { n: 100, ms: 160.94, mb: 48.46, allocs: '230.5K', multiplier: 3.27 },
  { n: 200, ms: 337.38, mb: 95.63, allocs: '460.3K', multiplier: 2.40 },
  { n: 250, ms: 441.40, mb: 118.89, allocs: '575.3K', multiplier: 2.18 },
  { n: 500, ms: 1104.51, mb: 236.85, allocs: '1.15M', multiplier: 1.51 },
]

export const LIBRARY_IMAGE = [
  { n: 2, ms: 11.07, mb: 4.17, allocs: '444' },
  { n: 5, ms: 13.43, mb: 6.13, allocs: '647' },
  { n: 10, ms: 14.61, mb: 6.81, allocs: '1.0K' },
  { n: 20, ms: 14.67, mb: 6.10, allocs: '1.4K' },
  { n: 50, ms: 17.42, mb: 7.32, allocs: '2.4K' },
  { n: 100, ms: 30.11, mb: 8.26, allocs: '4.1K' },
  { n: 200, ms: 63.03, mb: 19.35, allocs: '7.5K' },
  { n: 250, ms: 73.06, mb: 20.66, allocs: '9.2K' },
  { n: 500, ms: 142.59, mb: 52.00, allocs: '17.6K' },
]

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
  rssCrossover: 100,
}

export const LIBRARY_HEADLINE = {
  pages: LIBRARY_PDF[0].n,
  ms: LIBRARY_PDF[0].ms,
  wkMs: CLI_ROWS[0].wkMs,
  multiplier: relativeMultiplier(LIBRARY_PDF[0]),
  displayMultiplier: Math.round(relativeMultiplier(LIBRARY_PDF[0]) / 10) * 10,
}
