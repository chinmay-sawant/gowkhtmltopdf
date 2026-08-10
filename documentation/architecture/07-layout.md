# Layout engine & line breaking

## 1. Responsibility & position in the pipeline

`internal/layout` is the **style-resolution + formatting engine** of
gowkhtmltopdf: it turns the parsed, CSS-matched HTML tree into an
**absolute-positioned display list** that both output backends consume. It is
the largest single domain in the repository (~33,650 lines including tests,
27 production `.go` files, 62 test files, 244 test functions).

The package's own contract (doc comment on `internal/layout/layout.go:1`):

> turns the parsed HTML tree plus resolved styles into a display list:
> absolute-positioned drawing operations in a continuous canvas (y grows
> downward from the top of the page content area). Painting into a
> `pdf.Document` is done by `Paint` (paint.go).

Position in the pipeline (`README.md`, `documentation/architecture.md`):

```text
load → parse → CSS → layout → paginate → paint → write
                    ^^^^^^^^        ^^^^^^^^
                  layout engine   paint = part of internal/layout
```

The conversion package (`internal/convert`) drives the pipeline; for each
body object it calls `layout.LayoutContext` (or `layout.WithWorkspace` for the
certified page-island path) and then `layout.PaintContext` to paginate and
emit PDF content streams. `internal/imageout` reuses the *same* layout entry
points but rasterizes the display list instead of painting PDF. Headers and
footers are themselves laid out as small nested documents with the same
engine (`internal/convert/hf.go:319,426`), then painted as a single clipped
band via `layout.PaintBandContext`.

Layout owns **pagination as well as geometry**: page-break policies, table-row
integrity, thead repetition, sticky clamps, border-gap sealing, and the
`Result.Pages` / `Result.Locations` projections that outline and link
assembly depend on. Everything between "CSS cascade" and "drawing ops" is
this package.

### Scope summary (report-engine subset)

- Block & inline flow, margin collapsing between siblings
- Tables: separate borders model (collapsed handled as spacing suppression),
  colspan, rowspan, thead/tbody/tfoot grouping, `<thead>` repetition
- Images (PNG/JPEG, inline SVG rasterized via `internal/svg`), `hr`
- Lists with `list-style-type` markers (disc/circle/square/decimal/alpha/roman)
- Text wrapping with per-rune face fallback (bundled Liberation faces +
  opt-in `--font-path` registry)
- Float lite (`left`/`right` + `clear`, simple exclusion)
- Real `inline-block`, `box-sizing`, `position` relative/absolute/fixed lite,
  print-scoped `sticky`
- Partial flex (row/column subset), CSS grid lite, CSS multi-column lite
- Static 2D CSS transforms (paint-time CTM) and opacity
- `@media print|screen` gating, `@container` size queries, `:has`/`:target`
  pseudo matching (via `internal/css`), `::before`/`::after` content
- Pagination: `page-break-*` / `break-*`, orphans/widows (Rule 3 + heuristic),
  table-row integrity, thead repeat

**Not** in scope: JavaScript, general paint clipping (overflow is only parsed
for sticky scrollports), full CSS2.1 float/positioning behavior, flexbox/grid
intrinsic multi-pass cycles, full Multicol L1/L2 balancing.

## 2. Package / file map

Production files (`internal/layout`, 27 files, ~33,650 lines with tests).
Line counts are approximate from `wc -l`.

