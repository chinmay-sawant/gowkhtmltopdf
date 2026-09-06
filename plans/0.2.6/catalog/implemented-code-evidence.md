# Implemented property code evidence (2026-09-02)

Cross-check of all **355** `engine_status: implemented` rows in `mapping.json` against **non-test** Go under `internal/layout`.

Tests (`*_test.go`) were excluded.

## Summary

| Validation | Count | Meaning |
|------------|------:|----------|
| VERIFIED | 355 | apply arm: `case`, `raw["prop"]`, vendor alias, const case, or `prop ==` |
| UNVERIFIED | 0 | no non-test layout evidence |
| **Total** | **355** | |

Machine-readable list (every property + file + line): [`implemented-code-evidence.json`](implemented-code-evidence.json).

## File rollup (primary evidence file)

| File | Properties |
|------|----------:|
| `internal/layout/style_properties.go` | 166 |
| `internal/layout/style_cascade.go` | 90 |
| `internal/layout/style_advanced_props.go` | 56 |
| `internal/layout/style_paint_props.go` | 37 |
| `internal/layout/style_leftovers.go` | 6 |

## By primary file

### `internal/layout/style_properties.go` (166)

| Property | Evidence | Line |
|----------|----------|-----:|
| `accent-color` | case-string | 1406 |
| `align-content` | case-string | 228 |
| `align-items` | case-string | 228 |
| `align-self` | case-string | 228 |
| `background` | case-string | 1434 |
| `background-attachment` | case-string | 1463 |
| `background-clip` | case-string | 1459 |
| `background-color` | case-string | 1430 |
| `background-image` | case-string | 1436 |
| `background-origin` | case-string | 1461 |
| `background-position` | case-string | 1438 |
| `background-position-block` | case-string | 1449 |
| `background-position-inline` | case-string | 1447 |
| `background-position-x` | case-string | 1447 |
| `background-position-y` | case-string | 1449 |
| `background-repeat` | case-string | 1453 |
| `background-repeat-block` | case-string | 1457 |
| `background-repeat-inline` | case-string | 1455 |
| `background-repeat-x` | case-string | 1455 |
| `background-repeat-y` | case-string | 1457 |
| `background-size` | case-string | 1451 |
| `block-size` | case-const:propBlockSize | 864 |
| `border-bottom-color` | case-string | 1236 |
| `border-bottom-style` | case-string | 1236 |
| `border-bottom-width` | case-string | 1234 |
| `border-collapse` | case-string | 1723 |
| `border-image` | case-string | 1249 |
| `border-image-outset` | case-string | 1249 |
| `border-image-repeat` | case-string | 1249 |
| `border-image-slice` | case-string | 1249 |
| `border-image-source` | case-string | 1249 |
| `border-image-width` | case-string | 1249 |
| `border-left-color` | case-string | 1236 |
| `border-left-style` | case-string | 1236 |
| `border-left-width` | case-string | 1234 |
| `border-radius` | case-string | 1252 |
| `border-right-color` | case-string | 1236 |
| `border-right-style` | case-string | 1236 |
| `border-right-width` | case-string | 1234 |
| `border-spacing` | case-string | 1723 |
| `border-top-color` | case-string | 1236 |
| `border-top-style` | case-string | 1236 |
| `border-top-width` | case-string | 1234 |
| `bottom` | case-const:cssVerticalAlignBottom | 210 |
| `box-sizing` | case-string | 56 |
| `break-after` | case-string | 1725 |
| `break-before` | case-string | 1725 |
| `break-inside` | case-string | 1725 |
| `caption-side` | case-string | 1723 |
| `clear` | case-const:clearKeyword | 54 |
| `color` | case-string | 1398 |
| `column-count` | case-string | 501 |
| `column-fill` | case-string | 541 |
| `column-gap` | case-string | 226 |
| `column-rule-color` | case-string | 475 |
| `column-rule-style` | case-string | 471 |
| `column-rule-width` | case-string | 467 |
| `column-span` | case-string | 536 |
| `column-width` | case-string | 503 |
| `columns` | case-string | 505 |
| `container-name` | case-string | 1732 |
| `container-type` | case-string | 1732 |
| `direction` | case-string | 60 |
| `display` | case-string | 48 |
| `filter` | case-string | 156 |
| `flex-basis` | case-string | 232 |
| `flex-direction` | case-string | 228 |
| `flex-flow` | case-string | 228 |
| `flex-grow` | case-string | 232 |
| `flex-shrink` | case-string | 232 |
| `flex-wrap` | case-string | 228 |
| `float` | case-string | 52 |
| `grid` | case-string | 578 |
| `grid-area` | case-string | 572 |
| `grid-auto-flow` | case-string | 574 |
| `grid-column` | case-string | 589 |
| `grid-column-end` | case-string | 593 |
| `grid-column-start` | case-string | 591 |
| `grid-row` | case-string | 595 |
| `grid-row-end` | case-string | 599 |
| `grid-row-start` | case-string | 597 |
| `grid-template` | case-string | 576 |
| `grid-template-areas` | case-string | 570 |
| `grid-template-columns` | case-string | 566 |
| `grid-template-rows` | case-string | 568 |
| `height` | case-string | 639 |
| `hyphenate-character` | case-string | 1519 |
| `hyphens` | case-string | 1519 |
| `inline-size` | case-const:containerInlineSize | 864 |
| `justify-content` | case-string | 228 |
| `justify-items` | case-string | 228 |
| `justify-self` | case-string | 228 |
| `left` | case-const:floatLeft | 212 |
| `letter-spacing` | case-string | 1700 |
| `line-break` | case-string | 1519 |
| `line-height` | case-string | 1552 |
| `list-style` | case-string | 1679 |
| `list-style-position` | case-string | 1675 |
| `list-style-type` | case-string | 1671 |
| `margin-bottom` | case-string | 654 |
| `margin-break` | case-string | 1725 |
| `margin-left` | case-string | 656 |
| `margin-right` | case-string | 656 |
| `margin-top` | case-string | 654 |
| `max-block-size` | case-const:propMaxBlockSize | 864 |
| `max-height` | case-string | 643 |
| `max-inline-size` | case-const:propMaxInlineSize | 864 |
| `max-width` | case-string | 643 |
| `min-block-size` | case-const:propMinBlockSize | 864 |
| `min-height` | case-string | 641 |
| `min-inline-size` | case-const:propMinInlineSize | 864 |
| `min-width` | case-string | 641 |
| `opacity` | case-string | 154 |
| `order` | case-string | 232 |
| `orphans` | case-string | 1730 |
| `overflow` | case-string | 65 |
| `overflow-wrap` | case-string | 1615 |
| `overflow-x` | case-string | 65 |
| `overflow-y` | case-string | 65 |
| `padding-bottom` | case-string | 660 |
| `padding-left` | case-string | 660 |
| `padding-right` | case-string | 660 |
| `padding-top` | case-string | 660 |
| `page-break-after` | case-string | 1725 |
| `page-break-before` | case-string | 1725 |
| `page-break-inside` | case-string | 1725 |
| `place-content` | case-string | 228 |
| `place-items` | case-string | 228 |
| `place-self` | case-string | 228 |
| `position` | case-string | 50 |
| `right` | case-const:floatRight | 208 |
| `row-gap` | case-string | 226 |
| `tab-size` | case-string | 1519 |
| `table-layout` | case-string | 1723 |
| `text-align` | case-string | 1559 |
| `text-align-all` | case-string | 1519 |
| `text-align-last` | case-string | 1519 |
| `text-decoration` | case-string | 1649 |
| `text-decoration-color` | case-string | 1519 |
| `text-decoration-line` | case-string | 1519 |
| `text-decoration-style` | case-string | 1519 |
| `text-decoration-thickness` | case-string | 1519 |
| `text-indent` | case-string | 1708 |
| `text-justify` | case-string | 1519 |
| `text-shadow` | case-string | 1519 |
| `text-transform` | case-string | 1561 |
| `text-underline-offset` | case-string | 1519 |
| `text-underline-position` | case-string | 1519 |
| `text-wrap` | case-string | 1519 |
| `text-wrap-mode` | case-string | 1519 |
| `text-wrap-style` | case-string | 1519 |
| `top` | case-const:cssVerticalAlignTop | 206 |
| `transform` | case-string | 1923 |
| `transform-origin` | case-string | 1929 |
| `vertical-align` | case-string | 1563 |
| `visibility` | case-string | 158 |
| `white-space` | case-string | 1565 |
| `white-space-collapse` | case-string | 1519 |
| `white-space-trim` | case-string | 1519 |
| `widows` | case-string | 1730 |
| `width` | case-string | 639 |
| `word-break` | case-string | 1617 |
| `word-spacing` | case-string | 1702 |
| `word-wrap` | case-string | 1615 |
| `writing-mode` | case-string | 58 |
| `z-index` | case-string | 152 |

