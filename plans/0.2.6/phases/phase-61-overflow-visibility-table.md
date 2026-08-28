# Phase 61: Overflow, visibility, table

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 61
> **Status:** complete (8 Partial properties promoted to Implemented)
> **Estimated effort:** L
> **Owner:** `internal/layout` (and `internal/css` when parse changes)
> **Depends on:** Phase 60
> **Unblocks:** Phase 62

---

## Overview

overflow-x/y, visibility:collapse on rows, border-collapse, table-layout fixed, caption-side, vertical-align.

Bar: near-browser for **print media**. Flip mapping `engine_status` to `implemented` only with code + tests + matrix agreement. Goldens stay structural.

## Goals

- Close the Phase 61 Partial names listed in the parent plan inventory for this slice
- Keep `python3 scripts/css-catalog-map.py --check` green after mapping edits
- Do not grow `paint_flow.go` / `paint_pagination.go`; extract if touched

## Checklist

### 61.1 scope lock

- [x] 61.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: `overflow`, `overflow-x`, `overflow-y`, `visibility`, `border-collapse`, `table-layout`, `caption-side`, `vertical-align` (8 properties).

### 61.2 implementation

- [x] 61.2.1 Implement exit criteria for each owned name. Proof: `TestOverflowClip`, `TestStickyOverflow*`, `TestVisibilityHidden`, `TestBorderSpacing`, `TestTableLayout`, `TestTableLayoutFixedIgnoresContentMax`, `TestCaptionSideBottom`, `TestCaptionSideLeft`, `TestCaptionSideRight`, `TestTableCellVerticalAlignMiddle`.

### 61.3 catalog and docs

- [x] 61.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`. Proof: 145 implemented, 29 partial; `property-counts.md` updated.
- [x] 61.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes. Proof: Sections 2.1, 2.3, 2.5 updated; `make claim-scan` clean.

### 61.4 gates

- [x] 61.4.1 Targeted package tests exit 0. Proof: `go test ./internal/layout -run "TestTable.*|TestCaption.*|TestOverflow.*|TestVisibility.*"` exit 0.
- [x] 61.4.2 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: check ok (7 print-noop ignored, 147 apply arms mapped).
- [x] 61.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed). Proof: `make test` and `make lint` exit 0.

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 62.
