# 0.2.6 review - golden corpus memory profiling and benchmarks (2026-09-10)

> **Parent:** `plans/0.2.6/review/architecture-deepening-2026-09-10.md` - companion performance evidence
> **Status:** harness built, benchmarks and heap/stack profiles captured. All measurement rows are `[x]`. Phase 6 adds 13 `IMPROV` proposals; they are `[ ]` and none are implemented.
> **Estimated effort:** one profiling wave (harness plus three modes). Phase 6 is a suggestion list, not implementation work.
> **Date:** 2026-09-10
> **Toolchain:** go1.26.4 linux/amd64, 24 vCPU (i7-13700HX)
> **Harness:** `internal/profiling/` (new, opt-in; never runs in `make test`)

---

## Overview

This wave profiles the two supported features against the golden corpus:

- **PDF**: `Document.WritePDF` over 65 golden templates.
- **Image**: `ImageDocument.WriteImage` over the same templates, PNG and JPEG.

The corpus has 67 HTML files; two are header/footer companions for fixture-36 and are attached to
their body fixture rather than counted as documents. Four templates (complex-css, fixture-56,
fixture-60, font-examples) exceed the image raster budget at 1x and are recorded as skipped for
image modes only. PDF handles all 65.

The harness is new and lives in `internal/profiling/`. It is opt-in through environment variables,
so the normal suite neither runs the profiles nor the benchmarks:

- `GOWK_PROFILE_DIR` enables `TestProfileGoldenCorpus` (heap, allocs, goroutine profiles plus a
  per-fixture memory summary).
- `BenchmarkGoldenPDF`, `BenchmarkGoldenImagePNG`, `BenchmarkGoldenImageJPEG` time one full corpus
  pass per iteration; the `*ByFixture` variants time each template separately.
- No existing benchmark, Makefile target, or script was changed.

Generated artifacts live in `output/profiles/2026-09-10/` and are gitignored through the new
`output/profiles/` entry in `.gitignore`. Only this report and the README index are tracked.

## Executive summary

| Metric | PDF | PNG | JPEG |
|--------|----:|----:|-----:|
| Templates converted | 65 | 61 (+4 skipped) | 61 (+4 skipped) |
| Benchmark, whole corpus | 4.95 s/op | 11.14 s/op | 9.62 s/op |
| Benchmark B/op | 2.26 GB | 4.36 GB | 4.68 GB |
| Benchmark allocs/op | 21.7M | 8.6M | 105.6M |
| Profile total allocations | 2.18 GiB | 5.22 GiB | 5.53 GiB |
| Peak heap after one conversion | 61.4 MiB | 204.1 MiB | 232.2 MiB |
| Live heap after two GCs | 10.3 MiB | 10.3 MiB | 10.3 MiB |
| Stack in use, final | 864 KiB | 864 KiB | 1.0 MiB |
| Goroutines, before to after | 2 to 2 | 2 to 2 | 2 to 2 |

There is no leak-class retention: after two forced GCs the process keeps about 10.3 MiB live in
roughly 61k objects in every mode, and goroutines do not grow. The cost is allocation traffic, not
retained memory. PDF's top allocator is input reading (`io.ReadAll`, 858 MB, 39.5%); image modes are
dominated by rasterization buffers (`rasterizeContext`, 28% flat and 60% cumulative of 5.2 GiB).
JPEG allocates 105.6M objects, about 12x PNG, from its encoder.

The profiles also yield 13 ranked improvement proposals in Phase 6, led by a cross-conversion font
cache (about -38% PDF B/op warm) and a YCbCr fast path for `jpeg.Encode` (allocs/op from 105.6M to
an estimated 10-13M). None are implemented.

## Phase 1: Harness

### 1.1 New profiling package

- [x] **PROF-01** Create `internal/profiling/doc.go`, `profiling_test.go`, and `benchmark_test.go`
      only. The harness exercises the public `Document`/`ImageDocument` entry points, mirrors the
      golden corpus setup (A4, background, local files allowed, test fonts, fixture-36 companions),
      and skips nothing silently: unsupported image templates are recorded as skipped with the
      raster-budget reason. Proof: `go vet ./internal/profiling`, `gofmt -l internal/profiling`,
      and `go test ./internal/profiling -run '^$'` (compiles, no tests run).

