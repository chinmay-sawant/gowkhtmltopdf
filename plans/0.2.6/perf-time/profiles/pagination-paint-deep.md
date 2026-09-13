# Post-layout deep profile: pagination, paint, writer, GC

Read-only profiling for the 0.2.6 perf-time plan. No production code, test, or
checklist file was changed. Scratch under `/tmp/opencode/perf-time-pag/`;
raw evidence under `plans/0.2.6/perf-time/profiles/raw/pagination-paint/`.

Task: profile the committed post-perf-improve baseline, prepare a plan that
halves warm render time while keeping B/op at or below current values. This
file answers what the post-layout group costs, what a 2x plan can get from it,
and what must come from other groups.

## 0. Capture header and method

- date: 2026-09-11, 19:13 to 19:17 IST (13:43 to 13:44 UTC for the heavy
  capture).
- tree: committed working tree at session start, branch `chore/review-026`,
  `VERSION` 0.2.5. No git command ran.
- go: go1.26.4 linux/amd64. Host: 13th Gen Intel Core i7-13700HX, 24 CPUs,
  7.6 GiB RAM, Linux 6.6.87.2-microsoft-standard-WSL2.
- heavy-profile lock: `mkdir /tmp/opencode/perf-time-profile.lock` failed on
  first try (held by another agent). Waited 240 s (13:39:25 to 13:43:25),
  acquired, released at 13:44:00. Recorded in `raw/pagination-paint/capture-log.txt`.
- canonical binary: `go test -c -o /tmp/opencode/perf-time-pag/bin/convert.test
  ./internal/convert`, sha256
  `b40b8cdd8810ea211f3f51e98b6d496916eee346db64bd5a59d7d3f6aa3a2d46`.
- workload: `BenchmarkPDFPages/generic` from `internal/convert`
  (`internal/convert/benchmarks_test.go:409`), fixture
  `testdata/golden/benchmarks/templates/report.html.tmpl`, sha256
  `e3b5387bed117844c3502915cd3def253738959feb894435720f3532746e6974`.
- 500 pages: `-test.bench '^BenchmarkPDFPages$/^generic$/^500Pages$'
  -test.benchtime=1x -test.count=5` plus `-test.cpuprofile`. Rows:
  1785.6 (cold) / 1439.2 / 1439.6 / 1389.2 / 1404.7 ms. Profile duration
  7.56 s, total samples 8.000 s.
- 50 pages: same with `^50Pages$` and `-test.count=25`. Rows: 131.7 cold,
  then 117.3, 116.3, 113.7, 108.8, 107.6, 113.6, 110.9, 110.6, 108.5, 105.2,
  110.6, 104.7, 108.7, 114.5, 114.8, 112.3, 108.8, 116.3, 113.2, 115.0,
  108.6, 116.5, 109.2, 107.0 ms. Warm median 110.9 ms. Total samples 3.060 s.
- B/op cross-check at `-memprofilerate=1`, 2 pages cold then 500 warm in one
  process: `234,995,968 B/op`, 1,225,362 allocs/op, `output-bytes 1419234`,
  500 pages. That is +0.03 percent against the phase-7 committed row
  (234,923,560 B), so the allocation baseline is the same tree.
- baseline context: phase-7 warm matrix is 1,228.72 ms / 234.92 MB
  (`plans/0.2.6/perf-improve/results/phase-7/final-capture.md`).
- per-iteration conversions in this file: for 500 pages, `cum / 5`; for
  50 pages, `cum / 25`. Single-threaded phases spend CPU on one goroutine, so
  CPU seconds are a fair stand-in for wall milliseconds. The profile run has
  4 warm rows and 1 cold row at 500 pages, so every per-iteration number is
  slightly cold-weighted (cold row is 24 percent slower).
- profile shares are of total samples. `runtime.gcBgMarkWorker` runs on its own
  stacks and is disjoint from phase stacks. Allocator primitives are charged
  inside phase stacks, so they are inside the cumulative numbers below.

## 1. Top-line phase attribution

500 pages, 8.000 s samples (5 iterations). "phase CPU" is the flat sample sum
of every stack through the phase root (`pprof -show_from`, all nodes). "cum"
is the function's cumulative row. "ms/iter" is cum / 5.

| phase (root) | phase CPU | share | cum | ms/iter |
|---|---:|---:|---:|---:|
| `layoutContext` (layout) | 4.47 s | 55.88% | 4.47 s | 894 |
| `resolveStylesCtx` (style) | 2.50 s | 31.25% | 2.50 s | 500 |
| `(*engine).build` (box + display list) | 1.82 s | 22.75% | 1.82 s | 364 |
| `PaintContext` (pagination + paint) | 1.88 s | 23.50% | 1.88 s | 376 |
| `paginateOps` chain | 0.76 s | 9.50% | 0.76 s | 152 |
| `capTablePageBreaks` | 0.19 s | 2.38% | 0.19 s | 38 |
| `paintPages` | 0.57 s | 7.12% | 0.57 s | 114 |
| `buildStructureTree` | 0 s | 0% | 0 s | 0 |
| `populateLocations` | 0 s | 0% | 0 s | 0 |
| PDF `finalize` / `writeTo` | 0.78 s | 9.75% | 0.78 s | 156 |
| `flateBytes` | 0.76 s | 9.50% | 0.76 s | 152 |
| `runtime.gcBgMarkWorker` | 0.39 s | 4.88% | 0.39 s | 78 |
| allocator primitives (inside phases) | 0.51 s | 6.4% | n/a | 102 |
| `convert.Run` | 7.43 s | 92.88% | 7.43 s | 1,486 |

