import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import PageTitle from '../components/PageTitle'
import {
  CHART_PAGES,
  CLI_ROWS,
  CURRENT_CAPTURE,
  externalSpeedup,
  HEADLINE,
  HISTORY_DATE,
  INPROC_INLINE_HISTORY,
  INPROC_PDF_GENERIC,
  INPROC_PDF_GENERIC_HISTORY,
  INPROC_TEMPLATE_GENERIC_HISTORY,
  INPROC_WEB_FETCH_HISTORY,
  LIBRARY_HEADLINE,
  LIBRARY_IMAGE,
  LIBRARY_IMAGE_HISTORY,
  LIBRARY_PDF,
  LIBRARY_PDF_HISTORY,
  PUPPETEER_ROWS,
  SNAPSHOT,
  WEASYPRINT_ROWS,
  formatKiB,
  formatMs,
  formatRssDelta,
  formatSpeedup,
  rssDelta,
  speedup,
} from '../data/benchmarks'

const WORKLOAD_FILTERS = [
  { id: 'all', label: 'All Workloads' },
  { id: '2', label: '2 Pages' },
  { id: '10', label: '10 Pages' },
  { id: '100', label: '100 Pages' },
  { id: '500', label: '500 Pages' },
]

const METRIC_VIEWS = [
  { id: 'time', label: 'Execution Time (ms)', shortLabel: 'Time (ms)', desc: 'Median process wall time (lower is faster)' },
  { id: 'speedup', label: 'Speedup Factor (X)', shortLabel: 'Speedup (X)', desc: 'gowk acceleration multiplier vs wkhtmltopdf baseline' },
  { id: 'memory', label: 'Memory RSS (MB)', shortLabel: 'Memory (MB)', desc: 'Peak process memory footprint' },
]

const ENGINE_SUMMARY_PAGES = [2, 10, 50, 100]

