# Architecture

**gowkhtmltopdf** is a no-cgo HTML→PDF and HTML→image template engine. Three
entry points (the PDF CLI, the image CLI, and the root library) translate
settings into a request and then share one pipeline: load → parse → style →
layout → paginate → paint → write. There is no browser process and no remote
conversion service.

The package graph is a DAG. `internal/convert` is the orchestration hub.
Deep-dives with `file:line` references live under
[architecture/](architecture/).

## Package map

| Package | Responsibility |
|---------|----------------|
| `gowkhtmltopdf` (root `document.go`) | Public library: `Document` / `ImageDocument`, explicit `Content`, validation, writer-first conversion |
| `cmd/gowkhtmltopdf` | PDF CLI (`internal/app` + `internal/cli` only) |
| `cmd/gowkhtmltoimage` | Image CLI |
| `internal/app` | Command → engine adapter: sinks, TOC dump, `RunPDF` / `RunImage` |
| `internal/cli` | Document-shaped argv parse (`-o`, `--html`, `--url`, `--cover`, `--toc`), help, exit codes |
| `internal/settings` | wkhtmltopdf-style dotted settings, `UnitReal`, page sizes, Policy-A ignored keys |
| `internal/errs` | Shared sentinel errors (`ErrNilContext`, `ErrNilRequest`, …) |
| `internal/load` | URL guess, HTTP(S)/file/`data:`/inline HTML, ACL, cookies, auth, POST, timeouts/caps. **No stdin HTML** |
| `internal/html` | Tolerant tokenizer + tree (any tag accepted; no JS). Rendering is gated later by UA styles and layout |
| `internal/css` | CSS subset parse, selectors, cascade, media/container queries |
| `internal/layout` | Style resolve, block/inline/table/flex/grid/float, display list, `PaintContext` pagination |
| `internal/line` | Log-line severity protocol (`info:` / `warning:` / `error:`). **Not** line wrapping |
| `internal/outline` | Headings → outline tree / dump XML (no PDF/layout types) |
| `internal/convert` | PDF job orchestration (HF, TOC, links, copies, document info) |
| `internal/convert/prepare` | Shared document prep: load, parse, sheets, `@font-face` |
| `internal/convert/render` | Mode-neutral lifecycle: `RenderObjects` → `Assemble` → `Finalize` |
| `internal/convert/islands` | Certified page-island recognition for the **benchmark fixture only** |
| `internal/pdf` | PDF writer (default 1.4, opt-in 1.7 / 2.0 via `WriterPolicy`; opt-in `--pdf-profile` / `Document.PDFProfile`), TTF subset, Type0/CID, images, annotations, outlines, tagged structure |
| `internal/pdfprofile` | Leaf: canonical profile tokens, aliases, `Parse` / `IsPDFA*` / `IsPDFUA*` |
| `internal/imageout` | Raster path for one PNG/JPEG canvas |
| `internal/svg` | SVG-as-`<img>` rasterization (`tdewolff/canvas`) |

## Conversion pipeline

```text
input (file / URL / inline HTML)
        │
        ▼
   internal/load          fetch primary document under ACL / timeouts / caps
        │
        ▼
   internal/html          tolerant parse (no script execution)
        │
        ▼
   internal/css           author + UA sheets, cascade, @media / @container
        │
        ▼
   internal/layout        boxes + display list (text, rects, images, links)
        │
        ▼
   layout.PaintContext    paginate the display list (PDF) - page-break-*,
                          thead repeat, orphans/widows, sticky print
        │
        ├──────────────────────────────┐
        ▼                              ▼
  internal/pdf                   internal/imageout
  multi-page PDF                 one NRGBA canvas → PNG/JPEG
  (1.4 default / 1.7 & 2.0 opt-in; `--pdf-profile` for claims)
```

`internal/convert/prepare` is the shared front-half seam for both sinks
(load → parse → collect sheets → merge `@font-face`). Image mode does **not**
call `PaintContext`; it rasterizes the same display-list ops onto a single
canvas.

## `convert.Run` (PDF)

`Document` and `internal/app.RunPDF` build a `convert.Request` and call
`convert.Run`. The function is a thin owner of request state; stage order
lives in `internal/convert/render`:

1. **`Validate` / `ValidatePDF`** - explicit `Output` sink, copies/object
   limits, at least one renderable body object. Dump-outline also requires a
   separate `OutlineOutput`. Image jobs use `imageout.Request.Validate`
   instead of the PDF request path.
2. **Loader** - `load.NewLoaderWithError` so a bad proxy policy fails before
   fonts or the document exist.
3. **Fonts** - bundled default face (`pdf.DefaultFont`) plus an opt-in
   registry from `--font-path` / `--use-system-fonts`.
