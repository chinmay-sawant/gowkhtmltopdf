# Phase 74: 3D transforms

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 74
> **Status:** not started (perspective/transform-style/backface unsupported; 3D funcs rejected)
> **Estimated effort:** L
> **Owner:** `internal/layout`
> **Depends on:** Phase 73
> **Unblocks:** Phase 75
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

Today only **2D** `transform` / `transform-origin` work (`Matrix2D` in `internal/layout/transform.go`). Line 236 rejects 3D / perspective / `matrix3d`. There are **no** apply arms for `perspective`, `perspective-origin`, `transform-style`, `backface-visibility`.

## Owned names (4)

`backface-visibility`, `perspective`, `perspective-origin`, `transform-style`

## Work order (code)

1. Add `ResolvedStyle` fields for the four properties in `style.go`.
2. Add `case` arms in `applyTransformGroup` (`style_properties.go`).
3. Extend transform math beyond `Matrix2D` **or** define a print-static subset (e.g. flatten perspective to 2D) and document it in matrix notes. Either way, `parseOneTransformFunc` must accept the 3D functions you claim (`matrix3d`, `translate3d`, `rotateX`, …) instead of rejecting them.
4. Paint: apply resulting matrix in the existing CTM paint path.
5. Tests:
   - Replace reliance on `TestParseTransformNoneAnd3DRejected` for success.
   - Add tests that `perspective` / `rotateX` (or your claimed subset) affect paint transforms.
6. Update matrix: remove “3D permanent non-goal” for the subset you ship.

## Checklist

- [x] 74.1.1 Ownership list locked.
- [ ] 74.2.1 Fields + apply arms for the four properties.
- [ ] 74.2.2 Parse/paint path for claimed 3D functions (stop rejecting those).
- [ ] 74.2.3 Tests that fail on current reject path and pass after.
- [ ] 74.2.4 Mapping + matrix flip packets.
- [ ] 74.3.1 `go test ./internal/layout -run "TestTransform|TestPerspective|TestBackface"`; `--check`; gates; golden if paint changes.

## Forbidden proofs

- Citing `transform.go` reject comment / `TestParseTransformNoneAnd3DRejected` as Implemented proof
- Empty `code_path` Implemented rows
- 2D-only tests (`TestParseTransformTranslateRotateScale`) as 3D proof
