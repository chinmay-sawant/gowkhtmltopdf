# Benchmarks

Consolidated benchmark capture for gowkhtmltopdf. This page is the keeper for
the current numbers; the deep historical record stays in
[performance.md](performance.md) and
[testdata/golden/benchmarks/README.md](../testdata/golden/benchmarks/README.md).

## Current capture: 2026-09-13

- Engine: generic `bin/gowkhtmltopdf`, `VERSION` 0.2.6 (release tree
  `8aab63a`, `chore/review-026`), built with `make build`
- Host: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64, 13th Gen Intel Core
  i7-13700HX (24 CPUs), 7.6 GiB RAM
- Toolchain: `go version go1.26.4 linux/amd64`
- Fixture: `testdata/golden/benchmarks/templates/report.html.tmpl`
  (20 invoice rows per requested page), sha256 `e3b5387b...`
- CLI and external rows: cold full-process runs, median of 3 timed runs after
  one warmup; wall time via `/usr/bin/time`, peak RSS via `%M` (Puppeteer
  samples the process tree). The three engine tables share the gowk column
  from the same capture's `make bench-cli-compare` run
  (`--gowk-baseline=testdata/golden/benchmarks/cli-compare-results.csv`)
- In-process rows: `go test -benchmem -benchtime=1x -count=1`, one fresh
  process per round; the value is the median of three rounds and `B/op` and
  `allocs/op` are the median-time sample's raw values, never averages. Sizes
  run ascending in one process, so the 2-page row is cold and carries the
  one-time font load; every later row is warm
- Python rows: `make python-benchmarks`, 10 warm timed iterations after one
  warmup on the c-shared library, median shown
- Reproduce: `make build`, `make bench` (`make bench-cli-compare` plus
  `./scripts/bench-external.sh`), `make bench-engine`, `make bench-lib`,
  `make python-benchmarks`, and
  `./scripts/bench-performance-recovery.sh --mode=<mode>` (see the modes below)

Raw captures (gitignored, local): `plans/0.2.6/perf-review/results/2026-09-13/`.
Machine-generated committed artifacts: `testdata/golden/benchmarks/`
`cli-compare.md` / `weasyprint-compare.md` / `puppeteer-compare.md` plus their
`-results.csv` siblings.

## CLI vs wkhtmltopdf (cold full-process runs)

Source: `testdata/golden/benchmarks/cli-compare.md`. gowk runs
`--quiet --allow-local-files -o OUTPUT INPUT`; wkhtmltopdf 0.12.6.1 (patched
Qt) runs `--quiet --enable-local-file-access INPUT OUTPUT`.

| Pages | gowk time | wkhtmltopdf time | Speedup | gowk RSS | wkhtmltopdf RSS | gowk PDF | wkhtmltopdf PDF |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 13 ms | 258 ms | 19.68x | 19,584 KiB | 44,528 KiB | 34,210 B | 18,486 B |
| 5 | 18 ms | 269 ms | 14.68x | 22,080 KiB | 44,784 KiB | 42,795 B | 30,584 B |
| 10 | 24 ms | 279 ms | 11.65x | 24,384 KiB | 45,888 KiB | 57,239 B | 50,994 B |
| 20 | 35 ms | 310 ms | 8.75x | 26,304 KiB | 47,300 KiB | 84,680 B | 90,742 B |
| 50 | 67 ms | 393 ms | 5.84x | 29,376 KiB | 51,652 KiB | 167,525 B | 210,678 B |
| 100 | 124 ms | 532 ms | 4.30x | 35,520 KiB | 59,172 KiB | 306,321 B | 411,260 B |
| 200 | 240 ms | 814 ms | 3.39x | 45,888 KiB | 74,356 KiB | 583,670 B | 816,285 B |
| 250 | 279 ms | 973 ms | 3.49x | 52,032 KiB | 81,632 KiB | 722,322 B | 1,019,315 B |
| 500 | 573 ms | 1.718 s | 3.00x | 80,448 KiB | 123,068 KiB | 1,420,537 B | 2,036,776 B |

gowkhtmltopdf is faster and uses less peak RSS at every tested size, including
500 pages. Against the 2026-09-12 capture the 2-page speedup moved from 18.50x
to 19.68x and the 5-page from 14.37x to 14.68x; the larger sizes are flat
within host-window noise (4.30x at 100 pages, 3.00x at 500 pages). This
supersedes the older claim that gowk RSS was higher from 100 pages on.

## External engines (cold full-process runs, 2 / 10 / 50 / 100 pages)

### WeasyPrint 69.0

Source: `testdata/golden/benchmarks/weasyprint-compare.md`.

