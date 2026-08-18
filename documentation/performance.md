# Performance

Numbers on this page are **labeled snapshots**, not a live SLA. Host, GOCACHE
state, and whether a run used the **generic** convert path or the
**benchmark-only page-island** path all change wall time and RSS.

**Current snapshot: 2026-08-14.** Freshly built generic CLI versus installed
wkhtmltopdf 0.12.6.1 (patched Qt) on Linux amd64, 13th Gen Intel Core
i7-13700HX. Full matrices live in
[`testdata/golden/benchmarks/README.md`](../testdata/golden/benchmarks/README.md).
The tables further down keep older snapshots so reviews stay comparable.

Related:

- Benchmark implementation: [`internal/convert/benchmarks_test.go`](../internal/convert/benchmarks_test.go)
- Public library benchmark: [`document_bench_test.go`](../document_bench_test.go)
- CLI comparison: [`internal/convert/wk_compare_test.go`](../internal/convert/wk_compare_test.go)
- Recorded results and templates: [`testdata/golden/benchmarks/README.md`](../testdata/golden/benchmarks/README.md)
- Raw Go rows: [`testdata/golden/benchmarks/benchmark-results.txt`](../testdata/golden/benchmarks/benchmark-results.txt)
- Process comparison CSV: [`testdata/golden/benchmarks/cli-compare-results.csv`](../testdata/golden/benchmarks/cli-compare-results.csv)
- Phase 9.3 gate: `TestTenPageTableReportPerformance` in `internal/convert/perf_test.go`

---

## How to read these numbers

| Kind | What it measures | Where |
|------|------------------|-------|
| Direct CLI `/usr/bin/time` | Process elapsed time and peak RSS vs wkhtmltopdf | **Current 2026-08-14 table below** |
| Internal engine `go test -bench` | Direct `internal/convert` wall time, `B/op`, `allocs/op` | **Current 2026-08-14 generic matrix below** |
| Public library `go test -bench` | `Document.WritePDF` / `ImageDocument.WriteImage` wall time, `B/op`, `allocs/op` | `make bench-lib` |
| Phase 9.3 gate | Two full-pipeline runs of a 10-section invoice fixture; CI budget only | Historical timings below; CI still asserts **< 5 s** per run |

Page islands (`convert.NewBenchmarkPDFRequest`) are an **internal benchmark
opt-in**. They are not a user-facing CLI or library mode. Comparing an
island-era CLI number to today’s generic CLI is not a like-for-like fidelity
or RSS guarantee.

---

## Current CLI comparison vs wkhtmltopdf (2026-08-14)

Generic `gowkhtmltopdf` 0.2.1 versus `wkhtmltopdf 0.12.6.1 (with patched qt)`.
Same report fixture (20 invoice rows per requested page). Both binaries used
`--quiet --allow-local-files -o OUTPUT INPUT`. Each cell is the median of three timed
process runs after one warmup.

```sh
make bench-cli-compare

# Optional WeasyPrint + Puppeteer process comparisons
make bench
```

| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS | wkhtmltopdf RSS |
|------:|----------:|-----------------:|--------:|---------:|----------------:|
| 2 | 16 ms | 254 ms | 15.95x | 24,192 KiB | 44,852 KiB |
| 5 | 22 ms | 265 ms | 12.23x | 24,768 KiB | 45,396 KiB |
| 10 | 30 ms | 278 ms | 9.41x | 26,496 KiB | 46,200 KiB |
| 20 | 44 ms | 304 ms | 6.84x | 29,760 KiB | 47,156 KiB |
| 50 | 88 ms | 387 ms | 4.40x | 41,472 KiB | 51,824 KiB |
| 100 | 184 ms | 530 ms | 2.89x | 58,752 KiB | 58,976 KiB |
| 200 | 353 ms | 812 ms | 2.30x | 90,048 KiB | 74,336 KiB |
| 250 | 433 ms | 942 ms | 2.18x | 112,704 KiB | 81,636 KiB |
| 500 | **1.045 s** | **1.641 s** | **1.57x** | **199,872 KiB** | **123,264 KiB** |

gowkhtmltopdf was **faster at every tested size**. Short documents show the
largest gap (about **16x** at 2 pages) because wkhtmltopdf pays a ~250 ms
WebKit/process start. At 500 pages it is still about **1.6x** faster.

Peak RSS is **lower through 100 pages** and **higher from 200 pages** on
this generic path. PDF output is smaller from 50 pages onward. The
2026-08-09 island-era table later on this page is **not** a current
memory claim.

