import { useState } from 'react'
import { Link } from 'react-router-dom'
import PageTitle from '../components/PageTitle'

const CLI_CODE = `$ gowkhtmltopdf input.html output.pdf
# local files need explicit permission
$ gowkhtmltopdf --allow-local-files -o report.pdf report.html`

const GO_CODE = `doc := gowkhtmltopdf.Document{
  Pages: []gowkhtmltopdf.Page{{
    Source: gowkhtmltopdf.Content{
      HTML: []byte("<h1>Invoice</h1>"),
    },
  }},
  PageSize: "A4",
}
pdfBytes, err := doc.PDF(ctx)`

export default function LandingPage() {
  const [tab, setTab] = useState('cli')
  const [copied, setCopied] = useState(false)
  const code = tab === 'cli' ? CLI_CODE : GO_CODE

  const handleCopy = () => {
    if (typeof navigator !== 'undefined' && navigator.clipboard) {
      navigator.clipboard.writeText(tab === 'cli' ? 'gowkhtmltopdf input.html output.pdf' : GO_CODE).then(() => {
        setCopied(true)
        setTimeout(() => setCopied(false), 1800)
      })
    }
  }

  return (
    <div className="landing-minimal">
      <PageTitle />

      <section className="landing-hero-minimal" aria-labelledby="landing-title">
        <div className="landing-hero-main">
          <p className="landing-kicker">HTML to PDF &middot; Pure Go &middot; No browser, no cgo</p>
          <h1 id="landing-title">
            Your HTML,
            <br />
            <em>as a print-ready PDF.</em>
          </h1>
          <p className="landing-lede">
            One purpose: turn HTML you author into paginated PDFs. Invoices, reports, certificates,
            tables and multi-page documents with headers, footers and outlines. Two surfaces, same engine.
          </p>
          <div className="landing-actions">
            <Link className="button button-primary" to="/getting-started">
              Get started <span aria-hidden="true">-&gt;</span>
            </Link>
            <Link className="button button-secondary" to="/showcase">View samples</Link>
          </div>
          <p className="landing-micro">
            Drop-in binary <code>gowkhtmltopdf</code> or native Go library <code>Document</code>. Static build with <code>CGO_ENABLED=0</code>.
          </p>
        </div>

        <div className="landing-code-card" aria-label="Minimal conversion example">
          <div className="landing-code-topbar">
            <span className="landing-code-dots" aria-hidden="true"><i /><i /><i /></span>
            <div className="landing-code-tabs" role="tablist" aria-label="Code example">
              <button
                type="button"
                role="tab"
                aria-selected={tab === 'cli'}
                className={`landing-tab ${tab === 'cli' ? 'active' : ''}`}
                onClick={() => setTab('cli')}
              >
                CLI
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={tab === 'go'}
                className={`landing-tab ${tab === 'go' ? 'active' : ''}`}
                onClick={() => setTab('go')}
              >
                Go
              </button>
            </div>
            <button
              type="button"
              className={`landing-copy ${copied ? 'copied' : ''}`}
              onClick={handleCopy}
              aria-label="Copy example to clipboard"
            >
              {copied ? 'Copied' : 'Copy'}
            </button>
          </div>
          <pre><code>{code}</code></pre>
          <div className="landing-code-footer">
            <span>HTML input</span>
            <span aria-hidden="true">-&gt;</span>
            <span>PDF output</span>
            <span className="landing-code-hint">same HTML, same engine</span>
          </div>
        </div>
      </section>

      <section className="landing-flow" aria-labelledby="flow-heading">
        <div className="landing-section-head">
          <h2 id="flow-heading">One pipeline. HTML in, PDF out.</h2>
          <p>Every conversion runs the same in-repo path. No external browser, no wrapper process.</p>
        </div>
        <ol className="flow-grid">
          <li className="flow-card">
            <span className="flow-index">01</span>
            <h3>HTML you control</h3>
            <p>Authored templates, inline or linked CSS subset, local images when you allow it. Script tags are stripped. You own the markup.</p>
            <span className="flow-meta">Author &middot; template &middot; &lt;html&gt;</span>
          </li>
          <li className="flow-card flow-card-accent">
            <span className="flow-index">02</span>
            <h3>Engine</h3>
            <p>Load -&gt; parse -&gt; style -&gt; layout -&gt; paginate -&gt; paint. Tables, lists, page breaks, headers and footers in pure Go.</p>
            <span className="flow-meta">internal/ &middot; load to paint</span>
          </li>
          <li className="flow-card">
            <span className="flow-index">03</span>
            <h3>Print-ready file</h3>
            <p>PDF 1.4 by default, opt-in 1.7 / 2.0 and archival profiles. PNG or JPEG from the same display list for previews.</p>
            <span className="flow-meta">PDF &middot; PNG / JPEG &middot; xref valid</span>
          </li>
        </ol>
        <div className="flow-arrow" aria-hidden="true">
          <span>input.html</span><i /><span>engine</span><i /><span>output.pdf</span>
        </div>
      </section>

      <section className="landing-fit" aria-labelledby="fit-heading">
        <div className="landing-section-head">
          <h2 id="fit-heading">Built for documents, not for browsers.</h2>
          <p>This scope is deliberate. If you need JS or pixel-perfect Chrome parity, use a browser. If you need reliable documents from templates, this is the tool.</p>
        </div>
        <div className="fit-grid">
          <article className="fit-card fit-good">
            <span className="fit-kicker">Good for</span>
            <h3>Templates and tables</h3>
            <p>Invoices, receipts, certificates, statements, posters. Colspan, rowspan, repeated table headers and page-break control.</p>
          </article>
          <article className="fit-card fit-good">
            <span className="fit-kicker">Good for</span>
            <h3>Structured PDFs</h3>
            <p>Text headers and footers with [page] placeholders, TOC, PDF outlines, internal and external links, copies and collate.</p>
          </article>
          <article className="fit-card fit-limit">
            <span className="fit-kicker">Not for</span>
            <h3>JavaScript</h3>
            <p>No JS runtime. Flags like --enable-javascript are rejected. Render HTML first on your server, then convert.</p>
          </article>
          <article className="fit-card fit-limit">
            <span className="fit-kicker">Not for</span>
            <h3>Any website as PDF</h3>
            <p>Flex and grid are partial, floats are lite, and modern CSS is limited. The compatibility matrix is the contract.</p>
          </article>
        </div>
        <div className="fit-links">
          <Link className="text-link" to="/documentation/compatibility">Compatibility matrix <span aria-hidden="true">-&gt;</span></Link>
          <Link className="text-link" to="/documentation/security">Security notes <span aria-hidden="true">-&gt;</span></Link>
        </div>
      </section>

      <section className="landing-proof-minimal" aria-label="Proof points">
        <div className="proof-minimal-item">
          <strong>HTML -&gt; PDF</strong>
          <span>Single purpose, stable output. Same HTML, settings and fonts give same layout.</span>
        </div>
        <div className="proof-minimal-item">
          <strong>Static binary</strong>
          <span>CGO_ENABLED=0. No browser to install, no wrapper daemon.</span>
        </div>
        <div className="proof-minimal-item">
          <strong>Two surfaces</strong>
          <span>CLI work-alike and Go Document API share the pipeline.</span>
        </div>
        <div className="proof-minimal-item proof-minimal-note">
          <strong>Measured</strong>
          <span>2-page invoice: 17 ms CLI vs 259 ms wkhtmltopdf 0.12.6.1 on the reference host. Full numbers on <Link to="/benchmarks">benchmarks</Link>.</span>
        </div>
      </section>

      <section className="landing-samples-minimal" aria-labelledby="samples-heading">
        <div className="samples-copy">
          <h2 id="samples-heading">See the output before you ship.</h2>
          <p>Golden HTML fixtures rendered to committed PDFs and PNGs: invoices, certificates, storybooks, posters, contracts and layout cases.</p>
          <Link className="text-link" to="/showcase">Open sample library <span aria-hidden="true">-&gt;</span></Link>
        </div>
        <div className="samples-stack" aria-hidden="true">
          <div className="sample-paper paper-a"><span>invoice</span><i /></div>
          <div className="sample-paper paper-b"><span>report</span><i /><i /></div>
          <div className="sample-paper paper-c"><small>gowkhtmltopdf</small><strong>PDF</strong><span>HTML -&gt; PDF</span></div>
        </div>
      </section>

      <section className="landing-close" aria-labelledby="close-heading">
        <h2 id="close-heading">Start with one command.</h2>
        <p>Build once, run anywhere. No browser steps.</p>
        <div className="close-code">
          <code>gowkhtmltopdf input.html output.pdf</code>
          <Link className="button button-primary" to="/getting-started">Get started <span aria-hidden="true">-&gt;</span></Link>
        </div>
        <div className="close-links">
          <Link to="/getting-started">First conversion</Link>
          <span aria-hidden="true">&middot;</span>
          <Link to="/documentation/library-api">Go library</Link>
          <span aria-hidden="true">&middot;</span>
          <Link to="/benchmarks">Benchmarks</Link>
        </div>
      </section>
    </div>
  )
}
