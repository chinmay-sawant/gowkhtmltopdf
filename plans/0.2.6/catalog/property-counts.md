# property counts (generated from mapping.json)

Source: `mapping.json` `summary.properties_by_engine_status`, mirrored in `coverage-summary.json` `counts`.

Updated: 2026-09-02 (compositing support added for standard blend modes; `isolation` and `mix-blend-mode` remain partial because full transparency groups are not in the PDF writer).

| Kind | Total | implemented | partial | unsupported | ignored |
|------|------:|------------:|--------:|------------:|--------:|
| Properties | 818 | 355 | 2 | 461 | 0 |
| At-rules | 55 | 0 | 11 | 34 | 10 |
| Selectors | 158 | 14 | 10 | 113 | 21 |
| Functions | 162 | 5 | 24 | 109 | 24 |
| Units | 30 | 7 | 8 | 15 | 0 |

Notes:
- `implemented` includes 19 properties in §2.10 that are **Implemented (parsed, no visual effect for print)**: `background-attachment`, `box-decoration-break`, `contain`, `contain-intrinsic-size`, `contain-intrinsic-width`, `contain-intrinsic-height`, `contain-intrinsic-inline-size`, `contain-intrinsic-block-size`, `content-visibility`, `bookmark-label`, `bookmark-level`, `bookmark-state`, `footnote-display`, `footnote-policy`, `string-set` plus lite impl `margin-trim` (§2.1), `empty-cells` (§2.5). See `documentation/compatibility-matrix.md` §2.1, §2.4, §2.5, §2.10 and `style.go:296`, `style.go:334-335`, `style.go:343-349`, `style_advanced_props.go`, `style_properties.go:1516`, `background_image.go:291`.
- `background-blend-mode` is implemented for standard PDF/raster blend modes and applies per background layer. `isolation` and `mix-blend-mode` are partial: the flat display list has operation-level blending, while full element-group transparency semantics remain deferred.

Recount: `python3 scripts/css-catalog-map.py --check` (262 apply arms mapped). `go build ./...` passes.
