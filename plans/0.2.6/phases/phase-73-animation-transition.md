# Phase 73: Animation and transition

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 73
> **Status:** not started (28 names unsupported)
> **Estimated effort:** XL
> **Owner:** `internal/css` + `internal/layout`
> **Depends on:** Phase 72
> **Unblocks:** Phase 74
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

`@keyframes` is parse-skipped (`internal/css/css.go`). Animation/transition props fall through `applyIgnoredGroup`. Print policy may be “use cascaded end state only,” but that is **not** Implemented animation.

## Owned names (28)

`animation`, `animation-composition`, `animation-delay`, `animation-delay-end`, `animation-delay-start`, `animation-direction`, `animation-duration`, `animation-fill-mode`, `animation-iteration-count`, `animation-name`, `animation-play-state`, `animation-range`, `animation-range-center`, `animation-range-end`, `animation-range-start`, `animation-timeline`, `animation-timing-function`, `animation-trigger`, `transition`, `transition-behavior`, `transition-delay`, `transition-duration`, `transition-property`, `transition-timing-function`, `view-transition-class`, `view-transition-group`, `view-transition-name`, `view-transition-scope`

## Work order (code)

1. Decide print semantics in writing on this file before coding: e.g. apply `@keyframes` **to** / **100%** / last keyframe as used style for print, ignore timelines.
2. If that is the bar: parse `@keyframes` rules into a structure; resolve `animation-name` against them; bake used `transform`/opacity/etc. into computed style before layout.
3. Files: `internal/css/css.go` (stop skipping keyframes), new `internal/css/keyframes.go`, cascade hook in `style_cascade.go`, consumers already exist for transform/opacity.
4. Tests must show `animation: spin 1s` with `@keyframes` changes the **used** transform/opacity for print. `TestTransformKeyframesStaticCascaded` (static transform while animation ignored) is the **opposite** proof.

## Checklist

- [x] 73.1.1 Ownership list locked.
- [ ] 73.2.1 Write print semantics (to/from/timeline policy).
- [ ] 73.2.2 Parse keyframes + apply used values.
- [ ] 73.2.3 Tests proving animation affects used style.
- [ ] 73.2.4 Mapping flips only for delivered names.
- [ ] 73.3.1 Gates + `--check`.

## Forbidden proofs

- Tests that assert animation is ignored
- Skipping `@keyframes` while marking `animation-*` Implemented
