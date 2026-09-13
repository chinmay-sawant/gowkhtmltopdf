# Phase 4 pagination and paint: design note (PERFT-15 to PERFT-17)

> Read-only design for phase 4 of `plans/0.2.6/perf-time/phase-wise-checklist.md`.
> No production code, test, fixture, or checklist file was changed. No benchmark
> was run. All costs are the committed profile numbers from
> `plans/0.2.6/perf-time/profiles/pagination-paint-deep.md`; arithmetic on those
> numbers is labeled as arithmetic. A sibling session is editing
> `internal/layout` while this note was written, so line numbers are a snapshot
> and symbol names are the durable anchor. Line numbers below were re-verified
> after the sibling's seal edits.

## 0. Measured frame

The profile runs 5 iterations at 500 pages (4 warm + 1 cold) and reports
cumulative CPU samples. Per-iteration numbers are the profile's own `cum / 5`
rows. They are profile CPU, not wall time.

| item | measured | source |
|---|---|---|
| `opInPaintRange` | 120 ms total, 24 ms/iter; 174,000 calls; 87,000,000 span checks; `tablePaintRanges` 10 ms total | profile 2.1, counters |
| `paginationFixpoint` policies | avoidInside 60 ms, beforeAlways 10 ms, afterBreaks 70 ms, rowsIntact 10 ms, keepHeadingWithNext 0 ms, orphansWidows 20 ms; 170 ms total, 34 ms/iter | profile 2.1 |
| `validatePaintPageIndices` | 70 ms total, 14 ms/iter; 3 calls; 522,000 op checks | profile 2.2, counters |
| seal pass | `capTablePageBreaks` 190 ms, 38 ms/iter; `collectTableBorderSegments` 110 ms (22 ms/iter); `clusterVerticals` 20 ms plus 2.63 MB; `capTableMaxPage` 10 ms; map appends 5.74 MB; `collectBorderSegmentOps` slices 6.64 MB | profile 2.2, 3.3 |
| table passes after the fixpoint | `repeatTableHeaders` 60 ms (12 ms/iter); `normalizeTableRowGaps` 110 ms (22 ms/iter, 20 + 90 ms in the two `rowPaintBand` calls) | profile 2.1 |
| `beforeAlways` index rebuild | 90 ms (18 ms/iter) for `invalidateFlowIndex` + `ensureFlowIndex`; `resetIntBuffer` 6.50 MB | profile 2.1, 3.3 |
| per-build scratch | `newBreakScanState` 5.60 MB at 4 states; `sizePageBuckets` 5.56 MB | profile 3.3 |

Checklist budgets this phase at 80 to 120 ms and B/op down 10 to 20 MB. The
B/op line needs care: most of the named bytes belong to structures that are
built once per conversion (see section 4.3), and only the repeated builds
(`newBreakScanState` 4 times, `ensureFlowIndex` rebuild, `sizePageBuckets`)
actually allocate some of those bytes more than once. The note says for each
item whether the B/op fall is real on this workload or structural.

## 1. PERFT-15: `opInPaintRange` O(1)

### 1.1 Callers and the exact predicate

`opInPaintRange` has exactly one caller: `snapCrossingTextOps`
(`internal/layout/paint_pagination_fixpoint.go:145`). Grep confirms no other
call.

The call site builds `tableRanges := tablePaintRanges(res)` once
(`paint_pagination_fixpoint.go:137`) and then, per op:

- `paintOp.Fixed` skips viewport-stamped ops outright (`:145`).
- `opInPaintRange(idx, tableRanges)` skips ops inside any table op span.
- The remaining ops are switched by kind (`:149`). Only `OpText`, `OpBullet`,
  `OpImage`, `OpLinkURI` are candidates.
- The page index is `checkedFlowPageOfY(paintOp.Y, contentH)` (`:153`). This is
  raw truncation: zero edge bias.
- The boundary is `float64(page+1) * contentH`; an op snaps only when
  `paintOp.Y + opInkHeight(op) > boundary + 1e-9` (`:158-160`).

`tablePaintRanges` (`:171-187`) walks `flowBoxList(res)`, keeps boxes with
`kind == boxKindTable` and `opStart <= opEnd`, and appends
`paintRange{first: opStart, last: opEnd}`. `opInPaintRange` (`:189-197`) is a
linear scan and the predicate is exactly:

```text
exists span: span.first <= index && index <= span.last
```

It is an index-interval test. It contains no page index, no Y coordinate, and
no epsilon. Nested tables produce overlapping spans, and the inclusive bounds
keep that exact. The `1e-9` slack at `:159` is a comparison slack in the
caller, not part of this predicate. `layoutEpsilon` boundary bias lives in
`flowPageOfY` / `buildPageIndex` (`internal/layout/paint_flow_index.go:528-535`,
`:355-379`) and is not applied at `:153` because that lookup calls
`checkedFlowPageOfY` directly.

The reason table ops are excluded: table pagination owns them through
`rowsIntact` (`paint_flow_tables.go:10`), `normalizeTableRowGaps`
(`paint_flow_tables.go:43`) and `repeatTableHeaders`
(`paint_flow_tables.go:362`). Snapping a table op here would move it twice.

### 1.2 O(1) design

Build a membership table once per `snapCrossingTextOps` call, before the loop:

