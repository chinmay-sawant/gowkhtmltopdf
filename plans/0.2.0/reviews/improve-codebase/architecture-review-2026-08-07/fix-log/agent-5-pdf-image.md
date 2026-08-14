# Agent 5 — PDF/image/SVG follow-up

Scope honored: `internal/pdf/**`, `internal/imageout/**`, `internal/svg/**`,
and tests/benchmarks owned by those packages. No phase checklist or
`fix-contract.md` was edited. No Git commands were run.

## Per-CID status

### PDF-01 — collision-safe page resources and cloned-page ownership — **done**

- `internal/pdf/images.go` now allocates a page-local suffix when a caller
  reuses an image name (`I0` becomes `I0_1`, etc.). The emitted `Do` operator
  uses the allocated name, so body and header/footer image counters cannot
  overwrite one another in `/Resources`.
- `internal/pdf/content.go` now deep-copies resource maps in `cloneContent`:
  string maps, rune slices, and image-resource records are independent on a
  duplicated page; immutable parsed font pointers remain shared.
- Regression coverage: `TestImageResourceNamesDoNotCollideAcrossBands` and
  `TestDuplicatePageOwnsResourceMaps`.

### PAINT-01 — display-list traversal parity — **done in owned scope; integration note**

- `internal/imageout/imageout.go` now sorts raster operations with the same
  z-index, chrome-under-content, stable-source-order policy used by the PDF
  display-list painter. Existing `layout.StyleOf` and `layout.FakeBoldFor`
  remain the canonical semantic table, including alpha behavior.
- Regression coverage: `TestRasterPaintOrderMatchesPDFLayerPolicy`.
- The raster path intentionally preserves destination-specific behavior:
  transparent-canvas alpha compositing, raster scaling, and no PDF
  pagination/annotation emission.
- Header/footer traversal itself is owned by `internal/layout.PaintBand`; this
  package consumes the same policy but cannot edit that package under the
  ownership contract.

```go
// FIX-REVIEW: PAINT-01 PDF body/header/footer traversal remains owned by
// internal/layout.Paint and PaintBand; imageout consumes the same ordering and
// StyleOf policy without duplicating pagination or annotation semantics.
```

### FONT-01 — stable face identity and shared shaped runs — **done**

- `internal/pdf/fonts.go` fingerprints parsed face bytes with SHA-256; the
  document subset cache now keys by mode, face fingerprint, display name, and
  used runes. Distinct loaded faces with the same PostScript/display name no
  longer share a PDF font object.
- `internal/pdf/shape_gotext.go` adds `ShapeRun`, the canonical shaped text,
  rune sequence, and per-rune point advances. PDF `TextShow` and imageout
  `ttfDrawString` consume that result, keeping shaping and advance walks
  aligned for ligatures/presentation forms.
- Regression coverage: `TestFontCacheSeparatesLoadedFacesWithSameDisplayName`
  and `TestShapeRunKeepsTextAndAdvancesAligned`.
- Benchmark added: `BenchmarkShapeRun`.

### Shared preparation/context consumption — **verified**

`internal/imageout` already consumes `convert.CollectSheets`,
`convert.MergeFontFaces`, and `load.NewLoader` with the current context-aware
request path. No incompatible API was introduced by this slice, so no adapter
was needed and neither dependency package was edited.

## Files changed

- `internal/pdf/content.go`
- `internal/pdf/images.go`
- `internal/pdf/fonts.go`
- `internal/pdf/fonttype0.go`
- `internal/pdf/pdf.go`
- `internal/pdf/shape_gotext.go`
- `internal/pdf/image_test.go`
- `internal/pdf/pdf_test.go`
- `internal/pdf/font_test.go`
- `internal/pdf/shape_test.go`
- `internal/pdf/bench_test.go`
- `internal/imageout/imageout.go`
- `internal/imageout/ttfraster.go`
- `internal/imageout/raster_test.go`
- this fix log

`internal/svg/**` required no code change; its existing panic-to-error and
stdlib PNG encoding path was validated with the package tests.

## Validation

- `gofmt -w` on all changed Go files — passed
- `go test ./internal/pdf ./internal/imageout ./internal/svg` — passed
- `go test -race ./internal/pdf ./internal/imageout ./internal/svg` — passed
- `go test -run '^$' -bench 'Benchmark(ShapeRun|Write50Pages)$' ./internal/pdf -benchtime=100ms` — passed
  (`BenchmarkShapeRun`: 84,543 iterations, 1,286 ns/op in the captured run)
- `go test ./...` — passed
- `go vet ./...` — passed
- `go build ./...` — passed

## Remaining markers

- One exact `PAINT-01` integration marker is present in
  `internal/imageout/imageout.go`; it identifies the layout-owned
  `PaintBand` call site that the orchestrator must keep aligned.
