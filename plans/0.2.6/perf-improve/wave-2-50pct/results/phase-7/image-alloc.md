# PERF3-35 / PERF3-36 image allocation

Scope: `internal/imageout`. The public 500-tile canvas is 1024x4040.
Each pixel is 4 bytes, so the old paint target is 1024 x 4040 x 4 = 16,547,840
bytes. Snapshot M counted 26,414,016 B/op for that row, so that one buffer
was 62% of the total. The 13.21 MB half-target cannot land while it lives.

## 1. What the old path did

`rasterizeContextPolicy` allocated one full `*image.NRGBA`, painted every op
onto it, then `encodePNG` / `encodeFastPNG` (`pngfast.go`) walked rows.
Encoding was already streaming. The waste was the canvas.

`Render` still returns that full image for callers who want pixels. The B/op
row is `ImageDocument.WriteImage` -> `RunRequest`. That path no longer builds
the full canvas for large PNG.

## 2. Strip raster

Large PNG uses the same `directRaster` / `directRasterPixels` gate as the
existing fast encoder. It paints horizontal windows and feeds rows into that
encoder.

- One reused strip. At 1024 px wide the backing array is 1 MiB, 256 rows
  (`rasterStripBytes` in `tile_raster.go`). A 4040-row canvas reuses it 16
  times.
- The strip is an `*image.NRGBA` whose `Rect` is the window in canvas
  coordinates. `paint` already clips to `img.Bounds()`, so an op that
  crosses a strip edge writes only the pixels in that window.
- Ops whose conservative `paintOpBounds` miss the window are skipped.
- Padding is applied once up front. Xform lives on a shared extra pointer,
  so a per-strip offset would add padding on every window.
- Rows go to `pngRowEncoder` (`pngfast.go`), shared with `encodeFastPNG`.
  Filter type stays 0. Deflate level stays 2.

Small canvases stay on the old path: supersample plus `image/png`,
byte-for-byte. JPEG still allocates a full canvas. The gate is
`stripRasterApplies`: PNG and `plan.direct`.

Layout runs in `RenderObjects`. `Finalize` calls `encodeStripPNG` for large
PNG, or rasterize plus `writeEncodedOutput` for everything else.

## 3. Clip math

Two painters treated the clipped dest as the whole shape. That is invisible
on a full canvas when the op sits inside the picture, and wrong on a strip
cut.

- `paintRoundedFill` used the clipped rect as the rounded box, so a tall
  rounded fill would grow fake corners on every strip edge. It now tests
  `roundedContains` against the unclipped op rectangle and only iterates
  the window.
- `paintImage` scaled to the clipped size, which squashed a spanning image
  into each strip. It now scales to the full dest and draws the visible
  window with an adjusted source point.

On a full canvas, clip equals the op rect for in-bounds ops, so existing
IMG-01 / IMG-03 cases keep the same pixels.

## 4. Tests

Existing IMG-01 / IMG-03 tests were not rewritten. A few construction sites
in `blend_test.go`, `quality_policy_test.go`, and `raster_test.go` now call
`layout.Op.SetBlendMode` / `SetXform` / `SetTextTransform` / `SetPaintOpacity`
because `Op` extra payloads can no longer appear in composite literals.

New file `tile_raster_test.go`:

- `TestStripRasterMatchesFullCanvas`: 80x64 synthetic list (fills that
  cross an 8-row strip, a rounded fill, text) painted full
  (`rasterPolicyDirect`) vs tiled. Pixels and bounds match.
- `TestStripRasterPublicTileDimensions`: 1024x2056 and 1024x4040, full
  opacity. Auto policy vs tiled pixels match. Dimensions stay 1024x2056 and
  1024x4040. `encodeStripPNG` decodes back to the full-canvas pixels.
- `TestStripRasterBufferSmallerThanFullCanvas`: strip backing <= 1 MiB,
  full 500-tile canvas 16,547,840 bytes.
- `TestStripRasterAppliesOnlyLargePNG`: JPEG and small PNG stay off the
  strip path.

`go test ./internal/imageout -count=1`: ok, 0.243s.
`golangci-lint run ./internal/imageout`: clean.

## 5. PERF3-35 ranking and measured B/op

Prior public-image profiles
(`plans/0.2.6/perf-time/results/image/image-time.md`) after `pngfast.go`:

| phase | 500-tile share of WriteImage | B/op note |
|---|---:|---|
| fast PNG encode | 45.7% | already streaming rows |
| layout | 30.9% | sibling package |
| rasterizeContext | 23.5% | was the 16.5 MB canvas plus fill and glyph blend |

This tree, `go test . -run '^$' -bench '^BenchmarkLibraryImage$' -benchtime=1x -count=1 -benchmem`, one sample:

| tiles | height | output-bytes | B/op | Snapshot M B/op | 50% target |
|---:|---:|---:|---:|---:|---:|
| 250 | 2056 | 141,917 | 6,377,144 | 14,296,096 | 7,148,048 |
| 500 | 4040 | 282,749 | 10,398,984 | 26,414,016 | 13,207,008 |

500-tile B/op is 10.40 MB, under 13.21 MB. 250-tile B/op is 6.38 MB, under
7.15 MB. Encoded PNG size stayed at the filter-None trade (141,917 /
282,749). Dimensions stayed 1024x2056 and 1024x4040. Full opacity is still
checked by `validateLibraryImageOutput`.

Rows below the 2,097,152-pixel gate (2 through 200 tiles here) still take
the supersample path and still allocate a 2x canvas. That is the existing
small-canvas policy, not a regression of the 250/500 rows.

Time on this one sample is 20.5 ms / 52.3 ms. PERF3-37's 12.86 / 22.14 ms
line is a later measure, not this cut. Strip paint can cost extra op visits
on spanning fills; the B/op cut is the point of PERF3-36.

## 6. What did not change

- Filter type 0, deflate level 2, IDAT chunk size 8 KiB.
- `directRasterPixels` (2,097,152). Public 250/500 canvases still take the
  direct branch.
- `Render` still returns a full `*image.NRGBA`.
- Decoded strip pixels match the full-canvas direct path at 80x64,
  1024x2056, and 1024x4040.
