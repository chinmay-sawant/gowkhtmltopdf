# 88 - Canonical next-100 CSS coverage (v0.2.7)

> **Parent:** `plans/0.2.7/README.md`
> **Status:** planned
> **Estimated effort:** XL (8 batches plus closure: aliases S, SVG bake M, clip/mask L, ruby/gap M-L, honesty defer S)
> **Owner:** `internal/layout` (+ `internal/svg` for presentation bake, `internal/pdf` for clip evenodd / mask groups)
> **Depends on:** next-72 close (87.1-87.8), catalog 397 Implemented / 10 Partial / 411 Unsupported
> **Unblocks:** mapping recount past the next-72 ceiling; fixture-65 Effect cells
> **Honesty:** `plans/0.2.6/HONESTY-GATES.md` flip packet required per Implemented name
> **Scan evidence:** four explore agents (aliases/cascade, SVG presentation, mask/clip/filter, ruby/overflow/gap), 2026-09-17

---

## Overview

Implement the 100 properties in `next-100-properties.json` in dependency-ordered
batches. This list starts **after** the 72 names in `next-72-properties.json`.
Overlap with that file is zero.

Next-72 used up the leftover original tier-5 print worklist (fonts, object-fit,
shapes, Borders-4 defer, and friends). What remains of that tier outside the 72
is a handful of css-gaps-1 insets and placeholders. Next-100 is the next print
layer:

- vendor aliases whose unprefixed bases already paint
- inline SVG presentation baked onto `writeSVGPresentationAttrs`
- clip-path / mask lite
- CSS `zoom`, overflow-4 longhands, row-rule / rule, ruby over/under
- a Borders-4-style honesty-defer batch (mask-border, MathML, filter primitives)

Validation bar is the **gowk PDF** Effect cell on
`testdata/golden/fixture-65-next-100-props.html` (authoring contract only in this
commit; the HTML lands with the first 88.x implementation batch). Chrome labels
in the JSON are advisory.

### Catalog at plan open (2026-09-17)

| Status | Count |
|--------|------:|
| Implemented | 397 |
| Partial | 10 |
| Unsupported | 411 |
| Total | 818 |

Next-100 slice: 0 Implemented / 0 Partial / 100 Unsupported of 100 (81
implement-aim, 19 honest-defer). Aspirational ceiling if every implement-aim
name flipped Implemented is 397 + 81 = 478, before any Partial leftovers.

## Effort scale (same as 0.2.6 phase-79 / next-72)

| Scale | Meaning |
|-------|---------|
| **S** | Alias, thin apply, or consumer already nearly present; package tests in one sitting |
| **M** | New fields + apply + one consumer path; focused new Go file |
| **L** | Cross-cutting layout/paint (clip paths, mask groups, ruby boxes, vector-effect) |
| **XL** | Pagination-coupled or a full SVG filter graph (mask-border, backdrop-filter) |

## Architecture rules

1. **Pipeline ownership:** cascade apply (`style_*_props.go`) -> layout/paint/svg bake consumer -> package test -> matrix -> mapping last.
2. **Deep modules:** small apply interface registered on `styleGroups` in `style_cascade.go`.
3. **Do not grow** `style_properties.go` or `layout.go`. Extract before adding cases.
4. **SVG path:** CSS on inline SVG -> `ResolvedStyle` -> `writeSVGPresentationAttrs` in `layout_svg.go` -> `internal/svg.Rasterize`. Do not claim Implemented from store-only leftover fields.
5. **Clip/mask:** `clip-path` is not overflow clip. `mask-image` is not PNG `/SMask` alone. Forbidden proofs live in `HONESTY-GATES.md`.
6. **Aliases:** flip `-webkit-*` only when the unprefixed base is Implemented (`normalizeVendorPrefix`).
7. **Gates:** mid-batch package tests only. **No `make lint` in this ledger.** Full `make test` / `make golden` only in batch 88.9.

## Executive summary (batches)

