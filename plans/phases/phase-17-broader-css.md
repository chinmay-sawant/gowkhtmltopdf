# Phase 17 - Broader CSS: Position / Float Refinement / Partial Flex / Grid Lite

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** done (core + pending polish) on `master` via #16 / #17  
> **Estimated effort:** 2–4 months  
> **Depends on:** Phase 16 float lite + selectors  
> **Unblocks:** Phase 21 arbitrary websites; Tier 2 “leave wkhtmltopdf for most jobs”  
> **Tier:** 2 #6 · **Constraint:** stdlib-only; no full browser CSS

---

## Overview

Expand layout beyond invoice float-lite toward **partial flex**, **grid lite**,
refined **position**, and remaining float edge cases. Goal: enough CSS for many
report and simple marketing layouts - not Wikipedia chrome parity.

## Executive Summary

| Feature | Target | Status (2026-08-05) |
|---------|--------|---------------------|
| Float refinement | Nested floats, `%` widths, better wrapping | **Shipped** (lite) |
| `position: relative` / `absolute` | Offsets + containing-block subset | **Shipped** (lite) |
| `z-index` | Paint sort on positioned / chrome ops | **Shipped** (lite) |
| `display: flex` | Row/column + align/gap/grow/shrink/basis/order/wrap + Stage A deepen | **Shipped** (Stage A; see flex-grid-full) |
| `display: grid` | Columns/rows, spans, nested, `fr`, alignment, Stage C lite | **Shipped** (Stage B + Stage C lite; not full Grid L1/L3) |
| `position: fixed` / sticky | Print-safe fixed stamps + sticky | fixed lite shipped; sticky **print-scoped** (page scrollport) |

---

## Phase 17 checklist

### 17.1 Float refinement