| File | Lines | Responsibility |
|------|------:|----------------|
| `layout.go` | 1394 | Package entry points (`Layout`, `LayoutContext`, `WithWorkspace`), `Options`/`Result`/`Op`/`Workspace`/`ElementLocation` types, `engine` state, `build` dispatch (block/img/hr/table/flex/grid/multicol/out-of-flow), block width/height resolution, font-face selection & rune fallback |
| `style.go` | 660 | `ResolvedStyle` struct, `initialStyle()`, inheritance walk, `styleStore` interning (canonical `*ResolvedStyle` sharing), `resolveStyles*` entry points, `styleContext`, `sizeContainer` |
| `style_cascade.go` | 573 | `cascadeRaw` (UA + author + inline with specificity/order/!important), `matchedRules`, custom properties (`--*`) merge + var resolution, inherit table, cascade-win comparison |
| `style_properties.go` | 1253 | Property-group dispatch (`styleGroups` table): display/position/flex/multicol/grid/box/border/color/text/table-break/transform; all `apply*` setters |
| `style_values.go` | 1010 | Value parsers: lengths, font-size keywords, line-height, border widths, flex/grid shorthands, `uaDecls` UA table, `uaRules(name)` |
| `container.go` | 287 | `@container` size-query support: `findSizeContainer`, `measureSizeContainers`, `contentInlineSize` |
| `layout_flow.go` | 850 | In-flow child dispatch (`flowChildren`/`flowOneChild`), inline-run collection, list markers, float placement + packing, BFC float-state push/pop, image ref resolution (incl. SVG raster) |
| `inline.go` | 925 | Inline formatting: item packing into lines, float exclusion (`lineBounds`), overflow splitting, word-break policies, justification, line metrics, glue/sticky-tail handling |
| `inline_collect.go` | 740 | Inline item collection: text (pre/wrapped), `<br>`, `<img>`, inline-block, inline spans, href attachment, whitespace squeezing, `::before`/`::after` content, soft-wrap punctuation rules |
| `inline_paint.go` | 590 | Inline emission to ops: text runs (per-face), decoration (underline/line-through), inline-block/image paint, face-run splitting for fallback |
| `layout_tables.go` | 1002 | Table layout: row/col collection, cell placement (colspan/rowspan occupancy), column sizing (min/max/%/abs), row heights, border emission, `<thead>` header-row counting, rowspan line redistribution |
| `layout_measure.go` | 983 | Measurement passes: cell min/max-content, band baseline grouping (rowspan vertical distribution), `minContentWidth` per word-break policy, `layoutCell` |
| `flex.go` | 1286 | Flex layout: row/column, wrap, grow/shrink/basis, align/justify, order, min-main-size clamps, `applyRelativeOffset` |
| `grid.go` | 1664 | Grid layout: track defs (`fr`, `minmax`, `repeat`), auto-placement (row/column/dense), template areas, span resolution, row sizing, item placement/alignment |
| `multicol.go` | 504 | Multi-column lite: count/width/gap/span/fill, segment collection, spanner flow, single-column fallback |
| `float.go` | 234 | `floatState`: clear semantics, left/right placement, exclusion (`exclusion`), BFC establishment |
| `sticky.go` | 234 | Print-scoped sticky: `tagSticky`, `applyStickyPrint` (page content box scrollport), overflow scrollport @0, `clampStickyX/Y`, op shifting |
| `transform.go` | 879 | `Matrix2D`, transform-list parsing, `transform-origin`, `stampBoxTransforms` (composed CTM + opacity stamping) |
| `layout_chrome.go` | 480 | Background/border op generation (`prependChrome`), deferred chrome merge, border lines (solid/dashed/dotted), `markOpsFixed`, radius |
| `layout_images.go` | 317 | Replaced-element sizing: intrinsic ratio policy, `width`/`height` attrs + CSS, aspect-ratio preservation, `buildHR`, PNG/JPEG dimension sniffing |
| `paint.go` | 981 | `Paint`/`PaintContext` (pagination orchestration), `PaintBand`/`PaintBandContext` (single-band, used by HF), op→PDF dispatch, `PaintStyle`/`StyleOf` (shared fill/stroke/fake-bold policy), `populateLocations`, `canvasToPDF`, `roundedRectPath` |
| `paint_flow.go` | 1725 | Flow-index machinery (`shiftFlowY` etc.), page-break policies (`avoidInside`, `beforeAlways`, `afterBreaks`, `rowsIntact`, `keepHeadingWithNext`, `orphansWidows` + heuristic), thead repetition |
| `paint_pagination.go` | 1502 | `paginateOps`, `paginationFixpoint`, `snapCrossingTextOps`, `splitCrossingRects` (op fragmentation w/ identity), `capTablePageBreaks` (border-gap sealing), `stripOrphanRowChrome` |
| `paint_order.go` | — | `PaintOrder` — canonical z-order policy shared by PDF, band, and raster adapters |
| `pseudo_content.go` | 368 | `::before`/`::after` `content:` cascade + value parsing (strings, `attr()`, escapes) |
| `mnd_const.go` | — | Named magic numbers (magic-number linter constants) |
| `doc.go` | — | Package doc stub (real contract lives on `layout.go` header) |

Test files (62): the largest are `layout_test.go` (1374, core formatting +
pagination), `sticky_test.go` (904), `flex_test.go` (878), `grid_test.go`
(873), `fixture_bugs_test.go` (552), `style_store_test.go` (422). See §9.

### `internal/line` — naming note

The task scope names `internal/line` as part of this domain, but **`internal/line` is not text line breaking**: it is the engine's *log-line severity
protocol* (`line.Emit` / `line.SeverityOf` for `info:`/`warning:`/`error:`
prefixed lines). It is consumed by `internal/convert`, `internal/app`,
`internal/imageout`, and `api.go` — never by `internal/layout`. Actual line
breaking lives inside `internal/layout` (`inline.go`, `inline_collect.go`).
Flagged so future readers do not search the wrong package.

## 3. Key types, functions & entry points

### Entry points (the public surface of the domain)

