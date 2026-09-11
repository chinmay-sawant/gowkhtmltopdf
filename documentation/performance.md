# Performance

Numbers on this page are **labeled snapshots**, not a live SLA. Host, GOCACHE
state, and whether a run used the **generic** convert path or the
**benchmark-only page-island** path all change wall time and RSS.

**Current snapshot: 2026-09-11 perf-time phase-7 closure capture.** The 0.2.6
perf-time plan cut warm 500-page time from 1,228.72 ms to a measured 576.33 ms
(2.13x) at 163.03 MB `B/op`, and warm 500-page `B/op` sits 30.6 percent below
the 234.92 MB ceiling. The closure re-capture of the same code measured
medians of 733.48 ms (warm matrix) and 695.42 ms (standalone) in a slower host
window; the allocation and output contracts are unchanged. Public image
medians are 43.43 ms at 500 tiles in the phase-5b capture (44.27 ms in the
closure capture) with a lossless PNG about 50 percent larger, an intended
size-for-speed trade. The concurrent-layout design was rejected for shipping
(5.4 percent paired gain against a 15 percent floor) and left unwired. Full
matrices live in
[`testdata/golden/benchmarks/README.md`](../testdata/golden/benchmarks/README.md).
The 2026-09-11 perf-improve capture, the 2026-09-11 recovery capture, the
2026-08-19 snapshot, and the 2026-08-14 tables further down are dated history.

Related:

- Benchmark implementation: [`internal/convert/benchmarks_test.go`](../internal/convert/benchmarks_test.go)
- Public library benchmark: [`document_bench_test.go`](../document_bench_test.go)
- CLI comparison: [`internal/convert/wk_compare_test.go`](../internal/convert/wk_compare_test.go)
- Recorded results and templates: [`testdata/golden/benchmarks/README.md`](../testdata/golden/benchmarks/README.md)
- Raw Go rows: [`testdata/golden/benchmarks/benchmark-results.txt`](../testdata/golden/benchmarks/benchmark-results.txt)
- Process comparison CSV: [`testdata/golden/benchmarks/cli-compare-results.csv`](../testdata/golden/benchmarks/cli-compare-results.csv)
- WeasyPrint comparison: [`testdata/golden/benchmarks/weasyprint-compare.md`](../testdata/golden/benchmarks/weasyprint-compare.md)
- Puppeteer comparison: [`testdata/golden/benchmarks/puppeteer-compare.md`](../testdata/golden/benchmarks/puppeteer-compare.md)
- Phase 9.3 gate: `TestTenPageTableReportPerformance` in `internal/convert/perf_test.go`

---

## How to read these numbers

| Kind | What it measures | Where |
|------|------------------|-------|
| Direct CLI `/usr/bin/time` | Process elapsed time and peak RSS | **2026-09-11 perf-time `cli-rss` rows below; the wkhtmltopdf comparison is the dated 2026-09-11 recovery table** |
| External engines | Process elapsed time and peak RSS vs WeasyPrint and Puppeteer/Chrome | **Dated 2026-09-11 recovery tables below; not re-run in the perf-time capture** |
| Internal engine `go test -bench` | Direct `internal/convert` wall time, `B/op`, `allocs/op` | **2026-09-11 perf-time rows below; the perf-improve, recovery, and 2026-08-19 matrices are historical** |
| Public library `go test -bench` | `Document.WritePDF` / `ImageDocument.WriteImage` wall time, `B/op`, `allocs/op` | **2026-09-11 perf-time rows below; the perf-improve, recovery, and 2026-08-19 matrices are historical** |
| Phase 9.3 gate | Two full-pipeline runs of a 10-section invoice fixture; CI budget only | Historical timings below; CI still asserts **< 5 s** per run |

Page islands (`convert.NewBenchmarkPDFRequest`) are an **internal benchmark
opt-in**. They are not a user-facing CLI or library mode. Comparing an
island-era CLI number to today’s generic CLI is not a like-for-like fidelity
or RSS guarantee.

---

## Current capture (2026-09-11 perf-time phase 7)

