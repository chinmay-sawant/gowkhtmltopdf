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

Re-measured 2026-08-09 on `feature/optimization` after the certified
workspace path landed. Image rows were re-run in the same session.

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

### Snapshot comparison

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

### Earlier wkhtmltopdf reference

The performance ledger recorded a process-level run of wkhtmltopdf 0.12.6.1
against the same generated report matrix on the same WSL2 host. The table below
adds the current in-process Go benchmark metrics for context:

| Pages | wkhtmltopdf wall | wk peak RSS | Current Go PDF wall | Current Go B/op | Current Go allocs/op |
|---:|---:|---:|---:|---:|---:|
| 2 | 0.39s* | 34MB | 3.2ms | 2.6MB | 3,163 |
| 5 | 0.23s | 35MB | 5.9ms | 2.9MB | 6,131 |
| 10 | 0.25s | 35MB | 9.3ms | 3.6MB | 11,349 |
| 20 | 0.28s | 37MB | 19.6ms | 7.6MB | 21,815 |
| 50 | 0.37s | 42MB | 51.4ms | 17.0MB | 53,447 |
| 100 | 0.60s | 50MB | 104.1ms | 32.7MB | 106,229 |
| 200 | 0.89s | 66MB | 195.5ms | 64.1MB | 212,012 |
| 250 | 0.99s | 74MB | 251.2ms | 79.4MB | 264,905 |
| 500 | 2.05s | 114MB | 0.518s | 157.6MB | 529,451 |

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
