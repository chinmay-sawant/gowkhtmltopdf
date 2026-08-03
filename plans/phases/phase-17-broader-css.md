# Phase 17 - Broader CSS: Position / Float Refinement / Partial Flex

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** not started  
> **Estimated effort:** 2–4 months  
> **Depends on:** Phase 16 float lite + selectors  
> **Unblocks:** Phase 21 arbitrary websites; Tier 2 “leave wkhtmltopdf for most jobs”  
> **Tier:** 2 #6 · **Constraint:** stdlib-only; no full browser CSS

---

## Overview

Expand layout beyond invoice float-lite toward **partial flex**, refined **position**, and remaining float edge cases. Grid remains out of scope unless product amends. Goal: enough CSS for many report and simple marketing layouts - not Wikipedia chrome parity.

## Executive Summary

| Feature | Target |
|---------|--------|
| Float refinement | Nested floats, percentage widths, better wrapping |
| `position: relative` | Offset without leaving flow |
| `position: absolute` | Containing block subset if reports need overlays |
| `display: flex` | Row direction + basic alignment/gap |
| `display: grid` | **Deferred** (allowlist stays out) |

---

## Phase 17 checklist

### 17.1 Float refinement

- [ ] Float with percentage widths
- [ ] Consecutive left/right floats packing
- [ ] Clearfixes correctly across block boundaries
- [ ] Interaction with tables and lists documented + tested
- [ ] Fixtures for multi-float header/footer chrome

### 17.2 Position

- [ ] `position: relative` + `top`/`right`/`bottom`/`left` offsets applied at paint or layout
- [ ] Stacking: document z-order rules (simple tree order if no z-index)
- [ ] `[~]` `z-index` - only if needed
- [ ] `position: absolute`: containing block = nearest positioned ancestor or initial; out of flow
- [ ] Absolute with `left`/`top`/`width`/`height` subset for badges/watermarks
- [ ] `[~]` `position: fixed` - page-fixed in print is hard; prefer CLI HF; implement only with clear print semantics
- [ ] Matrix §2.2 updates

### 17.3 Partial flexbox (report-friendly)

- [ ] Parse `display: flex` / `inline-flex` into layout mode (not ignore → inline)
- [ ] Default `flex-direction: row`
- [ ] `flex-direction: column` basic
- [ ] `justify-content`: flex-start | flex-end | center | space-between (subset)
- [ ] `align-items`: stretch | flex-start | center | flex-end (subset)
- [ ] `gap` / `row-gap` / `column-gap` simple lengths
- [ ] Children: honor `flex-grow` 0/1 basic; `[~]` full flex algorithm deferred
- [ ] `[~]` `flex-wrap: wrap` - stage 2 if needed
- [ ] `[~]` `order` - optional
- [ ] Tests: 3-column metric row; header bar with space-between
- [ ] Path: new flex layout path in `internal/layout/`
- [ ] Matrix: flex Partial with listed properties only

### 17.4 Explicitly not this phase

- [~] CSS Grid / subgrid  
- [~] Multi-column `column-count`  
- [~] Sticky positioning  
- [~] Transforms, filters, animations, transitions  
- [~] Container queries, `:has()`  

### 17.5 Fixtures & corpus

- [ ] Add flex-based report fixture (no external CSS frameworks)
- [ ] Add relative/absolute badge fixture
- [ ] Ensure existing golden invoices do not regress

### 17.6 Docs & honesty

- [ ] Fidelity guide: “partial flex” definition table
- [ ] README deferred floats/flex rows updated
- [ ] Do **not** claim “full CSS3” or framework support (Bootstrap/Tailwind)

### 17.7 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Parent Phase 17 checked
- [ ] Next: **Phase 18** pagination and/or **19** fonts by need

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 16 | Phase 21 website layouts |
| Paint display list | Absolute positioned paint order |

---

## Out of scope

- Years of browser float/flex edge cases
- Pixel parity with Chrome layout tests
