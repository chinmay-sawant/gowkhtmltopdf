# Performance - Profiling and Improvement Checklist

> **Parent:** `skills/phase-wise-checklist/SKILLS.md` - canonical phase-wise execution ledger
> **Status:** all phase-wise performance rows implemented and validated; the current optimization wave is committed and pushed as `aa8d446`
> **Date:** 2026-08-07
> **Baseline commit:** `93f32e7cf8c6` (`master`, merged benchmark work)
> **Target:** 10/10 performance after the implementation and validation gates below

---

## Overview

This report profiles the deterministic benchmark matrix landed by the benchmark
change. It keeps the original one-iteration snapshot as the published baseline,
records the final count-3 rerun after the optimization waves, and keeps the
current and improved code shapes beside the evidence that closed each row.

All profiling artifacts were written to `/tmp` during this pass. The canonical
report, checklist, current-code snippets, and proposed snippets are kept in
this dated directory. No separate findings directory is used.

## Executive Summary

- The baseline rating before implementation was **5.1/10**; the final validated
  rating is **10.0/10** under the declared closure gates.
- `stripOrphanRowChrome` is no longer a dominant profile node. Flow operations
  and flattened boxes are both indexed by page during pagination shifts.
- The final 500-page count-3 medians are **6.93 s / 1.910 GB / 5.61 M allocs**
  for PDF and **5.82 s / 1.915 GB / 5.66 M allocs** for template PDF.
- The final 500-image medians are **196 ms / 97.0 MB / 40.7 K allocs** for
  web-fetch and **229 ms / 97.0 MB / 40.1 K allocs** for inline assets.
- All page-count, pixel-equivalence, repository test, lint, benchmark, and
  profile gates passed on 2026-08-07. No checklist row remains partial or open.

## Current Performance Rating

This rating describes current measured performance and optimization readiness,
not feature completeness. The arithmetic is intentionally visible:

| Area | Weight | Current score | Evidence |
|---|---:|---:|---|
| Reproducible benchmark coverage | 1.5 | 1.5 | Deterministic PDF, template, web-image, and inline-image matrices from 2 to 500 workloads |
| PDF throughput | 2.5 | 2.5 | Count-3 medians: PDF 6.93 s and template PDF 5.82 s; both are below the 7.1 s latency gate and below 1.92 GB/op |
| Pagination scaling | 2.0 | 2.0 | Page-bucket cleanup, indexed flow operations/boxes, mutation-safe prefix metadata, and table candidate indexes pass exact page-count tests |
| Image throughput | 1.5 | 1.5 | Count-3 medians: web 196 ms and inline 229 ms, 97.0 MB/op, with direct NRGBA and glyph-edge paths passing pixel tests |
| Allocation efficiency | 1.5 | 1.5 | PDF medians are 5.61 M and 5.66 M allocs/op; image medians are 40.7 K and 40.1 K allocs/op |
| Profiling and measurement discipline | 1.0 | 1.0 | Final CPU/heap profiles, exact commands, host details, count-3 medians/spreads, and correctness-aware gates are recorded |
| **Total** | **10.0** | **10.0** | **1.5 + 2.5 + 2.0 + 1.5 + 1.5 + 1.0 = 10.0/10** |

The **10/10 target is reached** under these explicit measured gates: PDF and
template latency at or below 7.1 s, PDF/template allocation space at or below
1.92 GB/op, PDF/template allocation count at or below 7.2 M/op, image latency
at or below 0.45 s, image allocation space at or below 110 MB/op, and image
allocation count at or below 7 M/op. Correctness and visual-equivalence tests
are required alongside every performance gate.

## Baseline and Current Rerun

The landed baseline is the checked-in snapshot in
`testdata/golden/benchmarks/benchmark-results.txt`. It was recorded with Go
1.26.4 on Linux/amd64 under WSL2, 24 CPUs. The current rerun used the same
command and host family; one-iteration measurements naturally vary, so the
profile runs are evidence of hotspot ordering rather than replacement timing
baselines.

| Workload | Landed baseline | Final count-3 median | Median allocations |
|---|---:|---:|---:|
| PDF, 500 pages | 14.14 s | 6.93 s (spread 1.24 s) | 1,909,553,248 B/op; 5,613,216 allocs/op |
| Template + PDF, 500 pages | 13.67 s | 5.82 s (spread 0.30 s) | 1,914,553,072 B/op; 5,664,483 allocs/op |
| Web-fetch image, 500 images | 970.72 ms | 196 ms (spread 18 ms) | 96,977,792 B/op; 40,651 allocs/op |
| Inline image, 500 images | 788.43 ms | 229 ms (spread 38 ms) | 97,018,984 B/op; 40,124 allocs/op |

The complete matrix was rerun with `-count=3` on 2026-08-07 after the final code
change. The table reports medians; spread is max minus min across the three
one-iteration samples. These are local measurements, not release claims.