`buildStructureTree` is called once per conversion and returns at
`internal/layout/tagging.go:30` because the generic benchmark document is not
PDF/UA. `populateLocations` is called once (`internal/layout/paint.go:214`) and
records no samples, but allocates 2.63 MB (see section 3.3).

50 pages, 3.060 s samples (25 iterations, all but the first warm):

| phase (root) | phase CPU | share | ms/iter |
|---|---:|---:|---:|
| `layoutContext` | 1.91 s | 62.42% | 76.4 |
| `resolveStylesCtx` | 1.07 s | 34.97% | 42.8 |
| `(*engine).build` | 0.77 s | 25.16% | 30.8 |
| `PaintContext` | 0.47 s | 15.36% | 18.8 |
| `paginateOps` | 0.12 s | 3.92% | 4.8 |
| `capTablePageBreaks` | 0.03 s | 0.98% | 1.2 |
| `paintPages` | 0.23 s | 7.52% | 9.2 |
| PDF `finalize` | 0.31 s | 10.13% | 12.4 |
| `flateBytes` | 0.30 s | 9.80% | 12.0 |
| `runtime.gcBgMarkWorker` | 0.15 s | 4.90% | 6.0 |
| `convert.Run` | 2.80 s | 91.50% | 112 |

Compression and the writer are size-linear (9.5 percent of samples at 500
pages, 9.8 percent at 50 pages). Pagination scales worse than linearly at the
large size (3.9 to 9.5 percent), mostly the linear range scan in section 2.

## 2. Function and source-line attribution (500 pages)

`cum` is the pprof cumulative row; `CPU` is the subtree flat sum; both from the
same profile so they are comparable. Source lines are the committed repo files
(the canonical binary was built from them, so pprof maps to repo line numbers).

### 2.1 The `paginateOps` chain

`paginateOps` (`internal/layout/paint_pagination_fixpoint.go:22`) cum 760 ms
(9.50 percent), subtree CPU 0.76 s. Own-line and callee breakdown from
`pprof -list`:

| source line | cum | ms/iter | note |
|---|---:|---:|---|
| `paint_pagination_fixpoint.go:27` `ensureFlowIndex` | 70 ms | 14 | first page + box index build |
| `paint_pagination_fixpoint.go:32` `settleBeforeAlways` (first) | 180 ms | 36 | see `beforeAlways` below |
| `paint_pagination_fixpoint.go:44` `keepImplicitAsides` loop | 0 | 0 | 1 iteration, no change (counter) |
| `paint_pagination_fixpoint.go:49` `snapCrossingTextOps` | 150 ms | 30 | 174,000 ops scanned |
| `paint_pagination_fixpoint.go:53` `paginationFixpoint` | 170 ms | 34 | 1 iteration, no changes |
| `paint_pagination_fixpoint.go:60` `repeatTableHeaders` | 60 ms | 12 | |
| `paint_pagination_fixpoint.go:64` `normalizeTableRowGaps` | 110 ms | 22 | 20 + 90 ms in the two `rowPaintBand` calls |
| `paint_pagination_fixpoint.go:67` `normalizeLeadingRoundedCallouts` | 10 ms | 2 | |
| `paint_pagination_fixpoint.go:70` `settleBeforeAlways` (second) | 10 ms | 2 | 1 pass, no change |

`settleBeforeAlways` (`paint_pagination_fixpoint.go:80`) cum 190 ms:
`beforeAlways` (`paint_flow_breaks.go:499`) cum 200 ms includes both settle
calls and the fixpoint call. Its lines:

| source line | cum | ms/iter | note |
|---|---:|---:|---|
| `paint_flow_breaks.go:506` `collectBeforeAlwaysTargets` | 30 ms | 6 | walks all 54,503 boxes |
| `paint_flow_breaks.go:529` `beforeAlwaysBatch` | 70 ms | 14 | 4 calls, 1,996 targets, 499 changed |
| `paint_flow_breaks.go:547` `applySuffixDifferences` | 10 ms | 2 | |
| `paint_flow_breaks.go:549` `ensureFlowIndex` rebuild | 90 ms | 18 | invalidate + full rebuild after a shift |

`paginationFixpoint` (`paint_pagination_fixpoint.go:98`) cum 170 ms. One
iteration, all six policies reported no change (section 4 counters), so these
are no-op walks:

| source line | cum | ms/iter |
|---|---:|---:|
| `paint_pagination_fixpoint.go:104` `avoidInside` (`paint_flow_breaks.go:36`) | 60 ms | 12 |
| `paint_pagination_fixpoint.go:105` `beforeAlways` | 10 ms | 2 |
| `paint_pagination_fixpoint.go:109` `afterBreaks` (`paint_flow_breaks.go:930`) | 70 ms | 14 |
| `paint_pagination_fixpoint.go:113` `rowsIntact` (`paint_flow_tables.go:10`) | 10 ms | 2 |
| `paint_pagination_fixpoint.go:117` `keepHeadingWithNext` | 0 | 0 |
| `paint_pagination_fixpoint.go:121` `orphansWidows` (`paint_flow_orphans.go:14`) | 20 ms | 4 |

`snapCrossingTextOps` (`paint_pagination_fixpoint.go:135`) cum 150 ms. Its cost
is almost entirely `opInPaintRange`
(`paint_pagination_fixpoint.go:189`): 120 ms flat, called 174,000 times per
conversion, and each call walks up to 500 table ranges: 87,000,000 span checks
(counters). `tablePaintRanges` (`:137`) is 10 ms. The snap switch below `:149`
records samples only through its callees.

`keepImplicitAsides`, `applyStickyPrint`, `buildStructureTree`, and
`populateLocations` have zero samples on this workload. The sticky pass is called
once and finds zero sticky boxes (counter). The structure pass returns at
`tagging.go:30` because `doc.IsUA()` is false.

### 2.2 Seals, paint, locations

`capTablePageBreaks` (`paint_pagination_seal.go:33`) cum 190 ms (2.38 percent),
38 ms/iter:

| source line | cum | ms/iter | note |
|---|---:|---:|---|
| `paint_pagination_seal.go:38` `capTableMaxPage` | 10 ms | 2 | full op scan |
| `paint_pagination_seal.go:45` `collectTableBorderSegments` | 110 ms | 22 | `collectBorderSegmentOps` two passes (count at `:337`, fill at `:342`), 63,000 verts + 55,000 horiz, then three map fills at `:320`, `:321`, `:326` |
| `paint_pagination_seal.go:62` `clusterVerticals` | 20 ms | 4 | map allocation at `:410` is 2.63 MB |
| `paint_pagination_seal.go:71` `sealPageTopClusters` | 10 ms | 2 | |
| `paint_pagination_seal.go:75` `sealPageBottomClusters` | 40 ms | 8 | |

`paintPages` (`paint.go:357`) cum 570 ms (7.12 percent), 114 ms/iter:

| source line | cum | ms/iter | note |
|---|---:|---:|---|
| `paint.go:406` `child.Grow(contentSizeHint(...))` | 100 ms | 20 | `Content.Grow` (`internal/pdf/content.go:89`); 20.56 MB allocated |
| `paint.go:409` `sortPaintIndices` | 50 ms | 10 | per-page index sort |
| `paint.go:429` `painter.paintOp(&res.Ops[idx])` | 400 ms | 80 | `drawPageOp`/`paintWrappedOp`; `drawLine` (`paint.go:1253`) 230 ms cum |
| `paint.go:395` loop over `res.Pages` | 0 | 0 | 500 pages, 174,000 ops painted (counter) |

`validatePaintPageIndices` (`paint.go:310`) cum 70 ms (14 ms/iter). It is
called three times (`paint.go:165`, `:184`, `:207`), 522,000 op checks per
conversion (counter). `splitCrossingRects`
(`paint_pagination_split.go:15`) cum 20 ms; the counter shows 0 crossing ops in
this fixture, so only the counting scan runs. `stripOrphanRowChrome`,
`stretchPaginatedChrome` (3 calls), and `fixedOpIndices` (0 fixed ops) record
no meaningful samples. `populateLocations` allocates 2.63 MB at
`paint.go:835` and records no samples.

### 2.3 PDF writer

`finalize` (`internal/pdf/pdf.go:818`) cum 770 ms (9.62 percent), 154 ms/iter.
`finalizePage` (`pdf.go:1105`) cum 770 ms; its line 1108 `flateBytes` is 760 ms
(152 ms/iter). `unionFontRunes` (`pdf.go:913`) records no samples.
`writeTo` (`pdf.go:725`) cum 780 ms, so serialization after finalize is about
10 ms total; `writePDFObject` (`pdf.go:609`) is 10 ms cum (2 ms/iter).
`computeTrailerID` does not run: the generic benchmark uses PDF 1.4, whose
trailer branch (`pdf.go:712`) writes no `/ID`.

`flateBytes` (`pdf.go:1641`) cum 760 ms, line breakdown:

| source line | cum | ms/iter |
|---|---:|---:|
| `pdf.go:1648` `state.zw.Reset` | 30 ms | 6 |
| `pdf.go:1651` `zw.Write(raw)` | 60 ms | 12 |
| `pdf.go:1652` `zw.Close()` (deflate) | 640 ms | 128 |
| `pdf.go:1654` result copy `append([]byte(nil), ...)` | 30 ms | 6 |

