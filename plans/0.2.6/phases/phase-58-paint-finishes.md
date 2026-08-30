# Phase 58: Paint finishes

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 58
> **Status:** complete (honest: outline/radius Implemented; box-shadow/background* kept Partial with spread support + honest notes)
> **Estimated effort:** M
> **Owner:** `internal/layout`
> **Depends on:** Phase 57
> **Unblocks:** Phase 59 (already honest) / Partial honesty for paint subset
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

Outline and border-radius are already Implemented with real paint. `box-shadow` deepened with `spread` support in paint and parsing; `box-shadow`, `background`, `background-image` kept honestly as Partial with exact documented subset in mapping and matrix.

## Owned reopen set

`box-shadow`, `background`, `background-image`

## Work order (code)

### box-shadow

1. Edit `internal/layout/box_shadow.go` and the apply path in `internal/layout/style_paint_props.go` / `style_properties.go`.
2. Spread support implemented: `BoxShadowSpread` parsed and expands the shadow bounding box in `appendBoxShadow`.
3. Kept Partial with notes: first un-inset layer, inset ignored, spread expanded, blur approximated.
4. Tests: `TestBoxShadowParse` (spread parsed), `TestBoxShadowSpreadFill` (spread fill offset and expanded dimensions).

### background / background-image

1. Kept Partial with exact honest notes in mapping.json and compatibility matrix §2.4: first `url(...)` layer, no-repeat, box-sized, gradients ignored.
2. Tests: `background_image_test.go`.

## Checklist

- [x] 58.R.1 Choose bar per property: deepen `box-shadow` with spread support; keep `box-shadow`/`background`/`background-image` Partial with honest notes. Proof: `internal/layout/box_shadow.go:88`, `style.go:232`.
- [x] 58.R.2 Code + tests for each deepened property (flip packet from `HONESTY-GATES.md`). Proof: `TestBoxShadowParse`, `TestBoxShadowSpreadFill` in `box_shadow_test.go`.
- [x] 58.R.3 Update matrix + mapping + `property-counts.md` + `coverage-summary.json`. Proof: Matrix §2.4 and `mapping.json` updated.
- [x] 58.R.4 `go test ./internal/layout -run "TestBoxShadow|TestBackground|TestRadius|TestOutline"`; `python3 scripts/css-catalog-map.py --check`; `make test` / `make lint`. Proof: all tests exit 0.

## Forbidden proofs

- `TestBoxShadowParse` asserting inset ignored
- Empty `code_path` Implemented rows
- Catalog-only edits

## Handoff

Next open honesty rows: Phase 61, 63, then Ignored reopen 69+.
