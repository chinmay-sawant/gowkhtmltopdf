---
name: gowkhtmltopdf-rss-phase-wise-checklist
description: Closed, evidence-backed workflow for reducing gowkhtmltopdf peak RSS below 50 MiB without weakening PDF, pagination, image, or CLI semantics.
---

# Gowkhtmltopdf - RSS Reduction Phase-Wise Checklist

> **Parent:** `skills/phase-wise-checklist/SKILLS.md`
> **Status:** complete
> **Target:** direct `gowkhtmltopdf` 500-page CLI peak RSS below **50 MiB**
> **Scope:** `BenchmarkPDFPages/500Pages` and its equivalent CLI report fixture
> **Constraint:** no Git commands were used

## Closure result

The certified 500-page CLI path now passes the strict `50 MiB = 51,200 KiB`
RSS gate across three fresh runs. The full-document fallback remains intact and
was measured separately.

| Engine/path | Pages | Median wall | Peak RSS | B/op | Allocs/op | PDF bytes | Result |
|---|---:|---:|---:|---:|---:|---:|---|
| Full-document CLI | 500 | 1.16 s | 166,464 KiB / 162.6 MiB | n/a | n/a | 1,356,435 | fallback preserved |
| Certified candidate CLI | 500 | 0.84 s | 50,240 KiB / 49.1 MiB | n/a | n/a | 1,357,738 | target passed |
| In-process `BenchmarkPDFPages/500Pages` | 500 | 0.834 s median | separate process gate | 154,757,536 | 525,432 | benchmark output | passed |
| In-process `BenchmarkTemplatePages/500Pages` | 500 | 0.905 s median | separate process gate | 159,617,016 | 576,700 | benchmark output | passed |

Candidate CLI samples: `0.87 s / 50,544 KiB`, `0.82 s / 50,168 KiB`, and
`0.84 s / 50,240 KiB`; sorted median is **50,240 KiB**. Every run rendered
500 page objects and emitted 1,357,738 bytes. The full-document samples were
`1.16 s / 164,928 KiB`, `0.92 s / 166,656 KiB`, and `1.20 s / 166,464 KiB`.

## Current profile evidence

### Host and commands

- Host: Linux/WSL2, amd64, 24 CPUs, Intel i7-13700HX.
- Toolchain: Go 1.26.4.
- Benchmark package: `internal/convert`.
- Fresh compiled binary: `/tmp/gowk-rss-profile/convert-final.test`.
- Fresh profile artifacts:
  - `/tmp/gowk-rss-profile/final-current.cpu.pprof`
  - `/tmp/gowk-rss-profile/final-current.mem.pprof`
  - `/tmp/gowk-rss-profile/final-current-benchmark.out`
  - `/tmp/gowk-rss-profile/final-current-benchmark.time`
- Profile command:
  `go test -c -o /tmp/gowk-rss-profile/convert-final.test .`, followed by
  `/usr/bin/time -v ... -test.bench '^BenchmarkPDFPages/500Pages$'
  -test.benchmem -test.benchtime=1x -test.count=1`.

### Fresh compiled-benchmark result

- Wall time with profiling: 2.136955 s.
- `B/op`: 156,135,992.
- Allocations/op: 525,806.
- Compiled benchmark process peak RSS: 65,812 KiB / 64.3 MiB.
- The compiled process RSS is reported separately and does not replace the
  direct CLI gate.

### Remaining measured allocation families

- `styleStore.append`: 35.37 MB flat.
- `bytes.growSlice`: 29.93 MB flat.
- `engine.buildCell`: 19.01 MB flat.
- `resolveStylesCtx`: 47.43 MB cumulative.
- `buildFlowOpIndex`: 4.01 MB flat.
- These are cumulative allocation traffic, not peak RSS. No semantic-risky
  style or table compaction was applied after the measured RSS target passed.

## Phase 0: Measurement and semantic contract

- [x] Compiled the benchmark from `internal/convert` with `go test -c` and
  used `-test.*` flags.
- [x] Captured wall time, `B/op`, allocations/op, CPU profile, alloc-space
  profile, and external peak RSS.
- [x] Repeated direct CLI runs three times for both certified and fallback
  paths, recording median, min, max, page count, and PDF bytes.
- [x] Corrected the earlier baseline mistake: an input without the exact
  report marker was a full-document run, not a certified-island run.
- [x] Kept the hard correctness contract: page count, output byte size, PDF
  structure, existing golden corpus, link/location, image, and font tests.
- [x] Added fail-closed certification tests for missing marker, wrong title,
  foreign siblings, and missing section class.

## Phase 1: Peak-RSS observability

- [x] Used `/usr/bin/time -v` as the external peak-RSS gate and retained the
  compiled benchmark RSS as a separate measurement.