### 2.4 50-page scaling of the same functions

Per iteration (25 iterations): `paginateOps` 4.8 ms, `capTablePageBreaks`
1.2 ms, `paintPages` 9.2 ms, `finalizePage` 12.4 ms, `flate` 12.0 ms,
`opInPaintRange` does not appear separately (it is inside `paginateOps`, whose
share falls from 9.5 percent at 500 pages to 3.9 percent at 50 pages).
`normalizeTableRowGaps` is 30 ms over 25 iterations (1.2 ms/iter) against
22 ms/iter at 500 pages, so its cost scales with table rows, not pages.

## 3. GC and retention

### 3.1 gctrace, 500 pages, count 5

Raw: `raw/pagination-paint/500-gctrace.txt`, parsed: `raw/pagination-paint/500-gctrace-parsed.txt`.

- 36 cycles: 28 normal plus 8 forced. Four forced cycles are harness setup
  points at tiny heaps (1 to 5 MB); four sit at iteration boundaries where the
  heap drops 212->6, 30->5, 50->6, and 168->6 MB. Those boundary GCs are the
  forced iteration boundaries, not warm inner cycles.
- Normal cycles: mark+scan CPU 847.3 ms, sweep 34.7 ms, STW 6.3 ms. Per
  iteration: 169 ms mark+scan, 7 ms sweep, 1.3 ms STW.
- Peak heap during a cycle 210 MB; maximum live-after-cycle 155 MB. A
  representative warm line: `gc 36 @6.93s 0%: 0.123+6.7+0.32 ms clock,
  2.7+0.1/35/65+0.32 ms cpu, 204->204->155 MB`.
- 50 pages: 58 cycles (30 normal, 28 forced, one forced boundary per
  iteration), 187.0 ms mark+scan over the normal cycles = 7.5 ms/iter, peak
  19 MB, live 16 MB; each boundary forced GC drops 27 MB to 5 MB
  (`raw/pagination-paint/50-gctrace-parsed.txt`).

The CPU profile agrees on the mutator side: `gcBgMarkWorker` 0.39 s
(78 ms/iter, 4.88 percent), allocator primitives 0.51 s (102 ms/iter,
6.4 percent). GC plus allocation machinery is 11.3 percent of samples at
500 pages and 9.7 percent at 50 pages. The gctrace CPU total (176 ms/iter)
overstates wall cost because mark workers run in parallel; the profile's
78 ms/iter is on-CPU mark time, and assists are charged to mutator stacks.

### 3.2 Phase-boundary live and peak heap, 500 pages

Probe: copy of the tree at `/tmp/opencode/perf-time-pag/mod` with a phase
boundary hook (`internal/convert/zz_perf_time_probe.go`, copy only). Each
point forces `runtime.GC()` and reads `MemStats`. Output:
`raw/pagination-paint/probe-500-retention.txt`. A 2-page document warms fonts first; the 500
run is measured in the same process.

| point | heap now (MB) | live after GC (MB) | total alloc (MB) | GCs so far |
|---|---:|---:|---:|---:|
| render start | 5.3 | 3.3 | 12.3 | 9 |
| after Layout | 149.2 | 131.0 | 178.3 | 14 |
| after Paint | 187.9 | 156.3 | 235.1 | 15 |
| after Assemble | 161.6 | 43.7 | 240.4 | 16 |
| before Write | 43.7 | 43.7 | 240.4 | 17 |
| after Write (inside Finalize) | 51.1 | 27.6 | 247.8 | 18 |
| after Run returns | 27.6 | 5.4 | 247.7 | 19 |

What the deltas say, measured:

- Layout allocates 166.0 MB and retains 131.0 MB live at its end. That live set
  is the display list (174,000 ops, 440 bytes each after phase 5, about
  76.6 MB used of the 89.5 MB preallocation), the box tree (54,503 boxes), the
  style store, and locations.
- Paint allocates 56.8 MB (the content buffers dominate, 20.56 MB charged at
  `pdf/content.go:89` via `paint.go:406`) and leaves 156.3 MB live.
- Assemble allocates 5.3 MB, but by its forced GC the Layout result is
  garbage: live falls from 156.3 MB to 43.7 MB. The 112.6 MB released is the
  op list, boxes, and locations. The retained 43.7 MB is the PDF document
  (500 content buffers plus fonts and page structures).
- Write allocates 7.4 MB and releases content buffers page by page in
  `finalizePage` (`pdf.go:1128`). At the Finalize frame the document and
  runContext are still reachable: 27.6 MB live. Once `Run` returns and the
  frame unwinds, a forced GC leaves 5.4 MB live, which is the caller-held
  request/output and process-level font state. The whole conversion leaves
  almost nothing behind.
