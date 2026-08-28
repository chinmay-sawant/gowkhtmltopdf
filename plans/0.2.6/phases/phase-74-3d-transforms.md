# Phase 74: 3D transforms

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 74
> **Status:** not started
> **Estimated effort:** L
> **Owner:** `internal/layout` / `internal/css` / catalog
> **Depends on:** Phase 73
> **Unblocks:** Phase 75

---

## Overview

perspective, perspective-origin, transform-style, backface-visibility for static print scenes.

Bar: **browser-level print** for the former Ignored set (247 names). Goldens stay structural unless amended. Flip mapping only with code + tests + matrix agreement.

**Count:** 4

## Goals

- Clear every owned name from the work list into Implemented, or mark `[~]` with an explicit reason
- Keep catalog counts honest after each promotion batch

## Checklist

### 74.1 scope lock

- [ ] 74.1.1 Own these 4 properties (from Phase 68 inventory): `backface-visibility`, `perspective`, `perspective-origin`, `transform-style`. Proof: names still `unsupported` (or listed) in `mapping.json` at phase start.

### 74.2 implementation

- [ ] 74.2.1 Implement browser-level print behavior for each owned name (or alias to an existing Implemented longhand). Proof: tests cited per promotion.
- [ ] 74.2.2 Flip each finished name to `engine_status: implemented` with matrix notes. Proof: mapping + matrix.

### 74.3 gates

- [ ] 74.3.1 Targeted package tests exit 0.
- [ ] 74.3.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 74.3.3 `make test` and `make lint` exit 0 before phase complete; `make golden` if paint/layout/pagination changed.


## Out of scope

JavaScript execution. Pixel-diff Chrome goldens as the default gate. New direct Go modules without sign-off. Growing `paint_flow.go` / `paint_pagination.go` past the soft cap without extracting.

## Handoff

Next is Phase 75.