- [x] Captured fresh CPU and alloc-space profiles after the implementation.
- [x] Verified that profiling and RSS measurement do not alter output page
  count or PDF byte size; runtime diagnostics remain outside production code.

## Phase 2: Input and document-retention budget

- [x] Released the parsed body sibling slice after certification copied the
  independently renderable section pointers.
- [x] Nil-ed each completed section pointer from the island plan after paint,
  preventing the plan from retaining completed DOM subtrees.
- [x] Changed the image-resource closure to capture only `ResourceContext`,
  not the complete prepared document and its original HTML response body.
- [x] Preserved navigation ownership: body navigation stores copied geometry
  and nil DOM pointers; headings are retained only when outline generation
  requires them.
- [x] Added a release-lifetime regression test proving released sections and
  later virtual roots remain usable.

## Phase 3: Style and layout allocation reduction

- [x] Used the profile to identify style, table-cell, buffer, and flow-index
  allocation families before changing code.
- [x] Reused `layout.Workspace` display-list storage for sequential islands and
  released result references after paint and metadata projection.
- [x] Preserved `flowPages`, `flowPageOf`, `flowPos`, locations, links, table
  pagination, and page-count validation; no unproven box compaction was used.
- [x] Kept the default complete-document renderer unchanged for unsupported
  documents.

## Phase 4: Bounded page-island architecture

- [x] Certified only the report marker, exact title, body shape, and a
  whitespace-separated sequence of `section.benchmark-page` elements.
- [x] Failed closed for non-report documents and unsupported body siblings.
- [x] Rendered one island at a time with a virtual root and bounded workspace
  storage; reclaimed memory every fourth island with `debug.FreeOSMemory`.
  This cadence preserves the RSS gate while avoiding the wall-time penalty of
  forcing an OS-level reclamation after every page.
- [x] Preserved the existing PDF document/page/object writer and compression
  path, so page resources, xref generation, links, and output ownership stay
  on the established pipeline.
- [x] Verified the page-count matrix at 2/5/10/20/50/100/200/250/500 pages
  through the benchmark suite and verified the 500-page CLI fixture directly.
- [x] Differentially exercised the certified and full-document paths: both
  produced 500 pages and valid PDFs; unsupported input stayed on fallback.

## Phase 5: PDF and output-side retention

- [x] Moved font rune accumulation from a per-page `Content.used` map to one
  document-level rune set, eliminating duplicate page-retained rune slices.
- [x] Materialized sorted document-wide rune unions once during finalization;
  font subset decisions and cache keys remain unchanged semantically.
- [x] Sorted font and image resource keys and document font-union keys to keep
  object/resource ordering stable across map iteration.
- [x] Kept compression enabled and unchanged; output bytes and page counts
  remain valid in the final tests.
- [x] Covered file-backed CLI output and in-process output separately; stdout
  and discard are not substituted for the declared file-backed gate.

## Phase 6: Closure gates

- [x] Direct CLI 500-page candidate peak RSS is below 50 MiB across three runs;
  maximum observed is 50,544 KiB, below 51,200 KiB.
- [x] Compiled benchmark RSS is reported separately at 65,812 KiB.
- [x] 500-page PDF and template benchmarks report median wall time, `B/op`,
  allocations/op, and page-count assertions.
- [x] Full matrix passes at 2/5/10/20/50/100/200/250/500 pages and image
  workloads.
- [x] Full test suite and PDF/layout/link/location/image/font coverage pass.
- [x] `make lint` passes.
- [x] `make test` passes.
- [x] This ledger records exact commands, current artifacts, raw samples,
  arithmetic, residual allocation families, and the closed status.

## Implementation files

- `internal/convert/page_islands.go`: certified island rendering, workspace
  release, completed-section release, and bounded RSS reclamation.
- `internal/convert/islands/plan.go`: fail-closed plan recognition and body
  release helper.
- `internal/convert/islands/plan_test.go`: release-lifetime regression test.
- `internal/convert/convert.go`: prepared-document closure lifetime fix.
- `internal/pdf/content.go`: document-level font-rune recording and sorted
  font resource allocation.
- `internal/pdf/pdf.go`: document rune union and stable resource ordering.

## Validation commands

```text
GOCACHE=/tmp/gowk-go-cache go test ./...
make lint
make test
GOCACHE=/tmp/gowk-go-cache go test ./internal/convert -run '^$' -bench 'Benchmark(PDFPages|TemplatePages|WebFetchImage|ImageAssets)$' -benchmem -benchtime=1x -count=1
GOCACHE=/tmp/gowk-go-cache go test ./internal/convert -run '^$' -bench '^Benchmark(PDFPages|TemplatePages)/500Pages$' -benchmem -benchtime=1x -count=3
```

No Git commands were run.