- Nine GC cycles run by the end of Write (GC count 9 to 18); ten by the time
  `Run` returns. Because Layout and Paint hold 130 to 156 MB live while
  allocating 223 MB, most cycles run with a large pointer-rich live set.

### 3.3 Retained structures that force scanning

The display list is the dominant scan surface: 174,000 `Op` values with nine
pointer words each (Text, Font, TextTransform, FontFeatures, URI, Image, Alt,
BlendMode, StructElem; see `plans/0.2.6/perf-improve/designs/phase-5-display-list.md`
section 1). For the generic template `StructElem` is nil everywhere and
`Image`, `URI`, `BlendMode`, `TextTransform`, and `FontFeatures` are mostly
empty, but the GC still walks every slot. This structure is layout-owned and
must stay live through Paint, so this group cannot remove it; it can only stop
re-walking it. `scanObject` is 0.21 s (42 ms/iter) and `scanObjectsSmall`
0.13 s (26 ms/iter) in the profile.

Structures this group owns that no longer serve a purpose at the point they
are retained:

- `Result.flowStore` scratch (pageOf/pos/counts/pages) is released by
  `releaseFlowIndex(res)` at `paint.go:245`, after `paintPages`. The last
  pagination reader (`normalizeTableRowGaps`, which reads `flowPages` directly)
  runs earlier, and a later reader rebuilds the index on demand by design
  (`internal/layout/paint_release_test.go:22-56` pins that contract). Whether
  any paint-path helper reads the index is not established here, so a plan
  option to release it before `paintPages` needs its own probe, not a profile
  claim.
- Per-build scratch churn is real: `resetIntBuffer` 6.50 MB
  (`paint_flow_index.go:77`), `sizePageBuckets` 5.56 MB
  (`paint_flow_index.go:111`), `newBreakScanState` 5.60 MB
  (`paint_flow_breaks.go:390`, 4 states), `collectBorderSegmentOps` 6.64 MB
  (`paint_pagination_seal.go:339-340`), `collectTableBorderSegments` map
  appends 5.74 MB (`:320`, `:321`, `:326`), `clusterVerticals` 2.63 MB
  (`:410`), `populateLocations` 2.63 MB (`paint.go:835`), `flateBytes` result
  copies 1.17 MB plus `flate.NewWriter` 1.33 MB. That is about 38 MB of
  scratch outside content buffers. Adding the 20.56 MB content-buffer `Grow`
  charged through `paint.go:406`, this group owns about 58 MB of the 235 MB
  B/op (25 percent).
- The `fontRuneSet` map is consumed by `unionFontRunes` and then replaced; it
  holds at most a few hundred runes, not a scan driver.
- `objectState.navigation` copies intent data out of the Layout result
  (`internal/convert/outline.go:71-73`), and `collectBodyNavigation`
  (`links.go:97`) ranges over all 174,000 ops once (70 ms flat, 14 ms/iter).
  It does not retain the op list.

Allocation attribution at 500 pages (rate-1 profile, profile total 257.4 MB
including the 2-page warmup and template execution; warm 500 B/op is
234.996 MB). Layout-owned sites: ops preallocation 89.50 MB
(`internal/layout/layout.go:1028`), `buildCell` 16.87 MB, `applyRestProps`
10.27 MB, `html.openElement` 7.12 MB, `splitTextByFace` 6.81 MB,
`appendTextToken` 6.60 MB, `resolveStylesCtx` 4.76 MB. This group owns about
58 MB of the 235 MB (25 percent). The GC mark attributable to that allocation
is about 25 percent of the 78 ms/iter mark, or about 20 ms; the allocator
primitives for those bytes are already inside the phase cumulative numbers
above.

## 4. Pagination fixpoint: iterations, calls, architecture

Counters were added to a copy of the tree only
(`internal/layout/zz_perf_time_counters.go` plus one to three increment lines
in the pagination functions). The counter build produces the same 500-page PDF
bytes (`1,419,234`) and the same 500 pages as the canonical binary. Raw:
`raw/pagination-paint/probe-500-counters.txt`, `raw/pagination-paint/probe-50-counters.txt`.

