# Overview

**gowkhtmltopdf** is a pure-Go, no-cgo converter that turns HTML into
PDF (and optionally PNG/JPEG). It is a clean-room work-alike of the
[wkhtmltopdf](https://wkhtmltopdf.org/) CLI surface, aimed at **controlled
server-generated reports** - invoices, statements, multi-page tables, headers
and footers, table of contents, and PDF bookmarks - not full browser parity.

## Design principles

| Principle | Meaning |
|-----------|---------|
| **From scratch** | Entire pipeline is implemented in this repo |
| **Pure Go** | Builds with `CGO_ENABLED=0`; Go modules provide shaping and raster support |
| **No browser** | No Chrome, WebKit, Qt, or headless browser process |
| **No cgo** | Build with `CGO_ENABLED=0`; static binaries |
| **No SaaS APIs** | No remote HTML→PDF services or SDKs |
| **Deterministic** | Same input + settings → same PDF bytes (fixed creation time) |

## What it is good for

- Invoice and statement HTML templates
- Multi-section reports with page breaks
- Tables (including `colspan`), lists, basic CSS boxes
- Local HTML files (with explicit ACL opt-in) and HTTP(S) URLs
- Text headers/footers with `[page]` / `[topage]` placeholders
- TOC objects and PDF document outlines

## What it is not

- A full CSS / browser layout engine (report-subset **Partial** flex, grid lite,
  float lite, and relative/absolute/fixed — not full CSS3; see the
  [compatibility matrix](compatibility-matrix.md))
- A JavaScript runtime (`<script>` is stripped; flags are no-ops)
- A full Unicode / complex-script stack (Type0/CID + Arabic OT via
  `go-text/typesetting` with presentation-form fallback; Indic Partial —
  see [fonts.md](fonts.md))
- Pixel-identical WebKit/wkhtmltopdf output

See the [fidelity guide](fidelity.md) for tiers and claims language, the
[compatibility matrix](compatibility-matrix.md) for the normative contract,
and the deferred list in the root [README](../README.md#deferred--not-planned).

## Pipeline (at a glance)

```text
input (file / URL / -)
        │
        ▼
   internal/load     fetch bytes, ACL, cookies, HTTP
        │
        ▼
   internal/html     allowlisted HTML tree
        │
        ▼
   internal/css      subset CSS + cascade
        │
        ▼
   internal/layout   boxes → display list
        │
        ▼
   paginate + paint  multi-page geometry
        │
        ├──────────────► internal/pdf     PDF 1.4 write
        │
        └──────────────► internal/imageout  PNG/JPEG raster
```

Orchestration for PDF lives in `internal/convert`; the public library wraps
that in `NewConverter` / `NewImageConverter`.

## Binaries and library

| Surface | Entry |
|---------|--------|
| PDF CLI | `cmd/gowkhtmltopdf` → `gowkhtmltopdf` |
| Image CLI | `cmd/gowkhtmltoimage` → `gowkhtmltoimage` |
| Go API | module root package `gowkhtmltopdf` |

## Status

MVP **v0.1.0**. Phases 0–9 of the canonical plan are implemented. Sample
PDFs under [`output/`](../output/) demonstrate the current quality bar for
report-style HTML.

## Next steps

- [Getting started](getting-started.md)
- [CLI reference](cli.md)
- [Library API](library-api.md)
- [Architecture](architecture.md)