### `internal/layout/style_cascade.go` (90)

| Property | Evidence | Line |
|----------|----------|-----:|
| `-webkit-align-content` | case-string | 1112 |
| `-webkit-align-items` | case-string | 1114 |
| `-webkit-align-self` | case-string | 1116 |
| `-webkit-border-bottom-left-radius` | case-string | 1088 |
| `-webkit-border-bottom-right-radius` | case-string | 1090 |
| `-webkit-border-radius` | case-string | 1082 |
| `-webkit-border-top-left-radius` | case-string | 1084 |
| `-webkit-border-top-right-radius` | case-string | 1086 |
| `-webkit-box-align` | case-string | 1124 |
| `-webkit-box-flex` | case-string | 1126 |
| `-webkit-box-ordinal-group` | case-string | 1128 |
| `-webkit-box-orient` | case-string | 1130 |
| `-webkit-box-pack` | case-string | 1132 |
| `-webkit-box-shadow` | case-string | 1120 |
| `-webkit-box-sizing` | case-string | 1080 |
| `-webkit-filter` | case-string | 1122 |
| `-webkit-flex` | case-string | 1096 |
| `-webkit-flex-basis` | case-string | 1098 |
| `-webkit-flex-direction` | case-string | 1100 |
| `-webkit-flex-flow` | case-string | 1102 |
| `-webkit-flex-grow` | case-string | 1104 |
| `-webkit-flex-shrink` | case-string | 1106 |
| `-webkit-flex-wrap` | case-string | 1108 |
| `-webkit-justify-content` | case-string | 1110 |
| `-webkit-order` | case-string | 1118 |
| `-webkit-text-fill-color` | case-string | 1134 |
| `-webkit-transform` | case-string | 1092 |
| `-webkit-transform-origin` | case-string | 1094 |
| `border` | case-const:borderProperty | 789 |
| `border-block` | case-const:cssPropBorderBlock | 730 |
| `border-block-color` | case-const:cssPropBorderBlockColor | 742 |
| `border-block-end` | case-const:cssPropBorderBlockEnd | 736 |
| `border-block-end-color` | case-const:cssPropBorderBlockEndColor | 748 |
| `border-block-end-style` | case-const:cssPropBorderBlockEndStyle | 760 |
| `border-block-end-width` | case-const:cssPropBorderBlockEndWidth | 772 |
| `border-block-start` | case-const:cssPropBorderBlockStart | 734 |
| `border-block-start-color` | case-const:cssPropBorderBlockStartColor | 746 |
| `border-block-start-style` | case-const:cssPropBorderBlockStartStyle | 758 |
| `border-block-start-width` | case-const:cssPropBorderBlockStartWidth | 770 |
| `border-block-style` | case-const:cssPropBorderBlockStyle | 754 |
| `border-block-width` | case-const:cssPropBorderBlockWidth | 766 |
| `border-bottom` | case-const:borderBottomProperty | 1033 |
| `border-color` | case-const:borderColorKeyword | 1033 |
| `border-inline` | case-const:cssPropBorderInline | 732 |
| `border-inline-color` | case-const:cssPropBorderInlineColor | 744 |
| `border-inline-end` | case-const:cssPropBorderInlineEnd | 740 |
| `border-inline-end-color` | case-const:cssPropBorderInlineEndColor | 752 |
| `border-inline-end-style` | case-const:cssPropBorderInlineEndStyle | 764 |
| `border-inline-end-width` | case-const:cssPropBorderInlineEndWidth | 776 |
| `border-inline-start` | case-const:cssPropBorderInlineStart | 738 |
| `border-inline-start-color` | case-const:cssPropBorderInlineStartColor | 750 |
| `border-inline-start-style` | case-const:cssPropBorderInlineStartStyle | 762 |
| `border-inline-start-width` | case-const:cssPropBorderInlineStartWidth | 774 |
| `border-inline-style` | case-const:cssPropBorderInlineStyle | 756 |
| `border-inline-width` | case-const:cssPropBorderInlineWidth | 768 |
| `border-left` | case-const:borderLeftProperty | 1033 |
| `border-right` | case-const:borderRightProperty | 1033 |
| `border-style` | case-const:borderStyleKeyword | 1033 |
| `border-top` | case-const:borderTopProperty | 1033 |
| `border-width` | case-const:borderWidthKeyword | 1033 |
| `column-rule` | case-string | 1033 |
| `container` | case-const:containerKeyword | 1033 |
| `flex` | case-const:flexKeyword | 1033 |
| `font` | raw | 921 |
| `font-family` | raw | 949 |
| `font-size` | raw | 929 |
| `font-style` | raw | 967 |
| `font-weight` | raw | 960 |
| `gap` | case-const:gapKeyword | 1033 |
| `inset` | case-const:insetKeyword | 1033 |
| `inset-block` | case-const:cssPropInsetBlock | 700 |
| `inset-block-end` | case-const:cssPropInsetBlockEnd | 716 |
| `inset-block-start` | case-const:cssPropInsetBlockStart | 714 |
| `inset-inline` | case-const:cssPropInsetInline | 707 |
| `inset-inline-end` | case-const:cssPropInsetInlineEnd | 720 |
| `inset-inline-start` | case-const:cssPropInsetInlineStart | 718 |
| `margin` | case-const:marginProperty | 787 |
| `margin-block` | case-const:cssPropMarginBlock | 649 |
| `margin-block-end` | case-string | 665 |
| `margin-block-start` | case-string | 663 |
| `margin-inline` | case-const:cssPropMarginInline | 656 |
| `margin-inline-end` | case-string | 669 |
| `margin-inline-start` | case-string | 667 |
| `padding` | case-const:paddingProperty | 787 |
| `padding-block` | case-const:cssPropPaddingBlock | 671 |
| `padding-block-end` | case-string | 687 |
| `padding-block-start` | case-string | 685 |
| `padding-inline` | case-const:cssPropPaddingInline | 678 |
| `padding-inline-end` | case-string | 691 |
| `padding-inline-start` | case-string | 689 |