```go
inTable := make([]bool, len(res.Ops)) // or a []uint64 bitset
for _, span := range tableRanges {
    for i := span.first; i <= span.last; i++ {
        inTable[i] = true
    }
}
```

Then the loop test is `paintOp.Fixed || inTable[idx]`. Exactness rests on
three facts:

1. The predicate is purely index-based, and `inTable` stores exactly the union
   of the inclusive spans.
2. The spans are collected before the loop and are not mutated inside it.
   `snapCrossingTextOps` never appends, removes, or reorders ops. It does move
   Y values (`:215`, `:240-249`), which does not affect index membership.
3. Nested or overlapping spans set the same bit, which matches the `exists`
   semantics.

Cost: `O(len(ops) + sum(span lengths))`. The profile's op inventory puts 174,000
ops, 500 tables, and 10,500 rows in the generic document, so the fill is one
bounded pass of the same order as the display list, and the 87M span checks
become 174,000 array reads.

B/op note: a transient `[]bool` is 174 KB, or 22 KB as a `[]uint64` bitset.
The plan's rule is that B/op never rises, so retain the membership buffer on
`Result` next to the flow stores (re-slice and clear per call, reset in
`releaseFlowIndex`) or use the sorted boundary index below, which allocates
only over the spans. Do not leave the buffer as a fresh per-conversion
allocation if the aggregate phase B/op is close to the pin.

Alternative when spans cover a small fraction of the list: a sorted boundary
index. Sort the spans by `first` (stable), compute `prefixMax[i] = max(last)`
over that order, then binary search for the last span with `first <= index`
and test `prefixMax[hi] >= index`. This is the classic interval stabbing test
and is O(log T) per query with T table boxes (500 here). It avoids the fill
pass and the `len(ops)` allocation. Both designs are exact; the bitset is
simpler and faster here, the sorted index is the fallback when the union is
sparse.

Do not convert this to a Y-range test. A non-table op can share a Y band with
table ops (`position:absolute` content, overlapping floats, snapped text), and
table ops move in Y while their membership stays index-based. A Y test would
change which ops snap.

### 1.3 Invalidation and stale-index detection

If the membership table is local to `snapCrossingTextOps`, there is nothing to
invalidate: one call per Paint (counter `snap=1`), and no op-list mutation
inside the call.

If it is cached on `Result` across calls, invalidate it whenever any of these
happens:

- `res.Ops` is replaced or reordered: `splitCrossingRects`
  (`paint_pagination_split.go:67-68`, including `remapBoxOpRanges`),
  `CloneResult` (`layout.go:214`).
- `res.Ops` is appended: `cloneHeaderOps`
  (`paint_flow_tables.go:696-710`), `sealBorderGap`
  (`paint_pagination_seal.go:102-105`), `sealStickySectionBottom`
  (`paint_pagination_seal.go:1000-1011`).
- The box tree is rebuilt or box op ranges are remapped.

A generation counter that changes with `len(res.Ops)` and a box-range remap
counter is enough; the members table itself is small.

Stale failure modes:

- Stale `true` for an index that is no longer inside a table: a crossing text
  op is not snapped, so the line stays on the previous page when it should
  move.
- Stale `false` for an index now inside a table (after a range remap): a table
  op snaps and table chrome double-moves.

Both are placement changes. Detection: a test helper that, after each
op-list/range mutation, asserts `inTable[i] == recompute(i)` for all i, plus
the golden fixtures that pin snaps: fixture 31 (Row 28 white background, the
comment at `paint_pagination_fixpoint.go:221-227`), fixture 56 and fixture 60.
The smallest first step avoids the whole question by keeping the structure
local.

### 1.4 First step and probe

First step: inside `snapCrossingTextOps`, keep `tablePaintRanges` and add the
`inTable` fill; delete `opInPaintRange`. Nothing else changes.

Falsifiable probe: a temporary counter for span checks must read 0 for the
generic workload while the op-scan counter stays 174,000; the 500-page output
reports `output-bytes 1419234`, 500 pages, and `make golden` is byte-identical.
On a crafted document with a table and a crossing text op outside any table,
the text op must still snap exactly once.

## 2. PERFT-16a: census-skip the no-op policy walks

### 2.1 Every scan that returns early on this workload

Counters (`profiles/raw/pagination-paint/probe-500-counters.txt`, one warm
conversion): `paginateOps=1`, `fpIter=1`, `fpPolicy=6`, and every fixpoint
policy reports `0` changes. `beforeAlways=4 chg=1 targets=1996 tchg=499`: the
one real forced-break pass is the first `settleBeforeAlways`; the other three
calls find every target already fresh.

