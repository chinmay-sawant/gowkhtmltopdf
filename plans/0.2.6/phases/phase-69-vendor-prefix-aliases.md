# Phase 69: Vendor-prefix aliases

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 69
> **Status:** not started (70 names unsupported; fake Implemented reverted)
> **Estimated effort:** M
> **Owner:** `internal/layout` cascade/apply
> **Depends on:** Phase 68
> **Unblocks:** Phase 70
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

Implement aliasing so `-webkit-*` / other owned prefixes map onto existing unprefixed engines where those engines already exist (flex, radius, transform 2D, box-shadow, …).

## Owned names (70)

`-webkit-align-content`, `-webkit-align-items`, `-webkit-align-self`, `-webkit-animation`, `-webkit-animation-delay`, `-webkit-animation-direction`, `-webkit-animation-duration`, `-webkit-animation-fill-mode`, `-webkit-animation-iteration-count`, `-webkit-animation-name`, `-webkit-animation-play-state`, `-webkit-animation-timing-function`, `-webkit-appearance`, `-webkit-backface-visibility`, `-webkit-background-clip`, `-webkit-background-origin`, `-webkit-background-size`, `-webkit-border-bottom-left-radius`, `-webkit-border-bottom-right-radius`, `-webkit-border-radius`, `-webkit-border-top-left-radius`, `-webkit-border-top-right-radius`, `-webkit-box-align`, `-webkit-box-flex`, `-webkit-box-ordinal-group`, `-webkit-box-orient`, `-webkit-box-pack`, `-webkit-box-shadow`, `-webkit-box-sizing`, `-webkit-filter`, `-webkit-flex`, `-webkit-flex-basis`, `-webkit-flex-direction`, `-webkit-flex-flow`, `-webkit-flex-grow`, `-webkit-flex-shrink`, `-webkit-flex-wrap`, `-webkit-justify-content`, `-webkit-line-clamp`, `-webkit-mask`, `-webkit-mask-box-image`, `-webkit-mask-box-image-outset`, `-webkit-mask-box-image-repeat`, `-webkit-mask-box-image-slice`, `-webkit-mask-box-image-source`, `-webkit-mask-box-image-width`, `-webkit-mask-clip`, `-webkit-mask-composite`, `-webkit-mask-image`, `-webkit-mask-origin`, `-webkit-mask-position`, `-webkit-mask-repeat`, `-webkit-mask-size`, `-webkit-order`, `-webkit-perspective`, `-webkit-perspective-origin`, `-webkit-text-fill-color`, `-webkit-text-size-adjust`, `-webkit-text-stroke`, `-webkit-text-stroke-color`, `-webkit-text-stroke-width`, `-webkit-transform`, `-webkit-transform-origin`, `-webkit-transform-style`, `-webkit-transition`, `-webkit-transition-delay`, `-webkit-transition-duration`, `-webkit-transition-property`, `-webkit-transition-timing-function`, `-webkit-user-select`

## Work order (code)

1. Add a prefix-strip / alias step **before** apply dispatch:
   - Preferred: `internal/layout/style_cascade.go` when reading declarations into apply, or a helper used by `applyStyleProp`.
2. Map only prefixes you claim (start with `-webkit-`). Example: `-webkit-flex-direction` → apply `flex-direction`.
3. Do **not** mark Implemented unless the unprefixed longhand already has an apply arm **and** a test sets the **prefixed** name and asserts the same `ResolvedStyle` effect.
4. Names whose unprefixed form is still unsupported stay unsupported (aliasing alone is not enough).

## Acceptance tests

- New: `TestWebkitFlexDirectionAliases` (and similar) in `internal/layout` using prefixed props.
- Proofs that only exercise unprefixed `TestFlex*` / `TestRadius*` are **forbidden** as close criteria.

## Checklist

- [x] 69.1.1 Ownership list locked (above).
- [ ] 69.2.1 Implement alias helper + wire into cascade/apply.
- [ ] 69.2.2 Prefixed-name tests for each alias you claim.
- [ ] 69.2.3 Flip mapping only for aliases with flip packets; leave others unsupported.
- [ ] 69.3.1 `go test ./internal/layout -run "TestWebkit|TestFlex|TestRadius|TestTransform|TestBoxShadow"`; `--check`; `make test` / `make lint`.

## Forbidden proofs

- Citing unprefixed tests only
- Mass-flipping all 70 because “webkit is like the standard property”
