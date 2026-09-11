# Phase 1 method (PERF3-05)

How to read every number in this folder. Cold and warm are not the same
row. `B/op` is never averaged. Published pins use the generic path only.

## warm vs cold

A conversion is **cold** when it is the first one in a fresh process.
The process still has to load fonts and parse this HTML for the first
time. A conversion is **warm** when an earlier conversion in the same
process already paid that cost.

Row 1 of a warm matrix is always cold. Later sizes in that process are
warm. A standalone 2-then-500 sample is cold at 2 pages and warm at 500
pages. A process that runs only `BenchmarkPDFPages/generic/500Pages` once
is a **cold 500-page** row: there was no smaller size first.

Do not mix a cold 500-page time with Snapshot M's warm 500-page median.
Do not mix a cold 500-page `B/op` with the warm-matrix `B/op`. The
standalone 500-page `B/op` sits a few MB above the warm matrix because
the matrix warms shared caches on 2..250 pages first.

## B/op is never averaged

`B/op` is cumulative allocation traffic for that one sample, not peak
RSS, not `/usr/bin/time %M`. When a capture takes three samples, the
published `B/op` is the raw value from the median-time sample. It is not
the mean of the three `B/op` numbers.

Time may be a median of three `ns/op` values. `B/op` and `allocs/op` stay
the single raw pair that belongs to that median-time sample.

CLI RSS is `/usr/bin/time %M` in KiB. It is not `B/op`. Do not convert
one into the other.

## generic path only

Published pins use `convert.NewPDFRequest` (`internal/convert/convert.go:114-121`).
That is the constructor `Document.WritePDF` and the CLI use
(`document.go:347`, `internal/app/pdf.go:44`).

`convert.NewBenchmarkPDFRequest` sets `benchmarkPageIslands` (`convert.go:123-135`).
That constructor is test-only. No pin in this folder uses it. No pin
turns on `benchmarkPageIslands`.

## pre-change pin on this tree

File: `results/phase-1/pre-change-500p.txt`.

Command:

```
go test ./internal/convert -run '^$' -bench '^BenchmarkPDFPages$/^generic$/^500Pages$' -benchtime=1x -count=1 -benchmem
```

Label: **standalone cold-process 500 pages**. The process ran only the
500-page generic sub-benchmark, so this is the first conversion, not a
warm-matrix 500-page row.

| field | value |
|---|---:|
| time | 671.14 ms (671,143,702 ns/op) |
| B/op | 169,955,488 B |
| allocs/op | 755,255 |
| output-bytes | 1,419,234 |
| pages | 500 |
| path | generic (`NewPDFRequest`) |
| host | 13th Gen Intel Core i7-13700HX, go linux/amd64 |
| date | 2026-09-11 |

## Snapshot M comparison

Snapshot M lives in `testdata/golden/benchmarks/benchmark-results.txt`.
It is the published current capture for this wave. `B/op` there is the
median-time sample's raw value, never averaged. Warm-matrix row 1 is
cold; 500 pages in that matrix is warm.

| row | Snapshot M | this tree pre-change pin |
|---|---|---|
| internal generic PDF, warm 500 pages | 733.48 ms / 163,021,712 B | not this pin (pin is cold, not warm) |
| internal standalone 500 pages (2p cold then 500p warm) | 695.42 ms / 167,865,712 B | closest published sibling; pin is 500p-only cold |
| this pin: 500p-only cold generic | (no Snapshot M row of this shape) | 671.14 ms / 169,955,488 B |

Against Snapshot M standalone median (the closest published 500-page
`B/op`): time 671.14 ms vs 695.42 ms (inside host noise of the faster
window). `B/op` 169,955,488 vs 167,865,712, about +1.2%. The extra
bytes match a cold first conversion that did not see a 2-page warmup.

Against Snapshot M warm-matrix 500 pages: do not compare. That row is
warm and its `B/op` is 163,021,712 B. The pin file already says this.

Same-source faster window named in the ledger: warm 500 pages 576.33 ms
/ 163,032,880 B. That is a warm row. This pin is not that row.

`output-bytes` 1,419,234 and page count 500 match Snapshot M at every
500-page generic row (warm matrix, standalone, CLI).

## later captures

Use the same labels in every header:

- cold or warm
- generic, never islands
- `B/op` from the median-time sample, never averaged
- `output-bytes` and page count as exact integers
