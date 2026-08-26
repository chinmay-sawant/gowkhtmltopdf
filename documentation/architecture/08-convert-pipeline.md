# Convert pipeline: PDF job orchestration

This document is the deep-dive architecture reference for `internal/convert`
and its companion packages (`internal/convert/prepare`, `internal/convert/render`,
`internal/convert/islands`, `internal/outline`). It expands the package-map row
in [../architecture.md](../architecture.md) ("PDF job orchestration (HF, TOC,
links, copies)") into the detail needed to navigate the code, change the
pipeline, or build on it.

## 1. Responsibility & position in the pipeline

`internal/convert` is the **orchestration hub** of the whole HTML→PDF engine.
It is the only package that coordinates a complete conversion: it owns the
job representation (`Request`), the per-object render loop, the document-wide
passes (TOC, outline, links, headers/footers, copies), and the final PDF
write. It is the consumer of every engine stage:

```text
internal/load (fetch) → internal/html (parse) → internal/css (style)
  → internal/layout + internal/line (layout & paginate)
  → internal/pdf (paint/store)          ← all coordinated by internal/convert
  → internal/imageout (image mode sink) ← same prepare/layout seam, different sink
```

Position in the module graph:

- **Above it:** `document.go` (library `Document` / `ImageDocument`) and `internal/app` (CLI
  adapters) call `convert.Run` / `convert.RunTypedPDF` with a fully-built
  `Request`. The CLI parser (`internal/cli`) produces a `cli.Command`; app
  adapters perform the translation.
- **Below it:** `internal/layout`, `internal/css`, `internal/html`,
  `internal/load`, `internal/pdf`, `internal/line`, `internal/outline`,
  `internal/settings` — all leaf/support packages that `convert` composes.
- **Beside it:** `internal/imageout` consumes the *same* `Request` type for
  image mode (sharing `prepare`), so the request struct is the seam between
  the two output sinks.

