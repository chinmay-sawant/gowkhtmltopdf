# Implemented property code evidence (2026-09-02; demotion + re-implementation update 2026-09-12; line refresh 2026-09-16)

Cross-check of the **354** current `engine_status: implemented` rows in `mapping.json` against **non-test** Go under `internal/layout`. The 3 CSS Fonts rows demoted by PT26-LAY-05 on 2026-09-12 stay in the machine-readable list with `validation: DEMOTED`; the 23 rows re-implemented the same day carry `evidence_kind: consumer-read` pointing at the layout/paint line that reads the field, and `isolation` plus `mix-blend-mode` completed 2026-09-12 with element-group consumers in layout, PDF, and imageout. All lines were re-derived 2026-09-16 after the v0.2.7 code wave; see `## Refresh procedure`. All lines were re-derived 2026-09-16 after the v0.2.7 code wave; see `## Refresh procedure`. All lines were re-derived 2026-09-16 after the v0.2.7 code wave; see `## Refresh procedure`.

Tests (`*_test.go`) were excluded.

## Summary

| Validation | Count | Meaning |
|------------|------:|----------|
| VERIFIED | 354 | apply arm (`case`, `raw["prop"]`, vendor alias, const case, or `prop ==`) plus a real consumer for the 26 rows re-implemented or completed on 2026-09-12 (24 re-implemented plus `isolation` and `mix-blend-mode`) |
| DEMOTED | 3 | parsed-and-stored no-op; field and apply arm removed 2026-09-12 (PT26-LAY-05) |
| UNVERIFIED | 0 | no non-test layout evidence |
| **Total** | **357** | |

Machine-readable list (every property + file + line): [`implemented-code-evidence.json`](implemented-code-evidence.json).

## File rollup (primary evidence file)

| File | Properties |
|------|----------:|
| `internal/layout/style_properties.go` | 153 |
| `internal/layout/style_cascade.go` | 90 |
| `internal/layout/style_flex_props.go` | 18 |
| `internal/layout/style_advanced_props.go` | 26 |
| `internal/layout/style_text_support_props.go` | 8 |
| `internal/layout/style_containment_props.go` | 7 |
| `internal/layout/style_color_adjust_props.go` | 5 |
| `internal/layout/style_image_adjust_props.go` | 3 |
| `internal/layout/style_paint_props.go` | 37 |
| `internal/layout/style_leftovers.go` | 6 |
| `internal/layout/style_font_variant_props.go` | 1 |

DEMOTED entries are not counted in this rollup.

Line cells name their file when the strongest evidence sits outside the section's file.

## By primary file

### `internal/layout/style_properties.go` (153)