| Symbol | File:line | Purpose |
|--------|-----------|---------|
| `Layout(root *html.Node, opts Options) (*Result, error)` | `layout.go:696` | Legacy background-context layout |
| `LayoutContext(ctx, root, opts) (*Result, error)` | `layout.go:703` | Cancellation-aware layout; observed at style pass and recursion boundaries |
| `WithWorkspace(ctx, root, opts, ws *Workspace) (*Result, error)` | `layout.go:712` | Sequential internal form borrowing display-list storage; `Workspace.Release` returns it |
| `Paint(doc *pdf.Document, res, opts)` / `PaintContext(...)` | `paint.go:62,69` | Paginate display list + paint into a `pdf.Document` |
| `PaintBand(p, c, ops, opts)` / `PaintBandContext(...)` | `paint.go:498,504` | Single-band paint on an existing page content stream (HTML HF); no pagination, no fixed stamps |
| `PaintOrder(ops []Op) []int` | `paint_order.go:8` | Canonical display-list z-order used by all three backends |
| `CloneResult(res) *Result` | `layout.go:116` | Deep copy for TOC page-count fixpoint (`internal/convert/toc.go:164`) |
| `Workspace.Release(res)` | `layout.go:216` | Return op storage; clears box/paint indexes (page-islands path) |

### Core data types

| Type | File:line | Notes |
|------|-----------|-------|
| `Options` | `layout.go:62` | Width/Height (content box pt), `Font`/`Faces`/`Registry`, `Sheets []*css.Stylesheet`, `Media` (`"print"`/`"screen"`/`""`), `Images func(src) ([]byte, error)`, `Background`, `DebugBoxes`, `Zoom`, `PrintLinkUnderline` |
| `Result` | `layout.go:81` | `Ops []Op`, canvas `Width`/`Height`, private box tree, `Pages [][]int` (page→op indices), `Locations []ElementLocation` |
| `Op` | `layout.go:271` | Discriminated by `OpKind`: `OpFillRect`, `OpStrokeRect`, `OpLine`, `OpText`, `OpImage`, `OpLinkURI`, `OpBullet`; carries `ID` (stable identity across fragmentation), font/size/letter-spacing/text-transform, image bytes, `Fixed`, `StickyID`, `ZIndex`, `Positioned`, `Xform`, `PaintOpacity`, `Radius` |
| `ElementLocation` | `layout.go:238` | `Node`, `Page`, `X/Y/W/H` canvas rect; consumed by `internal/outline.Lookup` and link assembly |
| `ResolvedStyle` | `style.go:65` | ~1.3 KB fully resolved style: display/position/float, box model, flex/grid/multicol fields, text fields, `Transform`/`Opacity`, `CustomProps` |
| `box` | `layout.go:~948` | Laid-out box: node, style pointer, border-box x/y/w/h, kind, op range `[opStart,opEnd]`, children, table rows/cells, sticky state, image ref |
| `engine` | `layout.go:333` | One layout run: options, ctx, styles map, op buffer, z-index stack, BFC float stack, caches (faces, images, rune fallback), `imgMaxW`/`inlineCBW` context widths |

### Style resolution entry points

| Symbol | File:line | Purpose |
|--------|-----------|---------|
| `resolveStylesForLayout(root, opts)` | `layout.go:825` | Cascade + optional `@container` re-cascade (measures size containers, one nested remount) |
| `resolveStylesCtx(root, ctx)` | `style.go:350` | Top-down walk; text nodes share parent style; `styleStore.intern` canonicalizes |
| `cascadeRaw(ctx, node) map[string]string` | `style_cascade.go:306` | Winning declaration per property across UA sheet → author sheets (media/container/selector gated) → inline style |
| `applyStyleProp` via `styleGroups` | `style_cascade.go:537,560` | Immutable 11-entry dispatch table for property groups |
| `inheritProps` | `style_cascade.go:172` | 20 inheritable property groups copied unless locally declared |

### Layout build dispatch

`engine.build` (`layout.go:996`) is the recursive tree walk:

```text
build(node, availW, x, y)
├─ display:none → nil
├─ <img> → buildImage        (layout_images.go:196)
├─ <hr>  → buildHR           (layout_images.go:225)
├─ position:fixed/absolute → buildOutOfFlow (layout.go:1298)
├─ buildInFlowDisplay        (layout.go:1367) — single dispatch:
│   ├─ flex/inline-flex  → buildFlex   (flex.go:57)
│   ├─ grid/subgrid      → buildGrid   (grid.go:26)
│   ├─ multicol          → buildMulticol (multicol.go:33)
│   ├─ table display     → buildTable  (layout_tables.go:9)
│   └─ else              → buildBlock  (layout.go:1118)
└─ finishBuiltBox: relative offset, sticky tag, fixed stamp
```

