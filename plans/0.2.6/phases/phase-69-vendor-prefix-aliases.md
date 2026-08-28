# Phase 69: Vendor-prefix aliases

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 69
> **Status:** not started
> **Estimated effort:** M
> **Owner:** `internal/layout` / `internal/css` / catalog
> **Depends on:** Phase 68
> **Unblocks:** Phase 70

---

## Overview

Implement or alias -webkit-/-moz-/-ms-/-o- prefixed properties to the unprefixed engines already shipped where possible.

Bar: **browser-level print** for the former Ignored set (247 names). Goldens stay structural unless amended. Flip mapping only with code + tests + matrix agreement.

**Count:** 70

## Goals

- Clear every owned name from the work list into Implemented, or mark `[~]` with an explicit reason
- Keep catalog counts honest after each promotion batch

## Checklist

### 69.1 scope lock

- [ ] 69.1.1 Own these 70 properties (from Phase 68 inventory): `-webkit-align-content`, `-webkit-align-items`, `-webkit-align-self`, `-webkit-animation`, `-webkit-animation-delay`, `-webkit-animation-direction`, `-webkit-animation-duration`, `-webkit-animation-fill-mode`, `-webkit-animation-iteration-count`, `-webkit-animation-name`, `-webkit-animation-play-state`, `-webkit-animation-timing-function`, `-webkit-appearance`, `-webkit-backface-visibility`, `-webkit-background-clip`, `-webkit-background-origin`, `-webkit-background-size`, `-webkit-border-bottom-left-radius`, `-webkit-border-bottom-right-radius`, `-webkit-border-radius`, `-webkit-border-top-left-radius`, `-webkit-border-top-right-radius`, `-webkit-box-align`, `-webkit-box-flex`, `-webkit-box-ordinal-group`, `-webkit-box-orient`, `-webkit-box-pack`, `-webkit-box-shadow`, `-webkit-box-sizing`, `-webkit-filter`, `-webkit-flex`, `-webkit-flex-basis`, `-webkit-flex-direction`, `-webkit-flex-flow`, `-webkit-flex-grow`, `-webkit-flex-shrink`, `-webkit-flex-wrap`, `-webkit-justify-content`, `-webkit-line-clamp`, `-webkit-mask`, `-webkit-mask-box-image`, `-webkit-mask-box-image-outset`, `-webkit-mask-box-image-repeat`, `-webkit-mask-box-image-slice`, `-webkit-mask-box-image-source`, `-webkit-mask-box-image-width`, `-webkit-mask-clip`, `-webkit-mask-composite`, `-webkit-mask-image`, `-webkit-mask-origin`, `-webkit-mask-position`, `-webkit-mask-repeat`, `-webkit-mask-size`, `-webkit-order`, `-webkit-perspective`, `-webkit-perspective-origin`, `-webkit-text-fill-color`, `-webkit-text-size-adjust`, `-webkit-text-stroke`, `-webkit-text-stroke-color`, `-webkit-text-stroke-width`, `-webkit-transform`, `-webkit-transform-origin`, `-webkit-transform-style`, `-webkit-transition`, `-webkit-transition-delay`, `-webkit-transition-duration`, `-webkit-transition-property`, `-webkit-transition-timing-function`, `-webkit-user-select`. Proof: names still `unsupported` (or listed) in `mapping.json` at phase start.

### 69.2 implementation

- [ ] 69.2.1 Implement browser-level print behavior for each owned name (or alias to an existing Implemented longhand). Proof: tests cited per promotion.
- [ ] 69.2.2 Flip each finished name to `engine_status: implemented` with matrix notes. Proof: mapping + matrix.

### 69.3 gates

- [ ] 69.3.1 Targeted package tests exit 0.
- [ ] 69.3.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 69.3.3 `make test` and `make lint` exit 0 before phase complete; `make golden` if paint/layout/pagination changed.


## Out of scope

JavaScript execution. Pixel-diff Chrome goldens as the default gate. New direct Go modules without sign-off. Growing `paint_flow.go` / `paint_pagination.go` past the soft cap without extracting.

## Handoff

Next is Phase 70.
