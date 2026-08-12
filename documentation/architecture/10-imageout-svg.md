# Image output (PNG/JPEG) & SVG raster

## 1. Responsibility & position in the pipeline

`internal/imageout` is the **image-mode engine** behind the `gowkhtmltoimage`
binary and the `ImageConverter` library API. Where `internal/convert` +
`internal/pdf` write a multi-page PDF, `imageout` renders the **same shared
upstream pipeline** — load → parse → style → layout — and then **paints the
layout display list into an in-memory `image.NRGBA` canvas** and encodes it as
a single PNG or JPEG image. It exists because wkhtmltopdf ships a sibling
`wkhtmltoimage` tool and this project mirrors that surface.

`internal/svg` is a small (450-line) satellite package that converts **SVG
images referenced by `<img src="*.svg">`** into raster PNG bytes so the
layout/paint stages can treat every image uniformly as PNG/JPEG pixels. It is
the one place the project calls into a third-party rasterizer
(`github.com/tdewolff/canvas`), which is allowlisted in the Makefile.

Positioning:

```
HTML (file | URL | stdin | inline SetBody)
  │
  ├─ cmd/gowkhtmltopdf ──► internal/convert ──► internal/pdf   (PDF 1.4 output)
  │
  └─ cmd/gowkhtmltoimage ──► internal/app.RunImage
                              └─► internal/imageout.RunRequest         (PNG/JPEG output)
                                    │  shared phases: load → parse (html) → style (css) → layout
                                    ▼
                              layout.Result (display list of Op)
                                    │  imageout.rasterizeContext
                                    ▼
                              image.NRGBA → encode → req.Output
```

The two modes share: `internal/load` (fetch/ACL), `internal/html` (parser),
`internal/css` (style sheets), `internal/convert/prepare` (document
preparation: stylesheet collection + `@font-face` merge), `internal/layout`
(style resolution + box layout + display list), `internal/pdf` (font
discovery, shaping, glyph contour geometry), and the render lifecycle seam
`internal/convert/render` (`RenderObjects` → `Assemble` → `Finalize`).
Everything **after the display list diverges**: PDF mode paginates and emits
content streams; image mode rasterizes `layout.Op` ops into pixels with a
cancellation-aware, single-page canvas.

**Key invariant:** text metrics, glyph forms, and paint semantics stay *on
one table* with PDF mode. Image text uses the same embedded TrueType faces and
the same `pdf.ShapeRun` text shaping, and paint-order / fake-bold / stroke
width decisions come from `layout.PaintOrder`, `layout.StyleOf`, and
`layout.FakeBoldFor` — so a report screenshot and its PDF match within raster
resolution limits. This is the "P5-01 / P5-02" consolidation from the
architecture review (see §6 and §10).

## 2. Package / file map

### internal/imageout

| File | Lines | Responsibility |
|------|------:|----------------|
| `doc.go` | 11 | Package contract: shared pipeline + TTF outline raster + shaping parity |
| `imageout.go` | 1521 | Render options, layout orchestration, supersampled rasterization, paint dispatch, image decode/scale caches, CLI/library adapters, format encode |
| `ttfraster.go` | 419 | TTF→bitmap glyph rasterizer (outline flatten, supersampled coverage AA, glyph atlas) |
| `imageout_test.go` | 512 | White-box: solid/transparent/crop/text/image rendering, scaler & downscale math, encode formats, CLI end-to-end, smart width |
| `fontface_test.go` | 231 | `@font-face` local custom font + ACL deny behavior through the Run path |
| `raster_test.go` | 164 | Paint-order parity with PDF layer policy, context cancellation, AA uniqueness, advance-width match with layout |
| `baseline_test.go` | 98 | Baseline stability of rasterized glyphs (no per-letter vertical bobbing) |
| `debug_test.go` | 42 | ASCII-art debug dump helper for image regions |

Total: ~2,998 lines.

### internal/svg

| File | Lines | Responsibility |
|------|------:|----------------|
| `raster.go` | 274 | `Rasterize` via tdewolff/canvas only; size parsing (viewBox/width/height), DPMM resolution, panic recovery |
| `raster_test.go` | 79 | Rect/path rasterization, not-SVG and broken-SVG error behavior |
| `wiki_logo_smoke_test.go` | 97 | Host-cached Wikipedia logo SVGs (gradients/groups/clipPaths/arcs) sanity rasterization |

Total: 450 lines.

