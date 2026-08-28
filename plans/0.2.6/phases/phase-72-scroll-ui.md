# Phase 72: Scroll and overscroll

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 72
> **Status:** not started (41 names unsupported)
> **Estimated effort:** L
> **Owner:** `internal/layout` / pagination as applicable
> **Depends on:** Phase 71
> **Unblocks:** Phase 73
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

PDF has no scrolling viewport. Browser-level **print** still needs honest behavior for scroll-margin/padding snap alignment used in fragmentation, or explicit unsupported.

Sticky scrollports and `overflow` clipping are **not** implementations of `scroll-margin-*`.

## Owned names (41)

`overscroll-behavior`, `overscroll-behavior-block`, `overscroll-behavior-inline`, `overscroll-behavior-x`, `overscroll-behavior-y`, `scroll-axis-lock`, `scroll-behavior`, `scroll-initial-target`, `scroll-margin`, `scroll-margin-block`, `scroll-margin-block-end`, `scroll-margin-block-start`, `scroll-margin-bottom`, `scroll-margin-inline`, `scroll-margin-inline-end`, `scroll-margin-inline-start`, `scroll-margin-left`, `scroll-margin-right`, `scroll-margin-top`, `scroll-marker-group`, `scroll-padding`, `scroll-padding-block`, `scroll-padding-block-end`, `scroll-padding-block-start`, `scroll-padding-bottom`, `scroll-padding-inline`, `scroll-padding-inline-end`, `scroll-padding-inline-start`, `scroll-padding-left`, `scroll-padding-right`, `scroll-padding-top`, `scroll-snap-align`, `scroll-snap-stop`, `scroll-snap-type`, `scroll-target-group`, `scroll-timeline`, `scroll-timeline-axis`, `scroll-timeline-name`, `scrollbar-color`, `scrollbar-gutter`, `scrollbar-width`

## Work order (code)

1. Pick a print-useful subset first: `scroll-margin*`, `scroll-padding*` affecting fragmentation/sticky containing blocks if you can define print semantics.
2. Add fields + apply arms in `style_properties.go`.
3. Consumer in `sticky.go` / pagination only if the semantics are real; otherwise leave unsupported.
4. `scroll-snap-*`, `scrollbar-*`, `scroll-timeline*`: implement only with a consumer; do not catalog-flip.

## Checklist

- [x] 72.1.1 Ownership list locked.
- [ ] 72.2.1 Written print semantics for each name you will implement (short notes in phase).
- [ ] 72.2.2 Fields + apply + consumer + tests using the **scroll-*** property names.
- [ ] 72.2.3 Mapping flips only for that subset.
- [ ] 72.3.1 Targeted tests; `--check`; gates.

## Forbidden proofs

- `TestSticky*` / `TestOverflow*` as proof of `scroll-margin` / `scroll-snap`