4. **`pdf.NewDocument` / `pdf.NewDocumentWithPolicy`** - one shared writer for
   the whole job, configured with the requested `WriterPolicy`.
5. **`render.Run`** on the PDF adapter (`pdfPipeline`):
   - **`RenderObjects`** - each `page`/`cover` object: prepare → layout
     (optional smart-shrink re-layout) → `layout.PaintContext` into the
     document. TOC objects are recorded for assembly, not painted yet.
   - **`Assemble`** - document-wide passes (below).
   - **`Finalize`** - `doc.Write` to `req.Output`.

### Assemble (PDF)

`pdfPipeline.Assemble` runs in this order:

| Pass | What it does |
|------|----------------|
| TOC | Two-pass page-count fixpoint; paint TOC pages; `ReorderPages` so TOC is first |
| Outline | Heading tree → PDF bookmarks; optional `--dump-outline` XML |
| Links | Body `#id` / `#name` GoTo, TOC entry targets, URI annotations |
| Document | Page/copy plan; **Info `/Title` from `--title` / `PdfGlobal.Title` only** (HTML `<title>` feeds HF `[doctitle]`, not the PDF Info dict); Producer, compression, grayscale, creation time |
| Copies | Materialize `--copies`; collate vs non-collate reorder |
| Headers/footers | Final pass with real page numbers; cover pages skipped; HTML HF is a single-band clamp |

Page islands are **not** a user-facing layout mode. They run only when a
test constructs `NewBenchmarkPDFRequest` and the HTML matches the certified
benchmark fixture. Production and CLI requests always take the generic
document renderer.

## Image mode

Image jobs use `imageout.Request` (`imageout.RunRequest`), also driven by
`render.Pipeline`:

- **Same front half:** `load` → `html` → `css` → `prepare` → `layout`.
- **`Assemble` is a no-op** (no TOC, outline, copies, or HF).
- **One canvas.** Extra page/TOC objects are ignored with a warning.
- **Text:** TTF outline anti-aliasing at **2× supersample** of the paint
  canvas (`rasterSS = 2`), then box-filter down. The old 5×7 bitmap font is
  **not** the primary path.
- SVG referenced by `<img>` is rasterized in `internal/svg` before layout
  treats it as PNG pixels.

## PDF writer

`internal/pdf` is version-aware and emits PDF **1.4** (default) or PDF **1.7**
/ **2.0** (opt-in via `WriterPolicy`):

| Topic | Behavior |
|-------|----------|
| Header | `%PDF-1.4` (default), `%PDF-1.7`, or `%PDF-2.0` (opt-in via `WriterPolicy`) |
| Trailer `/ID` | Deterministic 16-byte hex identifiers on PDF 1.7 and 2.0 (`/ID [ <a> <b> ]`); absent on 1.4 |
| Info & Metadata | Info dict kept on all versions (Latin-1 on 1.4, UTF-16BE + BOM on 1.7, UTF-8 text strings on 2.0); non-claiming XMP Metadata stream on 1.7 and 2.0 (Dublin Core + Producer, no conformance claims) |
| Classic xref | Standard counting xref table for 1.4, 1.7, and 2.0 (no xref streams) |
| Catalog `/Version` | **Not emitted** on any version - the file header is the sole version authority (matching the 1.7 sibling) |
| Streams | `/Filter /FlateDecode` via zlib (RFC 1950) |
| Latin fonts | Subset TTF, simple font, WinAnsi-style codes, `/Widths` in 1000 units/em |
| Unicode | Type0 / CIDFontType2 + Identity-H when a run has runes above U+00FF |
| Images | JPEG bytes as DCTDecode; PNG → Flate RGB + `/SMask` for alpha |
| Links | URI annotations and GoTo destinations; PDF/UA-2 also emits structure destinations (`/SD`) on internal named dests |
| Outlines | Catalog `/Outlines` after outline object refs exist; UA-2 outline dests can carry `/SD` |
| Info Title | `--title` / settings only - **not** `<title>` |
| Profiles | Empty `--pdf-profile` is unclaimed PDF (default still 1.4). `--pdf-version` is **not** a PDF/A or PDF/UA claim. `--pdf-profile` is: `PDF/A-3a`, `PDF/UA-1`, `PDF/A-3a+PDF/UA-1` (imply 1.7); `PDF/A-4`, `PDF/UA-2`, `PDF/A-4+PDF/UA-2` (imply 2.0). `Get("pdfprofile")` returns those canonical tokens (alias `a3a-ua1` stores `PDF/A-3a+PDF/UA-1`) |