Related files owned by other domains (consumed, not part of this package):
`internal/layout/layout_flow.go:53` (calls `svg.Rasterize` during image
resolution), `internal/layout/mnd_const.go:62` (`svgRasterMax = 1024`),
`internal/convert/render/pipeline.go` (lifecycle seam), `internal/app/image.go`
(`RunImage`), `internal/pdf/shape_gotext.go` (`ShapeRun`/`ShapeTextFont`),
`internal/pdf/glyph.go` (`GlyphContours`, `FlattenContour`, `ContourBounds`),
`internal/pdf/faces.go` (`DefaultFont`), `internal/pdf/registry.go`
(`ScanFontDirs`, `DefaultSystemFontDirs`).

## 3. Key types, functions & entry points

### Public entry points (imageout)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `RenderOptions` | `imageout.go:101` | One-call render config: `Width`, `Height`, `Font`, `Registry`, `Sheets`, `Media`, `Images` (fetch func), `Background`, `Transparent`, `Crop`, `SmartWidth`, `PrintLinkUnderline` |
| `Render(root, opts)` | `imageout.go:121` | Convenience wrapper over `RenderContext` with `context.Background()` |
| `RenderContext(ctx, root, opts)` | `imageout.go:128` | The core library render: validate ctx → default font → `layoutResult` → `rasterizeContext` → `applyCrop` |
| `Run(ctx, cmd, log)` | `imageout.go:1054` | CLI-facing adapter (P1-1): resolves format, validates request against a discard sink, opens `cmd.OpenOutput()`, delegates to `RunRequest` |
| `RunRequest(ctx, req, log)` | `imageout.go:1094` | CLI-independent engine seam on a `convert.Request`; used by `api.go` hook `convertHooks.executeImage` (`api.go:720`) and `ImageRequest` (`api.go:903`) |

### Layout & viewport logic

| Symbol | Location | Purpose |
|--------|----------|---------|
| `layoutResult` | `imageout.go:170` | Picks smart-width vs fixed viewport path |
| `layoutSmartWidth` | `imageout.go:244` | Re-layout loop growing viewport ×1.5 while `contentWidthPx(res) > viewport+0.5`; caps at `maxSmartViewport` (4096) and `maxSmartWidthLayouts` (8) passes |
| `contentWidthPx` | `imageout.go:281` | Rightmost painted edge of the display list (ignores `OpLinkURI` annotations) |
| `layoutOptions` | `imageout.go:219` | Maps `RenderOptions` → `layout.Options`; converts CSS px → points via `cssPxToPt = 0.75` (96 dpi) and applies minimum `Height` |
| `maxHeight` | `imageout.go:301` | Canvas height = max of content height and requested `--height` |
| `cropRect` | `imageout.go` | `settings.CropSettings` → `image.Rectangle`; unset values default to −1 → zero rect (no crop) |
| `mediaFor` | `imageout.go` | Resolves `"screen"`/`"print"` via `settings.ResolveMedia` (P1-4); image mode defaults to `"screen"`, `--print-media-type` forces print |

### Rasterization & resource budgets

| Symbol | Location | Purpose |
|--------|----------|---------|
| `rasterizeContext` | `imageout.go:328` | Allocates a `rasterSS`× supersampled NRGBA canvas (white or transparent), paints ops in `layout.PaintOrder`, box-filters down to final size |
| `rasterDimension` / `validateRasterSize` | `imageout.go:401/422` | Dimension guards: width/height ≤ 16,384; ≤ 64M pixels; ≤ 256 MiB backing bytes |
| `supersamplePixPool` | `imageout.go:321` | `sync.Pool` recycling of the large supersample pixel buffer across renders |
| `paint` (dispatch) | `imageout.go:755` | Switch over `layout.OpKind`: `OpFillRect`/`OpStrokeRect`/`OpLine`/`OpText`/`OpBullet`/`OpImage`; `OpLinkURI` paints nothing |
| `paintText` | `imageout.go:846` | Fractional-baseline text draw + fake-bold second pass (Latin-only gate in `layout.FakeBoldFor`) |
| `paintImage` | `imageout.go:877` | Decodes via per-run cache and draws at natural size (`draw.Draw` fast path) or via `scaleNearest` |
| `downscaleBox` / `downscaleBox2` | `imageout.go:642/698` | Exact average of `rasterSS×rasterSS` blocks (specialized 2×2 path) to final CSS-pixel size |
| `scaleNearest` (+ NRGBA/generic) | `imageout.go:967/975/1011` | Stdlib-only nearest-neighbour scaler (Go 1.26 removed `image/draw` scalers) |
| `rasterImageCache` / `decode` / `scaledImage` | `imageout.go:511/555/621` | Per-run decode + scale cache with byte/pixel budgets (FNV-1a key + `bytes.Equal` collision check; JPEG/PNG decoder separated) |
| `validateImageInput` | `imageout.go:450` | Encoded/decode guards before decoding: ≤32 MiB encoded, dims ≤16,384, ≤16M pixels, ≤128 MiB decode working set |

