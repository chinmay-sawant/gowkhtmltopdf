# Phase 73: Animation and transition

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 73
> **Status:** not started
> **Estimated effort:** XL
> **Owner:** `internal/layout` / `internal/css` / catalog
> **Depends on:** Phase 72
> **Unblocks:** Phase 74

---

## Overview

animation-* and transition-* for browser-level print: at minimum used/computed styles and first-frame/print rendering policy; timelines if claimed.

Bar: **browser-level print** for the former Ignored set (247 names). Goldens stay structural unless amended. Flip mapping only with code + tests + matrix agreement.

**Count:** 28

## Goals

- Clear every owned name from the work list into Implemented, or mark `[~]` with an explicit reason
- Keep catalog counts honest after each promotion batch

## Checklist

### 73.1 scope lock

- [ ] 73.1.1 Own these 28 properties (from Phase 68 inventory): `animation`, `animation-composition`, `animation-delay`, `animation-delay-end`, `animation-delay-start`, `animation-direction`, `animation-duration`, `animation-fill-mode`, `animation-iteration-count`, `animation-name`, `animation-play-state`, `animation-range`, `animation-range-center`, `animation-range-end`, `animation-range-start`, `animation-timeline`, `animation-timing-function`, `animation-trigger`, `transition`, `transition-behavior`, `transition-delay`, `transition-duration`, `transition-property`, `transition-timing-function`, `view-transition-class`, `view-transition-group`, `view-transition-name`, `view-transition-scope`. Proof: names still `unsupported` (or listed) in `mapping.json` at phase start.

### 73.2 implementation

- [ ] 73.2.1 Implement browser-level print behavior for each owned name (or alias to an existing Implemented longhand). Proof: tests cited per promotion.
- [ ] 73.2.2 Flip each finished name to `engine_status: implemented` with matrix notes. Proof: mapping + matrix.

### 73.3 gates

- [ ] 73.3.1 Targeted package tests exit 0.
- [ ] 73.3.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 73.3.3 `make test` and `make lint` exit 0 before phase complete; `make golden` if paint/layout/pagination changed.


## Out of scope

JavaScript execution. Pixel-diff Chrome goldens as the default gate. New direct Go modules without sign-off. Growing `paint_flow.go` / `paint_pagination.go` past the soft cap without extracting.

## Handoff

Next is Phase 74.