| Property | Evidence | Line |
|----------|----------|-----:|
| `accent-color` | case-string | 1363 |
| `background` | case-string | 1391 |
| `background-attachment` | case-string | 1424 |
| `background-clip` | case-string | 1420 |
| `background-color` | case-string | 1387 |
| `background-image` | case-string | 1393 |
| `background-origin` | case-string | 1422 |
| `background-position` | case-string | 1395 |
| `background-position-block` | case-string | 1406 |
| `background-position-inline` | case-string | 1404 |
| `background-position-x` | case-string | 1404 |
| `background-position-y` | case-string | 1406 |
| `background-repeat` | case-string | 1410 |
| `background-repeat-block` | case-string | 1418 |
| `background-repeat-inline` | case-string | 1416 |
| `background-repeat-x` | case-string | 1412 |
| `background-repeat-y` | case-string | 1414 |
| `background-size` | case-string | 1408 |
| `block-size` | case-const:propBlockSize | 743 |
| `border-bottom-color` | case-string | 1115 |
| `border-bottom-style` | case-string | 1115 |
| `border-bottom-width` | case-string | 1113 |
| `border-collapse` | case-string | 1710 |
| `border-image` | case-string | 1128 |
| `border-image-outset` | case-string | 1128 |
| `border-image-repeat` | case-string | 1128 |
| `border-image-slice` | case-string | 1128 |
| `border-image-source` | case-string | 1128 |
| `border-image-width` | case-string | 1128 |
| `border-left-color` | case-string | 1115 |
| `border-left-style` | case-string | 1115 |
| `border-left-width` | case-string | 1113 |
| `border-radius` | case-string | 1131 |
| `border-right-color` | case-string | 1115 |
| `border-right-style` | case-string | 1115 |
| `border-right-width` | case-string | 1113 |
| `border-spacing` | case-string | 1710 |
| `border-top-color` | case-string | 1115 |
| `border-top-style` | case-string | 1115 |
| `border-top-width` | case-string | 1113 |
| `bottom` | case-const:cssVerticalAlignBottom | 238 |
| `box-sizing` | case-string | 70 |
| `break-after` | case-string | 1712 |
| `break-before` | case-string | 1712 |
| `break-inside` | case-string | 1712 |
| `caption-side` | case-string | 1710 |
| `clear` | case-const:clearKeyword | 68 |
| `color` | case-string | 1355 |
| `column-count` | case-string | 321 |
| `column-fill` | case-string | 361 |
| `column-rule-color` | case-string | 295 |
| `column-rule-style` | case-string | 291 |
| `column-rule-width` | case-string | 287 |
| `column-span` | case-string | 356 |
| `column-width` | case-string | 323 |
| `columns` | case-string | 325 |
| `container-name` | case-string | 1719 |
| `container-type` | case-string | 1719 |
| `direction` | case-string | 82 |
| `display` | case-string | 62 |
| `filter` | case-string | 184 |
| `float` | case-string | 66 |
| `grid` | case-string | 419 |
| `grid-area` | case-string | 400 |
| `grid-auto-flow` | case-string | 411 |
| `grid-column` | case-string | 430 |
| `grid-column-end` | case-string | 434 |
| `grid-column-start` | case-string | 432 |
| `grid-row` | case-string | 436 |
| `grid-row-end` | case-string | 440 |
| `grid-row-start` | case-string | 438 |
| `grid-template` | case-string | 417 |
| `grid-template-areas` | case-string | 398 |
| `grid-template-columns` | case-string | 394 |
| `grid-template-rows` | case-string | 396 |
| `height` | case-string | 496 |
| `hyphenate-character` | case-string | 1480 |
| `hyphens` | case-string | 1480 |
| `inline-size` | case-const:containerInlineSize | 743 |
| `left` | case-const:floatLeft | 240 |
| `letter-spacing` | case-string | 1687 |
| `line-break` | case-string | 1480 |
| `line-height` | case-string | 1515 |
| `list-style` | case-string | 1666 |
| `list-style-position` | case-string | 1662 |
| `list-style-type` | case-string | 1658 |
| `margin-bottom` | case-string | 511 |
| `margin-break` | case-string | 1712 |
| `margin-left` | case-string | 513 |
| `margin-right` | case-string | 513 |
| `margin-top` | case-string | 511 |
| `max-block-size` | case-const:propMaxBlockSize | 743 |
| `max-height` | case-string | 500 |
| `max-inline-size` | case-const:propMaxInlineSize | 743 |
| `max-width` | case-string | 500 |
| `min-block-size` | case-const:propMinBlockSize | 743 |
| `min-height` | case-string | 498 |
| `min-inline-size` | case-const:propMinInlineSize | 743 |
| `min-width` | case-string | 498 |
| `opacity` | case-string | 182 |
| `orphans` | case-string | 1717 |
| `overflow` | case-string | 87 |
| `overflow-wrap` | case-string | 1602 |
| `overflow-x` | case-string | 87 |
| `overflow-y` | case-string | 87 |
| `padding-bottom` | case-string | 517 |
| `padding-left` | case-string | 517 |
| `padding-right` | case-string | 517 |
| `padding-top` | case-string | 517 |
| `page-break-after` | case-string | 1712 |
| `page-break-before` | case-string | 1712 |
| `page-break-inside` | case-string | 1712 |
| `position` | case-string | 64 |
| `right` | case-const:floatRight | 236 |
| `tab-size` | case-string | 1480 |
| `table-layout` | case-string | 1710 |
| `text-align` | case-string | 1522 |
| `text-align-all` | case-string | 1480 |
| `text-align-last` | case-string | 1480 |
| `text-decoration` | case-string | 1636 |
| `text-decoration-color` | case-string | 1480 |
| `text-decoration-line` | case-string | 1480 |
| `text-decoration-style` | case-string | 1480 |
| `text-decoration-thickness` | case-string | 1480 |
| `text-emphasis` | case-string | 1480 |
| `text-emphasis-color` | case-string | 1480 |
| `text-emphasis-position` | case-string | 1480 |
| `text-emphasis-skip` | case-string | 1480 |
| `text-emphasis-style` | case-string | 1480 |
| `text-indent` | case-string | 1695 |
| `text-justify` | case-string | 1480 |
| `text-shadow` | case-string | 1480 |
| `text-transform` | case-string | 1524 |
| `text-underline-offset` | case-string | 1480 |
| `text-underline-position` | case-string | 1480 |
| `text-wrap` | case-string | 1480 |
| `text-wrap-mode` | case-string | 1480 |
| `text-wrap-style` | case-string | 1480 |
| `top` | case-const:cssVerticalAlignTop | 234 |
| `transform` | case-string | 1887 |
| `transform-origin` | case-string | 1889 |
| `vertical-align` | case-string | 1526 |
| `visibility` | case-string | 186 |
| `white-space` | case-string | 1528 |
| `white-space-collapse` | case-string | 1480 |
| `white-space-trim` | case-string | 1480 |
| `widows` | case-string | 1717 |
| `width` | case-string | 496 |
| `word-break` | case-string | 1604 |
| `word-spacing` | case-string | 1689 |
| `word-wrap` | case-string | 1602 |
| `writing-mode` | case-string | 80 |
| `z-index` | case-string | 180 |