| counter | 500 pages | 50 pages |
|---|---:|---:|
| `PaintContext` calls | 1 | 1 |
| `validatePaintPageIndices` calls / op checks | 3 / 522,000 | 3 / 52,200 |
| `paginateOps` calls | 1 | 1 |
| `settleBeforeAlways` calls / passes / changed passes | 2 / 3 / 1 | 2 / 3 / 1 |
| `keepImplicitAsides` loop iterations / changed | 1 / 0 | 1 / 0 |
| `beforeAlways` calls / targets / changed / box slots scanned | 4 / 1,996 / 499 / 218,012 | 4 / 196 / 49 / 21,812 |
| `snapCrossingTextOps` calls / ops scanned | 1 / 174,000 | 1 / 17,400 |
| `opInPaintRange` calls / span checks | 174,000 / 87,000,000 | 17,400 / 870,000 |
| `paginationFixpoint` iterations / policy calls | 1 / 6 | 1 / 6 |
| fixpoint policy changes (avoid, before, after, rows, head, orphan) | 0,0,0,0,0,0 | 0,0,0,0,0,0 |
| `repeatTableHeaders` / `normalizeTableRowGaps` calls | 1 / 1 | 1 / 1 |
| `splitCrossingRects` calls / crossings / fragments | 1 / 0 / 0 | 1 / 0 / 0 |
| `capTablePageBreaks` calls / ops | 1 / 174,000 | 1 / 17,400 |
| `stripOrphanRowChrome` / `stretchPaginatedChrome` calls | 1 / 3 | 1 / 3 |
| `applyStickyPrint` calls / sticky boxes | 1 / 0 | 1 / 0 |
| `buildStructureTree` calls / skipped non-UA | 1 / 1 | 1 / 1 |
| `populateLocations` calls | 1 | 1 |
| `paintPages` calls / pages / ops painted | 1 / 500 / 174,000 | 1 / 50 / 17,400 |
| `fixedOpIndices` calls / fixed ops | 1 / 0 | 1 / 0 |

Architecture reading, measured:

1. The fixpoint never iterates on this fixture. It runs the six policies once
   and every policy reports no change. There is no repeated pass to remove by
   a single-pass page-assignment architecture; the current code already makes
   one effective pass for this document.
2. What the fixpoint removes if it is made cheaper is six no-op walks worth
   150 ms/5 = 30 ms/iter (avoid 12, after 14, orphan 4). A cheap, exact census
   per `Result` (does any box have `PageBreakInside: avoid`, `PageBreakAfter:
   always`, or non-default `orphans`/`widows`?) can skip those walks in one
   pass over boxes. That preserves placement exactly because the skipped
   policies can provably return false.
3. A precomputed break set cannot be cheaper here: `beforeAlways` does real
   work once (499 shifts on 1,996 targets), costs 40 ms/iter including the
   18 ms index rebuild, and any precomputation would still have to reproduce
   the batch's floating-point event order (phase 4 proved bit-identical
   placement depends on replay order).
4. The large, avoidable cost is predicate scans, not pass count:
   `opInPaintRange` 24 ms/iter (87M span checks), `collectTableBorderSegments`
   22 ms/iter, `normalizeTableRowGaps` 22 ms/iter, `beforeAlways` index
   rebuild 18 ms/iter, `validatePaintPageIndices` 14 ms/iter,
   `repeatTableHeaders` 12 ms/iter. Total 112 ms/iter of scans that either
   repeat the same question or answer it with a linear walk.
5. Streaming page sealing is not applicable to the current order: seals run
   after splits and sticky, and fixtures 60/61 depend on the global order
   (`plans/0.2.6/perf-improve/results/phase-3/seal-allocation.md`). A streaming
   redesign would be a new behavior risk with no measured pass to save.

## 5. Writer: flate time, byte provenance, options

### 5.1 Where the content bytes come from

Probe: the copy writes its output to scratch `nocompress-500.pdf` and
`compressed-500.pdf` under `/tmp/opencode/perf-time-pag/raw/` (the 12 MB
uncompressed file is not committed). Split by stream delimiters, cross-checked
against `/Length` entries (`raw/pagination-paint/pdf-stream-split.txt`):

| surface | compressed | uncompressed |
|---|---:|---:|
| total output bytes | 1,419,234 | 12,176,007 |
| stream bytes | 1,160,599 | 11,927,371 |
| non-stream bytes (dicts, xref, trailer) | 258,635 | 248,636 |
| stream count | 507 | 507 |

507 streams = 500 page content streams plus 7 embedded-resource streams (3
`FontFile2` font programs and 4 CMap/other streams, from the PDF object walk).
Page content is 11.9 MB raw, about 23.8 KB per page (max stream 23,829 B,
min 339 B), and compresses 10.3:1. `finalizePage` flates every page stream one
at a time (`pdf.go:1107`), and the compressor pool already reuses one zlib
writer per process (`flatePool`, `pdf.go:1633`; `Reset` at `:1648`). The only
copy left in `flateBytes` is the result copy at `:1654`, 6 ms/iter.

Dropping compression is not a speedup: the uncompressed 500-page row measured
1,822 ms warm in the first writer batch (single sample), against 1,411 to
1,477 ms for the same binary with compression, because 12 MB of raw streams
must be copied into the output buffer. Compression pays for itself.

### 5.2 In-process A/B of compression modes

Method: the probe binary switches the compressor between runs in one process
(`pdf.PerfTimeSetFlateMode`, copy only) and drains the state pool so the level
takes effect. Six cycles of `6s,6p,1s,1p` where `s` is serial and `p` is the
8-worker parallel path. Each conversion warms up first. Raw:
`raw/pagination-paint/writer-ab-clean.txt`. Per-cycle deltas against the same cycle's serial
level-6 row (negative is faster):

