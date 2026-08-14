# Tier 2 Subplan - Phase 17 Pending (Broader CSS honesty + optional float/table)

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md) — Pending (after #17)  
> **Status:** **done** — docs honesty, fixture-29, print sticky, flex/grid Stage A–C lite  
> **Estimated effort:** 0.5 day docs (via shared pass) + 0–2 days optional fixtures  
> **Depends on:** [00-shared-doc-honesty.md](00-shared-doc-honesty.md) for matrix/fidelity  
> **Constraint:** stdlib-only; no full browser CSS

---

## Overview

Phase 17 **core is shipped** (partial flex, grid lite, position/float/z-index), plus
follow-ons: print-scoped sticky, flex/grid deepen (Stage A–C lite), and float↔table
fixture-29. Remaining work is intentional non-goals (full browser CSS / Chrome parity)
and a few float packing edge cases.

## Executive Summary

| Pending item (parent) | Disposition | Primary work |
|-----------------------|-------------|--------------|
| Compatibility-matrix / fidelity MVP-gap rows | **[x] done** | Shared doc-honesty Pass 0 |
| Sticky positioning | **[x] done** | Print-scoped ([`sticky-print.md`](sticky-print.md); fixture-31) |
| Full Grid / full Flex | Stage A/B/C lite **[x]**; true L1/L3 + Chrome parity **[~]** | [`flex-grid-full.md`](flex-grid-full.md) |
| Richer float+table interaction fixtures | **[x] done** | `fixture-29-float-beside-table.html` |

---

## Phase 1: Evidence baseline (already scanned 2026-08-05)

### 1.1 Shipped paths (do not re-implement)

- [x] Flex: `internal/layout/flex.go` + `flex_test.go`; fixtures 25, 28 (+ Stage A deepen)
- [x] Grid: `internal/layout/grid.go` + `grid_test.go`; fixtures 28, 32–35 (Stage B/C lite)
- [x] Position: `layout.go` `buildAbsolute`/`buildFixed`/`applyRelativeOffset`; fixture 26/28
- [x] Print sticky: `sticky.go` / `applyStickyPrint`; fixture-31; `TestSticky*`
- [x] z-index lite: `paint.go` `sortPaintIndices`; `TestZIndexPaintOrder`
- [x] Float lite: `float.go` + `TestFloatWidthPercent`; fixtures 22, 29

### 1.2 Supported property subset (honesty bullets for matrix)

See matrix §2.7 / §2.8 and [`flex-grid-full.md`](flex-grid-full.md) for the current
Partial property tables (Stage A flex + Stage B/C grid lite including subgrid
copy-inherit and one-axis masonry pack).

**Position:**
- relative / absolute / fixed lite consumed
- sticky = print-scoped clamp (page content box = scrollport) — not overflow-scroll sticky

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

- [x] Matrix/README: sticky = print page-content-box scrollport (not relative-only alias)
- [x] Cite `sticky.go` / `applyStickyPrint` + `TestSticky*` / fixture-31
- [x] Parent sticky rows flipped to `[x]` (print-scoped); overflow-scroll sticky remains `[~]`

---

## Phase 3: Flex / Grid deepen (executed via flex-grid-full)

### 3.1 Record scope

- [x] Stage A flex deepen shipped — [`flex-grid-full.md`](flex-grid-full.md)
- [x] Stage B grid deepen shipped
- [x] Stage C lite shipped (cyclic `%` subset; subgrid copy-inherit; masonry pack lite)
- [~] True shared-track subgrid / full CSS Grid L3 masonry / Chrome layout-test parity — still deferred
- [x] Multi-column `column-count` lite — [`../tier-2-pending-3/multicol.md`](../tier-2-pending-3/multicol.md)
- [~] Static transforms, container queries / `:has()` — still deferred (pending-3)

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
- [x] Parent Phase 17 Pending table updated (matrix/sticky/fixture rows `[x]`; browser-parity flex/grid remain `[~]`)
- [x] Docs-only: no lint/test required; if fixtures added → `make lint` + `make test`

### 5.2 Next

- [x] Phase 18–20 pending closed; product next is **Phase 21**

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Shared doc-honesty | Closable matrix/fidelity pending |
| Shipped flex/grid/position/sticky | Accurate Partial property lists |
| Float fixture-29 | Stronger float↔table evidence for Phase 21 |

---

## Out of scope

- Pixel parity with Chrome layout tests
- Continuous-media sticky inside `overflow: auto` scrollers
- True shared-track subgrid / full CSS Grid L3 masonry
- Bootstrap/Tailwind framework support claims