| Pages | gowk time | WeasyPrint time | Speedup | gowk RSS | WeasyPrint RSS | gowk PDF | WeasyPrint PDF |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 13 ms | 639 ms | 49.18x | 19,584 KiB | 81,744 KiB | 34,210 B | 15,584 B |
| 10 | 24 ms | 1.435 s | 59.78x | 24,384 KiB | 110,976 KiB | 57,239 B | 45,174 B |
| 50 | 67 ms | 5.496 s | 82.04x | 29,376 KiB | 251,804 KiB | 167,525 B | 190,544 B |
| 100 | 124 ms | 10.953 s | 88.33x | 35,520 KiB | 427,372 KiB | 306,321 B | 372,868 B |

### Puppeteer / Chrome (puppeteer-core 24.43.1 + Chrome 143)

Source: `testdata/golden/benchmarks/puppeteer-compare.md`.

| Pages | gowk time | Puppeteer time | Speedup | gowk RSS | Puppeteer RSS | gowk PDF | Puppeteer PDF |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 13 ms | 1.452 s | 111.73x | 19,584 KiB | 942,964 KiB | 34,210 B | 134,319 B |
| 10 | 24 ms | 1.479 s | 61.63x | 24,384 KiB | 1,022,844 KiB | 57,239 B | 450,799 B |
| 50 | 67 ms | 1.785 s | 26.65x | 29,376 KiB | 1,114,080 KiB | 167,525 B | 1,981,892 B |
| 100 | 124 ms | 2.158 s | 17.40x | 35,520 KiB | 1,241,028 KiB | 306,321 B | 3,936,067 B |

Puppeteer RSS is the peak process-tree reading for the Node driver plus
headless Chrome, not a single-process `%M` value.

## In-process engine (internal, generic path; 2 pages cold, later rows warm)

Full ascending warm matrix in one process per round; the 2-page row is the
first conversion in its process and carries the one-time font load. Median of
three rounds.

| Pages | Time | B/op | allocs/op |
|---:|---:|---:|---:|
| 2 | 5.20 ms | 4.10 MB | 4,139 |
| 5 | 7.89 ms | 6.84 MB | 8,163 |
| 10 | 11.75 ms | 4.63 MB | 14,876 |
| 20 | 21.44 ms | 5.20 MB | 28,374 |
| 50 | 54.76 ms | 11.56 MB | 69,429 |
| 100 | 103.51 ms | 23.19 MB | 137,943 |
| 200 | 207.91 ms | 44.73 MB | 275,212 |
| 250 | 262.21 ms | 56.49 MB | 344,008 |
| 500 | 539.33 ms | 111.43 MB | 687,406 |

### Full workload matrix (`make bench-engine`)

The compact 2 / 10 / 100 / 500 view of the four internal workloads, median of
three fresh processes, `-benchtime=1x -count=1`. The PDF row runs after the
image workloads in the same process, so it is warm and does not match the
standalone recovery matrix above. Cells are time / B/op / allocs.

| Workload | 2 | 10 | 100 | 500 |
|---|---:|---:|---:|---:|
| PDF pages | 3.21 ms / 3.75 MB / 4,033 | 11.53 ms / 4.63 MB / 14,877 | 100.60 ms / 23.18 MB / 137,932 | 521.72 ms / 111.45 MB / 687,425 |
| Template + PDF pages | 3.34 ms / 2.12 MB / 4,159 | 11.04 ms / 3.13 MB / 15,858 | 104.47 ms / 24.36 MB / 148,135 | 539.47 ms / 116.27 MB / 738,635 |
| Web-fetch image tiles | 10.51 ms / 2.82 MB / 1,661 | 11.44 ms / 3.10 MB / 2,120 | 38.03 ms / 6.93 MB / 5,261 | 38.44 ms / 11.54 MB / 19,929 |
| Inline image tiles | 10.96 ms / 5.85 MB / 1,247 | 9.10 ms / 2.93 MB / 1,619 | 40.20 ms / 20.92 MB / 4,746 | 35.79 ms / 11.82 MB / 19,416 |

## Public Go library (`Document.WritePDF`; 2 pages cold, later rows warm)

The public API runs directly, without a CLI process or disk HTML. The rows
below come from the `public-pdf` recovery mode (the same `BenchmarkLibraryPDF`
benchmark), median of three fresh `1x` processes. The canonical `make bench-lib`
run measures the same benchmark warm at `-benchtime=10x`; its 500-page row is
493.99 ms at 109.35 MB `B/op`.

| Pages | Time | B/op | allocs/op |
|---:|---:|---:|---:|
| 2 | 5.50 ms | 4.11 MB | 4,151 |
| 5 | 8.99 ms | 6.87 MB | 8,179 |
| 10 | 13.41 ms | 3.85 MB | 14,854 |
| 20 | 21.92 ms | 6.09 MB | 28,421 |
| 50 | 54.49 ms | 12.55 MB | 69,474 |
| 100 | 107.20 ms | 23.52 MB | 137,948 |
| 200 | 214.32 ms | 45.39 MB | 275,217 |
| 250 | 267.34 ms | 57.33 MB | 344,028 |
| 500 | 554.56 ms | 113.09 MB | 687,420 |