| walk | function and call | cost | exact skip condition | must still run when |
|---|---|---|---|---|
| avoid inside | `avoidInside` `paint_flow_breaks.go:36`, called `paint_pagination_fixpoint.go:104` | 12 ms/iter | no box with `isAvoidInsideBreak(style)` (`:20-31`) outside a table and with `height > 0` | any box carries `page-break-inside: avoid`, `avoid-page`, or `avoid-column` |
| forced breaks | `beforeAlways` `:499`, called from `settleBeforeAlways` (`:86`) twice and the fixpoint (`:105`) | 2 ms/iter plus 6 ms for the two target walks across 4 calls | no box with `PageBreakBefore == pageBreakAlways` (collected at `:910-926`) | any forced break box exists. Here 1,996 targets exist, so this flag alone cannot skip the calls; only the retained target list helps (section 2.4) |
| after breaks | `afterBreaks` `:930`, called fixpoint `:109` | 14 ms/iter | no box with `PageBreakAfter` `always` or `avoid` (`:943-951`) | any `page-break-after` declaration exists |
| table rows | `rowsIntact` `paint_flow_tables.go:10`, called fixpoint `:113` | 2 ms/iter | no box with `len(rows) > 0`; exact geometry form: no row op band straddles a page (`shiftRowToPage` requires `hi > layoutOut`, `paint_flow_tables.go:263-266`) | any table row exists; the structure flag is true on this document (500 tables, 10,500 rows) |
| heading keep | `keepHeadingWithNext` `paint_flow_breaks.go:1170`, called fixpoint `:117` | 0 ms | no `h1` to `h6` box with a valid op range (`:1224-1231`) | any heading exists. Here 500 headings exist, but the function is already cheap |
| orphans and widows | `orphansWidows` `paint_flow_orphans.go:14`, called fixpoint `:121` | 4 ms/iter | no block box straddles a page boundary, see section 2.2 | any block box can straddle |

`keepImplicitAsides` (`paint_flow_breaks.go:69`) is not part of this bundle; it
runs before text snap and the counter shows one unchanged pass.

### 2.2 The orphan census must be geometric, not style-based

The profile's option text suggests a census on "non-default `orphans`/
`widows`". That flag is not exact on this code. `resolveOrphansWidows`
(`paint_flow_orphans.go:108-120`) defaults both values to 2, so every block
enforces Rule 3 with defaults, and the geometric fallback
(`orphansWidowsHeuristic`, `:139-169`) applies to blocks without countable
lines. Both move paths require a straddle first: `enforceOrphansWidows` returns
false when `hIdx <= layoutOut` (`:66-71`), and the heuristic returns false on
the same test (`:149-154`). Therefore the exact skip condition is: no
`boxKindBlock` box with `height > 0` and a valid op range satisfies

```text
int(box.y/contentH) != int((box.y+box.height)/contentH)
```

Because `box.y` moves during the fixpoint, this flag is only trustworthy while
no policy has reported a change since the census. `paginationFixpoint` already
computes `changed` (`paint_pagination_fixpoint.go:125`); use it to mark the
geometry census dirty and recompute before the next orphan use.

For the avoid-inside flag, mirror the walk's two guards exactly:
`!boxInsideTable(b) && b.height > 0 && isAvoidInsideBreak(b.style)`. For boxes
built from the DOM, `boxInsideTable` (`paint_flow_breaks.go:114-126`) subsumes
the walk's `inTable` traversal flag. Synthetic nil-node boxes under a table
only make the census conservative: the flag is set, so the walk runs.

The other walks are position-independent: avoid-inside and after-breaks are
style predicates, heading and table-row presence are structure predicates, and
`applyNamedPageBreaks` (`paint.go:176`, implementation `page_named.go:11-61`)
only flips `PageBreakBefore` before pagination starts. Compute the census after
`applyNamedPageBreaks` so named-page forced breaks are counted, and never
before it.

### 2.3 Census shape, placement, and lifetime

```go
type paginationCensus struct {
    hasAvoidInside  bool // style only
    hasAfterBreak   bool // style only
    hasForcedBefore bool // style only
    hasTableRows    bool // structure only
    hasHeadingKeep  bool // structure only
    blockStraddles  bool // geometry, dirty when policies change
    tableSpansPages bool // geometry, dirty when policies change
}
```

One walk over `flowBoxList(res)` fills all seven flags. Place it in
`PaintContext` after `applyNamedPageBreaks` (`paint.go:176`) and before
`paginateOps` (`paint.go:178`), or at `paginateOps` entry after
`ensureFlowIndex` (`paint_pagination_fixpoint.go:27`). Style and structure
flags are computed once per Paint; geometry flags are recomputed only when
`paginationFixpoint` reports a change (which never happens on the generic
workload).

The census must be cleared at the PDF-05 release point (`releaseFlowIndex`,
`paint_flow_index.go:138`, called from `paint.go:245`) so it does not survive
finalization. Because `CloneResult` (`layout.go:208`) clones boxes and rewrites
op slices, a clone must start with an empty census, and `Workspace.Release`
(`layout.go:319`) must clear it.

Cost honesty: one walk over 54,503 boxes replaces the 30 ms of avoid + after +
orphan walks, but the profile has no row that measures the combined census.
Whether it pays is a probe result, not a profile result. The orphan and table
flags need row or op arithmetic per box, so the probe must show the census
total below the current policy cost before the skip ships.

### 2.4 Secondary: the `beforeAlways` target walks and scan state

`beforeAlways` runs 4 times per conversion and each call rebuilds the target
list (`collectBeforeAlwaysTargets`, `paint_flow_breaks.go:474`, which calls
`collectBeforeAlwaysBoxes` `:910`) and a `breakScanState`
(`newBreakScanState` `:383`, `suffixDy` of `opCount+1` floats). The profile
charges 30 ms total to the target collection (6 ms/iter) and 5.60 MB to four
scan states. Two cheap structural wins:

- Retain one `breakScanState` in a `Result` store, reset `events[:0]`, clear
  and re-slice `suffixDy`, and reuse it for all `beforeAlways` calls. Safe
  until op count changes; re-slice when `opCount+1` grows.