### `internal/layout/style_cascade.go` (90)

| Property | Evidence | Line |
|----------|----------|-----:|
| `-webkit-align-content` | case-string | 1421 |
| `-webkit-align-items` | case-string | 1423 |
| `-webkit-align-self` | case-string | 1425 |
| `-webkit-border-bottom-left-radius` | case-string | 1397 |
| `-webkit-border-bottom-right-radius` | case-string | 1399 |
| `-webkit-border-radius` | case-string | 1391 |
| `-webkit-border-top-left-radius` | case-string | 1393 |
| `-webkit-border-top-right-radius` | case-string | 1395 |
| `-webkit-box-align` | case-string | 1433 |
| `-webkit-box-flex` | case-string | 1435 |
| `-webkit-box-ordinal-group` | case-string | 1437 |
| `-webkit-box-orient` | case-string | 1439 |
| `-webkit-box-pack` | case-string | 1441 |
| `-webkit-box-shadow` | case-string | 1429 |
| `-webkit-box-sizing` | case-string | 1389 |
| `-webkit-filter` | case-string | 1431 |
| `-webkit-flex` | case-string | 1405 |
| `-webkit-flex-basis` | case-string | 1407 |
| `-webkit-flex-direction` | case-string | 1409 |
| `-webkit-flex-flow` | case-string | 1411 |
| `-webkit-flex-grow` | case-string | 1413 |
| `-webkit-flex-shrink` | case-string | 1415 |
| `-webkit-flex-wrap` | case-string | 1417 |
| `-webkit-justify-content` | case-string | 1419 |
| `-webkit-order` | case-string | 1427 |
| `-webkit-text-fill-color` | case-string | 1443 |
| `-webkit-transform` | case-string | 1401 |
| `-webkit-transform-origin` | case-string | 1403 |
| `border` | case-const:borderProperty | 727 |
| `border-block` | case-const:cssPropBorderBlock | 979 |
| `border-block-color` | case-const:cssPropBorderBlockColor | 991 |
| `border-block-end` | case-const:cssPropBorderBlockEnd | 985 |
| `border-block-end-color` | case-const:cssPropBorderBlockEndColor | 997 |
| `border-block-end-style` | case-const:cssPropBorderBlockEndStyle | 1009 |
| `border-block-end-width` | case-const:cssPropBorderBlockEndWidth | 1021 |
| `border-block-start` | case-const:cssPropBorderBlockStart | 983 |
| `border-block-start-color` | case-const:cssPropBorderBlockStartColor | 995 |
| `border-block-start-style` | case-const:cssPropBorderBlockStartStyle | 1007 |
| `border-block-start-width` | case-const:cssPropBorderBlockStartWidth | 1019 |
| `border-block-style` | case-const:cssPropBorderBlockStyle | 1003 |
| `border-block-width` | case-const:cssPropBorderBlockWidth | 1015 |
| `border-bottom` | case-const:borderBottomProperty | style_properties.go:1111 |
| `border-color` | case-const:borderColorKeyword | style_properties.go:1115 |
| `border-inline` | case-const:cssPropBorderInline | 981 |
| `border-inline-color` | case-const:cssPropBorderInlineColor | 993 |
| `border-inline-end` | case-const:cssPropBorderInlineEnd | 989 |
| `border-inline-end-color` | case-const:cssPropBorderInlineEndColor | 1001 |
| `border-inline-end-style` | case-const:cssPropBorderInlineEndStyle | 1013 |
| `border-inline-end-width` | case-const:cssPropBorderInlineEndWidth | 1025 |
| `border-inline-start` | case-const:cssPropBorderInlineStart | 987 |
| `border-inline-start-color` | case-const:cssPropBorderInlineStartColor | 999 |
| `border-inline-start-style` | case-const:cssPropBorderInlineStartStyle | 1011 |
| `border-inline-start-width` | case-const:cssPropBorderInlineStartWidth | 1023 |
| `border-inline-style` | case-const:cssPropBorderInlineStyle | 1005 |
| `border-inline-width` | case-const:cssPropBorderInlineWidth | 1017 |
| `border-left` | case-const:borderLeftProperty | style_properties.go:1111 |
| `border-right` | case-const:borderRightProperty | style_properties.go:1111 |
| `border-style` | case-const:borderStyleKeyword | style_properties.go:1115 |
| `border-top` | case-const:borderTopProperty | style_properties.go:1111 |
| `border-width` | case-const:borderWidthKeyword | style_properties.go:1113 |
| `column-rule` | case-string | style_properties.go:285 |
| `container` | case-const:containerKeyword | style_properties.go:1719 |
| `flex` | case-const:flexKeyword | style_flex_props.go:25 |
| `font` | raw | 1170 |
| `font-family` | raw | 1198 |
| `font-size` | raw | 1178 |
| `font-style` | raw | 1216 |
| `font-weight` | raw | 1209 |
| `gap` | case-const:gapKeyword | style_flex_props.go:19 |
| `inset` | case-const:insetKeyword | style_properties.go:1014 |
| `inset-block` | case-const:cssPropInsetBlock | 949 |
| `inset-block-end` | case-const:cssPropInsetBlockEnd | 965 |
| `inset-block-start` | case-const:cssPropInsetBlockStart | 963 |
| `inset-inline` | case-const:cssPropInsetInline | 956 |
| `inset-inline-end` | case-const:cssPropInsetInlineEnd | 969 |
| `inset-inline-start` | case-const:cssPropInsetInlineStart | 967 |
| `margin` | case-const:marginProperty | 723 |
| `margin-block` | case-const:cssPropMarginBlock | 898 |
| `margin-block-end` | case-string | 914 |
| `margin-block-start` | case-string | 912 |
| `margin-inline` | case-const:cssPropMarginInline | 905 |
| `margin-inline-end` | case-string | 918 |
| `margin-inline-start` | case-string | 916 |
| `padding` | case-const:paddingProperty | 725 |
| `padding-block` | case-const:cssPropPaddingBlock | 920 |
| `padding-block-end` | case-string | 936 |
| `padding-block-start` | case-string | 934 |
| `padding-inline` | case-const:cssPropPaddingInline | 927 |
| `padding-inline-end` | case-string | 940 |
| `padding-inline-start` | case-string | 938 |