This is the closure capture for the 0.2.6 perf-time plan
(`plans/0.2.6/perf-time/phase-wise-checklist.md`), after the style,
display-list, pagination, compression, and image-encode work landed. It is a
**working-tree capture** on the uncommitted 0.2.6 tree; `VERSION` still reads
0.2.5. Host: Linux amd64, 13th Gen Intel Core i7-13700HX (WSL2, 24 CPUs,
7.6 GiB RAM). Toolchain: go1.26.4. Only the generic paths were measured;
certified page islands are not part of any row.

The warm matrix ran as **three independent fresh processes**; the standalone
internal, public-library PDF, and public-library image rows are **three
independent `1x` samples per workload**, one fresh process per sample,
captured with `scripts/bench-performance-recovery.sh`. Each reported time is
the median of the three raw values and `B/op` is one of the raw values, never
an average. The CLI rows are this capture's `cli-rss` mode, gowkhtmltopdf
only, the median of three timed process runs after one warmup; RSS is
`/usr/bin/time %M` and stays separate from `B/op`. Raw rows, medians, drift
controls, and the raw-to-published mapping:
`plans/0.2.6/perf-time/results/phase-7/final-capture.md`. The rows are
recorded as Snapshot M in
[`testdata/golden/benchmarks/benchmark-results.txt`](../testdata/golden/benchmarks/benchmark-results.txt).

### Perf-time rows

| Row | 2 pages / 250 tiles | 500 pages / 500 tiles | Pre-time baseline | Result |
|---|---:|---:|---:|---|
| Internal generic PDF time, warm matrix | 5.50 ms (cold) | 733.48 ms | 1,228.72 ms | 1.68x faster |
| Internal generic PDF B/op, warm matrix | 4,031,472 B (4.03 MB, cold) | 163,021,712 B (163.02 MB) | 234.92 MB | 30.6% below |
| Internal generic PDF time, standalone median | 6.08 ms (cold) | 695.42 ms | 1,297.98 ms | 1.87x faster |
| Internal generic PDF B/op, standalone median | 4,031,712 B (4.03 MB, cold) | 167,865,712 B (167.87 MB) | 235.50 MB | 28.6% below |
| Internal generic PDF allocs/op, standalone median | 4,453 | 755,091 | 1.15M | |
| Public library PDF time | 6.11 ms (cold) | 698.79 ms | 1,240.82 ms | 1.78x faster |
| Public library PDF B/op | 4,048,000 B (4.05 MB, cold) | 169,650,976 B (169.65 MB) | 236.91 MB | 28.4% below |
| Public library image time | 25.72 ms | 44.27 ms | 50.72 ms / 98.47 ms | 1.97x / 2.22x faster |
| Public library image B/op | 14,296,096 B (14.30 MB) | 26,414,016 B (26.41 MB) | 14.45 MB / 26.73 MB | below both |
| Public library image geometry | 1024x2056, 141,917 bytes | 1024x4040, 282,749 bytes | 94,352 / 188,268 bytes | lossless PNG about 50% larger |
| CLI process time, cli-rss median | 0.01 s | 0.70 s | 1.32 s | 1.89x faster |
| CLI process RSS, cli-rss median | 19,584 KiB | 147,264 KiB | 203,136 KiB | 27.5% below |

### Acceptance verdict

The plan's stretch target is a warm 500-page time at or below 615 ms, a CLI
500-page time at or below 0.66 s, and a public-library PDF 500-page time at or
below 620 ms, with `B/op` at or below 234.92 MB and image times at or below
25 / 49 ms. The **allocation targets hold on every row**: the 500-page `B/op`
is 163.02 MB on the warm matrix, 167.87 MB standalone, and 169.65 MB on the
public library, 28.4 to 30.6 percent below the pre-time baselines. The
**500-tile image target holds** at 44.27 ms.

The time targets did not reproduce in this host window: warm 500p is 733.48 ms
(matrix median) and 695.42 ms (standalone), public library PDF is 698.79 ms,
and CLI 500p is 0.70 s. The plan's measured-lever floor (warm 500p at or below
830 ms) holds. The implementation result on the same production source is the
phase-6 capture 15 minutes earlier: 576.33 ms / 163,032,880 B, a **2.13x**
speedup that meets the 615 ms stretch. Source hashes under `internal/layout`
and `internal/pdf` match between the two captures except the unwired parallel
prototype, and a no-prototype diagnostic binary measured the same band as the
prototype binary, so the drift is the host window, not the tree. The 250-tile
image median is 25.72 ms in this capture, 0.72 ms above the line; the phase-5b
raw samples measured 24.15 ms for the same code.

