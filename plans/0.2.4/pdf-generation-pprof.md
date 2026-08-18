# v0.2.4 PDF Generation pprof Report

> **Parent:** [Phase 39 — External Compare Benchmarks](phases/phase-39-external-benchmarks.md)
> **Status:** complete — validated 2026-08-18
> **Scope:** public library PDF generation through `Document.WritePDF`
> **Host:** Linux/amd64, 13th Gen Intel(R) Core(TM) i7-13700HX, Go 1.26.4

## Executive summary

This pass profiles `BenchmarkLibraryPDF/500Pages`, the public benchmark that
expands the shared `report.html.tmpl` fixture before the timer and then calls
`Document.WritePDF`. The benchmark's physical PDF page-count assertion passed,
so this is the library PDF path rather than a pre-rendered-PDF or CLI profile.

The public method is a thin boundary around the engine: `Document.WritePDF`
and its adapter are **86.74% cumulative CPU** in the profile, while the
internal PDF pipeline accounts for the work below it. The largest application
cost is layout construction and table/style processing, followed by painting
and PDF finalization. The profile does not identify the public API mapping as
a material hotspot.

The normal, non-profiled `make bench-lib` Snapshot G baseline for 500 pages is
**939.874 ms/op, 236,734,938 B/op, and 1,149,936 allocs/op**. The instrumented
profile timings are intentionally not performance baselines: CPU profiling
reported 1.700 s/op over three iterations, and exact allocation profiling
reported 9.930 s/op for one iteration.

## Workload and boundary

| Item | Value |
|---|---|
| Benchmark | `BenchmarkLibraryPDF/500Pages` |
| Public entry point | `Document.WritePDF` |
| Fixture | `testdata/golden/benchmarks/templates/report.html.tmpl` |
| Fixture data | 500 pages, 20 invoice rows per page |
| Outside timer | Template expansion and `Document` construction |
| Inside timer | Public-to-engine mapping, loading, layout, painting, PDF encoding |
| Correctness check | `%PDF-` header and exactly 500 physical `/Type /Page` entries |

This boundary matches the public benchmark source in
[`document_bench_test.go`](../../document_bench_test.go). It profiles library
use without starting `cmd/gowkhtmltopdf`.

## Reproduction commands

The raw profiles and test binary were generated under
`/tmp/gowkhtmltopdf-pprof-20260818/`; the durable artifact is this report.

```sh
profile_root=/tmp/gowkhtmltopdf-pprof-20260818
GOCACHE=/tmp/gowkhtmltopdf-go-cache \
  go test -c -o "$profile_root/document.test" .

GOCACHE=/tmp/gowkhtmltopdf-go-cache "$profile_root/document.test" \
  -test.run '^$' \
  -test.bench '^BenchmarkLibraryPDF/500Pages$' \
  -test.benchmem -test.benchtime=5s -test.count=1 \
  -test.cpuprofile="$profile_root/pdf-library.cpu.pprof"

GOCACHE=/tmp/gowkhtmltopdf-go-cache "$profile_root/document.test" \
  -test.run '^$' \
  -test.bench '^BenchmarkLibraryPDF/500Pages$' \
  -test.benchmem -test.benchtime=1x -test.count=1 \
  -test.memprofilerate=1 \
  -test.memprofile="$profile_root/pdf-library.heap.pprof"

go tool pprof -top -nodecount=30 \
  "$profile_root/document.test" "$profile_root/pdf-library.cpu.pprof"
go tool pprof -top -cum -nodecount=30 \
  "$profile_root/document.test" "$profile_root/pdf-library.cpu.pprof"
go tool pprof -top -inuse_space -nodecount=30 \
  "$profile_root/document.test" "$profile_root/pdf-library.heap.pprof"
go tool pprof -top -alloc_space -nodecount=30 \
  "$profile_root/document.test" "$profile_root/pdf-library.heap.pprof"
```

`-test.memprofilerate=1` makes the heap profile exact for the profiled
iteration, but adds substantial instrumentation overhead. The CPU profile
also runs with sampling enabled. Neither profile timing should replace the
checked-in benchmark matrix.