---

## Current internal engine matrix (2026-08-14, generic)

One iteration (`-benchtime=1x -count=1`). `B/op` is cumulative allocation
traffic, not process RSS.

```sh
make bench-engine
```

| Workload | 2 | 10 | 100 | 500 |
|----------|--:|---:|----:|----:|
| PDF pages | 3.66ms | 15.0ms | 149ms | **966ms / 224.3MB / 1.15M allocs** |
| Template + PDF pages | 3.63ms | 15.4ms | 165ms | **948ms / 228.9MB / 1.20M allocs** |
| Web-fetch image tiles | 10.7ms | 13.8ms | 42.6ms | 166ms |
| Inline image tiles | 14.6ms | 10.1ms | 38.6ms | 178ms |

Full size matrix and certified-islands rows: Snapshot F in
[`benchmark-results.txt`](../testdata/golden/benchmarks/benchmark-results.txt).

## Public library matrix

The public library benchmark calls `Document.WritePDF` and
`ImageDocument.WriteImage` directly. It constructs the public documents from
in-memory HTML before timing; public validation, mapping, and the full
renderer remain inside the timed calls. It does not start a CLI or read an
HTML file from disk.

```sh
make bench-lib
```

---

## Phase 9.3 gate (historical timings)

A 10-page invoice table report (10 sections × 40 line-item rows, repeated
`<thead>`, `page-break-before` sections) through the full pipeline (load →
parse → style → layout → paginate → paint → assemble → write).

| Measurement | Value |
|-------------|-------|
| Cold run (first of two) | **~140 ms** (120–149 ms across runs) |
| Warm run (second) | **~156 ms** (96–203 ms across runs) |
| Output size | **96,341 bytes** (10 pages) |

- **Command:** `go test ./internal/convert -run TestTenPageTableReportPerformance -v`
- **Machine:** go1.26.4 linux/amd64, Linux x86_64, 13th Gen Intel Core i7-13700HX (24 threads), **2026-08-03**
- **Budget asserted in CI:** < 5 s per run (generous — catches
  order-of-magnitude regressions only)
- Skipped under `go test -short`

These cold/warm samples are **historical**. The test still enforces the
page-count and 5 s budget; it does not require the PDF stream to match a
fixed byte length across runs.

---

## Benchmark matrix (historical in-process, 2026-08-09)

The reproducible Go benchmark matrix covers 2, 5, 10, 20, 50, 100, 200, 250,
and 500 pages for PDF and template rendering. Web-fetch and inline-image
benchmarks use the same sizes as **image tiles**, because image mode renders
one raster canvas rather than paginated PDF pages.

The Phase 9.3 gate above is a separate 10-section × 40-row invoice fixture.
The historical in-process snapshot below uses the checked-in benchmark
templates (20 realistic rows per page), so those timings are not directly
comparable with the Phase 9.3 gate or with the CLI tables further down.

| Workload | 2 | 5 | 10 | 20 | 50 | 100 | 200 | 250 | 500 |
|----------|--:|--:|---:|---:|---:|----:|----:|----:|----:|
| PDF pages | 5.2ms | 7.1ms | 16.3ms | 32.0ms | 72.2ms | 168ms | 316ms | 430ms | 0.87s |
| Template + PDF pages | 3.9ms | 6.1ms | 12.5ms | 23.6ms | 73.2ms | 165ms | 346ms | 422ms | 0.94s |
| Web-fetch image tiles | 18.53ms | 22.89ms | 22.95ms | 22.50ms | 32.36ms | 51.08ms | 89.32ms | 110.31ms | 200.89ms |
| Inline image tiles | 16.97ms | 20.29ms | 21.04ms | 24.24ms | 32.20ms | 49.67ms | 83.27ms | 101.90ms | 192.13ms |

PDF / Template: historical in-process perf-review snapshot (**2026-08-09**).

Against the pre-wave snapshot taken earlier the same day, the same
one-iteration 500-page measurements changed as follows:

| Metric | Pre-wave | Perf-wave | Change |
|--------|---------:|----------:|-------:|
| PDF time | 1.013s | 0.873s | **−13.8%** |
| PDF B/op | 392.2MB | 335.8MB | **−14.4%** |
| PDF allocs/op | 535,064 | 517,875 | **−3.2%** |
| Template + PDF time | 1.047s | 0.942s | **−10.1%** |
| Template + PDF B/op | 397.6MB | 340.2MB | **−14.4%** |
| Template + PDF allocs/op | 586,355 | 569,123 | **−2.9%** |

