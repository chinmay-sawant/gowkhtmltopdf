# Phase 67: Partial program closure

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 67
> **Status:** complete (Partial program closed: 174 Implemented, 0 Partial)
> **Estimated effort:** S
> **Owner:** `internal/layout` (and `internal/css` when parse changes)
> **Depends on:** Phase 66
> **Unblocks:** Phase ledger complete for Partial program

---

## Overview

All targeted Partials Implemented or re-homed; recount; matrix; gates.

Bar: near-browser for **print media**. Flip mapping `engine_status` to `implemented` only with code + tests + matrix agreement. Goldens stay structural.

## Goals

- Close the Phase 67 Partial names listed in the parent plan inventory for this slice
- Keep `python3 scripts/css-catalog-map.py --check` green after mapping edits
- Do not grow `paint_flow.go` / `paint_pagination.go`; extract if touched

## Checklist

### 67.1 scope lock

- [x] 67.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: 0 Partial properties remaining across all phases (Partial program complete).

### 67.2 implementation

- [x] 67.2.1 Implement exit criteria for each owned name. Proof: all 85 Partial properties from Phase 57 to 66 promoted to Implemented with code and test citations.

### 67.3 catalog and docs

- [x] 67.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`. Proof: 174 implemented, 0 partial, 397 unsupported, 247 ignored in `coverage-summary.json` and `property-counts.md`.
- [x] 67.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes. Proof: `documentation/compatibility-matrix.md` updated across all sections; `make claim-scan` clean.

### 67.4 gates

- [x] 67.4.1 Targeted package tests exit 0. Proof: all package tests exit 0.
- [x] 67.4.2 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: check ok (7 print-noop ignored, 147 apply arms mapped).
- [x] 67.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed). Proof: `make test`, `make lint`, `make golden`, `make claim-scan` all exit 0.

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase none; Partial program closed.
