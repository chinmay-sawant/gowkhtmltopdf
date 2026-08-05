# Tier 2 Subplan - Full Flexbox & Full CSS Grid (separate ledger)

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Status:** not started (large; track separately from sticky / Tier 2 pending close)  
> **Estimated effort:** 2–6 months staged  
> **Constraint:** stdlib-only layout engine; no browser embed  
> **Specs:** [CSS Flexbox L1](https://www.w3.org/TR/css-flexbox-1/), [CSS Grid L1](https://www.w3.org/TR/css-grid-1/)

---

## Overview

Grow today’s **report-friendly** `flex.go` / `grid.go` subsets toward **full**
Flexbox and Grid algorithms suitable for most marketing/report layouts. This
is intentionally a **separate** subplan so it does not block sticky, shaping,
or image-mode `@font-face`.

Honesty: “full” here means **spec-complete enough for real sites’ static CSS**,
not Chrome layout-test pixel parity.

## Executive Summary

| Area | Today (lite) | Target stages |
|------|--------------|---------------|
| Flex | row/column subset, wrap, grow/shrink, gap shared | Complete Flexbox L1 used properties |
| Grid | columns, span, nested, auto-flow row | Rows, areas, dense, alignment, `fr` resolution |
| Tests | fixture-25/28 + unit | Spec-driven fixtures per stage |

---

## Phase 1: Inventory & gates

### 1.1 Gap list vs code (evidence)

- [ ] Diff `internal/layout/flex.go` / `style.go` against Flexbox L1 property list;
      mark Implemented / Partial / Missing in this file’s appendix
- [ ] Diff `internal/layout/grid.go` against Grid L1; note `GridTemplateRows`
      stored but unused
- [ ] Freeze “Stage A done” definition (below) before coding Stage B

### 1.2 Staging policy

- [ ] Land **Stage A** (deepen current subset) before Stage B (new algorithms)
- [ ] Each stage: `make lint` + `make test` + new fixtures; no drive-by refactors

---

## Phase 2: Stage A — Flex deepen (stdlib)

### 2.1 Flex container

- [ ] Independent `row-gap` / `column-gap` (stop collapsing to single `Gap` if still shared)
- [ ] `flex-direction: row-reverse` / `column-reverse`
- [ ] `justify-content: space-around` / `space-evenly`
- [ ] `align-content` for multi-line wrap
- [ ] Column path: honor grow/shrink/justify/align (parity with row)
- [ ] Path: `internal/layout/flex.go`, `style.go`

### 2.2 Flex items

- [ ] Shorthand `flex: grow shrink basis`
- [ ] `align-self`
- [ ] Content-based **min-size** contribution (spec §4.5 lite — at least
      non-zero min-content for text)
- [ ] Percentage basis cyclic sizing: document + implement subset or honesty

### 2.3 Flex Stage A proof

- [ ] New fixtures (do not edit 25/28): wrap + reverse + space-evenly + column grow
- [ ] `go test ./internal/layout -run Flex -count=1`
- [ ] `make lint` / `make test`

---

## Phase 3: Stage B — Grid deepen (stdlib)

### 3.1 Tracks & placement

- [ ] Consume `grid-template-rows` in layout (currently unused)
- [ ] `grid-row` / `grid-row-start` / `grid-row-end` / row `span`
- [ ] `fr` distribution with min/max constraints (definite container width/height)
- [ ] `grid-template-areas` + `grid-area` name placement
- [ ] `grid-auto-flow: dense` (optional Stage B2)
- [ ] Path: `internal/layout/grid.go`, `style.go`

### 3.2 Alignment

- [ ] `justify-items` / `align-items` on grid; `justify-self` / `align-self` on items
- [ ] Gap already present — verify row/column independence

### 3.3 Grid Stage B proof

- [ ] Fixtures: areas, row span, `fr` + minmax lite
- [ ] `go test ./internal/layout -run Grid -count=1`
- [ ] `make lint` / `make test`

---

## Phase 4: Stage C — Hard Flex/Grid edges (optional)

### 4.1 Only if product still wants “full”

- [ ] Intrinsic sizing passes / nested percentage cycles
- [ ] Subgrid — **likely `[~]` forever** under report engine
- [ ] Masonry — out of scope

### 4.2 Explicit deferrals

- [~] Subgrid
- [~] Chrome layout-test parity
- [~] Container queries / `:has()` (separate CSS epic)

---

## Phase 5: Docs & closure

### 5.1 Honesty

- [ ] Matrix Flex/Grid rows: expand property tables per stage
- [ ] Phase 17 “full flex/grid” `[~]` → point here; flip stages to `[x]` as proven
- [ ] Do **not** claim Bootstrap/Tailwind support

### 5.2 Closure

- [ ] Stage A complete → record date
- [ ] Stage B complete → record date
- [ ] Stage C `[~]` or complete with listed gaps
- [ ] Next: Phase 21 site corpus stress against new layout

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 17 lite flex/grid | Starting code |
| Sticky subplan (orthogonal) | May parallelize |
| Paint / pagination | Fragmented flex/grid items |

---

## Out of scope

- JavaScript-driven layout
- CSS Houdini / custom layout
- Pixel-diff vs Chrome
