# Tier 2 Subplan - Phase 17 Pending (Broader CSS honesty + optional float/table)

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md) — Pending (after #17)  
> **Status:** docs honesty done (shared pass); optional float fixture shipped  
> **Estimated effort:** 0.5 day docs (via shared pass) + 0–2 days optional fixtures  
> **Depends on:** [00-shared-doc-honesty.md](00-shared-doc-honesty.md) for matrix/fidelity  
> **Constraint:** stdlib-only; no full browser CSS

---

## Overview

Phase 17 **core is shipped** (partial flex, grid lite, position/float/z-index). Remaining
work is (1) documentation honesty so matrix/fidelity stop claiming “no flex/grid”,
and (2) optional quality fixtures for float↔table edges. Sticky and full Grid/Flex
stay deferred.

## Executive Summary

| Pending item (parent) | Disposition | Primary work |
|-----------------------|-------------|--------------|
| Compatibility-matrix / fidelity MVP-gap rows | **Must** | Shared doc-honesty Pass 0 |
| Sticky positioning | `[~]` deferred | Honesty only: sticky ≡ relative-offset |
| Full Grid / full Flex | `[~]` deferred | Document supported subset; no algorithm expansion |
| Richer float+table interaction fixtures | **New** `fixture-29-float-beside-table.html` (do not edit fixture-22) |

---

## Phase 1: Evidence baseline (already scanned 2026-08-05)

### 1.1 Shipped paths (do not re-implement)

- [x] Flex: `internal/layout/flex.go` + `flex_test.go`; fixtures 25, 28
- [x] Grid lite: `internal/layout/grid.go` + `grid_test.go`; fixture 28
- [x] Position: `layout.go` `buildAbsolute`/`buildFixed`/`applyRelativeOffset`; fixture 26/28
- [x] z-index lite: `paint.go` `sortPaintIndices`; `TestZIndexPaintOrder`
- [x] Float lite: `float.go` + `TestFloatWidthPercent`; fixture 22

### 1.2 Supported property subset (honesty bullets for matrix)

**Flex (Partial)** — parsed + used:
- `display: flex | inline-flex`
- `flex-direction: row | column` (no `*-reverse`)
- `flex-wrap: nowrap | wrap | wrap-reverse`
- `justify-content: flex-start | flex-end | center | space-between | start | end`
- `align-items: stretch | flex-start | flex-end | center | start | end` (stretch does not grow height)
- `gap` / `row-gap` / `column-gap` → single shared `Gap`
- `flex-grow`, `flex-shrink`, `flex-basis` (length / `%` / `auto`); min/max-width clamp
- `order`
- Column path: order + gap only (no grow/shrink/justify on column)

**Not flex:** shorthand `flex:`; `align-self`; `align-content`; content-based min-size iterations

**Grid lite (Partial):**
- `display: grid | inline-grid`
- `grid-template-columns` (lengths, `Nfr`, `repeat(n, …)`)
- shared `gap`
- `grid-column` / start / end (`span N`, `start / end`)
- nested grids; auto-flow row occupancy
- `grid-template-rows` stored but **unused** in layout

**Not grid:** areas, dense auto-flow, row spans, `grid-row*`, justify/align on grid, named lines

**Position:**
- relative / absolute / fixed lite consumed
- sticky parsed but **aliased to relative offsets** (`layout.go` treats `sticky` like relative) — not true sticky pagination

---

## Phase 2: Documentation honesty (owned by shared pass)

### 2.1 Defer to shared matrix rewrite

- [x] Land [00-shared-doc-honesty.md](00-shared-doc-honesty.md) Phase 2.2 items (position/flex/grid/§5)
- [x] Land shared fidelity map + overview/README overview updates for float/flex/position/grid
- [x] Flip parent checklist rows:
  - [x] “Matrix §2.2 / fidelity MVP-gap”
  - [x] “Matrix: flex still listed as No”
  - [x] “Fidelity guide + compatibility-matrix still claim no flex/grid”
- [x] Proof: `rg -n 'Flexbox / Grid|position: fixed / absolute|No floats; no flex' documentation/` shows Partial/shipped wording only

### 2.2 Sticky honesty sentence

- [x] Matrix/README: explicit “sticky ≈ relative offsets; no page-edge stickiness”
- [x] Cite `applyRelativeOffset` + lack of `TestSticky`
- [x] Keep parent `[~] position: sticky - deferred`

---

## Phase 3: Deferred full Flex / Grid (confirmation only)

### 3.1 Record non-goals in checklist

- [x] Confirm full CSS Grid (areas, dense, row spans, complex `fr`) remains **out of scope**
- [x] Confirm full flex algorithm (cyclic %-sizing, content-based min-size) remains **out of scope**
- [x] Confirm multi-column `column-count`, transforms, container queries remain deferred
- [x] No code change required — honesty + parent `[~]` rows sufficient

---

## Phase 4: Optional float + table interaction quality

### 4.1 Current coverage

- [x] `fixture-22-float-invoice-chrome.html` — float chrome then in-flow table (sequential, not wrap-around)
- [x] Unit: `TestFloatLeftRightClear`, `TestFloatWidthPercent`
- [~] Wiki-like smoke has floated infobox — not a golden Phase 17 fixture

### 4.2 New fixtures (required for this subplan; do not edit fixture-22)

- [x] Add `testdata/golden/fixture-29-float-beside-table.html` — floated table beside wrapping prose + `clear`
- [x] Envelope in `internal/convert/golden_test.go` `fixturePageBounds`
- [x] Document in `testdata/golden/README.md`
- [x] Proof: `go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-29' -count=1` → pass
- [~] Leave remaining gaps documented: float inside table cell / table-inside-float packing (still `[~]`)

**Shipped on `feature/tier-2-pending-2`:** fixture-29 added (existing fixtures untouched).

---

## Phase 5: Closure gates

### 5.1 Required

- [x] Shared doc-honesty Pass 0 complete for Phase 17 claims
- [x] Parent Phase 17 Pending table updated (matrix rows `[x]`; sticky/full grid/flex remain `[~]`)
- [x] Docs-only: no lint/test required; if fixtures added → `make lint` + `make test`

### 5.2 Next

- [ ] Phase 18 pending (pagination honesty) or Phase 21 product work

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Shared doc-honesty | Closable matrix/fidelity pending |
| Shipped flex/grid/position | Accurate Partial property lists |
| Optional float fixture | Stronger float↔table evidence for Phase 21 |

---

## Out of scope

- Pixel parity with Chrome layout tests
- Implementing real `position: sticky`
- Expanding to full Flex/Grid algorithms
- Bootstrap/Tailwind framework support claims
