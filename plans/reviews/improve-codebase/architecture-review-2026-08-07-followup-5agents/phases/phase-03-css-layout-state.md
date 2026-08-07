# Phase 3 — CSS convergence and layout state

> **Parent:** [`../architecture-review-2026-08-07-followup.md`](../architecture-review-2026-08-07-followup.md)
> **Status:** not started
> **Depends on:** Phase 2

## Goal

Keep CSS query inputs correct across refinement and keep display-list ownership and
replaced-element geometry local to layout.

## Checklist

- [ ] **CSS-01** — include `sizeContainer.fontSize` in convergence equality and
  centralize the state comparison. Proof: nested `em` container-query regression and
  deep-container pass-count/allocations benchmark.
- [ ] **LAYOUT-01** — preserve stable display-list identity when `splitCrossingRects`
  or later paint-time rewriting rebuilds `Result.Ops`. Proof: element-location/page
  ownership regression after an earlier rectangle split and a 10k-op/100-page
  benchmark.
- [ ] **IMG-01** — centralize used image sizing for intrinsic dimensions, attributes,
  CSS dimensions, aspect ratio, max clamps, and float/table constraints. Proof:
  height-only, max-width/max-height, nested float/table, and loader-call tests.

## Required gate

- [ ] Run `make lint` and `make test`; run `go test ./internal/css ./internal/layout
  -count=1` and record the layout benchmark with dataset and cache state.