### CLI/engine adapters

| Symbol | Location | Purpose |
|--------|----------|---------|
| `imagePipeline` | `imageout.go:1152` | Adapter implementing `convert/render.Pipeline` (`RenderObjects`/`Assemble`/`Finalize`) with image-specific state |
| `RenderObjects` | `imageout.go:1162` | `prepareImageDocument` → fetch func → `RenderContext`; stores the `image.Image` |
| `Finalize` / `writeEncodedOutput` | `imageout.go:1209/1215` | Resolve format, composite transparent canvas onto white for JPEG (`onWhite`), encode, write to `req.Output` |
| `prepareImageDocument` | `imageout.go:1247` | Resolves media + SimplifyDOM profile, runs `convert.PrepareDocument` with `defaultViewportW/H = 768×576` |
| `makeImageFetcher` | `imageout.go:1288` | Wraps `prep.Resources.Fetch` with the `--no-images` gate and a bounded byte cache (64 fetches / 32 MiB) |
| `fontRegistry` | `imageout.go:1321` | Builds `pdf.Registry` from global `FontPaths` + system dirs (`ScanFontDirs`); nil when nothing to scan |
| `imageLoadGlobal` | `imageout.go` | ACL merge: `Image.Load` ⊕ `Global.Load.Allow` / `EnableLocalFileAccess` before `load.NewLoader` |
| `firstObject` | `imageout.go` | Picks the first page-like object; warns and ignores extra pages/TOC (single-image mode) |
| `resolveFormat` / `encode` / `onWhite` | `imageout.go` | PNG vs JPEG selection (`--format` wins, else `.jpg/.jpeg` extension, else PNG); JPEG quality clamp 1–100; JPEG transparency → warn + white composite |

### Glyph rasterizer (ttfraster.go)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `ttfDrawString` | `ttfraster.go:28` | Draws a shaped run (`pdf.ShapeRun`) at fractional baseline; advances from `run.Advances × pxPerPt` so image text width == layout width |
| `glyphAtlas` | `ttfraster.go:89` | Per-raster mutex-protected cache of glyph bitmaps keyed by `(face, pxSize, rune)`; cap `maxGlyphCache = 4096` with ~half-drop eviction (P5-05: no package-global mutable cache) |
| `drawGlyphAA` | `ttfraster.go:139` | Manual source-over blend of glyph alpha into NRGBA; one round per glyph keeps stems on a stable pixel grid |
| `rasterGlyph` | `ttfraster.go:207` | Flattens `pdf.GlyphContours` (more steps at small sizes), builds edge list, supersamples coverage (6 default, 8 for small text), returns alpha bitmap + origin offsets |
| `rasterGlyphAlpha` | `ttfraster.go:279` | Per-pixel coverage via active-edge parity (`pointInActiveEdges`) |
| `flattenGlyphContours` / `makeGlyphEdges` / `activeEdges` | `ttfraster.go:324/375/357` | Polygon flattening + scanline edge structure (y-min/max, x-at-min, dxdy) |

### SVG raster (internal/svg)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `Rasterize(data, maxSide)` | `raster.go:45` | Public: sniff SVG (BOM/`<svg`/`<?xml`), default `maxSide` 512, then `rasterizeCanvas` |
| `rasterizeCanvas` | `raster.go:67` | `canvas.ParseSVG` → `rasterizer.Draw` at computed DPMM → PNG bytes; `defer recover()` turns canvas panics into `errCanvasPanic` |
| `svgCSSPixelSize` | `raster.go:131` | Intrinsic CSS-pixel size from root `viewBox` (win) or `width/height` attrs, capped at `maxSide` (≤4096), min 100 default |
| `rootSVGSize` / `svgSizeAttrs` | `raster.go:171/204` | XML-decode only the first `<svg>` start tag; lenient decoder for malformed markup |
| `canvasDPMM` | `raster.go:108` | Resolution mapping (canvas mm → CSS px, 96 dpi fallback) so the longer edge fits `target` |

## 4. Data & control flow

