# Phase 3 — CSS convergence and layout state

> **Parent:** [`../architecture-review-2026-08-07-followup.md`](../architecture-review-2026-08-07-followup.md)
> **Status:** complete
> **Depends on:** Phase 2

## Goal

Keep CSS query inputs correct across refinement and keep display-list ownership,
replaced-element geometry, and cancellation local to layout.

## Checklist

- [x] **CSS-01** — `sameSizeContainerState` includes used `fontSize` and is the
  single convergence comparison. Nested container-query regression coverage
  passes; `BenchmarkUsedImageSize` captured `80.93 ns/op`, `48 B/op`, and
  `1 allocs/op` in the final run.
- [x] **LAYOUT-01** — stable non-zero `Op.ID` values and source-span remapping
  preserve box ranges and element locations after rectangle splitting.
  `TestSplitCrossingRectsRemapsBoxRangeAndPreservesIdentity` passes;
  `BenchmarkDisplayListIdentity10kOps100Pages` exercised 10,000 operations over
  100 pages at `8,554,807 ns/op`, `8,635,688 B/op`, and `92,644 allocs/op`.
- [x] **IMG-01** — `usedImageSize` is the one policy for intrinsic dimensions,
  attributes, CSS dimensions, aspect ratio, max clamps, float, inline, and
  table constraints. Height-only/constraint regression tests and the focused
  benchmark pass.
- [x] **X-02 layout seam** — `LayoutContext` adds cancellation checkpoints
  while preserving the `Layout` compatibility wrapper; paint-side context
  checkpoints were added for full PDF/header/footer traversal.

## Required gate

- [x] Final Phase 3 gate: `make lint`, `make test`, `go test -race ./...`, and
  `go test ./internal/css ./internal/layout -count=1` passed on 2026-08-07.
