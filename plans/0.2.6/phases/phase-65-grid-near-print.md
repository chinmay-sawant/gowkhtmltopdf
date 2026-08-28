# Phase 65: Grid near-print

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 65
> **Status:** not started
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

- [ ] 65.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: names pasted here before coding.

### 65.2 implementation

- [ ] 65.2.1 Implement exit criteria for each owned name. Proof: tests named beside each promotion.

### 65.3 catalog and docs

- [ ] 65.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`.
- [ ] 65.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes.

### 65.4 gates

- [ ] 65.4.1 Targeted package tests exit 0.
- [ ] 65.4.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 65.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed).

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 66.