- Cache the target list on `Result` until `splitCrossingRects` replaces
  `res.Ops`; op `start` indices do not change under Y shifts, and box pointers
  stay valid. `cloneHeaderOps` changes `len(Ops)` but not target starts, so the
  cache only needs invalidation when op ranges are remapped.

Both are secondary to the three named walks. They need their own allocation
probe because the 5.60 MB figure is from a rate-1 profile.

### 2.5 First step and probe

Step 1: the two style-only flags (`hasAvoidInside`, `hasAfterBreak`) gated in
`paginationFixpoint`. Leave the orphan, rows, heading and `beforeAlways` paths
untouched. Step 2: the geometry straddle census for `orphansWidows` and
`rowsIntact`. Step 3 (optional): the `breakScanState` and target-list reuse.

Probe for step 1: counters must show `avoid=0` and `afterBreaks=0` invocations
while `fpChg` stays `0,0,0,0,0,0`, `beforeAlways=4 chg=1 targets=1996
tchg=499` is unchanged, the 500-page output is `output-bytes 1419234` with 500
pages, and `make golden` is byte-identical. Unit probes: a document with
`page-break-inside: avoid` still moves the box (`TestPageBreakInsideAvoid`,
`layout_test.go:1234`), and a document with `page-break-after: always` still
breaks.

Probe for step 2: instrument the straddle census; a document with a block
straddling a boundary still runs the walk, fixture 56 Chrome regression
(`TestFixture56Domain08PageHasProgressAndNoStrayBottomLine`,
`internal/layout/fixture56_chrome_regression_test.go:352`) passes, and the
500-page counters still show the same page assignment.

## 3. PERFT-16b: collapse the three `validatePaintPageIndices` calls

### 3.1 What each call guards

`validatePaintPageIndices` (`paint.go:310-326`) checks two things: `contentH`
is finite and positive (`errInvalidContentHeight`), and every non-fixed op's
`Y` maps through `checkedFlowPageOfY` into the bounded page range
(`paint_flow_index.go:10` `maxFlowPageIndex = 16384`, `:535-555`), otherwise
`errOutOfRangePageIndex`.

| call | line | state validated | what it protects |
|---|---|---|---|
| 1 | `paint.go:165` | raw layout output, before `applyNamedPageBreaks` and `paginateOps`; also before the empty-ops early return at `:169` | rejects garbage input coordinates before any pass indexes them; also rejects an invalid `contentH` on the empty-document path |
| 2 | `paint.go:184` | after `paginateOps` and the first `stretchPaginatedChrome` | rejects a pagination shift or snap that produced an unrepresentable coordinate before `splitCrossingRects`, `fixedOpIndices`, `stripOrphanRowChrome`, `capTablePageBreaks`, and `applyStickyPrint` consume it |
| 3 | `paint.go:207` | after `splitCrossingRects`, `stripOrphanRowChrome`, `capTablePageBreaks`, `applyStickyPrint`, and the third stretch | the list that `buildPagesAfterSplits` (`:212`) will bucket and `populateLocations` (`:214`) will read |

### 3.2 Is one call at the latest point equivalent?

For the final output, yes. All three calls apply the same predicate to three
snapshots of the same slice; the last snapshot is the list that page buckets
and locations are derived from, and `splitCrossingRects` can only add ops, so
call 3 validates a superset of what calls 1 and 2 validated. Every reader
between the calls re-checks Y itself instead of trusting the validator:
`shiftFlowBounded` (`paint_flow_index.go:32-37`), `buildPageIndex`
(`:369-372`), `splitCrossingRects` (`paint_pagination_split.go:29-43`),
`rowChromeAbove` (`paint_pagination_fixpoint.go:317`),
`repeatTableHeaderOnPages` (`paint_flow_tables.go:406`), `capTableMaxPage`
(`paint_pagination_seal.go:275`), and `normalizeLeadingRoundedCallouts`
(`paint_flow_breaks.go:1249`). No pagination function indexes a bounded slice
with an unvalidated Y.

Two differences must be handled:

1. The empty-ops early return at `paint.go:169-174` currently relies on call 1
   for the `contentH` check. When collapsing, move the `contentH` validation
   next to its computation (`paint.go:155-158`) or into `opts.validate`
   (`:140`) so the empty path still rejects a non-finite or non-positive
   height.
2. Error timing changes for a state that is transiently invalid but valid at
   the end. Today that document errors at call 2; after the collapse it
   proceeds and likely produces correct output. No test pins these errors
   (grep over `*_test.go` in the repo for `errOutOfRangePageIndex`,
   `errInvalidContentHeight`, and `out-of-range page index` found none), so the
   collapse needs one new focused test to keep the final-call behavior pinned.

### 3.3 The invariant that must still hold

Before `buildPagesAfterSplits` runs, every non-fixed op `Y` must be finite and
map into `[0, maxFlowPageIndex * contentH)`, and `contentH` must be finite and
positive. If that fails, `buildPageIndex` returns false, `buildPagesAfterSplits`
sets `res.Pages = nil` (`paint.go:287-291`), and the document is written with
pages that have no content. The final call is what keeps that failure an error
instead of a silent empty PDF.

Profile cost of the removed scans: 70 ms total (14 ms/iter) over three calls;
the checklist budgets 8 ms for the two calls dropped. Keeping one call leaves
one third of the 14 ms, so the realistic reclaim is about 9 ms profile / 8 ms
checklist. This is a scan removal, not a placement change.

