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

## Recorded results

The following are one-iteration snapshots from Go 1.26.4 on Linux/amd64 under
WSL2 with 24 CPUs. The report template contains 20 realistic rows per page,
filling the available page without spilling into a second physical page.
PDF and template rows are page counts; image rows are image tile counts because
image mode renders one raster canvas.

Re-measured 2026-08-09 on the working tree (lint-cleanup wave, no benchmark
code changed). Image rows were re-run in the same session: the previously
recorded image numbers were stale carry-overs from an older era.

| Workload | 2 | 5 | 10 | 20 | 50 | 100 | 200 | 250 | 500 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| PDF pages | 5.8ms | 7.8ms | 14.2ms | 33.2ms | 79.8ms | 207ms | 384ms | 422ms | 1.01s |
| Template + PDF pages | 4.2ms | 7.7ms | 16.4ms | 27.0ms | 85.1ms | 175ms | 393ms | 439ms | 1.05s |
| Web-fetch image tiles | 19.93ms | 17.79ms | 22.38ms | 22.95ms | 33.34ms | 58.80ms | 92.40ms | 103.43ms | 199.89ms |
| Inline image tiles | 16.77ms | 14.97ms | 20.17ms | 19.06ms | 31.13ms | 51.24ms | 84.08ms | 93.75ms | 179.23ms |

PDF / Template: profile-guided residual optimization wave on
`feature/optimization` (2026-08-09), re-measured after the lint-cleanup wave.
The current 500-page PDF count-3 median is **0.936s / 392.8MB / 535K allocs**
(latest count-3 run; median across two count-3 runs is 1.067s / ~393MB /
~535K allocs), versus the published **2.10s / 1.48GB / 3.93M** bar. Image-tile
rows were re-measured in the same session (previously carried from an older
era).

### Snapshot comparison

The 500-page report dropped further versus the previous committed snapshot
(`aa8d446`, recorded 2026-08-09). The main driver is the working tree's
`smartShrinkMinOverflow` threshold in `internal/convert`: the benchmark report
overflows its content area by 0.00pt, so it no longer pays a second full
500-page smart-shrink layout pass. Both snapshots use the same one-iteration
matrix command:

| Metric | aa8d446 snapshot | Current snapshot | Change |
|---|---:|---:|---:|
| PDF time | 1.903s | 1.013s | **−46.8%** |
| PDF B/op | 678.6MB | 392.2MB | **−42.1%** |
| PDF allocs/op | 1.103M | 535,064 | **−51.5%** |
| Template + PDF time | 1.693s | 1.047s | **−38.2%** |
| Template + PDF B/op | 683.5MB | 397.6MB | **−41.8%** |
| Template + PDF allocs/op | 1.154M | 586,355 | **−49.2%** |
| Web-fetch 500 tiles | 970.72ms | 199.89ms | **−79.4%** |
| Inline 500 tiles | 788.43ms | 179.23ms | **−77.3%** |

The image rows are not directly comparable: the previous rows were carried
forward from an earlier era rather than re-measured at the wave; the current
rows are re-measured with the identical benchmark code on the same host.
The separate locked-gate result is the more stable count-3 median shown above;
the comparison table intentionally uses one iteration on both snapshots.

### Earlier wkhtmltopdf reference

The performance ledger recorded a process-level run of wkhtmltopdf 0.12.6.1
against the same generated report matrix on the same WSL2 host. The table below
adds the current in-process Go benchmark metrics for context:

| Pages | wkhtmltopdf wall | wk peak RSS | Current Go PDF wall | Current Go B/op | Current Go allocs/op |
|---:|---:|---:|---:|---:|---:|
| 2 | 0.39s* | 34MB | 5.8ms | 3.4MB | 3,472 |
| 5 | 0.23s | 35MB | 7.8ms | 5.2MB | 6,511 |
| 10 | 0.25s | 35MB | 14.2ms | 9.0MB | 11,837 |
| 20 | 0.28s | 37MB | 33.2ms | 16.8MB | 22,383 |
| 50 | 0.37s | 42MB | 79.8ms | 40.4MB | 54,342 |
| 100 | 0.60s | 50MB | 207.5ms | 79.6MB | 107,689 |
| 200 | 0.89s | 66MB | 383.6ms | 158.2MB | 214,516 |
| 250 | 0.99s | 74MB | 422.2ms | 196.9MB | 267,961 |
| 500 | 2.05s | 114MB | 1.013s | 392.2MB | 535,064 |

This is directional, not a strict apples-to-apples process comparison:
wkhtmltopdf wall/RSS includes its native process, while the current Go wall and
`B/op` columns are from the in-process `testing.B` benchmark. `B/op` is
cumulative allocation traffic, not peak RSS.

\* The first wkhtmltopdf invocation was colder because of Qt startup.

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