- [x] **PROF-02** Add heap/stack capture: `TestProfileGoldenCorpus` records per-fixture
      `TotalAlloc`, mallocs, frees, retained heap, `HeapInuse`, and `StackInuse`, then writes
      `heap-<mode>.pprof`, `allocs-<mode>.pprof`, `goroutine-<mode>.pprof`, and
      `stacks-<mode>.txt` plus JSON and Markdown summaries. Proof: three summary files exist and
      report `failures=0`.

- [x] **PROF-03** Add golden-corpus generation benchmarks: whole-corpus and per-template for PDF,
      PNG, and JPEG, all with `b.ReportAllocs()`. Proof: `bench-aggregate.txt` and
      `bench-by-fixture.txt` contain 65 PDF rows and 61 PNG plus 61 JPEG rows.

- [x] **PROF-04** Keep generated profiles out of version control. `.gitignore` now ignores
      `output/profiles/`. Proof: the ignore entry is scoped to the generated directory; the report
      and README stay under `plans/`.

## Phase 2: Benchmarks

### 2.1 Whole-corpus generation time

- [x] **BENCH-01** PDF: **4.95 s/op, 2.26 GB/op, 21,715,918 allocs/op** over 65 templates,
      median of 3 passes. Command:
      `go test ./internal/profiling -run '^$' -bench '^BenchmarkGolden(PDF|ImagePNG|ImageJPEG)$' -benchtime=3x -benchmem -count=1`.

- [x] **BENCH-02** PNG: **11.14 s/op, 4.36 GB/op, 8,577,746 allocs/op** over the 61 rasterable
      templates, median of 3 passes.

- [x] **BENCH-03** JPEG: **9.62 s/op, 4.68 GB/op, 105,586,188 allocs/op** over the same 61
      templates, median of 3 passes. The alloc count is the outlier: about 12x PNG for similar
      bytes.

- [x] **BENCH-04** Image budget skips recorded: complex-css (22415 px), fixture-56 (19228 px),
      font-examples (52518 px) exceed the 16384 max dimension, and fixture-60 exceeds the pixel
      budget at 100,057,088 pixels. These are feature limits of 1x rasterization, not harness
      errors.

### 2.2 Per-template generation time (top 8 per mode)

Per-template rows come from `-bench 'Golden.*ByFixture$' -benchtime=100ms`. Fast templates loop;
templates slower than 100 ms run once, so treat the largest values as single-shot.

| Rank | PDF | PNG | JPEG |
|-----:|-----|-----|------|
| 1 | font-examples 777 ms | fixture-59 1299 ms | fixture-59 1439 ms |
| 2 | complex-css 736 ms | fixture-53 1020 ms | fixture-57 877 ms |
| 3 | fixture-59 352 ms | fixture-49 979 ms | fixture-58 745 ms |
| 4 | fixture-58 330 ms | fixture-57 788 ms | fixture-62 572 ms |
| 5 | fixture-57 304 ms | fixture-58 748 ms | fixture-61 464 ms |
| 6 | fixture-54 284 ms | fixture-62 617 ms | fixture-49 427 ms |
| 7 | fixture-51 266 ms | fixture-47 506 ms | fixture-53 422 ms |
| 8 | fixture-60 253 ms | fixture-51 474 ms | fixture-43 351 ms |

The slow set is consistent: the two documentation-heavy templates (font-examples, complex-css) lead
PDF, and the poster/storybook templates (59, 53, 49, 57, 58) lead image modes. Image generation is
2-3x the PDF time for the same template.

## Phase 3: Heap profiles

Allocation numbers below are `go tool pprof -top -alloc-space` over the per-mode `allocs-*.pprof`
files. "Flat" is the site's own bytes; "cum" includes callees.