### 3.4 First step and probe

First step: move the `contentH` check next to `paint.go:155-158`, keep call 3
at `:207`, delete calls 1 and 2. No other change.

Probe: the validate counter must drop from `3/522000` to `1/174000`; a
synthetic Result with one op whose `Y` is beyond the bounded range must still
return `errOutOfRangePageIndex`; a Result with `len(Ops) == 0` and a
non-positive `contentH` must still return `errInvalidContentHeight`; the
500-page row and `make golden` are byte-identical.

## 4. PERFT-16c: seal segment index

### 4.1 What is built, and what is rebuilt between the seal calls

`capTablePageBreaks` (`paint_pagination_seal.go:33`) is the only seal entry
(`paint.go:200`). Its structures:

| structure | built at | lifetime | rebuilt between the seal calls? |
|---|---|---|---|
| `verts`, `horiz` slices | `collectBorderSegmentOps` `:336-362`; `horiz` grows in `sealBorderGap` `:107` | whole function; `horiz` grows as seals are appended | no |
| `vertStarts`, `vertEnds`, `horizByY` maps | `collectTableBorderSegments` `:313-327` | whole function; `horizByY` grows at `:109` | no |
| `starts` cluster map | `clusterVerticals(..., true)` `:62` | passed to `sealPageTopClusters` `:71` and `sealPageBottomClusters` `:75`; not rebuilt | no |
| `ends` cluster map | `clusterVerticals(..., false)` inside `sealPageBottomClusters` `:159` | only that loop | yes, this is the map built after the top seals and before the bottom seals |
| max page | `capTableMaxPage` `:38` (`:275-296`), separate full op scan | immediate | no |

So the one structure that is thrown away and would be rebuilt if a second
bottom pass ever ran is the end-cluster map; everything else is a single
build. `collectBorderSegmentOps` itself is two full visits over the line ops:
`countBorderSegmentOps` (`:337`, function `:366-383`) counts, then the fill
loop `:342-359` writes. That double scan is the concrete waste inside the
22 ms/iter component. The map appends (5.74 MB) and the cluster map
(2.63 MB) are the allocation churn.

### 4.2 Retention design with the PDF-05 release intact

Add a `sealScratch` to `Result` (next to `flowStore`, `layout.go:173-180`)
holding: `verts []vseg`, `horiz []hseg`, `vertStarts`, `vertEnds`, `horizByY`
maps, and the two cluster maps. At the top of `capTablePageBreaks`, clear the
maps and re-slice the slices to zero. A later seal pass reuses the capacity.

Release points must stay exact:

- `releaseFlowIndex` (`paint_flow_index.go:138`) is the PDF-05 release point
  and is called after `paintPages` (`paint.go:245`). Add the seal scratch reset
  there so finalization cannot retain it.
- Extend `assertPaginationIndexesReleased`
  (`internal/layout/paint_release_test.go:63-80`) so the new fields are
  pinned nil after Paint, next to the six flow fields.
- `CloneResult` (`layout.go:208`) must start the clone with an empty
  `sealScratch`, and `Workspace.Release` (`layout.go:319`) must clear it.

The retained slices must not alias anything that outlives the function. The
`horiz` slice is mutated by seal appends and read by `hCoverage`
(`paint_pagination_seal.go:455-503`); the cluster maps are read-only after
build. No live field of `Result` points into them, so resetting them after
Paint is lossless. This mirrors the existing `flowStore` ownership rule.

### 4.3 What retention can and cannot reclaim

Counters show `cap=1/174000`: `capTablePageBreaks` runs once per conversion
and `collectTableBorderSegments` runs once. Per-Result retention therefore
cannot remove the one build on this workload, and the B/op for the segment
family (6.64 + 5.74 + 2.63 MB) does not fall from retention alone. What can
fall inside the single call, with profile line costs as the bound:

- Fuse `capTableMaxPage` (`:38`, 10 ms total) into the fill scan of
  `collectBorderSegmentOps` or derive it from the collected Ys. Saves about
  2 ms/iter of the 38 ms seal pass, with an arithmetic bound of 2 ms.
- Replace the count pass (`:337`) with a fill into retained high-water slices
  from the seal scratch store. `collectBorderSegmentOps` is 70 ms total across
  the two visits; one visit plus a retained append is an arithmetic save of up
  to half, about 5 to 7 ms/iter.
- Stream the end clusters in `sealPageBottomClusters` instead of building the
  second map: iterate `vertEnds` buckets and fold each bucket into a cluster
  the way `clusterVerticals` does (`:402-431`). Saves part of the 20 ms
  `clusterVerticals` total and one map allocation, about 4 ms/iter and 1.3 MB.
  Order caution: the seal passes iterate cluster maps (`:119`, `:159`), so seal
  append order is map-iteration order today. If any byte-identical fixture
  depends on that order, the streamed form must sort the bucket keys and the
  probe must include the determinism test. The current corpus either emits no
  seals or is order-insensitive; the probe has to say which.
