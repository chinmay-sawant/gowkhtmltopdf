# Phase 4 design: sublinear forced-break shifts (PERF2-17..21)

Read-only design note for `plans/0.2.6/perf-improve/phase-wise-checklist.md`
phase 4. Evidence boundary: current source plus the 2026-09-11 captures under
`plans/0.2.6/perf-improve/profiles/`. No git command, benchmark, or make target
ran while writing this. Every runtime number is quoted from a profile and the
source is named inline.

## 0. What the profiles measured

- `shiftBoxesForForcedBreak` (`internal/layout/paint_flow_breaks.go:418`):
  flat 500 ms, cum 540 ms, 4.17 percent of 12.94 s of 500-page samples
  (`profiles/warm-pdf-cpu.md` section 6).
- The same function through `beforeAlways`: 540 ms flat, 0.54 s inside
  `beforeAlways` 0.77 s (5.95 percent); scaling 0.01 s at 50 pages, 0.05 s at
  100 pages, 0.54 s at 500 pages (`warm-pdf-cpu.md` section 9, candidate 2).
- `settleBeforeAlways` 0.72 s is all `beforeAlways`; `processBeforeAlwaysTarget`
  is 0.62 s and 87 percent of it is the shift call (`warm-pdf-cpu.md` section 4).
- Workload shape: the benchmark template marks every section
  `page-break-before: always` except the first
  (`testdata/golden/benchmarks/templates/report.html.tmpl:14-21`), so 500 pages
  carry 499 targets. The 500-page probe measured 174,000 ops and 54,503 boxes
  (`profiles/warm-pdf-alloc.md` section 5). A full scan per changed target is
  499 x 54,503 = 27.2M visits in the worst case. Whether the run is close to
  that worst case is a hypothesis; the counters in section 4 decide it.

## 1. Chain inventory

`beforeAlways` is the only caller of the per-target box scan. The `shiftFlow*`
family is the sibling policy path (avoid-inside, heading keep, orphans,
after-breaks, snapping) and is listed because the plan groups them; it iterates
page buckets, not the flat box list, so it is not the quadratic site.

| Function | file:line | What it iterates |
|---|---|---|
| `paginateOps` | `paint_pagination_fixpoint.go:22` | Driver. `ensureFlowIndex` (:27), `settleBeforeAlways` (:32), `keepImplicitAsides` loop (:39), `snapCrossingTextOps` (:49), `paginationFixpoint` (:53), `repeatTableHeaders` (:60), `normalizeTableRowGaps` (:64), `normalizeLeadingRoundedCallouts` (:67), second `settleBeforeAlways` (:70) |
| `settleBeforeAlways` | `:80` | Up to 10 calls to `beforeAlways` per invocation; called twice per `paginateOps` |
| `paginationFixpoint` | `:98` | Up to 10 iterations; each calls `beforeAlways` (:105) plus `avoidInside`, `afterBreaks`, `rowsIntact`, `keepHeadingWithNext`, `orphansWidows` |
| `beforeAlways` | `paint_flow_breaks.go:466` | `flowBoxList` once (:471); `collectBeforeAlwaysTargets` (:473); target loop (:482); `applySuffixDifferences` over all ops (:492); `invalidateFlowIndex` then `ensureFlowIndex` (:493-494) |
| `collectBeforeAlwaysTargets` | `:441` | `collectBeforeAlwaysBoxes` DOM preorder (:603), then `beforeAlwaysOpStart` (:579) per target, which can scan the box list suffix; stable sort by `start` (:452-456) |
| `newBreakScanState` | `:383` | Allocates `events` capacity targets and `suffixDy` length ops+1. Measured 5,570,560 B at 500 pages for `suffixDy`, 4 states per conversion (`warm-pdf-alloc.md` section 6) |
| `breakScanState.advance` | `:394` | Incremental op walk from the last `opIdx` to the target start; O(ops) total per `beforeAlways` call |
| `processBeforeAlwaysTarget` | `:499` | `state.advance` (:516), `forcedBreakTargetY` (:543), `state.applyBreak` (:534), `shiftBoxesForForcedBreak` (:535) |
| `shiftBoxesForForcedBreak` | `:418` | All boxes per changed target. Condition `b == targetBox || b.y > fromY || (b.y == fromY && b.opStart >= start)` at :420; `b.y += deltaY` at :421 |
| `applySuffixDifferences` | `:426` | All ops once; prefix sum of `suffixDy`; skips `Fixed` and `Pinned` (:433) |
| `shiftFlowY` | `paint_flow_index.go:17` | Wrapper: `shiftFlowBounded(res, from, to, fromY, +Inf, deltaY)`. Not called by `beforeAlways` |
| `shiftFlowBounded` | `:25` | `ensureFlowIndex` (:30), `shiftOpsRange` over the `[from,to]` op range (:39), `shiftFlowOps` over op page buckets (:41), `shiftFlowBoxes` over box page buckets (:51) |
| `shiftFlowOps` | `:96` | Page buckets from `startPage`, direction by sign of `deltaY`; `shiftOpsBucket` (:117) walks each bucket with swap-remove |
| `shiftFlowBoxes` | `:151` | Box page buckets from `startPage`; `shiftBoxesBucket` (:167); `skipBoxShift` (:202) predicate |