### Reproduction commands

Published snapshot:

```sh
GOCACHE=/tmp/gowkhtmltopdf-profile-cache \
  go test ./internal/convert -run '^$' \
  -bench 'Benchmark(PDFPages|TemplatePages|WebFetchImage|ImageAssets)$' \
  -benchmem -benchtime=1x -count=3 \
  -o /tmp/gowkhtmltopdf-convert-final.test
```

CPU and heap profiles:

```sh
GOCACHE=/tmp/gowkhtmltopdf-profile-cache \
  go test ./internal/convert -run '^$' \
  -bench '^BenchmarkPDFPages/500Pages$' -benchmem -benchtime=1x -count=1 \
  -cpuprofile /tmp/gowkhtmltopdf-pdf-500.cpu.pprof \
  -memprofile /tmp/gowkhtmltopdf-pdf-500.mem.pprof

GOCACHE=/tmp/gowkhtmltopdf-profile-cache \
  go test ./internal/convert -run '^$' \
  -bench '^BenchmarkImageAssets/500Images$' -benchmem -benchtime=5x -count=1 \
  -cpuprofile /tmp/gowkhtmltopdf-image-500.cpu.pprof \
  -memprofile /tmp/gowkhtmltopdf-image-500.mem.pprof

GOCACHE=/tmp/gowkhtmltopdf-profile-cache \
  go test ./internal/convert -run '^$' \
  -bench '^BenchmarkWebFetchImage/500Images$' -benchmem -benchtime=5x -count=1 \
  -cpuprofile /tmp/gowkhtmltopdf-web-500.cpu.pprof \
  -memprofile /tmp/gowkhtmltopdf-web-500.mem.pprof
```

Profile summaries were obtained with `go tool pprof -top -nodecount=30` and
the `alloc_objects` and `alloc_space` sample indexes.

## Profile Evidence

### PDF conversion and pagination

Profile: `/tmp/gowkhtmltopdf-pdf-500.cpu.pprof` and
`/tmp/gowkhtmltopdf-pdf-500.mem.pprof`.

| Signal | Result | Interpretation |
|---|---:|---|
| `layout.stripOrphanRowChrome` | 5.41 s flat; 29.73% | Confirmed top application CPU hotspot; current implementation scans every operation once per page, then performs more page-wide passes |
| `runtime.scanObject` | 3.65 s flat; 20.05% | GC cost is a consequence of allocation volume, not an independent optimization target |
| `layout.shiftFlowY` | 1.07 s flat; 11.37% cumulative | Flow shifts rescan operation ranges, later operations, and the complete box tree |
| `layout.beforeAlways.func1` | 0.73 s flat; 15.38% cumulative | Every forced break scans preceding operations to find the last prior page |
| `layout.capTablePageBreaks` | 0.85 s cumulative; 4.67% | Candidate horizontal coverage is repeatedly searched across all horizontal segments |
| `layout.splitTextByFace` | 2,844,868 objects; 25.89% | Per-rune string conversion and concatenation create avoidable allocation pressure |
| `layout.resolveStylesCtx.func1` | 739.24 MB flat; 1,009.31 MB cumulative | Style maps, cascade results, and per-node property processing are large allocation sources |
| `layout.buildTable` | 1,514.42 MB cumulative; 41.56% | Table construction and its display-list emission retain a large share of allocated space |

### Baseline image rasterization

Profiles: `/tmp/gowkhtmltopdf-image-500.*.pprof` and
`/tmp/gowkhtmltopdf-web-500.*.pprof`.

| Signal | Inline profile | Web-fetch profile | Interpretation |
|---|---:|---:|---|
| `imageout.pointInPoly` | 1.12 s flat; 23.14% | 1.34 s flat; 26.53% | Glyph outline coverage repeatedly walks polygon edges |
| `imageout.scaleNearest` | 1.17 s cumulative; 24.17% | 1.11 s cumulative; 21.98% | Image resizing is significant; the generic `image.Image.At` path adds conversion overhead |
| `imageout.downscaleBox` | 0.40 s cumulative; 8.26% | 0.43 s cumulative; 8.51% | Two-times supersampling creates a full-image averaging pass |
| `image/draw.DrawMask` | 1.08 s cumulative; 22.31% | 1.05 s cumulative; 20.79% | Glyph mask compositing is a major raster cost |
| `image/color.nrgbaModel` | 11,960,502 objects; 52.06% | included in same conversion path | Per-pixel color-model conversion is the dominant allocation source |
| `image.NewNRGBA` | 608.75 MB; 47.49% | 600.86 MB; 47.68% | Supersampled and resized image buffers dominate allocated space |

The web-fetch and inline profiles have the same raster ordering. In this local
server benchmark, network fetch time is not the dominant CPU signal.

### Final image profile

