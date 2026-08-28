# Phase 75: Anchor, offset, view timelines

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 75
> **Status:** not started
> **Estimated effort:** XL
> **Owner:** `internal/layout` / `internal/css` / catalog
> **Depends on:** Phase 74
> **Unblocks:** Phase 76

---

## Overview

anchor-*, offset-*, position-try*, view-timeline*, view-transition-adjacent names.

Bar: **browser-level print** for the former Ignored set (247 names). Goldens stay structural unless amended. Flip mapping only with code + tests + matrix agreement.

**Count:** 21

## Goals

- Clear every owned name from the work list into Implemented, or mark `[~]` with an explicit reason
- Keep catalog counts honest after each promotion batch

## Checklist

### 75.1 scope lock

- [ ] 75.1.1 Own these 21 properties (from Phase 68 inventory): `anchor-name`, `anchor-scope`, `offset`, `offset-anchor`, `offset-distance`, `offset-path`, `offset-position`, `offset-rotate`, `overflow-anchor`, `position-anchor`, `position-area`, `position-try`, `position-try-fallbacks`, `position-try-order`, `position-visibility`, `timeline-scope`, `view-timeline`, `view-timeline-axis`, `view-timeline-inset`, `view-timeline-name`, `will-change`. Proof: names still `unsupported` (or listed) in `mapping.json` at phase start.

### 75.2 implementation

- [ ] 75.2.1 Implement browser-level print behavior for each owned name (or alias to an existing Implemented longhand). Proof: tests cited per promotion.
- [ ] 75.2.2 Flip each finished name to `engine_status: implemented` with matrix notes. Proof: mapping + matrix.

### 75.3 gates

- [ ] 75.3.1 Targeted package tests exit 0.
- [ ] 75.3.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 75.3.3 `make test` and `make lint` exit 0 before phase complete; `make golden` if paint/layout/pagination changed.


## Out of scope

JavaScript execution. Pixel-diff Chrome goldens as the default gate. New direct Go modules without sign-off. Growing `paint_flow.go` / `paint_pagination.go` past the soft cap without extracting.

## Handoff

Next is Phase 76.
