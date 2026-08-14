# Tier 2 Pending-3 — Flex/Grid remaining hard edges

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Related:** [`../subplans-tier-2/flex-grid-full.md`](../subplans-tier-2/flex-grid-full.md)  
> **Status:** done (Phase 1 polish + Phase 2 Partial lite expand)  
> **Estimated effort:** 1–2 weeks flex polish; true joint-intrinsic subgrid / full L3 masonry remain Partial honesty  
> **Constraint:** stdlib-only  
> **Spec:** [CSS Flexbox 1](https://www.w3.org/TR/css-flexbox-1/) · [CSS Grid 2 Subgrid](https://www.w3.org/TR/css-grid-2/#subgrids) · [CSS Grid 3](https://www.w3.org/TR/css-grid-3/)

---

## Overview

Stage A–C **lite** already shipped (cyclic `%`, subgrid copy-inherit, masonry
shortest-stack). This ledger: (1) **flex content-based min-size + percentage
re-resolve** polish — **shipped**; (2) **Partial** subgrid / masonry expand
(shared-width track re-resolve honesty + masonry `grid-column` span on fixed
axis) with tests — still not joint intrinsic / full L3.

---

## Executive Summary

| Topic | Disposition |
|-------|-------------|
| Flex `min-size: auto` + % re-resolve on definite containers | **[x] Shipped** |
| Nested flex multi-pass intrinsic (deep) | Documented stop line (beyond polish) |
| True shared-track subgrid (joint Resolve Intrinsic) | **[x] Partial** — copy-inherit + same-width re-resolve; no joint pass |
| Full CSS Grid L3 masonry + spanning | **[x] Partial** — shortest-stack + fixed-axis span N |
| Chrome layout-test parity | Permanent non-goal |

---

## Phase 1: Flex polish (in scope)

### 1.1 Evidence

- [x] Cite `flex.go` cyclic `%` / `flexItemBaseWidth` / measure `noEmit` stretch pass
- [x] Cite fixture-33 cyclic height; fixture-32 flex deepen
- [x] Proof: `go test ./internal/layout -run Flex -count=1` baseline

### 1.2 Content-based minimum

- [x] Improve `min-width: auto` / content-based minimum on flex items in definite flex containers (`flexMinMainSize`)
- [x] Re-resolve percentages after min applied (`flexClampMainWidths`; `MinWidthPercent` / column `MinHeightPercent`)
- [x] Do not regress cyclic `%` → auto on indefinite CB
- [x] Path: `flex.go`, `style.go`
- [x] Proof: `TestFlexContentMinSizeDefiniteRow`, `TestFlexPercentChildDefiniteRow`, `TestFlexMinWidthPercentDefinite`, `TestFlexNestedSmoke`, `TestFlexBasisPercent*`

### 1.3 Explicit stop line

- [x] Full multi-line flex second layout of all stretched items under every intrinsic constraint — deferred beyond polish (documented Partial in matrix §2.7)
- [x] Infinite nested intrinsic recursion parity — deferred (documented)

---

## Phase 2: Subgrid / masonry (Partial shipped)

### 2.1 Honesty + lite expand

- [x] Matrix Partial for subgrid copy-inherit + masonry pack — updated
- [x] Parent phase-17 Full CSS Grid L1/L3 + Chrome parity — Partial polish pointer here
- [x] `flex-grid-full.md` Stage C explicit deferrals — keep as lite + this ledger

### 2.2 Partial improvements (shipped)

- [x] Subgrid: copy-inherit templates; same content-width → track sizes re-resolve from inherited template (`TestSubgridTrackWidthMatchesParent`) — **not** joint parent/child intrinsic
- [x] Masonry: shortest-stack + `grid-column: span N` on fixed axis (`TestMasonryColumnSpanPartial`) — **not** full L3 spanning search / reverse pack
- [x] Both-axes masonry → dense fallback (existing)

---

## Phase 3: Closure

### 3.1 Flex polish + Partial lite

- [x] `make lint` → PASS (`go vet ./...`, 2026-08-05)
- [x] `go test ./internal/layout ./internal/convert -count=1` → PASS (2026-08-05)
- [x] Matrix flex intrinsic / subgrid / masonry notes updated
- [x] Parent “full flex algorithm” → Partial polish `[x]` + deep multi-pass documented stop

### 3.2 If only honesty

- [x] N/A — polish + Partial expand shipped

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| flex-grid-full Stage A–C lite | Baseline |
| Phase 21 | Real-world pressure on remaining gaps |

---

## Out of scope

- Bootstrap/Tailwind claims
- Chrome layout-test suite parity
- Joint Resolve Intrinsic Track Sizes across subgrid subtrees
- Subgridded masonry