## 2. Invariants a faster scan must preserve

I1. Exact shift predicate. For a changed target `t`, a box `b` moves by
`deltaY_t` iff at that moment
`b == target.box || b.y > fromY_t || (b.y == fromY_t && b.opStart >= start_t)`,
where `fromY_t` is `target.box.y` before the shift. A faster scan must produce
the identical final Y for every box, not a close one. The plan gate is
identical final placements (PERF2-18, PERF2-19).

I2. Target order. Targets are processed in ascending `start` after the stable
sort at `paint_flow_breaks.go:452-456`. Every later target is shifted by every
earlier event: its `y` is at or below the earlier one, and when the `y` values
are equal the op-start ordering satisfies the boundary clause. This is what
makes a batched scan possible and must be re-verified before using it.

I3. Op and box shifts are separate. Ops move once through the `suffixDy`
difference array in `applySuffixDifferences` (`:426-439`), which skips `Fixed`
and `Pinned`. Boxes move through the per-target scan. A faster box scan must
not change op Y, and the two must stay coupled to the same `deltaY_t` events.

I4. Flow index invalidation. `beforeAlways` invalidates and rebuilds only when
something changed (`:488-494`). `shiftFlowBounded`, `shiftOpsOnly`, and
`normalizeTableRowGaps` depend on `res.flowPages`/`flowPageOf` describing the
current Y values. The invalidation point and the `flowPageSize` stamp must not
move.

I5. Pinned and fixed. Thead clones are appended at the end of the list and
`Pinned` (`paint_flow_tables.go:696-705`); `applySuffixDifferences` must not
move them. The box scan touches boxes, not those clones, but the note in
`paint_flow_tables.go:684` is the contract this phase must keep.

I6. No global Y ordering guarantee. `flowBoxList` is `flattenBoxes` preorder
(`layout.go:1222-1231`). Absolute, fixed, float, and negative-margin boxes can
break non-decreasing `y`. Any scan that assumes sorted Y must verify it and
fall back.

I7. Boundary equality is semantic. Equal-Y handling carries fixture work:
`fixture-31` Row 28 white background (`paint_pagination_fixpoint.go:223`),
`fixture-60` page-2 thead overlap (`paint_flow_tables.go:74`), and the
`fixture-21/28` empty-break-marker rule (`beforeAlwaysOpStart`, `:576-599`).
The bulk path must not widen or narrow the boundary set.

Fixtures and tests that pin the behavior:

| Pin | Where |
|---|---|
| Golden fixtures 62, 31, 60 | `testdata/golden/fixture-62-implemented-props-c.html`, `fixture-31-sticky-top.html`, `fixture-60-implemented-props-a.html`; plan PERF2-19 |
| fixture-62 transform restamp | `internal/layout/table_transform_center_test.go:41`, `writing_mode_width_test.go:10` |
| fixture-31 sticky and orphan rows | `internal/layout/sticky_test.go:448-952` |
| fixture-60 thead and seals | `internal/layout/table_thead_gap_test.go:18-53`, `fixture60_page_spill_test.go:15`, `fixture60_border_test.go:13` |
| `beforeAlways` driven directly | `internal/layout/fixture56_renderer_test.go:966-999` |
| Flow index maintenance | `internal/layout/architecture_followup_test.go:321-387` |
| Golden corpus and pagination suites | `make golden`, `go test ./internal/layout -run 'Test.*(Pagination\|Paginate\|Table.*Continuation\|FlowPage\|PaginateOps)' -count=1` |