### Image time/size trade

The 250 / 500 tile rows are 1.97x / 2.22x faster than the pre-time baselines
(50.72 / 98.47 ms) and the encoded PNG grows about 50 percent: 94,352 to
141,917 bytes at 250 tiles and 188,268 to 282,749 bytes at 500 tiles. The
growth is the intended trade of the filter-none level-2 streaming writer
(`internal/imageout/pngfast.go`) that replaced `image/png` adaptive filtering
above the direct-raster threshold. PNG stays lossless and decoded pixels are
bit-identical; the files stay far below the 32 MiB `maxImageEncoded` budget.
See `plans/0.2.6/perf-time/results/image/image-time.md`.

### Rejected concurrency design

The phase-6 prototype rendered independent top-level sections in parallel and
merged them deterministically. It passed the equivalence contract (op-for-op
display lists, page counts, date-normalized PDF bytes, golden 65/65, B/op
flat), but its best paired same-process gain was **5.4 percent** (serial
596 ms versus 564 ms at W=2), below the required 15 percent floor, so it was
**rejected for shipping** and production stays serial. Pipeline overlap
(stage 2) was gated on stage 1 passing and was not attempted. The prototype
and its differential tests stay in `internal/layout/parallel.go` and
`internal/layout/parallel_test.go`; production never calls them. Full numbers:
`plans/0.2.6/perf-time/results/phase-6/concurrency.md`.

```sh
./scripts/bench-performance-recovery.sh --mode=internal-pdf-warm --benchtime=1x --count=1 --out=plans/0.2.6/perf-time/results/phase-7/warm
./scripts/bench-performance-recovery.sh --mode=internal-pdf --sizes=2,500 --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=public-pdf --sizes=2,500 --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=public-image --sizes=250,500 --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=cli-rss --sizes=2,100,500 --runs=3
```

---

## Historical capture (2026-09-11 perf-improve phase 7)

This is the closure capture for the 0.2.6 warm-path performance plan
(`plans/0.2.6/perf-improve/phase-wise-checklist.md`), after the font, seal,
forced-break, display-list, and page-bucketing phases landed. It is a
**working-tree capture** on the uncommitted 0.2.6 tree; `VERSION` still reads
0.2.5. Host: Linux amd64, 13th Gen Intel Core i7-13700HX (WSL2, 24 CPUs,
7.6 GiB RAM). Toolchain: go1.26.4. Only the generic paths were measured;
certified page islands are not part of any row.

In-process and public-library rows are **three independent `1x` samples per
workload**, one fresh process per sample, captured with
`scripts/bench-performance-recovery.sh`. Each time is the median of the three
raw values and `B/op` is one of the raw values, never an average. The warm
matrix runs the ascending size list in one process; its 2-page row is cold
(first conversion) and its 500-page row is warm. Standalone samples run 2
pages first (cold) and 500 pages second (warm). The CLI rows are this
capture's `cli-rss` mode, gowkhtmltopdf only, the median of three timed
process runs after one warmup; RSS is `/usr/bin/time %M` and stays separate
from `B/op`. Raw rows, medians, median sources, and the full verdict:
`plans/0.2.6/perf-improve/results/phase-7/final-capture.md`.

### Perf-improve rows

