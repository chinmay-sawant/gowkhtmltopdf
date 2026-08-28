# Phase 71: Mask, clip, and filter

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 71
> **Status:** not started (`filter` Partial opacity-only; rest unsupported)
> **Estimated effort:** L
> **Owner:** `internal/layout` paint
> **Depends on:** Phase 70
> **Unblocks:** Phase 72
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

`filter` today only folds `opacity()` (`style_properties.go` + `transform.go` parseFilterOpacity). `overflow: clip` is unrelated to CSS `clip` / `clip-path`.

## Owned names (26)

`backdrop-filter`, `clip`, `clip-path`, `clip-rule`, `color-interpolation-filters`, `filter`, `flood-color`, `flood-opacity`, `lighting-color`, `mask`, `mask-border`, `mask-border-mode`, `mask-border-outset`, `mask-border-repeat`, `mask-border-slice`, `mask-border-source`, `mask-border-width`, `mask-clip`, `mask-composite`, `mask-image`, `mask-mode`, `mask-origin`, `mask-position`, `mask-repeat`, `mask-size`, `mask-type`

## Work order (code)

1. **clip-path / clip:** parse basic shapes or `inset()` into a clip path; apply in paint (`paint.go` / clip stack). New fields on `ResolvedStyle`.
2. **mask*:** at least `mask-image` url + mask paint path, or keep unsupported until you can ship a consumer.
3. **filter / backdrop-filter:** extend beyond `opacity()` only if you implement blur/etc. in paint; otherwise leave `filter` Partial with notes and `backdrop-filter` unsupported.
4. Tests must target the owned property names, not `TestOverflowClip*`.

## Checklist

- [x] 71.1.1 Ownership list locked.
- [ ] 71.2.1 Implement clip-path or document `[~]`.
- [ ] 71.2.2 Implement mask subset or leave unsupported.
- [ ] 71.2.3 Expand filter functions or keep Partial opacity-only with notes.
- [ ] 71.2.4 Flip packets + matrix.
- [ ] 71.3.1 Targeted layout paint tests; `--check`; gates.

## Forbidden proofs

- `TestOverflowClip` as proof of `clip-path`
- Marking full `filter` Implemented while only `opacity()` works
