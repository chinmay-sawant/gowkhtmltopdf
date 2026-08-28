# Phase 76: Pointer and form UI

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 76
> **Status:** not started
> **Estimated effort:** M
> **Owner:** `internal/layout` / `internal/css` / catalog
> **Depends on:** Phase 75
> **Unblocks:** Phase 77

---

## Overview

cursor, caret-color, resize, user-select, pointer-events, touch-action, appearance. PDF has no pointer; define print/UA fallbacks or paint-time equivalents.

Bar: **browser-level print** for the former Ignored set (247 names). Goldens stay structural unless amended. Flip mapping only with code + tests + matrix agreement.

**Count:** 7

## Goals

- Clear every owned name from the work list into Implemented, or mark `[~]` with an explicit reason
- Keep catalog counts honest after each promotion batch

## Checklist

### 76.1 scope lock

- [ ] 76.1.1 Own these 7 properties (from Phase 68 inventory): `appearance`, `caret-color`, `cursor`, `pointer-events`, `resize`, `touch-action`, `user-select`. Proof: names still `unsupported` (or listed) in `mapping.json` at phase start.

### 76.2 implementation

- [ ] 76.2.1 Implement browser-level print behavior for each owned name (or alias to an existing Implemented longhand). Proof: tests cited per promotion.
- [ ] 76.2.2 Flip each finished name to `engine_status: implemented` with matrix notes. Proof: mapping + matrix.

### 76.3 gates

- [ ] 76.3.1 Targeted package tests exit 0.
- [ ] 76.3.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 76.3.3 `make test` and `make lint` exit 0 before phase complete; `make golden` if paint/layout/pagination changed.


## Out of scope

JavaScript execution. Pixel-diff Chrome goldens as the default gate. New direct Go modules without sign-off. Growing `paint_flow.go` / `paint_pagination.go` past the soft cap without extracting.

## Handoff

Next is Phase 77.