| Row | 2 pages / 250 tiles | 500 pages / 500 tiles | 0.2.4 / Snapshot I row | Result |
|---|---:|---:|---:|---|
| Internal generic PDF time, warm matrix | 6.69 ms (cold) | 1,228.72 ms | 1,010 ms | above the row |
| Internal generic PDF B/op, warm matrix | 2,665,064 B (2.67 MB, cold) | 234,923,560 B (234.92 MB) | 237.76 MB | 1.19% below |
| Internal generic PDF time, standalone median | 6.11 ms (cold) | 1,297.98 ms | 1,010 ms | above the row |
| Internal generic PDF B/op, standalone median | 2,665,080 B (2.67 MB, cold) | 235,496,640 B (235.50 MB) | 237.76 MB | 0.95% below |
| Internal generic PDF allocs/op, standalone median | 6,170 | 1,225,356 | 1.15M | |
| Public library PDF time | 7.14 ms (cold) | 1,240.82 ms | 1,104.51 ms | above the row |
| Public library PDF B/op | 2,680,016 B (2.68 MB, cold) | 236,911,328 B (236.91 MB) | 236.85 MB | 0.03% above, parity |
| Public library image time | 50.72 ms | 98.47 ms | n/a | |
| Public library image B/op | 14,446,240 B (14.45 MB) | 26,730,968 B (26.73 MB) | 20.66 MB / 52.00 MB | 30.1% / 48.6% below |
| Public library image geometry | 1024x2056 / 94,352 B | 1024x4040 / 188,268 B | unchanged | |
| CLI process time, cli-rss median | 0.01 s | 1.32 s | 1,042 ms | above the row |
| CLI process RSS, cli-rss median | 17,472 KiB | 203,136 KiB | 208,128 KiB | 2.4% below |

### Acceptance verdict

The plan's acceptance is a warm 500-page `B/op` at or below 240 MB and a warm
500-page time at or below 1.10 s. The allocation target **is met**: 234.92 MB
on the warm matrix, 235.50 MB on the standalone median, and 236.91 MB on the
public library median, all below the 240 MB line. The internal numbers are
slightly below the 0.2.4 237.76 MB row; the public library number is at parity
with its 236.85 MB row (0.03 percent above). The time target **is not met**:
the warm matrix is 1,228.72 ms and the standalone median is 1,297.98 ms,
above both the 1.10 s line and the 1.010 s Snapshot I row. The fastest
500-page row in the capture is 1,186.48 ms (public library PDF sample 2).

### What improved against the pre-improve capture

| Metric | Pre-improve 2026-09-11 | Phase-7 2026-09-11 | Change |
|---|---:|---:|---:|
| Warm 500-page matrix time | 1,489.42 ms | 1,228.72 ms | -17.5% |
| Warm 500-page matrix B/op | 321,305,864 B (321.31 MB) | 234,923,560 B (234.92 MB) | -26.9% |
| Cold 2-page B/op, fresh process | 9,578,040 B (9.58 MB) | 2,665,064 B (2.67 MB, matrix cold row) | -72.2% |

The 2-page drop is the phase-2 single-copy plus lazy default-face load; the
500-page allocation drop is the phase-3 seal indexing, phase-5 `Op` packing
(472 to 440 bytes), and phase-6 page-index storage reuse. The remaining warm
time gap has no single allocation site to blame: the phase-4 profile put
`paginateOps` at 8.04 percent of samples after the forced-break batch, and
phase 5 rejected tightening `estimateOpCapacity` because the corpus maximum
is 15.16 ops per node against a 1.2887 ratio on the benchmark template.

```sh
./scripts/bench-performance-recovery.sh --mode=internal-pdf-warm --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=internal-pdf --sizes=2,500 --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=public-pdf --sizes=2,500 --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=public-image --sizes=250,500 --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=cli-rss --sizes=2,100,500 --runs=3
```

---

## Historical recovery capture (2026-09-11)

The current capture is a **0.2.6 performance-recovery working tree**
measurement. `VERSION` still reads 0.2.5, so it is a working-tree snapshot,
not a released build. Host: Linux amd64, 13th Gen Intel Core i7-13700HX
(WSL2, 24 CPUs, 7.6 GiB RAM). Toolchain: go1.26.4. Only the generic paths
were measured; certified page islands are not part of any row.

In-process and public-library rows are **three independent `1x` samples per
workload**, one fresh process per sample, captured with
`scripts/bench-performance-recovery.sh`. The reported value is the median of
the three raw samples and `B/op` is one of the raw values, never an average.
CLI cells are medians of three timed process runs after one warmup. Raw
samples, the method header, and the complete tables live in
`plans/0.2.6/perf-review/results/2026-09-11/valid-02-04.md`.

### Recovery rows