### 4.1 CLI path (gowkhtmltoimage)

1. `cmd/gowkhtmltoimage/main.go` → `cli.Parse(argv, cli.ModeImage)`; handles
   `ErrHelp` / `ErrVersion` / `ErrLicense`, then `app.RunImage` with a signal
   context (`os.Interrupt`, `SIGTERM`).
2. `internal/app/image.go` `RunImage` validates first:
   `convert.NewImageRequest(cmd.Global, cmd.Image, cmd.Objects, io.Discard)` +
   `ValidateImage()` (no output truncation on bad input), then owns output
   opening and calls the request engine.
3. `imageout.RunRequest`:
   - receives the already-normalized format and explicit output writer from
     `app.RunImage`;
   - validates the request again at the engine seam;
   - loads, lays out, rasterizes, and encodes into `req.Output`.
4. `RunRequest` (`imageout.go:1094`):
   - `req.ValidateImage()` again (seam contract);
   - `firstObject` — warns and ignores additional page objects and TOC
     (image mode renders exactly one page);
   - `load.NewLoader(imageLoadGlobal(req.Global, *req.Image))` — ACL =
     Image.Load merged with Global.Load (`--allow` / `--enable-local-file-access`);
   - `pdf.DefaultFont()` + `fontRegistry` (system + `--font-path`);
   - builds `imagePipeline` and runs `renderpipeline.Run` (the shared
     RenderObjects → Assemble → Finalize lifecycle with ctx checks between stages).

### 4.2 RenderObjects (single-page render)

```
RenderObjects
 ├─ prepareImageDocument(ctx, loader, obj, global, imgSet, registry, log)
 │    ├─ mediaFor(global, image, obj)          → "screen" default / print override
 │    ├─ SimplifyDOM profile resolution (convert.SimplifyDOMEnabled/Profile)
 │    └─ convert.PrepareDocument(ctx, loader, obj.Page, obj.Load, registry,
 │           PrepareOptions{ViewportW:768, ViewportH:576, MediaType, SimplifyDOM, ...})
 │           → load.Resource → html.ParseDocument → CollectSheets → MergeFontFaces
 │           → *PreparedDocument{Root, Sheets, Resources, Registry}
 │    (prep.Resource.Skip  →  error "load-error policy is skip; nothing to render")
 ├─ makeImageFetcher(ctx, imgSet, prep, cache)
 │    └─ --no-images gate (errImagesDisabled) → prep.Resources.Fetch (bounded cache)
 ├─ printLinkUnderline = Image|Global|Object Web flag OR
 └─ RenderContext(ctx, prep.Root, RenderOptions{Width, Height, Font, Registry,
      Sheets, Media, Images, Background, Transparent, Crop, SmartWidth, ...})
      ├─ layoutResult → layoutSmartWidth (repeat 1.5× until content fits, ≤8 passes)
      │      └─ layout.LayoutContext(ctx, root, layoutOptions(...)) → layout.Result{Ops, Width, Height}
      ├─ rasterizeContext(ctx, res, maxHeight(res,opts), Transparent)
      │      ├─ dims: width/height × (ptToPx × rasterSS) with budgets (validateRasterSize)
      │      ├─ pixBuffer from sync.Pool; NRGBA canvas (white, or alpha-0 when Transparent)
      │      ├─ glyph atlas + rasterImageCache per run
      │      ├─ for i := range layout.PaintOrder(res.Ops): paint(img, &op, pxPerPt, atlas, cache)
      │      │     └─ OpText/OpBullet → ttfDrawString (pdf.ShapeRun → drawGlyphAA, fake-bold pass)
      │      │     └─ OpImage    → rasterImageCache.decode → fast path / scaleNearest
      │      │     └─ OpLinkURI  → skip (annotations do not paint)
      │      └─ downscaleBox(img, rasterSS) → final CSS-pixel image
      └─ applyCrop(img, opts.Crop) → re-origin SubImage to (0,0)
```

`Assemble` is a no-op for image mode (no multi-page assembly). `Finalize`
runs `writeEncodedOutput`: format resolve → if JPEG and `Transparent` warn
"`--transparent ignored for JPEG output`" and composite `onWhite` → `encode`
with `limitedImageBuffer` (bounds encoded bytes to 32 MiB) → `req.Output.Write`.

### 4.3 Library path

