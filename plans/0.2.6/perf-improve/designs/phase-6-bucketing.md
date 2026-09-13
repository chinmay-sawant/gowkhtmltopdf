# Phase 6 design: page bucketing merge and duplicate scans (PERF2-26..29)

Read-only design note for `plans/0.2.6/perf-improve/phase-wise-checklist.md`
phase 6. Evidence boundary: current source plus the 2026-09-11 captures under
`plans/0.2.6/perf-improve/profiles/`. No git command, benchmark, or make target
ran while writing this. Sibling agents are editing `internal/layout`, so line
numbers below are the read snapshot of this session; function names are the
stable anchors.

## 0. What the profiles measured

- `bucketOpsByPage` (`internal/layout/paint.go:312`): flat 220 ms, cum 310 ms,
  1.70 percent of 500-page samples (`profiles/warm-pdf-cpu.md` section 6).
  Allocation 28,282,368 B at 500 pages with 4 calls per conversion from
  `alloc_objects` (`profiles/warm-pdf-alloc.md` section 6). Per call:
  `pageOf` plus `pos` total 11,141,120 B over four calls, per-page appends
  17,096,192 B, useful per-page payload 1,392,000 B. The plan's avoidable
  target is up to 24.1 MB (PERF2-27).
- `buildFlowOpIndex` (`paint_flow_index.go:265-266`) is a one-line wrapper over
  `bucketOpsByPage`; the historical note calls them "two names for one
  bucketing algorithm" (`profiles/historical-attribution.md:185-186`).
- `validatePaintPageIndices` (`paint.go:287-302`): flat 120 ms, cum 140 ms,
  0.93 percent, three full scans at `paint.go:165`, `:184`, `:207`
  (`warm-pdf-cpu.md` section 6 and section 9 candidate 4).
- `fixedOpIndices` (`paint.go:246-256`): 1,398,784 B at 500 pages
  (`warm-pdf-alloc.md` section 7); it reserves `len(ops)` capacity before it
  knows any op is fixed, and the benchmark has none
  (`historical-attribution.md:193-194`).
- `clearStructureElements` (`tagging.go:59-63`) scans all ops at
  `paint.go:216` although `buildStructureTree` early-returns when the document
  is not PDF/UA (`tagging.go:30-32`; `historical-attribution.md:190-192`).

## 1. The four bucketing calls and the unchanged-Y condition

`bucketOpsByPage` has three direct call sites; the measured total of four
invocations comes from `ensureFlowIndex` rebuilding twice.

| # | Invocation | Direct caller | Call site | Op list state |
|---|---|---|---|---|
| 1 | `ensureFlowIndex` | `paginateOps` | `paint_pagination_fixpoint.go:27` | After layout and `applyNamedPageBreaks` (`paint.go:176`) |
| 2 | `ensureFlowIndex` rebuild | `beforeAlways` tail | `paint_flow_breaks.go:493-494` | After the first forced-break suffix shift. Every later `beforeAlways` change would add another rebuild; the measured total says one |
| 3 | `pageIndexedOps` (`seal.go:629-630`) | `stripOrphanRowChrome` (`seal.go:586-591`) | `paint.go:196` | After `splitCrossingRects` (`paint.go:192`) replaced `res.Ops` (`paint_pagination_split.go:67`) and wrote fragment Ys |
| 4 | `pageBuckets` (`paint.go:352-364`) | `buildPagesAfterSplits` (`paint.go:260-261`) | `paint.go:212` | After `stripOrphanRowChrome`, `stretchPaginatedChrome` (`:197`), `capTablePageBreaks` (`:200`), `applyStickyPrint` (`:204`), `stretchPaginatedChrome` (`:205`) |

All four use the same content height and the same edge bias (`layoutEpsilon`).
The mapping of the fourth call needs the test-only counter below to be pinned;
it is the only reading consistent with four `pageOf` allocations per conversion
at every size.

Exact condition for reuse between two calls: the same `contentH` (exact float
compare), the same `len(ops)`, no write to any non-fixed op `Y`, and the same
edge-bias policy since the previous bucketing. An op-count change alone
invalidates every bucket index. The mutation sources between the calls:

