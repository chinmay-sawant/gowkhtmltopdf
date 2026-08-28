# Phase 58: Paint finishes

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 58
> **Status:** reopen (honesty: outline/radius Implemented; box-shadow/background* still Partial)
> **Estimated effort:** M
> **Owner:** `internal/layout`
> **Depends on:** Phase 57
> **Unblocks:** Phase 59 (already honest) / Partial honesty for paint subset
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

Outline and border-radius are already Implemented with real paint. This reopen only finishes (or honestly keeps Partial) **`box-shadow`**, **`background`**, **`background-image`**.

Do **not** flip those three to Implemented by editing `mapping.json` alone.

## Owned reopen set

`box-shadow`, `background`, `background-image`

## Work order (code)

### box-shadow

1. Edit `internal/layout/box_shadow.go` and the apply path in `internal/layout/style_paint_props.go` / `style_properties.go`.
2. Today: first un-inset layer only; inset rejected; spread ignored; blur approximated with stacked fills.
3. To claim Implemented: support **multi-layer**, **inset**, and **spread** in paint, **or** keep Partial and write that exact subset in mapping `notes` + matrix (do not call it Implemented).
4. Tests: extend `internal/layout/box_shadow_test.go`. A test that asserts inset is ignored is **not** Implemented proof.

### background / background-image

1. Edit `internal/layout/background_image.go` and shorthand apply (`applyBackgroundShorthand` in style props).
2. Today: color + first `url(...)`; gradients ignored; no-repeat; sized to box.
3. To claim Implemented: multiple layers and/or `background-repeat` / `background-size` / `background-position` consumers as claimed, **or** keep Partial with notes.
4. Tests: `background_image_test.go`. Fix matrix §2.4 vs §5 contradiction when you flip.

## Checklist

- [ ] 58.R.1 Choose bar per property: deepen code **or** keep Partial with honest notes (no fake Implemented).
- [ ] 58.R.2 Code + tests for each deepened property (flip packet from `HONESTY-GATES.md`).
- [ ] 58.R.3 Update matrix + mapping + `property-counts.md` + `coverage-summary.json`.
- [ ] 58.R.4 `go test ./internal/layout -run "TestBoxShadow|TestBackground|TestRadius|TestOutline"`; `python3 scripts/css-catalog-map.py --check`; `make test` / `make lint` (and `make golden` if paint changes).

## Forbidden proofs

- `TestBoxShadowParse` asserting inset ignored
- Empty `code_path` Implemented rows
- Catalog-only edits

## Handoff

Next open honesty rows: Phase 61, 63, then Ignored reopen 69+.
