# Benchmark templates

These templates are source fixtures for the Go benchmarks in
`internal/convert/benchmarks_test.go`. They are kept below the golden corpus so
benchmark inputs stay reviewable and reproducible without committing generated
PDF or PNG output. The checked-in result snapshot is
`benchmark-results.txt`.

The benchmark matrix uses 2, 5, 10, 20, 50, 100, 200, 250, and 500 page-sized
sections for PDF and template rendering. Image mode renders one raster canvas,
so its matching workloads use 2, 5, 10, 20, 50, 100, 200, 250, and 500 image
tiles.

Templates:

- `templates/report.html.tmpl` — paginated report used by the PDF and
  template-execution benchmarks.
- `templates/web-fetch-image.html.tmpl` — one-page image grid served by an
  in-process HTTP server for the web-fetch image benchmark.
- `templates/image-grid.html.tmpl` — the same image-grid workload with an
  inline data URL for the local image benchmark.
- `templates/movie-listing.html.tmpl` — a live movie/TV catalogue template
  populated from TVmaze show metadata and real poster URLs.

## Live movie listing benchmark

The live benchmark uses TVmaze's public `/shows?page=0` API and its HTTPS image
CDN. It is opt-in because it requires internet access, can vary with network
latency and current API data, and is subject to the provider's rate limits.
The deterministic local benchmark remains the baseline for CI.

Run the live API fetch benchmark:

```sh
GOWKHTMLTOPDF_LIVE_BENCHMARK=1 \
  go test ./internal/convert -run '^$' \
  -bench '^BenchmarkLiveMovieData$' -benchmem -benchtime=1x -count=1
```

Run selected live listing sizes (the full matrix is available by removing the
sub-benchmark filter):

```sh
GOWKHTMLTOPDF_LIVE_BENCHMARK=1 \
  go test ./internal/convert -run '^$' \
  -bench '^BenchmarkLiveMovieListing/(2Images|5Images|10Images)$' \
  -benchmem -benchtime=1x -count=1
```

Generate a viewable live-data sample in the ignored `output/` directory:

```sh
GOWKHTMLTOPDF_LIVE_BENCHMARK=1 \
GOWKHTMLTOPDF_GENERATE_BENCHMARK_OUTPUTS=1 \
  go test ./internal/convert -run '^TestGenerateLiveMovieOutput$' -count=1
```

This writes `live-movie-listing-010.pdf` and
`live-movie-listing-010.png`.

## Current 2026-08-12 benchmark snapshot

This is a fresh one-iteration capture from the current worktree. It is kept
separate from the historical/count-3 comparisons below: wall time is sensitive
to host state, while Go `B/op` is cumulative allocation traffic rather than
process RSS. The complete raw Go rows are in
[`benchmark-results.txt`](benchmark-results.txt), Snapshot D.

```sh
GOCACHE=/tmp/gowk-go-cache \
  go test ./internal/convert -run '^$' \
  -bench 'Benchmark(PDFPages|TemplatePages|WebFetchImage|ImageAssets)$' \
  -benchmem -benchtime=1x -count=1
```

| In-process workload | 2 | 10 | 100 | 500 |
|---|---:|---:|---:|---:|
| PDF pages | 4.47ms / 2.1MB / 4.9K allocs | 16.67ms / 4.9MB / 20.2K | 193.47ms / 35.5MB / 193.7K | **1359.33ms / 171.5MB / 966.5K** |
| Template + PDF pages | 6.68ms / 2.1MB / 5.1K | 23.12ms / 5.0MB / 21.3K | 206.15ms / 36.7MB / 203.9K | **1484.45ms / 176.4MB / 1017.8K** |
| Web-fetch image tiles | 23.61ms / 54.0MB / 7.8K | 21.00ms / 63.3MB / 9.3K | 61.98ms / 67.5MB / 13.1K | **254.81ms / 148.8MB / 29.7K** |
| Inline image tiles | 23.23ms / 55.1MB / 6.8K | 16.56ms / 57.4MB / 7.5K | 55.15ms / 61.7MB / 11.3K | **240.74ms / 143.4MB / 28.0K** |

### Current direct CLI process measurement

