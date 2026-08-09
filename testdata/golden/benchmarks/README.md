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

Re-measured 2026-08-09 on the working tree after the perf-review wave (5
parallel review agents → fix agents; see `skills/perf-review/SKILLS.md`).
Image rows re-run in the same session.

| Workload | 2 | 5 | 10 | 20 | 50 | 100 | 200 | 250 | 500 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| PDF pages | 5.2ms | 7.1ms | 16.3ms | 32.0ms | 72.2ms | 168ms | 316ms | 430ms | 0.87s |
| Template + PDF pages | 3.9ms | 6.1ms | 12.5ms | 23.6ms | 73.2ms | 165ms | 346ms | 422ms | 0.94s |
| Web-fetch image tiles | 18.53ms | 22.89ms | 22.95ms | 22.50ms | 32.36ms | 51.08ms | 89.32ms | 110.31ms | 200.89ms |
| Inline image tiles | 16.97ms | 20.29ms | 21.04ms | 24.24ms | 32.20ms | 49.67ms | 83.27ms | 101.90ms | 192.13ms |

PDF / Template: perf-review wave on the working tree (2026-08-09). The
current 500-page PDF count-3 median is **0.878s / 335.7MB / 517.9K allocs**,
versus the published **2.10s / 1.48GB / 3.93M** bar.

### Snapshot comparison

The 500-page report vs the pre-wave snapshot taken earlier the same day
(same one-iteration matrix command):

| Metric | Pre-wave snapshot | Perf-wave snapshot | Change |
|---|---:|---:|---:|
| PDF time | 1.013s | 0.873s | **−13.8%** |
| PDF B/op | 392.2MB | 335.8MB | **−14.4%** |
| PDF allocs/op | 535,064 | 517,875 | **−3.2%** |
| Template + PDF time | 1.047s | 0.942s | **−10.1%** |
| Template + PDF B/op | 397.6MB | 340.2MB | **−14.4%** |
| Template + PDF allocs/op | 586,355 | 569,123 | **−2.9%** |

B/op and allocs are deterministic for identical code paths, so the −14.4%
B/op cut is a real wave win; wall time on one-shot laptop runs is noisier
(±20% CPU state). Main drivers: single cmap lookup per rune
(`GlyphAdvancePoints`), no duplicate min-content re-measure, in-place
inline-item compaction, ASCII fast paths in TextShow/html scanning,
rune-union dedup in font subsetting, pointer-compare op sorts,
zero-crossing split fast path, `styleGroups` hoist.
The separate locked-gate result is the more stable count-3 median shown
above; the comparison table intentionally uses one iteration on both
snapshots.

### Earlier wkhtmltopdf reference

The performance ledger recorded a process-level run of wkhtmltopdf 0.12.6.1
against the same generated report matrix on the same WSL2 host. The table below
adds the current in-process Go benchmark metrics for context:

| Pages | wkhtmltopdf wall | wk peak RSS | Current Go PDF wall | Current Go B/op | Current Go allocs/op |
|---:|---:|---:|---:|---:|---:|
| 2 | 0.39s* | 34MB | 5.2ms | 3.1MB | 3,222 |
| 5 | 0.23s | 35MB | 7.1ms | 3.8MB | 6,121 |
| 10 | 0.25s | 35MB | 16.3ms | 7.9MB | 11,301 |
| 20 | 0.28s | 37MB | 32.0ms | 14.5MB | 21,500 |
| 50 | 0.37s | 42MB | 72.2ms | 34.7MB | 52,444 |
| 100 | 0.60s | 50MB | 168.0ms | 68.2MB | 104,042 |
| 200 | 0.89s | 66MB | 316.1ms | 135.4MB | 207,496 |
| 250 | 0.99s | 74MB | 430.1ms | 168.5MB | 259,246 |
| 500 | 2.05s | 114MB | 0.873s | 335.8MB | 517,875 |

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
