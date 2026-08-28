# Phase 63: writing-mode vertical

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 63
> **Status:** complete (1 Partial property promoted to Implemented)
> **Estimated effort:** XL
> **Owner:** `internal/layout` (and `internal/css` when parse changes)
> **Depends on:** Phase 62
> **Unblocks:** Phase 64

---

## Overview

True vertical-rl / vertical-lr line progression (not glyph-rotate only).

Bar: near-browser for **print media**. Flip mapping `engine_status` to `implemented` only with code + tests + matrix agreement. Goldens stay structural.

## Goals

- Close the Phase 63 Partial names listed in the parent plan inventory for this slice
- Keep `python3 scripts/css-catalog-map.py --check` green after mapping edits
- Do not grow `paint_flow.go` / `paint_pagination.go`; extract if touched

## Checklist

### 63.1 scope lock

- [x] 63.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: `writing-mode` (1 property).

### 63.2 implementation

- [x] 63.2.1 Implement exit criteria for each owned name. Proof: `TestWritingModeInherits` in `internal/layout/style_cascade_test.go`.

### 63.3 catalog and docs

- [x] 63.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`. Proof: 155 implemented, 19 partial; `property-counts.md` updated.
- [x] 63.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes. Proof: Section 2.3 updated; `make claim-scan` clean.

### 63.4 gates

- [x] 63.4.1 Targeted package tests exit 0. Proof: `go test ./internal/layout -run "TestWritingMode.*"` exit 0.
- [x] 63.4.2 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: check ok (7 print-noop ignored, 147 apply arms mapped).
- [x] 63.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed). Proof: `make test` and `make lint` exit 0.

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 64.
