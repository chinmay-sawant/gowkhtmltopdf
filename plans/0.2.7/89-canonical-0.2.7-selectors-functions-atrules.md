# 89 - Canonical selectors, functions, and at-rules coverage (v0.2.7)

> **Parent:** `plans/0.2.7/README.md`
> **Status:** planned
> **Estimated effort:** XL (8 batches: recount S-M, selectors M-L, math M, color/image L, conditional at-rules L, paged/font at-rules L-XL, closure M)
> **Owner:** `internal/css` (parse and match) with consumers in `internal/layout` (`style_cascade.go`, `gradient.go`, `shape_exclusion.go`, `page_named.go`)
> **Depends on:** 0.2.6 catalog honesty, 0.2.7 gate policy in `plans/0.2.7/README.md`
> **Unblocks:** a grammar catalog with the same honesty bar as the 818-property catalog; go-gpui shares this parser, so each selector and at-rule reaches the live window too
> **Honesty:** `plans/0.2.6/HONESTY-GATES.md` packets; 89.1 defines the grammar variant
> **Scan evidence:** code read 2026-10-03 on `chore/changes-for-go-gpui` at `2111b36`; file:line list below

---

## Overview

The property catalog has an owner, a checker, and fixtures. Selectors, functions, and at-rules have none of that. `plans/0.2.6/catalog/mapping.json` carries counts for all three, but the last audit left them stale, and `scripts/css-catalog-map.py` maps properties only. Its help text says so: "Map frozen CSS catalogs onto engine apply arms."

The stale counts undersell the engine. `@supports`, `@layer`, and `@property` are parsed and applied (`internal/css/css.go:259`), six gradient forms paint (`internal/layout/gradient.go:43`), `oklab()`, `oklch()`, `color-mix()`, and `light-dark()` convert (`internal/css/color_modern.go:21`), and attribute selectors match every operator with the `i` flag (`internal/css/match.go:299`, `internal/css/attr_iflag_test.go`). The catalog calls most of that unsupported.

This ledger has two jobs. First, put the three grammar categories on the same honesty footing as properties: recount against code, add a mechanical check so the counts cannot rot again, and write flip packets. Second, implement the gaps in dependency order so selectors, functions, and at-rules stop being the thinnest part of the compatibility matrix.

### Catalog at plan open (2026-10-03, `mapping.json`)

| Category | Implemented | Partial | Unsupported | Ignored | Total |
|----------|------------:|--------:|------------:|--------:|------:|
| Selectors | 14 | 10 | 113 | 21 | 158 |
| Functions | 5 | 24 | 109 | 24 | 162 |
| At-rules | 0 | 11 | 34 | 10 | 55 |

The counts are the reason this plan exists, and 89.1 replaces them.

### Code evidence the catalog does not carry

| Area | Evidence | Catalog says |
|------|----------|--------------|
| Attribute selectors | `internal/css/match.go:299` `matchAttrs`, ops at `:338` (`=` `~=` `*=`, `^=`, `$=`, `|=`), `i` flag at `:330`; `internal/css/attr_iflag_test.go` | Not tracked as a selector row at all |
| Functional pseudos | `internal/css/selector_parser.go:344` `:has()`, `:not()`, `:is()`, `:where()`; `internal/css/has.go`; match at `internal/css/match.go:400` | `:has()` partial, others implemented |
| Structural and state pseudos | `internal/css/match.go:395` `:root`, `:first-child`, `:last-child`, `:nth-child`, `:link`, `:visited`, `:focus`, `:focus-visible`, `:hover`, `:active`, `:checked`; `internal/css/match_state_test.go` | 14 implemented, 113 unsupported |
| Pseudo-elements | `internal/css/selector_parser.go:284` accepts only `::before` and `::after`; `:431` rejects `::first-line` and `::first-letter` | `::before` and `::after` partial, others unsupported |
| Gradients | `internal/layout/gradient.go:43` linear, radial, conic, and all three repeating forms | `linear-gradient()` and friends unsupported |
| Modern color | `internal/css/color_modern.go:21` oklab, oklch, color-mix, light-dark | Not tracked as implemented |
| Math | `internal/css/math_expr.go:164` calc, min, max, clamp only | calc/min/max/clamp partial |
| `@supports` | `internal/css/at_supports.go:5` decl, and, or, not tree; consumer `internal/layout/style_cascade.go:535` | Unsupported |
| `@layer` | `internal/css/at_layer.go:5` statements, blocks, cascade rank | Unsupported |
| `@property` | `internal/css/at_property.go:19` syntax, initial-value, inherits | Unsupported |
| `@media` | `internal/css/media.go:24` types, size features, orientation; consumer `internal/layout/style_cascade.go:527` | Partial |
| `@container` | `internal/css/container.go:26` size features with and/or/not; consumer `internal/layout/style_cascade.go:587` | Partial |
| `@page` | `internal/css/page_margin.go:15` quoted margin box content only; counter() and running() drop | Partial |

