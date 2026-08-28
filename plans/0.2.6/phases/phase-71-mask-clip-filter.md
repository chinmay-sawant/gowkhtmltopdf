# Phase 71: Mask, clip, and filter

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 71
> **Status:** not started
> **Estimated effort:** L
> **Owner:** `internal/layout` / `internal/css` / catalog
> **Depends on:** Phase 70
> **Unblocks:** Phase 72

---

## Overview

clip/clip-path/mask* plus filter and backdrop-filter for print raster/PDF paint.

Bar: **browser-level print** for the former Ignored set (247 names). Goldens stay structural unless amended. Flip mapping only with code + tests + matrix agreement.

**Count:** 26

## Goals

- Clear every owned name from the work list into Implemented, or mark `[~]` with an explicit reason
- Keep catalog counts honest after each promotion batch

## Checklist

### 71.1 scope lock

- [ ] 71.1.1 Own these 26 properties (from Phase 68 inventory): `backdrop-filter`, `clip`, `clip-path`, `clip-rule`, `color-interpolation-filters`, `filter`, `flood-color`, `flood-opacity`, `lighting-color`, `mask`, `mask-border`, `mask-border-mode`, `mask-border-outset`, `mask-border-repeat`, `mask-border-slice`, `mask-border-source`, `mask-border-width`, `mask-clip`, `mask-composite`, `mask-image`, `mask-mode`, `mask-origin`, `mask-position`, `mask-repeat`, `mask-size`, `mask-type`. Proof: names still `unsupported` (or listed) in `mapping.json` at phase start.

### 71.2 implementation

- [ ] 71.2.1 Implement browser-level print behavior for each owned name (or alias to an existing Implemented longhand). Proof: tests cited per promotion.
- [ ] 71.2.2 Flip each finished name to `engine_status: implemented` with matrix notes. Proof: mapping + matrix.

### 71.3 gates

- [ ] 71.3.1 Targeted package tests exit 0.
- [ ] 71.3.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 71.3.3 `make test` and `make lint` exit 0 before phase complete; `make golden` if paint/layout/pagination changed.


## Out of scope

JavaScript execution. Pixel-diff Chrome goldens as the default gate. New direct Go modules without sign-off. Growing `paint_flow.go` / `paint_pagination.go` past the soft cap without extracting.

## Handoff

Next is Phase 72.