### `internal/layout/style_flex_props.go` (18)

Extracted from `style_properties.go` on 2026-09-16 (v0.2.7 Phase 5.2, to keep that file under the size gate).

| Property | Evidence | Line |
|----------|----------|-----:|
| `align-content` | case-string | 21 |
| `align-items` | case-string | 21 |
| `align-self` | case-string | 21 |
| `column-gap` | case-string | 19 |
| `flex-basis` | case-string | 25 |
| `flex-direction` | case-string | 21 |
| `flex-flow` | case-string | 21 |
| `flex-grow` | case-string | 25 |
| `flex-shrink` | case-string | 25 |
| `flex-wrap` | case-string | 21 |
| `justify-content` | case-string | 21 |
| `justify-items` | case-string | 21 |
| `justify-self` | case-string | 21 |
| `order` | case-string | 25 |
| `place-content` | case-string | 21 |
| `place-items` | case-string | 21 |
| `place-self` | case-string | 21 |
| `row-gap` | case-string | 19 |

### `internal/layout/style_advanced_props.go` (26 verified, 3 demoted)

Consumer files for the two 2026-09-12 compositing rows: `layout_stacking.go` (`pushZ` group creation `:27`, `enterBlendIsolation` `:75`), `blend_group.go` (`BlendGroup.Isolate` `:32`), `paint_groups.go` (`target`/`enter`/`closeReady` sibling routing), `pdf/content.go` (transparency-group Form XObject buffered for `finalizeForms` `:367`), `pdf/pdf.go` (`finalizeForms` `:989`), `imageout/groups.go` (one group-buffer composite). Honest subset: page-split group fragments composite per page; group composite alpha is 1 (element opacity stays per descendant op, so once-at-group opacity is not implemented); group order follows the last member in the engine global paint order, not a full CSS stacking-context tree; `plus-lighter` stays unsupported; form `/BBox` is the page box; PDF/UA tagging inside forms was not veraPDF-validated.