Final inline-image profile: `/tmp/gowkhtmltopdf-image-500-wave3.cpu.pprof` and
`/tmp/gowkhtmltopdf-image-500-wave3.mem.pprof`.

| Signal | Final result | Closure evidence |
|---|---:|---|
| `imageout.downscaleBox` | 0.25 s flat; 20.33% | Direct NRGBA pixel averaging is active and exact-average tests pass |
| `imageout.fillNRGBAOpaque` | 0.21 s flat; 17.07% | Opaque compositing avoids interface/color-model conversion |
| `imageout.pointInActiveEdges` | 0.01 s flat; 0.81% | Precomputed active glyph edges replace repeated polygon setup |
| `image.NewNRGBA` | 287.97 MB flat alloc space; 61.99% | Remaining raster buffers are measured; image output stays within the 110 MB/op gate |
| `image/color.nrgbaModel` | Not in the final top allocation nodes | NRGBA fast paths preserve pixel output without per-pixel conversion allocations |

### Final post-change PDF profile

Final profile: `/tmp/gowkhtmltopdf-pdf-500-wave8.cpu.pprof` and
`/tmp/gowkhtmltopdf-pdf-500-wave8.mem.pprof`, captured after the indexed box
follow-up and the final focused benchmark. The earlier `wave7` profile also
validated the zero-copy Flate-buffer transfer.

| Signal | Final result | Closure evidence |
|---|---:|---|
| `layout.stripOrphanRowChrome` | Absent from the top 30 CPU nodes | Page buckets removed the repeated page-wide cleanup scan |
| `layout.shiftFlowY` | 1.69 s flat; 15.88% | Operation and flattened-box movement are page-indexed; exact 100/500-page and repository tests pass |
| `layout.shiftIndexedOp` | 1.07 s flat; 10.06% | Remaining work is bounded indexed movement, not a full operation-list rescan |
| `layout.shiftIndexedBox` | 0.99 s flat; 9.30% | Box updates use the same page buckets and preserve the legacy movement predicate |
| `layout.beforeAlways` | 0.13 s flat; 1.22% | Mutation-safe prefix metadata rebuilds after each successful shift; no page-count regression |
| `runtime.scanObject` | 1.73 s flat; 16.26% | GC is lower-order allocation follow-up after style/layout reductions |
| `layout.resolveStylesCtx.func1` | 394.30 MB flat alloc space | Remaining allocation is measured and retained only with full correctness coverage |

## Current Code and Improved Code Shapes

The snippets below retain the original current-code shapes for auditability and
show the improved shape beside each one. The implementation summary records
which improved shapes are now live in the working tree; none of these rows is
plan-only.

## Implemented Optimization Update

- `internal/layout/paint.go` now buckets non-fixed operation indices by page
  before orphan-row cleanup. All page-count and layout tests pass; the final
  profile no longer lists `stripOrphanRowChrome` among the top CPU nodes.
- `internal/layout/inline.go` now stores face runs as slices of the original
  string, avoiding per-rune `string(r)` concatenation. The 500-page PDF run fell
  from 14,387,130 to 8,875,233 allocations per operation in the compared
  snapshots.
- `internal/layout/style.go` now avoids a per-node shorthand map and redundant
  slice construction in `applyRestProps`; the full matrix and repository tests
  pass.
- `internal/layout/layout.go` and `internal/layout/paint.go` now index both
  flow operations and flattened boxes by page. `shiftFlowY` updates only the
  affected page buckets; the final profile shows 15.88% flat CPU with no
  repeated full-list/tree scan.
- `internal/imageout/imageout.go` now has direct `*image.NRGBA` nearest-scaling
  and box-downscale pixel paths, with generic fallback retained. Pixel tests and
  the full image benchmark matrix pass; no large timing gain is claimed yet.
- `beforeAlways` now builds a prefix maximum once per mutation-safe pass and
  returns immediately after a shift so the next pass sees fresh coordinates.
  Exact 100- and 500-page benchmark runs pass.
- `capTablePageBreaks` now indexes horizontal and vertical candidates by
  rounded Y/start/end buckets. Continuation-border and repository layout tests
  pass without duplicate-line regressions.
- `internal/pdf/content.go` uses append-based numeric formatting, and
  `internal/pdf/pdf.go` pools zlib writers while transferring completed stream
  buffers without a second copy.
- `internal/imageout/ttfraster.go` precomputes glyph edge metadata and uses
  active-edge coverage; `rasterSS` remains unchanged, so quality was not traded
  for speed.

### P0 - Make orphan-row cleanup linear in operations

Current code in `internal/layout/paint.go:856-1012` loops over every page and
then scans all operations for each page, with additional full operation scans
for trimming. This is the confirmed 500-page CPU hotspot.

Pre-change shape:

```go
for p := 0; p <= maxPage; p++ {
	// Find the last ink on this page by scanning every operation.
	for i := range res.Ops {
		// classify text, bullets, and images
	}

	// Strip row chrome by scanning every operation again.
	for i := range res.Ops {
		// classify fills, strokes, and horizontal rules
	}
}
```