Notable design: `buildInFlowDisplay` is shared between in-flow and
out-of-flow builds so abspos/fixed flex/grid containers get the correct
formatting context; a `figure{display:table;float:right}` heuristic
(`useBlockForTableDisplay`, `layout.go:1094`) routes table-*displayed but
non-tabular hosts to ordinary blocks.

## 4. Data & control flow

### Typical PDF body-object invocation

1. `internal/convert/convert.go:504` calls `layout.LayoutContext(ctx, root, state.bodyLayoutOpts(objectRender))` with `Width/Height` = content-box geometry from `hfGeom` (`convert/hf_geometry.go`), `Media:"print"`, the prepared stylesheet set, font registry, zoom, and the images fetch callback.
2. `layoutContext` (`layout.go:716`):
   - loads default faces via `pdf.LoadDefaultFaces()` unless overridden;
   - `resolveStylesForLayout` runs the cascade (and the `@container` second pass when size-container rules exist);
   - `newEngine` builds the engine with a `scale` from `zoomScale(opts.Zoom)`.
3. `finalizeResult` (`layout.go:784`): `eng.build(root, opts.Width, 0, 0)` recursively emits ops; `finalizeChrome` merges deferred background/border ops; box tree is flattened; `stampBoxTransforms` bakes CSS transforms/opacity into ops when any box needed it.
4. Back in `convert.go`, smart-shrinking may re-layout with an effective zoom (`convert.go:529`); relative link URIs resolved; external-link stripping.
5. `layout.PaintContext(ctx, run.doc, lres, paintOptions(state.geom))` (`convert.go:549`) paginates and paints. `Paint` (`paint.go:69`) sequence:
   - `paginateOps` → `snapCrossingTextOps` → `paginationFixpoint` (up to 10 iterations of avoidInside/beforeAlways/afterBreaks/rowsIntact/keepHeadingWithNext/orphansWidows) → `repeatTableHeaders`;
   - `splitCrossingRects` (rect ops fragmented at page boundaries, preserving `Op.ID`), `stripOrphanRowChrome`, `capTablePageBreaks` (border-gap sealing), `applyStickyPrint`;
   - `buildPagesAfterSplits` re-buckets ops into pages; `populateLocations` fills `Result.Locations`;
   - `paintPages` dispatches each op onto PDF content streams via `drawFill/drawStroke/drawLine/drawText/drawImage` (`paint.go:801+`).
6. `convert.go:555` collects headings via `collectObjectHeadings(root, lres, ...)`; `internal/outline.Lookup` maps heading nodes through `res.Locations`; links consume `res.Locations` for `#frag` anchors (`convert/links.go:42`).

### Canvas → PDF mapping

Layout uses a **y-down continuous canvas** with the origin at the top-left of
page 0's content area. `canvasToPDF` (`paint.go:789`) maps to PDF y-up:
`y_pdf = pageH - marginTop - y_canvas + pageIdx*contentH`. `hfGeom.pdfY`
(`convert/hf_geometry.go:27`) provides the same mapping for link/outline
destinations outside paint.

### Nested layouts (headers/footers, TOC)

- HTML header/footer: `internal/convert/hf.go:319` lays out the HF document
  once at content width; when placeholders (`[page]` etc.) are present it
  re-lays-out per page at draw time (`hf.go:426`) and paints via
  `PaintBandContext` clipped to the margin band (`hf.go:406-474`).
- TOC: `internal/convert/toc.go:202` lays out the generated TOC document; the
  page-count fixpoint re-paints into a scratch document via
  `layout.PaintContext(ctx, scratch, cloneResult(res), ...)` (`toc.go:164`).
- Page islands (benchmark mode): `internal/convert/page_islands.go:103` lays
  out each section with `layout.WithWorkspace` and `defer workspace.Release(res)`.

## 5. Cross-package dependencies

### Imports (what layout consumes)

| Import | Why |
|--------|-----|
| `internal/html` | `html.Node` tree walk, node types/attributes |
| `internal/css` | `Stylesheet`/`Rule`/`Selector`, `Specificity`, `MediaMatches`, `ParseInline`, `ParseColor`, `ParseLength`, `HasContainerRules` |
| `internal/pdf` | `pdf.Font` (metrics/glyph IDs), `pdf.FaceSet` (`LoadDefaultFaces`, `ResolveFamily`), `pdf.Registry` (opt-in `--font-path` faces), `pdf.Document`/`Page`/`Content` for paint |
| `internal/svg` | `svg.Rasterize` for inline SVG images (`layout_flow.go:53`) |
| stdlib | `context`, `errors`, `fmt`, `math`, `sort`, `strconv`, `strings`, `image`-adjacent (not for decode — that lives in pdf/imageout) |

