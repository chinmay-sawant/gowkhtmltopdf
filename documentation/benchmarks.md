# Benchmarks

Consolidated benchmark capture for gowkhtmltopdf. This page is the keeper for
the current numbers; the deep historical record stays in
[performance.md](performance.md) and
[testdata/golden/benchmarks/README.md](../testdata/golden/benchmarks/README.md).

## Current capture: 2026-09-12

- Engine: generic `bin/gowkhtmltopdf`, `VERSION` 0.2.5 on the 0.2.6 working
  tree (`chore/review-026`), built with `make build`
- Host: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64, 13th Gen Intel Core
  i7-13700HX (24 CPUs), 7.6 GiB RAM
- Toolchain: `go version go1.26.4 linux/amd64`
- Fixture: `testdata/golden/benchmarks/templates/report.html.tmpl`
  (20 invoice rows per requested page), sha256 `e3b5387b...`
- CLI and external rows: median of 3 timed runs after one warmup, wall time
  via `/usr/bin/time`, peak RSS via `%M` (Puppeteer samples the process tree)
- In-process rows: `go test -benchmem -benchtime=1x -count=1`, one fresh
  process per round; the value is the median of three rounds and `B/op` and
  `allocs/op` are the median-time sample's raw values, never averages
- Reproduce: `make build`, `make bench-cli-compare`,
  `./scripts/bench-external.sh`, and
  `./scripts/bench-performance-recovery.sh --mode=<mode>` (see the modes below)

Raw captures (gitignored, local): `plans/0.2.6/perf-review/results/2026-09-12/`.
Machine-generated committed artifacts: `testdata/golden/benchmarks/`
`cli-compare.md` / `weasyprint-compare.md` / `puppeteer-compare.md` plus their
`-results.csv` siblings.

## CLI vs wkhtmltopdf

Source: `testdata/golden/benchmarks/cli-compare.md`. gowk runs
`--quiet --allow-local-files -o OUTPUT INPUT`; wkhtmltopdf 0.12.6.1 (patched
Qt) runs `--quiet --enable-local-file-access INPUT OUTPUT`.

| Pages | gowk time | wkhtmltopdf time | Speedup | gowk RSS | wkhtmltopdf RSS | gowk PDF | wkhtmltopdf PDF |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 14 ms | 260 ms | 18.50x | 19,200 KiB | 44,720 KiB | 34,210 B | 18,486 B |
| 5 | 19 ms | 269 ms | 14.37x | 21,696 KiB | 45,168 KiB | 42,795 B | 30,584 B |
| 10 | 26 ms | 286 ms | 11.18x | 25,152 KiB | 46,016 KiB | 57,239 B | 50,994 B |
| 20 | 36 ms | 315 ms | 8.74x | 25,920 KiB | 47,620 KiB | 84,680 B | 90,742 B |
| 50 | 69 ms | 403 ms | 5.85x | 29,760 KiB | 52,128 KiB | 167,525 B | 210,678 B |
| 100 | 126 ms | 546 ms | 4.35x | 35,904 KiB | 59,460 KiB | 306,321 B | 411,260 B |
| 200 | 234 ms | 852 ms | 3.64x | 44,928 KiB | 74,308 KiB | 583,670 B | 816,285 B |
| 250 | 288 ms | 1.008 s | 3.50x | 51,264 KiB | 81,820 KiB | 722,322 B | 1,019,315 B |
| 500 | 562 ms | 1.760 s | 3.13x | 79,296 KiB | 123,076 KiB | 1,420,537 B | 2,036,776 B |

gowkhtmltopdf is faster and uses less peak RSS at every tested size, including
500 pages. This supersedes the older claim that gowk RSS was higher from 100
pages on.

## External engines (2 / 10 / 50 / 100 pages)

### WeasyPrint 69.0

Source: `testdata/golden/benchmarks/weasyprint-compare.md`.

| Pages | gowk time | WeasyPrint time | Speedup | gowk RSS | WeasyPrint RSS | gowk PDF | WeasyPrint PDF |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 16 ms | 634 ms | 40.86x | 19,584 KiB | 81,648 KiB | 34,210 B | 15,584 B |
| 10 | 27 ms | 1.434 s | 53.62x | 24,576 KiB | 111,104 KiB | 57,239 B | 45,174 B |
| 50 | 71 ms | 5.441 s | 76.88x | 29,760 KiB | 252,412 KiB | 167,525 B | 190,544 B |
| 100 | 124 ms | 11.072 s | 89.43x | 34,752 KiB | 427,468 KiB | 306,321 B | 372,867 B |

### Puppeteer / Chrome (puppeteer-core 24.43.1 + Chrome 143)

Source: `testdata/golden/benchmarks/puppeteer-compare.md`.

