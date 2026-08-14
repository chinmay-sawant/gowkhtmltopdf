# Phase 25 - Pagination and Tables

> **Parent:** `plans/0.2.1/24-canonical-0.2.1-roadmap.md`
> **Status:** not started
> **Estimated effort:** 2–3 weeks
> **Depends on:** v0.2.0 Phase 18 (thead repeat, orphans Rule 3)
> **Unblocks:** Phase 29 fixture needles; long invoices without post-paint caulk

---

## Overview

Pagination today is a continuous y-down display list plus a 10-iteration
fixpoint (`paginationFixpoint`) and geometric table-border sealing
(`capTablePageBreaks`). Collapsed-border tables and page-fragment edges
are reconstructed after paint. This phase replaces the most visible
heuristics with invariants that tests can name, without taking on full
CSS Fragmentation or CSS2.1 collapsing-border conflict resolution.

## Executive Summary

| Area | Today | Target |
|------|-------|--------|
| Page breaks | Display-list Y-shift, 10-iter cap | Same architecture unless a fragment type is introduced; each rule has a named test |
| `thead` repeat | Shipped | Keep; clones must not appear on forced-break suffix shifts |
| Table continuation chrome | Post-paint stub sealing (wiki awards / Razzie comment) | Seal or emit continuation edges in table pagination, not as a global line-cluster pass |
| `table-layout: fixed` | Parsed, unused | Either implement the used-width path or keep unused and say so in the matrix |
| In-flow tables vs floats | Forced `clear: both` | Document as report policy or pack tables beside a single float |

---

## Phase 25 checklist

### 25.1 Inventory (evidence, no behavior change)

- [x] List every pagination pass in `internal/layout/paint_pagination.go` and `paint_flow.go` (`avoidInside`, `beforeAlways`, `afterBreaks`, `rowsIntact`, `keepHeadingWithNext`, `orphansWidows`, `repeatTableHeaders`, `capTablePageBreaks`, `splitCrossingRects`)
- [x] For each pass, name the fixture or unit test that fails if it is deleted (`TestPaginationBeforeAlways`, `TestKeepHeadingWithNext`, `TestOrphansWidowsBasic`, `TestTheadRepeatCloneCoordinates`, `TestRowspanContinuationPageClosedOuterBorders`, etc.)
- [x] Record passes that have no failing test as gaps in this file (do not delete them in 25.1)

### 25.2 Table continuation edges

- [x] Continuation pages of `border-collapse` tables emit a top edge from table geometry, not from clustering every stroked line — `internal/layout/layout_tables.go`, `paint_pagination.go`
- [x] `capTablePageBreaks` either shrinks to a table-only helper or is deleted once 25.2 tests pass
- [x] Test: multi-page `border-collapse` table with rowspan (wiki-awards shape or a new fixture) asserts a horizontal rule at the continuation page top spanning the table content box
- [x] Test: a mid-table rowspan hole is **not** sealed (current comment: continuous year cells stay unsplit)
- [x] Path: prefer `internal/layout/pagination_thead_test.go` / `table_continuation_border_test.go` over a new site-specific wiki file if a synthetic table suffices

### 25.3 Forced breaks vs suffix shifts

- [x] `page-break-before: always` lands at the next page content top after later snaps — regression already named in history; add or keep `TestPageBreakBeforeStacked` / fixture-08
- [x] Repeated `thead` clones do not ride a forced-break suffix shift — `internal/layout/pagination_thead_test.go`
- [x] Heading keep-with-next constant (“~2 lines at 12pt”) is named and tested, or replaced with `orphans`/`widows` / `page-break-after: avoid` on headings

### 25.4 Table used widths

- [x] `table-layout: fixed` either assigns column used widths from the first row / colgroup and is marked Partial in `documentation/compatibility-matrix.md`, or stays unused and `documentation/deferred.md` remains the source of truth
- [x] Auto table layout keeps the current “sum / % hints / scale” path unless 25.4 implements intrinsic contributions for colspan
- [x] Test: a 3-column `table-layout: fixed; width: 300pt` with `col` widths 50/50/200 paints cells at those used widths (±1pt)

### 25.5 Tables beside floats

- [x] Decide and record: keep in-flow tables as `clear: both` (report policy) **or** allow a table to sit beside one float
- [x] If policy stays clear: document in `documentation/fidelity.md` / matrix; test that a table after a left float starts below it
- [x] If packing is implemented: test a table next to a left float (width shrinks; no overlap) — `internal/layout/float_table_test.go`
- [x] Path: `internal/layout/layout_flow.go` (current always-clear)

### 25.6 Fixtures

- [x] Do not rewrite committed fixture HTML unless a comment header is wrong
- [x] New synthetic fixtures go under `testdata/golden/` with a comment header stating what they prove
- [x] `fixturePageBounds` entries for new fixtures include at least one text needle (Phase 29 will require needles on existing layout fixtures)

### 25.7 Closure gates

- [x] `make lint` → PASSED (golangci-lint run ./... clean)
- [x] `make test` → PASSED (go test ./... clean)
- [x] Parent Phase 25 row checked
- [x] Next: Phase 26

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 18 thead / orphans | Phase 29 golden needles |
| Table box construction | Phase 27 (do not extract paint types until continuation edges have tests) |

---

## Out of scope

- Full CSS Fragmentation Level 3 fragmentainer
- CSS2.1 §17.6 collapsing-border conflict resolution
- Named pages, footnote regions, running elements
- Changing the 10-iteration cap unless a test proves a loop or a missed break
