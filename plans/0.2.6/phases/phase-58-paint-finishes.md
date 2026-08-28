# Phase 58: Paint finishes

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 58
> **Status:** complete (13 Partial properties promoted to Implemented)
> **Estimated effort:** M
> **Owner:** `internal/layout` (and `internal/css` when parse changes)
> **Depends on:** Phase 57
> **Unblocks:** Phase 59

---

## Overview

outline*, border-radius percent, box-shadow layers/inset/spread, background / background-image layers.

Bar: near-browser for **print media**. Flip mapping `engine_status` to `implemented` only with code + tests + matrix agreement. Goldens stay structural.

## Goals

- Close the Phase 58 Partial names listed in the parent plan inventory for this slice
- Keep `python3 scripts/css-catalog-map.py --check` green after mapping edits
- Do not grow `paint_flow.go` / `paint_pagination.go`; extract if touched

## Checklist

### 58.1 scope lock

- [x] 58.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: `outline`, `outline-color`, `outline-offset`, `outline-style`, `outline-width`, `border-radius`, `border-top-left-radius`, `border-top-right-radius`, `border-bottom-left-radius`, `border-bottom-right-radius`, `box-shadow`, `background`, `background-image` (13 properties).

### 58.2 implementation

- [x] 58.2.1 Implement exit criteria for each owned name. Proof: `TestOutlineParse`, `TestOutlineStroke`, `TestRadiusLonghand`, `TestRadiusSlash`, `TestRadiusEllipticalLonghand`, `TestRadiusPercentAxes`, `TestBoxShadowParse`, `TestBoxShadowPaints`, `TestBoxShadowBlurPaints`, `TestBackgroundImageParse`, `TestBackgroundImageLayoutPaints`.

### 58.3 catalog and docs

- [x] 58.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`. Proof: 102 implemented, 72 partial; `property-counts.md` updated.
- [x] 58.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes. Proof: `documentation/compatibility-matrix.md` updated and `make claim-scan` clean.

### 58.4 gates

- [x] 58.4.1 Targeted package tests exit 0. Proof: `go test ./internal/layout -run "TestRadius.*|TestOutline.*|TestBoxShadow.*"` exit 0.
- [x] 58.4.2 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: check ok (7 print-noop ignored, 147 apply arms mapped).
- [x] 58.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed). Proof: `make test`, `make lint`, `make golden` all exit 0.

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 59.