- Between 1 and 2: the forced-break shift itself. Not reusable.
- Between 2 and 3: `repeatTableHeaders` appends thead clones and calls
  `invalidateFlowIndex` (`paint_flow_tables.go:696` and `:709`);
  `normalizeTableRowGaps` (`paint_flow_tables.go:43`) reads `res.flowPages`
  directly through `rowChromeAbove` (`paint_pagination_fixpoint.go:310-331`)
  and shifts chrome; `normalizeLeadingRoundedCallouts` calls
  `shiftSamePageFromY`, which invalidates (`paint_flow_breaks.go:1004`); then
  `splitCrossingRects` replaces `res.Ops` and writes fragment Ys
  (`paint_pagination_split.go:67`, `:124`, `:130`, `:165`). On the benchmark
  the table clone append fires on every page, so call 3 always needs a rebuild.
- Between 3 and 4: `stripOrphanRowChrome` writes `Y` in `tightenLastRowOp`
  (`paint_pagination_seal.go:757`); `stretchPaginatedChrome` writes `Y` in
  `normalizeOwnVerticalChrome` (`paint_pagination_chrome.go:178-206`);
  `capTablePageBreaks` appends seal ops and invalidates through
  `sealBorderGap` (`paint_pagination_seal.go:80-105`); `applyStickyPrint`
  writes `Y` through `shiftStickyOps` (`sticky.go:158-167`) without any
  invalidation. All four fire on the benchmark, so call 4 also cannot reuse
  call 3.

Conclusion: on the generic benchmark the Y-unchanged reuse never triggers.
The phase-6 bytes must come from one retained storage that is reset and
recomputed in place, not from skipping invocations. This is a correction to any
expectation that the four calls collapse into fewer on the report workload.

## 2. PDF-04 and PDF-05 lifetime rules inherited

PDF-04 (`plans/0.2.6/perf-review/results/2026-09-11/pdf-02-05.md:147-157`):

- `paginateOps` returns only an error. The pre-split op-to-page map was
  deleted, not cached.
- The reason is exactly the condition above: between `paginateOps` and
  `buildPagesAfterSplits`, `splitCrossingRects` inserts ops and
  `capTablePageBreaks`/`applyStickyPrint` move Y. The pre-split assignment is
  stale as well as unused.
- Rule for this phase: never retain a bucketing result across an operation
  that can append, remove, or move ops. A result may be retained only as the
  live index and must be invalidated before the next mutating pass.
- Tests that need the settled pre-split assignment keep using the test-only
  `paginateOpsForTest` (`paint_flow_page_test.go:99-116`).

PDF-05 (`plans/0.2.6/perf-review/results/2026-09-11/pdf-02-05.md:190-224`):

- `PaintContext` releases the six pagination-only fields at `paint.go:239`
  after `paintPages`. `Ops`, `Pages`, `Locations`, `boxes`, and `root` must
  stay: conversion reads them at `internal/convert/convert.go:663-667` and
  rewrites link URIs at `:744-750`.
- `normalizeTableRowGaps` reads `res.flowPages` directly before the release
  point, so the release cannot move earlier than the end of `paintPages`.
- `ensureFlowIndex` rebuilds on demand after the release; the release is a
  cache drop, not a loss.
- `TestPaintReleasesPaginationIndexesAfterReaders` (`paint_release_test.go:24`)
  pins all six fields nil after Paint and requires the on-demand rebuild.

What may be reused, stated exactly:

- Reusable: the live page index within one stable `contentH`/Y phase; the
  backing arrays of `pageOf`, `pos`, `pages`, and the per-page bucket slices,
  reset with `[:0]` and reused on the next rebuild.
- Must be invalidated before: appends or replaces of `res.Ops`
  (`paint_pagination_split.go:67`, `paint_flow_tables.go:696`,
  `paint_pagination_seal.go:80-105`), any non-fixed op `Y` write
  (`paint_flow_breaks.go:437`, `sticky.go:166`, `paint_pagination_split.go:124`
  and `:130` and `:165`, `paint_pagination_fixpoint.go:215` and `:247` and
  `:279`, `paint_pagination_chrome.go:205` and `:389`,
  `paint_pagination_seal.go:757` and `:833`), any `pageSize` change, and
  `Workspace.Release` (`layout.go:298-315`).
- Must be released after: `paintPages`, at the current `paint.go:239` point.

Stale-index failure modes:

1. Op count changed without invalidation: bucket entries point at shifted
   indices or out of range. Pages get the wrong ops; ordered-text needles and
   page counts fail, and `validatePaintPageIndices` cannot see it.
2. Y changed without invalidation: bucket says page A, the op is now on page
   B. Text or a rect fragment lands on the wrong page. Golden catches this
   only where the fixture exercises the changed shift.