### 3.1 PDF

- [x] **HEAP-01** Capture PDF heap and allocs profiles plus per-fixture totals from one process.
      Profile totals: 2.18 GiB over 21.7M mallocs, peak per conversion 61.4 MiB, live after two GCs
      10.3 MiB in 61,004 objects.

- [x] **HEAP-02** Top PDF allocation sites:

      | Flat | Share | Site |
      |-----:|------:|------|
      | 858.1 MB | 39.5% | `io.ReadAll` |
      | 191.3 MB | 8.8% | `compress/flate.NewWriter` |
      | 162.8 MB | 7.5% | `image.NewRGBA` |
      | 95.2 MB | 4.4% | `bytes.growSlice` |
      | 81.4 MB | 3.7% | `image.NewNRGBA` |
      | 77.5 MB | 3.6% | `image.(*RGBA).At` |
      | 62.1 MB | 2.9% | `pdf.(*Font).parseCmap12` |
      | 60.1 MB | 2.8% | `layout.(*styleStore).append` |
      | 57.6 MB | 2.7% | `pdf.renderImagePixels` (135.6 MB cum) |
      | 39.5 MB | 1.8% | `pdf.flateBytes` (181.4 MB cum) |

### 3.2 PNG

- [x] **HEAP-03** Capture PNG profiles. Profile totals: 5.22 GiB over 9.6M mallocs, peak per
      conversion 204.1 MiB, live after two GCs 10.3 MiB in 61,019 objects.

- [x] **HEAP-04** Top PNG allocation sites:

      | Flat | Share | Site |
      |-----:|------:|------|
      | 1471.7 MB | 28.2% | `imageout.rasterizeContext` (3113.6 MB cum, 59.6%) |
      | 899.8 MB | 17.2% | `image.NewNRGBA` |
      | 887.6 MB | 17.0% | `io.ReadAll` |
      | 317.3 MB | 6.1% | `compress/flate.NewWriter` |
      | 270.4 MB | 5.2% | `imageout.rasterGlyphAlpha` |
      | 131.4 MB | 2.5% | `layout.appendDashedLineSegments` |
      | 112.8 MB | 2.2% | `layout.mergeDeferredChrome` |
      | 99.0 MB | 1.9% | `pdf.appendFlattened` |
      | 94.7 MB | 1.8% | `image.NewAlpha` |
      | 90.4 MB | 1.7% | `layout.(*engine).prependChrome` (627.1 MB cum) |

### 3.3 JPEG

- [x] **HEAP-05** Capture JPEG profiles. Profile totals: 5.53 GiB over 106.6M mallocs, peak per
      conversion 232.2 MiB, live after two GCs 10.3 MiB in 61,021 objects. JPEG trails PNG slightly
      in bytes but leads it 12x in allocation count.

- [x] **HEAP-06** Top JPEG allocation sites:

      | Flat | Share | Site |
      |-----:|------:|------|
      | 1471.2 MB | 26.6% | `imageout.rasterizeContext` (3111.0 MB cum, 56.2%) |
      | 905.1 MB | 16.3% | `image.NewNRGBA` |
      | 887.9 MB | 16.0% | `io.ReadAll` |
      | 370.5 MB | 6.7% | `image.(*NRGBA).At` |
      | 288.5 MB | 5.2% | `imageout.rasterGlyphAlpha` |
      | 258.3 MB | 4.7% | `compress/flate.NewWriter` |
      | 137.0 MB | 2.5% | `layout.appendDashedLineSegments` |
      | 112.6 MB | 2.0% | `layout.mergeDeferredChrome` |
      | 103.0 MB | 1.9% | `layout.(*styleStore).append` |
      | 92.0 MB | 1.7% | `pdf.appendFlattened` |

### 3.4 Retention check

- [x] **HEAP-07** Two forced GCs at the end of every mode return the heap to about 10.3 MiB in
      about 61k objects. Retained per-fixture values stay in the KiB range for the top
      contributors; the only larger retentions are parsed font/tree data for font-examples
      (18.2 MiB) and fixture-27 (17.6 MiB) in PDF. No growth pattern across the corpus.