- The full 22 ms only falls if the whole pass is skipped. That skip needs a
  proof that the document has no seal work: no table box spans a page boundary
  and no split block contributes two or more vertical stubs at a page top
  (`sealPageTopStubs`, `:523`). The generic fixture has one table per page, so
  the condition may hold, but the profile does not record how many seals
  `sealBorderGap` appended. The first probe must count seal appends; if the
  count is zero on this workload, the skip is the real win and needs fixtures
  31, 60, 61, and 62 to keep their current caps. If the count is nonzero, only
  the index work above is safe.

So the honest reading of the 22 ms component for this design note is: 5 to
12 ms reclaimable by intra-call cleanup, up to 22 ms only with a proven
no-seal skip, and no B/op fall from retention on a one-call workload.

### 4.4 First step and probe

First step: stream the end clusters (`sealPageBottomClusters`) and fuse
`capTableMaxPage` into `collectBorderSegmentOps`. Both are single-function
changes with identical inputs.

Probe: a counter on `sealBorderGap` appends must be unchanged on fixtures
31, 60, 61, and 62; a cluster-count equality check for the streamed ends
versus the map in a unit test; scan counters show one op visit instead of
three (max page + count + fill) per `capTablePageBreaks`; `make golden`
byte-identical; the `collectBorderSegmentOps` and `clusterVerticals`
allocation rows in a rate-1 profile are lower or absent.

## 5. PERFT-16d: table pass guard

### 5.1 What the two passes prove

`repeatTableHeaders` (`paint_flow_tables.go:362`) clones each table's thead
range onto every continuation page. `tableBoxes` (`:373-389`) selects tables
with `headerRows > 0 && headerRows < len(rows)`;
`repeatTableHeaderOnPages` (`:395`) computes the set of pages holding body
rows with `headerContinuationPages` (`:616-646`) and appends clones with
`cloneHeaderOps` (`:687`). It proves that every continuation page opens with
the header band. It cannot do anything when no body row sits on a page other
than the table's first body page. Cost 60 ms total, 12 ms/iter.

`normalizeTableRowGaps` (`paint_flow_tables.go:43`), the second table measure
pass, proves that same-page adjacent rows have no stale positive gap. It
measures each row's painted band with `rowPaintBand` (`:124`), called twice per
row pair (`:54`, `:56`), and pulls later same-page rows up with
`shiftTableRowsUp` (`:101-119`). The gap it removes is created when pagination
first moves a row to the next page and a later fixpoint pulls the table back
(header comment `:33-40`). It cannot find a stale gap when no table row has
moved across a page boundary during pagination and no table spans pages. Cost
110 ms total, 22 ms/iter (20 + 90 ms in the two band measurements). The two
passes together are the 12 + 22 = 34 ms the checklist budgets.

One caution: the pass also collapses any positive gap between same-page rows,
not only pagination-created gaps. So the guard must prove "no gap can exist or
be changed", not "no authored gap exists". The safe flag is state-based, not
content-based.

### 5.2 Cheapest flag

Compute `tableSpansPages` in the section 2 census walk: for each table box,
read each row's op range with `rowOpGeometry` (`:297-327`) and record whether
any two rows map to different pages. Add `any cell.paginationShifted`, which
`shiftRowToPage` (`:287`) and `shiftTableRowBoxes` (`:233`) set, and which is
never cleared. Then:

- Skip `repeatTableHeaders` when `!tableSpansPages`.
- Skip `normalizeTableRowGaps` when `!tableSpansPages && !anyRowShifted`.

Both flags are conservative: once any table spans pages or any row moved, the
passes run for the rest of the Paint. On the generic workload there is one
table per page and the counters show no crossings (`cross=0`), so both flags
are false and both passes skip, saving up to 34 ms. The census row walk costs
roughly the same as `rowsIntact` does today (which is why the two should share
one walk), so the probe must show the census total below 34 ms before this
ships.

Reading of the checklist term: PERFT-16 says "guard the table measure pass (up
to 34 ms)". The only 34 ms pair in the pagination profile is
`repeatTableHeaders` 12 plus `normalizeTableRowGaps` 22, and
`normalizeTableRowGaps` is literally the pass that measures row paint bands.
The layout-time `measureTableRows` no-emit pass (`layout_tables.go:656`) is a
different cost (590 ms cum, phase 3 scope) and is not this item.

If the flags cannot be computed cheaply inside the census, the fallback is the
existing per-table `headerContinuationPages` scan, which is the cost being
removed, so there is no point. Do not use `box.y + box.height` page math
alone: fixture 60 documents cell.y drift versus painted ops (`paint_flow_tables.go`
`:433-437`), and `rowYBounds` (`:731-751`) intentionally reads op bands, not
cell.y; `rowCellBounds` (`:762-771`) keeps the op-derived top.

### 5.3 First step and probe

First step: `tableSpansPages` computed in the census walk, skip
`repeatTableHeaders` only. That removes the 12 ms pass. Step 2: the
`normalizeTableRowGaps` skip with the `paginationShifted` term.

Probe: on the generic workload a skip counter shows `repeatHdr` and `normGaps`
skipped; on a document with a genuinely split table (fixtures 60 and 61 and
the `table_thead_gap_test.go` cases), the clone count and the gap-shift count
are identical to the pre-change counters; `make golden` byte-identical;
fixture 56 stays green.

## 6. PERFT-17: apply the forced-break suffix to the live flow index

### 6.1 Current flow and the exact precondition