`api.go` `ImageConverter.Convert` builds `convert.NewImageRequest`, then
`convertHooks.executeImage` (`api.go:720`) sets `req.Output` to a `bytes.Buffer`
and calls `imageout.RunRequest` directly — the CLI layer is not involved.
`ImageRequest` (`api.go:903`) uses the same `RunRequest` seam. `Render` /
`RenderContext` are also independently callable on a parsed `html.Node` with an
`Images` fetch function (used by tests and the `examples/image` demo).

### 4.4 SVG flow (inside layout)

`internal/layout/layout_flow.go:53` `resolveImage`: for each `<img src>`,
fetch bytes once (per-run cache), then:

```
if png, pw, ph, err := svg.Rasterize(data, svgRasterMax /*1024*/); err == nil {
    → ref.data, w, h = png, pw, ph     // SVG → PNG at intrinsic size
} else if w, h, jpeg, ok := imageDims(data); ok {
    → ref.w, h, isJPEG = w, h, jpeg    // PNG/JPEG dimensions from headers
}
```

So SVG reaches `imageout` only as already-rasterized PNG bytes. `svg.Rasterize`
is a **one-way dependency of layout**, not of imageout — PDF mode benefits too
(SVG-as-image in PDF pages). Any rasterization failure is a clean `error`
("no image"): the `<img>` is skipped at paint time, never a process crash.

## 5. Cross-package dependencies

### Imported by internal/imageout

| Import | Why |
|--------|-----|
| `internal/cli` | `cli.Command`, `OpenOutput`, parse results in the `Run` compatibility adapter |
| `internal/convert` | `Request`, `NewImageRequest`, `PreparedDocument`, `PrepareDocument`, `SimplifyDOM*` |
| `internal/convert/render` | The shared `Pipeline` lifecycle (`renderpipeline.Run`) |
| `internal/css` | `*css.Stylesheet` plumbing into layout |
| `internal/html` | `html.Node` tree from `PrepareDocument` |
| `internal/layout` | `LayoutContext`, `Options`, `Result`, `Op`, `PaintOrder`, `StyleOf`, `FakeBoldFor` |
| `internal/line` | Structured log emission (`line.Emit`, severities) |
| `internal/load` | `load.NewLoader` / policy application |
| `internal/pdf` | `Font`, `Registry`, `DefaultFont`, `ShapeRun`, `GlyphContours`, `FlattenContour`, `ContourBounds`, `AdvanceInPoints`, `ScanFontDirs` |
| `internal/settings` | `PdfGlobal`, `ImageGlobal`, `Web`, `LoadGlobal`, `LoadPage`, `PdfObject` |
| stdlib `image`, `image/color`, `image/draw`, `image/png`, `image/jpeg` | Canvas, compositing, encoding — **no cgo, no external raster engine** |

### Imported by internal/svg

`github.com/tdewolff/canvas` + `github.com/tdewolff/canvas/renderers/rasterizer`
(sole SVG render path, allowlisted via `//nolint:depguard` and Makefile), plus
stdlib `encoding/xml`, `image/png`. **No imageout or layout import** — layout
imports svg, making svg the lower layer.

### Import direction rule

```
gowkhtmltopdf (root api.go)
   └─► internal/app ──► internal/imageout ──► internal/convert/render
                       │      └─► internal/convert ──► internal/convert/prepare
                       │              └─► internal/layout ──► internal/svg
                       │                      └─► internal/pdf ──► internal/line
                       └─► internal/load, internal/css, internal/html, internal/settings
```

`imageout` never imports `internal/convert/render` internals beyond the
`Pipeline` interface (state stays in the adapter). `svg` never imports
anything internal — it is a leaf. The rule is: **domain arrows point inward
toward the leaf packages (`pdf`, `line`, `svg`), and both output engines sit
at the same level above `convert`**, consuming a common request/prepare/layout
contract (the P1-1 engine-seam goal).

## 6. Design decisions & trade-offs

1. **Shared pipeline, divergent sink.** The biggest structural decision: image
   mode reuses `load → html → css → layout` verbatim and only replaces the
   paint/write tail. Cost: `ImageGlobal` must map onto `PdfGlobal`-shaped
   settings (e.g. `Image.Load` + `Global.Load` ACL merge, `mediaFor` duplicating
   object/global web flags). Benefit: one layout engine, one set of golden
   fixtures, and screenshots that track the PDF.