Note: text shaping via `go-text/typesetting` lives in `internal/pdf`
(`shape_gotext.go`); layout itself measures via `Font.AdvanceInPoints` and
per-rune face fallback. `internal/layout` never imports `internal/convert`,
`internal/load`, or `internal/settings` — the dependency arrow points *out of*
convert into layout, never the reverse.

### Consumers (what depends on layout)

| Consumer | Usage |
|----------|-------|
| `internal/convert` | Body objects (`LayoutContext` + `PaintContext`), HTML HF (`LayoutContext` + `PaintBandContext`), TOC (`LayoutContext` + scratch `PaintContext`), page islands (`WithWorkspace` + `PaintContext`), links (`Result.Locations`), outline (`Result.Locations`) |
| `internal/imageout` | `LayoutContext` (`imageout.go:180,261`), then rasterizes `res.Ops` in `rasterPaintOrder` order (`imageout.go:496`) |
| `internal/convert/render` | Renders via the convert adapter; page ordering/copies are render's concern, layout supplies page counts through paint |
| `internal/outline` | `Location` projections are satisfied by `layout.ElementLocation` (`NodeRef`/`PageIndex`/`Bounds` at `layout.go:246-256`) |

### Import-direction rule

The dependency graph is strictly acyclic and layered:

```text
html ← css
  ↑      ↑
internal/layout ← internal/convert ← api.go / cmd/*
     ↓
internal/pdf, internal/svg
```

`internal/layout` sits one layer below `convert` and above `pdf`/`svg`. It
must never grow an import of convert/load/settings; new resource needs should
come through `Options` callbacks (`Images func(src) ([]byte, error)`), which
is exactly how convert injects the ACL-gated loader.

## 6. Design decisions & trade-offs

### Pure Go, no cgo, no browser

Layout is a hand-written CSS-subset engine. There is no WebKit/Chromium, no
HTML5 full parser, no full CSS2.1/CSS3 engine. The trade-off: controlled
report templates (invoices, statements, tables) render deterministically and
byte-stably, while arbitrary websites are explicitly out of scope
(`documentation/fidelity.md` — Tier 3 deferred). Every unsupported feature is
a conscious degrade, not an accident: unknown properties fall through
`applyIgnoredGroup`, unknown display values are ignored, JS is stripped at
load.

### Display-list architecture

The engine emits a **flat op list** in canvas space rather than a retained
box tree with per-node paint. This is what makes the *same* output consumable
by three backends (PDF `paint.go`, HF band `PaintBand`, raster
`imageout.go`) and what makes `--zoom`/smart-shrinking re-layout cheap. The
box tree is retained privately (not part of the exported `Result` contract
beyond `Locations`) so pagination can shift, split, and duplicate ops
(thead, sticky, rect fragmentation) without re-running layout.

### Two-phase build: measure then emit

Geometry precedes paint in the op stream: background/border ops are
**deferred** (`prependChrome`/`finalizeChrome`, `layout_chrome.go`) and
inserted *before* the content ops of a box, so paint order is
bg → borders → children. Sticky/fixed/transform boxes splice immediately
because they need op-index-stable stamps. This gives correct paint order
without a separate z-sort during build; the final z-order refinement (z-index
bands, positioned-over-in-flow) is resolved at paint time by `PaintOrder`.

### Single engine, three paint personalities

`Paint`, `PaintBand`, and the raster adapter share `PaintOrder` and
`StyleOf`/`FakeBoldFor` (`paint.go:428-481`) so fake-bold gating (Latin-only,
to avoid CJK streak artifacts), translucent-fill pre-composition, and stroke
min-width behave identically in PDF, HF, and PNG/JPEG. Pagination + fixed
stamps belong to `Paint` only; bands skip them.

### Pagination is a display-list rewrite, not a layout re-run

Page breaks, avoid policies, thead clones, sticky clamps, and rect splits are
implemented as **op/box transformations with a flow index** (`paint_flow.go`
`shiftFlowY` + per-page buckets), capped at 10 fixpoint iterations to bound
worst-case cost. Table rows never split; text ops move wholly at line level;
only rect-type ops fragment across pages. This is markedly cheaper than
re-layout per page and keeps `Result.Locations` consistent with the final
pagination via `populateLocations`.

### Determinism and byte stability

Fixed input + settings + metadata time produce repeatable output (the repo
requirement for reports). Layout contributes by: deterministic cascade
(specificity + source order, no hash iteration), interning styles
(`styleStore`), stable op `ID` assignment, and zoom scaling applied to style
lengths only (geometry stays in points).

### wkhtmltopdf work-alike compromises

- `--zoom` is applied in layout (`Options.Zoom` → `scalePt`) rather than as a
  page-size change; smart-shrinking multiplies user zoom (`convert.go:529`).
- Print media default: `convert.go:35-36` — PDF layout always uses `"print"`.
- HTML headers/footers are nested single-band documents clipped to margin
  bands, matching wkhtmltopdf's `--header-html` behavior.

