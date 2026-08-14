# gowkhtmltopdf — Architecture Deep-Dive

This directory holds the detailed architecture documentation for the
**gowkhtmltopdf** codebase: a pure-Go, no-cgo HTML→PDF (and HTML→image)
converter, a work-alike of [wkhtmltopdf](https://wkhtmltopdf.org/) built for
**authored HTML templates** (invoices, certificates, storybooks, posters,
statements, tables, multi-page documents with headers/footers, TOCs and PDF
outlines).

The top-level overview lives at [../architecture.md](../architecture.md).
Each document here is a self-contained deep-dive into one architecture domain,
grounded in the actual source (types, functions and `file:line` references).

---

## 1. The system at a glance

```
                     ┌──────────────────────────────────────────────────────────┐
                     │                 Three entry points                       │
                     │  cmd/gowkhtmltopdf   cmd/gowkhtmltoimage   api.go (lib)  │
                     │        │                    │                   │        │
                     │        ▼                    ▼                   ▼        │
                     │   internal/cli ──► internal/settings (dotted Set/Get)    │
                     └───────────────────────────┬──────────────────────────────┘
                                                 │  job seam:
                                                 ▼
                         convert.Request (PDF) / imageout.Request (image)
                                                 │
                     ┌───────────────────────────▼──────────────────────────────┐
                     │     convert.Run  or  imageout.RunRequest                 │
                     │  render.Pipeline: RenderObjects → Assemble → Finalize     │
                     └───────────────────────────┬──────────────────────────────┘
                                                 │
        ┌─────────── load ───────────┐           │
        │ URLs, ACL, HTTP, cookies   │           ▼
        └────────────────────────────┘   internal/html (allowlisted parser)
                                                 │
                                                 ▼
                                      internal/css (parse / selectors / cascade)
                                                 │
                                                 ▼
                         internal/layout  (style → boxes → display list)
                                                 │
                     ┌───────────────────────────┴──────────────────────────────┐
                     ▼                                                          ▼
         internal/pdf  (PDF writer: 1.4 default,             internal/imageout  (PNG/JPEG)
         1.7 / 2.0 opt-in)                                  TTF outline AA (2× SS); SVG via
         fonts, subsetting, outlines, annotations            internal/svg (tdewolff/canvas)
```

**Non-negotiable constraints** (enforced at the module boundary):

- **No cgo** — `CGO_ENABLED=0`; only Go stdlib plus a narrow, allowlisted pair
  of direct dependencies: `github.com/go-text/typesetting` (OpenType shaping)
  and `github.com/tdewolff/canvas` (SVG rasterization).
- **No browser or native converter process** — everything (load → parse →
  style → layout → paginate → paint → write) runs inside the Go binary.
- **No third-party PDF/HTML/CSS APIs or services.**
- **Controlled-report scope** — not a pixel-perfect clone of arbitrary
  websites. Flex/grid and positioning are "lite"; JavaScript is not executed.

Scale: ~240 Go files in `internal/` (layout is the largest domain) plus the
public `api.go`, two `cmd/` binaries, a React documentation site (`frontend/`,
deploys to `docs/`), golden fixtures (`testdata/`), committed samples
(`output/`), and phase ledgers (`plans/`).

---

## 2. Package map (expanded)

| Domain | Package(s) | Responsibility | Deep-dive |
|--------|-----------|----------------|-----------|
| Entrypoints & CLI | `cmd/gowkhtmltopdf`, `cmd/gowkhtmltoimage`, `internal/cli` | argv → `cli.Command` → settings; multi-object grammar (`page`/`cover`/`toc`) | [01-entrypoints-cli.md](01-entrypoints-cli.md) |
| Public library API | root `api.go` | `Converter` / `ImageConverter`, dotted `Set`/`Get`, typed builders, `Convert(ctx)` | [02-library-api.md](02-library-api.md) |
| Settings & errors | `internal/settings`, `internal/errs` | wkhtmltopdf-style dotted settings, `UnitReal`, page sizes, reflection-based key tables | [03-settings.md](03-settings.md) |
| Load layer | `internal/load` | URL guessing, HTTP/file/`data:`/inline HTML, ACL, cookies, auth, POST, timeouts/caps. No stdin HTML | [04-load.md](04-load.md) |
| HTML parser | `internal/html` | Tolerant tokenizer + tree (any tag accepted), entities, no JS | [05-html-parser.md](05-html-parser.md) |
| CSS subsystem | `internal/css` | CSS subset parse, selectors, cascade, media queries, `:has`, container rules (`:target` never matches) | [06-css.md](06-css.md) |
| Layout engine | `internal/layout` | Style cascade, block/inline/table/flex/grid/float/multicol, pagination, paint ops. `internal/line` is log severity, not wrapping | [07-layout.md](07-layout.md) |
| Convert pipeline | `internal/convert` (+ `prepare/`, `render/`, `islands/`), `internal/outline` | Job orchestration: HF, TOC, outline, links, copies/collate; islands are benchmark-only | [08-convert-pipeline.md](08-convert-pipeline.md) |
| PDF writer | `internal/pdf` (+ `assets/`) | PDF writer (default 1.4, opt-in 1.7 / 2.0 via `WriterPolicy`), font subsetting, Type0/CID, images, annotations, outlines | [09-pdf-writer.md](09-pdf-writer.md) |
| Image output & SVG | `internal/imageout`, `internal/svg` | PNG/JPEG raster path, TTF outline AA (2× supersample), SVG→raster | [10-imageout-svg.md](10-imageout-svg.md) |

---

## 3. Cross-domain dependencies & import-direction rules

The dependency graph is deliberately a **DAG pointing "down" toward the
trust boundary**; nothing points back up.

```
api.go (root) ───────────────────────────────┐   (never imports internal/cli)
cmd/* ─► internal/cli ─► internal/settings    │
        (cli never imports cmd or api.go)     │
                                             ▼
   internal/settings  ◄── leaf settings model (dotted Set/Get)
   internal/errs      ◄── leaf sentinels; consumed by api.go and internal/app
   internal/load      ◄── LOWEST internal package (imports settings + stdlib);
                          it is the trust boundary: ACL/timeouts/caps live here
   internal/html      ◄── feeds css matching + layout; no DOM beyond what layout needs
   internal/css       ◄── feeds layout style resolution
   internal/layout    ◄── produces the display list for pdf OR imageout
   internal/line      ◄── log severity protocol only (not line wrapping)
   internal/outline   ◄── pure headings→outline; no layout/pdf types (locationReader seam)
   internal/convert   ◄── the hub; its subpackages prepare/, render/, islands/
                          never import convert (cycle rule)
   internal/pdf       ◄── writer sink; layout/imageout reuse faces + shaping
   internal/imageout  ◄── image sink; Assemble is a no-op; one canvas
   internal/svg       ◄── leaf consumed by layout (SVG-as-image)
```

Key rules that keep the architecture sound:

1. **The engine never parses argv.** `internal/cli` is a leaf that writes
   through the dotted settings `Set` API; the parser knows nothing about
   rendering.
2. **One job seam.** Library (`api.go`) and both binaries (`internal/app`)
   funnel into `convert.Request` + `convert.Run` (PDF) or `imageout.Request` +
   `imageout.RunRequest` (image). Command translation stays at the application
   edge; nothing else invokes the pipeline directly.
3. **`cli.Command` *is* the settings payload** — no separate DTO; settings
   flow down and are never imported by `layout` or `pdf`.
4. **Everything crosses the trust boundary through two seams:**
   `load.Loader.Load` (primary documents) and `load.ResourceContext.Fetch`
   (linked CSS/images/fonts) — so subresources inherit the identical ACL,
   timeout, and size-cap policy.
5. **Image mode is not a parallel engine.** It shares the full
   load → html → css → layout front half, then diverges only at the
   paint/write tail (`render.Pipeline` lifecycle with `Assemble` as a no-op).

---

## 4. Two binaries, one library

| Surface | Entry | What it produces | How |
|---------|-------|------------------|-----|
| `gowkhtmltopdf` | `cmd/gowkhtmltopdf/main.go` | PDF | `internal/cli` → `internal/app.RunPDF` → settings/request engine |
| `gowkhtmltoimage` | `cmd/gowkhtmltoimage/main.go` | PNG/JPEG | `internal/cli` → `internal/app.RunImage` → settings/request engine |
| Library | `api.go` | PDF or image in memory | `Converter`/`ImageConverter` → `convertHooks` → `convert.Request` with a caller-supplied `Output` writer |

The library never imports `internal/cli`; `cmd/` never imports the root
package. Deep-copy ownership (`AddObject`, `Output`) makes the library safe
under mutation races (validated with `-race`).

---

## 5. PDF vs image mode

```
             ┌─── load ──► html ──► css ──► layout ──► paginate ──┐
             │                                                   │
        display list (text, rects, images, links)                │
             │                                                   │
   ┌─────────▼─────────┐                           ┌─────────────▼─────────────┐
   │  internal/pdf     │                           │  internal/imageout        │
   │  content streams, │                           │  TTF outline AA           │
   │  Flate, xref,     │                           │  (2× supersample canvas,  │
   │  font subsetting  │                           │  not 5×7 bitmap), encode  │
   │  Type0/CID, WOFF  │                           │  SVG via internal/svg      │
   └───────────────────┘                           └───────────────────────────┘
```

- PDF mode: PDF **1.4** (default), PDF **1.7**, or PDF **2.0** (opt-in via
  `WriterPolicy`; `--pdf-version` / `WithPDFVersion`),
  zlib Flate streams, subset TTF (Liberation Sans/Serif/Mono + DejaVu fallback),
  `/Widths` in 1000-unit em, WinAnsi-style Latin-1 codes (UTF-16BE + BOM for 1.7 Unicode Info,
  UTF-8 text strings on 2.0),
  Type0/CID + Identity-H for runes above U+00FF, Catalog outlines, URI + GoTo annotations,
  deterministic trailer `/ID` and non-claiming XMP metadata stream on 1.7 and 2.0.
  PDF 2.0 is a **version**, not PDF/A-4 or PDF/UA-2. Info `/Title` comes from
  `--title`, not `<title>`.
- Image mode: one canvas (`Assemble` is a no-op), `--transparent` support
  (only fill-alpha diverges from PDF paint semantics), no temp files; the
  only third-party raster call in the project is the allowlisted
  `tdewolff/canvas` SVG path.

---

## 6. Security architecture (summary)

Full model: [../THREAT-MODEL.md](../THREAT-MODEL.md) and
[../integration-security.md](../integration-security.md).

- **Local file access is denied by default**; enabled only via
  `--enable-local-file-access` / settings, with `--allow` prefix expansion and
  symlink resolution in `internal/load`.
- **HTTP hardening**: connect/response timeouts (30 s / 60 s), max 10
  redirects, 100 MiB body cap enforced on both `Content-Length` and the read
  side; cookies and auth supported but scoped by the same loader.
- **No JavaScript, ever.** The HTML parser allowlists tags/attributes; script
  execution and interactive elements are dropped at parse time.
- **Resource budgets** (bodies, links, copies) are enforced at the
  orchestration layer (`internal/convert`), not scattered through callers.
- **Header/footer raw-markup rejection** happens at the convert layer.

---

## 7. Document index

| Doc | Domain | Highlights |
|-----|--------|-----------|
| [01-entrypoints-cli.md](01-entrypoints-cli.md) | Entrypoints & CLI | multi-object grammar, flag→setting mapping, PDF vs image binaries, exit codes |
| [02-library-api.md](02-library-api.md) | Public library API | `Converter`/`ImageConverter`, dotted + typed dialects, deep-copy boundary, hooks |
| [03-settings.md](03-settings.md) | Settings & errors | reflection key tables, `UnitReal`, page sizes, Policy-A ignored keys, leaf packages |
| [04-load.md](04-load.md) | Load layer | ACL, URL guessing, network hardening, charset gate, `abort\|skip\|ignore` policy |
| [05-html-parser.md](05-html-parser.md) | HTML parser | allowlisted tokenizer, tree model, entities, tolerance, no-JS policy |
| [06-css.md](06-css.md) | CSS subsystem | selector support, cascade, media/container queries, value parsing, degrade rules |
| [07-layout.md](07-layout.md) | Layout engine | style cascade, all formatting contexts, line breaking/shaping, pagination, display list |
| [08-convert-pipeline.md](08-convert-pipeline.md) | Convert pipeline | `Request`, 3-stage lifecycle, HF/TOC two-pass fixpoint, outline; page islands are **benchmark-only** |
| [09-pdf-writer.md](09-pdf-writer.md) | PDF writer | PDF 1.4 default / 1.7 & 2.0 opt-in model, font subsetting, Type0/CID, outlines, byte stability |
| [10-imageout-svg.md](10-imageout-svg.md) | Image output & SVG | raster path, TTF outline AA, SVG rasterization, fidelity limits vs PDF |

---

## 8. Reading order

1. **Start here**, then [../architecture.md](../architecture.md) for the
   one-page package map.
2. Follow the pipeline: 05 (HTML) → 06 (CSS) → 07 (layout) → 09 (PDF writer)
   or 10 (image out).
3. Understand how jobs are assembled: 01 (CLI) → 03 (settings) → 04 (load) →
   08 (convert orchestration).
4. Understand the library contract: 02 (public API).

---

## 9. How these documents are maintained

Each deep-dive was produced by a subagent that read the source in its domain
and grounded every claim in real types/functions with `file:line` references.
To keep them honest, re-run the same workflow whenever a domain's packages
change materially, and update the affected sections rather than appending.

Known upstream gaps worth reconciling (from the domain reviews):

- JS-related **CLI** flags (`--enable-javascript`, `--javascript-delay`, …)
  are **unknown options**. `load.WaitJSDelay` / `load.WarnJSStubs` do not
  exist. [THREAT-MODEL.md](../THREAT-MODEL.md) §1 matches this. `04-load.md`
  §6.1 may still mention the old names — treat that paragraph as historical.
- `--xsl-style-sheet` is unimplemented (built-in TOC template fallback);
  `[subject]` expands empty; HTML header/footer is single-band clamped.
- The typed settings builder covers global PDF options only; object/image
  options still require dotted `Set` (no compile-time discoverability).
- `PdfGlobal.PageSize` is the canonical page-size name; `Size` stores only
  custom width/height measurements. The former duplicate `Size.PageSize`
  field was removed and settings parity tests protect the single source of
  truth.
- Page islands (`internal/convert/islands`) are a benchmark-only
  optimization, not a user feature. Production/CLI requests never opt in.
