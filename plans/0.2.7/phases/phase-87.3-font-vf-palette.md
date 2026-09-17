# Phase 87.3: Font optical-sizing, palette, variation-settings (honesty)

> **Parent:** `../87-canonical-0.2.7-next-72.md`
> **Status:** planned
> **Estimated effort:** S (honest Partial/docs) or L (real PDF instancing)
> **Owner:** `internal/layout` (+ `internal/pdf` if real instancing)
> **Depends on:** 87.2 optional
> **Unblocks:** mapping honesty for three demoted/re-added font rows
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

These three already have apply arms and fields (`style_font_variant_props.go`,
`style.go`), plus `resolveFontVariants` in `layout.go` that intentionally stays
on the **default instance** even when fvar/COLR exist. Catalog still marks them
unsupported. Choose an honest bar; do not flip Implemented without a consumer that
changes glyphs/metrics/paint.

## Properties (3)

`font-optical-sizing`, `font-palette`, `font-variation-settings`

## Seams

| Piece | Location |
|-------|----------|
| Apply | `style_font_variant_props.go:40-71` |
| Fields | `style.go:367-370` |
| Consumer no-op | `resolveFontVariants` `layout.go:735-767` |
| Capability probes | `Font.HasVariationAxes` / `HasColorPalette` `registry.go:318-351` |
| Tests | `TestApplyFontVariantProps`, `TestResolveFontVariants*` |

## Checklist

### 87.3.1 choose bar

- [ ] 87.3.1.1 Product choice recorded in this file:
  - **A (S):** Keep Unsupported or mark Partial with matrix note "parsed; PDF embeds default instance only".
  - **B (L):** Real `opsz`/`wght`/`wdth` instancing that updates embedded metrics + outlines, and/or COLR palette paint.

### 87.3.2 path A (honesty only)

- [ ] 87.3.2.1 Matrix § fonts rows document static no-op; mapping stays `unsupported` or moves to `partial` with `code_path` and notes. No fake Implemented.
- [ ] 87.3.2.2 Package test still asserts no-op behavior (`TestResolveFontVariantsStaticBundledFaces`).

### 87.3.3 path B (real instancing; only if chosen)

- [ ] 87.3.3.1 Variation axes applied with correct glyf/hmtx for embed.
- [ ] 87.3.3.2 Palette selection paints COLR/CPAL or documented fallback.
- [ ] 87.3.3.3 Tests prove glyph/metrics/paint change vs default. Flip Implemented only then.

### 87.3.R batch gate (package only)

- [ ] 87.3.R.1 `go test ./internal/layout -run 'TestApplyFontVariant|TestResolveFontVariants' -count=1` exit 0.
- [ ] 87.3.R.2 Mapping/matrix match the chosen bar. **No `make test` / `make lint`.**

## Out of scope

Feature-settings pipeline (87.2). New direct modules for font engines.