## 3. Approaches

### Approach A (recommended): one-pass batch with nested-suffix prefix sums

Key property. Targets ascend in `start` and, under I2, every later target is
inside every earlier event's affected set. Therefore the affected box sets are
nested suffixes (strict `y > fromY`), and every box inside the current suffix
has received the same accumulated delta `D[t-1]`. Under a Y-sorted box list,
the strict suffix boundary for event `t` is found by comparing original Y:
within the unprocessed suffix, the common `D[t-1]` cancels, so
`first index with original y > original target y` is the split. That split
moves monotonically forward. The target loop already defers op Y writes
(`applySuffixDifferences` runs last), so the state machine needs no change.

Precondition check, once per `beforeAlways` call, O(boxes):
`boxes[i].y <= boxes[i+1].y` for all i, and target Ys non-decreasing. If either
fails, run today's full scan and return.

Data, scoped to one `beforeAlways` call (stack or `breakScanState` scratch, no
new per-call allocation):
- `D []float64`, prefix sums `D[t] = D[t-1] + deltaY_t`, length events+1.
- `lo []int32`, `lo[t]` = first box index with original `y > original target y`.
- `extra []float64`, length boxes or a side list, accumulated deltas for the
  equal-Y boundary boxes only.

Pseudocode:

```go
if !boxesNonDecreasing(boxes) || !targetYsNonDecreasing(targets) {
    runExistingPerTargetScan(...) // unchanged semantics
    return
}

for _, target := range targets {
    boxY := target.box.y + D[len(D)-1] // all earlier events shifted every later target
    state.advance(ops, start)
    targetY, fresh := forcedBreakTargetY(boxY, state.maxEff, contentH)
    if fresh || abs(targetY-boxY) <= layoutCoordEpsilon { continue }
    delta := targetY - boxY
    state.applyBreak(start, delta, opCount)
    D = append(D, D[len(D)-1]+delta)
    lo = append(lo, firstIndexGreater(boxes, target.box.y))
    applyBoundaryRun(target, delta, D, lo, extra) // exact predicate, equal-y run
}

k := 0
for i := range boxes {
    for k < len(lo) && int(lo[k]) <= i { k++ }
    boxes[i].y += D[k] + extra[i]
}
```

Boundary run. The equal-Y run around the target's original Y is contiguous in
the sorted list. Each event evaluates the exact I1 predicate on the run using
the box's virtual Y, which is
`original y + D[number of earlier strict events covering it] + extra[i]`.
An earlier strict event covers a run box exactly when that earlier target's
original Y was strictly smaller (then its split `lo` is below the run), so the
count is known from the recorded targets. Cap the run at a fixed size (for
example 64 boxes); if a run is larger, fall back to the existing full scan for
that call. Runs are normally 1 to 3 boxes (a section and a co-located sibling).

Cost. O(ops + boxes + targets + run work) per `beforeAlways` call. At 500 pages
that is roughly 174,000 + 54,503 + 499 instead of up to 27.2M box visits.

Failure modes and guards:
- Precondition wrongly trusted (unsorted boxes): a box with `y > fromY` before
  the split is never moved. Golden page counts and ordered text fail, but only
  for fixtures that exercise the shape. Guard: the O(boxes) check above.
- Boundary set widened or narrowed: placements change under the exactness gate.
  Guard: table-driven unit tests with duplicate-Y boxes, plus fixtures 31, 60,
  62 and the `make golden` corpus.
- Negative `deltaY`: nesting still holds (the proof is in target order, not in
  the sign), but the sorted-list invariant can be locally broken after moving
  a boundary subset. The batch does not need sortedness after the last event;
  the next `beforeAlways` call re-checks it. Guard: re-order detection in the
  precondition stays, and no cursor is carried across calls.