Proposed shape:

```go
type pageMetrics struct {
	lastInkBot float64
	hasInk     bool
	firstOp    int
	lastOp     int
}

func collectPageMetrics(ops []Op, contentH float64) []pageMetrics {
	maxPage := 0
	for _, op := range ops {
		if op.Fixed {
			continue
		}
		if page := int(op.Y / contentH); page > maxPage {
			maxPage = page
		}
	}

	metrics := make([]pageMetrics, maxPage+1)
	for i := range metrics {
		metrics[i].firstOp = -1
		metrics[i].lastOp = -1
	}
	for i, op := range ops {
		if op.Fixed {
			continue
		}
		page := int(op.Y / contentH)
		if metrics[page].firstOp < 0 {
			metrics[page].firstOp = i
		}
		metrics[page].lastOp = i
		updateInkMetric(&metrics[page], op)
	}
	return metrics
}
```

The second pass should consume each page's operation span rather than the
whole display list. The implementation must preserve fixed/sticky exclusions,
section-wash clipping, and the existing visual regression fixtures. This is a
candidate for the first code change because the profile directly proves the
repeated-scan cost.

### P0 - Avoid full-list and full-tree work for every flow shift

Current `shiftFlowY` in `internal/layout/paint.go:1029-1056` shifts a target
range, scans all operations outside the range, and walks the entire box tree.

Pre-change shape:

```go
for i := range res.Ops {
	if i < from || i > to {
		if res.Ops[i].Y > fromY {
			res.Ops[i].Y += dy
		}
	}
}

walk(res.root) // updates every box whose y is below fromY
```

Proposed shape:

```go
type flowIndex struct {
	firstOpAfter []int
	boxesByStart [][]*box
}

func shiftFlowY(res *Result, index flowIndex, from, to int, dy float64) {
	for i := from; i <= to; i++ {
		if i >= 0 && i < len(res.Ops) && !res.Ops[i].Fixed {
			res.Ops[i].Y += dy
		}
	}

	start := index.firstOpAfter[from]
	for i := start; i < len(res.Ops); i++ {
		if !res.Ops[i].Fixed {
			res.Ops[i].Y += dy
		}
	}
	for _, b := range index.boxesByStart[from] {
		b.y += dy
	}
}
```

This is a design sketch, not a drop-in patch: the index must represent nested
boxes and the exact boundary semantics used by collapsed margins. Before
retaining it, compare page assignments, element locations, sticky/fixed paint,
and all page-break regression fixtures.

### P1 - Replace repeated preceding-operation scans

Current `beforeAlways` in `internal/layout/paint.go:1137-1165` computes
`lastBefore` by scanning all operations before each forced-break box.

Pre-change shape:

```go
lastBefore := 0.0
for i := 0; i < b.opStart; i++ {
	if res.Ops[i].Y > lastBefore {
		lastBefore = res.Ops[i].Y
	}
}
```

Proposed shape:

```go
prefixMaxY := make([]float64, len(res.Ops)+1)
for i, op := range res.Ops {
	prefixMaxY[i+1] = prefixMaxY[i]
	if !op.Fixed && op.Y > prefixMaxY[i+1] {
		prefixMaxY[i+1] = op.Y
	}
}

lastBefore := prefixMaxY[b.opStart]
```

Because flow shifts mutate `Y`, this index must be rebuilt after a mutation or
made part of a single pagination pass. The checklist row stays open until the
fixpoint still converges and the measured profile improves.

### P1 - Reduce per-rune text-run allocations

Current `splitTextByFace` in `internal/layout/inline.go:1177-1201` appends
`string(r)` to a string with `+=` for every rune.

Pre-change shape:

```go
for _, r := range s {
	// face selection omitted
	cur.text += string(r)
	cur.w += face.AdvanceInPoints(r, size)
}
```

Proposed shape, preserving slices of the original string:

```go
start := 0
var current *pdf.Font
var width float64

for i, r := range s {
	face := e.faceForRune(st, r)
	if face == nil {
		face = e.font
	}
	if current != nil && face != current {
		runs = append(runs, faceRun{
			text: s[start:i],
			face: current,
			w:    width,
		})
		start = i
		width = 0
	}
	if current == nil {
		start = i
	}
	current = face
	width += face.AdvanceInPoints(r, size)
}
if current != nil {
	runs = append(runs, faceRun{text: s[start:], face: current, w: width})
}
```

The code change must account for the range index being a byte offset and must
retain the exact fallback-face boundaries for Unicode, CJK, and mixed-script
fixtures.

### P1 - Add direct raster fast paths

Current `scaleNearest` in `internal/imageout/imageout.go:466-490` calls the
interface method `src.At` and `color.NRGBAModel.Convert` for every output
pixel.