## Phase 4: Stack and goroutines

- [x] **STACK-01** Goroutine profiles (`stacks-*.txt`, `goroutine-*.pprof`) show a constant total
      of 2: the test runner and the runtime main. No engine goroutine persists between conversions,
      and every mode reports 2 before and 2 after.

- [x] **STACK-02** Stack memory stays flat: final `StackInuse` is 864 KiB for PDF and PNG and
      1.0 MiB for JPEG, with `StackSys` equal to `StackInuse` in each run. The engine is
      single-goroutine, so there is no stack pool growth to report.

## Phase 5: Hotspot evidence (inputs to Phase 6)

These are the raw profile hotspots. Phase 6 turns them into ranked proposals, each with its own
before/after benchmark plan.

- **Input reading is the single biggest PDF allocator.** `io.ReadAll` at 858 MB (39.5%) reads
  fixture files, fonts, and embedded assets again for every conversion. A cross-conversion byte
  cache for fonts/assets (keyed by resolved path) is the obvious experiment.
- **Image rasterization dominates both image modes.** `rasterizeContext` is 28% flat and 60%
  cumulative, with `image.NewNRGBA` adding another 900 MB. The supersample pool exists
  (`BenchmarkSupersamplePool`); per-template buffers still churn.
- **JPEG encoder allocation count is the outlier.** 105.6M allocs/op versus 8.6M for PNG, with
  `image.(*NRGBA).At` and `compress/flate.NewWriter` in the top rows. Encoder buffer reuse is the
  first thing to try.
- **Layout chrome traffic repeats across formatting contexts.**
  `appendDashedLineSegments` (131-137 MB), `mergeDeferredChrome` (113 MB), `prependChrome`
  (90 MB flat, 627 MB cumulative), and `styleStore.append` (60-103 MB) appear in all three modes.
  These are the best shared-target candidates.
- **Font cmap parsing shows up in every mode.** `parseCmap12` is 62 MB in PDF and 62 MB in JPEG;
  fonts are cached per conversion, not across the corpus.
- **Four templates cannot rasterize at 1x.** complex-css (22415 px), font-examples (52518 px),
  fixture-56 (19228 px), and fixture-60 (100M pixels) exceed the image budget. This is a documented
  feature limit; a zoomed-down profile variant would be needed to cover them.

## Phase 6: Suggested improvements (IMPROV)

Ranked proposals from the Phase 3 hotspots. None are implemented; every row starts `[ ]` and each
needs its own before/after benchmark before code lands. Expected wins are estimates from the
captured profiles, not measured results.

| Row | Area | Expected win | Risk |
|-----|------|--------------|------|
| IMPROV-01 | PDF | about -824 MB corpus allocations, warm `BenchmarkGoldenPDF` B/op -38% | shared immutable `*Font` plus a bounded cache |
| IMPROV-02 | JPEG | allocs/op 105.6M to about 10-13M, B/op about -230 MB | YCbCr conversion must match stdlib bytes |
| IMPROV-03 | Image | about -446 MB (8% of the PNG profile) | cache budget policy, retained-heap ceiling |
| IMPROV-04 | Image | about -396 MB (7.6%) | none, scratch reset per glyph |
| IMPROV-05 | Layout | about 300-400 MB across the three profiles | none, same crop bytes reused |
| IMPROV-06 | Layout | about -200 MB across profiles | capacity only, ops unchanged |
| IMPROV-07 | PDF | about -78 MB (3.6%) | bit-exact concrete color accessors |
| IMPROV-08 | PNG | about -210 MB on fixture-62 | clip margin must stay conservative |
| IMPROV-09 | Layout | 23-113 MB per profile when spare capacity fits | in-place merge order must stay exact |
| IMPROV-10 | Image | about -26 to -53 MB in both image modes | conversion must match `color.NRGBAModel` |
| IMPROV-11 | PDF | about -405 MB cold, subsumed warm by IMPROV-01 | keep the size cap and race fallback |
| IMPROV-12 | JPEG | about -55 MB | bounded pooled buffer retention |
| IMPROV-13 | Image | none, actionable budget errors and docs | none |