## Public Go library image (`ImageDocument.WriteImage`; 2 pages cold, later rows warm)

Tile counts are the benchmark's requested output tiles, 1024 px wide. The PNG
writer is lossless with filter-none level-2 compression; 250 tiles encode to
141,917 B and 500 tiles to 282,749 B.

| Tiles | Time | B/op | allocs/op |
|---:|---:|---:|---:|
| 2 | 14.26 ms | 11.91 MB | 490 |
| 5 | 11.40 ms | 3.45 MB | 578 |
| 10 | 13.33 ms | 3.68 MB | 873 |
| 20 | 14.10 ms | 3.82 MB | 1,137 |
| 50 | 16.67 ms | 4.20 MB | 1,918 |
| 100 | 28.82 ms | 20.00 MB | 3,215 |
| 200 | 59.39 ms | 37.09 MB | 5,793 |
| 250 | 16.86 ms | 6.38 MB | 7,179 |
| 500 | 33.51 ms | 9.92 MB | 13,705 |

## Public Python library (`make python-benchmarks`; warm medians)

The opt-in c-shared Python bindings on the same fixture and page data:
`Document.pdf()` for PDF and `ImageDocument.image()` for PNG tiles. One warmup
plus 10 timed iterations per size; the median is shown. This is a warm library
measurement, not a fresh-process CLI row. The Python path runs the same
in-process engine, so the rows land close to the public Go library: 3.52 ms at
2 pages and 504.93 ms at 500 pages, with 500 image tiles at 31.66 ms.

| Pages | PDF median | Tiles | Image median |
|---:|---:|---:|---:|
| 2 | 3.52 ms | 2 | 9.67 ms |
| 5 | 5.75 ms | 5 | 10.01 ms |
| 10 | 10.76 ms | 10 | 11.91 ms |
| 20 | 21.91 ms | 20 | 11.76 ms |
| 50 | 49.63 ms | 50 | 15.06 ms |
| 100 | 98.44 ms | 100 | 25.19 ms |
| 200 | 198.74 ms | 200 | 47.29 ms |
| 250 | 246.38 ms | 250 | 15.65 ms |
| 500 | 504.93 ms | 500 | 31.66 ms |

## Library rows relative to the wkhtmltopdf CLI (indicative)

The library paths do not launch a process, so these ratios are not exactly
like-for-like: the wkhtmltopdf value includes process startup while the
library value is the in-process conversion. Both sides are from the same
2026-09-13 capture.

| Pages | wkhtmltopdf CLI | Public library PDF | Multiplier | In-process PDF | Multiplier |
|---:|---:|---:|---:|---:|---:|
| 2 | 258 ms | 5.50 ms | 46.91x | 5.20 ms | 49.62x |
| 5 | 269 ms | 8.99 ms | 29.92x | 7.89 ms | 34.09x |
| 10 | 279 ms | 13.41 ms | 20.81x | 11.75 ms | 23.74x |
| 20 | 310 ms | 21.92 ms | 14.14x | 21.44 ms | 14.46x |
| 50 | 393 ms | 54.49 ms | 7.21x | 54.76 ms | 7.18x |
| 100 | 532 ms | 107.20 ms | 4.96x | 103.51 ms | 5.14x |
| 200 | 814 ms | 214.32 ms | 3.80x | 207.91 ms | 3.91x |
| 250 | 973 ms | 267.34 ms | 3.64x | 262.21 ms | 3.71x |
| 500 | 1.718 s | 554.56 ms | 3.10x | 539.33 ms | 3.19x |

## CLI process, gowk only (`cli-rss`; cold full-process runs)

`/usr/bin/time %e` at 10 ms resolution, so the small sizes quantize. The CLI
comparison table above is the vs-wkhtmltopdf view; this is the gowk-only
cross-check. Median of 3 timed runs after one warmup.

| Pages | Time | Peak RSS |
|---:|---:|---:|
| 2 | 10 ms | 19,968 KiB |
| 5 | 10 ms | 22,080 KiB |
| 10 | 20 ms | 24,768 KiB |
| 20 | 30 ms | 26,496 KiB |
| 50 | 60 ms | 29,568 KiB |
| 100 | 120 ms | 35,712 KiB |
| 200 | 240 ms | 45,696 KiB |
| 250 | 290 ms | 51,264 KiB |
| 500 | 570 ms | 79,296 KiB |

## Historical captures

- 2026-09-12 full capture: the previous consolidated capture (0.2.5 working
  tree, 18.50x at 2 pages and 3.13x at 500 pages). Full tables remain in
  [performance.md](performance.md).
- 2026-09-11 perf-time phase-7 closure capture: 2-page and 500-page rows in
  [performance.md](performance.md).
- 2026-09-11 recovery capture: the previous CLI vs wkhtmltopdf table.
- 2026-08-19 0.2.4 matrices: full internal, public PDF, and image rows in
  [testdata/golden/benchmarks/README.md](../testdata/golden/benchmarks/README.md).
