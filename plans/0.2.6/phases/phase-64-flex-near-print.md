# Phase 64: Flex near-print

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 64
> **Status:** not started
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

- [ ] 64.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: names pasted here before coding.

### 64.2 implementation

- [ ] 64.2.1 Implement exit criteria for each owned name. Proof: tests named beside each promotion.

### 64.3 catalog and docs

- [ ] 64.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`.
- [ ] 64.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes.

### 64.4 gates

- [ ] 64.4.1 Targeted package tests exit 0.
- [ ] 64.4.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 64.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed).

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 65.
