# Phase 64: Flex near-print

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 64
> **Status:** complete (honest print subset; not near-browser-complete flex)
> **Estimated effort:** L
> **Owner:** `internal/layout` (and `internal/css` when parse changes)
> **Depends on:** Phase 63
> **Unblocks:** Phase 65

---

## Overview

flex-flow, align-content, place-* near-browser print flex.

Bar: near-browser for **print media**. Flip mapping `engine_status` to `implemented` only with code + tests + matrix agreement. Goldens stay structural.

## Goals

- Close the Phase 64 Partial names listed in the parent plan inventory for this slice
- Keep `python3 scripts/css-catalog-map.py --check` green after mapping edits
- Do not grow `paint_flow.go` / `paint_pagination.go`; extract if touched

## Checklist

### 64.1 scope lock

- [x] 64.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: `align-content`, `flex-flow`, `place-content`, `place-items`, `place-self` (5 properties).

### 64.2 implementation

- [x] 64.2.1 Implement exit criteria for each owned name. Proof: `TestFlexFlowShorthand`, `TestPlaceShorthands`, `TestAlignContentStretch`.

### 64.3 catalog and docs

- [x] 64.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`. Proof: 160 implemented, 14 partial; `property-counts.md` updated.
- [x] 64.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes. Proof: Section 2.7 updated; `make claim-scan` clean.

### 64.4 gates

- [x] 64.4.1 Targeted package tests exit 0. Proof: `go test ./internal/layout -run "TestFlexFlow.*|TestPlace.*|TestAlignContent.*"` exit 0.
- [x] 64.4.2 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: check ok (7 print-noop ignored, 147 apply arms mapped).
- [x] 64.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed). Proof: `make test` and `make lint` exit 0.


## Honesty audit (2026-08-28)

Audit 2026-08-28: owned shorthands/align-content kept Implemented with row-wrap limits.


## Agent guard

Read `../HONESTY-GATES.md` before any mapping flip. This phase is marked complete only for the **print subset documented in the matrix**. Do not broaden to Chrome-complete or re-fake Implemented rows with empty `code_path`.

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 65.
