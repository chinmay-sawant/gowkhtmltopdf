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

export const INPROC_SNAPSHOT_DATE = '2026-08-14'

export const CHART_PAGES = [2, 10, 50, 100, 500]

export const INPROC_PDF_GENERIC = [
  { n: 2, ms: 3.66, mb: 2.26, allocs: '5.8K' },
  { n: 5, ms: 7.06, mb: 3.69, allocs: '12.7K' },
  { n: 10, ms: 15.0, mb: 5.92, allocs: '24.1K' },
  { n: 20, ms: 28.3, mb: 10.3, allocs: '46.9K' },
  { n: 50, ms: 77.1, mb: 23.7, allocs: '115.6K' },
  { n: 100, ms: 149, mb: 46.1, allocs: '230.2K' },
  { n: 200, ms: 336, mb: 91.0, allocs: '459.8K' },
  { n: 250, ms: 417, mb: 112.7, allocs: '574.6K' },
  { n: 500, ms: 966, mb: 224.3, allocs: '1.15M' },
]

export const INPROC_TEMPLATE_GENERIC = [
  { n: 2, ms: 3.63, mb: 2.28, allocs: '6.0K' },
  { n: 5, ms: 7.56, mb: 2.93, allocs: '13.1K' },
  { n: 10, ms: 15.4, mb: 6.07, allocs: '25.1K' },
  { n: 20, ms: 31.1, mb: 10.6, allocs: '48.9K' },
  { n: 50, ms: 77.6, mb: 24.3, allocs: '120.7K' },
  { n: 100, ms: 165, mb: 47.3, allocs: '240.5K' },
  { n: 200, ms: 336, mb: 93.2, allocs: '480.2K' },
  { n: 250, ms: 421, mb: 115.0, allocs: '600.1K' },
  { n: 500, ms: 948, mb: 228.9, allocs: '1.20M' },
]

export const INPROC_WEB_FETCH = [
  { n: 2, ms: 10.7, mb: 7.45, allocs: '2.0K' },
  { n: 5, ms: 12.5, mb: 7.73, allocs: '2.2K' },
  { n: 10, ms: 13.8, mb: 8.34, allocs: '2.6K' },
  { n: 20, ms: 16.0, mb: 10.0, allocs: '3.1K' },
  { n: 50, ms: 26.0, mb: 15.1, allocs: '4.5K' },
  { n: 100, ms: 42.6, mb: 23.7, allocs: '6.9K' },
  { n: 200, ms: 73.2, mb: 40.7, allocs: '11.5K' },
  { n: 250, ms: 84.8, mb: 49.1, allocs: '13.9K' },
  { n: 500, ms: 166, mb: 91.8, allocs: '25.5K' },
]

export const INPROC_INLINE = [
  { n: 2, ms: 14.6, mb: 14.0, allocs: '2.1K' },
  { n: 5, ms: 10.2, mb: 7.25, allocs: '1.7K' },
  { n: 10, ms: 10.1, mb: 7.86, allocs: '2.1K' },
  { n: 20, ms: 13.2, mb: 9.58, allocs: '2.5K' },
  { n: 50, ms: 21.6, mb: 14.7, allocs: '3.9K' },
  { n: 100, ms: 38.6, mb: 23.3, allocs: '6.3K' },
  { n: 200, ms: 67.7, mb: 40.4, allocs: '11.0K' },
  { n: 250, ms: 87.1, mb: 48.9, allocs: '13.3K' },
  { n: 500, ms: 178, mb: 91.7, allocs: '25.0K' },
]

export function speedup(row) {
  return row.speedup ?? row.wkMs / row.gowkMs
}

export function externalSpeedup(row) {
  return row.speedup ?? row.engineMs / row.gowkMs
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