function formatPdfSize(n) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)} MB`
  return `${(n / 1000).toFixed(1)} KB`
}

function rssTone(row) {
  const delta = rssDelta(row)
  if (Math.abs(delta) < 0.02) return 'even'
  return delta > 0 ? 'better' : 'worse'
}

function formatMb(kib) {
  return `${(kib / 1024).toFixed(1)} MB`
}

function CompareChart({ rows, metricView }) {
  const maxMs = useMemo(() => Math.max(...rows.map((r) => r.wkMs), 1), [rows])
  const maxSpeedup = useMemo(
    () =>
      Math.max(
        ...rows.map((r) => speedup(r)),
        ...rows.map((r) => {
          const libraryRow = LIBRARY_PDF.find((item) => item.n === r.pages)
          return libraryRow ? r.wkMs / libraryRow.ms : 0
        }),
        1,
      ),
    [rows],
  )
  const maxRss = useMemo(() => Math.max(...rows.map((r) => Math.max(r.gowkRss, r.wkRss)), 1), [rows])

  return (
    <div
      className="bench-chart"
      role="region"
      aria-label={`Visual comparison of ${metricView} across workloads`}
    >
      {rows.map((row) => {
        const speed = speedup(row)
        const libraryRow = LIBRARY_PDF.find((item) => item.n === row.pages)
        const librarySpeed = libraryRow ? row.wkMs / libraryRow.ms : null
        const gowkRssMb = row.gowkRss / 1024
        const wkRssMb = row.wkRss / 1024

        let gowkWidth = '0%'
        let wkWidth = '0%'
        let libWidth = null
        let gowkLabel = ''
        let wkLabel = ''
        let libLabel = ''
        let note = null

        if (metricView === 'time') {
          gowkWidth = `${Math.max(4, (row.gowkMs / maxMs) * 100)}%`
          wkWidth = `${Math.max(4, (row.wkMs / maxMs) * 100)}%`
          gowkLabel = formatMs(row.gowkMs)
          wkLabel = formatMs(row.wkMs)
          if (libraryRow) {
            libWidth = `${Math.max(4, (libraryRow.ms / maxMs) * 100)}%`
            libLabel = formatMs(libraryRow.ms)
          }
          note = (
            <p className="bench-pair-note">
              <strong>{formatSpeedup(speed)}</strong> faster CLI
              {librarySpeed !== null && (
                <>
                  {' ·'}
                  <br />
                  <strong>{formatSpeedup(librarySpeed)}</strong> faster Go library
                </>
              )}
            </p>
          )
        } else if (metricView === 'speedup') {
          gowkWidth = `${Math.max(6, (speed / maxSpeedup) * 100)}%`
          wkWidth = `${Math.max(6, (1.0 / maxSpeedup) * 100)}%`
          gowkLabel = `${formatSpeedup(speed)}`
          if (librarySpeed !== null) {
            libWidth = `${Math.max(6, (librarySpeed / maxSpeedup) * 100)}%`
            libLabel = formatSpeedup(librarySpeed)
          }
          wkLabel = '1.00x baseline'
          note = (
            <p className="bench-pair-note">
              gowk renders in <strong>{(100 / speed).toFixed(0)}%</strong> of baseline time
            </p>
          )
        } else if (metricView === 'memory') {
          gowkWidth = `${Math.max(4, (row.gowkRss / maxRss) * 100)}%`
          wkWidth = `${Math.max(4, (row.wkRss / maxRss) * 100)}%`
          gowkLabel = `${gowkRssMb.toFixed(1)} MB`
          wkLabel = `${wkRssMb.toFixed(1)} MB`
          note = (
            <p className="bench-pair-note">
              <strong className={rssTone(row) === 'better' ? 'bench-text-better' : 'bench-text-worse'}>
                {formatRssDelta(row)}
              </strong>
            </p>
          )
        }

        return (
          <article className="bench-pair" key={row.pages}>
            <div className="bench-pair-head">
              <h3>{row.pages} pages</h3>
              <span className="bench-pair-badge">{formatPdfSize(row.gowkBytes)} PDF</span>
            </div>
            <div className="bench-bars">
              {libWidth !== null && (
                <div className="bench-bar-row">
                  <span className="bench-engine">gowk lib</span>
                  <div className="bench-bar-track">
                    <div
                      className="bench-bar bench-bar-lib"
                      style={{ width: libWidth }}
                      title={`gowkhtmltopdf Go library: ${libLabel}`}
                    />
                  </div>
                  <span className="bench-bar-time">{libLabel}</span>
                </div>
              )}
              <div className="bench-bar-row">
                <span className="bench-engine">gowk cli</span>
                <div className="bench-bar-track">
                  <div
                    className="bench-bar bench-bar-gowk"
                    style={{ width: gowkWidth }}
                    title={`gowkhtmltopdf CLI: ${gowkLabel}`}
                  />
                </div>
                <span className="bench-bar-time">{gowkLabel}</span>
              </div>
              <div className="bench-bar-row">
                <span className="bench-engine">wkhtml</span>
                <div className="bench-bar-track">
                  <div
                    className="bench-bar bench-bar-wk"
                    style={{ width: wkWidth }}
                    title={`wkhtmltopdf: ${wkLabel}`}
                  />
                </div>
                <span className="bench-bar-time">{wkLabel}</span>
              </div>
            </div>
            {note}
          </article>
        )
      })}
    </div>
  )
}

function SummaryCliTable({ activeFilter }) {
  const rows = CLI_ROWS.filter((row) => CHART_PAGES.includes(row.pages))

  return (
    <div className="table-scroll">
      <table>
        <thead>
          <tr>
            <th scope="col">Pages</th>
            <th scope="col">gowk cli</th>
            <th scope="col">gowk lib</th>
            <th scope="col">wkhtml</th>
            <th scope="col">Speedup</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const isMatch = activeFilter === 'all' || activeFilter === String(row.pages)
            const isDimmed = activeFilter !== 'all' && !isMatch
            const libraryRow = LIBRARY_PDF.find((item) => item.n === row.pages)
            return (
              <tr
                key={row.pages}
                className={`${isDimmed ? 'bench-row-dimmed' : ''} ${
                  isMatch && activeFilter !== 'all' ? 'bench-row-highlight' : ''
                }`}
              >
                <td>{row.pages}</td>
                <td>{formatMs(row.gowkMs)}</td>
                <td>{libraryRow ? formatMs(libraryRow.ms) : '-'}</td>
                <td>{formatMs(row.wkMs)}</td>
                <td>
                  <span className="bench-speedup">{formatSpeedup(speedup(row))}</span>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function EngineSummaryTable() {
  const engines = [
    { name: 'WeasyPrint', rows: WEASYPRINT_ROWS },
    { name: 'Puppeteer / Chrome', rows: PUPPETEER_ROWS },
  ]

  return (
    <div className="table-scroll">
      <table>
        <thead>
          <tr>
            <th scope="col">Engine</th>
            {ENGINE_SUMMARY_PAGES.map((pages) => (
              <th scope="col" key={pages}>
                {pages} pages
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {engines.map((engine) => (
            <tr key={engine.name}>
              <th scope="row">{engine.name}</th>
              {ENGINE_SUMMARY_PAGES.map((pages) => {
                const row = engine.rows.find((item) => item.pages === pages)
                return (
                  <td key={pages}>
                    {row ? (
                      <span className="bench-speedup">{formatSpeedup(externalSpeedup(row))}</span>
                    ) : (
                      '-'
                    )}
                  </td>
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function CompareTable({ activeFilter }) {
  return (
    <div className="table-scroll bench-matrix">
      <table>
        <thead>
          <tr>
            <th rowSpan={2} scope="col">
              Pages
            </th>
            <th colSpan={3} scope="colgroup">
              Wall time
            </th>
            <th colSpan={3} scope="colgroup">
              Peak RSS
            </th>
            <th colSpan={2} scope="colgroup">
              PDF size
            </th>
          </tr>
          <tr>
            <th scope="col">gowk</th>
            <th scope="col">wkhtml</th>
            <th scope="col">Speedup</th>
            <th scope="col">gowk</th>
            <th scope="col">wkhtml</th>
            <th scope="col">vs wkhtml</th>
            <th scope="col">gowk</th>
            <th scope="col">wkhtml</th>
          </tr>
        </thead>
        <tbody>
          {CLI_ROWS.map((row) => {
            const isMatch = activeFilter === 'all' || activeFilter === String(row.pages)
            const isDimmed = activeFilter !== 'all' && activeFilter !== String(row.pages)
            return (
              <tr
                key={row.pages}
                className={`${isMatch && activeFilter !== 'all' ? 'bench-row-highlight' : ''} ${
                  isDimmed ? 'bench-row-dimmed' : ''
                }`}
              >
                <th scope="row">
                  {row.pages}
                  {activeFilter === String(row.pages) && <span className="bench-row-pin"> •</span>}
                </th>
                <td>{formatMs(row.gowkMs)}</td>
                <td>{formatMs(row.wkMs)}</td>
                <td>
                  <span className="bench-speedup">{formatSpeedup(speedup(row))}</span>
                </td>
                <td>
                  {formatMb(row.gowkRss)} ({formatKiB(row.gowkRss)})
                </td>
                <td>
                  {formatMb(row.wkRss)} ({formatKiB(row.wkRss)})
                </td>
                <td>
                  <span className={`bench-rss bench-rss-${rssTone(row)}`}>{formatRssDelta(row)}</span>
                </td>
                <td>{formatPdfSize(row.gowkBytes)}</td>
                <td>{formatPdfSize(row.wkBytes)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function InprocTable({ heading, rows, unit }) {
  return (
    <section className="table-block">
      <h3 className="table-block-heading">{heading}</h3>
      <div className="table-scroll">
        <table>
          <thead>
            <tr>
              <th>{unit}</th>
              {rows.map((row) => (
                <th key={row.n}>{row.n}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>Time</td>
              {rows.map((row) => (
                <td key={row.n}>{formatMs(row.ms)}</td>
              ))}
            </tr>
            <tr>
              <td>B/op</td>
              {rows.map((row) => (
                <td key={row.n}>{row.mb} MB</td>
              ))}
            </tr>
            <tr>
              <td>allocs/op</td>
              {rows.map((row) => (
                <td key={row.n}>{row.allocs}</td>
              ))}
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  )
}

function HardwareGrid() {
  return (
    <div className="bench-spec-grid">
      <div className="bench-spec-item">
        <span className="bench-spec-label">Host Processor</span>
        <span className="bench-spec-value">13th Gen Intel Core i7-13700HX (24 CPUs, WSL2)</span>
      </div>
      <div className="bench-spec-item">
        <span className="bench-spec-label">Operating System</span>
        <span className="bench-spec-value">Linux 6.x Kernel (WSL2 / Debian GNU/Linux 12, glibc 2.36)</span>
      </div>
      <div className="bench-spec-item">
        <span className="bench-spec-label">gowkhtmltopdf Engine</span>
        <span className="bench-spec-value">
          <code>CGO_ENABLED=0</code> Pure-Go generic binary (VERSION 0.2.5, go1.26.4), zero native C
          bindings
        </span>
      </div>
      <div className="bench-spec-item">
        <span className="bench-spec-label">wkhtmltopdf Baseline</span>
        <span className="bench-spec-value">
          <code>wkhtmltopdf 0.12.6.1</code> (patched Qt 4.8.7 WebKit, fontconfig, freetype2)
        </span>
      </div>
      <div className="bench-spec-item">
        <span className="bench-spec-label">Execution Flags</span>
        <span className="bench-spec-value">
          <code>{SNAPSHOT.flags}</code>
        </span>
      </div>
      <div className="bench-spec-item">
        <span className="bench-spec-label">Measurement Method</span>
        <span className="bench-spec-value">
          CLI: 1 discard warmup + median of 3 timed runs via <code>/usr/bin/time %M</code>.
          In-process: median of three fresh <code>1x</code> processes.
        </span>
      </div>
      <div className="bench-spec-item">
        <span className="bench-spec-label">Benchmark Fixture</span>
        <span className="bench-spec-value">
          <code>{SNAPSHOT.fixture}</code>
        </span>
      </div>
      <div className="bench-spec-item">
        <span className="bench-spec-label">Memory Baseline</span>
        <span className="bench-spec-value">Peak Resident Set Size (RSS) from OS process supervisor</span>
      </div>
    </div>
  )
}

export default function BenchmarksPage() {
  const [activeFilter, setActiveFilter] = useState('all')
  const [metricView, setMetricView] = useState('time')

  // Filtered rows for the chart
  const displayedChartRows = useMemo(() => {
    if (activeFilter === 'all') {
      return CLI_ROWS.filter((r) => CHART_PAGES.includes(r.pages))
    }
    const match = CLI_ROWS.find((r) => String(r.pages) === activeFilter)
    return match ? [match] : CLI_ROWS.filter((r) => CHART_PAGES.includes(r.pages))
  }, [activeFilter])

  const activeMetricObj = METRIC_VIEWS.find((m) => m.id === metricView) || METRIC_VIEWS[0]

  return (
    <>
      <PageTitle title="Benchmarks" />
      <section className="bench-hero" aria-labelledby="bench-title">
        <div>
          <p className="bench-kicker">Snapshot {SNAPSHOT.date} · generic CLI vs wkhtmltopdf</p>
          <h1 id="bench-title">
            Up to <em>{HEADLINE.smallSpeedup.toFixed(0)}x faster</em>
            <br />
            than wkhtmltopdf.
          </h1>
          <p className="lede">
            The current generic <code>gowkhtmltopdf</code> binary was timed against the installed
            wkhtmltopdf {SNAPSHOT.wkhtml.replace('wkhtmltopdf ', '')} on the same report fixture.
            It was faster and used less peak RSS at every tested size, 2 through 500 pages. The
            public Go library removes the process boundary altogether.
          </p>
        </div>
        <div className="bench-hero-stats" aria-label="Headline comparison">
          <div>
            <strong>{HEADLINE.smallSpeedup.toFixed(1)}x</strong>
            <span>
              {HEADLINE.smallPages} pages · {formatMs(HEADLINE.smallGowk)} vs {formatMs(HEADLINE.smallWk)}
            </span>
          </div>
          <div>
            <strong>{HEADLINE.largeSpeedup.toFixed(2)}x</strong>
            <span>
              {HEADLINE.largePages} pages · {formatMs(HEADLINE.largeGowk)} vs {formatMs(HEADLINE.largeWk)}
            </span>
          </div>
          <div>
            <strong>~{LIBRARY_HEADLINE.displayMultiplier}x</strong>
            <span>2 pages · public Go library vs wkhtmltopdf CLI</span>
          </div>
        </div>
      </section>

      <section className="bench-section" aria-labelledby="bench-chart-heading">
        <div className="section-heading-row">
          <div>
            <h2 id="bench-chart-heading">Direct process comparison</h2>
            <p className="section-aside bench-explanation">
              Same HTML, same flags (<code>{SNAPSHOT.flags}</code>), median of three timed runs after
              one warmup. {activeMetricObj.desc}. <code>gowk lib</code> is the {CURRENT_CAPTURE.date}
              {' '}in-process capture (median of three fresh <code>1x</code> processes), not process
              wall time.
            </p>
          </div>
        </div>

        <div className="bench-toolbar" role="toolbar" aria-label="Benchmark view controls">
          <div className="bench-control-group">
            <span className="bench-control-label" id="filter-workload-label">
              Workload Filter:
            </span>
            <div className="bench-tabs" role="tablist" aria-labelledby="filter-workload-label">
              {WORKLOAD_FILTERS.map((wf) => (
                <button
                  key={wf.id}
                  type="button"
                  role="tab"
                  aria-selected={activeFilter === wf.id}
                  className={`bench-tab-btn ${activeFilter === wf.id ? 'active' : ''}`}
                  onClick={() => setActiveFilter(wf.id)}
                >
                  {wf.label}
                </button>
              ))}
            </div>
          </div>

          <div className="bench-control-group">
            <span className="bench-control-label" id="metric-view-label">
              Metric View:
            </span>
            <div
              className="bench-tabs bench-metric-tabs"
              role="radiogroup"
              aria-labelledby="metric-view-label"
            >
              {METRIC_VIEWS.map((mv) => (
                <button
                  key={mv.id}
                  type="button"
                  role="radio"
                  aria-checked={metricView === mv.id}
                  className={`bench-tab-btn ${metricView === mv.id ? 'active' : ''}`}
                  onClick={() => setMetricView(mv.id)}
                >
                  {mv.label}
                </button>
              ))}
            </div>
          </div>
        </div>

        <CompareChart rows={displayedChartRows} metricView={metricView} />

        <section className="table-block">
          <h3 className="table-block-heading">Exact numbers</h3>
          <SummaryCliTable activeFilter={activeFilter} />
        </section>

        <details className="bench-details">
          <summary>Full matrix: 2 to 500 pages with RSS and PDF sizes</summary>
          <div className="bench-details-body">
            <CompareTable activeFilter={activeFilter} />
          </div>
        </details>
      </section>

      <section className="bench-section" aria-labelledby="bench-external-heading">
        <div className="section-heading-row">
          <div>
            <h2 id="bench-external-heading">Against other engines</h2>
            <p className="lede">
              The same report fixture was printed through WeasyPrint and Puppeteer/Chrome. Cells are
              the gowkhtmltopdf speedup at that page count.
            </p>
          </div>
        </div>
        <section className="table-block">
          <h3 className="table-block-heading">Speedup vs each engine</h3>
          <EngineSummaryTable />
        </section>
        <p className="section-aside bench-explanation">
          Puppeteer RSS is the peak process-tree reading for Node plus headless Chrome, not a
          single-process <code>%M</code> value. Full time, RSS, and PDF rows for both engines live in
          the performance notes.
        </p>
      </section>

      <aside className="callout callout-info bench-callout" role="note">
        <div className="callout-marker" aria-hidden="true">
          i
        </div>
        <div className="callout-body">
          <span className="callout-kicker">How to read memory</span>
          <h3 className="callout-title">Faster and lighter at every size.</h3>
          <p>
            On this generic CLI path, gowkhtmltopdf beats wkhtmltopdf on wall time and uses less
            peak RSS at every tested size, including 500 pages (79,296 KiB vs 123,076 KiB). The
            500-page PDF is also smaller (1.42 MB vs 2.04 MB). Earlier captures that showed higher
            gowk RSS from 100 pages on are historical and do not describe the current converter.
          </p>
        </div>
      </aside>

      <section className="bench-section bench-method" aria-labelledby="bench-method-heading">
        <h2 id="bench-method-heading">How this was measured</h2>
        <p>
          Snapshot {SNAPSHOT.date}: {SNAPSHOT.gowk} against {SNAPSHOT.wkhtml} on {SNAPSHOT.host}.
          Fixture: <code>{SNAPSHOT.fixture}</code>. Method: {SNAPSHOT.method}. Numbers are a labeled
          snapshot, not an SLA.
        </p>

        <details className="bench-details">
          <summary>Environment details and reproduce commands</summary>
          <div className="bench-details-body">
            <HardwareGrid />
            <pre>
              <code>{`make bench