### `internal/layout/style_advanced_props.go` (57)

| Property | Evidence | Line |
|----------|----------|-----:|
| `background-blend-mode` | case-string | 123 |
| `bookmark-label` | case-string | 30 |
| `bookmark-level` | case-string | 21 |
| `bookmark-state` | case-string | 33 |
| `box-decoration-break` | case-string | 84 |
| `color-adjust` | case-string | 102 |
| `color-scheme` | case-string | 112 |
| `contain` | case-string | 135 |
| `contain-intrinsic-block-size` | case-string | 132 |
| `contain-intrinsic-height` | case-string | 126 |
| `contain-intrinsic-inline-size` | case-string | 129 |
| `contain-intrinsic-size` | case-string | 120 |
| `contain-intrinsic-width` | case-string | 123 |
| `content-visibility` | case-string | 138 |
| `dynamic-range-limit` | case-string | 115 |
| `empty-cells` | case-string | 50 |
| `font-language-override` | case-string | 153 |
| `font-optical-sizing` | case-string | 148 |
| `font-palette` | case-string | 156 |
| `font-variation-settings` | case-string | 145 |
| `footnote-display` | case-string | 38 |
| `footnote-policy` | case-string | 43 |
| `forced-color-adjust` | case-string | 107 |
| `image-orientation` | case-string | 89 |
| `image-resolution` | case-string | 94 |
| `line-clamp` | case-string | 58 |
| `margin-trim` | case-string | 76 |
| `max-lines` | case-string | 67 |
| `object-view-box` | case-string | 99 |
| `overflow-clip-margin-block` | case-string | 234 |
| `overflow-clip-margin-block-end` | case-string | 234 |
| `overflow-clip-margin-block-start` | case-string | 234 |
| `overflow-clip-margin-bottom` | case-string | 225 |
| `overflow-clip-margin-inline` | case-string | 231 |
| `overflow-clip-margin-inline-end` | case-string | 231 |
| `overflow-clip-margin-inline-start` | case-string | 231 |
| `overflow-clip-margin-left` | case-string | 228 |
| `overflow-clip-margin-right` | case-string | 222 |
| `overflow-clip-margin-top` | case-string | 219 |
| `print-color-adjust` | case-string | 102 |
| `string-set` | case-string | 48 |
| `text-combine-upright` | case-string | 170 |
| `text-decoration-inset` | case-string | 216 |
| `text-decoration-skip` | case-string | 199 |
| `text-decoration-skip-box` | case-string | 207 |
| `text-decoration-skip-ink` | case-string | 202 |
| `text-decoration-skip-self` | case-string | 210 |
| `text-decoration-skip-spaces` | case-string | 213 |
| `text-emphasis` | case-string | 181 |
| `text-emphasis-color` | case-string | 184 |
| `text-emphasis-position` | case-string | 190 |
| `text-emphasis-skip` | case-string | 196 |
| `text-emphasis-style` | case-string | 193 |
| `text-orientation` | case-string | 173 |
| `text-overflow` | case-string | 53 |
| `unicode-bidi` | case-string | 178 |