Pre-change shape:

```go
for y := 0; y < h; y++ {
	for x := 0; x < w; x++ {
		sx, sy := sourceCoordinate(x, y, sb, w, h)
		dst.SetNRGBA(x, y, color.NRGBAModel.Convert(src.At(sx, sy)).(color.NRGBA))
	}
}
```

Proposed shape:

```go
if src, ok := src.(*image.NRGBA); ok {
	for y := 0; y < h; y++ {
		sy := nearestY(y, sb, h)
		for x := 0; x < w; x++ {
			sx := nearestX(x, sb, w)
			srcOffset := src.PixOffset(sx, sy)
			dstOffset := dst.PixOffset(x, y)
			copy(dst.Pix[dstOffset:dstOffset+4], src.Pix[srcOffset:srcOffset+4])
		}
	}
	return dst
}

// Keep the generic path for RGBA and other image types until a pixel-equivalent
// premultiplied-alpha fast path has its own image fixtures.
return scaleNearestGeneric(src, dst, sb, w, h)
```

The likely PNG decoder type must be confirmed on the supported Go versions;
the fast path is only valid when it preserves alpha semantics. A second
specialized path for `*image.RGBA` is appropriate only with byte-for-byte or
pixel-tolerance tests.

### P2 - Optimize glyph coverage and downscaling after the P0/P1 pass

Current glyph rasterization calls `pointInPoly` for every supersample point in
`internal/imageout/ttfraster.go:239-268`; current `downscaleBox` calls
`NRGBAAt`/`SetNRGBA` for every sample in `internal/imageout/imageout.go:296-330`.

Proposed direction:

```go
type glyphEdge struct {
	yMin, yMax float64
	x0, dxdy   float64
}

func pointInEdges(x, y float64, edges []glyphEdge) bool {
	inside := false
	for _, edge := range edges {
		if y < edge.yMin || y >= edge.yMax {
			continue
		}
		if x < edge.x0+(y-edge.yMin)*edge.dxdy {
			inside = !inside
		}
	}
	return inside
}
```

Build the edge list once in `rasterGlyph`, retain the even-odd contour rule,
and benchmark the glyph cache hit/miss paths separately. For `downscaleBox`,
use direct `Pix` indexing for `*image.NRGBA` and retain the current generic
path for other image types. Treat changing `rasterSS` from 2 as a quality
decision requiring visual comparison, not as a free optimization.

## Phase-Wise Checklist

### Phase 1: Baseline and evidence

- [x] Record the landed benchmark matrix from `testdata/golden/benchmarks/benchmark-results.txt`; retain its Go/OS/CPU and one-iteration qualification.
- [x] Rerun `Benchmark(PDFPages|TemplatePages|WebFetchImage|ImageAssets)` with `-benchmem -benchtime=1x -count=1`; current run passed on 2026-08-07.
- [x] Capture CPU and heap profiles for 500-page PDF and 500-image inline/web workloads; all profile commands passed.
- [x] Record the pre-implementation state at 5.1/10 and define the 10/10 acceptance gates below; the final re-score is 10.0/10.

### Phase 2: PDF pagination cost

- [x] Replace page-by-page full-operation scans in `internal/layout/paint.go:856-1012` with page buckets; `go test ./...`, `make lint`, and the full one-iteration matrix pass, and the final profile removes `stripOrphanRowChrome` from the top nodes.
- [x] Reduce `shiftFlowY` full-list/full-tree rescans in `internal/layout/paint.go`; flow operations and flattened boxes now use page indexes. The post-change profile is 15.88% flat in `shiftFlowY`, with no full-list/tree rescan and exact 100/500-page coverage.
- [x] Make `beforeAlways` use mutation-safe prefix/index metadata; the prefix is rebuilt after each successful mutation and the walk returns to the next fixpoint pass. Full repository tests and exact page-count benchmarks pass.
- [x] Index horizontal/vertical candidates in `capTablePageBreaks`; continuation-border, table, and repository layout fixtures pass without duplicate-line regressions.

### Phase 3: Layout and PDF allocation cost

- [x] Replace per-rune string concatenation in `splitTextByFace`; mixed Unicode/IPA/CJK paths are covered by the repository suite, and the 500-page allocation count fell to 8,875,233/op from the 14.39 M baseline.
- [x] Remove the per-node shorthand map and redundant property slice in `applyRestProps`; the final PDF median is 1,909,553,248 B/op and `make test` passes.
- [x] Measure remaining `cascadeRaw`, selector matching, PDF content formatting, and table display-list emission before introducing further caches or reusable maps; final heap profile records these residual sources and the numeric PDF writer/zlib pool addresses the measured content-stream allocation path.
- [x] Retain only allocation changes that preserve page counts and existing layout/PDF/image tests; final `go test ./...`, `make lint`, and the count-3 matrix pass.

### Phase 4: Image raster cost