## 7. Notable patterns & invariants

### Cascade invariants

- **Three-tier cascade** (`cascadeRaw`): UA sheet (specificity 0, order −1)
  → author sheets (media + `@container` + selector gated) → inline `style`
  attribute (spec `1<<maxIntShift`). `!important` is a separate layer above
  all normal declarations regardless of origin.
- **One winner map** per element (recycled `cascadeWins`), not six maps.
- **Inheritance is explicit**: a fixed `inheritableProps` table (20 groups)
  with closure-based copies — chosen over map lookups after inheritance was
  ~40% of alloc_objects on a 500-page PDF.
- **Custom properties** (`--*`) are merged parent→child, resolved with
  `resolveRawVars`, but never interned (they stay unique per element to avoid
  exposing mutability through the shared `*ResolvedStyle`).

### Style interning

`styleStore.intern` (`style.go:464`) canonicalizes identical
`ResolvedStyle`s to a single pointer per layout run (coarse discriminator key
+ full value compare). Boxes hold a `*ResolvedStyle` (~1.3 KB avoided per
box; table cells dominate box counts). Custom-property styles bypass
interning. Tested by `style_store_test.go`.

### Two-pass / fixpoint patterns

- **`@container` double cascade**: pass 1 without container sizes; measure
  size containers; re-cascade with the container map; one nested remount when
  container types change (`resolveStylesForLayout`, `layout.go:825`).
- **Pagination fixpoint**: up to 10 iterations over
  avoidInside / beforeAlways / afterBreaks / rowsIntact /
  keepHeadingWithNext / orphansWidows (`paint_pagination.go:490`).
- **TOC page-count fixpoint** (in convert) relies on `CloneResult` +
  scratch `PaintContext`.

### Op identity

Every op carries a monotonically increasing `ID` (`engine.add`,
`layout.go:628`). Rect fragmentation keeps the source `ID` on all fragments
(`splitCrossingRects`), so element → op-range ownership can be remapped after
splits (`remapBoxOpRanges`). Legacy/test-constructed ops get IDs at paint
time (`assignOpIDs`).

### Pooling & allocation discipline

The engine is deliberately single-threaded and reuses scratch storage:
`inlineItemPool`, `bfcPool`, `faceByStyle`/`faceByRune` caches, `imgCache`
(one decode per src per run), `Workspace` (display-list storage reuse for
page islands), per-page exact-capacity buckets in `pageBuckets`. Context
cancellation is checked at recursion boundaries (`checkContext`, cheap
enough to run per build call).

### Degrade rules

- Unknown property → `applyIgnoredGroup` (parse, discard).
- Unknown display/position values → ignored; invalid values leave the
  previous cascade value.
- `height:%` with indefinite containing block → treated as `auto`
  ("cyclic % honesty"), same for `width:%` in indefinite CBs.
- SVG images rasterize at ≤1024 px (`svgRasterMax`), SVG failures fall back
  to PNG/JPEG sniffing, then to no image.
- `border-collapse: collapse` parsed but modeled as spacing suppression
  (separate-border machinery, `tableSpacing`).

### Operator-policy extension points (small, deliberate)

`Options.PrintLinkUnderline` is the one opt-in post-cascade policy
(after-cascade `text-decoration:underline` for `a[href]`), mirroring the
CLI's `--print-link-underline`. `Options.Background` gates background fill
(`web.background`). `Options.DebugBoxes` outlines every box for golden tests.

## 8. Security considerations

`internal/layout` itself parses no network input and executes no untrusted
code, but it is the **processing core** that touches data loaded under the
ACL (`documentation/THREAT-MODEL.md`):

- **Image fetching** is indirect: layout never opens URLs; it calls the
  injected `Options.Images` callback (`layout_flow.go:31`), which
  `internal/convert` wires to the ACL-gated loader. SVG rasterization
  (`internal/svg.Rasterize`) runs on bytes that already passed the ACL and
  size caps (`svgRasterMax = 1024`).
- **HTML parsing** is allowlisted (`internal/html`): scripts and active
  content are dropped before layout; layout therefore never executes JS or
  fetches at CSS time.
- **Fonts** come from the bundled Liberation faces or the opt-in
  `--font-path`/`--use-system-fonts` registry (admin-controlled); font parsing
  lives in `internal/pdf` (which parses TTF/WOFF with bounded readers per the
  threat model §resources).
- **Bounded work**: pagination fixpoint caps at 10 iterations; rect
  fragmentation has `paginationGuardMax`; raster size is validated in
  `imageout`. `Workspace` release prevents unbounded op-buffer retention.
- **Byte-stability**: layout produces no random identifiers, so converted
  documents leak nothing about the host (relevant to report reuse).