## CPU profile evidence

The CPU profile ran three iterations and collected 11.92 seconds of samples
over a 10.49-second profiled process. The top cumulative path was:

| Function | Cumulative samples | Share |
|---|---:|---:|
| `Document.WritePDF` → `internal/convert.Run` | 10.34 s | 86.74% |
| `pdfPipeline.RenderObjects` | 9.38 s | 78.69% |
| `layout.LayoutContext` | 5.79 s | 48.57% |
| `layout.PaintContext` | 3.25 s | 27.27% |
| `layout.finalizeResult` | 2.88 s | 24.16% |
| `layout.(*engine).build` | 2.87 s | 24.08% |
| `layout.buildTable` | 2.71 s | 22.73% |

The largest flat CPU nodes were:

| Function | Flat samples | Share |
|---|---:|---:|
| `internal/layout.beforeAlways` | 0.82 s | 6.88% |
| `internal/runtime/maps.ctrlGroup.matchH2` | 0.62 s | 5.20% |
| `runtime.scanObject` | 0.50 s | 4.19% |
| `compress/flate.(*compressor).findMatch` | 0.33 s | 2.77% |
| `internal/layout.(*engine).add` | 0.31 s | 2.60% |
| `internal/layout.buildFlowOpIndex` | 0.12 s | 1.01% |
| `internal/layout.validatePaintPageIndices` | 0.12 s | 1.01% |

The actionable signal is therefore layout and display-list construction, with
compression and garbage collection as secondary costs. The public API seam is
not the bottleneck in this workload.

## Heap profile evidence

The exact one-iteration allocation profile recorded 245.47 MB of sampled
allocation space. The leading allocation sites were:

| Function | Flat allocation | Cumulative allocation |
|---|---:|---:|
| `internal/layout.layoutContext` | 77.27 MB | 125.15 MB |
| `bytes.growSlice` | 26.63 MB | 26.63 MB |
| `internal/layout.(*engine).buildCell` | 17.62 MB | 17.62 MB |
| `internal/layout.collectTableBorderSegments` | 13.64 MB | 19.94 MB |
| `internal/layout.buildFlowOpIndex` | 8.30 MB | 8.30 MB |
| `internal/html.openElement` | 6.76 MB | 9.65 MB |
| `internal/layout.applyCascadeDeclaration` | 6.50 MB | 6.50 MB |
| `internal/layout.(*engine).splitTextByFace` | 6.47 MB | 6.47 MB |
| `internal/layout.collectBorderSegmentOps` | 6.30 MB | 6.30 MB |
| `bytes.Clone` | 6.03 MB | 6.03 MB |

The retained heap snapshot was only 5.25 MB at the capture point. Its largest
retained objects were `bytes.Clone` (3.07 MB flat, 58.48%), the Flate writer
(1.58 MB cumulative, 29.99%), and default-font loading (3.49 MB cumulative,
66.43%). Retained heap and per-operation allocation space measure different
things; the small retained snapshot does not contradict the 236.7 MB/op
allocation baseline.

## Findings and follow-up boundary

- The dominant optimization surface is internal layout construction, notably
  table cells, style resolution, flow-operation indexes, and border segments.
- PDF finalization and Flate compression are measurable, but they are below the
  layout phase in cumulative CPU for this report fixture.
- The public `Document` API adds the intended adapter boundary without creating
  a material profile hotspot.
- This report is diagnostic evidence only. Any optimization should rerun the
  same page-count assertion and the visual/correctness gates before changing a
  benchmark baseline.

## Validation record

- The compiled benchmark binary passed the CPU profile run and the exact heap
  profile run.
- `go tool pprof` successfully produced flat CPU, cumulative CPU, retained
  heap, and allocated heap summaries from the generated profiles.
- The benchmark's `%PDF-` and 500-page physical page-count checks passed during
  both profile runs.
- The Phase 39 checklist records this report as completed evidence; no open or
  partial checklist marker was introduced.