| cycle | 6s (ms) | 6p delta | 1s delta | 1p delta |
|---:|---:|---:|---:|---:|
| 0 | 1159.1 | -141.1 | +49.5 | -75.3 |
| 1 | 1155.1 | -163.6 | -63.5 | -135.0 |
| 2 | 1272.7 | -159.9 | -146.5 | -1.3 |
| 3 | 1347.8 | -199.7 | -196.0 | -240.2 |
| 4 | 1082.4 | +11.4 | +16.7 | -25.8 |
| 5 | 1142.2 | -33.8 | -0.1 | -53.2 |
| median | 1157.1 | **-150.5** | -31.8 | -64.3 |

| mode | output bytes | delta bytes | B/op delta vs 6s |
|---|---:|---:|---:|
| 6s (today: level 6, serial) | 1,419,234 | 0 | baseline (230.8 to 235.7 MB per run) |
| 6p (level 6, 8 workers) | 1,419,234 | 0 | +6.0 MB (236.9 to 237.7 MB, every cycle) |
| 1s (level 1, serial) | 1,662,157 | +242,923 (+17.1%) | +0.6 to +1.3 MB |
| 1p (level 1, 8 workers) | 1,662,157 | +242,923 (+17.1%) | +9.0 MB |

Findings:

- Parallel level 6 is the only writer option with a measured wall win and
  byte-identical output: median -150.5 ms per conversion, range -199.7 to
  +11.4 over six cycles. That is close to the 152 ms/iter of flate CPU in the
  profile, so parallelization removes essentially the whole flate wall.
- The naive parallel implementation raises B/op by 6.0 MB per conversion
  (`raw/pagination-paint/writer-ab-alloc.txt`), consistent with about nine 663 KB zlib writers
  (`zlib.NewWriterLevel`, `pdf.go:1645`) being created per conversion because
  the pool does not reliably recycle them across conversion boundaries. A
  plan implementation must retain a fixed worker set (states owned for the
  process lifetime or per `Document`) and prove warm B/op does not rise. That
  design adds no per-conversion allocation except the compressed-stream map
  (500 entries plus slices, under 0.5 MB).
- Level 1 saves only about 32 ms median (not additive with parallel, which
  already removes the compute wall), changes every compressed stream, and
  grows output by 17.1 percent. It is not worth taking on its own; it could
  ride along only if the product accepts larger files and the byte-identity
  rule is relaxed for the writer.
- Stream reuse and writer pooling already exist; the result copy is 6 ms/iter.
  Serialization outside flate is about 4 ms/iter and not a lever.

## 6. Ranked options for a 2x cut from this group

Expected milliseconds are per 500-page conversion. "CPU" rows come from
`cum / 5` in the canonical profile; "measured wall" rows come from the
in-process A/B. B/op effects are from the rate-1 profile and the AB allocation
deltas.

| rank | option | expected ms | B/op effect | correctness risk | falsifiable probe |
|---:|---|---:|---|---|---|
| 1 | Parallel level-6 page-stream compression with retained workers | 150 (measured wall) | must be designed flat; naive +6.0 MB | low-medium: streams are independent; bytes must stay identical | AB harness `raw/pagination-paint/writer-ab-clean.txt`; require median delta <= -100 ms, output bytes 1,419,234, warm B/op delta <= +0.5 MB |
| 2 | Pagination predicate elimination: O(1) `opInPaintRange` + census skip for avoid/afterBreaks/orphans + validate dedupe + seal-segment index reuse + table-pass guard | 80 to 120 total (24 + 30 + 8 + 22 + up to 34 measured components) | falls: reuse `newBreakScanState`/`sizePageBuckets`/segment scratch, about 10 to 20 MB | medium: every skipped pass must be provably no-op; fixtures 31/60/62 pin the placement | counters must show identical changed counts (499 shifts) and output bytes; `make golden`; final profile must show `opInPaintRange` gone and the policy walks reduced |
| 3 | Avoid the `ensureFlowIndex` rebuild after the single forced-break shift (apply the suffix difference to the live index) | 18 (CPU) | falls about 6.5 MB (`resetIntBuffer`) | medium: stale index membership already changed fixture-56 in the rejected phase-6 epoch experiment | after the change, one `beforeAlways` shift must produce the same page/box assignment and byte-identical PDF |
| 4 | Paint buffer sizing: exact `contentSizeHint` or pooled content buffers | up to 20 (100 ms/5 `Grow` flat) | falls up to 20.5 MB `Buffer.Grow` | low-medium: buffer capacity only; no content change | `Buffer.Grow` bytes per conversion and `memclr` share, with output bytes unchanged |
| 5 | GC-attributable scratch cuts alone (without options 2 to 4) | about 20 (25 percent of the 78 ms/iter mark) | falls | low | gctrace mark+scan per warm iteration below 169 ms |

Overlaps to respect when planning: option 2 contains the scan fixes that also
make option 3's rebuild cheaper; option 5 is the GC dividend of options 2 to 4
and must not be claimed twice.

### Ceiling for this group

