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

| Workload | 2 | 5 | 10 | 20 | 50 | 100 | 200 | 250 | 500 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| PDF pages | 7.8ms | 16.0ms | 33.0ms | 59.9ms | 159ms | 348ms | 700ms | 854ms | 3.09s |
| Template + PDF pages | 6.0ms | 15.2ms | 31.8ms | 57.5ms | 165ms | 324ms | 709ms | 891ms | 3.03s |
| Web-fetch image tiles | 257.33ms | 258.05ms | 281.10ms | 310.47ms | 356.66ms | 413.68ms | 506.42ms | 564.00ms | 970.72ms |
| Inline image tiles | 209.50ms | 220.61ms | 255.35ms | 282.33ms | 303.54ms | 340.31ms | 439.46ms | 491.22ms | 788.43ms |

PDF / Template rows: post-optimization baseline on `feature/optimization`
(2026-08-08). 500-page PDF is count-3 median (~3.09s / 2.47GB / 5.91M allocs);
smaller sizes are one-shot. Image-tile rows are still the prior full-matrix
snapshot.

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