- [ ] **IMPROV-01 · PDF** Cache parsed font files across conversions.
      `internal/pdf/registry.go:370` (`scanFontFile`). `go tool pprof -peek=io.ReadAll` attributes
      747.8 MB (87%) of the PDF profile to `scanFontFile`; all 65 conversions rescan the same
      10 TTF files (cum 838.5 MB). Change: bounded, mutex-guarded cache keyed on absolute path +
      size + mtime, shaped like `semanticRegexCache`, sharing immutable parsed fonts; keep the
      per-conversion `*Registry` fresh because `mergeFontFace` mutates it. Expected: same-corpus
      B/op about -38%; a cold CLI process gains nothing. Proof: `BenchmarkGoldenPDF -benchmem
      -benchtime=3x` before/after plus a profile rerun confirming `io.ReadAll` and `parseCmap12`
      leave the top rows, and a warm-vs-cold output hash. Not: caching a whole `*Registry`
      (font-face leakage) or raw bytes (the parse cost remains).

- [ ] **IMPROV-02 · JPEG** Hand `jpeg.Encode` a `*image.YCbCr` instead of `*image.NRGBA`.
      `internal/imageout/imageout.go:2028` (`jpeg.Encode`). The stdlib has direct `Pix` paths for
      RGBA and YCbCr; NRGBA goes through `toYCbCr`, which boxes a color per pixel (`NRGBA.At`
      370.5 MB, the 97M allocs/op gap versus PNG). Change: one `nrgbaToYCbCr420` conversion matching
      stdlib semantics (2x2 chroma average, clamped coordinates) before encoding. Expected:
      allocs/op about 105.6M to 10-13M and B/op about -230 MB; output bytes stay identical. Proof:
      the same benchmark before/after, plus `-peek='jpeg\.toYCbCr'` losing its `NRGBA.At` leaf.
      Not: alpha flattening only (still NRGBA) or a `*image.RGBA` conversion (4 bytes per pixel
      instead of 1.5 and still runs RGBToYCbCr in the encoder).

- [ ] **IMPROV-03 · Image** Byte-budgeted, GC-proof supersample canvas cache.
      `internal/imageout/imageout.go:425-474`. `rasterizeContext` spends 1.44 GB flat on canvas
      makes; 48 sub-32 MiB canvases (446 MB) re-make per fixture because `sync.Pool` is emptied by
      the harness GCs and buffers above the 32 MiB cap are never retained. Change: mutex-guarded
      buffer cache returning the smallest fitting buffer, bounded per buffer and in total bytes.
      Expected: about -446 MB at today's retention bound; raising the budget recovers the 1.0 GB
      large-buffer cohort but pins up to the largest canvas. Proof: PNG benchmarks before/after plus
      the final-heap-after-GC check. Not: `sync.Pool` size classes (GC still clears them) or
      silently raising retention.

- [ ] **IMPROV-04 · Image** Reuse glyph edge and active-row scratch.
      `internal/imageout/ttfraster.go:294-303`, `:389-394`. `rasterGlyphAlpha` allocates a fresh
      active-row slice per glyph (270.4 MB flat) and `makeGlyphEdgeList`/`makeGlyphEdges` add
      127 MB cum. Change: one `glyphScratch` in a `sync.Pool` (edges plus active rows) reset per
      glyph. Expected: about -396 MB (7.6%) with identical coverage math. Proof: PNG and JPEG
      benchmarks plus `go test ./internal/imageout`; profile flat at `ttfraster.go:302` and `:394`.
      Not: a cross-render glyph bitmap cache; the atlas is deliberately per run.

