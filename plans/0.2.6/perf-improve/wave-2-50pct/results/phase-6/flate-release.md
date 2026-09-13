# PERF3-30: flate-and-release per page window

Date: 2026-09-11.
Owner: `internal/pdf` only. No convert, layout, or imageout edit. No git
command. `make lint` and full `make test` were out of scope.

## What was wrong

`finalizePages` compressed every page first, then materialized every page
(`internal/pdf/flate_parallel.go` before this change). For a 500-page
document that meant 500 raw content buffers and 500 compressed copies were
reachable at the same time. The compressed slice lived until the last
`finalizePage` ran.

`finalizePage` (`internal/pdf/pdf.go:1106-1127`) already stored the stream
on the object (`setStream` at `pdf.go:290-298`) and dropped the raw builder
(`releaseBuffer` at `internal/pdf/content.go:98`). The extra peak was the
all-pages compressed slice, not the object table.

## What changed

With compression on, `finalizePages` (`internal/pdf/flate_parallel.go:90-106`)
now:

1. Single page or `GOMAXPROCS==1`: `finalizePagesSerialFlate` (`:118-127`)
   flates one page with `flateBytes`, then `finalizePage`. Peak extra
   compressed copy is 1. The retained worker set is never started.
2. Otherwise: `finalizePagesWindowed` (`:133-142`) takes windows of
   `maxPageFlateWorkers` (8, `:14`). Each window is compressed on
   `retainedPageFlate` (`:38`, same process-lifetime workers as PERFT-20),
   then finalized in index order by `finalizeCompressedWindow` (`:148-170`).
   After each `finalizePage`, the window aliases for that slot are cleared
   (`raws[index] = nil`, `streams[index] = nil`) so the raw backing array
   is not kept alive by the window slice.

Compression off is unchanged: `finalizePage` still aliases
`page.content.Bytes()`.

Output order is the page index, never worker completion order. A window of
20 pages (the new test) is 8 + 8 + 4, so the short last window is covered.
Workers are still retained for the process; a per-call flate pool is still
rejected (measured +6.0 MB B/op in PERFT-20).

## Files

| file | sha256 | lines |
|---|---|---|
| `internal/pdf/flate_parallel.go` | `f03b077fb886ec57e86b80e7f9e2d71a23e22659a4d5ae8d9376083c6d5d8ca5` | 170 |
| `internal/pdf/flate_release_test.go` (new) | `a64e39f6acfcfa9686c15477c60d1ab9de902a102da905fc49b3542b3ecb96d1` | 84 |

Existing `*_test.go` files were not edited.

## Tests

Command:

```
go test ./internal/pdf -count=1
```

Result: `ok github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf 7.079s`

Targeted re-run of writer contracts:

```
go test ./internal/pdf -count=1 -v -run 'TestFlateReleaseTwoWritesByteIdentical|TestParallelCompressionDeterministic|TestSerialAndParallelPageStreamsByteIdentical|TestContentLifetime|TestRetainedPoolMatchesSerialFlate|TestDeterministicOutput'
```

Result: `ok ... 0.028s`. All named tests PASS, including:

- `TestFlateReleaseTwoWritesByteIdentical` (new): two independent 20-page
  writes are byte-identical; a second `WriteTo` of the same document is
  identical and does not panic; every page `Content.Bytes` len and cap are
  0 after Write (PDF-06).
- `TestSerialAndParallelPageStreamsByteIdentical` (unchanged): serial
  `GOMAXPROCS=1` path and retained-pool path still match.
- `TestParallelCompressionDeterministic` (unchanged).
- `TestContentLifetimeRawBuffersReleasedAfterWrite` (unchanged).
- `TestContentLifetimeFailedFinalizeIsTerminal` (unchanged).
- `TestDeterministicOutput` (unchanged).

500-page convert `output-bytes` 1,419,234 was not re-measured. This owner
cannot run convert. The writer still flates each page with the same zlib
level and still emits streams in page order, so that envelope is unchanged
from this package. Pin is the existing pdf write tests plus the new
two-write test.

## Not done (later phase-6 rows)

- PERF3-31 CLI 500p RSS
- PERF3-32 overlap layout N+1 with compress N (needs phase 4)
- PERF3-33 warm 500p measure
- PERF3-34 phase `make lint` / `make test` gate
