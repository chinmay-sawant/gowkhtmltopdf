export const SNAPSHOT = {
  date: '2026-08-14',
  host: 'Linux amd64, 13th Gen Intel Core i7-13700HX (WSL2, 24 CPUs)',
  go: 'go1.26.4',
  gowk: 'gowkhtmltopdf 0.2.0, freshly built generic CLI',
  wkhtml: 'wkhtmltopdf 0.12.6.1 (with patched qt)',
  flags: '--quiet --allow-local-files -o OUTPUT INPUT',
  method: 'median of 3 timed process runs after 1 warmup',
  fixture: 'report.html.tmpl, 20 invoice rows per requested page',
}

export const CLI_ROWS = [
  { pages: 2, gowkMs: 16, wkMs: 254, gowkRss: 24192, wkRss: 44852, gowkBytes: 42501, wkBytes: 18486 },
  { pages: 5, gowkMs: 22, wkMs: 265, gowkRss: 24768, wkRss: 45396, gowkBytes: 50919, wkBytes: 30584 },
  { pages: 10, gowkMs: 30, wkMs: 278, gowkRss: 26496, wkRss: 46200, gowkBytes: 65072, wkBytes: 50994 },
  { pages: 20, gowkMs: 44, wkMs: 304, gowkRss: 29760, wkRss: 47156, gowkBytes: 91899, wkBytes: 90742 },
  { pages: 50, gowkMs: 88, wkMs: 387, gowkRss: 41472, wkRss: 51824, gowkBytes: 172926, wkBytes: 210678 },
  { pages: 100, gowkMs: 184, wkMs: 530, gowkRss: 58752, wkRss: 58976, gowkBytes: 308714, wkBytes: 411260 },
  { pages: 200, gowkMs: 353, wkMs: 812, gowkRss: 90048, wkRss: 74336, gowkBytes: 579862, wkBytes: 816285 },
  { pages: 250, gowkMs: 433, wkMs: 942, gowkRss: 112704, wkRss: 81636, gowkBytes: 715476, wkBytes: 1019315 },
  { pages: 500, gowkMs: 1045, wkMs: 1641, gowkRss: 199872, wkRss: 123264, gowkBytes: 1398479, wkBytes: 2036776 },
]

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
  return row.wkMs / row.gowkMs
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