- [ ] **IMPROV-05 · Layout** Cache border-image slice crops on the resolved image ref.
      `internal/layout/border_image.go:468`, `:482`, `internal/layout/layout_flow.go:63`.
      `cropBorderImage` is 109-287 MB per profile (660 MB across all three) because each of
      fixture-60's 8 border-image elements re-encodes the same 24 source rects; `resolveImage`
      caches the decode only. Change: add a per-ref `map[image.Rectangle][]byte` crop cache; refs
      are engine-local and single-goroutine. Expected: about 300-400 MB less across the profiles
      with identical bytes. Proof: benchmarks for all modes, `go test ./internal/layout`, and
      `make golden`. Not: a package-global cache or decode caching.

- [ ] **IMPROV-06 · Layout** Size the dashed-border op slice once per box.
      `internal/layout/layout_chrome.go:57`, `:137`. Four-sided dashed boxes allocate n + 2n + 3n +
      4n op slots for 4n ops; the site is 131-137 MB in image modes. Change: compute the per-side
      segment upper bound, sum the active sides, and allocate once; keep the existing growth
      fallback. Expected: about -200 MB across profiles, capacity only, op order unchanged. Proof:
      benchmarks plus layout tests plus `make golden`. Not: `slices.Grow` or append growth, which
      over-allocates.

- [ ] **IMPROV-07 · PDF** Read concrete pixel types in `renderImagePixels`.
      `internal/pdf/images.go:419-454`. `img.At(...)` boxes a color per pixel, 77.5 MB. Change:
      type switch once for `*image.RGBA` and `*image.NRGBA` using the concrete accessors with the
      same RGBA math, generic `At` fallback. Expected: about -78 MB (3.6%) per PDF run. Proof: PDF
      benchmarks and `go test ./internal/pdf -run 'TestAddPNGImage|TestGrayscalePNG'`, with a
      bit-exact comparison against the generic path. Not: caching decoded PNGs (retains 230 MB) or
      hand-rolled premultiplication (differs from stdlib rounding).

- [ ] **IMPROV-08 · PNG** Bound the blend scratch to the op rectangle.
      `internal/imageout/blend.go:18`. `paintBlended` copies the whole canvas for one
      `mix-blend-mode` element (210.6 MB, 4% of the PNG profile). Change: extract the conservative
      bounds helper already used by `paintTransformedOp` and allocate the scratch at the op rect.
      Expected: about -210 MB. Proof: fixture-62 PNG benchmark before/after plus decoded pixel
      parity and `make golden`. Not: a reused full-canvas scratch (peak RSS unchanged).

- [ ] **IMPROV-09 · Layout** Merge deferred chrome in place from the op buffer's spare capacity.
      `internal/layout/layout_chrome.go:674`, allocation at `:702`. The merge allocates a second
      display list (23-113 MB per profile) although `e.ops` has near-final capacity. Change: when
      spare capacity fits, fill backwards in place; otherwise one exact copy. Expected: the site
      goes to zero when it fits. Proof: PDF, PNG, and JPEG benchmarks, layout tests, `make golden`,
      and a profile recheck. Not: a second workspace buffer (the main path does not use
      `Workspace`).

- [ ] **IMPROV-10 · Image** Remove per-pixel color boxing in `scaleNearestGeneric`.
      `internal/imageout/imageout.go:1639-1676`. `color.NRGBAModel.Convert(src.At(...))` boxes the
      source and converted colors; `image.(*YCbCr).At` is 26.5 MB plus 1.7M sampled objects, and the
      PNG profile shows the same at the NRGBA model. Change: fast paths for `*image.YCbCr`,
      `*image.NRGBA`, and `*image.RGBA` with the stdlib formula, `Convert` fallback. Expected:
      about -26 to -53 MB in both image modes. Proof: PNG and JPEG benchmarks. Not: an
      `x/image/draw` dependency, which the allowlist forbids.

