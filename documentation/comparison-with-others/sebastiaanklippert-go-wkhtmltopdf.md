# Comparison: gowkhtmltopdf vs SebastiaanKlippert/go-wkhtmltopdf

This document compares **gowkhtmltopdf** (this project) with
[SebastiaanKlippert/go-wkhtmltopdf](https://github.com/SebastiaanKlippert/go-wkhtmltopdf).

## Short answer

**Yes: that library is a typed Go wrapper around the external `wkhtmltopdf` binary.**
It does not implement HTML layout or PDF generation itself. It finds the binary,
builds command-line arguments from option structs, and runs the process via
`exec.Command` / `exec.CommandContext`.

The real work stays in the native `wkhtmltopdf` tool (historically Qt/WebKit).
Their own README describes the package as a "Golang commandline wrapper for
wkhtmltopdf" and notes that **wkhtmltopdf is unmaintained / archived**, and that
new projects should consider alternatives.

**gowkhtmltopdf** is a different product category: a pure-Go, no-cgo
work-alike engine (load, parse, CSS subset, layout, paginate, paint, PDF/image
write) that does not shell out to `wkhtmltopdf` at all.

## What SebastiaanKlippert/go-wkhtmltopdf does

- Resolves the path to the `wkhtmltopdf` executable (PATH, env, or `SetPath`)
- Exposes options as typed struct fields for IDE completion
- Orders page / cover / TOC options into argv correctly
- Supports stdin HTML via a `PageReader`, stdout or file output, optional
  custom `io.Writer`
- Optional JSON save/load so a job can be prepared on a client and executed
  on a server that has the binary (for example AWS Lambda setups)

The Go package time cost is negligible; conversion speed and fidelity are those
of the installed `wkhtmltopdf` binary and the time to fetch remote HTML.

## What gowkhtmltopdf does

- Implements the full pipeline in-process (see [architecture.md](../architecture.md))
- Ships as static binaries (`gowkhtmltopdf`, `gowkhtmltoimage`) and a Go library API
- Uses a Go-native in-repo pipeline with `CGO_ENABLED=0` (Go modules provide
  shaping and raster support)
- Requires **no cgo**, no Qt, no WebKit, no Chrome, no external converter binary
- Targets **authored HTML templates and documents** (invoices, certificates,
  storybooks, posters, tables, multi-page docs with headers/footers, TOC,
  outlines), not full browser print parity

## Head-to-head

| Dimension | SebastiaanKlippert/go-wkhtmltopdf | gowkhtmltopdf |
|-----------|-------------------------------------|---------------|
| **What it is** | Process wrapper around `wkhtmltopdf` | Clean-room HTML to layout to PDF/image engine in pure Go |
| **Runtime deps** | Must install `wkhtmltopdf` (and its Qt stack) | None: static binary, `CGO_ENABLED=0` |
| **Go modules** | Thin library; still needs the native binary | Go-native dependencies; no browser or native converter process |
| **Rendering** | Full (legacy) WebKit via wkhtmltopdf | In-repo print CSS subset: load, HTML, CSS, layout, paginate, paint, PDF write |
| **PDF version / profiles** | Whatever the installed `wkhtmltopdf` emits (typically unclaimed 1.4; no first-class A-3a/A-4 or UA-1/UA-2) | PDF 1.4 default. **0.2.2:** opt-in `--pdf-version` 1.7 / 2.0 (not a claim) and `--pdf-profile` for PDF/A-3a, PDF/UA-1, PDF/A-4, PDF/UA-2 |
| **Deployment** | OS package or static wkhtmltopdf binary plus PATH setup | Single self-contained Go binary |
| **Security surface** | Spawned process and full browser engine; harder to bound | Explicit ACL, local files off by default, HTTP timeouts and body limits, documented threat model |
| **Determinism** | Depends on binary version, fonts, and OS | Repeatable **layout**. Default CLI **bytes** are not hash-stable (`CreationDate` / `[date]` / `[time]` use the wall clock unless the library injects `Now`) |
| **Maintenance risk** | Tied to archived upstream `wkhtmltopdf` | Full stack owned in this repository |
| **Fidelity** | Closer to "real browser" print for complex pages | Report-oriented CSS subset; no JavaScript; limited fonts today |
| **Outputs** | PDF only (via the binary) | PDF and image mode (`gowkhtmltoimage`) |
| **CLI surface** | Caller builds options in Go; binary is the CLI | Document-shaped `-o/--output`, `--html`/`--url`, positional pages, and `--cover`/`--toc` |

## How gowkhtmltopdf is better

1. **Implementation, not a wrapper**  
   Their package builds flags and `exec`s. gowkhtmltopdf owns load, HTML, CSS,
   layout, pagination, paint, and PDF write inside this repository
   (`internal/load`, `html`, `css`, `layout`, `pdf`, `imageout`).

2. **Ops and embed friendliness**  
   No "is wkhtmltopdf on PATH?", no Qt shared libraries, no Docker layers full
   of fontconfig or X11 dependencies. One static Go binary is enough.

3. **Security posture for servers**  
   gowkhtmltopdf default-denies local file access, supports selective `--allow`
   prefixes, applies HTTP limits, and documents a threat model. A wrapper
   inherits whatever the old WebKit binary does with URLs, files, and scripts.

4. **Future-proof ownership**  
   Their quality ceiling is frozen with an abandoned native binary.
   gowkhtmltopdf can keep improving CSS, fonts, and pagination without waiting
   on Qt-era WebKit.

5. **Auditable PDFs, including opt-in archival / tagged profiles**  
   Pure Go writer and an embedded Liberation subset. 0.2.2 can
   claim PDF/A-3a / A-4 and PDF/UA-1 / UA-2. Default CLI bytes are still
   not hash-stable (wall-clock dates). Golden tests check structure, not
   file hashes.

6. **Library plus CLI work-alike**
   `Document` / `ImageDocument` use typed structs without shelling out to a
   separate process, and the CLI accepts the same explicit sources and output
   model.

## Where SebastiaanKlippert/go-wkhtmltopdf still wins

Be honest about trade-offs:

- **Complex website fidelity today:** real WebKit still beats a print CSS
  subset for arbitrary pages, JavaScript-heavy sites, flex, grid, and similar
  layout models.
- **Maturity and mindshare:** long-lived package with a simple API many
  codebases already use.
- **JSON preparer pattern:** useful if a fleet already runs the native binary
  and needs client-side job prep with server-side execution.

## Bottom line

| | |
|--|--|
| **Their package** | A good ergonomic shell around an external, unmaintained converter |
| **gowkhtmltopdf** | A pure-Go PDF engine based on HTML templates — without any wrappers — not "make calling the binary nicer" |

Use **SebastiaanKlippert/go-wkhtmltopdf** only if you need WebKit-level print
fidelity and accept carrying a dead native toolchain.

Use **gowkhtmltopdf** when deployability, zero native deps, security defaults,
long-term ownership, and (on 0.2.2) opt-in PDF/A + PDF/UA matter more than
full browser parity.

## Related docs

- [Overview](../overview.md)
- [Architecture](../architecture.md)
- [Integration and security](../integration-security.md)
- [Threat model](../THREAT-MODEL.md)
- [Compatibility matrix](../compatibility-matrix.md)