Measured per-iteration CPU from the 500-page profile, per-iteration and as a
share of the 8.000 s samples:

- `PaintContext` 376 ms (23.5 percent): pagination 152, seals 38, paint 114,
  validations and page bookkeeping 72.
- PDF writer 154 ms (9.6 percent).
- Navigation/outline collection after paint (`collectBodyNavigation`) 14 ms.
- GC mark attributable to this group's own allocations: group-owned
  allocation is about 58 MB of 235 MB (25 percent, including the content
  buffers), so 25 percent of the 78 ms/iter mark is roughly 20 ms. The
  allocator primitives charged inside the group's own phases are already in
  the 376 ms and 154 ms above. The other 75 percent of GC is driven by
  layout's display list, boxes, and styles.

Group total: about 570 ms per profiled iteration out of 1,600 ms of CPU samples
(35.6 percent). Applying that share to the canonical warm row, this group is
about 430 to 450 ms of the 1,228.72 ms. If the group became completely free,
warm 500-page time would be about 780 to 800 ms, a 1.53x to 1.57x speedup, not
2x.

The realistic sum of options 1 to 5, respecting the overlap note, is 250 to
310 ms per conversion, which is 40 to 50 percent of the 614 ms that halving
from 1,228.72 ms requires. The remaining 300 to 360 ms must come from the
layout group: style resolution is 500 ms/iter (31.3 percent of samples,
42.8 ms/iter even at 50 pages) and box construction plus display list is
364 ms/iter (22.8 percent). The display-list preallocation alone is 89.5 MB,
and its record size and pointer density drive both `(*engine).add` and the GC
scan of 174,000 objects. A 2x plan that does not cut style resolution and
display-list work cannot meet its target, no matter how much pagination and
the writer improve.

Falsifiable whole-plan probe: rebuild the canonical benchmark binary after
each option wave, run `-test.bench '^BenchmarkPDFPages$/^generic$/^500Pages$'
-test.benchtime=1x -test.count=5` and the rate-1 memory profile, and require
(a) warm median at or below 1,228.72 ms until the final wave, (b) B/op at or
below 234,995,968, (c) `output-bytes` 1,419,234 for options 1 and 3, and
(d) `make golden` with the same page envelopes.

## 7. Evidence files and disclosure

Raw under `plans/0.2.6/perf-time/profiles/raw/pagination-paint/`. The top-level
`raw/` directory is shared by sibling perf-time agents; this task's files were
moved into the `pagination-paint/` subdirectory to avoid name collisions, and
only byte-identical duplicates were removed from the top level.

- captures: `capture-log.txt`, `500-cpu-run.txt`, `50-cpu-run.txt`,
  `500-mem-run.txt`, `500-gctrace.txt`, `50-gctrace.txt`,
  `500-gctrace-parsed.txt`, `50-gctrace-parsed.txt`
- profiles: `500-cpu.pprof`, `50-cpu.pprof`, `500-mem.pprof`
- pprof text: `500-top-flat.txt`, `500-top-cum.txt`, `500-top-flat-full.txt`,
  `500-top-cum-full.txt`, `50-*` equivalents, `500-subtrees.tsv`,
  `50-subtrees.tsv`, `500-mem-top-alloc.txt`, `500-mem-top-alloc-cum.txt`, and
  `500-list-*.txt` / `500-list2-*.txt` / `50-list-*.txt` per function
- probes: `probe-500-counters.txt`, `probe-50-counters.txt`,
  `probe-500-retention.txt`
- writer: `writer-ab-clean.txt`, `writer-ab-alloc.txt`, `writer-control*.txt`,
  `writer-parallel*.txt`, `writer-level1*.txt`, `writer-nocompress.txt`,
  `pdf-stream-split.txt`
- scripts: `pprof-report.sh`, `parse-gctrace.py`, `pdf-stream-split.py`

Instrumentation disclosure. The canonical profiles, gctrace, and B/op rows use
a binary built from the committed tree with no edits. Counter, retention, and
writer experiments use a copy at `/tmp/opencode/perf-time-pag/mod/` with these
copy-only files and edits: `internal/layout/zz_perf_time_counters.go`;
increment lines in `paint_pagination_fixpoint.go`, `paint_flow_breaks.go`,
`paint.go`, `paint_pagination_seal.go`, `paint_pagination_split.go`,
`sticky.go`, `tagging.go`, `paint_flow_tables.go`, `paint_flow_orphans.go`,
`paint_pagination_chrome.go`; `internal/convert/zz_perf_time_probe.go` and
`zz_perf_time_probe_test.go`; probe points in copy
`convert.go`/`pdf_pipeline.go`; `internal/pdf/zz_perf_time_flate.go` and the
`zlib.NewWriterLevel` line plus the parallel-flate branch in copy `pdf.go`.
The probe asserts 500 pages and prints `output-bytes 1419234` for the default
path, matching the canonical capture. No probe file exists in the committed
tree; the committed tree was not modified by this task.