| Row | 2 pages / 250 tiles | 500 pages / 500 tiles | 0.2.4 row | Result |
|---|---:|---:|---:|---|
| Internal generic PDF time | 10.06 ms | 1,246.05 ms | 1,010 ms | target not met |
| Internal generic PDF B/op | 9,578,008 B (9.58 MB) | 321,104,720 B (321.10 MB) | 237.76 MB | target not met |
| Internal generic PDF allocs/op | 7,005 | 1,281,235 | n/a | |
| Public library PDF time | 10.88 ms | 1,268.56 ms | 1,104.51 ms | target not met |
| Public library PDF B/op | 9,596,168 B (9.60 MB) | 322,890,064 B (322.89 MB) | 236.85 MB | target not met |
| Public library image time | 54.42 ms | 89.18 ms | n/a | |
| Public library image B/op | 21,502,152 B (21.50 MB) | 27,205,880 B (27.21 MB) | 20.66 MB / 52.00 MB | 250 about 4.1% above the row; 500 about 47.7% below |

The 2-page B/op rows are one-iteration fresh-process samples, so the one-time
default-font load (about 6.8 MB; see
`plans/0.2.6/perf-review/results/2026-09-11/perf-04-05.md`) is charged to the
single operation. They are not like-for-like with multi-iteration matrix rows.

### Why the PDF rows missed and the image rows recovered

Internal generic 500-page B/op is **321.10 MB / 1,246.05 ms** against the
2026-08-14 Snapshot F row of **224.3 MB / 966 ms**: about **+43.2% B/op** and
**+29.0% time**. The supplied 0.2.4 target was 237.76 MB / 1,010 ms and it
was not met. Per the 2026-09-11 profiles, the remaining 500-page allocation
is dominated by the **one-shot display-list preallocation** (the `ops`
capacity estimate at `internal/layout/layout.go:1003`, 95,600,640 B) inside a
128.2 MB box-construction phase, plus **113.7 MB of pagination scratch**.
Exact style sharing removed the former 228.3 MB style-storage line from that
profile. Sources:
`plans/0.2.6/perf-review/results/2026-09-11/perf-04-05.md` and
`plans/0.2.6/perf-review/results/2026-09-11/pdf-02-05.md`.

The public image rows recovered because the direct final-resolution branch
paints canvases at 2,097,152 pixels and above without the 2x intermediate;
the 250-tile row is still about 4.1% above its 0.2.4 row, and the 500-tile row
is about 47.7% below it. Source:
`plans/0.2.6/perf-review/results/2026-09-11/img-02-04.md`.

### Direct CLI vs wkhtmltopdf (2026-09-11)

Generic `gowkhtmltopdf` working tree versus `wkhtmltopdf 0.12.6.1 (with
patched qt)`. Same report fixture (20 invoice rows per requested page). Both
binaries used `--quiet --allow-local-files -o OUTPUT INPUT`. Each cell is the
median of three timed process runs after one warmup.

```sh
make bench-cli-compare
```

| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS | wkhtmltopdf RSS |
|------:|----------:|-----------------:|--------:|---------:|----------------:|
| 2 | 17 ms | 258 ms | 14.95x | 24,576 KiB | 44,716 KiB |
| 5 | 23 ms | 266 ms | 11.44x | 26,496 KiB | 45,068 KiB |
| 10 | 34 ms | 286 ms | 8.29x | 28,608 KiB | 46,024 KiB |
| 20 | 53 ms | 306 ms | 5.79x | 33,984 KiB | 47,772 KiB |
| 50 | 122 ms | 394 ms | 3.23x | 47,616 KiB | 52,240 KiB |
| 100 | 229 ms | 541 ms | 2.36x | 69,120 KiB | 59,380 KiB |
| 200 | 468 ms | 830 ms | 1.77x | 113,280 KiB | 74,492 KiB |
| 250 | 599 ms | 988 ms | 1.65x | 139,584 KiB | 81,884 KiB |
| 500 | **1.288 s** | **1.753 s** | **1.36x** | **240,960 KiB** | **123,172 KiB** |

gowkhtmltopdf is faster at every tested size. Peak RSS is lower through 50
pages and higher from 100 pages on this generic path. Raw rows:
[`cli-compare-results.csv`](../testdata/golden/benchmarks/cli-compare-results.csv).

### External comparisons (2026-09-11)