| Property | Evidence | Line |
|----------|----------|-----:|
| `background-blend-mode` | case-string | 74 |
| `bookmark-label` | case-string | 30 |
| `bookmark-level` | case-string | 21 |
| `bookmark-state` | case-string | 33 |
| `box-decoration-break` | case-string | 62 |
| `empty-cells` | case-string | 55 |
| `font-optical-sizing` | case-string (parsed, no consumer; demoted 2026-09-12) | - |
| `font-palette` | case-string (parsed, no consumer; demoted 2026-09-12) | - |
| `font-variation-settings` | case-string (parsed, no consumer; demoted 2026-09-12) | - |
| `footnote-display` | case-string | 38 |
| `footnote-policy` | case-string | 43 |
| `isolation` | consumer-read | 89 |
| `line-clamp` | case-string | 26 |
| `margin-trim` | case-string | 49 |
| `max-lines` | case-string | 40 |
| `mix-blend-mode` | consumer-read | 69 |
| `overflow-clip-margin-block` | case-string | 111 |
| `overflow-clip-margin-block-end` | case-string | 111 |
| `overflow-clip-margin-block-start` | case-string | 111 |
| `overflow-clip-margin-bottom` | case-string | 105 |
| `overflow-clip-margin-inline` | case-string | 111 |
| `overflow-clip-margin-inline-end` | case-string | 111 |
| `overflow-clip-margin-inline-start` | case-string | 111 |
| `overflow-clip-margin-left` | case-string | 108 |
| `overflow-clip-margin-right` | case-string | 102 |
| `overflow-clip-margin-top` | case-string | 99 |
| `string-set` | case-string | 48 |
| `text-decoration-skip-ink` | case-string | 94 |
| `text-overflow` | case-string | 21 |

### `internal/layout/style_text_support_props.go` (8)

Consumer files: `inline_paint.go` (run rotation and decoration geometry) and `inline_collect.go` (bidi scopes).

| Property | Evidence | Line |
|----------|----------|-----:|
| `text-combine-upright` | consumer-read | inline_paint.go:382 |
| `text-decoration-inset` | consumer-read | inline_paint.go:849 |
| `text-decoration-skip` | consumer-read | inline_paint.go:559 |
| `text-decoration-skip-box` | consumer-read | inline_paint.go:559 |
| `text-decoration-skip-self` | consumer-read | inline_paint.go:552 |
| `text-decoration-skip-spaces` | consumer-read | inline_paint.go:563 |
| `text-orientation` | consumer-read | inline_paint.go:376 |
| `unicode-bidi` | consumer-read | inline_collect.go:26 |

### `internal/layout/style_containment_props.go` (7)

Consumer files: `layout_flow.go` (size containment, `content-visibility:hidden`, intrinsic placeholder widths) plus the field readers in this file (`containsSize`, `containmentIntrinsicWidth`, `containmentIntrinsicHeight`).

| Property | Evidence | Line |
|----------|----------|-----:|
| `contain` | consumer-read | 234 |
| `contain-intrinsic-block-size` | consumer-read | 261 |
| `contain-intrinsic-height` | consumer-read | 270 |
| `contain-intrinsic-inline-size` | consumer-read | 264 |
| `contain-intrinsic-size` | consumer-read | 256 |
| `contain-intrinsic-width` | consumer-read | 256 |
| `content-visibility` | consumer-read | layout_flow.go:254 |

### `internal/layout/style_color_adjust_props.go` (5)

Consumer files: `background_image.go` (color-adjust paint gate) and `paint.go` (scheme, forced colors, sRGB clamp).

| Property | Evidence | Line |
|----------|----------|-----:|
| `color-adjust` | consumer-read | background_image.go:32 |
| `color-scheme` | consumer-read | paint.go:111 |
| `dynamic-range-limit` | consumer-read | paint.go:116 |
| `forced-color-adjust` | consumer-read | paint.go:115 |
| `print-color-adjust` | consumer-read | background_image.go:32 |

### `internal/layout/style_image_adjust_props.go` (3)

Consumer file: `layout_images.go` (EXIF orientation, resolution scale, view-box crop).

| Property | Evidence | Line |
|----------|----------|-----:|
| `image-orientation` | consumer-read | layout_images.go:224 |
| `image-resolution` | consumer-read | layout_images.go:256 |
| `object-view-box` | consumer-read | layout_images.go:326 |