2. **Pure-Go raster text — no FreeType, no hinting.** The stdlib has no font
   hinting, so `imageout` rasterizes TrueType **outlines itself**
   (`internal/pdf` provides contours; `ttfraster.go` flattens, scanline-fills
   with supersampled coverage). Trade-off: no hinting means glyph edges rely on
   `rasterSS = 2` supersampling + box-filter downscaling to stabilize small
   text. Larger glyphs get sharper coverage (subsample 6, or 8 below 16px).
   Documented in `fidelity.md` as "TTF outline AA, 2× supersample — shipped".

3. **Paint semantics on one table (P5-01).** `layout.StyleOf`/`FakeBoldFor`
   drive both PDF and raster stroke width, fake-bold, and order (`PaintOrder`);
   the only deliberate divergence is **fill alpha**: PDF pre-composites
   translucent fills against white paper, raster keeps raw `Op.Alpha` and
   `draw.Over` so `--transparent` can show through. The comment in `paint`
   (`imageout.go:755`) calls this out explicitly.

4. **Supersample-then-box-filter** instead of analytic AA for everything:
   simple, exact, and good enough for reports; cost is 4× the pixel buffer,
   mitigated by the `sync.Pool` — but the pool holds one large buffer per
   render, so concurrent large renders each allocate (bounded by caps).

5. **Aggressive resource budgets** (`rasterDimension`, `validateRasterSize`,
   image decode limits, cache caps, bounded encode buffer). Rationale: image
   mode is URL-reachable in web deployments; unbounded dimensions or decode
   work would be a memory-DoS vector (mirrors `THREAT-MODEL.md` subresource
   policy).

6. **Nearest-neighbour scaling for `<img>`** (Go 1.26 removed `image/draw`
   scalers). Deliberate: stdlib-only, deterministic, cheap. Trade-off:
   downscaling large photos looks blocky vs. a filtered scaler — acceptable
   for report screenshots (logos/grids are typically paint-at-natural-size).

7. **canvas as the *sole* SVG path.** No second rasterizer, no ImageMagick
   shell fallback — otherwise `svg` would grow an entire second render pipeline
   or reintroduce a native dependency. Accepts that exotic SVG may fail
   (cleanly, via `errNotSVG`/`errCanvasPanic`), in which case the `<img>` is
   skipped. `//nolint:depguard` + Makefile comment make this an explicit,
   auditable exception to the no-third-party-raster rule.

8. **Canvas panic containment.** Unknown paths in tdewolff/canvas can panic;
   the `defer recover()` in `rasterizeCanvas` converts that into `errCanvasPanic`
   so one bad `<img src="bad.svg">` never kills a web process.

9. **`Assemble` no-op isolates PDF-specific assembly.** Multi-page assembly
   (TOC ordering, copies/collate, HF) remains in `convert`; the render seam
   still runs it for uniformity and future reuse (P5-02).

## 7. Notable patterns & invariants

- **Policy-in-one-place loading (P2-07):** full load policy (proxy, ACL,
  local-access) is merged before `NewLoader`; no post-construction field pokes
  on `Loader`. `imageLoadGlobal` ORs `Image.Load` with `Global.Load`.
- **Validate-before-open:** both `Run` and `app.RunImage` validate the request
  against `io.Discard` before `cmd.OpenOutput()` so a bad request never
  truncates the user's chosen output path.
- **One quiet bit (Policy A):** `--quiet` lives on `Global.Quiet`; image mode
  honors it by discarding the log writer.
- **Per-run, capped caches:** glyph atlas (4096 entries, half-drop eviction),
  decoded-image cache (64 entries / 32 MiB raw / 64 MiB mem), scaled cache
  (128 entries / 64 MiB), image fetch cache (64 fetches / 32 MiB) — no
  package-global mutable state (P5-05), so concurrent `Render`s are safe.
- **Fractional baselines, single rounding:** `baseX/baseY` stay fractional;
  glyph origins round once per glyph → shared baseline, no per-letter bobbing
  (regression test `TestGlyphBaselineStable`).
- **Bounded encode:** `limitedImageBuffer` caps encoded output at 32 MiB.
- **Format sniffing hierarchy:** `--format` flag → output extension (`.jpg`/
  `.jpeg`) → PNG default; `--transparent` + JPEG warns and composites white.
- **Smart width:** default ON (`SmartWidth: true` in `DefaultImageGlobal`),
  1.5× growth, 4096px and 8-pass caps; return value is the last layout even
  when still overflowing (explicit bounded fallback).
- **Skip-on-failure image policy:** bad fetches/decode/SVG are logged-warn
  level at most and skipped; `--no-images` (`web.images=false`) returns a
  static error without crashing.
