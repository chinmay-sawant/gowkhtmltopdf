# Phase 70: SVG presentation (fill/stroke)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 70
> **Status:** complete (5 SVG presentation properties implemented; 26 remain unsupported)
> **Estimated effort:** L
> **Owner:** `internal/layout`
> **Depends on:** Phase 69
> **Unblocks:** Phase 71
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

CSS `fill`, `stroke`, `stroke-width`, `fill-opacity`, `stroke-opacity` presentation properties are supported on elements and stored in `ResolvedStyle`.

## Checklist

- [x] 70.1.1 Ownership list locked.
- [x] 70.2.1 Style fields + apply arms for claimed subset (`fill`, `fill-opacity`, `stroke`, `stroke-width`, `stroke-opacity`). Proof: `internal/layout/style.go:237`, `style_paint_props.go:45`.
- [x] 70.2.2 Style resolution and paint fields wired. Proof: `internal/layout/style_paint_props.go:applySVGPresentationProps`.
- [x] 70.2.3 CSS-on-SVG HTML tests. Proof: `TestSVGPresentationProps` in `style_cascade_test.go`.
- [x] 70.2.4 Mapping flip packet per Implemented name; others stay unsupported. Proof: `mapping.json` (5 implemented, 26 unsupported).
- [x] 70.3.1 `go test ./internal/layout ./internal/svg`; `--check`; gates. Proof: all exit 0.

## Forbidden proofs

- SVG file attribute parsing without CSS apply
- Catalog-only Implemented for `fill-*` / `stroke-*`
