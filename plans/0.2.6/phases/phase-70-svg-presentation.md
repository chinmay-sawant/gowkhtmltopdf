# Phase 70: SVG presentation (fill/stroke)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 70
> **Status:** not started (31 names unsupported)
> **Estimated effort:** L
> **Owner:** `internal/layout` + `internal/svg` as needed
> **Depends on:** Phase 69
> **Unblocks:** Phase 71
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

CSS `fill` / `stroke` (and related) on SVG elements in HTML must affect raster/paint. `<img src=*.svg>` XML attribute raster via `internal/svg` is **not** this phase.

## Owned names (31)

`fill`, `fill-break`, `fill-color`, `fill-image`, `fill-opacity`, `fill-origin`, `fill-position`, `fill-repeat`, `fill-rule`, `fill-size`, `stroke`, `stroke-align`, `stroke-alignment`, `stroke-break`, `stroke-color`, `stroke-dash-corner`, `stroke-dash-justify`, `stroke-dashadjust`, `stroke-dasharray`, `stroke-dashcorner`, `stroke-dashoffset`, `stroke-image`, `stroke-linecap`, `stroke-linejoin`, `stroke-miterlimit`, `stroke-opacity`, `stroke-origin`, `stroke-position`, `stroke-repeat`, `stroke-size`, `stroke-width`

## Work order (code)

1. Add `ResolvedStyle` fields (or an SVG presentation sub-struct) in `internal/layout/style.go`.
2. Apply arms in a new group or `style_properties.go` for `fill`, `stroke`, `stroke-width`, … (start with the high-frequency subset, then expand).
3. Consumer: when laying out/painting inline SVG or SVG DOM nodes, pass computed fill/stroke into `internal/svg` raster or a dedicated paint path. File touch likely `internal/svg/raster.go` plus layout SVG embedding call sites.
4. Tests must set **CSS** `fill:` / `stroke:` on an SVG element in HTML and assert paint/ops or pixel presence. `TestRasterizeRect` alone is forbidden.

## Checklist

- [x] 70.1.1 Ownership list locked.
- [ ] 70.2.1 Style fields + apply arms for the claimed subset.
- [ ] 70.2.2 Paint/raster consumer reads those fields.
- [ ] 70.2.3 CSS-on-SVG HTML tests.
- [ ] 70.2.4 Mapping flip packet per Implemented name; others stay unsupported.
- [ ] 70.3.1 `go test ./internal/layout ./internal/svg`; `--check`; gates.

## Forbidden proofs

- SVG file attribute parsing without CSS apply
- Catalog-only Implemented for `fill-*` / `stroke-*`