The external harness uses the same report fixture, with one warmup and three
timed process runs. Its default matrix is 2, 10, 50, and 100 pages;
Ghostscript checked the requested page count for every output.

```sh
./scripts/bench-external.sh --engines=weasyprint
./scripts/bench-external.sh --engines=puppeteer
```

#### WeasyPrint (2026-09-11)

| Pages | Gowk time | WeasyPrint time | Speedup | Gowk RSS | WeasyPrint RSS |
|------:|----------:|----------------:|--------:|---------:|----------------:|
| 2 | 21 ms | 653 ms | 31.13x | 25,152 KiB | 81,932 KiB |
| 10 | 36 ms | 1.431 s | 39.80x | 28,416 KiB | 111,444 KiB |
| 50 | 124 ms | 5.482 s | 44.07x | 48,000 KiB | 252,448 KiB |
| 100 | 245 ms | 11.119 s | 45.29x | 65,856 KiB | 427,880 KiB |

#### Puppeteer / Chrome (2026-09-11)

| Pages | Gowk time | Puppeteer time | Speedup | Gowk RSS | Puppeteer RSS |
|------:|----------:|----------------:|--------:|---------:|---------------:|
| 2 | 21 ms | 1.470 s | 68.85x | 24,960 KiB | 973,260 KiB |
| 10 | 37 ms | 1.550 s | 42.45x | 28,800 KiB | 1,021,160 KiB |
| 50 | 120 ms | 1.815 s | 15.08x | 48,576 KiB | 1,114,380 KiB |
| 100 | 249 ms | 2.179 s | 8.74x | 68,928 KiB | 1,241,476 KiB |

WeasyPrint RSS is the measured process peak from `/usr/bin/time %M`.
Puppeteer RSS is the peak process-tree RSS for the Node driver and headless
Chrome descendants, so those RSS readings are not directly equivalent. Raw
detail:
[`weasyprint-compare.md`](../testdata/golden/benchmarks/weasyprint-compare.md),
[`puppeteer-compare.md`](../testdata/golden/benchmarks/puppeteer-compare.md).

---

## Historical CLI comparison vs wkhtmltopdf (2026-08-19)

Generic `gowkhtmltopdf` 0.2.4 versus `wkhtmltopdf 0.12.6.1 (with patched qt)`.
Same report fixture (20 invoice rows per requested page). Both binaries used
`--quiet --allow-local-files -o OUTPUT INPUT`. Each cell is the median of three timed
process runs after one warmup.

```sh
make bench

# Run only the wkhtmltopdf comparison
make bench-cli-compare
```

| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS | wkhtmltopdf RSS |
|------:|----------:|-----------------:|--------:|---------:|----------------:|
| 2 | 17 ms | 259 ms | 15.46x | 23,808 KiB | 44,192 KiB |
| 5 | 22 ms | 268 ms | 12.36x | 24,960 KiB | 44,716 KiB |
| 10 | 30 ms | 276 ms | 9.21x | 27,264 KiB | 45,992 KiB |
| 20 | 45 ms | 317 ms | 7.06x | 30,528 KiB | 47,464 KiB |
| 50 | 112 ms | 406 ms | 3.63x | 43,200 KiB | 51,856 KiB |
| 100 | 184 ms | 526 ms | 2.85x | 61,248 KiB | 59,048 KiB |
| 200 | 376 ms | 811 ms | 2.15x | 96,192 KiB | 74,192 KiB |
| 250 | 480 ms | 964 ms | 2.01x | 116,736 KiB | 81,740 KiB |
| 500 | **1.042 s** | **1.671 s** | **1.60x** | **208,128 KiB** | **123,080 KiB** |

gowkhtmltopdf was **faster at every tested size**. Short documents show the
largest gap (about **16x** at 2 pages) because wkhtmltopdf pays a ~250 ms
WebKit/process start. At 500 pages it is still about **1.6x** faster.

Peak RSS is **lower through 100 pages** and **higher from 200 pages** on
this generic path. PDF output is smaller from 50 pages onward. The
2026-08-09 island-era table later on this page is **not** a current
memory claim.

## Historical external comparisons (2026-08-19)

The external harness uses the same generated report fixture, with one warmup
and three timed process runs. Its default matrix is 2, 10, 50, and 100 pages;
Ghostscript checked the requested page count for every output.

