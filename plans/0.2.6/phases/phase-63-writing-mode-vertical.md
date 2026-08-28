# Phase 63: writing-mode vertical

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 63
> **Status:** reopen (writing-mode is Partial: glyph-rotate only)
> **Estimated effort:** XL
> **Owner:** `internal/layout`
> **Depends on:** Phase 62
> **Unblocks:** logical-box vertical mapping leftovers
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

`writing-mode` is **Partial**. Vertical values set `RotateDeg == -90` and bump box height from rotated run width. Block/IFC flow stays horizontal (`inline_paint.go`, `layout.go`).

Implemented requires real vertical line progression for `vertical-rl` / `vertical-lr`.

## Work order (code)

1. **Stop treating rotate-as-done.** Do not flip mapping while `writingModeRotate` is the only vertical effect (`internal/layout/inline_paint.go`).
2. Teach line building / inline layout to advance on the block axis for vertical modes:
   - Likely files: `internal/layout/inline.go`, `inline_paint.go`, `layout.go`, measure helpers in `layout_measure.go`.
3. Distinguish `vertical-rl` vs `vertical-lr` (block-direction sign / packing).
4. Orthogonal flows: logical margin/padding mapping when writing-mode is vertical (ties to Phase 59).
5. Tests:
   - New: vertical text advances downward/leftward as claimed (box geometry assertions), not merely `RotateDeg == -90`.
   - Existing `TestWritingModeInherits` is **insufficient** alone.

## Checklist

- [x] 63.1.1 Owned property: `writing-mode` (still in mapping as partial after honesty revert).
- [ ] 63.2.1 Implement vertical line progression (not glyph-rotate-only).
- [ ] 63.2.2 Tests that fail under rotate-only and pass under real vertical flow.
- [ ] 63.2.3 Update matrix §2.3 / §3 and `documentation/deferred.md` so they stop contradicting each other.
- [ ] 63.2.4 Flip mapping to Implemented only with flip packet; set `code_path` + notes.
- [ ] 63.3.1 `go test ./internal/layout -run "TestWritingMode"`; `--check`; `make test` / `make lint` / `make golden` if geometry changes.

## Forbidden proofs

- `RotateDeg == -90` as sole success criterion
- Inherit-only tests
- Catalog flip without layout consumer change

## Handoff

Phase 67 closure stays reopen until this and other honesty demotes are resolved or explicitly `[~]`.