`B/op` and allocs are deterministic for identical code paths, so the −14.4%
`B/op` cut is a real wave win (wall time on one-shot laptop runs is noisier).
Main drivers recorded for that wave: single cmap lookup per rune
(`GlyphAdvancePoints`), no duplicate min-content re-measure, in-place
inline-item compaction, ASCII fast paths in `TextShow`/HTML scanning,
rune-union dedup in font subsetting, pointer-compare op sorts, zero-crossing
split fast path.

---

## Direct CLI comparison vs wkhtmltopdf (historical, island-era)

The **2026-08-09** table below is **historical pre-CR-02 / island-era CLI**.
Ordinary CLI documents no longer take the page-island path.

**Current documented process snapshot:**
[`testdata/golden/benchmarks/README.md`](../testdata/golden/benchmarks/README.md)
— Snapshot D generic 500-page process RSS **54,632 KiB / 960 ms** on that
host. Treat later snapshots in that file as newer than this section.

Historical 2026-08-09 process-level measurements used identical report
fixtures, three runs per size, and `/usr/bin/time` for wall time and peak
RSS. wkhtmltopdf was 0.12.6.1. All output files passed the expected
page-count check.

| Pages | Gowk time | wkhtmltopdf time | Gowk RSS | wkhtmltopdf RSS |
|------:|----------:|-----------------:|---------:|----------------:|
| 2 | 10 ms | 220 ms | 18,432 KiB | 35,268 KiB |
| 5 | 10 ms | 230 ms | 17,208 KiB | 35,528 KiB |
| 10 | 20 ms | 250 ms | 17,832 KiB | 36,408 KiB |
| 20 | 30 ms | 270 ms | 18,052 KiB | 38,080 KiB |
| 50 | 70 ms | 360 ms | 20,644 KiB | 43,052 KiB |
| 100 | 140 ms | 500 ms | 23,340 KiB | 51,092 KiB |
| 200 | 300 ms | 800 ms | 30,256 KiB | 67,576 KiB |
| 250 | 380 ms | 940 ms | 34,200 KiB | 75,948 KiB |
| 500 | 890 ms | 1,720 ms | 50,888 KiB | 116,512 KiB |

On **that** island-era snapshot, gowkhtmltopdf was faster and used less RSS
at every tested size. At 500 pages it was approximately 1.9× faster and used
56.3% less RSS. That observation does **not** license a claim that the
current generic CLI is always faster than wkhtmltopdf. Use wkhtmltopdf when
legacy wkhtmltopdf print compatibility is the primary requirement; use
gowkhtmltopdf when you want an in-process HTML template engine and the documented
CSS subset.

The full matrix, including PDF bytes, commands, and measurement caveats, is
in the [benchmark documentation](../testdata/golden/benchmarks/README.md).

---

## Page islands (benchmark-only)

Certified page islands exist so large, regular report templates can be timed
without pretending every HTML document is island-shaped.

- **User-facing path:** `NewPDFRequest` / the CLI / the public library API —
  **generic** layout. No island opt-in.
- **Benchmark path:** `convert.NewBenchmarkPDFRequest` sets
  `benchmarkPageIslands`. Used by `BenchmarkPDFPages/certified-islands` and
  related tests only.

Do not compare a certified-islands `B/op` or RSS row to a generic HTML
fidelity review without saying so.

---

## How to measure

Current snapshot commands:

```sh
make bench-cli-compare
make bench
make bench-engine
make bench-lib
```

Internal engine matrix:

```sh
go test ./internal/convert -run '^$' \
  -bench 'Benchmark(PDFPages|TemplatePages|WebFetchImage|ImageAssets)$' \
  -benchmem -benchtime=1x -count=1
```

Phase 9.3 gate:

```sh
go test ./internal/convert -run TestTenPageTableReportPerformance -v
```

CPU profile (pprof):

```sh
go test ./internal/convert -run TestTenPageTableReportPerformance \
  -cpuprofile /tmp/cpu.pprof
go tool pprof -top /tmp/cpu.pprof
```

Live movie/TV listing benchmark (opt-in, real TVmaze API data and poster
CDN — not CI):

```sh
GOWKHTMLTOPDF_LIVE_BENCHMARK=1 \
  go test ./internal/convert -run '^$' \
  -bench '^BenchmarkLiveMovieListing/(2Images|5Images|10Images)$' \
  -benchmem -benchtime=1x -count=1
```

Instructions: [live movie listing benchmark](../testdata/golden/benchmarks/README.md#live-movie-listing-benchmark).