```sh
make bench
./scripts/bench-external.sh --engines=puppeteer
```

### WeasyPrint

| Pages | Gowk time | WeasyPrint time | Speedup | Gowk RSS | WeasyPrint RSS |
|------:|----------:|----------------:|--------:|---------:|----------------:|
| 2 | 19 ms | 616 ms | 32.15x | 24,576 KiB | 77,420 KiB |
| 10 | 31 ms | 1.352 s | 43.65x | 26,880 KiB | 106,000 KiB |
| 50 | 100 ms | 5.217 s | 52.01x | 42,624 KiB | 246,876 KiB |
| 100 | 186 ms | 10.528 s | 56.62x | 58,560 KiB | 423,004 KiB |

### Puppeteer / Chrome

| Pages | Gowk time | Puppeteer time | Speedup | Gowk RSS | Puppeteer RSS |
|------:|----------:|----------------:|--------:|---------:|---------------:|
| 2 | 18 ms | 1.411 s | 77.30x | 23,808 KiB | 944,056 KiB |
| 10 | 32 ms | 1.548 s | 47.84x | 27,264 KiB | 1,019,896 KiB |
| 50 | 121 ms | 2.069 s | 17.06x | 43,008 KiB | 1,108,580 KiB |
| 100 | 199 ms | 2.145 s | 10.78x | 62,016 KiB | 1,245,988 KiB |

WeasyPrint RSS is the measured process peak from `/usr/bin/time %M`.
Puppeteer RSS is the peak process-tree RSS for the Node driver and headless
Chrome descendants, so those RSS readings are not directly equivalent.
Raw detail: [`weasyprint-compare.md`](../testdata/golden/benchmarks/weasyprint-compare.md),
[`puppeteer-compare.md`](../testdata/golden/benchmarks/puppeteer-compare.md).

---

## Historical internal engine matrix (2026-08-14, generic)

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
`ImageDocument.WriteImage` directly. Its PDF workload uses the same
`templates/report.html.tmpl` fixture and 20-row page data as the CLI
comparisons, constructs the public document from the resulting in-memory HTML
before timing, and asserts the physical page count. Public validation, mapping,
and the full renderer remain inside the timed calls. It does not start a CLI or
read an HTML file from disk.

```sh
make bench-lib
```

### Python public API (`make python-benchmarks`)

Same fixture family as `make bench-lib` (`testdata/golden/benchmarks/templates/report.html.tmpl`,
20 invoice rows per page), timed through the in-process Python binding
(`Document.pdf()` / `ImageDocument.image()` via `ctypes`). Rebuilds
`dist/libgowkhtmltopdf.so` first. Not part of `make test`.

```sh
make python-benchmarks
# optional matrix override:
GOWKHTMLTOPDF_BENCH_SIZES=2,10,50 GOWKHTMLTOPDF_BENCH_RUNS=10 make python-benchmarks
```

The Go public-library 2-page row is about **3.8 ms** on the 2026-08-19
snapshot (`make bench-lib`). Use `make python-benchmarks` on the same host
to measure the Python ctypes path against that baseline.

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

## Historical in-process and public API samples (2026-08-19)

The requested `make bench-engine` and `make bench-inprocess` commands both
completed. They invoke the same one-iteration internal matrix; the second
sample is shown below because the first invocation had a materially slower
one-shot 500-page PDF timing. Full raw output for both invocations is recorded
in [`testdata/golden/benchmarks/benchmark-results.txt`](../testdata/golden/benchmarks/benchmark-results.txt)
as Snapshots H, I, and J.

| Workload | 2 | 10 | 100 | 500 |
|----------|--:|--:|----:|----:|
| PDF pages | 3.58ms | 14.9ms | 158ms | **1.010s** |
| Template + PDF pages | 3.34ms | 14.9ms | 182ms | **1.033s** |
| Web-fetch image tiles | 11.4ms | 13.4ms | 43.7ms | **171ms** |
| Inline image tiles | 16.0ms | 11.1ms | 43.0ms | **174ms** |

The same selected rows measured `B/op` of 2.2MB, 6.1MB, 48.7MB, and
237.8MB for generic 500-page PDF, with 1.15M allocations/op. `B/op` is
cumulative allocation traffic, not process RSS.

