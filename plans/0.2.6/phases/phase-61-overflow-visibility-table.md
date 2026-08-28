# Phase 61: Overflow, visibility, table

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 61
> **Status:** not started
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

- [ ] 61.1.1 List exact Partial property names owned by this phase (from current `mapping.json`). Proof: names pasted here before coding.

### 61.2 implementation

- [ ] 61.2.1 Implement exit criteria for each owned name. Proof: tests named beside each promotion.

### 61.3 catalog and docs

- [ ] 61.3.1 Flip promoted rows to `implemented` in `catalog/mapping.json`; recount `coverage-summary.json` and `property-counts.md`.
- [ ] 61.3.2 Update `documentation/compatibility-matrix.md` rows to Implemented with honest notes.

### 61.4 gates

- [ ] 61.4.1 Targeted package tests exit 0.
- [ ] 61.4.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 61.4.3 Before calling the phase done: `make test` and `make lint` exit 0 (and `make golden` if paint/layout/pagination changed).

## Out of scope

Animation, 3D, filter blur, speech, pointer UI, new direct modules, Chrome pixel goldens.

## Handoff

Next is Phase 62.