3. A retained buffer consumed after release: nil fields force a rebuild
   mid-paint, which changes the release contract and can retain the index
   through PDF finalization again (the PDF-05 regression).
4. Scratch aliased into `res.Pages` and then invalidated: `Pages` is either
   cleared with the cache or silently mutated by the next reset. Keep
   ownership separate (copy buckets into `res.Pages`).

## 3. Merge design

One implementation, one owner, one storage.

```go
// paint_flow_index.go, next to ensureFlowIndex
type pageIndex struct {
    pages    [][]int
    pageOf   []int
    pos      []int
    pageSize float64
    epoch    uint64 // res.opEpoch at build time
}

func buildPageIndex(ops []Op, pageSize, edgeBias float64, into *pageIndex) bool
```

- `bucketOpsByPage` (`paint.go:312`) is the implementation body; move it next
  to `ensureFlowIndex` so the file that owns the flow index owns the builder.
- `buildFlowOpIndex` (`paint_flow_index.go:265-266`) is deleted.
  `pageBuckets` (`paint.go:352`) and `pageIndexedOps` (`seal.go:629`) become
  `ensureFlowIndex` plus a read of the live fields.
- `Result` keeps one `pageIndex` (the live flow index). Do not add a second
  transient struct unless a call needs a result that survives a mutation,
  which by PDF-04 none does.
- Add `res.opEpoch`, incremented by every append/remove/replace of `Ops` and
  every non-fixed op `Y` write. Split `invalidateFlowIndex` into
  `markFlowStale(res)` (stamp stale, keep storage) and `releaseFlowIndex(res)`
  (nil all six fields for PDF-05). `ensureFlowIndex` rebuilds when the epoch,
  `pageSize`, or op count no longer matches. This closes the existing gap:
  `applyStickyPrint` (`sticky.go:158-167`), `normalizeOwnVerticalChrome`
  (`paint_pagination_chrome.go:178-206`), and `tightenLastRowOp`
  (`paint_pagination_seal.go:757`) write Y today without invalidating. They are
  harmless now because call 4 recomputes fresh, but they are exactly the stale
  paths a cache would expose.
- Reset, do not reallocate: `buildPageIndex` sets
  `pageOf = pageOf[:len(ops)]`, `pos = pos[:len(ops)]`,
  `pages = pages[:maxPage+1]`, and `pages[p] = pages[p][:0]` before the
  counting pass. Recompute every page assignment every call; only the storage
  is reused.
- `buildPagesAfterSplits` (`paint.go:260-277`) calls `ensureFlowIndex` then
  fills `res.Pages` by copying the buckets with their exact capacity. Copying
  keeps `res.Pages` and the live index from aliasing; cost is one O(ops) int
  copy, no second bucketing pass.

Measured expectation: the first call allocates as today; calls 2 to 4 reuse
`pageOf`, `pos`, and the per-page bucket storage, which removes up to
8.35 MB of scratch plus most of the 15.7 MB bucket churn at 500 pages. Verify
against the `paint.go:312` allocation line, not against a call count: the
number of recomputations stays four on the benchmark.

## 4. Tests and ownership probes

Keep green (existing contract):

- `TestPageBoundaryBucketersAgree` (`paint_flow_page_test.go:14`) pins the
  shared edge bias at `k*contentH`; after the merge it must test the one
  remaining function.
- `TestShiftOpsOnlyMaintainsFlowIndex` and
  `TestShiftFlowYNegativeMaintainsFlowIndex`
  (`architecture_followup_test.go:321`, `:344`) pin in-place index updates.
- `TestGeneratedPaginationOpsInvalidateFlowIndex` (`:371`) pins appends.
- `TestPaintReleasesPaginationIndexesAfterReaders` (`paint_release_test.go:24`)
  pins the PDF-05 release and on-demand rebuild.
- `TestPaginateOpsDoesNotBuildDiscardedPageMap`
  (`paint_flow_page_test.go:128`) pins PDF-04.
- Pagination and table suites: `pagination_thead_test.go`,
  `table_continuation_border_test.go`, `pagination_ctx_test.go`.

New probes:

1. `TestPageIndexReuseAllocatesNothing`: `testing.AllocsPerRun` around a
   second `ensureFlowIndex` rebuild on an unchanged `Result`; the second
   rebuild must allocate 0 (today it allocates `pageOf`, `pos`, `pages`, and
   the per-page slices).