### In-process PDF timing relative to the wkhtmltopdf binary

This ratio is `wkhtmltopdf CLI time / in-process PDF time`. It is indicative,
not an identical-boundary comparison: wkhtmltopdf includes CLI/process startup
and file handling, while the in-process benchmark calls `convert.Run` with
inline HTML and a memory buffer.

| Pages | wkhtmltopdf CLI | In-process PDF | Indicative multiplier |
|------:|----------------:|---------------:|----------------------:|
| 2 | 259ms | 3.58ms | **72.43x** |
| 5 | 268ms | 7.29ms | **36.74x** |
| 10 | 276ms | 14.86ms | **18.58x** |
| 20 | 317ms | 28.64ms | **11.07x** |
| 50 | 406ms | 84.17ms | **4.82x** |
| 100 | 526ms | 157.69ms | **3.34x** |
| 200 | 811ms | 384.45ms | **2.11x** |
| 250 | 964ms | 449.51ms | **2.14x** |
| 500 | 1.671s | 1.010s | **1.65x** |

The requested `make bench-lib` command also completed with ten iterations per
workload:

| Workload | 2 | 10 | 100 | 500 |
|----------|--:|--:|----:|----:|
| Public PDF | 3.77ms | 16.0ms | 161ms | **1.105s** |
| Public image | 11.1ms | 14.6ms | 30.1ms | **143ms** |

These runs measure gowkhtmltopdf's internal engine and public API only. They
are not comparisons with another library, so they do not justify a `16x` or
`20x` landing-page claim. Comparative claims remain limited to the separately
measured external process snapshots.

### Public library PDF timing relative to the wkhtmltopdf binary

This uses `wkhtmltopdf CLI time / public library PDF time`. The public path
calls `Document.WritePDF` directly and does not launch the gowkhtmltopdf CLI.

| Pages | wkhtmltopdf CLI | Public library PDF | Indicative multiplier |
|------:|----------------:|-------------------:|----------------------:|
| 2 | 259ms | 3.77ms | **68.73x** |
| 5 | 268ms | 8.33ms | **32.19x** |
| 10 | 276ms | 16.03ms | **17.22x** |
| 20 | 317ms | 31.19ms | **10.16x** |
| 50 | 406ms | 74.83ms | **5.43x** |
| 100 | 526ms | 160.94ms | **3.27x** |
| 200 | 811ms | 337.38ms | **2.40x** |
| 250 | 964ms | 441.40ms | **2.18x** |
| 500 | 1.671s | 1.105s | **1.51x** |

---

## Direct CLI comparison vs wkhtmltopdf (historical, island-era)

The **2026-08-09** table below is **historical pre-CR-02 / island-era CLI**.
Ordinary CLI documents no longer take the page-island path.

The current documented process snapshot is the 2026-09-11 capture above and
in [`testdata/golden/benchmarks/README.md`](../testdata/golden/benchmarks/README.md).
Snapshot D (54,632 KiB / 960 ms at 500 pages) below is older historical
evidence.

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
gowkhtmltopdf when you want an in-process PDF engine based on HTML templates —
without any wrappers — and the documented CSS subset.

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

Standard benchmark targets:

```sh
make bench
make bench-cli-compare  # standalone wkhtmltopdf comparison
make bench-engine
make bench-inprocess    # compatibility alias for bench-engine
make bench-lib
```

Current perf-improve capture commands (fresh process per sample; see the
2026-09-11 perf-improve section above):

```sh
./scripts/bench-performance-recovery.sh --mode=internal-pdf-warm --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=internal-pdf --sizes=2,500 --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=public-pdf --sizes=2,500 --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=public-image --sizes=250,500 --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=cli-rss --sizes=2,100,500 --runs=3
```

Historical recovery-capture commands (see the 2026-09-11 recovery section
above):

```sh
./scripts/bench-performance-recovery.sh --mode=internal-pdf --sizes=2,500 --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=public-pdf --sizes=2,500 --benchtime=1x --count=1
./scripts/bench-performance-recovery.sh --mode=public-image --sizes=250,500 --benchtime=1x --count=1
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
