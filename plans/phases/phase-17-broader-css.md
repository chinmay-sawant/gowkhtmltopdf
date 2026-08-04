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
| `display: flex` | Row/column + align/gap/grow/shrink/basis/order/wrap | **Shipped** (partial) |
| `display: grid` | Columns, spans, nested grids | **Shipped** (lite; not full Grid) |
| `position: fixed` / sticky | Print-safe fixed stamps | fixed lite shipped; sticky **deferred** |

---

## Phase 17 checklist

### 17.1 Float refinement

- [x] Float with percentage widths (`feature/tier-2-pending` / #17)
- [x] Consecutive left/right floats packing (right packing improvement)
- [x] Clearfixes correctly across block boundaries (phase 16 + tests)
- [~] Interaction with tables and lists documented + tested (best-effort; edge cases remain)
- [~] Fixtures for multi-float header/footer chrome (invoice float fixture exists; richer chrome optional)

### 17.2 Position

- [x] `position: relative` + `top`/`right`/`bottom`/`left` offsets
- [x] Stacking: tree order by default; lite `z-index` when set (ops + chrome paint sort)
- [x] Lite `z-index` on positioned boxes
- [x] `position: absolute`: containing block subset; out of flow; `left`/`top`/`right`/`bottom`/`width`/`height` subset
- [x] `position: fixed` lite (stamped on every page in paint)
- [~] `position: sticky` - deferred
- [ ] Matrix §2.2 / fidelity “MVP gap” rows still stale in places — refresh honesty tables

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
- [ ] Matrix: flex still listed as “No” in places — update to Partial with property list

### 17.4 Grid lite (amended into Tier 2)

- [x] `display: grid` / basic column tracks
- [x] Occupancy placement; `grid-column: span N` / start–end
- [x] Nested grids
- [x] Tests: `grid_test.go`, `fixture-28-flex-wrap-grid-fixed`
- [~] Full CSS Grid (areas, dense auto-flow, row spans, `fr` complexity) — **out of scope**

### 17.5 Explicitly not this phase (still deferred)

- [~] Multi-column `column-count`
- [~] Sticky positioning
- [~] Transforms, filters, animations, transitions
- [~] Container queries, `:has()`
- [~] Full flex algorithm (content-based min-size iterations, percentage cyclic sizing)

### 17.6 Fixtures & corpus

- [x] Flex-based report fixtures (`fixture-25`, `fixture-28`)
- [x] Relative/absolute / fixed lite (`fixture-26`)
- [x] Existing golden invoices do not regress (`make test` / samples)

### 17.7 Docs & honesty

- [x] README / PR notes for flex, grid lite, z-index
- [ ] Fidelity guide + compatibility-matrix still claim “no flex/grid” in older rows — **pending doc sync**
- [x] Do **not** claim “full CSS3” or framework support (Bootstrap/Tailwind)

### 17.8 Closure gates

- [x] `make lint` / `make test` (CI on #16 / #17)
- [x] Parent Phase 17 core checked
- [ ] Remaining: matrix/fidelity honesty pass (see Pending)
- [x] Next: Phase 18/19/20 polish; then **Phase 21** when product prioritizes

---

## Pending (after #17)

| Item | Notes |
|------|--------|
| Compatibility-matrix / fidelity MVP-gap rows | Still say flex/grid ignored or thead absent in places |
| Sticky positioning | Explicitly deferred |
| Full Grid / full Flex | Deferred (stdlib report subset only) |
| Richer float+table interaction fixtures | Optional quality |

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