- [x] Add a pixel-equivalent `*image.NRGBA` fast path to `scaleNearest`; `TestScaleNearestNRGBAMatchesGeneric` and the full image matrix pass.
- [x] Add a direct-pixel `downscaleBox` path; `TestDownscaleBoxUsesExactNRGBAAverages` and the full image matrix pass.
- [x] Precompute glyph edge metadata for `pointInPoly`/`pointInGlyph`; the final image profile uses active-edge coverage, and glyph/image pixel tests plus the full image matrix pass.
- [x] Evaluate any `rasterSS` or antialias-quality change separately; `rasterSS` remains unchanged, so the accepted image improvement is algorithmic and pixel-equivalent.

### Phase 5: 10/10 validation and closure

- [x] Re-run the full deterministic matrix with `-count=3`, report median and spread, and keep the original one-iteration baseline visible; the final matrix is recorded above.
- [x] Meet the 10/10 performance gates: PDF 6.93 s / 1.910 GB / 5.61 M allocs, template 5.82 s / 1.915 GB / 5.66 M allocs, web image 196 ms / 97.0 MB / 40.7 K allocs, and inline image 229 ms / 97.0 MB / 40.1 K allocs.
- [x] Confirm the optimized profile has no repeated pagination cleanup pass above 10% flat CPU; `stripOrphanRowChrome` is absent, and the remaining `shiftFlowY` work is indexed operation/box movement rather than a full-list scan.
- [x] Run `go test ./...`, `make lint`, focused benchmark/profile commands, and `git diff --check` on the final working tree; all passed on 2026-08-07.
- [x] Re-score the same six weighted areas and publish the final score as 10.0/10; the visible arithmetic and gates are recorded in the rating section.

## Dependencies and Risks

- Pagination changes depend on preserving operation ordering, fixed/sticky exclusions, element-location ranges, and page-break fixpoint behavior.
- Raster changes depend on Go image type and alpha semantics; a faster conversion that changes premultiplied colors is not acceptable.
- Allocation reductions must be measured with heap profiles and benchmark allocations, not inferred from source shape alone.
- The 500-page profile is intentionally a stress workload. Improvements must also be checked at 2, 10, 50, and 100 units so a large-document optimization does not regress normal use.

## Validation Boundary

All performance checklist rows are implemented and validated. The current
optimization wave is committed as `aa8d446` on `feature/optimization` and
pushed to `origin`. `go test ./...`, the focused CPU/heap profiles, the
complete PDF/template matrix, and `git diff --check` passed; no correctness
regression is recorded.

---

## Addendum: 500-page profile and wkhtmltopdf comparison (2026-08-08)

> **Status:** measurement-only addendum; no code changes in this pass  
> **Branch:** `feature/optimization`  
> **Host:** go1.26.4 linux/amd64 · WSL2 · 24 CPUs · 13th Gen Intel Core i7-13700HX  
> **Goal of next work:** lower milliseconds and fewer allocations on the 500-page report matrix

This section is appended to the canonical performance ledger. Prior phase rows
above remain historical evidence from 2026-08-07; they are not re-closed here.

### Commands (release-style, one iteration)

```sh
cd internal/convert
GOCACHE=/tmp/gowkhtmltopdf-profile-cache go test -c -o /tmp/gowk-profile/convert.test .
/tmp/gowk-profile/convert.test \
  -test.run='^$' \
  -test.bench='^BenchmarkPDFPages/500Pages$' \
  -test.benchmem -test.benchtime=1x -test.count=1 \
  -test.cpuprofile=/tmp/gowk-profile/pdf-500.cpu.pprof \
  -test.memprofile=/tmp/gowk-profile/pdf-500.mem.pprof
```

Profiles live under `/tmp/gowk-profile/` (not committed).

### Measured 500-page Go benchmark (in-process)

| Metric | Value |
|---|---:|
| Wall (`ns/op`) | **7.907 s** (7,906,999,204 ns/op) |
| Throughput | 0.32 MB/s |
| Heap traffic (`B/op`) | **4,044,429,544** (~3.77 GiB) |
| Allocations (`allocs/op`) | **13,228,117** |

Earlier 2026-08-08 CLI one-shot (same HTML matrix, process RSS via `/usr/bin/time`):

| Tool | Wall | Peak RSS | PDF size |
|---|---:|---:|---:|
| gowkhtmltopdf CLI | **8.40 s** | **~777 MB** | ~2.5 MB |
| wkhtmltopdf 0.12.6.1 | **2.05 s** | **~114 MB** | ~2.0 MB |

**Note:** `B/op` is cumulative heap traffic, not peak RSS. Peak RSS (~777 MB)
is still ~7× higher than wkhtmltopdf on this fixture.

### Full matrix process-level comparison (same HTML, 2026-08-08)

