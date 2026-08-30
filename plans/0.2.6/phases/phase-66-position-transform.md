# Phase 66: Position / transform / stacking

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 66
> **Status:** complete (static 2D transform + print position lite)
> **Estimated effort:** XL
> **Owner:** `internal/layout` (and `internal/css` when parse changes)
> **Depends on:** Phase 65
> **Unblocks:** Phase 67

---

## Overview

position, transform, display (after flex/grid), container-type.

Bar: near-browser for **print media**. Flip mapping `engine_status` to `implemented` only with code + tests + matrix agreement. Goldens stay structural.

## Goals

- Close the Phase 66 Partial names listed in the parent plan inventory for this slice
- Keep `python3 scripts/css-catalog-map.py --check` green after mapping edits
- Do not grow `paint_flow.go` / `paint_pagination.go`; extract if touched

## Checklist

### 66.1 scope lock

- [x] 66.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: `position`, `transform`, `display`, `container-type` (4 properties).

### 66.2 implementation

- [x] 66.2.1 Implement exit criteria for each owned name. Proof: `TestContainer*`, `TestDisplayNone`, `TestTableLayout`, `TestInlineBlockBesideText`, `TestSticky*`, `TestStickyOverflow*`, `TestTransform*`.

### 66.3 catalog and docs

- [x] 66.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`. Proof: 174 implemented, 0 partial; `property-counts.md` updated.
- [x] 66.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes. Proof: Sections 2.2, 2.10, and 3 updated; `make claim-scan` clean.

### 66.4 gates

- [x] 66.4.1 Targeted package tests exit 0. Proof: `go test ./internal/layout -run "TestContainer.*|TestSticky.*|TestTransform.*"` exit 0.
- [x] 66.4.2 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: check ok (7 print-noop ignored, 147 apply arms mapped).
- [x] 66.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed). Proof: `make test` and `make lint` exit 0.


## Honesty audit (2026-08-28)

Audit 2026-08-28: transform is 2D only; 3D still rejected. Mapping notes must stay narrow.


## Agent guard

Read `../HONESTY-GATES.md` before any mapping flip. This phase is marked complete only for the **print subset documented in the matrix**. Do not broaden to Chrome-complete or re-fake Implemented rows with empty `code_path`.

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 67.
