import { Link } from 'react-router-dom'
import { CLI_ROWS, HEADLINE, formatMs, speedup } from '../data/benchmarks'

const HOME_BENCH_PAGES = [2, 10, 100, 500]
const HOME_BENCH = CLI_ROWS.filter((row) => HOME_BENCH_PAGES.includes(row.pages))

function homeSpeedup(n) {
  if (n >= 10) return `${n.toFixed(0)}x`
  return `${n.toFixed(1)}x`
}

const CAPABILITIES = [
  {
    index: '01',
    title: 'Reports that paginate',
    body: 'Tables, page breaks, headers, footers, TOCs, outlines, links, and multi-page documents are first-class output.',
    link: '/documentation/compatibility',
    label: 'See compatibility',
  },
  {
    index: '02',
    title: 'A Go-native workflow',
    body: 'Use the familiar command line or embed the converter as a Go library with context-aware conversion and typed settings.',
    link: '/getting-started',
    label: 'Start converting',
  },
  {
    index: '03',
    title: 'A smaller trust surface',
    body: 'No browser process, no JavaScript engine, and local file access is denied by default. Read the integration guidance before accepting untrusted URLs.',
    link: '/documentation/security',
    label: 'Read security notes',
  },
]

const OUTPUTS = [
  ['PDF', 'Multi-page documents with print-oriented layout'],
  ['PNG / JPEG', 'Raster output for previews and image workflows'],
  ['CLI + API', 'Drop-in command line ergonomics or native Go control'],
  ['CGO=0', 'Static, portable builds without a browser dependency'],
]

const COMMAND = [
  '$ gowkhtmltopdf \\',
  '  --page-size A4 \\',
  '  report.html report.pdf',
  '',
  'load -> parse -> style -> layout',
  'paginate -> paint -> write PDF 1.4',
  '',
  'wrote report.pdf',
].join('\n')