The pipeline it drives (mirroring wkhtmltopdf's `PdfConverterPrivate`):

1. validate the request (explicit output sink, mode invariants, limits);
2. build the `load.Loader` (proxy policy fails fast here);
3. load the default font + opt-in font registry;
4. for each object: **prepare** (load → parse → stylesheets → @font-face)
   → **layout** (with smart-shrink retry) → paint ops into the shared
   single `pdf.Document` (constructed via `pdf.NewDocumentWithPolicy` with
   the request's `WriterPolicy` from `PolicyForGlobal` — version and optional
   `--pdf-profile`);
5. assemble: TOC (two-pass page-count fixpoint) → outline → links → document
   info → copies → headers/footers;
6. write the finished PDF to the `io.Writer` sink.

The lifecycle is delegated to `internal/convert/render`'s three-stage
`Pipeline` interface (`RenderObjects` → `Assemble` → `Finalize`); the
`pdfPipeline` adapter in convert/pdf_pipeline.go implements each stage with
the PDF-specific detail.

## 2. Package / file map

All line counts are approximate (`wc -l` at 2026-08 write time) and include
test files where listed.

### 2.1 `internal/convert` (root package: job model + orchestration + passes)

| File | Responsibility | Lines |
|------|----------------|-------|
| `request.go` | Type-safe `PDFRequest` / `ImageRequest` API, `ToRequest` adapters, `RunTypedPDF` | 76 |
| `convert.go` | `Request` union type, limits, `Validate[PDF|Image]`, `Run`, `renderObject`, `runContext`, smart-shrink | 616 |
| `convert_helpers.go` | Page geometry (`pageGeometry`), CSS @page margin override, media resolve, link URI resolution, `loadFontRegistry`, `DefaultTOCXSL` | 227 |
| `prepare.go` | Thin aliases re-exporting the `internal/convert/prepare` seam (`PrepareDocument`, `CollectSheets`, `MergeFontFaces`) | 57 |
| `simplify.go` | Aliases for the DOM-simplification profiles (`AppendSimplifySheet`, `SimplifyDOMEnabled/Profile`) | 26 |
| `hf.go` | Header/footer engine: placeholders, text bands, HTML HF child-document load/layout/draw, auto margins, `drawHeadersFootersResult` | 766 |
| `hf_geometry.go` | `hfGeom` page geometry (points, content area, y-down→y-up conversion helpers) | 44 |
| `toc.go` | TOC generation: effective TOC settings, built-in HTML template (`genTOCHTML`), two-pass fixed-point render (`renderTOCObjects`) | 282 |
| `outline.go` | Per-object state `objectState`, placements, heading collection/flattening, `--exclude-from-outline`, `emitOutline` → `pdf.Outline` | 214 |
| `links.go` | Same-document links: `bodyNavigation` projection, `applyInternalLinks`, `applyTOCLinks`, id index, URI stripping | 291 |
| `page_plan.go` | `pagePlan` owner model over `render.Plan`; `tocFirstOrder`, copy materialization/order helpers, `percent` progress | 191 |
| `page_islands.go` | Certified per-section rendering path for the benchmark fixture (recognition in `islands`, layout driver here) | 175 |
| `pdf_pipeline.go` | `pdfPipeline` adapter implementing `render.Pipeline`: stage order, assembleTOC/Outline/Links/Document/Copies/HF, Finalize write | 186 |
| `doc.go` | Package overview + HTTP-status error note | 6 |
| `convert_test.go` | End-to-end `RunPDF*` tests: objects, copies, media, progress, quiet, cancel, smart-shrink | 704 |
| `seams_test.go` | Sink contract tests (`Run` requires explicit output/outline sinks), mode-specific request constructors, `PrepareDocument` seam | 158 |
| `phase6_test.go` | Phase-6 curated behavior: HF text/HTML, TOC, outline wiring, auto margins, links, cover exclusion, placeholders | 414 |
| `golden_test.go` | Golden-fixture comparisons produced from `testdata/golden/*.html` (regenerate with `make samples`) | 626 |
| `hf_links_test.go` | Header/footer + link interactions (dest wiring, page history) | 385 |
| `links_resolve_test.go` | `collectBodyNavigation`, `buildBodyIDIndex`, `resolveRelativeLinkURIs` | 118 |
| `page_islands_test.go` | Certified island rendering contract (one page per section, fail-closed) | 69 |
| `simplify_test.go` | DOM simplify profiles + `AppendSimplifySheet` ordering | 259 |
| `fontface_test.go` | @font-face merge, WOFF/TTF parsing, registry handoff | 393 |
| `benchmarks_test.go` / `perf_test.go` / `quality_test.go` / `web_fixtures_test.go` / `wk_compare_test.go` / `benchmarks_image_assets_test.go` | Benchmark-report islands, perf seams, quality gates, web fixture parity | ~1.4k combined |

### 2.2 `internal/convert/prepare`: shared load/parse/style/font phase

| File | Responsibility | Lines |
|------|----------------|-------|
| `prepare.go` | `Options` / `Prepared` types; `Document()` load→parse→sheets→fonts; `ResourceContext` (fetch seam, `CollectSheets`, `MergeFontFaces`) | 200 |
| `styles.go` | Sheet collector (`<style>` + `<link>` in document order), media gating, rule-count limits, @font-face fetching (WOFF1/TTF/OTF; WOFF2/EOT/data: rejected) | 243 |
| `simplify.go` | `SimplifyDOM` Chrome/MediaWiki hide-sheet profiles | 61 |

This package is shared verbatim by the image pipeline (`internal/imageout`
calls `prepare.Document` via `ResourceContext`).

### 2.3 `internal/convert/render`: lifecycle + page-index model

| File | Responsibility | Lines |
|------|----------------|-------|
| `pipeline.go` | `Pipeline` interface (`RenderObjects` / `Assemble` / `Finalize`) and `Run` stage driver with cancellation checks between stages | 56 |
| `plan.go` | `Plan` logical page-index model: `OwnerOf`, `Remap` (collate/non-collate), `Ranges`, `MaterializeCopies`, `NonCollateOrder`; copy limit | 187 |
| `pipeline_test.go` | Stage-cancel behavior tests | 78 |
| `plan_test.go` | Copy/collate page mapping tests | 39 |

### 2.4 `internal/convert/islands`: certified page islands (benchmark fixture)

| File | Responsibility | Lines |
|------|----------------|-------|
| `plan.go` | `BenchmarkPlan` fixture certification (marker + title + `section.benchmark-page` body), `ReleaseBenchmarkBodyChildren`, shallow virtual DOM view `Root` | 153 |
| `plan_test.go` | Certification and sibling-release contract | 41 |

### 2.5 `internal/outline`: heading tree & dump XML

| File | Responsibility | Lines |
|------|----------------|-------|
| `outline.go` | `Heading`/`Location` value types, `CollectHeadings`, `Lookup` (layout location merge), `AssignAnchors` (`__WKANCHOR_*`), `BuildTreeBy` level-stack tree with clamping, `Flatten`, `SectionOfBy` (wkhtmltopdf section cache), `DumpOutlineXMLBy`, `CollapseWS` | 465 |
| `outline_test.go` | Tree construction, ordering, XML dump, section lookup | 448 |

`internal/outline` is deliberately **pure tree construction**: converting
canvas coordinates into PDF page space and wiring page object refs is
`internal/convert`'s job (via `emitOutline` and geometry helpers), keeping
`outline` independent of `pdf` and `layout` result types (it only needs a
small `Location` projection).

## 3. Key types, functions & entry points

### 3.1 Job model

| Symbol | Location | Purpose |
|--------|----------|---------|
| `type Request struct` | convert.go | PDF pipeline input: `Global settings.PdfGlobal`, `Objects []settings.PdfObject`, `Now func() time.Time`, `Output io.Writer`, `OutlineOutput io.Writer`. Independent of the CLI parser; adapters build it. Image jobs use `imageout.Request` instead. |
| `func NewPDFRequest(...)` | convert.go | PDF request constructor used by `Document` and `app.BuildPDFRequest`. |
| `func (r *Request) Validate()` / `ValidatePDF()` | convert.go | Explicit output-sink contract before any loading; canonical `PageSize`; object/copies limits; DumpOutline requires `OutlineOutput`. |
| `func Run(ctx, req, log, progress)` | convert.go | Full PDF pipeline entry: validate → loader → font/registry → `runContext` → `render.Run(ctx, &pdfPipeline{run})`. |
| `func app.RunPDF(ctx, cmd, log, progress, outline)` | app/pdf.go | Application-boundary CLI adapter: validates before opening output, builds `Request`, owns document/outline sinks, and calls `convert.Run`. |

### 3.2 Execution state

| Symbol | Location | Purpose |
|--------|----------|---------|
| `type runContext struct` | convert.go:244 | One conversion lifecycle: request, loader, default font, registry, single `pdf.Document`, log/progress, accumulated `tocs`/`bodies`/`headings`, `tocTotal`, `plan`, outline `exclude` selectors. |
| `func (run) report(phase, value)` | convert.go:260 | Progress + log funnel, honoring `req.Global.Quiet`. |
| `func (run) renderObjects(ctx)` | convert.go:270 | Per-object loop: context break, TOC objects vs body objects dispatch (`initTOCState` / `renderObject`), collecting `tocs` and `bodies`. |
| `func renderObject(ctx, run, obj, idx)` | convert.go:414 | The core per-body-object lifecycle: `PrepareDocument` → skip policy → `applyCSSPageMargins` → images callback → `layout.LayoutContext` → smart-shrink re-layout → relative-link resolution → `--no-external-links` strip → `layout.PaintContext` into the shared doc → `objectState` bookkeeping. |
| `type objectRenderContext struct` | convert.go:563 | Render knobs passed down to `bodyLayoutOpts` (global, object, font, registry, sheets, zoom, imagesFn, printLinkUnderline). |

### 3.3 Per-object state

| Symbol | Location | Purpose |
|--------|----------|---------|
| `type objectState struct` | outline.go:42 | Everything one object contributes: identity/settings, geometry (`hfGeom`), header/footer, merged `--replace`, effective TOC, HTML HF layouts, registry/resources, headings, navigation projection, TOC artifacts, and placement (`objectPlacement`). |
| `type objectPlacement struct` | outline.go:25 | Document page indices: `pages`/`offset` set after body paint (pre-TOC), `start` set after TOC reorder. |
| `type bodyNavigation struct` | links.go:24 | Post-paint projection for links/HF: `ids map[string]layout.ElementLocation` + `links []bodyLinkIntent`; drops the display list and DOM pointers. |

### 3.4 Lifecycle adapter

| Symbol | Location | Purpose |
|--------|----------|---------|
| `type pdfPipeline struct` | pdf_pipeline.go:10 | Implements `render.Pipeline` for PDF: `RenderObjects` → `Assemble` → `Finalize`. |
| `func (p) Assemble(ctx)` | pdf_pipeline.go:32 | Ordered passes: `assembleTOC` → `assembleOutline` → `assembleLinks` → `assembleDocument` → `assembleCopies` → `assembleHeadersFooters`. |

### 3.5 Header/footer engine

| Symbol | Location | Purpose |
|--------|----------|---------|
| `type hfParms struct` | hf.go:24 | Per-page substitution state: `page/topage/frompage`, `date/clock`, `title/doctitle/webpage`, `section/subsection`, merged `replaces`. |
| `func (p hfParms) substitute(src)` | hf.go:48 | `--replace` map first, then known `[placeholder]` tokens; unknown tokens stay literal (wkhtmltopdf parity). |
| `func drawTextHF(...)` | hf.go:121 | Paints left/center/right text bands + optional separator line into the margin band using the embedded font. |
| `type htmlHFLayout struct` | hf.go:196 | Cached child-document layout for `--header-html`/`--footer-html`; `perPage` forces one layout per page, otherwise the display list is reused. |
| `func loadHTMLHF(...)` | hf.go:226 | Fetch under ACL, detect placeholders on pristine markup, collect sheets, merge @font-face, layout at content width. |
| `func drawHTMLHF(...)` | hf.go:406 | Clips the nested HF layout to the margin band and paints it; wires fragment/URI links needing document context. |
| `func effectiveMargins(...)` | hf.go:596 | Replaces `auto` (`-1`) top/bottom margins with measured HF height + spacing, so body layout reserves the bands. |
| `func drawHeadersFootersResult(...)` | hf.go:637 | Final pass after the whole document exists: per-page draw with final page numbers, cover-page skip, `hfDrawResult` warning collection. |

### 3.6 TOC engine

| Symbol | Location | Purpose |
|--------|----------|---------|
| `func effectiveTOC(...)` | toc.go:23 | Object-level TOC settings overlay on global; scalars replace, booleans OR. |
| `func genTOCHTML(...)` | toc.go:70 | Built-in default-look TOC template: caption `<h1>`, per-entry indented `<div data-wk-target=...>`, dotted leaders, inline page number. |
| `func paintCount(...)` | toc.go:162 | Measures a layout's page count in a scratch document without mutating the result. |
| `func layoutTOC(...)` | toc.go:175 | Generates + lays out the TOC document for one TOC object with a `tocGuess` page offset. |
| `func renderTOCObjects(...)` | toc.go:221 | **Two-pass fixed point**: iteration 1 measures with `tocGuess=0`; iteration 2 renumbers with the measured total and keeps final layouts; then paints. Returns total TOC pages. |

### 3.7 Outline & page model

| Symbol | Location | Purpose |
|--------|----------|---------|
| `func collectObjectHeadings(...)` | outline.go:102 | Walks h1..h6, honors `UseOutline`/`IncludeInOutline` object gates, merges layout locations. |
| `func flatHeadings(...)` | outline.go:139 | Concatenates per-object headings, fills `DocPage` once, assigns stable anchors, sorts by (DocPage, Y, X). |
| `func emitOutline(...)` | outline.go:184 | Converts the tree to `pdf.Outline` nodes with final page refs and PDF y-up coordinates. |
| `func newPagePlan(...)` | page_plan.go:42 | Builds `pagePlan` (owner list) from object page counts; enforces page/copy limits; delegates mapping to `render.Plan`. |
| `type pagePlan struct` | page_plan.go:29 | `OwnerOf` (final page → object), `Remap` (logical dest → final page in src copy group), `Ranges`, `LogicalN`. |
| `func tocFirstOrder(...)` | page_plan.go:163 | Reorder permutation placing TOC pages first. |

### 3.8 Links

| Symbol | Location | Purpose |
|--------|----------|---------|
| `func applyInternalLinks(...)` | links.go:242 | Fragment (`#id`) ops → GoTo annotations via body id index. |
| `func applyTOCLinks(...)` | links.go:117 | Forward links (entry→heading) and back links (heading→entry) once final pages exist; uses `data-wk-target` anchors. |
| `func collectBodyNavigation(...)` | links.go:33 | Projects ids/links post-paint; nils node pointers so DOM can be GC'd. |
| `func stripLinkURIs(...)` | links.go:77 | Neutralizes external link ops in place (`layout.DeactivateOp`) when `--no-external-links`. |
| `func resolveRelativeLinkURIs(...)` / `resolveRelativeLinkURI(...)` | convert_helpers.go:178 / 192 | Rewrites relative OpLinkURI values against the page base (skips `#`, schemes, mailto). |

### 3.9 Certified islands (benchmark path)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `func benchmarkPageIslandPlan(root)` | page_islands.go:37 | Delegates fixture recognition to `islands.BenchmarkPlan`; fails closed for anything else. |
| `func renderBenchmarkPageIslands(...)` | page_islands.go:46 | Renders each certified `section.benchmark-page` as exactly one page using a shared `layout.Workspace`; trims memory every N islands; appends headings/navigation per island. |
| `func (island) render(...)` | page_islands.go:100 | Per-section layout/paint; errors if an island expands past one page (`errCertifiedIslandExpanded`). |

## 4. Data & control flow

### 4.1 Typical library conversion (`Document.WritePDF`)

```text
document.go Document.WritePDF(ctx, writer)
 └─ convert.Run(ctx, req, lineLog, progress)            convert.go:310
    ├─ req.ValidatePDF()                                (sink + mode invariants first)
    ├─ load.NewLoaderWithError(req.Global.Load)          proxy policy fails fast
    ├─ pdf.DefaultFont() + loadFontRegistry(...)         embedded font + --font-path
    ├─ runContext{ ... single pdf.NewDocumentWithPolicy(policy) ... }
    │        policy := PolicyForGlobal(req.Global)        convert.go:254:
    │                                                     empty version + empty profile
    │                                                     → unclaimed PDF14 (default);
    │                                                     "1.7" → PDF17, "2.0" → PDF20;
    │                                                     `--pdf-profile` / PdfProfile
    │                                                     implies 1.7 (A-3a/UA-1) or 2.0
    │                                                     (A-4/UA-2); version alone is
    │                                                     not a claim; garbage version
    │                                                     → ErrInvalidPDFVersion before
    │                                                     any document exists
    └─ render.Run(ctx, &pdfPipeline{run})                render/pipeline.go:26
       ├─ stage 1 RenderObjects → run.renderObjects      convert.go:270
       │    ├─ per TOC object:  initTOCState (geometry, HF, effective TOC, margins)
       │    └─ per body object: renderObject             convert.go:414
       │         ├─ PrepareDocument → prepare.Document   load→parse→sheets→@font-face
       │         ├─ ApplyCSSPageMargins(geom, sheets)    @page margin shorthand
       │         ├─ layout.LayoutContext(...)            block/inline/table/flex/...
       │         ├─ smart-shrink: measure → zoom → re-layout   (contentW2 > contentW+0.1)
       │         ├─ resolveRelativeLinkURIs / stripLinkURIs (--external-links, --no-external-links)
       │         ├─ layout.PaintContext(doc, ...)        paint ops into pdf.Document
       │         └─ state: pages/offset/headings/navigation
       ├─ stage 2 Assemble → pdfPipeline.Assemble       pdf_pipeline.go:32
       │    ├─ assembleTOC    renderTOCObjects (two-pass fixed point) + doc.ReorderPages(tocFirstOrder)
       │    ├─ assembleOutline BuildTreeBy → emitOutline → doc.SetOutline; DumpOutline XML
       │    ├─ assembleLinks   applyTOCLinks + applyInternalLinks
       │    ├─ assembleDocument plan, Title/Producer/Compression/Grayscale/CreationTime
       │    ├─ assembleCopies  MaterializeCopies (+ non-collate ReorderPages)
       │    └─ assembleHeadersFooters drawHeadersFootersResult (needs final page count)
       └─ stage 3 Finalize → doc.Write(req.Output)       pdf_pipeline.go:127
```

Context cancellation is checked at every object boundary (convert.go:274) and
between render stages (render/pipeline.go:31), so a cancelled request never
enters an expensive later phase.

### 4.2 CLI flow

```text
cmd/gowkhtmltopdf/main.go
 └─ app.RunPDF(ctx, cmd, log, nil, outline)             internal/app/pdf.go
     ├─ BuildPDFRequest(cmd, io.Discard, outline)       validate before creating a file
     ├─ cmd.OpenOutput()                                real sink only after validation
     └─ convert.Run(ctx, req, log, progress)
```

`app.RunPDF` (internal/app/pdf.go) is the command adapter for callers that
hold a `cli.Command`; it owns validation and open/close handling before calling
the CLI-independent `convert.Run` seam.

### 4.3 Image-mode divergence

`Document` / `app.RunImage` build an `imageout.Request` and call
`imageout.RunRequest(ctx, req, log)`. `internal/imageout` reuses the same
`prepare` phase (`prepare.Document` / `ResourceContext`) and
`internal/layout`, but rasterizes the display list instead of painting into
a `pdf.Document`. `convert` is not in the image-mode call path — the shared
seam is `prepare` + `render`, not a convert image request type.

### 4.4 Cross-domain interfaces (the seams)

| Interface / seam | Defined | Implemented by | Consumed by |
|------------------|---------|----------------|-------------|
| `render.Pipeline` | render/pipeline.go:12 | `pdfPipeline` (convert/pdf_pipeline.go:10) | `render.Run` (render/pipeline.go:26) |
| `prepare.Options` / `prepare.Prepared` | prepare/prepare.go | `prepare.Document` | `renderObject` (convert.go:423), `internal/imageout` |
| `ResourceContext` (Fetch/CollectSheets/MergeFontFaces) | prepare/prepare.go | `prepare.NewResourceContext` | body + HTML HF paths (`renderObject`, `loadHTMLHF`) |
| `layout.Options` / `layout.Result` / `layout.PaintContext` / `layout.PaintBandContext` | internal/layout | layout engine | `renderObject`, TOC, HF draw |
| `outline.Heading` / `outline.BuildTreeBy` / `DumpOutlineXMLBy` | internal/outline | pure tree code | TOC/outline/links/HF section placeholders |
| `pdf.Document` (paint, ReorderPages, DuplicatePage, Write, Outline) | internal/pdf | PDF writer | all passes |
| `settings.PdfGlobal` / `PdfObject` / `TableOfContent` / `HeaderFooter` | internal/settings | settings model | every pass (dotted `Get`/`Set` from CLI/engine; Document fields are the public overlay) |

## 5. Cross-package dependencies

Import graph observed in the package (excluding tests):

```text
internal/app ─────┬─ internal/cli        (Command → Request adapters)
                  └─ internal/convert
internal/convert ─┬─ internal/convert/prepare   (load/parse/style/font phase)
                  ├─ internal/convert/render    (lifecycle driver + page-index model)
                  ├─ internal/convert/islands   (certified benchmark islands)
                  ├─ internal/css      (stylesheets, @page margins, selectors for outline Exclude)
                  ├─ internal/html     (parse documents/HF/TOC; node walking)
                  ├─ internal/layout   (LayoutContext, PaintContext, PaintBand, Op model)
                  ├─ internal/line     (log emitter: line.Emit with severity)
                  ├─ internal/load     (Loader, ResourceContext)
                  ├─ internal/outline  (heading tree, dump XML, sections)
                  ├─ internal/pdf      (Document, Font, Registry, Outline)
                  └─ internal/settings (PdfGlobal/PdfObject/etc.)
```

Import-direction rule (enforced by the internal/ directory structure, no
cycles allowed):

- `convert` may import every other internal package.
- `convert`'s subpackages (`prepare`, `render`, `islands`) may NOT import
  `convert` (convert imports them, so a back-edge would be a cycle). They
  depend only on `internal/html`, `internal/css`, `internal/layout`,
  `internal/load`, `internal/pdf`, `internal/settings`, and stdlib.
- `outline` imports only `internal/css`, `internal/html` (+ stdlib) — it
  deliberately avoids `layout` result types via the `locationReader`
  interface and its own `Location` value type (outline.go:21, 29).
- `load` and `pdf` are leaves imported by everything below `convert`.

Who depends on `convert` (i.e. the callers above the seam):

| Caller | Use |
|--------|-----|
| `api.go` | `convert.Run` (line 705, via `convertHooks.executePDF`), `convert.RunTypedPDF` (line 883), `convert.Request` in request adapters |
| `internal/app/pdf.go` | `convert.Run` (line 88) from `app.RunPDF` |
| `cmd/gowkhtmltopdf/main.go` | `convert.DefaultTOCXSL()` (line 49) for `--dump-default-toc-xsl` |
| `internal/app/image.go`, `internal/imageout/imageout.go` | share `convert.Request` / `prepare` seams (image mode) |

## 6. Design decisions & trade-offs

1. **One `pdf.Document` for the whole job** (doc.go). Every object paints
   into the same document (constructed via `pdf.NewDocumentWithPolicy` with
   the request's version policy) so page numbers, TOC, outline links, and
   headers/footers all see a single, final page set. wkhtmltopdf's
   `PdfConverterPrivate` mirrors this: one PDF, one page stream per object.
   Trade-off: intermediate layout results must be projected down to
   navigation/heading metadata before the next object is loaded (memory
   discipline), which is exactly what `bodyNavigation` and the heading lists
   do.

2. **Render lifecycle separated from PDF specifics.** `render.Pipeline` +
   `render.Run` (render/pipeline.go) own *stage ordering and cancellation*;
   `pdfPipeline` owns *PDF details*. This is what lets `Run` be mode-agnostic
   at the top and keeps stage order auditable in one interface. The trade-off
   is indirection: readers must jump between `pdf_pipeline.go` and
   `render/pipeline.go` to see the full flow.

3. **PDF `Request` + separate image request.** PDF mode consumes
   `convert.Request` via `NewPDFRequest`. Image mode uses `imageout.Request`
   and never enters `convert.Run`. Shared prep lives in `prepare`; mode
   boundaries are package seams, not a validate-image method on the PDF
   request.

4. **TOC two-pass fixed point instead of a full solver.** Page numbers inside
   TOC entries depend on how many pages the TOC itself occupies. The engine
   measures with `tocGuess=0`, renumbers with the measured total, measures
   again, and keeps the final layout. Two iterations at most; residual drift
   (a TOC whose renumbered length changed) is documented as rare (toc.go:221
   comment). Trade-off: bounded work instead of exactness.

5. **Smart-shrinking as a width re-layout, not a scale on output.**
   `renderObject` measures the actual op extents (`measuredWidth`,
   convert_helpers.go:150 — rect/image ops, not text), and if content exceeds
   the content area by more than `smartShrinkMinOverflow` (0.1pt,
   convert.go:30), it re-lays out the whole object at `zoom = contentW/contentW2`
   via `layout.Options.Zoom`. This preserves page geometry and only pays for
   a second layout when the overflow is real (sub-tenth-point float noise is
   ignored — a 500-page benchmark report would otherwise re-layout pointlessly).

6. **HF as a final document-wide pass, not per-object ink.** Headers/footers
   need `[topage]`, section/subsection, and final page indices, so they are
   painted only after the whole document (pages + copies) exists
   (`drawHeadersFootersResult`). Auto margins reserve the bands at layout
   time (`effectiveMargins`); the draw pass clips ink to the same bands
   (`drawHTMLHF`). HTML HF documents are full child conversions (fetch under
   ACL, stylesheets, @font-face merge) reused per page unless they contain
   placeholders.

7. **`prepare` exists so PDF and image mode cannot drift.** The
   load→parse→style→font phase is identical for both sinks; extracting it
   into `internal/convert/prepare` (re-exported through the thin aliases in
   convert/prepare.go for compatibility) is the seam that keeps fidelity
   consistent.

8. **Certified island rendering is an explicit, narrow benchmark optimization.**
   Ordinary requests never select the per-section path from document prose.
   Only the internal `NewBenchmarkPDFRequest` opts into recognition of the
   generated report fixture (comment marker + title + `section.benchmark-page`
   body, `islands/plan.go`). Everything else stays on the complete-document
   layout path. The benchmark path clones a parent-consistent virtual tree;
   `debug.FreeOSMemory()` every 4 islands and a shared `layout.Workspace` bound
   peak memory for that explicitly owned workload.

9. **`internal/outline` stays pure.** It computes trees/XML/lookups but never
   touches PDF coordinates or page refs; `emitOutline` and `hfGeom.pdfXY`
   live in convert. This is why `outline` can be unit-tested without a PDF
   harness and reused wherever a heading tree is needed.

## 7. Notable patterns & invariants

- **Explicit-output-sink invariant.** `Run` refuses to run with a nil writer
  (`ErrMissingOutput`) and requires `OutlineOutput` when `DumpOutline` is set
  (`ErrMissingOutlineOutput`) — diagnostics can never be appended to a PDF
  stream (convert.go:77-84; enforced in `Validate`, convert.go:137).
- **Validation before side effects.** `Validate*` runs before the loader,
  fonts, or output files are touched; `internal/app/pdf.go` validates with
  `io.Discard` before `cmd.OpenOutput()` truncates a file.
- **Safety bounds as top-level constants.** `maxConversionObjects=10_000`,
  `maxConversionCopies=1_000`, `maxConversionPages=100_000`,
  `maxStylesheetRules=1_000_000` (convert.go:41-46) guard the pipeline's
  slice/copy multipliers; the page-index model adds a copies×pages product
  check (page_plan.go:55-68). The stylesheet rule budget is also enforced in
  `prepare/styles.go` (`maxStylesheetRules`, soft warn at 25k rules).
- **Progress as (phase, percent) callbacks.** `run.report` funnels progress
  and log lines; percent comes from `percent(i,n)` (page_plan.go:12). CLI
  shows phase lines unless `--quiet`; `progressComplete=100` is the final
  report in `Finalize`.
- **Two-pass / fixed-point passes.** TOC page counts (toc.go:221) and, in
  the wider engine, efforts to re-measure after renumbering. The invariant
  "TOC first, bodies after" is encoded in `tocFirstOrder` (page_plan.go:163)
  and `render.Plan` owner order.
- **Placement lifecycle discipline.** `objectPlacement.pages/offset` are set
  pre-TOC; `start` is rewritten post-reorder (outline.go:25 comment); every
  later pass (links, HF, outline) recomputes final page indices through
  `pagePlan.Remap` instead of caching raw offsets.
- **Merged `--replace` maps.** `mergedReplaces` (convert.go:598) merges all
  four surfaces (global+object × header+footer) so CLI (which stores
  `--replace` on header only) and library users see the same substitution set.
- **Fail-closed certification.** Islands, TOC XSL (falls back to the built-in
  template with a warning, toc.go:176), and unknown HF markup (treated as
  raw, warned) all fail closed rather than guess.
- **Nil-driven policy propagation.** `prep.Resource.Skip` lets the load
  error policy (`--load-error-handling skip`) propagate as a `(nil, nil)`
  object outcome handled by `renderObjects` (convert.go:438-441).
- **Registry handoff is explicit.** `effectiveMargins`/`loadHTMLHF` return
  the HF-extended `*pdf.Registry`; `renderObject` assigns it back to both
  local and state copies ("explicit handshake" comment at convert.go:482),
  so @font-face faces loaded for a header are visible to the body layout.

## 8. Security considerations

`convert` itself does no I/O beyond the `io.Writer` sinks and the loader it
constructs, but it is the enforcement point for several security rules:

- **Loader construction is the proxy/ACL boundary.** `load.NewLoaderWithError`
  runs at the top of `Run` (convert.go:317) *before* fonts/layout state so
  invalid proxy policy is reported immediately. All subresource fetches go
  through `ResourceContext` built on that loader, inheriting the ACL
  (local-file denied unless `--allow-local-files`, `--allow`
  prefixes), timeouts, redirect limits, and body-size caps documented in
  [../THREAT-MODEL.md](../THREAT-MODEL.md).
- **Raw markup vs URL disambiguation in HF.** `loadHTMLHF` treats a
  header/footer value that "looks like HTML" as raw markup and ignores it
  with a warning (hf.go:226-240) — the value is never interpreted as a URL,
  blocking a local-path injection vector through header/footer settings.
- **Resource fetch for links, stylesheets, fonts, images** is name-based and
  ACL-checked through the same loader (prepare/styles.go `collectLink` +
  `fetchFontFace`; HF images via `state.imagesFn` gated by
  `req.Global.Web.Images`, returning `errImagesDisabled`).
- **Output sinks are caller-supplied writers.** `convert` never opens paths;
  open/close stays in `internal/app`. This keeps the
  engine testable and prevents path injection from settings.
- **Dump-outline gets its own sink.** `OutlineOutput` is separate from
  `Output` so XML diagnostics cannot be interleaved into a PDF byte stream
  even when a caller misconfigures (convert.go:163-166).
- **Resource limits are denial-of-service guards** (object/copy/page/rule
  budgets, section 7), complementing the loader's size caps.

## 9. Testing & verification

- **End-to-end `RunPDF*` tests** (convert_test.go): single/multi page,
  styles/tables/images, linked vs screen-only stylesheets, stdout output,
  missing file, copies collate/non-collate, three-object job, progress/quiet,
  context cancellation, smart-shrinking zoom correctness.
- **Seam contract tests** (seams_test.go): `Run` requires explicit output and
  outline sinks; writer errors propagate; `NewPDFRequest` / `ValidatePDF`
  invariants; `PrepareDocument` binds the shared resource context. Image
  request validation lives under `internal/imageout`.
- **Phase-6 behavior tests** (phase6_test.go): text HF, `[page]`/`[frompage]`
  placeholders, section/subsection, TOC, outline on/off, internal link
  destinations, HTML header with per-page placeholders, raw-markup rejection,
  CWD-relative HF paths, auto margins, cover pages without HF, external links
  default-on and disable-honored.
- **Golden fixtures** (golden_test.go + `testdata/golden/*.html`, compare
  against committed `output/` PDFs, regenerate via `make samples`): the
  shipped regression corpus incl. invoices, purchase orders, TOC documents,
  certificates, and the benchmark report.
- **Focused unit tests**: `links_resolve_test.go` (navigation projection,
  id-index duplicates-last, URI resolution), `page_islands_test.go`,
  `render/plan_test.go` (copy/collate mapping), `render/pipeline_test.go`
  (stage cancellation), `islands/plan_test.go` (certification + sibling
  release), `internal/outline/outline_test.go` (tree, XML dump, sections).
- **Perf/quality gates**: `perf_test.go`, `quality_test.go`,
  `benchmarks_test.go`, `wk_compare_test.go` (behavioral parity with
  wkhtmltopdf for the fixture corpus).
- Standard test runner: `go test ./internal/convert/... ./internal/outline/...`
  (CI: `.github/workflows/ci.yml`).

## 10. Known limitations, deferred items & open questions

- **`--xsl-style-sheet` is not implemented.** TOC XSLT is a built-in Go
  template (toc.go:70 `genTOCHTML`); a user-supplied XSL emits a warning and
  falls back (toc.go:175-180). `DefaultTOCXSL()` exists only for
  `--dump-default-toc-xsl` compatibility (convert_helpers.go).
- **TOC fixed point is approximate.** Two iterations at most; if the
  renumber changed the TOC page count, entry page numbers can be off by the
  delta — documented in `renderTOCObjects` (toc.go:221). Rare in practice
  (TOC length is mostly stable under renumbering).
- **`[subject]` placeholder expands to empty** — no Subject setting in this
  build (hf.go:82).
- **HTML headers/footers are single-band clamps.** Content taller than the
  margin band is clipped; no independent multi-page HF (hf.go:209 comment).
- **Fonts**: only the embedded Liberation Sans for general text; `@font-face`
  supports WOFF1/TTF/OTF and rejects WOFF2/EOT/`data:` (prepare/styles.go
  `fetchFontFace`) with warnings. See [../fonts.md](../fonts.md).
- **Islands path is fixture-locked**: remaining benchmark pages and
  open-web documents always take the full-document layout path
  (islands/plan.go:27).
- **Controlled-report scope**: per [../deferred.md](../deferred.md) and the
  fidelity tiers in [../fidelity.md](../fidelity.md) (Tier 1 closed, Tier 2
  core shipped), the pipeline is not optimized for JavaScript-heavy SPAs;
  `--simplify-dom` hides navigation/chrome rather than executing scripts.
- **JavaScript execution is out of scope entirely** (compatibility matrix,
  deferred): the pipeline processes server-rendered HTML only.
- **Open question**: whether a strict HF failure policy (abort on missing
  bands) should be exposed — `hfDrawResult` already collects warnings and
  `Err()` aggregates them (hf.go:354-376), but the compatibility adapter
  (`drawHeadersFooters`) only emits warnings today.

## 11. Related documents

- [../architecture.md](../architecture.md) — high-level package map and
  pipeline (this document expands its `internal/convert` row).
- [../library-api.md](../library-api.md) — how `Document` builds internal `Request`s and
  calls `convert.Run`.
- [../cli.md](../cli.md) — document grammar (`-o`, `--html`, `--url`, `--cover`, `--toc`),
  headers/footers, TOC and outline flags routed into `settings`.
- [../fidelity.md](../fidelity.md) — tiers and degrade rules that constrain
  what the pipeline must do.
- [../THREAT-MODEL.md](../THREAT-MODEL.md) — loader ACL, timeouts, body caps
  that `convert`'s resource seam enforces.
- [../compatibility-matrix.md](../compatibility-matrix.md) — per-element /
  per-property support contract.
- [../fonts.md](../fonts.md) — embedded font, `--font-path`, @font-face
  merge policy.
- [../deferred.md](../deferred.md) — workload prioritization and what stays
  out of scope.
- Sibling architecture docs in this directory: [01-entrypoints-cli.md](01-entrypoints-cli.md)
  (CLI parser and main binaries), [02-library-api.md](02-library-api.md)
  (public API and hooks), [03-settings.md](03-settings.md) (dotted settings
  consumed here), [04-load.md](04-load.md) (loader and ACL used by
  `prepare`/HF), [05-html-parser.md](05-html-parser.md) (node model),
  [06-css.md](06-css.md) (sheets, selectors), [07-layout.md](07-layout.md)
  (`layout.Options`/`Result`/paint consumed by every pass),
  [09-pdf-writer.md](09-pdf-writer.md) (`pdf.Document` sink and object
  wiring), [10-imageout-svg.md](10-imageout-svg.md) (image-mode twin that
  shares `prepare`).