Explicit out-of-scope boundaries: `--pdf-version` alone is a **version**
choice, not a PDF/A or PDF/UA claim. Profiles (claiming XMP, OutputIntent +
sRGB, `/MarkInfo`, structure tree, PDF 2.0 `/Namespace`) are opt-in via
`--pdf-profile` / `Document.PDFProfile`. Encryption, forms, signatures, and
object/xref streams are rejected. Do not claim flavours beyond the tokens
above.

Bundled faces are Liberation Sans **and** Serif **and** Mono (R/B/I/BI), plus
DejaVu Sans Regular+Bold as Unicode fallback. See [fonts.md](fonts.md).

OpenType shaping uses the allowlisted `go-text/typesetting` module (pure-Go
HarfBuzz port). There is no CGO HarfBuzz.

## Import DAG

Nothing points back up. Cycles are forbidden.

```text
api.go  ──► convert, imageout, settings, errs, line     (never imports cli)
cmd/*   ──► app, cli                                    (never imports the root package)
cli     ──► settings                                    (never imports cmd, api, convert)
app     ──► cli, convert, imageout

pdfprofile, errs, line, html, svg   leaves
settings ──► pdfprofile               dotted Set/Get + profile Parse
load     ──► settings                 trust boundary (ACL / timeouts / caps)
css      ──► html
outline  ──► html, css                headings only (locationReader seam)
layout   ──► html, css, pdf, errs     display list + PaintContext
pdf      ──► pdfprofile             sink (also used by imageout for faces/shaping)
imageout ──► prepare, render, load, layout, pdf, settings
convert  ──► load, html, css, layout, line, outline, pdf, settings, prepare, render, islands
                                      the PDF hub

prepare / render / islands  ──► never import convert
```

Rules that keep this sound:

1. **The engine never parses argv.** `internal/cli` writes through dotted
   `Set`; `cli.Command` *is* the settings payload. Root `Document` fields are
   the typed public overlay (no library dotted Set).
2. **One job seam.** Library and CLI adapters build `convert.Request` (PDF)
   or `imageout.Request` (image). Nothing else invokes the pipeline.
3. **Trust boundary.** Primary documents go through `load.Loader.Load`;
   CSS/images/fonts go through `load.ResourceContext.Fetch` → `FetchSub`
   (same ACL, timeout, and body cap).
4. **Image mode is not a parallel engine.** It shares `prepare` + layout and
   diverges at paint/write; it does **not** climb into `convert`.

## Security (summary)

Full model: [THREAT-MODEL.md](THREAT-MODEL.md),
[integration-security.md](integration-security.md).

- Local files are **denied by default**. Opt in with
  `--allow-local-files` and/or `--allow` prefixes. Paths are
  symlink-resolved before the prefix check.
- HTTP: 30 s connect / 60 s response defaults, max 10 redirects, 100 MiB
  body cap on both `Content-Length` and the read side.
- No JavaScript execution. JS-related CLI flags are **unknown options**.
  The library may still store a few inert dotted keys; none evaluate scripts.
- Resource budgets (objects, copies, pages, stylesheet rules) live in
  `internal/convert`, not in callers.
- Header/footer values that look like raw markup (not a URL) are rejected
  at the convert layer.

## Deep-dives

Start at [architecture/README.md](architecture/README.md), then:

| Doc | Domain |
|-----|--------|
| [01-entrypoints-cli.md](architecture/01-entrypoints-cli.md) | CLI grammar, flags, exit codes |
| [02-library-api.md](architecture/02-library-api.md) | `Document` / `ImageDocument` typed fields |
| [03-settings.md](architecture/03-settings.md) | Dotted keys, `UnitReal`, ignored keys |
| [04-load.md](architecture/04-load.md) | ACL, HTTP, `FetchSub` |
| [05-html-parser.md](architecture/05-html-parser.md) | Allowlisted HTML |
| [06-css.md](architecture/06-css.md) | Selectors, cascade, queries |
| [07-layout.md](architecture/07-layout.md) | Formatting contexts, pagination |
| [08-convert-pipeline.md](architecture/08-convert-pipeline.md) | `Run`, Assemble, HF/TOC/islands |
| [09-pdf-writer.md](architecture/09-pdf-writer.md) | PDF 1.4 / 1.7 / 2.0 writer, `--pdf-profile` claims, fonts, images |
| [10-imageout-svg.md](architecture/10-imageout-svg.md) | Raster path, SVG-as-img |

Fidelity and font contracts: [fidelity.md](fidelity.md), [fonts.md](fonts.md).

## Extension points (intentionally small)

The public API is settings-driven (dotted names) **or** the typed
`PDFRequest` / `ImageRequest` path. It is not a plugin framework.
New CSS properties or elements require changes inside `internal/css` and
`internal/layout` and an update to the compatibility matrix.