### Approach B (conservative): Y-order index with a moving start cursor

Build `boxYOrder []int` over `res.boxes` with `sort.SliceStable` by `y` at the
start of each `beforeAlways` call (or cache on `Result` keyed by a box-Y
epoch). For each changed target, binary search the first entry with
`y > fromY_t`, then run the existing exact predicate only on
`[lo, len(boxYOrder))`. Keep `lo` monotone across targets. Reset the cursor to
zero whenever `deltaY < 0` or the boundary run moved a box.

Cost. Visits fall from all boxes per event to the strictly-below suffix. On the
500-page benchmark that is about half the visits, roughly 0.25 s, which is
about 2 percent of the 12.94 s profile and below the phase's 4 percent gate.
It is a fallback for documents where Approach A cannot pass the precondition,
not the primary fix.

Failure modes: a box moved by a negative or boundary shift sits before the
cursor and is missed; a stale cached order after a direct `box.y` write misses
a moved box. Guards: cursor reset on negative deltas and boundary moves, no
cursor reuse across calls, and full-scan fallback when the order is stale.

## 4. Counters and measurement

Test-only counters, no production cost when the hook is nil. Add to
`paint_flow_breaks.go`:

```go
type forceBreakStats struct {
    calls, targets, changed int // beforeAlways invocations, targets seen, targets moved
    visits, moves           int // shiftBoxesForForcedBreak loop iterations / condition true
    boxCount, opCount       int
}
var forceBreakHook func(forceBreakStats) // test hook, nil in production
```

- Increment in the existing loops; call the hook once at the end of
  `beforeAlways` only when it is non-nil.
- Probe test `TestForceBreakCounters500` in package `layout`: build a synthetic
  500-section document with `page-break-before: always` and the per-page shape
  of `report.html.tmpl` (h1, p, 20-row table), run `LayoutContext` and
  `PaintContext`, and log the stats. Assert the pre-change quadratic shape
  (`visits` is a large multiple of `boxCount`) and, after the change, that
  `visits <= opCount + 4*boxCount + targets` and `changed` is unchanged.

Falsifiable measurement, warm 500 pages (commands copied from
`warm-pdf-cpu.md` section 9 header):

```sh
go test -c -o /tmp/opencode/perf-improve/convert.test ./internal/convert
/tmp/opencode/perf-improve/convert.test -test.run '^$' \
  -test.bench '^BenchmarkPDFPages$/^generic$/^500Pages$' \
  -test.benchtime=1x -test.count=5 -test.benchmem \
  -test.cpuprofile=/tmp/opencode/perf-improve/500-cpu-after.pprof
go tool pprof -list 'shiftBoxesForForcedBreak|beforeAlways' \
  /tmp/opencode/perf-improve/convert.test /tmp/opencode/perf-improve/500-cpu-after.pprof
```

Targets: `beforeAlways` cum below 0.3 s at 500 pages and
`shiftBoxesForForcedBreak` out of the top 60; warm 500-page time at least
4 percent below the phase-1 baseline with `paginateOps` share down from
12.75 percent (PERF2-20); `make golden` 65/65 and the pagination suites exit 0.

## 5. Ranked recommendation and smallest first step

1. Approach A, the one-pass batch with the sortedness precondition and the
   equal-run fallback. It is the only design that removes the
   targets x boxes product. Risk is concentrated in the boundary run and the
   precondition, both testable.
2. Approach B if Approach A cannot pass the precondition on fixtures 62, 31,
   and 60. Do not claim the 4 percent gate with B alone; re-scope the gate to
   the measured half of the flat time instead.
3. Reuse `suffixDy` and `events` across the four `newBreakScanState`
   allocations (5,570,560 B at 500 pages, `warm-pdf-alloc.md` rank 6). This is
   an allocation item, not the CPU lever; keep it out of the phase gate.

Smallest first step: land only the counters and `TestForceBreakCounters500`,
record the 500-page counts, and confirm the quadratic shape. No production
algorithm change before those counts exist.
