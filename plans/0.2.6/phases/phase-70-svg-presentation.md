# Phase 70: SVG presentation (fill/stroke)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 70
> **Status:** not started
> **Estimated effort:** L
> **Owner:** `internal/layout` / `internal/css` / catalog
> **Depends on:** Phase 69
> **Unblocks:** Phase 71

---

## Overview

Honor SVG presentation attributes/properties used in print SVG (fill, stroke, and related).

Bar: **browser-level print** for the former Ignored set (247 names). Goldens stay structural unless amended. Flip mapping only with code + tests + matrix agreement.

**Count:** 31

## Goals

- Clear every owned name from the work list into Implemented, or mark `[~]` with an explicit reason
- Keep catalog counts honest after each promotion batch

## Checklist

### 70.1 scope lock

- [ ] 70.1.1 Own these 31 properties (from Phase 68 inventory): `fill`, `fill-break`, `fill-color`, `fill-image`, `fill-opacity`, `fill-origin`, `fill-position`, `fill-repeat`, `fill-rule`, `fill-size`, `stroke`, `stroke-align`, `stroke-alignment`, `stroke-break`, `stroke-color`, `stroke-dash-corner`, `stroke-dash-justify`, `stroke-dashadjust`, `stroke-dasharray`, `stroke-dashcorner`, `stroke-dashoffset`, `stroke-image`, `stroke-linecap`, `stroke-linejoin`, `stroke-miterlimit`, `stroke-opacity`, `stroke-origin`, `stroke-position`, `stroke-repeat`, `stroke-size`, `stroke-width`. Proof: names still `unsupported` (or listed) in `mapping.json` at phase start.

### 70.2 implementation

- [ ] 70.2.1 Implement browser-level print behavior for each owned name (or alias to an existing Implemented longhand). Proof: tests cited per promotion.
- [ ] 70.2.2 Flip each finished name to `engine_status: implemented` with matrix notes. Proof: mapping + matrix.

### 70.3 gates

- [ ] 70.3.1 Targeted package tests exit 0.
- [ ] 70.3.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 70.3.3 `make test` and `make lint` exit 0 before phase complete; `make golden` if paint/layout/pagination changed.


## Out of scope

JavaScript execution. Pixel-diff Chrome goldens as the default gate. New direct Go modules without sign-off. Growing `paint_flow.go` / `paint_pagination.go` past the soft cap without extracting.

## Handoff

Next is Phase 71.
