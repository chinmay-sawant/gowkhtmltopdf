# Tier 2 Pending-3 — Float ↔ table packing edge cases

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Related:** [`../subplans-tier-2/phase-17-pending.md`](../subplans-tier-2/phase-17-pending.md)  
> **Status:** not started  
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
| Floated table beside prose | fixture-29 | Keep |
| Float inside `td` | Possible via `flowChildren` / untested | Pack + clear; baseline documented |
| Table after float (same BFC) | Underspecified | **Always clear below** (document “may shrink” unsupported) |
| `float` on `table-cell` | Untested | Blockify per CSS Display |

---

## Phase 1: Evidence baseline

### 1.1 Current seams

- [ ] Cite `internal/layout/float.go` (`placeFloat`, `floatState`, exclusions)
- [ ] Cite `flowChildren` / `emitCell` (`y=0` absolute cy risk)
- [ ] Cite fixtures 22, 29 + `TestFloatLeftRightClear`, `TestFloatWidthPercent`
- [ ] Proof: `go test ./internal/layout -run Float -count=1`

### 1.2 Policy decisions (lock before code)

- [ ] Table-as-BFC beside float: **clear below** (deterministic); do not shrink-to-fit beside
- [ ] Float inside cell: pack per §9.5 inside cell BFC; recommend `vertical-align: top` in fixtures
- [ ] Cell with only floats: baseline = content-edge bottom (CSS2); document
- [ ] Floated `table`: float applies to **table wrapper** box

---

## Phase 2: Implementation

### 2.1 Float inside table cell

- [ ] Ensure `placeFloat` coordinates correct inside `emitCell` / `buildCell` measure
- [ ] Line boxes wrap around floats inside cell; `clear` works
- [ ] Fix double-layout measure vs emit bugs if found
- [ ] Path: `float.go`, table cell builders in `layout.go`
- [ ] Proof: `TestFloatInsideTableCell` — icon float + wrapping text; neighbor cell `vertical-align: top`

### 2.2 Table beside / after float

- [ ] After left/right float, following `table` clears below float margin box (no overlap)
- [ ] Document unsupported: narrow table squeezed beside float (Chrome “may”)
- [ ] Proof: `TestTableClearsFloat` — float then table; table Y ≥ float bottom

### 2.3 Float on table-internal display

- [ ] `float ≠ none` on `table-cell` / row → computed display becomes block (before table fixup)
- [ ] Document resulting grid fixup / anonymous boxes best-effort
- [ ] Proof: unit test does not panic; documented behavior snapshot

### 2.4 Floated table wrapper

- [ ] Confirm floated `table` participates via wrapper (fixture-29 regression)
- [ ] Caption included in float size if present (best-effort)
- [ ] Proof: fixture-29 still green

---

## Phase 3: Fixtures & tests

### 3.1 New fixtures (do not edit 22/29)

- [ ] `testdata/golden/fixture-38-float-inside-td.html` — float in cell + clear
- [ ] Optional envelope in `fixturePageBounds` (loose page count)
- [ ] Document in `testdata/golden/README.md`
- [ ] Unit tests under `internal/layout` for clear-below policy

### 3.2 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Flip phase-17 float edge `[~]` → `[x]` when proven
- [ ] Next: multicol

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
