# Phase 75: Anchor, offset, view timelines

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 75
> **Status:** not started (21 names unsupported)
> **Estimated effort:** XL
> **Owner:** `internal/layout` / `internal/css`
> **Depends on:** Phase 74
> **Unblocks:** Phase 76
> **Honesty:** `../HONESTY-GATES.md`

---

## Owned names (21)

`anchor-name`, `anchor-scope`, `offset`, `offset-anchor`, `offset-distance`, `offset-path`, `offset-position`, `offset-rotate`, `overflow-anchor`, `position-anchor`, `position-area`, `position-try`, `position-try-fallbacks`, `position-try-order`, `position-visibility`, `timeline-scope`, `view-timeline`, `view-timeline-axis`, `view-timeline-inset`, `view-timeline-name`, `will-change`

## Work order (code)

1. These names currently hit `applyIgnoredGroup` only. Citing `style_cascade.go` fallthrough or `applyIgnoredGroup` is **forbidden** as proof.
2. Implement in dependency order if you ship any:
   - `anchor-name` / `position-anchor` + abspos positioning adjustment in absolute layout (`buildAbsolute` / related).
   - `offset-*` motion path only if you have a paint consumer.
   - `view-timeline*` / `timeline-scope` need a timeline model; for print, document `[~]` unless you bake a used value.
3. Touch files will be new (`anchor.go`) plus `style.go` / `style_properties.go` / absolute positioning.
4. Tests must set the owned property names and assert box positions/paint change.

## Checklist

- [x] 75.1.1 Ownership list locked.
- [ ] 75.2.1 Choose subset + write semantics (or leave all unsupported).
- [ ] 75.2.2 Fields + apply + consumer + tests for that subset.
- [ ] 75.2.3 Mapping flips only for delivered names.
- [ ] 75.3.1 Gates.

## Forbidden proofs

- Ignore fallthrough line citations
- `overflow_clip.go` as anchor proof
