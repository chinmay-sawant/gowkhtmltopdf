# Overview

**gowkhtmltopdf** is a pure-Go, no-cgo HTML template engine that turns HTML into PDF
(and optionally PNG/JPEG). It is a clean-room work-alike of the
[wkhtmltopdf](https://wkhtmltopdf.org/) CLI surface, aimed at **controlled
server-generated templates and documents** — invoices, receipts, certificates,
storybooks, posters, statements, multi-page tables, headers and footers,
table of contents, and PDF bookmarks — not full browser parity.

The entire pipeline (load → parse → style → layout → paginate → paint → write)
is implemented in this repository. There is no Chrome/WebKit process, no Qt,
and no remote HTML→PDF service. The Go module graph is allowlisted: the
standard library plus
[`go-text/typesetting`](https://github.com/go-text/typesetting) (OpenType
shaping) and [`tdewolff/canvas`](https://github.com/tdewolff/canvas) (SVG
rasterization). Builds are intended to run with `CGO_ENABLED=0`.

**Status:** **v0.2.4** is the current release (`VERSION`). Phases 0–9 of the
[canonical plan](../plans/0.1.0/00-canonical-pure-go-rewrite.md) are implemented.
Tier 1 and Tier 2 core (phases 10–20) are shipped as a print CSS subset.
PRs #45–#47 add opt-in `--pdf-version` 1.7 / 2.0 and `--pdf-profile`
(PDF/A-3a, PDF/A-4, PDF/UA-1, PDF/UA-2). Remaining gaps live in
[deferred.md](deferred.md). Progressive post-MVP goals (including URL → decent
print) are goals, not shipped feature claims — see [fidelity.md](fidelity.md).

## Design principles

| Principle | Meaning |
|-----------|---------|
| **From scratch** | The converter pipeline is implemented in this repo |
| **Pure Go** | Build with `CGO_ENABLED=0`; two allowlisted Go modules provide shaping and SVG raster |
| **No browser** | No Chrome, WebKit, Qt, or headless browser process |
| **No cgo** | Static binaries; no native converter library |
| **No SaaS APIs** | No remote HTML→PDF services or SDKs |
| **Template-oriented** | Controlled HTML/CSS templates first; arbitrary websites are a later, weaker goal |
| **Honest degrade** | Unknown CSS is ignored; missing images are skipped; the process should not crash |
| **Repeatable layout** | Same HTML + settings + fonts produce the same layout. CLI PDF **bytes** are not hash-stable unless you inject `Now` (CreationDate / `[date]` / `[time]` use the wall clock by default) |

## What it is good for

- Invoice, receipt, purchase-order, and statement HTML templates
- Multi-section reports with `page-break-*` and repeating `<thead>`
- Tables (`colspan`, `rowspan`, captions), lists, images (PNG/JPEG/SVG-as-`<img>`)
- Local HTML files (ACL opt-in) and HTTP(S) URLs
- Text and nested HTML headers/footers with `[page]` / `[topage]` placeholders
- TOC objects and PDF document outlines (bookmarks)
- In-process embedding from Go (`Document.WritePDF` / `ImageDocument.WriteImage`)
- Opt-in PDF 1.7 / 2.0 (`--pdf-version` / `WithPDFVersion`) and opt-in PDF/A + PDF/UA profiles (`--pdf-profile` / `WithPDFProfile`). Default output is **unclaimed PDF 1.4**; a version flag is not a conformance claim.

Typical path:

```text
application data
        → server-side HTML template
        → gowkhtmltopdf
        → PDF
```

That template path is the product. Pasting an arbitrary public URL is
supported as an **input** (the process can fetch HTTP), not as a promise of
Chrome-quality print. See [fidelity.md — Arbitrary websites](fidelity.md#arbitrary-websites-phase-21).

## What it is not

- A full CSS / browser layout engine. Flex, grid, float, and positioning are
  a documented **print CSS subset** (Partial), not CSS3 / Chrome parity. See the
  [compatibility matrix](compatibility-matrix.md).
- A JavaScript runtime. `<script>` is not executed; JS-related wkhtmltopdf
  flags are **not registered** (unknown option).
- A full Unicode / complex-script stack. Type0/CID + Arabic OpenType via
  `go-text/typesetting` (GSUB) with a presentation-form fallback; Indic is
  Partial; CJK needs a capable face on `--font-path`. See [fonts.md](fonts.md).
- Pixel-identical WebKit / wkhtmltopdf / Chrome output.
- An archival or accessible PDF by default. PDF/A and PDF/UA are **opt-in
  profiles**; `--pdf-version` / `WithPDFVersion` alone does not claim them.
- A wrapper around the `wkhtmltopdf` binary. That is a different product
  category — see
  [comparison-with-others/sebastiaanklippert-go-wkhtmltopdf.md](comparison-with-others/sebastiaanklippert-go-wkhtmltopdf.md).

## Pipeline (at a glance)

```text
input (file / URL / inline HTML)
        │
        ▼
   internal/load       fetch bytes, ACL, cookies, HTTP, network policy
        │
        ▼
   internal/html       tolerant HTML tree (no JS)
        │
        ▼
   internal/css        subset CSS + cascade + media / :has / @container
        │
        ▼
   internal/layout     boxes → display list (y-down canvas)
        │
        ▼
   paginate + paint    multi-page geometry (layout-owned)
        │
        ├──────────────► internal/pdf        PDF write (1.4 default; 1.7 / 2.0 opt-in; profiles opt-in)
        │
        └──────────────► internal/imageout   PNG/JPEG raster
```

Orchestration for PDF lives in `internal/convert` (`RenderObjects` →
`Assemble` → `Finalize`). The public library wraps that in `Document` /
`ImageDocument`. Image mode shares load → parse → style → layout, then
rasterizes one canvas (no pagination, TOC, outline, copies, or headers).

One-page package map: [architecture.md](architecture.md).
Domain deep-dives: [architecture/](architecture/).

## Binaries and library

| Surface | Entry | Output |
|---------|--------|--------|
| PDF CLI | `cmd/gowkhtmltopdf` → `gowkhtmltopdf` | PDF (unclaimed 1.4 default; 1.7 / 2.0 and PDF/A+UA profiles opt-in) |
| Image CLI | `cmd/gowkhtmltoimage` → `gowkhtmltoimage` | PNG or JPEG |
| Go API | module root package `gowkhtmltopdf` | PDF (same version/profile rules) or image in memory / `io.Writer` |

Both CLIs share `internal/cli` and `internal/settings`. The library never
imports `internal/cli`. `cmd/` never imports the root package.

## Feature snapshot

Supported for report HTML:

- Boxes, margins, padding, borders, `box-sizing`
- Tables (`colspan`, `rowspan`, `<caption>`, `<thead>` repeat)
- Lists (`disc` / `decimal` / alpha / roman)
- Images: PNG, JPEG (DCT pass-through in PDF), SVG-as-`<img>` (rasterized)
- Colors, `background-color`, static 2D `transform`, opacity
- `page-break-*`, CSS `orphans` / `widows` (Rule 3 + heuristic)
- Report-subset flex, grid, float, relative/absolute/fixed, print-scoped sticky
- Multicol lite, `:has()`, `@container` size queries
- Text / HTML headers and footers, TOC, PDF outlines, internal and external links
- PDF 1.4 default (unclaimed). Opt-in `--pdf-version 1.7` / `2.0` (version only) and `--pdf-profile a3a-ua1` / `a4-ua2` (claiming XMP, OutputIntent, tagged structure).
- Local files (deny by default) and HTTP(S) with timeouts, body caps, optional restricted network policy

Authoritative rows: [compatibility-matrix.md](compatibility-matrix.md).
Honesty about tiers: [fidelity.md](fidelity.md).

## Security in one paragraph

The converter is an HTTP client (and optional file reader) that runs **in
your process**. Local files are denied unless you opt in.
`--restrict-network` / `RestrictedNetworkPolicy()` blocks private destinations;
the default CLI policy is historical (any `http`/`https` host, including
localhost). There is no JavaScript engine. Prefer generating HTML yourself
and converting that. Details: [THREAT-MODEL.md](THREAT-MODEL.md) and
[integration-security.md](integration-security.md).

## How this project was built

This is a clean-room rewrite, not a binding to wkhtmltopdf/Qt/WebKit.

| Stage | Role |
|-------|------|
| Plans and architecture | Phase ledgers under [`plans/`](../plans), scope freeze, execution map |
| Bulk implementation | Settings/CLI, loader, PDF writer, HTML/CSS/layout, pagination, HF/TOC, image mode, library API |
| Last-mile correctness | Viewer-valid PDFs: Flate/zlib, Catalog outlines, glyph `/Widths`, Latin-1 encoding, `make samples` |

None of that changes the product rule: no third-party PDF/HTML/CSS APIs, no
browser process, and no cgo.

## Next steps

- [Getting started](getting-started.md)
- [CLI reference](cli.md)
- [Library API](library-api.md)
- [Fidelity guide](fidelity.md)
- [Architecture](architecture.md)