### Catalog policy for grammar rows

89.1 locks these rules before any status moves:

1. **Implemented** needs a parse path, a consumer that changes layout or paint, and a package test. A parsed-and-stored field with no consumer stays Partial or Unsupported, same as the property rule in `HONESTY-GATES.md`.
2. **Partial** names the supported subset in the row notes, like the property rows do.
3. **Ignored** is permanent only for print no-ops (animation, transition, scroll-driven). A live host difference, such as `:hover`, is Partial for the engine and Implemented for go-gpui host state, and the row says which context.
4. Every flip lands with a grammar packet in `plans/0.2.6/catalog/` and a checker entry, then the count moves.

## Effort scale (same as 87 and 88)

| Scale | Meaning |
|-------|---------|
| **S** | Parser or matcher case plus tests in one sitting |
| **M** | New grammar node plus one consumer path and a focused new file |
| **L** | Cross-cutting: line fragments, cascade origins, pagination, or a backend-specific paint path |
| **XL** | Pagination-coupled page chrome or a full value grammar |

## Architecture rules

1. **Grammar ownership:** parse in `internal/css`, gate in `internal/layout/style_cascade.go`, paint in the layout consumer, test in the package that owns the consumer, catalog last.
2. **Do not grow** `internal/css/css.go` (1031), `selector_parser.go` (637), `match.go` (780), or `container.go` (736) without extracting a cohesive piece first. New cases go in focused files such as `selector_pseudo.go`, `math_funcs.go`, `at_counter_style.go`.
3. **One checker.** Extend `scripts/css-catalog-map.py` or add `scripts/css-grammar-catalog.py`; no manual count edits. The recount is not done until `--check` fails on drift.
4. **Print bar.** Validation is the gowk PDF and image outputs, not Chrome. A feature that only affects live interaction is Partial with the reason recorded.
5. **Gates:** mid-batch targeted `go test ./internal/css ./internal/layout -run '...'`. `make test` and `make golden` only in batch 89.8. No `make lint` in this ledger.

## Executive summary (batches)

