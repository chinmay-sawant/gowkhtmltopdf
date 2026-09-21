# Phase 87.2: Font features, variants, synthesis, width, size-adjust

> **Parent:** `../87-canonical-0.2.7-next-72.md`
> **Status:** complete
> **Estimated effort:** M–L (18 properties)
> **Owner:** `internal/layout` + `internal/pdf` (shaper)
> **Depends on:** 87.1 preferred (not hard)
> **Unblocks:** 87.3
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.
> **Proof:** `_proof-87.2.md`

---

## Overview

Wire CSS font feature / variant / synthesis / width / size-adjust into the existing
go-text shaping path. Today `pdf.ParseFontFeatureSettings` and
`ShapeTextFontWithFeatures*` exist, but paint still passes **nil** features.

Do not put feature payloads on the hot `Op` struct; use `opExtra` (same pattern as
text language). Do not grow `style_properties.go`. Replace the no-op
`applyFontPropsWave4` stub with real groups.

## Properties (18)

`font-feature-settings`, `font-kerning`, `font-size-adjust`, `font-stretch`,
`font-synthesis`, `font-synthesis-position`, `font-synthesis-small-caps`,
`font-synthesis-style`, `font-synthesis-weight`, `font-variant`,
`font-variant-alternates`, `font-variant-caps`, `font-variant-east-asian`,
`font-variant-emoji`, `font-variant-ligatures`, `font-variant-numeric`,
`font-variant-position`, `font-width`

(Optical-sizing / palette / variation-settings are batch 87.3.)

## Architecture seams (from scan)

| Piece | Location |
|-------|----------|
| Wave-4 no-op stub | `style_font_props.go` (removed; groups on `styleGroups`) |
| Variant apply (other names) | `style_font_variant_props.go` (rejects `font-feature-settings`) |
| Feature parser / shaper | `internal/pdf/shape_gotext.go` |
| Paint features | `internal/pdf/content.go` `TextShowLanguageFeatures` |
| Fake bold gated | `paint.go` `FakeBoldFor`; `Op.NoFakeBold` |
| Registry lookup | weight/italic only (`registry.go`); no width argument |
| Inherit mask | coalesced CSS Fonts shaping cluster (60/64 entries) |

## New files

- `style_font_feature_props.go` (+ test): feature-settings, kerning, variant shorthand/longhands
- `style_font_synthesis_props.go` (+ test): synthesis shorthand + longhands → `NoFakeBold` / future oblique
- `style_font_width_props.go` (+ test): `font-width` + alias `font-stretch`
- `style_font_size_adjust_props.go` (+ test): size-adjust metrics

Register each on `styleGroups`. Regenerate style intern after new fields.

## Checklist

### 87.2.1 scope lock

- [x] 87.2.1.1 Lock the 18 names above. Note honest Partial for `font-variant-alternates` (needs `@font-feature-values`) and `font-variant-emoji` (color-font) if full OT path is out of reach.

### 87.2.2 feature pipeline

- [x] 87.2.2.1 Apply + store `font-feature-settings` / `font-kerning`; map kerning none to disable `kern` tag when shaper allows.
- [x] 87.2.2.2 Attach features via `opExtra` from `inline_paint.go` emit paths; pass into `TextShow*` / `ShapeTextFontWithFeatures*`.
- [x] 87.2.2.3 Tests: `TestApplyFontFeatureSettings`, `TestFontFeatureSettingsReachShaper`, `TestFontKerningNone`.
- [x] 87.2.2.4 Mirror PDF path in imageout text shaping if it shares the nil-features bug.

### 87.2.3 font-variant*

- [x] 87.2.3.1 Expand `font-variant` shorthand into longhands; map caps/ligatures/numeric/position/east-asian keywords to OT tags.
- [x] 87.2.3.2 `font-variant-alternates` / `font-variant-emoji`: implement lite or leave Unsupported/Partial with matrix note (do not fake Implemented).
- [x] 87.2.3.3 Tests: `TestFontVariantCapsMapsToSmcp`, `TestFontVariantShorthandExpands`.

### 87.2.4 synthesis

- [x] 87.2.4.1 `font-synthesis-weight: none` (and shorthand) sets `NoFakeBold` / gates `FakeBoldFor`.
- [x] 87.2.4.2 Style / small-caps / position synthesis: implement only with a real consumer; otherwise Partial/Unsupported.
- [x] 87.2.4.3 Test: `TestFontSynthesisWeightNoneDisablesFakeBold`.

### 87.2.5 width / stretch / size-adjust

- [x] 87.2.5.1 `font-width` + `font-stretch` alias; extend face lookup when width masters exist (Liberation may no-op; document).
- [x] 87.2.5.2 `font-size-adjust`: scale used size from x-height / chosen metric; need OS/2 sxHeight helper if missing.
- [x] 87.2.5.3 Tests: `TestFontStretchAliasesToWidth`, `TestFontSizeAdjustScalesUsedSize`.

### 87.2.R batch gate (package only)

- [x] 87.2.R.1 `go test ./internal/layout -run 'TestFontFeature|TestFontKerning|TestFontVariant|TestFontSynthesis|TestFontWidth|TestFontStretch|TestFontSizeAdjust' -count=1` exit 0.
- [x] 87.2.R.2 `go test ./internal/pdf -run 'TestParseFontFeature|TestShapeTextFontWithFeatures' -count=1` exit 0.
- [x] 87.2.R.3 Flip packets + matrix only for names with real consumers. **No `make test` / `make lint`.**

## Out of scope

`font-optical-sizing`, `font-palette`, `font-variation-settings` (87.3). COLR/CPAL paint. Variable-font glyf instancing.
