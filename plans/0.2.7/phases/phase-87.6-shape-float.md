# Phase 87.6: CSS Shapes + page-float extras

> **Parent:** `../87-canonical-0.2.7-next-72.md`
> **Status:** complete
> **Estimated effort:** L–XL (8 properties)
> **Owner:** `internal/layout` (float + inline + pagination for page floats)
> **Depends on:** existing rectangular floats (`float.go`); 87.5 optional
> **Unblocks:** richer wrap demos on fixture-64
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

Today floats are CSS2 rectangular exclusions only (`float.go` `exclusion` ~
158-181; `inline.go` `lineBounds` ~351-363). There is no CSS Shapes path and no
page-float model.

Keep rectangular placement; add a parallel exclusion query for shaped contours.
Do not grow `layout.go` / `style_properties.go`.

## Properties (8)

**Shape (5):** `shape-image-threshold`, `shape-inside`, `shape-margin`,
`shape-outside`, `shape-padding`

**Float page (3):** `float-defer`, `float-offset`, `float-reference`

## Honest bar (recommended)

| Name | Bar |
|------|-----|
| `shape-outside` | Partial/Implemented lite: `circle()` / `ellipse()` / `inset()` with `float:left|right` |
| `shape-margin` | With outside |
| `shape-image-threshold` | Partial only if alpha contour exists; else store+no-op is not Implemented |
| `shape-inside`, `shape-padding` | Prefer stay Unsupported (Chrome no; interior fitting) |
| `float-offset` | Lite length nudge on `placeFloat` |
| `float-reference` | Lite: `inline` = current BFC; `page`/`column` need pagination |
| `float-defer` | XL / Partial unless a tiny defer model is intentional |

## New files

- `style_shape_props.go` + `shape_exclusion.go`
- `style_float_page_props.go`
- Optional pagination hooks near `pagination_ctx.go` / `paint_flow_*` for page floats only if shipping

## Checklist

### 87.6.1 shape-outside lite

- [x] 87.6.1.1 Apply `shape-outside` / `shape-margin`; parse basic shapes.
- [x] 87.6.1.2 `shape_exclusion.go` returns per-line intervals; `lineBounds` consults it when present.
- [x] 87.6.1.3 Tests: `TestShapeOutsideCircleShortensLines`, `TestShapeMarginExpandsExclusion`.
- [x] 87.6.1.4 Flip only with consumer proof; matrix names the basic-shape subset.

### 87.6.2 shape-image-threshold / inside / padding

- [x] 87.6.2.1 Decide: implement alpha contour (**L**) or leave Unsupported.
- [x] 87.6.2.2 `shape-inside` / `shape-padding`: default stay Unsupported unless product insists.

### 87.6.3 float-offset / reference / defer

- [x] 87.6.3.1 Apply `float-offset` / `float-reference` in `style_float_page_props.go` (`float-defer` omitted as Unsupported).
- [x] 87.6.3.2 `float-offset` consumer on `placeFloat` (`layout_flow.go` place path) without growing `layout.go` (extract helper if needed).
- [x] 87.6.3.3 `float-reference:inline` documents current BFC; page/column Partial.
- [x] 87.6.3.4 `float-defer`: implement tiny lite or leave Unsupported/Partial with note.
- [x] 87.6.3.5 Tests matching the shipped subset.

### 87.6.R batch gate (package only)

- [x] 87.6.R.1 `go test ./internal/layout -run 'TestShapeOutside|TestShapeMargin|TestFloatOffset|TestFloatReference|TestFloatDefer' -count=1` exit 0.
- [x] 87.6.R.2 Mapping flips only for proven subset. **No `make test` / `make lint`.**

## Out of scope

Borders-4 clip (87.7). Full CSS Exclusions / Regions (permanent non-goals).
