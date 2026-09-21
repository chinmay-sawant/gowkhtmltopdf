# Phase 88.4: SVG geometry, markers, and text

> **Parent:** `../88-canonical-0.2.7-next-100.md`
> **Status:** planned
> **Estimated effort:** M-L (20 properties)
> **Owner:** `internal/layout` + `internal/svg`
> **Depends on:** 88.3 preferred
> **Unblocks:** 88.5
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

Bake SVG2 geometry CSS (`cx`/`cy`/`r`/`rx`/`ry`/`x`/`y`/`d`), marker urls, SVG text baselines, `image-rendering` on HTML rasters, `text-rendering`, `vector-effect: non-scaling-stroke`, and map fill-stroke-3 `fill-color` / `stroke-color` onto existing fill/stroke when the syntax matches.

## Properties (20)

`cx`, `cy`, `r`, `rx`, `ry`, `x`, `y`, `d`, `marker`, `marker-start`, `marker-mid`, `marker-end`, `dominant-baseline`, `alignment-baseline`, `baseline-shift`, `image-rendering`, `text-rendering`, `vector-effect`, `fill-color`, `stroke-color`

| # | Property | Family | Effort | Demo |
|--:|----------|--------|--------|------|
| 24 | `cx` | svg | M | `60` |
| 25 | `cy` | svg | M | `40` |
| 26 | `r` | svg | M | `25` |
| 27 | `rx` | svg | M | `20` |
| 28 | `ry` | svg | M | `10` |
| 29 | `x` | svg | M | `16` |
| 30 | `y` | svg | M | `20` |
| 31 | `d` | svg | M | `path("M10,50 L90,50")` |
| 32 | `marker` | svg | M | `url(#arr)` |
| 33 | `marker-start` | svg | M | `url(#arr)` |
| 34 | `marker-mid` | svg | M | `url(#arr)` |
| 35 | `marker-end` | svg | M | `url(#arr)` |
| 36 | `dominant-baseline` | svg | M | `hanging` |
| 37 | `alignment-baseline` | svg | M | `middle` |
| 38 | `baseline-shift` | svg | M | `super` |
| 39 | `image-rendering` | svg | M | `pixelated` |
| 40 | `text-rendering` | svg | M | `geometricPrecision` |
| 41 | `vector-effect` | svg | L | `non-scaling-stroke` |
| 42 | `fill-color` | svg | S | `#c0392b` |
| 43 | `stroke-color` | svg | S | `#2c3e50` |

## Architecture

- New `style_svg_geom_props.go` and `style_svg_marker_props.go`.
- Geometry used values override XML attributes at bake time.
- `image-rendering: pixelated` is an HTML img scaler change, not only SVG.
- `vector-effect` is L: stroke width vs CTM in `internal/svg`.

## Checklist

### 88.4.1 scope lock

- [ ] 88.4.1.1 Confirm the 20 names below against `../next-100-properties.json`. Proof: list in `_proof-88.4.md`.

### 88.4.2 `cx`

- [ ] 88.4.2.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_geom_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.2.2 Consumer reads the field. Demo `60`.
- [ ] 88.4.2.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.3 `cy`

- [ ] 88.4.3.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_geom_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.3.2 Consumer reads the field. Demo `40`.
- [ ] 88.4.3.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.4 `r`

- [ ] 88.4.4.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_geom_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.4.2 Consumer reads the field. Demo `25`.
- [ ] 88.4.4.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.5 `rx`

- [ ] 88.4.5.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_geom_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.5.2 Consumer reads the field. Demo `20`.
- [ ] 88.4.5.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.6 `ry`

- [ ] 88.4.6.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_geom_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.6.2 Consumer reads the field. Demo `10`.
- [ ] 88.4.6.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.7 `x`

- [ ] 88.4.7.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_geom_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.7.2 Consumer reads the field. Demo `16`.
- [ ] 88.4.7.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.8 `y`

- [ ] 88.4.8.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_geom_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.8.2 Consumer reads the field. Demo `20`.
- [ ] 88.4.8.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.9 `d`

- [ ] 88.4.9.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_geom_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.9.2 Consumer reads the field. Demo `path("M10,50 L90,50")`.
- [ ] 88.4.9.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.10 `marker`

- [ ] 88.4.10.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_marker_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.10.2 Consumer reads the field. Demo `url(#arr)`.
- [ ] 88.4.10.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.11 `marker-start`

- [ ] 88.4.11.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_marker_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.11.2 Consumer reads the field. Demo `url(#arr)`.
- [ ] 88.4.11.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.12 `marker-mid`

- [ ] 88.4.12.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_marker_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.12.2 Consumer reads the field. Demo `url(#arr)`.
- [ ] 88.4.12.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.13 `marker-end`

- [ ] 88.4.13.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_svg_marker_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.13.2 Consumer reads the field. Demo `url(#arr)`.
- [ ] 88.4.13.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.14 `dominant-baseline`

- [ ] 88.4.14.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.14.2 Consumer reads the field. Demo `hanging`.
- [ ] 88.4.14.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.15 `alignment-baseline`

- [ ] 88.4.15.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.15.2 Consumer reads the field. Demo `middle`.
- [ ] 88.4.15.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.16 `baseline-shift`

- [ ] 88.4.16.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.16.2 Consumer reads the field. Demo `super`.
- [ ] 88.4.16.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.17 `image-rendering`

- [ ] 88.4.17.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_image_adjust_props.go; internal/imageout`).
- [ ] 88.4.17.2 Consumer reads the field. Demo `pixelated`.
- [ ] 88.4.17.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.18 `text-rendering`

- [ ] 88.4.18.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_font_feature_props.go; internal/layout/layout_svg.go`).
- [ ] 88.4.18.2 Consumer reads the field. Demo `geometricPrecision`.
- [ ] 88.4.18.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.19 `vector-effect`

- [ ] 88.4.19.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go; internal/svg`).
- [ ] 88.4.19.2 Consumer reads the field. Demo `non-scaling-stroke`.
- [ ] 88.4.19.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.20 `fill-color`

- [ ] 88.4.20.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go`).
- [ ] 88.4.20.2 Consumer reads the field. Demo `#c0392b`.
- [ ] 88.4.20.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.21 `stroke-color`

- [ ] 88.4.21.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_paint_props.go`).
- [ ] 88.4.21.2 Consumer reads the field. Demo `#2c3e50`.
- [ ] 88.4.21.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.4.R batch gate (package only)

- [ ] 88.4.R.1 Targeted `go test ./internal/<pkg> -run '…' -count=1` exit 0 (skip for 88.8 defer-only).
- [ ] 88.4.R.2 `python3 scripts/css-catalog-map.py --check` if apply arms added.
- [ ] 88.4.R.3 Mapping + matrix only for properties with flip packets. **Do not** run `make test` / `make lint` here.

## Out of scope

fill-stroke-3 drafts (fill-image, stroke-align, ...). Those stay out of this 100 except fill-color/stroke-color.