- **Extension points:** intentionally small — the library API path is
  `ImageConverter` + `ImageSettings` dotted `Set`, and the engine seam is
  `convert.Request` + `RunRequest`; there is no plugin system.

## 8. Security considerations

- **Local-file ACL** (`documentation/THREAT-MODEL.md`): local access denied
  unless `--enable-local-file-access` / settings; `--allow` prefixes whitelist
  paths. `imageLoadGlobal` merges `Global.Load` ACL into `Image.Load` so the
  image binary cannot bypass the PDF binary's policy via the loader.
- **Subresource fetching:** images, stylesheets, fonts fetched via
  `prep.Resources.Fetch` (loader ACL applies); fetcher cache bounded
  (64 fetches / 32 MiB); layout's `resolveImage` caches one ref per src and
  stores a nil sentinel on failure (no re-fetch loops).
- **Decode hardening:** `validateImageInput` prevents oversized PNG/JPEG
  decode (dims ≤ 16,384; ≤ 16M pixels; ≤ 128 MiB working set; ≤ 32 MiB
  encoded). The raster canvas itself is capped (≤ 64M pixels, ≤ 256 MiB).
- **Malformed SVG = clean error, not panic:** `defer recover()` in
  `rasterizeCanvas`; `errNotSVG` sniffs before handing bytes to canvas.
- **No shell/exec anywhere:** SVG and image conversion never spawn a process
  (documented explicitly in `svg/doc.go` and the Makefile allowlist), and
  `--no-images` is the hard opt-out for hostile documents.
- **JPEG transparency:** `--transparent` is honored for PNG; for JPEG the
  engine warns and composites onto white (JPEG has no alpha channel).

## 9. Testing & verification

### internal/imageout (white-box, `//nolint:testpackage`)

| Test | File | Verifies |
|------|------|----------|
| `TestRenderSolidColor`, `TestRenderTransparent`, `TestRenderCrop` | `imageout_test.go` | Exact pixel colors; alpha-0 background under `--transparent`; crop re-origin + pixel parity |
| `TestRenderText`, `TestRenderImageDataURI` | `imageout_test.go` | Text ink present; `<img data:…>` renders at natural size |
| `TestScaleNearest*`, `TestDownscaleBoxUsesExactNRGBAAverages` | `imageout_test.go` | Scalers correct; box filter produces exact channel averages |
| `TestEncodeFormats`, `TestEncodeJPEGQualityChangesSize` | `imageout_test.go` | PNG/JPEG encode round-trip; quality changes compressed size |
| `TestRunEndToEnd` | `imageout_test.go` | Full CLI: `--width`/`--format`/`--quality` reach decoded output |
| `TestSmartWidth` | `imageout_test.go` | 1.5× grow to 1536; fixed stays 1024; default width 1024 |
| `TestRasterPaintOrderMatchesPDFLayerPolicy` | `raster_test.go` | `layout.PaintOrder` z-index policy applied identically |
| `TestRenderContextHonorsCancellation` | `raster_test.go` | Pre-cancelled ctx → `context.Canceled` |
| `TestTTFRasterAntiAliased` | `raster_test.go` | AA produces >10 unique grey levels (vs old 5×7 bitmap) |
| `TestTTFAdvanceMatchesLayoutWidth` | `raster_test.go` | Ink span tracks `AdvanceInPoints × ptToPx` (metrics parity) |
| `TestGlyphBaselineStable` | `baseline_test.go` | Non-descender bottoms within 1px (no bobbing) |
| `TestFontFaceLocalUsesCustom`, `TestFontFaceACLDeny` | `fontface_test.go` | `@font-face` local file registers (ACL-allowed) / registers with warning (denied) through the real Run path |
| `TestDebugDataURIRegion` | `debug_test.go` | ASCII dump helper for image regions |

### internal/svg (black-box `svg_test`)

| Test | File | Verifies |
|------|------|----------|
| `TestRasterizeRect`, `TestRasterizePath` | `raster_test.go` | Canvas-only path: correct size + PNG magic bytes |
| `TestRasterizeNotSVG` | `raster_test.go` | Non-SVG → `errNotSVG`, empty result |
| `TestRasterizeBrokenSVG` | `raster_test.go` | Malformed path → clean error (panic containment); tolerated success must still be a real image |
| `TestRasterizeWikiWordmark`, `TestRasterizeArcPath` | `wiki_logo_smoke_test.go` | Host-cached Wikipedia logos (gradients/groups/clipPaths); arc `path d` — nonzero pixel checks |

### Integration / golden