| Batch | Phase file (authored at batch start) | Scope | Effort | Primary seams |
|-------|--------------------------------------|-------|--------|---------------|
| 89.1 Recount | `phases/phase-89.1-grammar-recount.md` | Selector, function, and at-rule statuses against code; checker; flip packets | S-M | `scripts/css-catalog-map.py`, `plans/0.2.6/catalog/` |
| 89.2 Selectors A | `phases/phase-89.2-selectors-structural-pseudo-elements.md` | `:empty`, `:only-child`, `:only-of-type`, `:nth-last-child()`, `:lang()`, `::first-line`, `::first-letter`, `::marker` | L | `selector_parser.go`, `match.go`, inline paint, list markers |
| 89.3 Selectors B | `phases/phase-89.3-selectors-functional-attribute-state.md` | `:has()` completion, nth `of S`, forgiving `:is()`, state pseudos, attribute edge cases | M-L | `has.go`, `match.go`, `style_cascade.go` specificity |
| 89.4 Math | `phases/phase-89.4-math-functions.md` | Full calc, min, max, clamp; round, mod, rem, abs, sign; trig and exponential honest decision | M-L | `math_expr.go`, `values.go`, consumers |
| 89.5 Color and image | `phases/phase-89.5-color-image-functions.md` | lab, lch, hwb, color(), relative color, gradients completion, image-set, cross-fade, counter, attr, polygon | L | `color_modern.go`, `gradient.go`, `values.go`, `shape_exclusion.go` |
| 89.6 Conditional at-rules | `phases/phase-89.6-conditional-at-rules.md` | `@media` ranges, `@supports` selector()/font-tech(), `@layer` import and revert-layer, `@property` validation, `@container` style queries | L | `media.go`, `at_supports.go`, `at_layer.go`, `at_property.go`, `container.go` |
| 89.7 Paged and font at-rules | `phases/phase-89.7-paged-font-at-rules.md` | `@page` corners and counters, `@import` layer/supports, `@counter-style`, `@font-face` descriptors, palette/feature values decision | L-XL | `page_margin.go`, `import.go`, list markers, font registry |
| 89.8 Closure | `phases/phase-89.8-closure-integration.md` | Fixture-67, matrix section, recount, full gates | M | **`make test` + `make golden` only here** |

Phase files are authored before their batch starts, the way 87.x and 88.x were
authored with their waves. This parent is the canonical ledger either way.

## Baseline by category

Slices are the completion target, not the recount result. 89.1 fills in exact rows.

- **Selectors:** add `:empty`, `:only-child`, `:only-of-type`, `:nth-last-child()`, `:lang()`, `:defined`, `:disabled`, `:enabled`, `:indeterminate`, `::marker`, `::first-line`, `::first-letter`; complete `:has()` and `:nth-child(An+B of S)`; keep `::selection`, `::backdrop`, `:scope`, `:target` as recorded print or interaction deferrals.
- **Functions:** complete calc, min, max, clamp; add round, mod, rem, abs, sign; decide trig and exponential families; add lab, lch, hwb, color(), relative color, contrast-color; complete gradients with hints, double-position stops, and interpolation spaces; add image-set, cross-fade, and polygon; complete counter, counters, attr; record env, element, paint, and device-cmyk as deferrals.
- **At-rules:** complete `@media`, `@supports`, `@layer`, `@property`, `@container`; add margin box corners, page counters, and running() to `@page`; add layer() and supports() to `@import`; add `@counter-style`; extend `@font-face` descriptors; record `@scope`, `@color-profile`, `@font-feature-values`, `@font-palette-values`, and `@charset` with a decision.

## Out of scope

- JavaScript, animation, transition, scroll-driven, and 3D transform families. They stay Ignored with the reason in the row.
- Property semantics work owned by 87 and 88. This ledger touches properties only when a selector, function, or at-rule gates them.
- Grid, flex, and pagination semantics changes beyond what a grammar item needs.
- go-gpui-only host behavior. The engine records host state as Partial; the window integration is a go-gpui plan.

## Risks and limits

- The recount in 89.1 can flip dozens of rows with no new code. That is the point, and it is also the danger: a flip without a written evidence path is exactly the over-claim `HONESTY-GATES.md` exists to stop. The checker and the packet are the guard.
- `::first-line` and `::first-letter` touch line building and pagination. A wrong split paints a styled fragment on the wrong page, and only a golden fixture catches it.
- `@property` computed values sit on the cascade hot path. A registration that forces re-resolution per declaration can cost more than the feature is worth on long documents.
- `@layer` plus `revert-layer` changes cascade origin handling, and `!important` reverses layer order per spec. Existing fixtures may depend on the current order, so land this behind the existing golden corpus.
- `@page` margin boxes share geometry with the CLI header and footer path. Read `internal/convert/hf.go` before changing box placement.
- The checker is the only thing that keeps this ledger honest after 89.8. If 89.1 ships the statuses without the check, the counts rot again and this file becomes the next stale artifact.