- [ ] **IMPROV-11 · PDF** Read stat-sized font files in `scanFontFile`.
      `internal/pdf/registry.go:386`. `io.ReadAll` allocates 11.5 MB to hold 5.2 MB of fonts
      (482 MB in intermediate chunks plus 375 MB in the final copy). Change: allocate from the
      already-known `info.Size()` and use `io.ReadFull`, keeping the
      `io.ReadAll(io.LimitReader(...))` fallback for short or grown reads. Expected: about -405 MB
      if measured before IMPROV-01; after IMPROV-01 it only helps the first scan and the one-shot
      CLI. Proof: a cold-corpus benchmark or a `BenchmarkScanFontFile` micro-benchmark. Not:
      `os.ReadFile` (reopens the path and drops the 32 MiB cap).

- [ ] **IMPROV-12 · JPEG** Stop the encode buffer growing from zero and skip the bufio layer.
      `internal/imageout/imageout.go:2011-2054`, `:1850-1866`. `bytes.Buffer` growth is 54.4 MB and
      `jpeg.Encode` wraps a `bufio.Writer` because `limitedImageBuffer` has no `Flush`. Change: add
      `Flush() error { return nil }` and pool or presize the buffer, bounded by `maxImageEncoded`.
      Expected: about -55 MB, same bytes. Proof: JPEG benchmark B/op. Not: streaming straight into
      `req.Output` (breaks the no-partial-output guarantee).

- [ ] **IMPROV-13 · Image** Make the raster budget failure actionable and document the envelope.
      `internal/imageout/imageout.go:531-570`, `:44-49`. The four skips report supersampled
      dimensions with no unit, limit, or remedy; the CSS heights are actually 11208, 26259, and
      9614 px. Change: include the CSS px size, the limit, and the maximum fitting zoom (for
      example `--zoom <= 0.73`) while keeping the wrapped `errRasterTooLarge`, and document the
      envelope in `documentation/architecture/10-imageout-svg.md` and `documentation/cli.md`.
      Expected: no allocation change; the four skips become actionable. Proof:
      `go test ./internal/imageout -run TestRaster -count=1` and `make test`. Not: an automatic
      zoomed fallback or tiled render (changes output geometry or just moves the buffer).

## Dependencies

```text
PROF-01..04 (harness)        -- independent, already landed
BENCH-01..04 (timing)        -- needs PROF-01..03 only
HEAP-01..07 (profiles)       -- needs PROF-02
STACK-01..02 (goroutines)    -- produced by the same runs as HEAP-*
IMPROV-01 (font cache)       -- independent; warm-path only, largest PDF win
IMPROV-02 (JPEG YCbCr)       -- independent; largest alloc-count win
IMPROV-03..12                -- independent slices, each needs its own before/after benchmark
IMPROV-11                    -- subsumed by IMPROV-01 on the warm path; keep for the cold path
IMPROV-13 (budget UX)        -- error message plus docs only
```

## Artifacts

All under `output/profiles/2026-09-10/` and gitignored:

- `heap-{pdf,image-png,image-jpeg}.pprof`, `allocs-*.pprof`, `goroutine-*.pprof`
- `stacks-*.txt`, `summary-*.json`, `summary-*.md`
- `bench-aggregate.txt`, `bench-by-fixture.txt`, `top-alloc-*.txt`, `top-inuse-*.txt`

Reproduce:

```text
go test -c -o output/profiles/2026-09-10/profiling.test ./internal/profiling
GOWK_PROFILE_DIR=output/profiles/2026-09-10 GOWK_PROFILE_MODE=pdf \
  ./output/profiles/2026-09-10/profiling.test -test.run '^TestProfileGoldenCorpus$' -test.count=1
go tool pprof -top -nodecount=14 -alloc_space \
  output/profiles/2026-09-10/profiling.test output/profiles/2026-09-10/allocs-pdf.pprof
```

## What this wave did not do

- No source changes and no fixes to any hotspot. Phase 6 lists 13 ranked improvement proposals;
  each has its own before/after benchmark plan and none are implemented.
- No CPU or block profiling; this wave covers heap and stack only, as requested.
- No changes to existing benchmarks, Makefile bench targets, or `scripts/bench-external.sh`.
- No commits, pushes, or any git command. The harness is new code under `internal/profiling/`,
  and the generated profiles are ignored by `.gitignore`.
