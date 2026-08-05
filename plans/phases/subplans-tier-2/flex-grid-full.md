# Tier 2 Subplan - Full Flexbox & Full CSS Grid (separate ledger)

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Status:** Stage A + Stage B (areas/dense + minmax) + Stage C subset (fixture-32/33/34/35)  
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

- [x] Diff `internal/layout/flex.go` / `style.go` against Flexbox L1 property list;
      mark Implemented / Partial / Missing in this file’s appendix
- [x] Diff `internal/layout/grid.go` against Grid L1; note `GridTemplateRows`
      stored but unused → now consumed when height definite
- [x] Freeze “Stage A done” definition (below) before coding Stage B

### 1.2 Staging policy

- [x] Land **Stage A** (deepen current subset) before Stage B (new algorithms)
- [x] Each stage: `make lint` + `make test` + new fixtures; no drive-by refactors
      (`make lint` + layout/convert tests + fixture-32; full `make test` as CI)

---

## Phase 2: Stage A — Flex deepen (stdlib)

### 2.1 Flex container

- [x] Independent `row-gap` / `column-gap` (stop collapsing to single `Gap` if still shared)
- [x] `flex-direction: row-reverse` / `column-reverse`
- [x] `justify-content: space-around` / `space-evenly`
- [x] `align-content` for multi-line wrap
- [x] Column path: honor grow/shrink/justify/align (parity with row)
- [x] Path: `internal/layout/flex.go`, `style.go`

### 2.2 Flex items

- [x] Shorthand `flex: grow shrink basis`
- [x] `align-self`
- [x] Content-based **min-size** contribution (spec §4.5 lite — at least
      non-zero min-content for text)
- [x] Percentage basis cyclic sizing: document + implement subset
      (definite CB resolves %; indefinite → auto/content — fixture-33)

### 2.3 Flex Stage A proof

- [x] New fixtures (do not edit 25/28): wrap + reverse + space-evenly + column grow
      → `testdata/golden/fixture-32-flex-grid-full.html`
- [x] `go test ./internal/layout -run Flex -count=1`
- [x] `make lint` / `go test ./internal/layout ./internal/convert -count=1` — **2026-08-05**

---

## Phase 3: Stage B — Grid deepen (stdlib)

### 3.1 Tracks & placement

- [x] Consume `grid-template-rows` in layout (currently unused)
- [x] `grid-row` / `grid-row-start` / `grid-row-end` / row `span`
- [x] `fr` distribution with min/max constraints (definite container width/height)
      — full `minmax(min,max)` with lengths / `%` (definite) / `fr` / `auto` /
      `min-content` / `max-content` (report-engine subset); `fr` keeps min floors
- [x] `grid-template-areas` + `grid-area` name placement
- [x] `grid-auto-flow: dense` (optional Stage B2)
- [x] Path: `internal/layout/grid.go`, `style.go`

### 3.2 Alignment

- [x] `justify-items` / `align-items` on grid; `justify-self` / `align-self` on items
      (default stretch fills grid area; start/center/end + self overrides)
- [x] Gap already present — verify row/column independence

### 3.3 Grid Stage B proof

- [x] Fixtures: areas, row span, `fr` + minmax lite
      → row span + `fr` rows in fixture-32; areas + dense in fixture-34
- [x] `minmax` + intrinsic + subgrid + masonry → fixture-35
- [x] `go test ./internal/layout -run Grid -count=1`
- [x] `make lint` / `go test ./internal/layout ./internal/convert -count=1` — **2026-08-05**

---

## Phase 4: Stage C — Hard Flex/Grid edges (optional)

### 4.1 Only if product still wants “full”

- [x] Intrinsic sizing passes / nested percentage cycles (subset):
      grid/flex children with `width`/`height: %` against indefinite ancestors
      resolve as auto/content-based (not 0/NaN); measure-pass lite for
      `min-content`/`max-content` track mins via text measure APIs
- [x] Subgrid report-engine lite: `display:subgrid` → nested `grid` that
      copy-inherits parent `grid-template-columns`/`rows`/`areas` when present
      (fixture-35). Honesty: no true shared track sizing across subtrees
- [x] Masonry report-engine lite: `grid-template-rows: masonry` OR
      `grid-template-columns: masonry` packs into the non-masonry axis by
      shortest-stack packing; both axes → dense fallback (fixture-35).
      Honesty: not full CSS Grid L3 masonry (no masonry spanning / shared tracks)

### 4.2 Explicit deferrals

- [~] Subgrid (true shared tracks) — beyond copy-inherit lite above
- [~] Full CSS Grid L3 masonry (spanning / shared tracks)
- [~] Chrome layout-test parity
- [~] Container queries / `:has()` (separate CSS epic)

---

## Phase 5: Docs & closure

### 5.1 Honesty

- [x] Matrix Flex/Grid rows: expand property tables per stage (§2.7 / §2.8)
- [x] Phase 17 “full flex/grid” `[~]` → point here; flip stages to `[x]` as proven
- [x] Do **not** claim Bootstrap/Tailwind support

### 5.2 Closure

- [x] Stage A complete → **2026-08-05**
- [x] Stage B complete → **2026-08-05**
- [x] Stage C lite complete with honesty gaps recorded (subgrid copy-inherit + masonry pack
      shipped as `[x]`; true shared-track subgrid / full L3 masonry / Chrome parity remain `[~]`;
      no multi-pass flex intrinsic beyond cyclic `%`)
- [x] Next: Phase 21 site corpus stress against new layout

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
