# Agent 3 — CSS and layout follow-up

## Scope

- CSS-01: container-query convergence state.
- LAYOUT-01: display-list identity and location ownership across pagination rewrites.
- IMG-01: one used-image sizing policy for intrinsic, HTML attribute, CSS,
  aspect-ratio, max, float, inline, and table-cell paths.
- X-02: compatible context-aware layout entry point and checkpoints.

## Changes

- Added `sameSizeContainerState` for container convergence and included used
  `fontSize` alongside inline size and names. This prevents an `em`-based
  container query from incorrectly converging when only the container's font
  size changes.
- Added stable non-zero `Op.ID` values for layout-generated operations and
  preserved the source identity on pagination fragments.
- Changed `splitCrossingRects` to retain source-to-output spans and remap the
  complete box operation range. Element ownership, sticky/fixed stamping, and
  later `Locations` calculation now include every fragment of a split rect.
- Centralized image used-size calculation in `engine.usedImageSize`. It is
  shared by block/inline image construction and intrinsic width measurement;
  it applies intrinsic dimensions, HTML width/height attributes, CSS absolute
  and percentage dimensions, intrinsic aspect-ratio preservation, max-width,
  max-height, and containing-block limits from floats, inline formatting, and
  table cells.
- Added `LayoutContext(ctx, root, opts)`. Existing `Layout` callers remain
  source-compatible through a background-context wrapper. Cancellation is
  checked before style work, at recursive box construction boundaries, while
  flowing children, and before display-list emission.
- Added focused regression tests and `BenchmarkUsedImageSize` in
  `internal/layout/architecture_followup_test.go`.

## Validation

- `go test ./internal/css ./internal/layout` — PASS.
- `go test ./internal/layout -run 'Test(ContainerStateEqualityIncludesFontSize|SplitCrossingRectsRemapsBoxRangeAndPreservesIdentity|UsedImageSizeUsesOneAspectAndConstraintPolicy|LayoutContextHonorsCancellation)$'` — PASS.
- `go test ./internal/layout -run '^$' -bench BenchmarkUsedImageSize -benchmem` — PASS (`79.47 ns/op`, `48 B/op`, `1 allocs/op` in this run).
- `go test ./...` — PASS.

No Git commands were run and no phase checklist or `fix-contract.md` file was
modified.
