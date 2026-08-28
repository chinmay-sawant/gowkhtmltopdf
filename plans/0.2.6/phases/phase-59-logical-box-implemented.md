# Phase 59: Logical box Implemented

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 59
> **Status:** complete (25 Partial properties promoted to Implemented)
> **Estimated effort:** M
> **Owner:** `internal/layout` (and `internal/css` when parse changes)
> **Depends on:** Phase 58
> **Unblocks:** Phase 60

---

## Overview

margin/padding/inset/*-size logical properties for horizontal-tb; vertical waits on 63.

Bar: near-browser for **print media**. Flip mapping `engine_status` to `implemented` only with code + tests + matrix agreement. Goldens stay structural.

## Goals

- Close the Phase 59 Partial names listed in the parent plan inventory for this slice
- Keep `python3 scripts/css-catalog-map.py --check` green after mapping edits
- Do not grow `paint_flow.go` / `paint_pagination.go`; extract if touched

## Checklist

### 59.1 scope lock

- [x] 59.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: `margin-block`, `margin-block-start`, `margin-block-end`, `margin-inline`, `margin-inline-start`, `margin-inline-end`, `padding-block`, `padding-block-start`, `padding-block-end`, `padding-inline`, `padding-inline-start`, `padding-inline-end`, `inset`, `inset-block`, `inset-block-start`, `inset-block-end`, `inset-inline`, `inset-inline-start`, `inset-inline-end`, `block-size`, `inline-size`, `min-block-size`, `min-inline-size`, `max-block-size`, `max-inline-size` (25 properties).

### 59.2 implementation

- [x] 59.2.1 Implement exit criteria for each owned name. Proof: `TestLogicalMargin`, `TestLogicalPadding`, `TestLogicalInset`, `TestLogicalSize` in `internal/layout/css_apply_test.go`.

### 59.3 catalog and docs

- [x] 59.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`. Proof: 127 implemented, 47 partial; `property-counts.md` updated.
- [x] 59.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes. Proof: Section 2.1 updated; `make claim-scan` clean.

### 59.4 gates

- [x] 59.4.1 Targeted package tests exit 0. Proof: `go test ./internal/layout -run "TestLogical.*"` exit 0.
- [x] 59.4.2 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: check ok (7 print-noop ignored, 147 apply arms mapped).
- [x] 59.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed). Proof: `make test` and `make lint` exit 0.

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 60.