The PDF binary was freshly built from the current checkout and run once per
size under `/usr/bin/time -f '%e %M'`, with `--quiet
--enable-local-file-access`. The generated report used the benchmark report
CSS, marker, and 20 rows per requested page; Ghostscript verified every output
page count. This is an application-process measurement: elapsed time is in
milliseconds and peak RSS is in KiB. It is **not** directly comparable to Go
`B/op`, and the one-run values should not be read as a stable regression claim.
The timer reports centisecond resolution, so the 10–20 ms rows are especially
coarse.

| Requested / rendered pages | Elapsed | Peak RSS | PDF bytes |
|---:|---:|---:|---:|
| 2 / 2 | 10 ms | 22,464 KiB | 43,145 |
| 5 / 5 | 20 ms | 23,040 KiB | 51,460 |
| 10 / 10 | 20 ms | 23,616 KiB | 65,358 |
| 20 / 20 | 40 ms | 23,808 KiB | 91,863 |
| 50 / 50 | 80 ms | 26,264 KiB | 172,078 |
| 100 / 100 | 170 ms | 28,648 KiB | 305,592 |
| 200 / 200 | 340 ms | 35,404 KiB | 573,416 |
| 250 / 250 | 460 ms | 38,896 KiB | 708,125 |
| 500 / 500 | **960 ms** | **54,632 KiB** | **1,385,450** |

The report marker activates the current certified page-island path. Therefore
this direct matrix measures that workload only; it is not a generic-HTML
fidelity or RSS guarantee. See the critical review's CR-02 before broadening
its performance claim.

## Historical in-process recorded results

The following are historical one-iteration in-process snapshots from Go 1.26.4
on Linux/amd64 under WSL2 with 24 CPUs. They are retained for allocation and
benchmark-history context; the current direct CLI comparison is documented in
the section below. The report template contains 20 realistic rows per page,
filling the available page without spilling into a second physical page.
PDF and template rows are page counts; image rows are image tile counts because
image mode renders one raster canvas.

Historical snapshot re-measured 2026-08-09 after the certified workspace path
landed. Image rows were re-run in the same session.

| Workload | 2 | 5 | 10 | 20 | 50 | 100 | 200 | 250 | 500 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| PDF pages | 3.18ms | 5.87ms | 9.34ms | 19.63ms | 51.41ms | 104.14ms | 195.48ms | 251.24ms | 518.12ms |
| Template + PDF pages | 4.86ms | 6.49ms | 13.35ms | 23.14ms | 55.58ms | 108.49ms | 212.37ms | 277.45ms | 552.12ms |
| Web-fetch image tiles | 17.99ms | 16.97ms | 25.96ms | 25.16ms | 34.46ms | 54.84ms | 93.96ms | 117.38ms | 208.25ms |
| Inline image tiles | 16.52ms | 14.47ms | 17.22ms | 20.52ms | 30.00ms | 49.48ms | 85.71ms | 99.72ms | 182.83ms |

PDF / Template: certified workspace path (2026-08-09). The current 500-page
PDF count-3 median is **0.500s / 157.6MB / 529.4K allocs**, versus the
published **2.10s / 1.48GB / 3.93M** bar. Generated PDF artifacts are local
only and are not part of the benchmark snapshot commit.

### Historical snapshot comparison

The 500-page report versus the published perf-wave snapshot (same
one-iteration matrix command):

| Metric | Perf-wave snapshot | Workspace snapshot | Change |
|---|---:|---:|---:|
| PDF time | 0.873s | 0.518s | **−40.7%** |
| PDF B/op | 335.8MB | 157.6MB | **−53.1%** |
| PDF allocs/op | 517,875 | 529,451 | **+2.2%** |
| Template + PDF time | 0.942s | 0.552s | **−41.4%** |
| Template + PDF B/op | 340.2MB | 162.5MB | **−52.2%** |
| Template + PDF allocs/op | 569,123 | 580,715 | **+2.0%** |

B/op is deterministic for identical code paths; wall time on one-shot laptop
runs is noisier (±20% CPU state). The workspace reuses a certified page
island's display-list backing storage, substantially reducing allocation
traffic while retaining a small increase in allocation count.
The separate locked-gate result is the more stable count-3 median shown
above; the comparison table intentionally uses one iteration on both
snapshots.

### Direct CLI comparison: gowkhtmltopdf vs wkhtmltopdf

