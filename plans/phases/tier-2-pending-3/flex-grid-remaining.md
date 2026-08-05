# Tier 2 Pending-3 — Flex/Grid remaining hard edges

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Related:** [`../subplans-tier-2/flex-grid-full.md`](../subplans-tier-2/flex-grid-full.md)  
> **Status:** not started (optional polish + honesty)  
> **Estimated effort:** 1–2 weeks flex polish; **true subgrid / L3 masonry = large / keep `[~]`**  
> **Constraint:** stdlib-only  
> **Spec:** [CSS Flexbox 1](https://www.w3.org/TR/css-flexbox-1/) · [CSS Grid 2 Subgrid](https://www.w3.org/TR/css-grid-2/#subgrids) · [CSS Grid 3](https://www.w3.org/TR/css-grid-3/)

---

## Overview

Stage A–C **lite** already shipped (cyclic `%`, subgrid copy-inherit, masonry
shortest-stack). This ledger covers (1) a bounded **flex content-based min-size
+ percentage re-resolve** polish, and (2) explicit confirmation that **true
shared-track subgrid**, **full L3 masonry spanning**, and **Chrome layout-test
parity** remain deferred unless product amends scope.

---

## Executive Summary

| Topic | Disposition |
|-------|-------------|
| Flex `min-size: auto` + % re-resolve on definite containers | **Should** implement |
| Nested flex multi-pass intrinsic (deep) | `[~]` beyond polish |
| True shared-track subgrid | `[~]` deferred |
| Full CSS Grid L3 masonry + spanning | `[~]` deferred (spec still fluid) |
| Chrome layout-test parity | `[~]` permanent non-goal |

---

## Phase 1: Flex polish (in scope)

### 1.1 Evidence

- [ ] Cite `flex.go` cyclic `%` / `flexItemBaseWidth` / measure `noEmit` stretch pass
- [ ] Cite fixture-33 cyclic height; fixture-32 flex deepen
- [ ] Proof: `go test ./internal/layout -run Flex -count=1` baseline

### 1.2 Content-based minimum

- [ ] Improve `min-width: auto` / content-based minimum on flex items in definite flex containers
- [ ] Re-resolve percentages after min applied where css-sizing-3 requires
- [ ] Do not regress cyclic `%` → auto on indefinite CB
- [ ] Path: `flex.go`
- [ ] Proof: unit tests for `%` child inside definite row; nested flex smoke

### 1.3 Explicit stop line

- [~] Full multi-line flex second layout of all stretched items under every intrinsic constraint — defer if polish sufficient
- [~] Infinite nested intrinsic recursion parity — defer

---

## Phase 2: Subgrid / masonry (confirm deferral)

### 2.1 Honesty only (default)

- [ ] Matrix already Partial for subgrid copy-inherit + masonry pack — verify still accurate
- [ ] Parent phase-17 `[~]` Full CSS Grid L1/L3 + Chrome parity — keep with pointer here
- [ ] `flex-grid-full.md` Stage C explicit deferrals — keep `[~]`

### 2.2 Only if product amends (do not start without amendment)

- [~] Shared track sizing: subgrid items participate in parent Resolve Intrinsic Track Sizes
- [~] Masonry spanning + mixed heights per Grid L3 (wait for CR/syntax freeze)
- [~] New fixtures beyond 35 for true shared tracks

---

## Phase 3: Closure

### 3.1 If flex polish ships

- [ ] `make lint` / `make test`
- [ ] Matrix flex intrinsic note updated
- [ ] Flip “full flex algorithm” parent row to Partial polish `[x]` + remaining `[~]` deep multi-pass

### 3.2 If only honesty

- [ ] Docs-only: no lint/test required per skill
- [ ] Confirm `[~]` reasons + next gate = Phase 21 corpus stress

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
- Subgridded masonry