| Pages | gowk time | Puppeteer time | Speedup | gowk RSS | Puppeteer RSS | gowk PDF | Puppeteer PDF |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 16 ms | 1.445 s | 92.85x | 19,200 KiB | 940,600 KiB | 34,210 B | 134,319 B |
| 10 | 26 ms | 1.488 s | 57.16x | 24,192 KiB | 1,019,952 KiB | 57,239 B | 450,799 B |
| 50 | 72 ms | 1.801 s | 25.10x | 29,760 KiB | 1,119,360 KiB | 167,525 B | 1,981,892 B |
| 100 | 127 ms | 2.178 s | 17.16x | 35,136 KiB | 1,240,920 KiB | 306,321 B | 3,936,067 B |

Puppeteer RSS is the peak process-tree reading for the Node driver plus
headless Chrome, not a single-process `%M` value.

## In-process engine (internal, generic path)

Full ascending warm matrix in one process per round; the 2-page row is the
first conversion in its process and carries the one-time font load. Median of
three rounds.

| Pages | Time | B/op | allocs/op |
|---:|---:|---:|---:|
| 2 | 5.14 ms | 4.09 MB | 4,136 |
| 5 | 7.75 ms | 6.84 MB | 8,165 |
| 10 | 11.77 ms | 4.63 MB | 14,879 |
| 20 | 21.24 ms | 6.02 MB | 28,411 |
| 50 | 53.54 ms | 13.24 MB | 69,498 |
| 100 | 106.56 ms | 23.26 MB | 137,955 |
| 200 | 204.88 ms | 44.66 MB | 275,206 |
| 250 | 259.06 ms | 56.50 MB | 344,017 |
| 500 | 535.34 ms | 111.44 MB | 687,421 |

## Public Go library (`Document.WritePDF`)

`make bench-lib` calls the public API directly, without a CLI process or disk
HTML. Median of three fresh 1x processes.

| Pages | Time | B/op | allocs/op |
|---:|---:|---:|---:|
| 2 | 5.75 ms | 4.11 MB | 4,144 |
| 5 | 8.83 ms | 6.87 MB | 8,179 |
| 10 | 13.71 ms | 4.67 MB | 14,882 |
| 20 | 21.13 ms | 6.09 MB | 28,417 |
| 50 | 51.83 ms | 12.59 MB | 69,478 |
| 100 | 107.16 ms | 23.45 MB | 137,942 |
| 200 | 222.68 ms | 45.32 MB | 275,212 |
| 250 | 266.55 ms | 57.32 MB | 344,020 |
| 500 | 532.24 ms | 113.10 MB | 687,423 |

## Public Go library image (`ImageDocument.WriteImage`)

Tile counts are the benchmark's requested output tiles, 1024 px wide. The PNG
writer is lossless with filter-none level-2 compression; 250 tiles encode to
141,917 B and 500 tiles to 282,749 B.

| Tiles | Time | B/op | allocs/op |
|---:|---:|---:|---:|
| 2 | 14.12 ms | 11.91 MB | 490 |
| 5 | 12.25 ms | 3.45 MB | 578 |
| 10 | 13.75 ms | 3.68 MB | 872 |
| 20 | 12.96 ms | 3.82 MB | 1,137 |
| 50 | 17.39 ms | 4.20 MB | 1,917 |
| 100 | 30.16 ms | 20.00 MB | 3,215 |
| 200 | 55.80 ms | 37.09 MB | 5,793 |
| 250 | 16.78 ms | 6.38 MB | 7,179 |
| 500 | 34.02 ms | 10.40 MB | 13,705 |

## CLI process, gowk only (`cli-rss`)

`/usr/bin/time %e` at 10 ms resolution, so the small sizes quantize. The CLI
comparison table above is the vs-wkhtmltopdf view; this is the gowk-only
cross-check. Median of 3 timed runs after one warmup.

| Pages | Time | Peak RSS |
|---:|---:|---:|
| 2 | 10 ms | 19,584 KiB |
| 5 | 10 ms | 22,080 KiB |
| 10 | 20 ms | 24,768 KiB |
| 20 | 30 ms | 26,496 KiB |
| 50 | 70 ms | 29,760 KiB |
| 100 | 120 ms | 34,752 KiB |
| 200 | 220 ms | 46,464 KiB |
| 250 | 280 ms | 51,840 KiB |
| 500 | 560 ms | 80,064 KiB |

## Historical captures

- 2026-09-11 perf-time phase-7 closure capture: 2-page and 500-page rows in
  [performance.md](performance.md).
- 2026-09-11 recovery capture: the previous CLI vs wkhtmltopdf table.
- 2026-08-19 0.2.4 matrices: full internal, public PDF, and image rows in
  [testdata/golden/benchmarks/README.md](../testdata/golden/benchmarks/README.md).