- The threat model notes parsers run in-process and are the trust boundary;
  layout's own code paths (cascade map sizes, style interning) are pure
  Go memory operations bounded by document size and the op capacity estimate
  (`estimateOpCapacity`, `layout.go:919`).

## 9. Testing & verification

244 test functions across 62 files in `internal/layout`, plus layout-driven
golden fixtures in `internal/convert`.

### Core unit suites

| Test file | Coverage |
|-----------|----------|
| `layout_test.go` (1374) | Block stacking, widths/margins, margin collapse, padding/border box, text align/wrapping, `white-space:pre`, font em/inherit, cascade + inline style, link pseudo color, IPA glyph fallback, background fill, display:none, bullets, bold/underline, tables + colspan, image intrinsic sizing, paginate-and-paint, single page, debug boxes, `hr`, zoom, page-break parsing/application, boundary fill split, table row no-split |
| `flex_test.go` (878) / `grid_test.go` (873) | 24 flex + 21 grid tests: direction, wrap, justify/align, grow/shrink/basis, order, gaps; tracks, fr/minmax, auto-placement, dense, areas, spans |
| `sticky_test.go` (904) | 13 tests: overflow scrollport @0, no page clones, clamp top/CB limits, continuation pages, fixture-31 split fills preserve paint order |
| `transform_test.go` (349) | 9 tests: parse, origin, matrix math, stamping |
| `multicolumn_test.go`, `multicol_test.go` (246) | 5 tests: props, used count/width, span:all, lines don't straddle pages |
| `orphans_widows_test.go` (273) | 5 tests: Rule 3 with countable lines + geometric heuristic |
| `overflow_wrap_test.go` (200) | 5 tests: break-word/anywhere/keep-all |
| `pagination_thead_test.go` | thead repeat across pages |
| `table_rowspan_test.go`, `table_continuation_border_test.go`, `table_ref_stack_test.go`, `table_collapse_grid_test.go`, `table_empty_row_test.go`, `table_avoid_blank_test.go` | Table edge cases: rowspan occupancy, continuation borders, empty rows, avoid-blank bands |
| `style_store_test.go` (422) | Interning policy, pointer stability across chunks, shared vs distinct cascade results |
| `container_test.go` (287) | `@container` named/unnamed/or/not, nearest-wins, layout switch |
| `media_query_test.go`, `has_test.go`, `hlist_pseudo_test.go`, `pseudo_content` tests | Media queries, `:has`, pseudo-elements/content |
| `wiki_*` tests (infobox float, print chrome), `float_*` tests | Wiki-corpus-driven float behavior |
| `architecture_followup_test.go` (298) | Container-state equality, split-crossing identity remap, image size policy, context cancellation, flow-index maintenance, benchmarks |
| `fixture_bugs_test.go` (552), `fixture_render_regression_test.go`, `requested_fixture_regression_test.go` | Regression tests for golden fixture bugs |

### Golden fixtures

`internal/convert/golden_test.go` (TestGoldenCorpus, TestGoldenCorpusAllFixtures)
renders 57 fixtures in `testdata/golden/fixture-*.html` and asserts structural
output properties plus PNG baselines under `testdata/golden/assets`. Layout
subsystems are exercised by dedicated fixtures: floats (22/29/38), flex
(25/28/32/33), grid (28/32/34/35), multicol (39), transforms (40), sticky
(31), orphans/widows (30/37), thead repeat (23), position lite (26),
typography (18), invoices (01/02/03/16/21), certificates (47), contracts (46).

### Benchmarks

`internal/convert/benchmarks_test.go` (BenchmarkPDFPages, BenchmarkTemplatePages,
BenchmarkLiveMovieData, etc.) and layout-local benchmarks
(`architecture_followup_test.go` BenchmarkUsedImageSize,
BenchmarkDisplayListIdentity10kOps100Pages). CI runs `go test ./...`,
golangci-lint (`.golangci.yml`), and `make samples` regenerates the committed
`output/` PDFs/PNGs.

## 10. Known limitations, deferred items & open questions

Cross-referenced with `documentation/compatibility-matrix.md` (normative),
`documentation/fidelity.md`, `documentation/deferred.md`, and
`plans/phases/*`:

- **Positioning is "lite"**: absolute/fixed lack full CSS2.1 static-position
  rules and complex z-index stacking; fixed under a transformed ancestor is
  treated as absolute (`buildOutOfFlowIfPositioned`). Fixtures 26/28.
- **Sticky is print-scoped**: default scrollport = page content box; overflow
  boxes act as scrollports at offset 0; **no continuation-page clones**
  (deliberate, `sticky.go`, `TestStickyNotFixedReplication`).
- **Float lite**: simple exclusion; no shrink-beside for in-flow tables
  (tables always clear below floats); `float` blockifies table-internal
  displays (CSS2.1 §9.7).
