import { Link } from 'react-router-dom'
import PageTitle from '../components/PageTitle'
import {
  CHART_PAGES,
  CLI_ROWS,
  HEADLINE,
  INPROC_INLINE,
  INPROC_PDF_GENERIC,
  INPROC_TEMPLATE_GENERIC,
  INPROC_WEB_FETCH,
  SNAPSHOT,
  formatKiB,
  formatMs,
  formatRssDelta,
  formatSpeedup,
  rssDelta,
  speedup,
} from '../data/benchmarks'

const MAX_MS = Math.max(...CLI_ROWS.map((r) => r.wkMs))
const CHART_ROWS = CLI_ROWS.filter((r) => CHART_PAGES.includes(r.pages))

function barWidth(ms) {
  return `${Math.max(3, (ms / MAX_MS) * 100)}%`
}

function CompareChart() {
  return (
    <div className="bench-chart" role="img" aria-label="Wall-time comparison of gowkhtmltopdf and wkhtmltopdf">
      {CHART_ROWS.map((row) => (
        <article className="bench-pair" key={row.pages}>
          <h3>{row.pages} pages</h3>
          <div className="bench-bars">
            <div className="bench-bar-row">
              <span className="bench-engine">gowk</span>
              <div className="bench-bar-track">
                <div className="bench-bar bench-bar-gowk" style={{ width: barWidth(row.gowkMs) }} />
              </div>
              <span className="bench-bar-time">{formatMs(row.gowkMs)}</span>
            </div>
            <div className="bench-bar-row">
              <span className="bench-engine">wkhtml</span>
              <div className="bench-bar-track">
                <div className="bench-bar bench-bar-wk" style={{ width: barWidth(row.wkMs) }} />
              </div>
              <span className="bench-bar-time">{formatMs(row.wkMs)}</span>
            </div>
          </div>
          <p className="bench-pair-note">
            <strong>{formatSpeedup(speedup(row))}</strong> faster
          </p>
        </article>
      ))}
    </div>
  )
}

function formatPdfSize(n) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)} MB`
  return `${(n / 1000).toFixed(1)} KB`
}

function rssTone(row) {
  const delta = rssDelta(row)
  if (Math.abs(delta) < 0.02) return 'even'
  return delta > 0 ? 'better' : 'worse'
}

function CompareTable() {
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
          {CLI_ROWS.map((row) => (
            <tr key={row.pages}>
              <th scope="row">{row.pages}</th>
              <td>{formatMs(row.gowkMs)}</td>
              <td>{formatMs(row.wkMs)}</td>
              <td>
                <span className="bench-speedup">{formatSpeedup(speedup(row))}</span>
              </td>
              <td>{formatKiB(row.gowkRss)}</td>
              <td>{formatKiB(row.wkRss)}</td>
              <td>
                <span className={`bench-rss bench-rss-${rssTone(row)}`}>{formatRssDelta(row)}</span>
              </td>
              <td>{formatPdfSize(row.gowkBytes)}</td>
              <td>{formatPdfSize(row.wkBytes)}</td>
            </tr>
          ))}
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

export default function BenchmarksPage() {
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
            Faster at every tested size. The largest gap is on short documents, where WebKit pays
            a cold-start tax that the in-process Go engine does not.
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
        </div>
      </section>

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

      <section className="bench-section" aria-labelledby="bench-chart-heading">
        <div className="section-heading-row">
          <h2 id="bench-chart-heading">Direct process comparison</h2>
          <p className="section-aside">
            Same HTML, same flags (<code>{SNAPSHOT.flags}</code>), median of three timed runs after
            one warmup. Peak RSS is from <code>/usr/bin/time %M</code>.
          </p>
        </div>
        <CompareChart />
      </section>

      <section className="bench-section" aria-labelledby="bench-table-heading">
        <h2 id="bench-table-heading">Full CLI matrix</h2>
        <p className="lede">
          Requested page counts match rendered page counts. PDF byte counts are the last timed
          output. Memory is peak RSS, not Go <code>B/op</code>.
        </p>
        <CompareTable />
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
          <h2 id="bench-inproc-heading">In-process Go benchmarks</h2>
          <p className="section-aside">
            <code>go test -bench</code> inside the test process. <code>B/op</code> is cumulative
            allocation traffic, not peak RSS.
          </p>
        </div>
        <InprocTable heading="PDF pages (generic request)" rows={INPROC_PDF_GENERIC} unit="Pages" />
        <InprocTable
          heading="Template + PDF pages (generic request)"
          rows={INPROC_TEMPLATE_GENERIC}
          unit="Pages"
        />
        <InprocTable heading="Web-fetch image tiles" rows={INPROC_WEB_FETCH} unit="Tiles" />
        <InprocTable heading="Inline image tiles" rows={INPROC_INLINE} unit="Tiles" />
      </section>

      <section className="bench-section bench-method" aria-labelledby="bench-method-heading">
        <h2 id="bench-method-heading">How this was measured</h2>
        <ul>
          <li>
            Host: {SNAPSHOT.host}. Toolchain: {SNAPSHOT.go}. Date: {SNAPSHOT.date}.
          </li>
          <li>
            {SNAPSHOT.gowk} versus {SNAPSHOT.wkhtml}.
          </li>
          <li>
            Fixture: {SNAPSHOT.fixture}. Method: {SNAPSHOT.method}.
          </li>
          <li>This is the generic convert path. Page islands are a benchmark-only opt-in.</li>
          <li>Numbers are a labeled snapshot, not an SLA. Reproduce on your machine.</li>
        </ul>
        <pre>
          <code>{`make bench-cli-compare
make bench`}</code>
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