- `make samples` (`Makefile`) generates committed `output/fixture-01-simple-invoice.png`
  and `output/fixture-21-detailed-report.png` via `gowkhtmltoimage`; the docs
  site showcase (`docs/assets/*.png`, `frontend/`) is built from these and the
  `docs/assets/` fixtures compare image-mode screenshots against Chrome/WK
  thumbnails (per `documentation/samples.md`).
- `TestFontFace*` use `collectFontLayout` to replay the *same* load + sheet
  collection + font-face merge used by image and PDF runs, asserting the
  `@font-face` Custom face is actually attached by layout.

## 10. Known limitations, deferred items & open questions

- **No font hinting** — small text relies on 2× supersampling; below ~8px
  edges are softer than hinted/system renderers. Documented as a fidelity
  tier-1 success ("image mode not blocky 5×7 text") but not a hinting claim
  (`documentation/fidelity.md`).
- **Nearest-neighbour `<img>` scaling** — large photo downscaling is blocky
  (accepted; Go 1.26 removed stdlib scalers). Natural-size logos/grids are
  exact. An analytic/filtered scaler is a possible future improvement.
- **Single-page only** — extra page objects and TOC are ignored with a warning;
  no "render each page of a multi-page document" mode (aligns with
  wkhtmltoimage, which is single-shot).
- **Reverse for JPEG**: `--transparent` is PNG-only; JPEG always composites
  onto white with a warning.
- **SVG fidelity bounded by tdewolff/canvas** — unsupported SVG features or
  malformed paths yield a clean "no image" (element skipped). No second
  rasterizer means no fallback for exotic SVG (fonts in SVG, filters, complex
  animations); see `documentation/deferred.md` for the URL-printing roadmap
  that keeps this posture.
- **SVG size heuristic** — `svgCSSPixelSize` scans only the root `<svg>`
  viewBox/width/height; CSS-sized SVG (no intrinsic size) defaults to 100px;
  `maxSide` 512 default / 1024 via layout's `svgRasterMax`, 4096 hard cap.
- **Canvas panic containment is best-effort** — `recover` covers the draw
  call, but a hypothetical future canvas release could panic outside the
  protected frame; tests (`TestRasterizeBrokenSVG`) pin current behavior.
- **dPI/quality knobs for PDF** remain ignored (honest matrix entry in
  `compatibility-matrix.md`); image mode honors `--quality` for JPEG only.
- **Per-run caches are not shared across renders** — a document with many
  repeated images pays re-decode per render; acceptable for single-shot CLI,
  a repeated-conversion library workload could reuse a cache (open question).
- **History:** the review-plan chain (`plans/reviews/improve-codebase/
  architecture-review-2026-08-07/phases/phase-01-engine-seam-and-surface.md`
  P1-1/P1-4, `phase-05-output-fonts-raster.md` P5-01..P5-07) records the
  consolidation that produced the current shape (shared `Request` seam, shared
  paint table, per-raster glyph atlas, shared sheet/font-face collection).

## 11. Related documents

- `documentation/architecture.md` — top-level package map and pipeline (this
  doc expands the `internal/imageout` row).
- `documentation/fidelity.md` — tier-1 claim "image mode not blocky 5×7 text";
  shipped TTF outline AA + 2× supersample; DPI/quality matrix honesty.
- `documentation/cli.md` — `gowkhtmltoimage` flags: `--width/--height/--format/
  --quality/--transparent/--crop-x/y/w/h`, smart width behavior.
- `documentation/library-api.md` — `ImageConverter` / `ImageSettings` usage.
- `documentation/samples.md` — committed `output/*.png` and showcase fixtures.
- `documentation/compatibility-matrix.md` — image-mode support matrix entries
  (images, transparency, quality knobs).
- `documentation/THREAT-MODEL.md` — local-file ACL and subresource policy that
  `imageLoadGlobal` and the fetch/decode budgets enforce.
- `documentation/deferred.md` — deferred URL→print items that keep the SVG
  posture.
- `plans/00-canonical-pure-go-rewrite.md` and the review phase ledgers (P1-1,
  P1-4, P5-01..P5-07) — the engineering history behind this shape.
- Sibling architecture docs in this directory: `01-entrypoints-cli.md`,
  `02-library-api.md`, `03-settings.md`, `04-load.md`, `05-html-parser.md`,
  `06-css.md`, `07-layout.md`, `08-convert-pipeline.md`, `09-pdf-writer.md`,
  `10-imageout-svg.md` (this document).