### `internal/layout/style_paint_props.go` (37)

| Property | Evidence | Line |
|----------|----------|-----:|
| `border-block-end-radius` | case-string | 255 |
| `border-block-start-radius` | case-string | 255 |
| `border-bottom-left-radius` | case-string | 241 |
| `border-bottom-radius` | case-string | 246 |
| `border-bottom-right-radius` | case-string | 239 |
| `border-end-end-radius` | case-string | 255 |
| `border-end-start-radius` | case-string | 255 |
| `border-inline-end-radius` | case-string | 255 |
| `border-inline-start-radius` | case-string | 255 |
| `border-left-radius` | case-string | 249 |
| `border-right-radius` | case-string | 252 |
| `border-start-end-radius` | case-string | 255 |
| `border-start-start-radius` | case-string | 255 |
| `border-top-left-radius` | case-string | 235 |
| `border-top-radius` | case-string | 243 |
| `border-top-right-radius` | case-string | 237 |
| `box-shadow` | case-const:boxShadowProp | 124 |
| `box-shadow-blur` | case-string | 126 |
| `box-shadow-color` | case-string | 130 |
| `box-shadow-offset` | case-string | 132 |
| `box-shadow-position` | case-string | 134 |
| `box-shadow-spread` | case-string | 128 |
| `content` | case-const:propContent | 459 |
| `counter-increment` | case-string | 453 |
| `counter-reset` | case-string | 449 |
| `fill` | case-string | 54 |
| `fill-opacity` | case-string | 63 |
| `list-style-image` | case-string | 457 |
| `outline` | prop-eq | 42 |
| `outline-color` | case-string | 106 |
| `outline-offset` | case-string | 111 |
| `outline-style` | case-string | 102 |
| `outline-width` | case-string | 98 |
| `quotes` | case-string | 443 |
| `stroke` | case-string | 67 |
| `stroke-opacity` | case-string | 77 |
| `stroke-width` | case-string | 72 |

### `internal/layout/style_leftovers.go` (6)

| Property | Evidence | Line |
|----------|----------|-----:|
| `overflow-clip-margin` | case-string | style_advanced_props.go:111 |
| `page` | case-string | style_properties.go:1715 |
| `rotate` | case-string | 27 |
| `scale` | case-string | 29 |
| `transform-box` | case-string | style_properties.go:1905 |
| `translate` | case-string | 31 |

### `internal/layout/style_font_variant_props.go` (1)

| Property | Evidence | Line |
|----------|----------|-----:|
| `font-language-override` | case-string | 48 |

## Refresh procedure (2026-09-16)

Every `file:line` in this list was re-derived against the post-v0.2.7 tree
after the code wave moved apply arms (flex/gap to `style_flex_props.go`, text
props out of `style_properties.go`, shorthands between cascade and paint).

1. Scan every non-test `internal/layout/*.go` for case labels inside
   `switch prop { ... }` blocks (quoted names and package consts that resolve
   to a quoted property name), for `raw["name"]` lookups in
   `style_cascade.go`, and for `prop == "name"` checks. Drop sites in
   `internalCustomPropWriters` (a membership loop, not an apply arm).
2. For case-kind rows prefer, in order: a site in the stored primary file, a
   site in the `mapping.json` `code_path` file, then the first site in sorted
   `(file, line)` order; within a file take the smallest line and prefer
   `apply*`/`set*`/`expand*`/`assign*`/`boxShorthand*`/`normalizeVendorPrefix`/
   `remapWebkitValue` functions.
3. For `consumer-read` rows derive the `ResolvedStyle` field from the
   definition file's apply arm and point at the first non-comment line that
   reads it, preferring the stored primary file, then the row's `files` order.
   The `contain*` rows point at the helper readers in
   `style_containment_props.go` (`:234`, `:256`, `:261`, `:264`, `:270`).
4. `bookmark-label`, `bookmark-level`, `bookmark-state`, `footnote-display`,
   `footnote-policy`, and `string-set` kept their 2026-09-12 lines: no
   `switch prop` apply arm exists for them in the current tree. The first two
   are consumed in `internal/outline/outline.go:145` and `:151`, outside this
   list's layout scope; the other four have no non-test consumer at all.
5. The `inset`/`top`/`right`/`bottom`/`left` rows were refreshed on the
   2026-09-16 follow-up pass after Phase 9.4 landed (`relative_percent.go`,
   `applyRelativeOffset` in `flex.go`); their apply-arm primary stays per step 2.