`beforeAlways` (`paint_flow_breaks.go:499-552`) collects targets, tries
`beforeAlwaysBatch` (`:586-617`), falls back to `beforeAlwaysScan` (`:556-573`)
when the batch shape fails, applies the resulting difference array with
`applySuffixDifferences` (`:459-472`), then does the expensive part:

```go
applySuffixDifferences(res.Ops, suffixDy)   // :547
invalidateFlowIndex(res)                    // :548
ensureFlowIndex(res, contentH)              // :549
```

The rebuild is `90 ms / 5 = 18 ms/iter` and `resetIntBuffer`
(`paint_flow_index.go:75-84`) accounts for 6.50 MB, including the box index
rebuild (`ensureFlowBoxIndex`, `:417`).

The suffix-difference representation is already the right shape for an in-place
update: `applySuffixDifferences` walks ops in index order accumulating
`cum += suffixDy[idx]`, and every op with `cum != 0`, not `Fixed`, and not
`Pinned` moves by exactly `cum`. The fast path must satisfy:

1. **Non-decreasing Y.** All per-target deltas are positive. The batch path
   enforces this: `deltaY < 0` bails to the scan (`:716-718`), and accepted
   deltas are greater than `layoutCoordEpsilon` (`:710-712`). The cumulative
   shift is therefore non-decreasing in op index, so an op's new page is
   greater than or equal to its old page and no op can cross backward over an
   earlier one. The scan path can produce negative deltas and must keep using
   the rebuild.
2. **Single shift.** One application of one difference array, with no reader
   between the deltas being computed and written. `beforeAlwaysBatch` already
   replays boxes (`batch.applyBoxes`, `:612`) in the same call, so the op side
   is the only remaining write.
3. **Delta coverage.** `suffixDy` covers exactly the ops that move: shifted op
   `i` moves by `sum(suffixDy[0..i])`, unshifted ops, `Fixed` ops, and
   `Pinned` ops do not move. The index update must use the same three-way skip
   as `applySuffixDifferences` (`:466`), or membership and Y will disagree.
4. **Live, current index.** `len(res.flowPageOf) == len(res.Ops)` and
   `res.flowPageSize` matches the current content height, and the membership
   must actually match current Y values. The rebuild currently guarantees the
   last part by construction; the fast path has to prove it.

For (4), verify membership in the same pass before trusting it: for each
non-fixed op, compare `res.flowPageOf[i]` with
`flowPageOfY(res.Ops[i].Y, res.flowPageSize, layoutEpsilon)`
(`paint_flow_index.go:528-535`). On the first mismatch, stop and fall back to
`invalidateFlowIndex` + `ensureFlowIndex`, which rebuilds from the current
(partially shifted) coordinates and is still correct. The verification pass
costs one division per op, roughly 2 to 3 ms, against an 18 ms rebuild. If the
index is already nil or the page size differs, skip the fast path immediately.
Boxes do not need the same proof: recompute every box's page from `box.y` when
the box index is live (`len(res.flowBoxPage) == len(res.boxes)`), using the
same page function as `shiftIndexedBox` (`:501-528`), and derive the bucket
membership. A recompute repairs a stale box index instead of trusting it.

The in-place op update reuses the existing primitives: `shiftIndexedOp`
(`:478-504`) already adds a delta, recomputes the page with the same edge bias,
swaps the bucket, and invalidates on an out-of-range result. The fast path can
loop the suffix array and call it, or inline the same three steps.

### 6.2 The fixture-56 failure the phase-6 variant paid for

`plans/0.2.6/perf-improve/results/phase-6/bucketing.md:65-87` records the
rejected live-index epoch experiment. It kept the live fields populated after
mutations, and `snapCrossingTextOps` read a stale bucket: op 8278 was recorded
on page 7 while its Y sat on page 11, so the stale reader shifted it when the
rebuild path would not have. The cascade added 2 header-clone ops and changed
the fixture-56 PDF. The failing test was
`TestFixture56Domain08PageHasProgressAndNoStrayBottomLine`
(`internal/layout/fixture56_chrome_regression_test.go:352`) with
`domain-09 box.y=11607.88 desynced from chrome Y=11617.65`.

This is exactly the risk PERFT-17 reintroduces: once a live membership is
updated in place instead of rebuilt, a later reader (`normalizeTableRowGaps`
reads `res.flowPages` through the row-chrome path, and `shiftFlowY` reads the
buckets) trusts membership that must match Y. The membership verification in
6.1 is the mitigation, and the phase-6 evidence is why the gate must be
"mismatch means rebuild", not "mismatch means repair the one op".

### 6.3 The ownership test that pins it

Add one test, for example
`TestBeforeAlwaysSuffixIndexMatchesRebuild` in `paint_flow_reuse_test.go`:

1. Build a Result with at least two forced breaks and content that forces one
   or more positive shifts (`TestPageBreakBeforeAlways`, `layout_test.go:1210`,
   is the smallest shape; fixture 56 is the domain case).
2. `ensureFlowIndex(res, contentH)` and snapshot the six live fields.
3. Call `beforeAlways(res, contentH)` on the fast path.
4. Assert for every `i`: `res.flowPageOf[i]` equals
   `flowPageOfY(res.Ops[i].Y, res.flowPageSize, layoutEpsilon)`, and
   `res.flowPages[res.flowPageOf[i]][res.flowPos[i]] == i`, and every bucket
   entry agrees with `flowPageOf`. Assert the same for boxes.
