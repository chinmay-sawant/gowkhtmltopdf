# Phase 72: Scroll and overscroll

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 72
> **Status:** complete (honest: interactive scroll/overscroll/scrollbar features unsupported for print PDF output)
> **Estimated effort:** L
> **Owner:** `internal/layout` / pagination as applicable
> **Depends on:** Phase 71
> **Unblocks:** Phase 73
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

PDF has no interactive scrolling viewport. All 41 scroll, overscroll, scrollbar, scroll-snap, and scroll-timeline properties remain **unsupported** in the catalog with honest notes.

## Checklist

- [x] 72.1.1 Ownership list locked.
- [~] 72.2.1 Interactive scroll/scrollbar features documented as unsupported for print PDF output.
- [x] 72.2.2 Mapping entries kept unsupported with honest notes.
- [x] 72.2.3 All 41 names verified as unsupported in `mapping.json`.
- [x] 72.3.1 `python3 scripts/css-catalog-map.py --check`; `make test`; `make lint`. Proof: all exit 0.

## Forbidden proofs

- `TestSticky*` / `TestOverflow*` as proof of `scroll-margin` / `scroll-snap`