2. `TestPageIndexStaleAfterEveryMutator`: table-driven over the mutation
   kinds: append (`splitCrossingRects`, `repeatTableHeaders`,
   `capTablePageBreaks`), Y write (`shiftIndexedOp`, `applySuffixDifferences`,
   `shiftStickyOps`, `normalizeOwnVerticalChrome`, `tightenLastRowOp`),
   page-size change, and release. Each case asserts the epoch differs and the
   next build matches a from-scratch bucket.
3. `TestPagesDoNotAliasLiveIndex`: after `buildPagesAfterSplits`, mutate an op
   through a flow shift and assert `res.Pages` still reflects the paint-time
   assignment.
4. `TestBucketCallCountPerPaint`: test hook counter around `buildPageIndex`;
   record 4 calls per conversion before the change and assert at least 4 after
   (the win is storage, not fewer calls). If a document shape does avoid
   mutations between calls, assert the reuse path is taken.
5. `TestPaintReleasesPaginationIndexesAfterReaders` extended: assert the
   retained `pageIndex` storage is nil after release and that the arrays are
   not reachable from `res.Pages`.

Ownership rule the probes must protect: the live index is owned by
`ensureFlowIndex`, every mutation path marks it stale, `buildPagesAfterSplits`
copies out of it, and `paint.go:239` releases it.

## 5. Duplicate all-op scans and skip conditions

1. `validatePaintPageIndices`, three calls at `paint.go:165`, `:184`, `:207`
   (`paint.go:287-302`), 120 ms flat.
   - Epoch skip: run a call only when `res.opEpoch != res.validatedEpoch` or
     `contentH` differs from the validated height. Call 1 always runs once per
     paint; call 2 runs because pagination moved ops; call 3 runs because
     splits, strip, stretch, cap, and sticky moved or added ops. On the
     benchmark the epoch alone saves nothing.
   - O(1) guard: maintain `res.maxOpY` over non-fixed ops in the same
     centralized mutation helpers, then each check is a bound compare against
     `maxFlowPageIndex*contentH`. This removes the 0.12 s. Failure mode: a
     direct Y write that does not update `maxOpY` hides an out-of-range op.
     There are dozens of direct write sites, so land this only after the epoch
     is complete and add a mutator-coverage test.
2. `clearStructureElements` (`tagging.go:59-63`, called `paint.go:216`).
   - Safe skip when `!doc.IsUA() && !res.hasStructElems`. `buildStructureTree`
     early-returns for non-UA (`tagging.go:30-32`) and `cloneOps` nils
     `StructElem` (`layout.go:233`). Set `res.hasStructElems = true` at the end
     of `buildStructureTree` and false after a clear, so a result painted once
     into a UA document still gets cleared before a non-UA repaint. The
     benchmark is a fresh non-UA document and skips the scan.
3. `fixedOpIndices` (`paint.go:246-256`, called `paint.go:188`).
   - Safe skip when `res.hasFixedOps == false`. Set the flag where `Fixed` is
     stamped: `markOpsFixed` (`layout_chrome.go:12-20`) and the subtree stamp
     at `layout_chrome.go:951-957`. Return nil without
     `make([]int, 0, len(res.Ops))`.
   - `buildPagesAfterSplits` ignores the parameter today (`_ []int`,
     `paint.go:260`), `contentSizeHint` only takes `len`, and `paintPages`
     (`paint.go:395`) ranges over it, so nil is safe. This removes the
     1,398,784 B line at 500 pages and the scan.

## 6. Ranked recommendation and smallest first step

1. The merge plus retained index storage. It is the only phase-6 change that
   moves a measured allocation line by more than 1.5 MB, and it is the
   prerequisite for the skip guards because they all need the epoch.
2. `fixedOpIndices` early-out: 1.4 MB and a full scan, trivially exact.
3. `clearStructureElements` skip: narrow condition, no measured allocation,
   removes 174,000 writes per conversion on the benchmark.
4. `validatePaintPageIndices` O(1) guard: 0.12 s, but only safe once every
   Y-writing path is instrumented. Do it last.

Smallest first step: add `res.opEpoch` with increments at every op `Y` write
and append site, including the currently unguarded writers (`shiftStickyOps`,
`normalizeOwnVerticalChrome`, `tightenLastRowOp`, and the chrome writes in
`snapOpForward`/`shiftNearestOwnedChrome`), add the test-only bucket-call
counter, and change no behavior. Run
`go test ./internal/layout -run 'Test.*(Pagination|Paginate|Table.*Continuation|FlowPage|PaginateOps)' -count=1`
and `TestPageIndexStaleAfterEveryMutator`. That proves invalidation coverage
before any storage is retained.
