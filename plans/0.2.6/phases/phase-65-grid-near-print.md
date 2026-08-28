# Phase 65: Grid near-print

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 65
> **Status:** complete (honest print Stage B/C lite; not Grid L1/Chrome)
> **Estimated effort:** XL
> **Owner:** `internal/layout` (and `internal/css` when parse changes)
> **Depends on:** Phase 64
> **Unblocks:** Phase 66

---

## Overview

grid / grid-template* / grid-row|column* track sizing and placement.

Bar: near-browser for **print media**. Flip mapping `engine_status` to `implemented` only with code + tests + matrix agreement. Goldens stay structural.

## Goals

- Close the Phase 65 Partial names listed in the parent plan inventory for this slice
- Keep `python3 scripts/css-catalog-map.py --check` green after mapping edits
- Do not grow `paint_flow.go` / `paint_pagination.go`; extract if touched

## Checklist

### 65.1 scope lock

- [x] 65.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: `grid`, `grid-template`, `grid-template-columns`, `grid-template-rows`, `grid-row`, `grid-row-start`, `grid-row-end`, `grid-column`, `grid-column-start`, `grid-column-end` (10 properties).

### 65.2 implementation

- [x] 65.2.1 Implement exit criteria for each owned name. Proof: `TestGridTemplateShorthand`, `TestGridRowSpan*`, `TestGridRowGapVsColumnGap`, `TestInlineGridIsInlineLevel`, `TestGridPlacement*`.

### 65.3 catalog and docs

- [x] 65.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`. Proof: 170 implemented, 4 partial; `property-counts.md` updated.
- [x] 65.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes. Proof: Section 2.8 updated; `make claim-scan` clean.

### 65.4 gates

- [x] 65.4.1 Targeted package tests exit 0. Proof: `go test ./internal/layout -run "TestGrid.*"` exit 0.
- [x] 65.4.2 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: check ok (7 print-noop ignored, 147 apply arms mapped).
- [x] 65.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed). Proof: `make test` and `make lint` exit 0.


## Honesty audit (2026-08-28)

Audit 2026-08-28: owned grid names kept Implemented as print subset.


## Agent guard

Read `../HONESTY-GATES.md` before any mapping flip. This phase is marked complete only for the **print subset documented in the matrix**. Do not broaden to Chrome-complete or re-fake Implemented rows with empty `code_path`.

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 66.
