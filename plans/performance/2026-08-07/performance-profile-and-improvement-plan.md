# Performance - Profiling and Improvement Checklist

> **Parent:** `skills/phase-wise-checklist/SKILLS.md` - canonical phase-wise execution ledger
> **Status:** profiling complete; implementation not started
> **Date:** 2026-08-07
> **Baseline commit:** `93f32e7cf8c6` (`master`, merged benchmark work)
> **Target:** 10/10 performance after the implementation and validation gates below

---

## Overview

This report profiles the deterministic benchmark matrix landed by the benchmark
change. It keeps the original one-iteration snapshot as the published baseline,
records a current rerun on the same checkout, and identifies measured hot paths
for the next performance-improvement branch.

All profiling artifacts were written to `/tmp` during this pass. The canonical
report, checklist, current-code snippets, and proposed snippets are kept in
this dated directory. No separate findings directory is used.

## Executive Summary

- The current performance rating is **5.1/10**.
- The strongest confirmed PDF hotspot is pagination cleanup: `stripOrphanRowChrome`
  accounts for **5.41 s flat / 29.73%** of CPU samples in the 500-page profile.
- PDF allocation pressure is material: the current 500-page rerun reports
  **3,797,733,888 B/op** and **14,387,130 allocs/op**.
- Image mode is dominated by raster work, not the local HTTP fetch: `pointInPoly`
  is **23.14% flat**, while `scaleNearest` is **24.17% cumulative** in the
  500-image inline profile.
- The first implementation wave should remove repeated page-wide scans and
  avoid per-pixel interface/color conversions. Correctness and visual output
  must remain unchanged.

## Current Performance Rating

This rating describes current measured performance and optimization readiness,
not feature completeness. The arithmetic is intentionally visible:

| Area | Weight | Current score | Evidence |
|---|---:|---:|---|
| Reproducible benchmark coverage | 1.5 | 1.5 | Deterministic PDF, template, web-image, and inline-image matrices from 2 to 500 workloads |
| PDF throughput | 2.5 | 0.8 | 500-page PDF is 14.14 s in the landed snapshot and 14.15 s in the current rerun |
| Pagination scaling | 2.0 | 0.4 | Repeated page/operation scans are visible in `stripOrphanRowChrome`, `beforeAlways`, and `shiftFlowY` |
| Image throughput | 1.5 | 0.9 | 500-image inline rerun is 839.7 ms; raster work remains expensive but bounded |
| Allocation efficiency | 1.5 | 0.5 | 3.80 GB and 14.39 M allocations per 500-page PDF operation |
| Profiling and measurement discipline | 1.0 | 1.0 | Current CPU/heap profiles, exact commands, host details, and correctness-aware gates are recorded |
| **Total** | **10.0** | **5.1** | **0.8 + 0.4 + 0.9 + 0.5 + 1.5 + 1.0 = 5.1/10** |

The **10/10 target** is reached only when the proposed gates in Phase 5 pass
on a repeated benchmark run and the full test/lint suite remains green. The
thresholds are optimization targets, not claims about the current branch.

## Baseline and Current Rerun

The landed baseline is the checked-in snapshot in
`testdata/golden/benchmarks/benchmark-results.txt`. It was recorded with Go
1.26.4 on Linux/amd64 under WSL2, 24 CPUs. The current rerun used the same
command and host family; one-iteration measurements naturally vary, so the
profile runs are evidence of hotspot ordering rather than replacement timing
baselines.

| Workload | Landed baseline | Current rerun | Current allocations |
|---|---:|---:|---:|
| PDF, 500 pages | 14.14 s | 14.15 s | 3,797,733,888 B/op; 14,387,130 allocs/op |
| Template + PDF, 500 pages | 13.67 s | 12.30 s | 3,802,615,784 B/op; 14,438,471 allocs/op |
| Web-fetch image, 500 images | 970.72 ms | 821.09 ms | 214,811,758 B/op; 13,893,226 allocs/op |
| Inline image, 500 images | 788.43 ms | 839.68 ms | 215,184,744 B/op; 13,892,716 allocs/op |

The complete matrix remains in the benchmark fixture report; these 500-unit
rows are the profile targets because they expose scaling costs most clearly.

### Reproduction commands

