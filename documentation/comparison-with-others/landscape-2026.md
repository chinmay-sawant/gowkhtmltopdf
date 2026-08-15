# gowkhtmltopdf vs the HTML-to-PDF landscape (2026)

Research-backed comparison (updated Aug 2026 after PRs [#45](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/45)–[#47](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/47)) of **gowkhtmltopdf** (this project) against
the main HTML-to-PDF converters: Puppeteer / Playwright / Chromium,
wkhtmltopdf, WeasyPrint, and Prince XML. Sources include official docs, GitHub
issues, and published benchmarks. Vendor benchmarks are identified as
self-interested, and the performance figures below are directional rather than
like-for-like measurements.

The current release is **v0.2.2**. Opt-in PDF 1.7 / 2.0 and PDF/A + PDF/UA
profiles ship in this version. Scores below assume that 0.2.2 tree.

## Score card

**gowkhtmltopdf: 8.0/10 overall** (was 7.5). The jump is standards support:
opt-in PDF/A-3a, PDF/A-4, PDF/UA-1, and PDF/UA-2 via `--pdf-profile`, with
veraPDF-passing fixtures. Still held back by a print CSS subset, no
JavaScript, no forms/encryption, and CLI creation-time metadata.

Role scores: **8.8/10 for authored HTML templates** (was 8.5) and **6.0/10
as a general website converter** (was 5.5–6). A 9/10 template score still
needs byte-stable default CLI output.

| Axis (this project) | Score | Why it moved |
|---------------------|------:|--------------|
| Authored templates | 8.8 | Tagged/archival profiles on invoices and certificates |
| Deployment / ops | 9.0 | Still a ~14 MB static `CGO_ENABLED=0` binary |
| PDF standards | 8.0 | A-3a / A-4 / UA-1 / UA-2 opt-in; no A-1/A-2, forms, or Factur-X |
| Security posture | 7.5 | No JS engine; loader/SSRF/DoS still real |
| Open-web / JS | 3.5 | Unchanged: no V8, not Chrome print |
| **Overall** | **8.0** | Weighted toward the template product, not the open web |

## Core comparison

| | gowkhtmltopdf | Puppeteer / Chromium | wkhtmltopdf | WeasyPrint | Prince XML |
|---|---|---|---|---|---|
| **Architecture** | Pure Go, no CGO/browser; one allowlisted shaping module | Blink engine controlled through a browser process | Qt WebKit, frozen and archived | Own Python engine (Pango/HarfBuzz) | Proprietary C++ engine |
| **JavaScript** | **None**; no JS or process execution | Full V8 — SPAs, charts | Old WebKit JS, unreliable | **None** | Sandboxed ES6, no network |
| **CSS fidelity** | Documented **print CSS subset**: flex/grid stages, float lite, `:has()`, `@container`, multicol, print sticky. Not Chrome | Full modern CSS; print pagination still needs testing | No grid, broken flex | Full flex, partial/mostly grid, strong free Paged Media | Deepest Paged Media + grid |
| **TOC / outlines** | ✓ generated TOC + native PDF outlines | Outline support is experimental; **TOC not a native PDF option** | ✓ both (XSLT) | ✓ bookmarks | ✓✓ |
| **Determinism** | **Not guaranteed by default**; CLI currently writes a varying creation time | Timestamps, OS fonts, browser version, and rendering environment vary | Mostly | Near, with explicit reproducibility controls | Depends on inputs and runtime |
| **Security** | No JS/exec, but HTTP(S)/data subresource fetches, optional file reads, and in-process parser/layout DoS remain | Browser sandbox and deployment configuration matter; `--no-sandbox` materially increases impact | **CVE-2022-35583 (9.8 SSRF)**, archived | `<link>`/image fetch and resource-loading risks | Engine and deployment history require review |

## Operations

| | gowkhtmltopdf | Puppeteer / Chromium | wkhtmltopdf | WeasyPrint | Prince XML |
|---|---|---|---|---|---|
| **Latency / PDF** | Repository fixtures: ms–low s; validate on target workload | Published figures vary: cold startup versus warm pooled rendering | Published figures vary by fixture and platform | Published figures vary by fixture and platform | Measure with the vendor's supported deployment |
| **Memory** | Small process on repository fixtures; measure at target concurrency | Browser-plus-page memory varies substantially with browser version, pages, and concurrency | Published single-process figures are workload-dependent | Published figures are workload-dependent | Measure with the vendor's supported deployment |
| **Footprint** | **~14 MB when released as a stripped `CGO_ENABLED=0` binary; no runtime shared libraries** | 100–450 MB + browser/system libs; Docker image size varies | ~15 MB plus aging Qt/runtime libraries | ~30–50 MB + system libs | Small executable plus commercial runtime/license model |
| **PDF standards** | **0.2.2:** opt-in `--pdf-profile` for PDF/A-3a, PDF/UA-1, dual `a3a-ua1`; PDF/A-4, PDF/UA-2, dual `a4-ua2` (veraPDF on committed fixtures). No A-1/A-2, forms, encryption, or Factur-X. Version flags are not a claim | No PDF/A/forms/encryption in the basic PDF API; tagged PDF and outlines are experimental | No | **PDF/A-1/2/3, PDF/UA, forms, Factur-X**; validity still needs validation | Strong standards/commercial extras (PDF/A+X, forms, signatures); verify the required profile |
| **Fonts / CJK** | Partial Type0/CID; pure-Go OpenType shaping where supported; Indic/CJK/vertical limits remain | Full + web fonts, OS-dependent | Poor | Good, fontconfig quirks | Best |
| **License** | **MIT, free, active** | Apache-2.0, free | LGPL, **archived 2023** | BSD, free, active | Commercial site pricing is quoted; official FAQ says it starts around $2,000/year, while $3,800 is a separate per-server non-commercial tier |
| **Score /10** | **8.0 overall** | 7.5 | 3.0 | 7.5 | 9.0 (at price) |

## When to choose

| Tool | Choose when… |
|---|---|
| **gowkhtmltopdf** | Authored HTML templates in Go (invoices, certificates, storybooks, posters, tables, statements) needing a small offline binary, no JavaScript, controlled file/HTTP access, and opt-in PDF/A-3a / A-4 and PDF/UA-1 / UA-2. Also the path off archived wkhtmltopdf |
| **Puppeteer / Chromium** | Templates need **modern CSS/JS** (charts, SPAs, arbitrary sites) and you can afford browser operations plus a materially larger resource budget |
| **wkhtmltopdf** | Avoid for new deployments; it is archived and has a documented unpatched 9.8 SSRF vulnerability. Migrate legacy uses when practical |
| **WeasyPrint** | Free-budget **PDF/A-1/A-2**, AcroForm, or **Factur-X / e-invoice**. Overlap with this project on A-3 / UA; WeasyPrint still wider on older parts and forms |
| **Prince XML** | Enterprise extras this engine does not claim: PDF/X, forms, signatures, deepest CSS paged media — with a commercial license |

## Evidence and interpretation

- The normal conversion path currently calls `SetCreationTime(time.Now())`, so
  byte-identical output is not guaranteed by default. The deterministic PDF
  writer fallback is useful for tests, but it is not the same as a stable CLI
  contract.
- “No JavaScript” is a meaningful security reduction, not a zero-attack-surface
  claim. The loader can fetch HTTP(S) and data subresources, and local-file
  access is an explicit but real capability. Treat untrusted HTML as able to
  drive network egress and resource consumption.
- The 8.0 overall is a judgment call, not a universal benchmark. The 0.5
  lift from 7.5 is the 0.2.2 compliance slice (claiming XMP, OutputIntent,
  tagged structure / MCR / UA list nesting), not a CSS or JS leap.
  Capability, deployment, security, standards, and maintenance should still
  be scored separately when the target is not controlled reports.
- Performance values must be reproduced with the same fixture, tool version,
  OS, cold/warm mode, and concurrency. The cited PDF4 benchmark, for example,
  separates cold and warm browser runs and reports five-run medians on macOS
  arm64; those values should not be merged with unrelated concurrency figures.

## Sources

- [Puppeteer PDF options](https://pptr.dev/api/puppeteer.pdfoptions)
- [Chromium sandbox overview](https://chromium.googlesource.com/chromium/src/sandbox/)
- [wkhtmltopdf archived repository](https://github.com/wkhtmltopdf/wkhtmltopdf)
- [NVD record for CVE-2022-35583](https://nvd.nist.gov/vuln/detail/CVE-2022-35583)
- [WeasyPrint specialized PDFs and forms](https://doc.courtbouillon.org/weasyprint/stable/common_use_cases.html)
- [Prince license FAQ](https://www.princexml.com/purchase/license_faq/)
- [PDF4.dev 2026 benchmark](https://pdf4.dev/blog/html-to-pdf-benchmark-2026)
- [KYOTU wkhtmltopdf/Puppeteer benchmark](https://www.kyotutechnology.com/de/pdf-generation-libraries-performance-wkhtmltopdf-vs-puppeteer/)

## Related docs

- [Overview](../overview.md)
- [Compatibility matrix](../compatibility-matrix.md)
- [Comparison: gowkhtmltopdf vs SebastiaanKlippert/go-wkhtmltopdf](sebastiaanklippert-go-wkhtmltopdf.md)
- [Integration and security](../integration-security.md)
- [Threat model](../THREAT-MODEL.md)