- **Flex/grid**: no deep intrinsic multi-pass cycles; `%` basis cyclic sizing
  treats indefinite CB as auto; `subgrid` → ordinary grid; masonry stripped
  to dense auto-flow. Bootstrap/Tailwind/Chrome layout-test parity explicitly
  out of scope (`compatibility-matrix.md §2.7/§2.8`, plans
  `subplans-tier-2/flex-grid-full.md`).
- **Multicol**: report lite only — no column rules, no L2 integer spans,
  no overflow columns; `break-*:column` aliased to page breaks; column boxes
  do not straddle pages (§2.9, `plans/phases/tier-2-pending-3/multicol.md`).
- **Tables**: `border-collapse: collapse` not truly modeled (spacing
  suppression only); `table-layout: fixed` parsed but auto-only; `<caption>`
  and `table-column(-group)` not rendered; rowspan limited (`table_rowspan_test.go`
  covers the supported subset).
- **Overflow is not clipping**: `overflow` parsed only for sticky scrollport
  selection; no general paint clipping.
- **Text**: no full hyphenation; justification expands inter-word spaces only
  (max 1em); `text-transform` variants limited; shaping fallback keeps
  combining marks after the base for non-Latin (`documentation/fonts.md` —
  honest shaping limits); CJK relies on `--font-path` faces via the registry.
- **Orphans/widows**: Rule 3 applies only when line boxes are countable; the
  geometric short-block heuristic remains for nested/uncountable cases
  (fixtures 30/37).
- **No vertical writing modes beyond parsing** (`writing-mode` parsed into
  `ResolvedStyle.WritingMode`, layout assumes horizontal-tb).
- **HTML HF band**: single-page, clipped; taller content is clipped rather
  than paginated (documented in `hf.go`).
- **Zoom interacts with absolute positioning/viewport percentages** in
  corners (fixed uses `opts.Height` unscaled — `resolveAbsY`); known edge,
  not yet a tracked issue.

Open questions worth a follow-up review:

1. `height:%` resolution depends on deferred `absCBHeights` recorded during
   the in-flow pass; nested deferred abs-positioned chains inside `overflow`
   containers are not covered by tests.
2. `paginationFixpoint` iteration cap (10) is a heuristic bound; adversarial
   documents could in principle oscillate (guarded, but unproven).
3. Style interning skips custom-property styles — multi-page reports using
   many `--*` values pay per-element allocation (documented trade-off).

## 11. Related documents

- Pipeline overview & package map: [`../architecture.md`](../architecture.md)
- Support matrix (normative property-by-property status, §2.5–2.9, §3):
  [`../compatibility-matrix.md`](../compatibility-matrix.md)
- Fidelity tiers and claim language:
  [`../fidelity.md`](../fidelity.md)
- Font discovery, Type0/CJK, honest shaping limits:
  [`../fonts.md`](../fonts.md)
- Security model and ACL (layout consumes ACL-gated resources):
  [`../THREAT-MODEL.md`](../THREAT-MODEL.md)
- Deferred items and product priorities:
  [`../deferred.md`](../deferred.md)
- Phase plans for layout/pagination:
  [`../../plans/phases/phase-04-html-css-layout.md`](../../plans/phases/phase-04-html-css-layout.md),
  [`../../plans/phases/phase-05-pagination-print.md`](../../plans/phases/phase-05-pagination-print.md),
  [`../../plans/phases/phase-16-invoice-css.md`](../../plans/phases/phase-16-invoice-css.md),
  [`../../plans/phases/phase-17-broader-css.md`](../../plans/phases/phase-17-broader-css.md),
  [`../../plans/phases/phase-18-pagination-polish.md`](../../plans/phases/phase-18-pagination-polish.md)

### Sibling architecture deep-dives (this directory)

| Doc | Domain |
|-----|--------|
| [01-entrypoints-cli.md](01-entrypoints-cli.md) | CLI binaries + multi-object grammar |
| [02-library-api.md](02-library-api.md) | Public library API (`Converter`/`ImageConverter`) |
| [03-settings.md](03-settings.md) | Settings system & errors |
| [04-load.md](04-load.md) | Load layer (URLs, ACL, I/O) |
| [05-html-parser.md](05-html-parser.md) | HTML tokenizer & tree |
| [06-css.md](06-css.md) | CSS subsystem (parse, selectors, cascade) |
| **07-layout.md** | **This document — layout engine & line breaking** |
| [08-convert-pipeline.md](08-convert-pipeline.md) | PDF/Image job orchestration, HF/TOC/outline/links |
| [09-pdf-writer.md](09-pdf-writer.md) | PDF 1.4 writer, fonts, subsets |
| [10-imageout-svg.md](10-imageout-svg.md) | PNG/JPEG raster path & SVG rasterization |