make bench-engine
make bench-lib`}</code>
            </pre>
          </div>
        </details>

        <details className="bench-details">
          <summary>
            In-process engine and Go library ({CURRENT_CAPTURE.date} capture) with B/op and
            allocs/op
          </summary>
          <div className="bench-details-body">
            <p className="section-aside">
              Three independent <code>1x</code> samples per workload, one fresh process each; the
              median is shown and <code>B/op</code> is never averaged. <code>B/op</code> is
              cumulative allocation traffic, not peak RSS, and the 2-page rows include the one-time
              default-font work.
            </p>
            <InprocTable heading="Internal engine PDF pages" rows={INPROC_PDF_GENERIC} unit="Pages" />
            <InprocTable
              heading="Public PDF pages (Document.WritePDF)"
              rows={LIBRARY_PDF}
              unit="Pages"
            />
            <InprocTable
              heading="Public image tiles (ImageDocument.WriteImage)"
              rows={LIBRARY_IMAGE}
              unit="Tiles"
            />
            <p className="section-aside">
              The image rows trade about 50 percent more lossless PNG bytes for roughly 2x faster
              encoding. Raw samples: <code>{CURRENT_CAPTURE.raw}</code>.
            </p>
          </div>
        </details>

        <details className="bench-details">
          <summary>Historical snapshots ({HISTORY_DATE}, 0.2.4)</summary>
          <div className="bench-details-body">
            <p className="section-aside">
              Dated rows kept for comparison. They do not describe the current generic converter.
            </p>
            <InprocTable heading="Internal PDF pages" rows={INPROC_PDF_GENERIC_HISTORY} unit="Pages" />
            <InprocTable
              heading="Template + PDF pages"
              rows={INPROC_TEMPLATE_GENERIC_HISTORY}
              unit="Pages"
            />
            <InprocTable heading="Web-fetch image tiles" rows={INPROC_WEB_FETCH_HISTORY} unit="Tiles" />
            <InprocTable heading="Inline image tiles" rows={INPROC_INLINE_HISTORY} unit="Tiles" />
            <InprocTable heading="Public PDF pages" rows={LIBRARY_PDF_HISTORY} unit="Pages" />
            <InprocTable heading="Public image tiles" rows={LIBRARY_IMAGE_HISTORY} unit="Tiles" />
          </div>
        </details>

        <p>
          Full tables, RSS details, and historical captures live in the{' '}
          <Link to="/documentation/performance">performance notes</Link>. The consolidated current
          capture is <code>documentation/benchmarks.md</code>; the machine-written artifacts are{' '}
          <code>testdata/golden/benchmarks/*-compare.md</code>.
        </p>
      </section>
    </>
  )
}
