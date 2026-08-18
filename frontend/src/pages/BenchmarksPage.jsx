import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import PageTitle from '../components/PageTitle'
import Footer from '../components/Footer'
import {
  CHART_PAGES,
  CLI_ROWS,
  externalSpeedup,
  HEADLINE,
  LIBRARY_HEADLINE,
  LIBRARY_IMAGE,
  LIBRARY_PDF,
  INPROC_SNAPSHOT_DATE,
  INPROC_INLINE,
  INPROC_PDF_GENERIC,
  INPROC_TEMPLATE_GENERIC,
  INPROC_WEB_FETCH,
  PUPPETEER_ROWS,
  SNAPSHOT,
  WEASYPRINT_ROWS,
  formatKiB,
  formatMs,
  formatRssDelta,
  formatSpeedup,
  relativeMultiplier,
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
  const maxSpeedup = useMemo(() => Math.max(...rows.map((r) => speedup(r)), 1), [rows])
  const maxRss = useMemo(() => Math.max(...rows.map((r) => Math.max(r.gowkRss, r.wkRss)), 1), [rows])

  return (
    <div
      className="bench-chart"
      role="region"
      aria-label={`Visual comparison of ${metricView} across workloads`}
    >
      {rows.map((row) => {
        const speed = speedup(row)
        const gowkRssMb = row.gowkRss / 1024
        const wkRssMb = row.wkRss / 1024

        let gowkWidth = '0%'
        let wkWidth = '0%'
        let gowkLabel = ''
        let wkLabel = ''
        let note = null

        if (metricView === 'time') {
          gowkWidth = `${Math.max(4, (row.gowkMs / maxMs) * 100)}%`
          wkWidth = `${Math.max(4, (row.wkMs / maxMs) * 100)}%`
          gowkLabel = formatMs(row.gowkMs)
          wkLabel = formatMs(row.wkMs)
          note = (
            <p className="bench-pair-note">
              <strong>{formatSpeedup(speed)}</strong> faster
            </p>
          )
        } else if (metricView === 'speedup') {
          gowkWidth = `${Math.max(6, (speed / maxSpeedup) * 100)}%`
          wkWidth = `${Math.max(6, (1.0 / maxSpeedup) * 100)}%`
          gowkLabel = `${formatSpeedup(speed)}`
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
              <div className="bench-bar-row">
                <span className="bench-engine">gowk</span>
                <div className="bench-bar-track">
                  <div
                    className="bench-bar bench-bar-gowk"
                    style={{ width: gowkWidth }}
                    title={`gowkhtmltopdf: ${gowkLabel}`}
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
                <td>{formatMb(row.gowkRss)} ({formatKiB(row.gowkRss)})</td>
                <td>{formatMb(row.wkRss)} ({formatKiB(row.wkRss)})</td>
                <td>
                  <span className={`bench-rss bench-rss-${rssTone(row)}`}>
                    {formatRssDelta(row)}
                  </span>
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

function ExternalTable({ engine, rows, rssNote }) {
  return (
    <section className="table-block">
      <h3 className="table-block-heading">gowkhtmltopdf vs {engine}</h3>
      <div className="table-scroll">
        <table>
          <thead>
            <tr>
              <th scope="col">Pages</th>
              <th scope="col">gowk time</th>
              <th scope="col">{engine} time</th>
              <th scope="col">Speedup</th>
              <th scope="col">gowk RSS</th>
              <th scope="col">{engine} RSS</th>
              <th scope="col">gowk PDF</th>
              <th scope="col">{engine} PDF</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.pages}>
                <th scope="row">{row.pages}</th>
                <td>{formatMs(row.gowkMs)}</td>
                <td>{formatMs(row.engineMs)}</td>
                <td>
                  <span className="bench-speedup">{formatSpeedup(externalSpeedup(row))}</span>
                </td>
                <td>{formatKiB(row.gowkRss)}</td>
                <td>{formatKiB(row.engineRss)}</td>
                <td>{formatPdfSize(row.gowkBytes)}</td>
                <td>{formatPdfSize(row.engineBytes)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="section-aside">{rssNote}</p>
    </section>
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

function RelativeTimingTable({ heading, rows, pathLabel }) {
  return (
    <section className="table-block">
      <h3 className="table-block-heading">{heading}</h3>
      <p className="section-aside">
        Indicative ratio: <code>wkhtmltopdf CLI time / {pathLabel} time</code>. The CLI includes
        process startup and file handling; the Go path runs directly in the current process.
      </p>
      <div className="table-scroll">
        <table>
          <thead>
            <tr>
              <th scope="col">Pages</th>
              <th scope="col">wkhtmltopdf CLI</th>
              <th scope="col">{pathLabel}</th>
              <th scope="col">Indicative multiplier</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => {
              const baseline = CLI_ROWS.find((item) => item.pages === row.n)
              const multiplier = relativeMultiplier(row)

              return (
                <tr key={row.n}>
                  <th scope="row">{row.n}</th>
                  <td>{baseline ? formatMs(baseline.wkMs) : '—'}</td>
                  <td>{formatMs(row.ms)}</td>
                  <td>
                    <span className="bench-speedup">
                      {multiplier === null ? '—' : formatSpeedup(multiplier)}
                    </span>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </section>
  )
}

function HardwareSpecCard() {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <div className="bench-spec-card">
      <button
        type="button"
        className="bench-spec-header"
        onClick={() => setIsOpen(!isOpen)}
        aria-expanded={isOpen}
        aria-controls="bench-spec-details"
      >
        <div className="bench-spec-header-main">
          <div className="bench-spec-icon-badge">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <rect x="2" y="2" width="20" height="8" rx="2" ry="2" />
              <rect x="2" y="14" width="20" height="8" rx="2" ry="2" />
              <line x1="6" y1="6" x2="6.01" y2="6" />
              <line x1="6" y1="18" x2="6.01" y2="18" />
            </svg>
          </div>
          <div>
            <div className="bench-spec-title">Hardware & Test Environment Specification</div>
            <div className="bench-spec-subtitle">
              13th Gen Intel Core i7-13700HX · Linux WSL2 · cgo=0 (Pure Go) vs Qt WebKit 0.12.6.1
            </div>
          </div>
        </div>
        <div className="bench-spec-toggle">
          <span className="bench-spec-toggle-text">{isOpen ? 'Hide Specs' : 'View Specs'}</span>
          <svg
            className={`bench-spec-chevron ${isOpen ? 'open' : ''}`}
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
          >
            <polyline points="6 9 12 15 18 9" />
          </svg>
        </div>
      </button>

      {isOpen && (
        <div className="bench-spec-content" id="bench-spec-details">
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
                <code>CGO_ENABLED=0</code> Pure-Go generic binary (v0.2.4, go1.26.4), zero native C bindings
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
                1 discard warmup run + median of 3 measured iterations via <code>/usr/bin/time %M</code>
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
          <div className="bench-spec-footnote">
            <span>Snapshot Tag: {SNAPSHOT.date}</span>
            <span>·</span>
            <span>Reproduce locally with <code>make bench</code></span>
          </div>
        </div>
      )}
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
            It is faster at every tested size. The public Go library removes the process boundary
            altogether: its 2-page result is about 70x faster than the wkhtmltopdf CLI baseline.
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
            <strong>every size</strong>
            <span>2 through 500 pages, gowk was the faster process</span>
          </div>
          <div>
            <strong>~{LIBRARY_HEADLINE.displayMultiplier}x</strong>
            <span>2 pages · public Go library vs wkhtmltopdf CLI</span>
          </div>
        </div>
      </section>

      {/* Hardware & Test Environment Specification Card */}
      <HardwareSpecCard />

      <div className="statband">
        <div className="statband-item">
          <div className="statband-value">{formatMs(HEADLINE.smallGowk)}</div>
          <div className="statband-label">2-page report, gowk CLI</div>
        </div>
        <div className="statband-item">
          <div className="statband-value">{formatMs(HEADLINE.smallWk)}</div>
          <div className="statband-label">same report, wkhtmltopdf</div>
        </div>
        <div className="statband-item">
          <div className="statband-value">{formatMs(HEADLINE.largeGowk)}</div>
          <div className="statband-label">500-page report, gowk CLI</div>
        </div>
        <div className="statband-item">
          <div className="statband-value">{formatMs(HEADLINE.largeWk)}</div>
          <div className="statband-label">500-page report, wkhtmltopdf</div>
        </div>
      </div>

      {/* Direct Process Comparison Section with Interactive Controls */}
      <section className="bench-section" aria-labelledby="bench-chart-heading">
        <div className="section-heading-row">
          <div>
            <h2 id="bench-chart-heading">Direct process comparison</h2>
            <p className="section-aside">
              Same HTML, same flags (<code>{SNAPSHOT.flags}</code>), median of three timed runs after
              one warmup. {activeMetricObj.desc}.
            </p>
          </div>
        </div>

        {/* Interactive Filter & Metric Control Toolbar */}
        <div className="bench-toolbar" role="toolbar" aria-label="Benchmark view controls">
          {/* Workload Filter Tabs */}
          <div className="bench-control-group">
            <span className="bench-control-label" id="filter-workload-label">
              Workload Filter:
            </span>
            <div
              className="bench-tabs"
              role="tablist"
              aria-labelledby="filter-workload-label"
            >
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

          {/* Metric View Switcher */}
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
      </section>

      <section className="bench-section" aria-labelledby="bench-table-heading">
        <div className="section-heading-row">
          <div>
            <h2 id="bench-table-heading">Full CLI matrix</h2>
            <p className="lede">
              Requested page counts match rendered page counts. PDF byte counts are the last timed
              output. Memory is peak RSS, not Go <code>B/op</code>.
            </p>
          </div>
        </div>
        <CompareTable activeFilter={activeFilter} />
      </section>

      <section className="bench-section" aria-labelledby="bench-external-heading">
        <div className="section-heading-row">
          <div>
            <h2 id="bench-external-heading">External renderer comparisons</h2>
            <p className="lede">
              The same report fixture was printed through WeasyPrint and Puppeteer/Chrome. These
              matrices use the external harness&apos;s 2, 10, 50, and 100 page sizes.
            </p>
          </div>
        </div>
        <ExternalTable
          engine="WeasyPrint"
          rows={WEASYPRINT_ROWS}
          rssNote="WeasyPrint RSS is the measured process peak from /usr/bin/time %M."
        />
        <ExternalTable
          engine="Puppeteer / Chrome"
          rows={PUPPETEER_ROWS}
          rssNote="Puppeteer RSS is the peak process-tree RSS for the Node driver and headless Chrome descendants; it is not directly equivalent to a single-process %M reading."
        />
      </section>

      <aside className="callout callout-info" role="note">
        <div className="callout-marker" aria-hidden="true">
          i
        </div>
        <div className="callout-body">
          <span className="callout-kicker">How to read memory</span>
          <h3 className="callout-title">Faster at every size. Lower RSS only through 100 pages.</h3>
          <p>
            On this generic CLI path, gowkhtmltopdf uses less peak RSS from 2 through 100 pages and
            more RSS from 200 through 500 pages. The 500-page PDF is still smaller (1.40 MB vs 2.04
            MB). Earlier island-era snapshots that claimed lower RSS at every size are historical
            and do not describe the current generic converter.
          </p>
        </div>
      </aside>

      <section className="bench-section" aria-labelledby="bench-inproc-heading">
        <div className="section-heading-row">
          <h2 id="bench-inproc-heading">In-process Go benchmarks ({INPROC_SNAPSHOT_DATE})</h2>
          <p className="section-aside">
            <code>go test -bench</code> inside the test process. <code>B/op</code> is cumulative
            allocation traffic, not peak RSS.
          </p>
        </div>
        <InprocTable heading="PDF pages (generic request)" rows={INPROC_PDF_GENERIC} unit="Pages" />
        <RelativeTimingTable
          heading="In-process PDF multiplier vs wkhtmltopdf CLI"
          rows={INPROC_PDF_GENERIC}
          pathLabel="in-process Go PDF"
        />
        <InprocTable
          heading="Template + PDF pages (generic request)"
          rows={INPROC_TEMPLATE_GENERIC}
          unit="Pages"
        />
        <InprocTable heading="Web-fetch image tiles" rows={INPROC_WEB_FETCH} unit="Tiles" />
        <InprocTable heading="Inline image tiles" rows={INPROC_INLINE} unit="Tiles" />
      </section>

      <section className="bench-section" aria-labelledby="bench-library-heading">
        <div className="section-heading-row">
          <h2 id="bench-library-heading">Public Go library benchmarks ({INPROC_SNAPSHOT_DATE})</h2>
          <p className="section-aside">
            <code>make bench-lib</code> calls <code>Document.WritePDF</code> and{' '}
            <code>ImageDocument.WriteImage</code> directly, without starting the CLI or reading
            HTML from disk.
          </p>
        </div>
        <InprocTable heading="Public PDF pages" rows={LIBRARY_PDF} unit="Pages" />
        <RelativeTimingTable
          heading="Public library PDF multiplier vs wkhtmltopdf CLI"
          rows={LIBRARY_PDF}
          pathLabel="public Go library PDF"
        />
        <InprocTable heading="Public image tiles" rows={LIBRARY_IMAGE} unit="Tiles" />
      </section>

      <section className="bench-section bench-method" aria-labelledby="bench-method-heading">
        <h2 id="bench-method-heading">How this was measured</h2>
        <ul>
          <li>
            Host: {SNAPSHOT.host}. Toolchain: {SNAPSHOT.go}. Date: {SNAPSHOT.date}.
          </li>
          <li>
            {SNAPSHOT.gowk} versus {SNAPSHOT.wkhtml} for the CLI matrix. WeasyPrint and Puppeteer
            use the external print scripts documented in the benchmark README.
          </li>
          <li>
            Fixture: {SNAPSHOT.fixture}. Method: {SNAPSHOT.method}.
          </li>
          <li>This is the generic convert path. Page islands are a benchmark-only opt-in.</li>
          <li>Numbers are a labeled snapshot, not an SLA. Reproduce on your machine.</li>
        </ul>
        <pre>
          <code>{`make bench
make bench-engine
make bench-inprocess  # compatibility alias
make bench-lib`}</code>
        </pre>
        <p>
          Full tables, historical snapshots, and caveats live in the{' '}
          <Link to="/documentation/performance">performance notes</Link>. The source of truth is{' '}
          <code>testdata/golden/benchmarks/README.md</code> and{' '}
          <code>documentation/performance.md</code>.
        </p>
      </section>
    </>
  )
}