- [x] Float with percentage widths (`feature/tier-2-pending` / #17)
- [x] Consecutive left/right floats packing (right packing improvement)
- [x] Clearfixes correctly across block boundaries (phase 16 + tests)
- [x] Interaction with tables documented + tested (fixture-29 float-beside-table; best-effort)
- [~] Float-inside-table-cell / table-inside-float packing edge cases (still open)
- [x] Fixtures for float beside table (`fixture-29-float-beside-table.html`; richer chrome still optional beyond 22/29)

### 17.2 Position

- [x] `position: relative` + `top`/`right`/`bottom`/`left` offsets
- [x] Stacking: tree order by default; lite `z-index` when set (ops + chrome paint sort)
- [x] Lite `z-index` on positioned boxes
- [x] `position: absolute`: containing block subset; out of flow; `left`/`top`/`right`/`bottom`/`width`/`height` subset
- [x] `position: fixed` lite (stamped on every page in paint)
- [x] `position: sticky` — print-scoped full sticky ([`subplans-tier-2/sticky-print.md`](subplans-tier-2/sticky-print.md); page content box = scrollport; fixture-31)
- [x] Matrix §2.2 / fidelity “MVP gap” rows — refreshed via shared doc-honesty pass

### 17.3 Partial flexbox (report-friendly)

- [x] Parse `display: flex` / `inline-flex` into layout mode
- [x] Default `flex-direction: row`
- [x] `flex-direction: column` basic
- [x] `justify-content`: flex-start | flex-end | center | space-between (subset)
- [x] `align-items`: stretch | flex-start | center | flex-end (subset)
- [x] `gap` / `row-gap` / `column-gap` simple lengths
- [x] Children: `flex-grow`, `flex-shrink`, `flex-basis` (%/length)
- [x] Post grow/shrink **min/max-width clamp**
- [x] `flex-wrap: wrap` / `wrap-reverse` (basic)
- [x] `order`
- [x] Tests: flex fixtures (`fixture-25`, `fixture-28`) + layout unit tests
- [x] Path: `internal/layout/flex.go`
- [x] Matrix: flex → Partial with property list (shared doc-honesty pass)
- [x] Stage A deepen (reverse, space-around/evenly, align-content/self, flex shorthand, independent gaps, `%` basis cyclic subset) — **2026-08-05**; see [`subplans-tier-2/flex-grid-full.md`](subplans-tier-2/flex-grid-full.md)

### 17.4 Grid lite (amended into Tier 2)

- [x] `display: grid` / basic column tracks
- [x] Occupancy placement; `grid-column: span N` / start–end
- [x] Nested grids
- [x] Tests: `grid_test.go`, `fixture-28-flex-wrap-grid-fixed`
- [x] Stage B (template-rows, row span, `fr`/`minmax`, areas, dense, justify/align stretch) — **2026-08-05**; see [`subplans-tier-2/flex-grid-full.md`](subplans-tier-2/flex-grid-full.md)
- [x] Stage C lite (cyclic `%` subset; subgrid copy-inherit; masonry pack) — [`subplans-tier-2/flex-grid-full.md`](subplans-tier-2/flex-grid-full.md); fixture-35
- [~] Full CSS Grid L1/L3 + Chrome parity — true shared-track subgrid / full L3 masonry / pixel parity still deferred

### 17.5 Explicitly not this phase (still deferred)

- [~] Multi-column `column-count`
- [x] Sticky positioning — print-scoped ([`subplans-tier-2/sticky-print.md`](subplans-tier-2/sticky-print.md))
- [~] Transforms, filters, animations, transitions
- [~] Container queries, `:has()`
- [~] Full flex algorithm (multi-pass intrinsic iterations beyond cyclic `%` subset) — see [`subplans-tier-2/flex-grid-full.md`](subplans-tier-2/flex-grid-full.md) Stage C

### 17.6 Fixtures & corpus

- [x] Flex-based report fixtures (`fixture-25`, `fixture-28`)
- [x] Relative/absolute / fixed lite (`fixture-26`)
- [x] Existing golden invoices do not regress (`make test` / samples)

### 17.7 Docs & honesty

- [x] README / PR notes for flex, grid lite, z-index
- [x] Fidelity guide + compatibility-matrix Partial flex/grid wording (shared doc-honesty pass)
- [x] Do **not** claim “full CSS3” or framework support (Bootstrap/Tailwind)

### 17.8 Closure gates

- [x] `make lint` / `make test` (CI on #16 / #17)
- [x] Parent Phase 17 core checked
- [x] Matrix/fidelity honesty pass (shared doc-honesty)
- [x] Next: Phase 18/19/20 polish; then **Phase 21** when product prioritizes

---

## Pending (after #17)

> **Execution subplan:** [`subplans-tier-2/phase-17-pending.md`](subplans-tier-2/phase-17-pending.md)  
> **Shared doc honesty:** [`subplans-tier-2/00-shared-doc-honesty.md`](subplans-tier-2/00-shared-doc-honesty.md) — **done** for matrix/fidelity

| Item | Notes |
|------|--------|
| Compatibility-matrix / fidelity MVP-gap rows | **[x]** Shared doc-honesty pass |
| Sticky positioning | **[x]** print-scoped ([`sticky-print.md`](subplans-tier-2/sticky-print.md); fixture-31) |
| Full Grid / full Flex | Stage A + Stage B + Stage C lite **done** 2026-08-05; remaining `[~]` = true shared-track subgrid / full L3 masonry / Chrome parity → [`subplans-tier-2/flex-grid-full.md`](subplans-tier-2/flex-grid-full.md); next **Phase 21** corpus |
| Richer float+table interaction fixtures | **[x]** `fixture-29-float-beside-table.html` (do not edit 22); cell packing edges still `[~]` |

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 16 | Phase 21 website layouts |
| Paint display list | Absolute / fixed / z-index paint order |

---

## Out of scope

- Years of browser float/flex/grid edge cases
- Pixel parity with Chrome layout tests
