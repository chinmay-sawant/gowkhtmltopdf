# property counts (generated from mapping.json)

Source: `mapping.json` `summary.properties_by_engine_status`, mirrored in `coverage-summary.json` `counts`.

Updated: 2026-09-12. PT26-LAY-04/05 demoted 27 parsed-and-stored no-op rows; 24 of them were re-implemented the same day with apply arms plus real consumers and flipped back to implemented (containment 7, print color adjust 5, image adjust 3, text support 8, `font-language-override` 1). The three remaining font rows (`font-optical-sizing`, `font-palette`, `font-variation-settings`) stay unsupported pending variable-font instancing and COLR/CPAL painting. `mix-blend-mode` and `isolation` moved from partial to implemented the same day with element-level transparency groups: the PDF writer emits one Form XObject per group with `/Group /S /Transparency /I true /CS /DeviceRGB` (plus `/BM` ExtGState for blend modes), and the PNG path composites each group buffer once from a transparent scratch backdrop.

| Kind | Total | implemented | partial | unsupported | ignored |
|------|------:|------------:|--------:|------------:|--------:|
| Properties | 818 | 354 | 0 | 464 | 0 |
| At-rules | 55 | 0 | 11 | 34 | 10 |
| Selectors | 158 | 14 | 10 | 113 | 21 |
| Functions | 162 | 5 | 24 | 109 | 24 |
| Units | 30 | 7 | 8 | 15 | 0 |

Notes:
- `implemented` includes the §2.10 parsed-no-op and lite properties that still have a reader: `background-attachment` (`style.go:309`, `style_properties.go:1607`, `background_image.go:291`), `box-decoration-break` (`style.go:348`, `style_advanced_props.go:62`), `bookmark-label`, `bookmark-level`, `bookmark-state`, `footnote-display`, `footnote-policy`, `string-set`, plus lite impl `margin-trim` (§2.1, `style.go:347`) and `empty-cells` (§2.5). See `documentation/compatibility-matrix.md` §2.1, §2.4, §2.5, §2.10.
- `contain`, the `contain-intrinsic-*` longhands, and `content-visibility` were demoted by PT26-LAY-04 on 2026-09-12 and re-implemented the same day with consumers in `layout_flow.go`, `layout_measure.go`, and `overflow_clip.go`; see `documentation/compatibility-matrix.md` §2.2. The full 24-row flip record is in `implemented-code-evidence.md`.
- `background-blend-mode` is implemented for standard PDF/raster blend modes and applies per background layer. `mix-blend-mode` and `isolation` are implemented with element-level transparency groups. Honest subset: page-split group fragments composite per page; group composite alpha is 1 (element opacity stays per descendant op); group order follows the last member in the engine's global paint order; `plus-lighter` remains unsupported; form `/BBox` is the page box (conservative, never clips).

Recount: `python3 scripts/css-catalog-map.py --check` (258 apply arms mapped, exit 0). `go build ./...` passes.
