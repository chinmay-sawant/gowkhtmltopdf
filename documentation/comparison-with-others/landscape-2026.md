# gowkhtmltopdf vs the HTML-to-PDF landscape (2026)

Research-backed comparison (Aug 2026) of **gowkhtmltopdf** (this project) against
the main HTML-to-PDF converters: Puppeteer / Playwright / Chromium,
wkhtmltopdf, WeasyPrint, and Prince XML. Sources: official docs, GitHub
issues, and published benchmarks (pdf4.dev Q3-2026, KYOTU 2023, vendor blogs
flagged where self-interested).

## Score card

**gowkhtmltopdf: 8.0/10** — perfect for its niche (deterministic reports), held
back by partial CSS, no JS, no PDF/A. 9/10 for controlled reports; 6/10 as a
general converter.

## Core comparison

| | gowkhtmltopdf | Puppeteer / Chromium | wkhtmltopdf | WeasyPrint | Prince XML |
|---|---|---|---|---|---|
| **Architecture** | Pure-Go, stdlib-only, no browser | Blink engine, full browser process | Qt WebKit, frozen ~2012 | Own Python engine (Pango/HarfBuzz) | Proprietary C++ engine |
| **JavaScript** | **None** (safe by design) | Full V8 — SPAs, charts | Old WebKit JS, unreliable | **None** | Sandboxed ES6, no network |
| **CSS fidelity** | Flex/grid **lite**, no modern CSS | Full modern CSS, broken flex/grid pagination | No grid, broken flex | Full flex, mostly grid, best free Paged Media | Deepest Paged Media + grid |
| **TOC / outlines** | ✓ native both | Outlines ✓ (buggy); **TOC not native** (#1778) | ✓ both (XSLT) | ✓ bookmarks | ✓✓ |
| **Determinism** | **✓ byte-identical guaranteed** | **✗** timestamps + OS fonts vary | Mostly | ✓ near (`SOURCE_DATE_EPOCH`) | ✓ |
| **Security** | ~zero surface — no JS, no fetch | `--no-sandbox` + JS = RCE; SSRF via IMDS | **CVE-2022-35583 (9.8)**, dead | `<link>`/image fetch (Lyft exploit) | XXE history |

## Operations

| | gowkhtmltopdf | Puppeteer / Chromium | wkhtmltopdf | WeasyPrint | Prince XML |
|---|---|---|---|---|---|
| **Latency / PDF** | ms–low s, no startup | 1.8–2.5 s cold; 3–55 ms warm pooled | ~0.36 s | 227–629 ms cold-only | <1 s |
| **Memory** | flat, tiny process | **600 MB–1.3 GB peak** at concurrency | ~35 MiB | ~240 MB flat | tiny |
| **Footprint** | **14 MB static binary, zero deps** | 100–450 MB + ~20 libs; Docker 0.4–2.5 GB | ~15 MB, rotting Qt libs | ~30–50 MB + system libs | ~16 MB |
| **PDF standards** | No PDF/A | No PDF/A, forms, encryption | No | **PDF/A-1/2/3, PDF/UA, forms** | Everything + signatures |
| **Fonts / CJK** | Partial (Type0/CID, Arabic, no HarfBuzz) | Full + web fonts, OS-dependent | Poor | Good, fontconfig quirks | Best |
| **License** | **MIT, free, active** | Apache-2.0, free | LGPL, **archived 2023** | BSD, free, active | **$3,800 + $950/yr** |
| **Score /10** | **8.0** | 7.5 | 3.0 | 7.5 | 9.0 (at price) |

## When to choose

| Tool | Choose when… |
|---|---|
| **gowkhtmltopdf** | Server-generated reports (invoices, tables, statements) needing **deterministic bytes, tiny offline binary, zero security surface** — or migrating off wkhtmltopdf |
| **Puppeteer / Chromium** | Templates need **modern CSS/JS** (charts, SPAs, arbitrary sites) and you can afford browser ops + 100–500× resources |
| **wkhtmltopdf** | **Never** — archived, unpatched 9.8 CVE |
| **WeasyPrint** | **PDF/A, PDF/UA, e-invoice (Factur-X)** compliance on a free budget |
| **Prince XML** | Enterprise compliance (PDF/A+X, forms, signatures) with budget, keeping HTML/CSS input |

## Related docs

- [Overview](../overview.md)
- [Compatibility matrix](../compatibility-matrix.md)
- [Comparison: gowkhtmltopdf vs SebastiaanKlippert/go-wkhtmltopdf](sebastiaanklippert-go-wkhtmltopdf.md)
- [Integration and security](../integration-security.md)
- [Threat model](../THREAT-MODEL.md)
