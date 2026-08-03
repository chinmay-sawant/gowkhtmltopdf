# Overview

**gowkhtmltopdf** is a pure-Go, **stdlib-only** converter that turns HTML into
PDF (and optionally PNG/JPEG). It is a clean-room work-alike of the
[wkhtmltopdf](https://wkhtmltopdf.org/) CLI surface, aimed at **controlled
server-generated reports** - invoices, statements, multi-page tables, headers
and footers, table of contents, and PDF bookmarks - not full browser parity.

## Design principles

| Principle | Meaning |
|-----------|---------|
| **From scratch** | Entire pipeline is implemented in this repo |
| **Stdlib only** | `go.mod` has zero third-party modules |
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

- A full CSS engine (no flex/grid/floats/position as layout)
- A JavaScript runtime (`<script>` is stripped; flags are no-ops)
- A Unicode/CJK typesetting stack (single embedded Latin font today)
- Pixel-identical WebKit/wkhtmltopdf output

See [compatibility-matrix.md](compatibility-matrix.md) and the deferred list
in the root [README](../README.md#deferred--not-planned).

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