export default function LandingPage() {
  return (
    <div className="landing-page">
      <section className="landing-hero" aria-labelledby="landing-title">
        <div className="landing-hero-copy">
          <h1 id="landing-title">Print-ready documents,<br /><em>from HTML you control.</em></h1>
          <p className="landing-claim">
            Up to {HEADLINE.smallSpeedup.toFixed(0)}x faster than wkhtmltopdf.
          </p>
          <p className="landing-lede">
            gowkhtmltopdf turns controlled, server-generated HTML into dependable PDF and image output without starting a browser process.
          </p>
          <div className="landing-actions">
            <Link className="button button-primary" to="/getting-started">Get started <span aria-hidden="true">→</span></Link>
            <Link className="button button-secondary" to="/benchmarks">See the benchmarks</Link>
          </div>
          <p className="landing-note">Best for invoices, statements, tables, and multi-page reports. JavaScript is not executed.</p>
        </div>

        <div className="terminal-card" aria-label="Example command line conversion">
          <div className="terminal-topbar">
            <span className="terminal-lights" aria-hidden="true"><i /><i /><i /></span>
            <span>terminal</span>
            <span className="terminal-status"><span aria-hidden="true" /> ready</span>
          </div>
          <pre><code>{COMMAND}</code></pre>
          <div className="terminal-footer"><span>HTML to PDF</span><span>controlled report workflow</span></div>
        </div>
      </section>

      <section className="landing-proof" aria-label="Product highlights">
        <div className="proof-intro">
          <h2>One focused pipeline.</h2>
          <p>For teams that own the HTML and care about the output.</p>
        </div>
        <div className="proof-grid">
          {OUTPUTS.map(([value, label]) => (
            <div className="proof-item" key={value}>
              <strong>{value}</strong>
              <span>{label}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="landing-bench" aria-labelledby="landing-bench-heading">
        <div className="landing-bench-copy">
          <p className="landing-bench-kicker">Measured against wkhtmltopdf 0.12.6.1</p>
          <h2 id="landing-bench-heading">
            Up to {HEADLINE.smallSpeedup.toFixed(0)}x faster
            <br />
            on the same report.
          </h2>
          <p>
            Generic CLI, identical HTML, median of three process runs. A 2-page invoice is{' '}
            {formatMs(HEADLINE.smallGowk)} versus {formatMs(HEADLINE.smallWk)}; 500 pages still
            finish first.
          </p>
          <Link className="text-link" to="/benchmarks">
            Full benchmark comparison <span aria-hidden="true">→</span>
          </Link>
        </div>
        <div className="landing-bench-grid">
          {HOME_BENCH.map((row) => (
            <article className="landing-bench-card" key={row.pages}>
              <small>{row.pages} pages</small>
              <strong>{homeSpeedup(speedup(row))}</strong>
              <span>
                {formatMs(row.gowkMs)} vs {formatMs(row.wkMs)}
              </span>
            </article>
          ))}
        </div>
      </section>

      <section className="landing-section" aria-labelledby="capability-heading">
        <div className="section-heading-row">
          <h2 id="capability-heading">The document path,<br />made deliberate.</h2>
          <p className="section-aside">Designed for predictable print output, not for pretending to be a general-purpose browser.</p>
        </div>
        <div className="capability-grid">
          {CAPABILITIES.map((item) => (
            <article className="capability-card" key={item.index}>
              <span className="card-index">{item.index}</span>
              <h3>{item.title}</h3>
              <p>{item.body}</p>
              <Link to={item.link}>{item.label} <span aria-hidden="true">↗</span></Link>
            </article>
          ))}
        </div>
      </section>

      <section className="landing-section landing-showcase" aria-labelledby="showcase-heading">
        <div className="showcase-copy">
          <h2 id="showcase-heading">See the pages<br /><em>before you ship.</em></h2>
          <p>Explore committed samples generated from golden HTML fixtures: invoices, contracts, reports, CSS layout cases, and a five-page dossier.</p>
          <Link className="text-link" to="/showcase">Open the sample library <span aria-hidden="true">→</span></Link>
        </div>
        <div className="paper-stack" aria-hidden="true">
          <div className="paper paper-back"><span>invoice</span><i /></div>
          <div className="paper paper-middle"><span>report</span><i /><i /></div>
          <div className="paper paper-front"><small>gowkhtmltopdf</small><strong>PDF</strong><span>04 / 48</span></div>
        </div>
      </section>

      <section className="landing-boundary" aria-labelledby="boundary-heading">
        <div className="boundary-symbol" aria-hidden="true">!</div>
        <div>
          <h2 id="boundary-heading">This is a document renderer,<br />not a browser replacement.</h2>
          <p>There is no DOM-driven JavaScript runtime, CSSOM, or browser painting engine. For arbitrary SPA fidelity, use server-rendered HTML or a browser automation tool. For controlled reports, the smaller surface is the point.</p>
          <Link className="text-link" to="/documentation/compatibility">Read the compatibility guide <span aria-hidden="true">→</span></Link>
        </div>
      </section>

      <section className="landing-next" aria-labelledby="next-heading">
        <h2 id="next-heading">Start with the surface<br />you already use.</h2>
        <div className="next-links">
          <Link to="/getting-started"><span>01</span><strong>First conversion</strong><small>Build, run, and make your first PDF</small><b aria-hidden="true">→</b></Link>
          <Link to="/documentation/library-api"><span>02</span><strong>Go library</strong><small>Context-aware conversion and typed settings</small><b aria-hidden="true">→</b></Link>
          <Link to="/benchmarks"><span>03</span><strong>Benchmarks</strong><small>How much faster than wkhtmltopdf</small><b aria-hidden="true">→</b></Link>
          <Link to="/dossier"><span>04</span><strong>Issue dossier</strong><small>1329 upstream issues mapped to coverage</small><b aria-hidden="true">→</b></Link>
        </div>
      </section>
    </div>
  )
}