5. Equivalence: clone the Result before the call, run the same sequence with a
   test hook that disables the fast path (mirroring `forceBreakBatchDisabled`,
   `paint_flow_breaks.go:457`), and compare every op Y and the full membership.
6. Dirty-index case: move one op's Y behind the index's back, then assert the
   fast path refuses and the rebuild path produced membership.

Existing anchors to keep green: `TestPageIndexRebuildReusesStorage`
(`paint_flow_reuse_test.go:16`), `TestPageIndexScratchReusesStorage` (`:44`),
`TestPagesDoNotAliasPageIndex` (`:63`), and
`TestPaintReleasesPaginationIndexesAfterReaders`
(`paint_release_test.go:24`). The fixture-56 byte compare and `make golden`
are the placement proof.

### 6.4 First step and probe

First step: add the fast path behind a test hook, default off, plus the
membership test. Then flip the default when the equivalence and fixture runs
are green.

Probe: the equivalence test above; `TestFixture56Domain08PageHasProgressAndNoStrayBottomLine`
passes; 500-page `output-bytes 1419234` and page count 500; the rate-1 profile
drops `resetIntBuffer` (6.50 MB) and the `ensureFlowIndex` rebuild row; the
`beforeAlways` counted shifts stay `chg=1 targets=1996 tchg=499`.

## 7. Ranking, batching, and equivalence scope

Ranked by measured payoff over risk. Payoff is the profile number; risk is the
placement surface the change can touch.

| rank | item | payoff | B/op | risk | why |
|---:|---|---:|---:|---|---|
| 1 | PERFT-15 `opInPaintRange` membership table | 24 ms/iter | flat if retained; 22 to 174 KB transient | low | self-contained predicate; no placement semantics changed |
| 2 | PERFT-16a avoid + after census | 26 ms/iter (12 + 14) | small | low | pure style flags; cannot change which box moves, only whether a no-op walk runs |
| 3 | PERFT-16b validate collapse | 8 to 9 ms (checklist) / about 9 ms (profile) | none | low | removes two guard scans; no successful output changes |
| 4 | PERFT-17 suffix applied to the live index | 18 ms/iter | about 6.5 MB | high | the phase-6 stale-index variant changed fixture 56 |
| 5 | PERFT-16d table pass guard | up to 34 ms/iter (12 + 22) | small | medium-high | skip depends on a per-table page-span flag that must stay exact under shifts |
| 6 | PERFT-16c seal segment index | 5 to 12 ms reclaimable of 22 ms; up to 22 with a proven no-seal skip | about 1.3 MB intra-call | medium-high | cap placement is pinned by fixtures 31, 60, 61, 62 |
| 7 | PERFT-16a orphan straddle census and `breakScanState` reuse | 4 ms + up to 5.6 MB | up to 5.6 MB | medium | geometry liveness must track every policy change |

Batching guidance:

- Safe in one change: `opInPaintRange` and the validate collapse. Both are
  scan removals with no effect on successful placement, and one counter run
  plus one golden run proves both.
- Safe in one change: `avoidInside` and `afterBreaks` census flags. They share
  the census walk and are style-only.
- Batch the orphan straddle census with `rowsIntact` (one geometry walk for
  both), but ship it after the style census lands.
- Separate equivalence proofs: the table pass guard (fixture 56 and 60/61
  row semantics), the seal cleanup (fixtures 31/60/61/62 seals), and the
  `beforeAlways` in-place index (fixture-56 byte compare plus the membership
  test). These three must not ride along with the census or with each other.
- Do not batch `opInPaintRange` with the census: the first is one function and
  needs no policy counter, while the census changes when policies run and
  needs the `fpChg` counter evidence. Keeping them apart keeps each probe
  falsifiable.

## 8. Evidence and disclosure

Evidence read (all committed unless noted): `plans/0.2.6/perf-time/phase-wise-checklist.md`,
`plans/0.2.6/perf-time/profiles/pagination-paint-deep.md`,
`plans/0.2.6/perf-time/profiles/time-cpu.md`,
`plans/0.2.6/perf-time/profiles/display-deep.md`,
`plans/0.2.6/perf-time/results/phase-1/budget.md`,
`plans/0.2.6/perf-improve/results/phase-6/bucketing.md`,
`plans/0.2.6/perf-improve/designs/phase-6-bucketing.md`,
`plans/0.2.6/perf-time/profiles/raw/pagination-paint/probe-500-counters.txt`,
`plans/0.2.6/perf-time/profiles/raw/pagination-paint/500-list-capTablePageBreaks.txt`,
`plans/0.2.6/perf-time/profiles/raw/pagination-paint/500-list2-normalizeTableRowGaps.txt`,
`plans/0.2.6/perf-time/profiles/raw/pagination-paint/500-list2-repeatTableHeaders.txt`,
`plans/0.2.6/perf-time/profiles/raw/pagination-paint/500-list-PaintContext.txt`,
and the source files cited inline.

What this note did not do: no benchmark, no `make` target, no profile run, no
code edit, no git command. The 500-page CPU rows are read from the committed
profile, not re-measured. The census cost in section 2, the seal reclaim in
section 4.3, and the fast-path verification cost in section 6.1 are arithmetic
bounds on profile line items, and each has a probe that must be run before the
change ships. The sibling session editing `internal/layout` may move symbols
after this snapshot; re-grep the symbol before applying a diff.