Published snapshot:

```sh
GOCACHE=/tmp/gowkhtmltopdf-profile-cache \
  go test ./internal/convert -run '^$' \
  -bench 'Benchmark(PDFPages|TemplatePages|WebFetchImage|ImageAssets)$' \
  -benchmem -benchtime=1x -count=1
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

### Image rasterization

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

## Current Code and Proposed Improvements

The snippets below are intentionally plan-only. They describe bounded changes
to validate with focused tests and benchmarks; no production implementation is
claimed by this report.

### P0 - Make orphan-row cleanup linear in operations

Current code in `internal/layout/paint.go:856-1012` loops over every page and
then scans all operations for each page, with additional full operation scans
for trimming. This is the confirmed 500-page CPU hotspot.

Current shape:

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

Current shape:

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

Current shape:

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

Current shape:

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

Current shape:

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
- [x] Rate the current state at 5.1/10 and define the 10/10 acceptance gates below.

### Phase 2: PDF pagination cost

- [ ] Replace page-by-page full-operation scans in `internal/layout/paint.go:856-1012` with page buckets or equivalent indexed spans; prove identical page assignment and fixture output.
- [ ] Reduce `shiftFlowY` full-list/full-tree rescans in `internal/layout/paint.go:1029-1056`; prove collapsed-margin, sticky/fixed, and nested-box semantics.
- [ ] Make `beforeAlways` use mutation-safe prefix/index metadata; prove forced-section ordering and fixpoint convergence.
- [ ] Index horizontal/vertical candidates in `capTablePageBreaks`; prove continuation-border fixtures remain closed without duplicate lines.

### Phase 3: Layout and PDF allocation cost

- [ ] Replace per-rune string concatenation in `splitTextByFace`; cover mixed Unicode fallback and compare `alloc_objects` before/after.
- [ ] Measure `applyRestProps`, `cascadeRaw`, selector matching, and table display-list emission with focused benchmarks before introducing caches or reusable maps.
- [ ] Only retain allocation changes that preserve PDF bytes/page counts where the existing tests require determinism and preserve all layout golden expectations.

### Phase 4: Image raster cost

- [ ] Add a pixel-equivalent `*image.NRGBA` fast path to `scaleNearest`; retain and test the generic fallback.
- [ ] Add a direct-pixel `downscaleBox` path and benchmark supersampled output against current PNG pixels.
- [ ] Precompute glyph edge metadata for `pointInPoly`/`pointInGlyph`; prove glyph cache hit/miss behavior and text visual parity.
- [ ] Evaluate any `rasterSS` or antialias-quality change separately; do not mix quality changes with algorithmic speedups.

### Phase 5: 10/10 validation and closure

- [ ] Re-run the full deterministic matrix with `-count=3`, report median and spread, and keep the original one-iteration baseline visible.
- [ ] Meet the proposed 10/10 performance gates: 500-page PDF and template conversion at or below 7.1 s, at or below 1.9 GB/op and 7.2 M allocs/op; 500-image inline and web conversion at or below 0.45 s, at or below 110 MB/op and 7.0 M allocs/op.
- [ ] Confirm the optimized profile no longer has a single pagination cleanup pass above 10% flat CPU and that no replacement hotspot violates correctness or memory constraints.
- [ ] Run `go test ./...`, `make lint`, focused benchmark/profile commands, and `git diff --check`; record exact outcomes before closing this phase.
- [ ] Re-score the same six weighted areas and publish the final score as 10/10 only if every target gate and correctness gate passes.

## Dependencies and Risks

- Pagination changes depend on preserving operation ordering, fixed/sticky exclusions, element-location ranges, and page-break fixpoint behavior.
- Raster changes depend on Go image type and alpha semantics; a faster conversion that changes premultiplied colors is not acceptable.
- Allocation reductions must be measured with heap profiles and benchmark allocations, not inferred from source shape alone.
- The 500-page profile is intentionally a stress workload. Improvements must also be checked at 2, 10, 50, and 100 units so a large-document optimization does not regress normal use.

## Validation Boundary

This commit contains documentation only. The profiling commands and benchmark
reruns passed on 2026-08-07. No production implementation has been made, so
the Phase 2–5 implementation rows remain unchecked and the current score stays
5.1/10.
