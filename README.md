# gowkhtmltopdf

Pure-Go, stdlib-only HTML→PDF (and HTML→image) converter - a work-alike for
[wkhtmltopdf](https://wkhtmltopdf.org/) built for **controlled reports**:
invoices, statements, tables, multi-page documents with headers/footers,
TOCs and PDF outlines.

**Built from scratch.** No third-party PDF/HTML/CSS APIs or services, no
Chrome/WebKit embedding, no cgo. The pipeline (load → parse → style → layout →
paint → PDF write) is implemented in this repository. Runtime deps are the Go
standard library **plus** a narrow exception for OpenType shaping via
[`go-text/typesetting`](plans/amendments/2026-08-05-gotext-typesetting.md)
(landed in `go.mod`).

- **Go standard library by default** - the only direct third-party require is
  `github.com/go-text/typesetting` (OpenType shaping; `CGO_ENABLED=0`)
- Two static binaries: `gowkhtmltopdf` (PDF) and `gowkhtmltoimage` (PNG/JPEG)
- Idiomatic Go library API (`gowkhtmltopdf` root package)
- Deterministic output: identical input bytes → identical PDF bytes
- **License:** [MIT](LICENSE) - Copyright (c) 2026 Chinmay Sawant

**Status:** MVP (v0.1.0). Phases 0–9 of the [canonical plan](plans/00-canonical-pure-go-rewrite.md)
are implemented; remaining gaps are listed under
[Deferred / not planned](#deferred--not-planned). Progressive post-MVP goals
(including URL → decent print) are under
[Progressive goals](#progressive-goals-post-mvp) — not MVP feature claims.

**Docs:** start at **[documentation/overview.md](documentation/overview.md)** (full index:
[documentation/README.md](documentation/README.md)).
**Contributing:** [CONTRIBUTIONS.md](CONTRIBUTIONS.md).

---

## Overview

gowkhtmltopdf turns **server-generated HTML** into multi-page PDFs without a
browser process. It is designed for report-style documents - not for pixel-perfect
clones of arbitrary websites.

| You need… | This project |
|-----------|----------------|
| Invoices / tables / page breaks in pure Go | Yes |
| Headers, footers, TOC, PDF bookmarks | Yes |
| Zero native deps / offline static binary | Yes |
| Full CSS (flex, grid, absolute/fixed) or JavaScript | **Partial** flex/grid lite + position lite; **No** JS (see deferred) |
| CJK / complex Unicode fonts | **Partial** — Type0/CID + `--font-path`; Arabic joining; no HarfBuzz |

```text
HTML (file | URL | stdin)
        → load → parse → CSS → layout → paginate → paint
        → PDF 1.4  (or PNG/JPEG in image mode)
```

**Quick taste:**

```sh
make build
./bin/gowkhtmltopdf --enable-local-file-access \
  testdata/golden/fixture-01-simple-invoice.html /tmp/invoice.pdf
# Committed samples: output/   |  regenerate: make samples
```

| Path | What |
|------|------|
| [documentation/getting-started.md](documentation/getting-started.md) | Install and first conversion |
| [documentation/fidelity.md](documentation/fidelity.md) | Fidelity guide (tiers, claims, degrade rules) |
| [documentation/cli.md](documentation/cli.md) | CLI flags and multi-object grammar |
| [documentation/library-api.md](documentation/library-api.md) | Go API |
| [documentation/architecture.md](documentation/architecture.md) | Package map and pipeline |
| [documentation/samples.md](documentation/samples.md) | Golden fixtures and `output/` |
| [output/](output/) | Sample PDFs/PNG checked into the repo |
| [documentation/compatibility-matrix.md](documentation/compatibility-matrix.md) | Support matrix |
| [documentation/THREAT-MODEL.md](documentation/THREAT-MODEL.md) | Security / ACL |
| [documentation/integration-security.md](documentation/integration-security.md) | Gin/web embedding: SSRF, local files, preferred patterns |

---

## How this project was built (AI-assisted, stdlib-only)

This codebase is a **clean-room pure-Go rewrite**, not a binding to
wkhtmltopdf/Qt/WebKit and not a wrapper around any commercial HTML→PDF API.

| Stage | Tooling | Role |
|---|---|---|
| Plans & architecture | **Grok 4.5 (high)** | Phase ledgers under [`plans/`](plans/), scope freeze, feasibility, and the execution map |
| Bulk implementation (~90%) | **DeepSeek** | Phases of settings/CLI, loader, PDF writer, HTML/CSS/layout, pagination, HF/TOC, image mode, library API, hardening |
| Last-mile correctness (~10%) | **Grok 4.5** | Workable, viewer-valid PDFs: Flate/zlib, Catalog outlines, CLI page-scoped flags, glyph `/Widths` (1000-unit em), Latin-1 text encoding; `make samples` green |

DeepSeek was able to drive most of the phase implementation and could emit
PDF *files*, but the output was not reliably **workable** in real viewers
(empty/malformed catalogs, wrong stream compression, broken font advances).
Handing the final pass to Grok 4.5 with a concrete gate - run `make samples`,
fix whatever fails, and open generated PDFs - closed that last gap. Font
letter-spacing is fixed for Latin text; complex pages (e.g. full Wikipedia
articles) still need follow-up for Unicode/CID fonts and richer CSS.

None of that changes the product rule: **no third-party PDF/HTML/CSS APIs** and
**no cgo** — only the Go stdlib, in-tree assets (Liberation Sans), and the
documented shaping exception ([`go-text/typesetting`](plans/amendments/2026-08-05-gotext-typesetting.md))
already in `go.mod`.

---

## Feature status

Supported: no-JavaScript server-generated HTML, a documented CSS subset
(boxes, margins/padding/borders, tables with `colspan`, lists, images,
colors/backgrounds, `page-break-*`), local files (ACL-protected) and
HTTP(S) fetch, multi-page PDF (page size, margins, orientation,
grayscale, compression), text headers/footers with `[page]`-style
placeholders, TOC, PDF outlines, internal/external links, copies/collate.

The authoritative per-element/per-property status is
**[documentation/compatibility-matrix.md](documentation/compatibility-matrix.md)**.
Security posture: **[documentation/THREAT-MODEL.md](documentation/THREAT-MODEL.md)**;
embedding in Gin/APIs: **[documentation/integration-security.md](documentation/integration-security.md)**.

---

## Install & build

Requires Go 1.26+. Build is fully offline - no modules are downloaded.

```sh
# both binaries, static (no cgo anywhere in the graph):
CGO_ENABLED=0 go build ./cmd/gowkhtmltopdf ./cmd/gowkhtmltoimage
```

Reproducible build - always build with `CGO_ENABLED=0` (the default here
anyway, since there is no cgo); the resulting binaries are statically
linked ELF/Mach-O executables with no runtime library requirements.

Version stamping (see [Versioning](#versioning)):

```sh
go build -ldflags "-X gowkhtmltopdf/internal/cli.Version=$(cat VERSION)" ./cmd/gowkhtmltopdf
```

---

## Usage

### gowkhtmltopdf (HTML → PDF)

```
gowkhtmltopdf [GLOBAL OPTIONS] [OBJECT]... <output file>
  OBJECT: [PAGE OPTIONS] page <input> | [TOC OPTIONS] toc | [COVER OPTIONS] cover <input>
  Output "-" writes the PDF to stdout. Run --help / --extended-help for all flags.
```

Basic conversion (note: `--enable-local-file-access` belongs *after* a
`page`/`cover` keyword - page-scoped flags open the current object):

```sh
gowkhtmltopdf page --enable-local-file-access invoice.html invoice.pdf
gowkhtmltopdf --page-size A4 --orientation Landscape report.html report.pdf
gowkhtmltopdf --enable-local-file-access --quiet report.html report.pdf -   # to stdout
# Remote URL (SSRF / untrusted HTML — see documentation/cli.md#remote-url-security)
gowkhtmltopdf 'https://example.com/report.html' out.pdf
```

Multi-object document with header/footer, TOC, and outline:

```sh
gowkhtmltopdf \
  --header-left "gowkhtmltopdf" --header-right "[title]" --header-line \
  --footer-center "[page] / [topage]" --footer-line \
  --toc --toc-header-text "Report contents" --disable-dotted-lines \
  --outline --outline-depth 4 --title "Invoice Report" \
  toc page --enable-local-file-access cover.html \
      page --enable-local-file-access chapter1.html \
      page --enable-local-file-access chapter2.html \
  book.pdf
```

Placeholders in text headers/footers: `[page]`, `[topage]`, `[frompage]`,
`[date]`, `[time]`, `[title]`, `[doctitle]`, `[webpage]`, `[section]`,
`[subsection]`; arbitrary text via `--replace key value`.

### gowkhtmltoimage (HTML → PNG/JPEG)

```sh
gowkhtmltoimage --width 1024 page --enable-local-file-access report.html report.png
gowkhtmltoimage --format jpg --quality 90 --height 800 report.html report.jpg
```

Image mode renders a 1024 px smart-width viewport by default; text uses a
bitmap font (no anti-aliasing - see limits).

### Library API (Go)

`import gowkhtmltopdf` (root package, module `gowkhtmltopdf`):

```go
c := gowkhtmltopdf.NewConverter()
c.Global().Set("size.pagesize", "A4")
c.Global().Set("margin.top", "15")
c.Global().Set("enablelocalfileaccess", "true")

obj := gowkhtmltopdf.NewObjectSettings().SetPage("invoice.html")
obj.Set("load.blocklocalfileaccess", "false")
c.AddObject(obj)

if err := c.Convert(context.Background()); err != nil { panic(err) }
os.WriteFile("out.pdf", c.Output(), 0o644)
```

`NewImageConverter` mirrors this for PNG/JPEG output (`AddObject(page)`,
`Set("format", "png"|"jpg")`, `Convert`, `Output`). Worked examples:
[examples/pdf](examples/pdf/) and [examples/image](examples/image/).

---

## Performance

Phase 9.3 gate: a 10-page invoice table report (10 sections × 40
line-item rows, repeated `<thead>`, `page-break-before` sections) through
the full pipeline (load → parse → style → layout → paginate → paint →
assemble → write).

| Measurement | Value |
|---|---|
| Cold run (first of two) | **~140 ms** (120–149 ms across runs) |
| Warm run (second) | **~156 ms** (96–203 ms across runs) |
| Output size | **96,341 bytes** (10 pages, byte-identical every run) |

- **Command:** `go test ./internal/convert -run TestTenPageTableReportPerformance -v`
- **Machine:** go1.26.4 linux/amd64, Linux x86_64, 13th Gen Intel Core i7-13700HX (24 threads), 2026-08-03
- **Budget asserted in CI:** < 5 s per run (generous - catches
  order-of-magnitude regressions only)

### Benchmark matrix

The reproducible Go benchmark matrix covers 2, 5, 10, 20, 50, 100, 200, 250,
and 500 pages for PDF and template rendering. Web-fetch and inline-image
benchmarks use 2, 5, 10, 20, 50, 100, 200, 250, and 500 image tiles because
image mode renders one raster canvas rather than paginated PDF pages.

The Phase 9.3 gate above is a separate 10-section × 40-row invoice fixture;
the matrix below uses the checked-in benchmark templates (20 realistic rows per
page), so those timings are not directly comparable.

| Workload | 2 | 5 | 10 | 20 | 50 | 100 | 200 | 250 | 500 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| PDF pages | 6.7ms | 11.6ms | 22.2ms | 41.1ms | 115ms | 249ms | 481ms | 590ms | 1.90s |
| Template + PDF pages | 5.2ms | 9.2ms | 18.6ms | 48.5ms | 117ms | 223ms | 459ms | 588ms | 1.69s |
| Web-fetch image tiles | 257.33ms | 258.05ms | 281.10ms | 310.47ms | 356.66ms | 413.68ms | 506.42ms | 564.00ms | 970.72ms |
| Inline image tiles | 209.50ms | 220.61ms | 255.35ms | 282.33ms | 303.54ms | 340.31ms | 439.46ms | 491.22ms | 788.43ms |

PDF / Template: profile-guided residual optimization wave (2026-08-08).
The locked 500-page PDF count-3 median is **1.628s / 678.8MB / 1.103M
allocs**, versus the published **2.10s / 1.48GB / 3.93M** bar. Image-tile rows
are unchanged.

- [Benchmark implementation](internal/convert/benchmarks_test.go)
- [Benchmark templates and recorded results](testdata/golden/benchmarks/README.md)
- [Raw generated benchmark output](testdata/golden/benchmarks/benchmark-results.txt)

Live movie/TV listing benchmark (opt-in, real TVmaze API data and poster CDN):

```sh
GOWKHTMLTOPDF_LIVE_BENCHMARK=1 \
  go test ./internal/convert -run '^$' \
  -bench '^BenchmarkLiveMovieListing/(2Images|5Images|10Images)$' \
  -benchmem -benchtime=1x -count=1
```

See the [live benchmark instructions](testdata/golden/benchmarks/README.md#live-movie-listing-benchmark).

Snapshot command:

```sh
go test ./internal/convert -run '^$' \
  -bench 'Benchmark(PDFPages|TemplatePages|WebFetchImage|ImageAssets)$' \
  -benchmem -benchtime=1x -count=1
```

CPU profiling (stdlib pprof):

```sh
go test ./internal/convert -run TestTenPageTableReportPerformance \
  -cpuprofile /tmp/cpu.pprof
go tool pprof -top /tmp/cpu.pprof
```

---

## Versioning

`VERSION` (currently `0.1.0`) is the single source of truth for the
release number. The CLI `--version` output is stamped at build time with
`-X`; an unstamped build reports the `0.1.0-dev` fallback:

```sh
go build -ldflags "-X gowkhtmltopdf/internal/cli.Version=$(cat VERSION)" ./cmd/gowkhtmltopdf ./cmd/gowkhtmltoimage
./gowkhtmltopdf --version   # Name: gowkhtmltopdf / Version: 0.1.0
```

Release history: [CHANGELOG.md](CHANGELOG.md).

---

## Progressive goals (post-MVP)

Active ledger: **[plans/10-canonical-post-mvp-roadmap.md](plans/10-canonical-post-mvp-roadmap.md)**.
These are **goals**, not MVP feature claims, until their phase acceptance gates pass.

| Goal | Phase | Status |
|------|-------|--------|
| Broader CSS / pagination / fonts / HF edges (Tier 2 core) | 17–20 | Shipped core; see fidelity + matrix |
| **URL → decent print** (readable title + body on wiki/marketing-class HTML; not pixel parity) | **21** | Product contract + docs in progress; acceptance **not** met — do not list as a shipped feature |
| Staged JavaScript | 22 | Not started |
| Open-web / browser competition | 23 | Deferred (not planned under pure-stdlib) |

“Decent print” criteria and explicit non-claims (no Wikipedia visual parity, no
marketing pixel match): **[documentation/fidelity.md](documentation/fidelity.md#arbitrary-websites-phase-21)**.
CLI URL security (SSRF, untrusted HTML): **[documentation/cli.md](documentation/cli.md#remote-url-security)**.

---

## Deferred / not planned

Every deliberate deferral from the phase ledgers (`[~]` items), with its
next gate. Active post-MVP execution ledger:
**[plans/10-canonical-post-mvp-roadmap.md](plans/10-canonical-post-mvp-roadmap.md)**
(phases 10–23). Product fidelity framing:
**[documentation/fidelity.md](documentation/fidelity.md)**. Full WebKit
parity remains **not planned**.

| Deferred | Status / reason | Next gate |
|---|---|---|
| JavaScript / WebKit features (`--enable-javascript`, `--run-script`, `--window-status`, plugins) | No JS engine in stdlib; flags accepted with warnings; `<script>` stripped at load | Phase 22 staged (see post-MVP roadmap) |
| Floats / positioned layout (`float`, `clear`, `position: relative/absolute/fixed`) | **Float lite + relative/absolute/fixed**; sticky = print page scrollport + overflow@0 | Chrome scroll sticky pixel parity non-goal |
| Richer selectors (attribute `[attr=…]`, `:first-child`, `:nth-child`, sibling `+`/`~`) | **Shipped** for presence/exact attr, first/last/nth-child, siblings | Hover/link pseudos still ignored |
| Multi-font bold/italic (Liberation Sans family) | **Shipped** - Regular/Bold/Italic/BoldItalic embedded | Further families: `--font-path` (phase 19) |
| Flexbox / Grid (`display: flex|grid`) | **Partial** flex min-size polish + grid lite + Partial subgrid/masonry | Joint subgrid intrinsic / full L3 masonry / Chrome parity out |
| CJK fonts / complex-script shaping | **Type0/CID + font-path**; **OT Arabic** via `go-text/typesetting` (GSUB) + presentation-form fallback; vertical-rl **rotated CJK** | **No CGO HarfBuzz**; Indic **Partial**; Hangul needs a Hangul face |
| HTML character entities (`&amp;` …) | **Shipped** (stdlib unescape in text + attrs) | — |
| `z-index` | **Lite** on positioned boxes (paint sort) | Stacking contexts / opacity still lite |
| AcroForm forms (`--enable-forms`) | No form model in the PDF writer | Intermediate roadmap (forms) |
| XSLT TOC stylesheets (`--xsl-style-sheet`) | No XSLT in stdlib; flag warns + ignores; default Go-template TOC used | Not planned |
| SVG image output (`--format svg`) | No stdlib SVG encoder | Not planned |
| BMP image output | No demand; PNG/JPEG covered by `image/*` | Not planned |
| SOCKS5 proxy | stdlib `net/http` has no SOCKS5; HTTP(S) proxy only | Not planned |
| Text anti-aliasing in image mode | **Shipped** pure-Go TTF outline raster with coverage AA (same faces as PDF) | Residual: no FreeType hinting |
| Inline `<a href="#x">` source-rect links | **Shipped** for inline text runs with paint boxes; GoTo via `applyInternalLinks` | Cases without geometry still skipped |
| Cross-object URL map (`urlToPageObj`) | Same-document anchors within multi-object jobs via body offsets | Full cross-object URL map still lite |
| `resolveRelativeLinks` | **Shipped** (`--resolve-relative-links` / `--keep-relative-links`) | — |
| HTML header/footer nested documents | **Partial** — child layout + registry/`@font-face` + clipped band; `#id` → body only | Browser HF / running elements out |
| `[topage]` with copies | **Corrected** when HF drawn after copies | — |
| `[subject]` placeholder | Expands empty (no setting field upstream either) | Not planned |
| `dump-outline` TOC page offset | **TOC offset included** via `DumpOutlineXMLOffset` | — |
| Table header repeat across pages | **Shipped** (`table-header-group` / `<thead>` repeat) | Nested-table edge cases documented |
| Smart-shrinking scale-to-width re-layout | **Wired** via `Options.Zoom` + smart-shrink re-layout | — |
| PDF encryption / PDF/A / ICC | Absent or irrelevant upstream too | Not planned |
| C ABI (`wkhtmltopdf_*` cgo exports) | Stdlib mandate forbids cgo | Only if consumer demand |
| `--read-args-from-stdin` batch loop | Flag accepted; batch loop not implemented | Not planned |

---

## Documents

| Document | Purpose |
|---|---|
| **[documentation/README.md](documentation/README.md)** | **Documentation index** |
| [documentation/overview.md](documentation/overview.md) | Product overview and design principles |
| [documentation/fidelity.md](documentation/fidelity.md) | Fidelity guide (tiers, claims, degrade rules) |
| [documentation/getting-started.md](documentation/getting-started.md) | Install, first PDF, library snippet |
| [documentation/architecture.md](documentation/architecture.md) | Pipeline and packages |
| [documentation/cli.md](documentation/cli.md) | CLI usage |
| [documentation/library-api.md](documentation/library-api.md) | Go library API |
| [documentation/samples.md](documentation/samples.md) | Fixtures and committed samples |
| [documentation/compatibility-matrix.md](documentation/compatibility-matrix.md) | Per-element/per-property support matrix |
| [documentation/THREAT-MODEL.md](documentation/THREAT-MODEL.md) | Security / threat model |
| [documentation/integration-security.md](documentation/integration-security.md) | Gin/HTTP integration security (SSRF, preferred patterns; same class as wkhtmltopdf) |
| [plans/00-canonical-pure-go-rewrite.md](plans/00-canonical-pure-go-rewrite.md) | Canonical execution ledger (phase status) |
| [plans/phases/](plans/phases/) | Per-phase ledgers (details + deferral notes) |
| [skills/PR_TEMPLATE/](skills/PR_TEMPLATE/) | Pull-request body template |
| [CHANGELOG.md](CHANGELOG.md) | Release history |
| [examples/](examples/) | Library-API example programs |
| [output/](output/) | Regenerable sample PDFs/PNG (`make samples`) |

## License

[MIT License](LICENSE) - Copyright (c) 2026 **Chinmay Sawant**.

Independent clean-room reimplementation of the wkhtmltopdf CLI/behavior
(wkhtmltopdf itself is LGPL; see the license note in
[plans/00-canonical-pure-go-rewrite.md](plans/00-canonical-pure-go-rewrite.md)).
This project does **not** link to or redistribute wkhtmltopdf, Qt, or WebKit.
