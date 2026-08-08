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
| PDF pages | 6.7ms | 11.6ms | 22.2ms | 41.1ms | 115ms | 249ms | 481ms | 590ms | 1.90s |
| Template + PDF pages | 5.2ms | 9.2ms | 18.6ms | 48.5ms | 117ms | 223ms | 459ms | 588ms | 1.69s |
| Web-fetch image tiles | 257.33ms | 258.05ms | 281.10ms | 310.47ms | 356.66ms | 413.68ms | 506.42ms | 564.00ms | 970.72ms |
| Inline image tiles | 209.50ms | 220.61ms | 255.35ms | 282.33ms | 303.54ms | 340.31ms | 439.46ms | 491.22ms | 788.43ms |

PDF / Template: profile-guided residual optimization wave on
`feature/optimization` (2026-08-09). The locked 500-page PDF count-3 median is
**1.628s / 678.8MB / 1.103M allocs**, versus the published
**2.10s / 1.48GB / 3.93M** bar. Image-tile rows are unchanged.

### Original snapshot comparison

The first committed benchmark snapshot on this branch is `2a0f18b`
(`chore: add conversion benchmarks`). The current matrix below uses the same
one-iteration command, so this is the direct recorded comparison for the
500-page report:

| Metric | Original snapshot | Current snapshot | Change |
|---|---:|---:|---:|
| PDF time | 14.135s | 1.903s | **−86.5%** |
| PDF B/op | 3.80GB | 678.6MB | **−82.1%** |
| PDF allocs/op | 14.35M | 1.103M | **−92.3%** |
| Template + PDF time | 13.670s | 1.693s | **−87.6%** |
| Template + PDF B/op | 3.80GB | 683.5MB | **−82.0%** |
| Template + PDF allocs/op | 14.40M | 1.154M | **−92.0%** |

The separate locked-gate result is the more stable count-3 median shown above;
the comparison table intentionally uses one iteration on both snapshots.

### Earlier wkhtmltopdf reference

The performance ledger recorded a process-level run of wkhtmltopdf 0.12.6.1
against the same generated report matrix on the same WSL2 host. The table below
adds the current in-process Go benchmark metrics for context:

| Pages | wkhtmltopdf wall | wk peak RSS | Current Go PDF wall | Current Go B/op | Current Go allocs/op |
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

This is directional, not a strict apples-to-apples process comparison:
wkhtmltopdf wall/RSS includes its native process, while the current Go wall and
`B/op` columns are from the in-process `testing.B` benchmark. `B/op` is
cumulative allocation traffic, not peak RSS. The current profiled benchmark
process reached about **391MiB RSS** at 500 pages; a fresh current CLI
process-level matrix has not been recorded in this snapshot.

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
