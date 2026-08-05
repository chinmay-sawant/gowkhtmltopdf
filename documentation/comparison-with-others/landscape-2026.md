# gowkhtmltopdf vs the HTML-to-PDF landscape (2026)

Research-backed comparison (Aug 2026) of **gowkhtmltopdf** (this project) against
the main HTML-to-PDF converters: Puppeteer / Playwright / Chromium,
wkhtmltopdf, WeasyPrint, and Prince XML. Sources include official docs, GitHub
issues, and published benchmarks. Vendor benchmarks are identified as
self-interested, and the performance figures below are directional rather than
like-for-like measurements.

## Score card

**gowkhtmltopdf: 7.5/10 overall** — a strong fit for trusted, controlled reports,
held back by partial CSS, no JavaScript, no PDF/A, and the current default
creation-time metadata. **8.5/10 for controlled reports** and **5.5–6/10 as a
general converter** are more useful role-specific scores. A 9/10 niche score
depends on making normal CLI output byte-stable.

## Core comparison

| | gowkhtmltopdf | Puppeteer / Chromium | wkhtmltopdf | WeasyPrint | Prince XML |
|---|---|---|---|---|---|
| **Architecture** | Pure Go, no CGO/browser; one allowlisted shaping module | Blink engine controlled through a browser process | Qt WebKit, frozen and archived | Own Python engine (Pango/HarfBuzz) | Proprietary C++ engine |
| **JavaScript** | **None**; no JS or process execution | Full V8 — SPAs, charts | Old WebKit JS, unreliable | **None** | Sandboxed ES6, no network |
| **CSS fidelity** | Flex/grid/position **lite**; many modern properties missing | Full modern CSS; print pagination still needs testing | No grid, broken flex | Full flex, partial/mostly grid, strong free Paged Media | Deepest Paged Media + grid |
| **TOC / outlines** | ✓ generated TOC + native PDF outlines | Outline support is experimental; **TOC not a native PDF option** | ✓ both (XSLT) | ✓ bookmarks | ✓✓ |
| **Determinism** | **Not guaranteed by default**; CLI currently writes a varying creation time | Timestamps, OS fonts, browser version, and rendering environment vary | Mostly | Near, with explicit reproducibility controls | Depends on inputs and runtime |
| **Security** | No JS/exec, but HTTP(S)/data subresource fetches, optional file reads, and in-process parser/layout DoS remain | Browser sandbox and deployment configuration matter; `--no-sandbox` materially increases impact | **CVE-2022-35583 (9.8 SSRF)**, archived | `<link>`/image fetch and resource-loading risks | Engine and deployment history require review |

## Operations

| | gowkhtmltopdf | Puppeteer / Chromium | wkhtmltopdf | WeasyPrint | Prince XML |
|---|---|---|---|---|---|
| **Latency / PDF** | Repository fixtures: ms–low s; validate on target workload | Published figures vary: cold startup versus warm pooled rendering | Published figures vary by fixture and platform | Published figures vary by fixture and platform | Measure with the vendor's supported deployment |
| **Memory** | Small process on repository fixtures; measure at target concurrency | Browser-plus-page memory varies substantially with browser version, pages, and concurrency | Published single-process figures are workload-dependent | Published figures are workload-dependent | Measure with the vendor's supported deployment |
| **Footprint** | **~14 MB when released as a stripped `CGO_ENABLED=0` binary; no runtime shared libraries** | 100–450 MB + browser/system libs; Docker image size varies | ~15 MB plus aging Qt/runtime libraries | ~30–50 MB + system libs | Small executable plus commercial runtime/license model |
| **PDF standards** | No PDF/A, forms, or encryption | No PDF/A/forms/encryption in the basic PDF API; tagged PDF and outlines are experimental | No | **PDF/A-1/2/3, PDF/UA, forms**; validity still needs validation | Strong standards/commercial extras; verify the required profile |
| **Fonts / CJK** | Partial Type0/CID; pure-Go OpenType shaping where supported; Indic/CJK/vertical limits remain | Full + web fonts, OS-dependent | Poor | Good, fontconfig quirks | Best |
| **License** | **MIT, free, active** | Apache-2.0, free | LGPL, **archived 2023** | BSD, free, active | Commercial site pricing is quoted; official FAQ says it starts around $2,000/year, while $3,800 is a separate per-server non-commercial tier |
| **Score /10** | **7.5 overall** | 7.5 | 3.0 | 7.5 | 9.0 (at price) |

## When to choose

| Tool | Choose when… |
|---|---|
| **gowkhtmltopdf** | Trusted server-generated reports (invoices, tables, statements) needing a small offline Go binary, predictable layout, no JavaScript, and controlled resource access — or migrating off wkhtmltopdf |
| **Puppeteer / Chromium** | Templates need **modern CSS/JS** (charts, SPAs, arbitrary sites) and you can afford browser operations plus a materially larger resource budget |
| **wkhtmltopdf** | Avoid for new deployments; it is archived and has a documented unpatched 9.8 SSRF vulnerability. Migrate legacy uses when practical |
| **WeasyPrint** | **PDF/A, PDF/UA, e-invoice (Factur-X)** compliance on a free budget |
| **Prince XML** | Enterprise compliance (PDF/A+X, forms, signatures) with budget, keeping HTML/CSS input |

## Evidence and interpretation

- The normal conversion path currently calls `SetCreationTime(time.Now())`, so
  byte-identical output is not guaranteed by default. The deterministic PDF
  writer fallback is useful for tests, but it is not the same as a stable CLI
  contract.
- “No JavaScript” is a meaningful security reduction, not a zero-attack-surface
  claim. The loader can fetch HTTP(S) and data subresources, and local-file
  access is an explicit but real capability. Treat untrusted HTML as able to
  drive network egress and resource consumption.
- The score is a judgment call, not a universal benchmark. Capability,
  deployment footprint, security posture, standards support, and maintenance
  should be scored separately when the target workload is not controlled
  reports.
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