| Batch | Phase file | Props | Effort | Primary seams |
|-------|------------|------:|--------|---------------|
| 88.1 Quick aliases + `all` | [phase-88.1-quick-aliases-cascade.md](phases/phase-88.1-quick-aliases-cascade.md) | 5 | S-M | `normalizeVendorPrefix`, `all` expand |
| 88.2 Text stroke + inline leftovers | [phase-88.2-text-stroke-inline.md](phases/phase-88.2-text-stroke-inline.md) | 6 | M-L | `style_text_stroke_props.go` |
| 88.3 SVG paint bake | [phase-88.3-svg-paint-bake.md](phases/phase-88.3-svg-paint-bake.md) | 12 | S-M | `style_leftovers.go`, `layout_svg.go` |
| 88.4 SVG geometry / markers / text | [phase-88.4-svg-geometry-markers-text.md](phases/phase-88.4-svg-geometry-markers-text.md) | 20 | M-L | `style_svg_geom_props.go`, img scaler |
| 88.5 Clip and mask lite | [phase-88.5-clip-mask-lite.md](phases/phase-88.5-clip-mask-lite.md) | 13 | L | `clip_path.go`, `mask_image.go` |
| 88.6 WebKit mask aliases | [phase-88.6-webkit-mask-aliases.md](phases/phase-88.6-webkit-mask-aliases.md) | 8 | S | after 88.5 bases |
| 88.7 Zoom, overflow, gap, ruby | [phase-88.7-zoom-overflow-gap-ruby.md](phases/phase-88.7-zoom-overflow-gap-ruby.md) | 17 | M-L | `style_zoom_props.go`, `ruby.go`, gap rules |
| 88.8 Honesty defer | [phase-88.8-honesty-defer.md](phases/phase-88.8-honesty-defer.md) | 19 | S defer | catalog only |
| 88.9 Closure | [phase-88.9-closure-integration.md](phases/phase-88.9-closure-integration.md) | - | M | **`make test` + `make golden` only here** |

Property math: 5+6+12+20+13+8+17+19 = **100**.

## Recommended new Go files (do not grow allowlisted giants)

| File | Owns |
|------|------|
| `style_text_stroke_props.go` | `-webkit-text-stroke*` |
| `style_svg_geom_props.go` | `cx` `cy` `r` `rx` `ry` `x` `y` `d` |
| `style_svg_marker_props.go` | `marker` / `marker-*` |
| `style_mask_clip_props.go` | clip + mask apply |
| `clip_path.go` | paint clip (not overflow) |
| `mask_image.go` | single PNG/luminance mask |
| `style_zoom_props.go` | CSS `zoom` |
| `style_ruby_props.go` + `ruby.go` | ruby box, over/under |

Prefer extending `style_leftovers.go`, `layout_svg.go`, `style_advanced_props.go`,
and `style_gap_props.go` when the change is small.

## Why these 100 (and not animation)

Original triage leftover outside next-72:

| Bucket | Remaining unsupported | In this 100 |
|--------|----------------------:|------------:|
| `C_vendor_prefix_aliases` with Implemented bases | 3 background + line-clamp | yes |
| `B_svg_presentation` print-core | 30 + fill-color/stroke-color | yes |
| `B_mask_clip_filter_effects` lite | 13 | yes |
| `-webkit-mask*` after those bases | 8 | yes |
| Ruby / overflow-4 / zoom / gap core | 21 keep from the niche scan | 17 implement + 4 defer |
| `all`, leftover E_text/E_inline, text-stroke | 6 | yes |
| mask-border + filter primitives + path-length | 19 | honesty defer |
| Animation, speech, scroll, pointer, 3D | 155 print-noop | **no** |
| Gap inset-cap / corner-shape drafts | 34+27 | **no** |

## Out of scope

- JavaScript, animation, scroll UI, speech, 3D
- Next-72 names (including the 19 Borders-4 / shape-inside leftovers)
- Growing `style_properties.go` / `layout.go` without a compensating extract
- `make lint` inside any batch (owner runs after)
- Claiming Implemented from apply-only, leftover store-only, overflow clip, or fixture authorship

## Handoff

Start at [phase-88.1-quick-aliases-cascade.md](phases/phase-88.1-quick-aliases-cascade.md).
Close the program only via [phase-88.9-closure-integration.md](phases/phase-88.9-closure-integration.md).