### `internal/layout/style_paint_props.go` (37)

| Property | Evidence | Line |
|----------|----------|-----:|
| `border-block-end-radius` | case-string | 277 |
| `border-block-start-radius` | case-string | 277 |
| `border-bottom-left-radius` | case-string | 263 |
| `border-bottom-radius` | case-string | 268 |
| `border-bottom-right-radius` | case-string | 261 |
| `border-end-end-radius` | case-string | 277 |
| `border-end-start-radius` | case-string | 277 |
| `border-inline-end-radius` | case-string | 277 |
| `border-inline-start-radius` | case-string | 277 |
| `border-left-radius` | case-string | 271 |
| `border-right-radius` | case-string | 274 |
| `border-start-end-radius` | case-string | 277 |
| `border-start-start-radius` | case-string | 277 |
| `border-top-left-radius` | case-string | 257 |
| `border-top-radius` | case-string | 265 |
| `border-top-right-radius` | case-string | 259 |
| `box-shadow` | case-const:boxShadowProp | 121 |
| `box-shadow-blur` | case-string | 139 |
| `box-shadow-color` | case-string | 123 |
| `box-shadow-offset` | case-string | 128 |
| `box-shadow-position` | case-string | 149 |
| `box-shadow-spread` | case-string | 144 |
| `content` | case-const:propContent | 407 |
| `counter-increment` | case-string | 401 |
| `counter-reset` | case-string | 397 |
| `fill` | case-string | 54 |
| `fill-opacity` | case-string | 59 |
| `list-style-image` | case-string | 405 |
| `outline` | prop-eq | 42 |
| `outline-color` | case-string | 102 |
| `outline-offset` | case-string | 107 |
| `outline-style` | case-string | 98 |
| `outline-width` | case-string | 94 |
| `quotes` | case-string | 391 |
| `stroke` | case-string | 63 |
| `stroke-opacity` | case-string | 73 |
| `stroke-width` | case-string | 68 |

### `internal/layout/style_leftovers.go` (6)

| Property | Evidence | Line |
|----------|----------|-----:|
| `overflow-clip-margin` | case-string | 15 |
| `page` | case-string | 88 |
| `rotate` | case-string | 81 |
| `scale` | case-string | 83 |
| `transform-box` | case-string | 69 |
| `translate` | case-string | 85 |