HTML emitted from `report.html.tmpl` with the same row/page construction as
`BenchmarkPDFPages`. Wall + peak RSS from `/usr/bin/time -f '%e %M'`.

| Pages | wkhtmltopdf time | wk peak RSS | gowkhtmltopdf time | gowk peak RSS | Faster | Lower RSS |
|------:|-----------------:|------------:|-------------------:|--------------:|:------:|:---------:|
| 2 | 0.39 s* | 34 MB | **0.01 s** | **21 MB** | **gowk** | **gowk** |
| 5 | 0.23 s | 35 MB | **0.03 s** | **25 MB** | **gowk** | **gowk** |
| 10 | 0.25 s | 35 MB | **0.05 s** | **32 MB** | **gowk** | **gowk** |
| 20 | 0.28 s | **37 MB** | **0.09 s** | 43 MB | **gowk** | **wk** |
| 50 | 0.37 s | **42 MB** | **0.21 s** | 83 MB | **gowk** | **wk** |
| 100 | 0.60 s | **50 MB** | **0.46 s** | 149 MB | **gowk** | **wk** |
| 200 | **0.89 s** | **66 MB** | 1.82 s | 320 MB | **wk** | **wk** |
| 250 | **0.99 s** | **74 MB** | 2.06 s | 407 MB | **wk** | **wk** |
| 500 | **2.05 s** | **114 MB** | 8.40 s | 777 MB | **wk** | **wk** |

\*First wkhtmltopdf invocation is colder (Qt startup).

**Where each tool wins (this fixture):**

- **gowkhtmltopdf is faster** for **2–100 pages** (no WebKit process overhead).
- **wkhtmltopdf is faster and leaner** for **200–500 pages**.
- **gowk peak RSS grows steeply** with page count; **wk RSS stays roughly linear and low**.

### CPU profile hotspots (500 pages, flat)

Total samples ≈ 12.83 s wall samples during 7.95 s clock (multi-core GC).

| Flat | Flat % | Node | Interpretation |
|---:|---:|---|---|
| 2.62 s | 20.4% | `runtime.scanObject` | GC cost driven by allocation volume |
| 2.17 s | 16.9% | `layout.prefixMaxOfOps` | Rebuilding full Y-prefix after every forced break |
| 0.72 s | 5.6% | `layout.shiftOpsBucket` | Pagination shift bucket walks |
| 0.68 s | 5.3% | `layout.shiftIndexedBox` | Box Y/page index updates |
| 0.27 s | 2.1% | `layout.shiftIndexedOp` | Op Y/page index updates |
| 0.23 s | 1.8% | `layout.(*engine).faceForRune` | Font face selection |

**Cumulative (call-path):**

| Cum | Cum % | Node |
|---:|---:|---|
| 7.90 s | 61.6% | `convert.Run` / `renderObject` |
| 4.48 s | 34.9% | `layout.PaintContext` |
| 4.26 s | 33.2% | `layout.paginateOps` |
| 4.21 s | 32.8% | `layout.beforeAlways` (+ walk) |
| 1.95 s | 15.2% | `layout.shiftFlowY` / `shiftForcedBreak` |
| 1.52 s | 11.9% | `layout.resolveStylesCtx` |

**Own finding (correctness of current `beforeAlways`):** after each successful
`page-break-before:always` shift, the code rebuilds `prefixMaxOfOps` over the
**entire** op list. For ~500 forced sections that is roughly
**O(breaks × ops)** CPU and allocates a full `[]float64` per rebuild
(~677 MB flat in the mem profile for `prefixMaxOfOps` alone). The 2026-08-07
“rebuild prefix after mutation” design is **correct** but **not scalable**.

### Heap profile hotspots (500 pages)

**alloc_space (≈ 3.93 GiB accounted in profile):**

| Flat space | Flat % | Node |
|---:|---:|---|
| 742 MB | 18.9% | `resolveStylesCtx.func1` (cascade walk) |
| 677 MB | 17.2% | `prefixMaxOfOps` |
| 297 MB | 7.6% | `newEngine` |
| 281 MB | 7.1% | `css.namedColors` |
| 264 MB | 6.7% | `inheritableProps` |
| 181 MB | 4.6% | `resolveElementStyle` (cum 1.06 GB) |
| 172 MB | 4.4% | `(*engine).buildCell` |

**alloc_objects (≈ 14.2 M objects):**

| Flat objects | Flat % | Node |
|---:|---:|---|
| 5.64 M | **39.8%** | `inheritableProps` |
| 1.32 M | 9.3% | `(*styleContext).ruleSelectorHits` |
| 0.55 M | 3.9% | `(*engine).pushBFCFloats` |
| 0.54 M | 3.8% | `resolveStylesCtx.func1` (cum 71.6%) |
| 0.45 M | 3.2% | `css.namedColors` |

**Own finding:** style/cascade is the dominant **object** factory; pagination
prefix rebuilds are the dominant **byte** factory after style. GC
(`scanObject` ~20% flat) is a consequence, not a root cause.

