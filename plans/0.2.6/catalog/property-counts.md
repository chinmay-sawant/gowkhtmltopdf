# property counts (generated from mapping.json)

Source: `mapping.json` `summary.properties_by_engine_status`, mirrored in `coverage-summary.json` `counts`.

Updated: 2026-09-12 (ponytail audit PT26-LAY-04/05 removed the parsed-and-stored no-op fields for 27 properties and demoted their rows to unsupported; compositing support from 2026-09-02 stays, with `isolation` and `mix-blend-mode` partial because full transparency groups are not in the PDF writer).

| Kind | Total | implemented | partial | unsupported | ignored |
|------|------:|------------:|--------:|------------:|--------:|
| Properties | 818 | 328 | 2 | 488 | 0 |
| At-rules | 55 | 0 | 11 | 34 | 10 |
| Selectors | 158 | 14 | 10 | 113 | 21 |
| Functions | 162 | 5 | 24 | 109 | 24 |
| Units | 30 | 7 | 8 | 15 | 0 |

Notes:
- `implemented` includes the §2.10 parsed-no-op and lite properties that still have a reader: `background-attachment` (`style.go:309`, `style_properties.go:1607`, `background_image.go:291`), `box-decoration-break` (`style.go:348`, `style_advanced_props.go:62`), `bookmark-label`, `bookmark-level`, `bookmark-state`, `footnote-display`, `footnote-policy`, `string-set`, plus lite impl `margin-trim` (§2.1, `style.go:347`) and `empty-cells` (§2.5). See `documentation/compatibility-matrix.md` §2.1, §2.4, §2.5, §2.10.
- `contain`, the `contain-intrinsic-*` longhands, and `content-visibility` were removed from that list on 2026-09-12: ponytail audit PT26-LAY-04 deleted the unread `ResolvedStyle` fields and apply arms, and `mapping.json` demoted their rows `implemented -> unsupported`.
- `background-blend-mode` is implemented for standard PDF/raster blend modes and applies per background layer. `isolation` and `mix-blend-mode` are partial: the flat display list has operation-level blending, while full element-group transparency semantics remain deferred.

Recount: `python3 scripts/css-catalog-map.py --check` (258 apply arms mapped, exit 0). `go build ./...` passes.
