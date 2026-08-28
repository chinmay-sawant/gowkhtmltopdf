# Phase 72: Scroll and overscroll

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 72
> **Status:** not started
> **Estimated effort:** L
> **Owner:** `internal/layout` / `internal/css` / catalog
> **Depends on:** Phase 71
> **Unblocks:** Phase 73

---

## Overview

scroll-* / overscroll-* / scrollbar-* behavior that affects print fragmentation or layout; document PDF no-ops honestly where scroll cannot exist.

Bar: **browser-level print** for the former Ignored set (247 names). Goldens stay structural unless amended. Flip mapping only with code + tests + matrix agreement.

**Count:** 41

## Goals

- Clear every owned name from the work list into Implemented, or mark `[~]` with an explicit reason
- Keep catalog counts honest after each promotion batch

## Checklist

### 72.1 scope lock

- [ ] 72.1.1 Own these 41 properties (from Phase 68 inventory): `overscroll-behavior`, `overscroll-behavior-block`, `overscroll-behavior-inline`, `overscroll-behavior-x`, `overscroll-behavior-y`, `scroll-axis-lock`, `scroll-behavior`, `scroll-initial-target`, `scroll-margin`, `scroll-margin-block`, `scroll-margin-block-end`, `scroll-margin-block-start`, `scroll-margin-bottom`, `scroll-margin-inline`, `scroll-margin-inline-end`, `scroll-margin-inline-start`, `scroll-margin-left`, `scroll-margin-right`, `scroll-margin-top`, `scroll-marker-group`, `scroll-padding`, `scroll-padding-block`, `scroll-padding-block-end`, `scroll-padding-block-start`, `scroll-padding-bottom`, `scroll-padding-inline`, `scroll-padding-inline-end`, `scroll-padding-inline-start`, `scroll-padding-left`, `scroll-padding-right`, `scroll-padding-top`, `scroll-snap-align`, `scroll-snap-stop`, `scroll-snap-type`, `scroll-target-group`, `scroll-timeline`, `scroll-timeline-axis`, `scroll-timeline-name`, `scrollbar-color`, `scrollbar-gutter`, `scrollbar-width`. Proof: names still `unsupported` (or listed) in `mapping.json` at phase start.

### 72.2 implementation

- [ ] 72.2.1 Implement browser-level print behavior for each owned name (or alias to an existing Implemented longhand). Proof: tests cited per promotion.
- [ ] 72.2.2 Flip each finished name to `engine_status: implemented` with matrix notes. Proof: mapping + matrix.

### 72.3 gates

- [ ] 72.3.1 Targeted package tests exit 0.
- [ ] 72.3.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 72.3.3 `make test` and `make lint` exit 0 before phase complete; `make golden` if paint/layout/pagination changed.


## Out of scope

JavaScript execution. Pixel-diff Chrome goldens as the default gate. New direct Go modules without sign-off. Growing `paint_flow.go` / `paint_pagination.go` past the soft cap without extracting.

## Handoff

Next is Phase 73.
