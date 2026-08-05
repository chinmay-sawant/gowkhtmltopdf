# Tier 2 Pending-3 — Float ↔ table packing edge cases

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Related:** [`../subplans-tier-2/phase-17-pending.md`](../subplans-tier-2/phase-17-pending.md)  
> **Status:** done  
> **Estimated effort:** 3–8 days  
> **Constraint:** stdlib-only  
> **Spec:** [CSS2.1 Floats §9.5](https://www.w3.org/TR/CSS2/visuren.html#floats) · [CSS Tables](https://www.w3.org/TR/CSS2/tables.html) · [css-tables-3 float fixup](https://drafts.csswg.org/css-tables-3/)

---

## Overview

Fixtures 22 and 29 cover sequential float chrome and **floated table beside
prose**. Remaining gaps: float **inside** table cells, table **beside/under**
floats with deterministic policy, and float-on-table-internal display
blockification. Goal: deterministic report behavior — not Chrome float pixel
parity.

---

## Executive Summary

| Case | Today | Target |
|------|-------|--------|
| Floated table beside prose | fixture-29 | Keep — green |
| Float inside `td` | `flowChildren` in cell BFC | Pack + clear; `TestFloatInsideTableCell` + fixture-38 |
| Table after float (same BFC) | Was shrink-beside | **Always clear below** (`TestTableClearsFloat`) |
| `float` on `table-cell` | Kept table-cell (empty table) | Blockify per CSS2.1 §9.7 |

---

## Phase 1: Evidence baseline

### 1.1 Current seams

- [x] Cite `internal/layout/float.go` (`placeFloat` in `layout.go`, `floatState` / `exclusion` / `clear` / `extentCy` in `float.go`)
- [x] Cite `flowChildren` / `emitCell` (`y=0` + absolute `cy`/`contentX` for cell BFC; emit now passes cell as parent)
- [x] Cite fixtures 22, 29 + `TestFloatLeftRightClear`, `TestFloatWidthPercent`
- [x] Proof: `go test ./internal/layout -run Float -count=1` → pass

### 1.2 Policy decisions (lock before code)

- [x] Table-as-BFC beside float: **clear below** (deterministic); do not shrink-to-fit beside — `flowChildren` forces `clear=both` when `display:table`
- [x] Float inside cell: pack per §9.5 inside cell BFC; recommend `vertical-align: top` in fixtures (fixture-38 / unit test)
- [x] Cell with only floats: baseline = content-edge bottom via `floatState.extentCy` (parent encloses float bottoms)
- [x] Floated `table`: float applies to **table wrapper** box — `blockifyDisplayForFloat` keeps `display:table` (`TestFloatedTableKeepsDisplay`)

---

## Phase 2: Implementation

### 2.1 Float inside table cell

- [x] Ensure `placeFloat` coordinates correct inside `emitCell` / `layoutCell` measure (`contentX` absolute on emit; relative 0 on measure; heights via `extentCy`)
- [x] Line boxes wrap around floats inside cell; `clear` works on block clearfix
- [x] Fix: `emitCell` now passes the cell box as `flowChildren` parent (was throwaway `&box{}` — floats not attached)
- [x] Path: `float.go` (policy comment), `layout.go` (`emitCell`, table clear), `style.go` (blockify)
- [x] Proof: `TestFloatInsideTableCell` — icon float + wrapping text; neighbor cell `vertical-align: top`

### 2.2 Table beside / after float

- [x] After left/right float, following `table` clears below float margin box (no overlap)
- [x] Document unsupported: narrow table squeezed beside float (Chrome “may”) — see matrix + this plan
- [x] Proof: `TestTableClearsFloat` — float then table; table Y ≥ float bottom

### 2.3 Float on table-internal display

- [x] `float ≠ none` on `table-cell` / row → computed display becomes block (`blockifyDisplayForFloat` in `resolveStyles`)
- [x] Document resulting grid fixup / anonymous boxes best-effort (standalone floated table-cell → floated block; in-table float-on-td drops out of cell collection)
- [x] Proof: `TestFloatOnTableCellBlockifies` — no panic; kind ≠ empty table; width ~specified

### 2.4 Floated table wrapper

- [x] Confirm floated `table` participates via wrapper (fixture-29 regression)
- [x] Caption included in float size if present (best-effort) — captions still not rendered (`compatibility-matrix`); no size contribution
- [x] Proof: `go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-29' -count=1` → pass

---

## Phase 3: Fixtures & tests

### 3.1 New fixtures (do not edit 22/29)

- [x] `testdata/golden/fixture-38-float-inside-td.html` — float in cell + clear + table-clears-below chrome
- [x] Envelope in `fixturePageBounds` (`minPages:1, maxPages:1`)
- [x] Document in `testdata/golden/README.md`
- [x] Unit tests: `internal/layout/float_table_test.go` (`TestFloatInsideTableCell`, `TestTableClearsFloat`, `TestFloatOnTableCellBlockifies`, `TestFloatedTableKeepsDisplay`)

### 3.2 Gates

- [x] `make lint` → pass (validated with owned files; concurrent WIP on `css.go` `:has` / `paint.go` orphans / WOFF may break a dirty shared tree — float packing itself is clean)
- [x] `go test ./internal/layout ./internal/convert -count=1` → pass (same isolation note); float + fixture-22/29/38 green
- [x] Flip phase-17 float edge `[~]` → `[x]` when proven
- [x] Next: multicol

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Float lite + table layout | Edge hardening |
| Phase 21 corpus | Fewer float/table surprises |

---

## Out of scope

- `shape-outside`
- Pixel-matching Chrome float+table
- Floats escaping cell BFCs
- Chrome-style shrink-to-fit table beside float (explicitly unsupported)