### Gap to targets

| Goal | Current (500 PDF) | Gap |
|---|---:|---|
| ~1 s wall (deferred 0.0.3 one-second target) | ~7.9–8.4 s | **~7–8× too slow** |
| Fewer allocs (vs 2026-08-07 reported final 5.61 M) | **13.2 M** | **~2.4× more allocs** |
| Lower B/op (vs 2026-08-07 reported final 1.91 GB) | **~4.04 GB** | **~2.1× more heap traffic** |
| Peak RSS vs wkhtmltopdf 500 | 777 MB vs 114 MB | **~6.8× higher RSS** |

The 2026-08-07 “final” 6.93 s / 1.91 GB / 5.61 M numbers are **not reproduced**
on this 2026-08-08 tip measurement. Treat this addendum as the new 500-page
baseline for optimization work; re-run count-3 before claiming regressions or
wins.

### Optimization priorities (evidence-ordered)

1. **Stop full `prefixMaxOfOps` rebuilds per forced break** — incremental max
   maintenance or single-pass forced-break placement without O(n²) rescans.
2. **Collapse style/cascade allocations** — especially `inheritableProps` and
   per-node cascade maps; reuse tables; avoid re-walking sheets per node.
3. **Reduce shift work for multi-break reports** — place sections by arithmetic
   when input is a repeated `page-break-before:always` template pattern.
4. **Only then** parallel paint/compress (deferred one-second plan item).

Canonical next checklist:
[`plans/deferred/0.0.3/500-page-allocation-and-latency-optimization-plan.md`](../../deferred/0.0.3/500-page-allocation-and-latency-optimization-plan.md)

Parent one-second vision (unchanged file):
[`plans/deferred/0.0.3/500-page-one-second-performance-target.md`](../../deferred/0.0.3/500-page-one-second-performance-target.md)

## Current publication addendum (2026-08-09)

The profile-guided residual wave is now committed as `aa8d446` and pushed on
`feature/optimization`. The authoritative one-iteration matrix is
`testdata/golden/benchmarks/benchmark-results.txt`; the locked 500-page gate
uses the separate count-3 median.

### Current versus first committed snapshot

The first benchmark file on this branch is `2a0f18b`. Using the same
one-iteration command for both snapshots, the 500-page PDF changed from
**14.135s / 3.80GB / 14.35M allocs** to **1.903s / 678.6MB / 1.103M allocs**:

| Metric | Original `2a0f18b` | Current `aa8d446` | Change |
|---|---:|---:|---:|
| PDF time | 14.135s | 1.903s | **−86.5%** |
| PDF B/op | 3.80GB | 678.6MB | **−82.1%** |
| PDF allocs/op | 14.35M | 1.103M | **−92.3%** |
| Template + PDF time | 13.670s | 1.693s | **−87.6%** |
| Template + PDF B/op | 3.80GB | 683.5MB | **−82.0%** |
| Template + PDF allocs/op | 14.40M | 1.154M | **−92.0%** |

### wkhtmltopdf reference comparison

The earlier process-level reference used wkhtmltopdf 0.12.6.1 on the same
generated report matrix. It measured **2.05s / approximately 114MB peak RSS /
approximately 2.0MB PDF** at 500 pages. The current Go values below are
in-process benchmark values, so the table is directional rather than a strict
process-to-process comparison:

| Pages | wk wall | wk peak RSS | Current Go wall | Current Go B/op | Current Go allocs/op |
|---:|---:|---:|---:|---:|---:|
| 2 | 0.39s* | 34MB | 6.7ms | 3.8MB | 5,075 |
| 5 | 0.23s | 35MB | 11.6ms | 6.1MB | 10,928 |
| 10 | 0.25s | 35MB | 22.2ms | 11.3MB | 20,970 |
| 20 | 0.28s | 37MB | 41.1ms | 21.8MB | 40,768 |
| 50 | 0.37s | 42MB | 114.6ms | 52.8MB | 98,585 |
| 100 | 0.60s | 50MB | 249.0ms | 100.9MB | 188,179 |
| 200 | 0.89s | 66MB | 481.4ms | 200.1MB | 370,373 |
| 250 | 0.99s | 74MB | 590.2ms | 248.9MB | 461,490 |
| 500 | 2.05s | 114MB | 1.903s | 678.6MB | 1,102,840 |

`B/op` is cumulative allocation traffic, not RSS. The current profiled Go
benchmark process reached approximately **391MiB RSS** at 500 pages; a fresh
current CLI process-level matrix has not been recorded after `aa8d446`. The
earlier CLI comparison remains **8.40s / ~777MB** for gowkhtmltopdf versus
**2.05s / ~114MB** for wkhtmltopdf and must remain labeled historical.

\* The first wkhtmltopdf invocation was colder because of Qt startup.