This matrix was re-measured on 2026-08-09 using the same generated report
fixtures for both command-line engines. The Go binary was freshly built from
the current source (`go build ./cmd/gowkhtmltopdf`); wkhtmltopdf was
0.12.6.1. Each cell is the median of three process runs. Timing is wall time
in milliseconds; RSS is peak resident set size from `/usr/bin/time -f '%M'` in
KiB. Both commands used `--quiet --enable-local-file-access` and every output
passed the expected page-count check.

| Pages | Gowk time | wkhtmltopdf time | Gowk RSS | wkhtmltopdf RSS | Gowk PDF bytes | wkhtmltopdf PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|
| 2 | 10 ms | 220 ms | 18,432 KiB | 35,268 KiB | 21,626 | 18,486 |
| 5 | 10 ms | 230 ms | 17,208 KiB | 35,528 KiB | 29,921 | 30,584 |
| 10 | 20 ms | 250 ms | 17,832 KiB | 36,408 KiB | 43,905 | 50,994 |
| 20 | 30 ms | 270 ms | 18,052 KiB | 38,080 KiB | 70,379 | 90,742 |
| 50 | 70 ms | 360 ms | 20,644 KiB | 43,052 KiB | 150,192 | 210,678 |
| 100 | 140 ms | 500 ms | 23,340 KiB | 51,092 KiB | 284,139 | 411,260 |
| 200 | 300 ms | 800 ms | 30,256 KiB | 67,576 KiB | 551,741 | 816,285 |
| 250 | 380 ms | 940 ms | 34,200 KiB | 75,948 KiB | 685,323 | 1,019,315 |
| 500 | 890 ms | 1,720 ms | 50,888 KiB | 116,512 KiB | 1,357,738 | 2,036,776 |

Gowkhtmltopdf was faster and used less RSS at every tested page size. At 500
pages it was approximately 1.9x faster and used 56.3% less RSS. Gowk PDFs were
smaller from 5 pages onward; wkhtmltopdf produced the smaller 2-page PDF.

| Use case | Preferred engine | Reason |
|---|---|---|
| Memory-constrained server/container | gowkhtmltopdf | Lower RSS across the full matrix |
| High-volume or large PDF generation | gowkhtmltopdf | Faster and lower RSS at 50–500 pages |
| Smallest PDF at exactly 2 pages | wkhtmltopdf | 18,486 bytes versus 21,626 bytes |
| Legacy Qt/WebKit rendering compatibility | wkhtmltopdf | Preserves the established renderer |
| General default for this workload | gowkhtmltopdf | Faster, lower RSS, and smaller PDFs from 5 pages onward |

This is a direct process comparison. It should not be conflated with Go's
in-process `B/op` metric: `B/op` is cumulative allocation traffic, not peak
RSS. The benchmark fixture and renderer settings are controlled, so results
should be treated as workload-specific rather than a universal browser-engine
ranking.

The raw `go test` output, including allocations, is in
[`benchmark-results.txt`](benchmark-results.txt). The benchmark implementation
is [`internal/convert/benchmarks_test.go`](../../../internal/convert/benchmarks_test.go).

## Local generated artifacts

To save viewable PDF and PNG artifacts for every matrix size into
`output/`, run (needs network for TVmaze poster CDN tiles):

```sh
GOWKHTMLTOPDF_GENERATE_BENCHMARK_OUTPUTS=1 \
  go test ./internal/convert -run '^TestGenerateBenchmarkOutputs$' -count=1
```

| Files | Source |
|-------|--------|
| `pdf-pages-*.pdf` | Local report template → PDF |
| `template-pages-*.pdf` | Template execution + PDF |
| `inline-images-*.png` | Synthetic PNG as `data:` URLs (offline) |
| `web-fetch-images-*.png` | **Real TVmaze CDN poster URLs** fetched over HTTPS |
| `live-movie-listing-010.*` | Live catalogue (separate test; see above) |

Timed `BenchmarkWebFetchImage` still uses an in-process `httptest` server so
CI remains offline and the recorded timing matrix stays reproducible. Artifact
generation intentionally hits the public CDN so samples match real-world fetch.

Run the focused benchmarks with:

```sh
go test ./internal/convert -run '^$' -bench 'Benchmark(PDFPages|TemplatePages|WebFetchImage|ImageAssets)$' -benchmem -count=1
```

To reproduce the checked-in one-iteration snapshot exactly, add
`-benchtime=1x` to the command above.
