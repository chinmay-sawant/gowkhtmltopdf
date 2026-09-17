# Phase 88.3: SVG paint bake

> **Parent:** `../88-canonical-0.2.7-next-100.md`
> **Status:** planned
> **Estimated effort:** S-M (12 properties)
> **Owner:** `internal/layout` + `internal/svg`
> **Depends on:** none
> **Unblocks:** 88.4
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

Inline SVG CSS already stores dash/cap/join/miterlimit, and `writeSVGPresentationAttrs` already bakes fill/stroke/opacity/width. This batch bakes the stored dash fields and adds apply+bake for fill-rule, paint-order, stops, text-anchor, and two hints. Do not flip Implemented on store-only.

## Properties (12)

`stroke-dasharray`, `stroke-dashoffset`, `stroke-linecap`, `stroke-linejoin`, `stroke-miterlimit`, `fill-rule`, `paint-order`, `stop-color`, `stop-opacity`, `text-anchor`, `shape-rendering`, `color-interpolation`

| # | Property | Family | Effort | Demo |
|--:|----------|--------|--------|------|
| 12 | `stroke-dasharray` | svg | S | `8 4` |
| 13 | `stroke-dashoffset` | svg | S | `4` |
| 14 | `stroke-linecap` | svg | S | `round` |
| 15 | `stroke-linejoin` | svg | S | `bevel` |
| 16 | `stroke-miterlimit` | svg | S | `2` |
| 17 | `fill-rule` | svg | S | `evenodd` |
| 18 | `paint-order` | svg | M | `stroke fill` |
| 19 | `stop-color` | svg | M | `#e67e22` |
| 20 | `stop-opacity` | svg | S | `0.3` |
| 21 | `text-anchor` | svg | S | `middle` |
| 22 | `shape-rendering` | svg | S | `crispEdges` |
| 23 | `color-interpolation` | svg | S | `linearRGB` |

## Architecture

- Extend `writeSVGPresentationAttrs` in `layout_svg.go`.
- Fill the leftover hole: `applySVGPresentationProps` lists fill-rule and friends, then `applyLeftoversProps` returns false.
- Raster consumer stays `internal/svg.Rasterize` via canvas.
- Hints (`shape-rendering`, `color-interpolation`) stay Partial if canvas ignores them.

## Checklist

### 88.3.1 scope lock

- [ ] 88.3.1.1 Confirm the 12 names below against `../next-100-properties.json`. Proof: list in `_proof-88.3.md`.

### 88.3.2 `stroke-dasharray`

- [ ] 88.3.2.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_leftovers.go; internal/layout/layout_svg.go`).
- [ ] 88.3.2.2 Consumer reads the field. Demo `8 4`.
- [ ] 88.3.2.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.3 `stroke-dashoffset`

- [ ] 88.3.3.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_leftovers.go; internal/layout/layout_svg.go`).
- [ ] 88.3.3.2 Consumer reads the field. Demo `4`.
- [ ] 88.3.3.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.4 `stroke-linecap`

- [ ] 88.3.4.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_leftovers.go; internal/layout/layout_svg.go`).
- [ ] 88.3.4.2 Consumer reads the field. Demo `round`.
- [ ] 88.3.4.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.5 `stroke-linejoin`

- [ ] 88.3.5.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_leftovers.go; internal/layout/layout_svg.go`).
- [ ] 88.3.5.2 Consumer reads the field. Demo `bevel`.
- [ ] 88.3.5.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.6 `stroke-miterlimit`

- [ ] 88.3.6.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_leftovers.go; internal/layout/layout_svg.go`).
- [ ] 88.3.6.2 Consumer reads the field. Demo `2`.
- [ ] 88.3.6.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.7 `fill-rule`

- [ ] 88.3.7.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go; internal/layout/layout_svg.go`).
- [ ] 88.3.7.2 Consumer reads the field. Demo `evenodd`.
- [ ] 88.3.7.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.8 `paint-order`

- [ ] 88.3.8.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go; internal/layout/layout_svg.go`).
- [ ] 88.3.8.2 Consumer reads the field. Demo `stroke fill`.
- [ ] 88.3.8.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.9 `stop-color`

- [ ] 88.3.9.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go; internal/layout/layout_svg.go`).
- [ ] 88.3.9.2 Consumer reads the field. Demo `#e67e22`.
- [ ] 88.3.9.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.10 `stop-opacity`

- [ ] 88.3.10.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go; internal/layout/layout_svg.go`).
- [ ] 88.3.10.2 Consumer reads the field. Demo `0.3`.
- [ ] 88.3.10.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.11 `text-anchor`

- [ ] 88.3.11.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go; internal/layout/layout_svg.go`).
- [ ] 88.3.11.2 Consumer reads the field. Demo `middle`.
- [ ] 88.3.11.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.12 `shape-rendering`

- [ ] 88.3.12.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go; internal/layout/layout_svg.go`).
- [ ] 88.3.12.2 Consumer reads the field. Demo `crispEdges`.
- [ ] 88.3.12.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.13 `color-interpolation`

- [ ] 88.3.13.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go; internal/layout/layout_svg.go`).
- [ ] 88.3.13.2 Consumer reads the field. Demo `linearRGB`.
- [ ] 88.3.13.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.3.R batch gate (package only)

- [ ] 88.3.R.1 Targeted `go test ./internal/<pkg> -run '…' -count=1` exit 0 (skip for 88.8 defer-only).
- [ ] 88.3.R.2 `python3 scripts/css-catalog-map.py --check` if apply arms added.
- [ ] 88.3.R.3 Mapping + matrix only for properties with flip packets. **Do not** run `make test` / `make lint` here.

## Out of scope

Geometry CSS (`cx`/`d`), markers, vector-effect.
