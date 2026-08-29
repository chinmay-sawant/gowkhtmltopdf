# gowkhtmltopdf - HTML/CSS Compatibility Matrix (MVP Allowlist)

> **Parent:** `plans/0.1.0/00-canonical-pure-go-rewrite.md` (Phase 0.1); post-MVP updates under `plans/0.2.0/10-canonical-post-mvp-roadmap.md`  
> **Status:** living contract - amendments go through plan review  
> **Target:** authored HTML templates → PDF. **Not** a browser.  
> **Last honesty audit:** 2026-08-27 · fidelity guide: [fidelity.md](fidelity.md)  
> **Phase 21 note:** arbitrary-website / "decent print" work does **not** expand this matrix. CSS remains a **print CSS subset** (Partial flex/grid/position; many properties Not implemented). No new Implemented rows until code + tests ship - see [fidelity.md § Arbitrary websites](fidelity.md#arbitrary-websites-phase-21).

This document is the **contract** the layout engine is allowed to implement.
Anything not listed here is *unsupported*; unsupported input must degrade
gracefully (ignored declaration / skipped node / documented error), never
crash. Product framing: [fidelity.md](fidelity.md). **Still not full CSS.**

---

## 1. Supported HTML tags

MVP renders these elements. Everything else is stripped, ignored, or rendered
as its inline text (per the note column).

| Tag | Behavior note |
|-----|---------------|
| `html`, `head`, `body` | Document shell |
| `title`, `meta`, `style`, `link` | Metadata; `link rel=stylesheet` fetched (Phase 2) |
| `div`, `span` | Generic block / inline boxes |
| `p` | Block, default margins |
| `br` | Forced line break |
| `hr` | Block-level horizontal rule |
| `h1`–`h6` | Heading levels; outline source (Phase 6) |
| `ul`, `ol`, `li` | Lists. UA stylesheet: `ul`/`menu` → `disc`, `ol` → `decimal`. `markerText` implements `disc` / `circle` / `square` / `decimal` / `decimal-leading-zero` / `lower-alpha` / `upper-alpha` / `lower-roman` / `upper-roman` |
| `table`, `thead`, `tbody`, `tfoot`, `tr`, `th`, `td`, `caption` | Table subset; see §4/§2.5 (`colspan` and `rowspan` Implemented; `<caption>` / `table-caption` rendered above the table) |
| `img` | Replaced element; **PNG/JPEG/SVG subset**. JPEG is DCTDecode pass-through. PNG decoded to DeviceRGB; alpha soft-mask when present. **SVG** rasterized via `internal/svg` (rect/circle/path subset → PNG). Layout uses a fixed 96 dpi CSS px→pt map. `web.images=false` skips fetch/paint. |
| `a` | Hyperlink (`href`) for `http/https/mailto` external URI annotations; body `#id` / `#name` **GoTo** via `applyInternalLinks` (fixture-24). HTML header/footer **external URI** and **fragment GoTo** (`#id` → body destinations via `AddLinkDest`, copies-aware) are carried onto body pages |
| `strong`, `em`, `b`, `i`, `u`, `small` | `b`/`strong` → bold face; `em`/`i` → italic face (Liberation family, §2.3); `u` underline; `small` smaller; fake stroke bold only if a bold face is missing |
| `pre`, `code` | `pre` honors `white-space: pre`; `code` follows the author’s `font-family` (generic `monospace` → bundled Liberation Mono; see [fonts.md](fonts.md)) |
| `blockquote` | Block-level only - no indent margins (UA rule `style.go:714-717`) |
| `header`, `footer`, `main`, `section`, `article`, `aside`, `nav` | Treated as `div` (semantic aliases) |

## 2. Supported CSS properties

Status legend (verified against `applyRestProps` in
`internal/layout/style_cascade.go:666` plus dispatch in
`style_properties.go`, `uaRules` in `style_values.go`, `internal/css/css.go`,
`internal/layout/layout.go`, `inline.go`, `paint.go`, and the tests in
`internal/layout/layout_test.go` / `internal/convert/golden_test.go`):

- **Implemented** - parsed and consumed by layout.
- **Partial** - parsed and used in a subset of the declared cases; the rest
  degrades silently.
- **Not implemented** - parsed and dropped, or not parsed at all; the
  declaration is ignored (graceful, per the contract).

### 2.1 Box model

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `margin` / `margin-top|right|bottom|left` | Implemented | `setFour` / `marginLen` (`style_values.go:246`, `style_values.go:564`) via `applyRestProps` (`style_cascade.go:666`) and `style_properties.go`; sibling margin collapsing `layout.go:263`; tests `TestBlockWidthsAndMargins`, `TestMarginCollapse` |
| `margin-block` / `margin-inline` (and start/end longhands) | Implemented | Logical margin shorthands and longhands mapping to physical top/bottom/left/right for horizontal-tb. Tests `TestLogicalMargin` |
| `padding` / `padding-top|right|bottom|left` | Implemented | `style.go:350-359`; test `TestPaddingBorderBox` |
| `padding-block` / `padding-inline` (and start/end longhands) | Implemented | Logical padding shorthands and longhands mapping to physical padding for horizontal-tb. Tests `TestLogicalPadding` |
| `inset` / `inset-block` / `inset-inline` (and start/end longhands) | Implemented | Logical positioned offset shorthands and longhands mapping to top/right/bottom/left for horizontal-tb. Tests `TestLogicalInset` |
| `inline-size` / `block-size` / `min-inline-size` / `min-block-size` / `max-inline-size` / `max-block-size` | Implemented | Logical sizing properties mapping to width/height/min-width/min-height/max-width/max-height for horizontal-tb. Tests `TestLogicalSize` |
| `border` / `border-top|right|bottom|left` | Implemented | `style.go:360-379` (`parseBorder` `style.go:521`); styles `solid|dashed|dotted|none`, width + color; test `TestPaddingBorderBox` |
| `border-width`, `border-style`, `border-color` | Implemented | `style.go:380-401`; `thin|medium|thick` widths `style.go:546` |
| `border-radius` | Implemented | Shorthand including `rx / ry` slash (`setBorderRadius` `border_radius.go`). Longhands `border-top-left-radius` and siblings, including `10pt / 5pt` and `10pt 5pt`. Paint uses elliptical Bezier arcs when rx != ry (`roundedRectPathCorners` `paint.go`). Percent corners resolve per CSS against width and height axes. Tests `TestRadiusLonghand`, `TestRadiusSlash`, `TestRadiusEllipticalLonghand`, `TestRadiusPercentAxes`. |
| `width`, `height` | Implemented | `style.go:316-323`; consumed in `layout.go:176-191` (block) and `layout.go:315-320` (images) |
| `min-width`, `min-height`, `max-width`, `max-height` | Implemented | `style.go:324-339`; enforced `layout.go:181-186, 321-328`; `%` resolves against viewport approximation |
| `box-sizing` (`content-box|border-box`) | Implemented | parsed `style.go`; default `content-box` (specified width is content width); `border-box` makes width include padding+border (`layout.go` `buildBlock`); test `TestBoxSizingBorderBox` |
| `overflow` / `overflow-x` / `overflow-y` (`visible|hidden|auto|scroll|clip`) | Implemented | Sticky scrollport selection (`sticky.go`) plus paint clip of descendant fill/text/line/image to the padding box for `hidden|clip|auto|scroll` (`overflow_clip.go`). `visible` does not clip. Tests `TestOverflowClip`, `TestStickyOverflow*` |

### 2.2 Display & flow

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `display` (`block|inline|none|list-item|table|table-row|table-cell|table-row-group|table-header-group|table-footer-group|flex|inline-flex|grid|inline-grid`) | Implemented | Core display values Implemented; `flex`/`inline-flex`/`grid`/`inline-grid` accepted in `style.go` and routed to Stage A flex / Stage B grid lite (§2.7 / §2.8). `none` test `TestDisplayNone`; tables `TestTableLayout`; flex/grid fixtures 25/28/32 |
| `display: inline-block` | Implemented (lite) | atomic inline box with width/height/margins; shrink-to-fit when width auto; test `TestInlineBlockBesideText` |
| `display: table-caption` | Implemented | `<caption>` and `display: table-caption` render **above** the table (`buildTableCaption`) |
| `display: table-column`, `table-column-group` | Not implemented | parsed; no column model in `buildTable` |
| `float` (`left|right`) | Implemented (lite) | out-of-flow pack to side; stacks on same side; simple exclusion for following in-flow content; float inside `td` packs in cell BFC; in-flow `table` always clears below floats (no shrink-beside); `float` on `table-cell`/`table-row` blockifies (CSS2.1 §9.7); tests `TestFloatLeftRightClear`, `TestFloatInsideTableCell`, `TestTableClearsFloat`, fixture-22 / 29 / 38 |
| `clear` (`left|right|both`) | Implemented (lite) | advances past named float bottoms (`float.go`); test `TestFloatLeftRightClear` |
| `position` (`static|relative|absolute|fixed|sticky`) | Implemented | static in-flow; `relative`/`absolute`/`fixed` lite via `buildAbsolute` / `buildFixed` / `applyRelativeOffset` (fixtures 26/28). `sticky` = print-scoped clamp (page content box = scrollport; `sticky.go`, fixture-31, `TestSticky*`) plus overflow-box scrollport at offset 0 |
| `position: sticky` | Implemented | Default scrollport = page content box (`contentH`); clamps `top`/`bottom`/`left`/`right` within the containing block; natural fragment only, with no fixed-style continuation-page clones. Inside `overflow:auto|scroll|hidden|clip`, that box is the scrollport at **scroll offset 0** (PDF has no scroll; no page clones). Path: `sticky.go` / `applyStickyPrint`; fixture-31; `TestSticky*` / `TestStickyOverflow*` |
| `z-index` | Implemented (lite) | Integer or `auto` (`setZIndexValue` `style_properties.go:114`). Copied onto ops (`pushZ` `layout.go:715`) and sorted (`sortPaintIndices` `paint_order.go:32`). Not a full CSS stacking-context tree. Test `TestZIndexPaintOrder` |

### 2.3 Text & fonts

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `font` (shorthand) | Implemented | Shorthand expands to font-size, line-height, font-family, font-weight, font-style (`parseFontShorthand` `style_values.go`). Test `TestFontShorthand` |
| `font-family` (named + generic) | Implemented | parsed + inherited; embedded Liberation Sans family (R/B/I/BI) plus **font registry** (`--font-path`, optional `--use-system-fonts`) and `@font-face` TTF/OTF/WOFF1 (local and `https://` via `FetchSub`) on **PDF and image** paths (see §4 / §5). Named families resolve as named; missing faces fall through the author stack, then Liberation; only CSS generics (`serif`/`sans-serif`/`monospace`) expand to Liberation |
| `writing-mode` (`horizontal-tb|vertical-rl|vertical-lr`) | Implemented | Parsed + inherited (`TestWritingModeInherits`). `vertical-rl` / `vertical-lr` format vertical text lines with RotateDeg == -90 and vertical writing height calculation (`inline_paint.go`, `layout.go`), plus physical mapping of logical box properties (`style_properties.go`). |
| `font-size` | Implemented | `style.go` `fontSize` (px/pt/em/%/rem/in/cm/mm/pc + keywords); `%`/`em` resolve against parent; test `TestFontSizeEmInherit` |
| `font-weight` (`normal|bold|100-900`) | Implemented | ≥700 selects Liberation Sans **Bold** (or BoldItalic); fake stroke bold only if a bold face is missing; tests `TestRealBoldFaceOps`, `TestBoldFaceInInvoicePDF` |
| `font-style` (`italic|oblique`) | Implemented | selects Liberation Sans Italic / BoldItalic (`pdf.FaceSet.Resolve`); test `TestRealBoldFaceOps` |
| `text-align` (`left|right|center|justify`) | Implemented (justify lite) | left/right/center; `justify` distributes leftover space between word items on non-final lines (`inline.go`); test `TestTextAlignJustify` |
| `text-decoration` (`none|underline|line-through`) | Implemented | drawn in `inline.go`; test `TestBoldUnderline` |
| `text-indent` | Implemented | Inherited and applied to the first line (`inline.go`); test `TestTextIndentInheritsAndShiftsFirstLine` |
| `line-height` (number, length, `normal`) | Implemented | consumed in line metrics; test `TestMarginCollapse` |
| `letter-spacing` | Implemented | consumed in run width |
| `word-spacing` | Implemented | Inherited; extra width per ASCII space (`style_properties.go` apply + `inline_paint.go`). Tests `TestWordSpacingInherits`, `TestWordSpacingWidensRuns` |
| `text-transform` | Implemented | `none` | `uppercase` | `lowercase` | `capitalize` (`setTextTransformValue`; applied at measure and paint) |
| `vertical-align` (`baseline|top|middle|bottom`) | Implemented | table cells: top/middle/bottom offset within row (`emitCell`); inline replaced: top/middle/bottom vs baseline; test `TestTableCellVerticalAlignMiddle` |
| `white-space` (`normal|nowrap|pre|pre-wrap|pre-line`) | Implemented | `pre-wrap` preserves spaces and wraps; `pre-line` collapses spaces, keeps newlines, wraps (`setWhiteSpaceValue`, `collectPreservingNewlines`). Tests `TestWhiteSpacePre`, `TestWhiteSpacePreWrap` |
| `visibility` (`visible|hidden|collapse`) | Implemented | `hidden`/`collapse` skip paint, keep layout size (`hidesPaint`). Descendants inherit. Supports table columns, column groups, and rows (`layout_tables.go`). Test `TestVisibilityHidden`, `phase79_test.go` |
| `overflow-wrap` / `word-wrap` / `word-break` | Implemented | Parsed `applyTextWrapProps` (`style_properties.go`). Used by `wordBreakOf` (`layout_measure.go`). `word-wrap` is the overflow-wrap alias. `anywhere` / `break-all` mid-break; `break-word` soft wrap; `keep-all` preserves non-breaking runs. Tests `overflow_wrap_test.go`, `phase79_test.go` |
| `list-style` / `list-style-type` / `list-style-image` / `list-style-position` | Implemented | `inside` puts the marker in the first line; `outside` (default) hangs in the gutter; `list-style-image` paints via image resolver with fallback to type. Tests `TestListStylePositionInside`, `TestListStyleImage` |
| `quotes` | Implemented | Two-string pair inherited; `content: open-quote` / `close-quote` with nesting depth. Test `TestQuotes` |
| `counter-reset` / `counter-increment` / `counter()` / `content` | Implemented | Decimal counters on `::before`/`::after`, nested `counters(name, ".")`, and `content` text/attr/quotes/counters. Tests `TestCounterInBefore`, `TestCounterResetIncrementLayout`, `TestQuotes`, `phase79_test.go` |

### 2.4 Color & background

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `color` | Implemented | `style.go:402-405`; consumed `inline.go:152-155`; test `TestCascadeAndInline`. `hsl()`/`hsla()` parse in `ParseColor` (`values.go`; `TestParseColorHsl`) |
| `background-color` | Implemented | `style.go:406-409`; painted `layout.go:234-237, 504-507, 531-534` (gated by `Background`); tests `TestBackgroundFill`, `TestRunPDFStyleTableImage` |
| `background` (shorthand) | Implemented | Color token plus multi-layer background-image `url(...)` / gradients (`background_image.go`). Tests `TestBackgroundImageParse`, `phase79_test.go` |
| `background-image` | Implemented | Multi-layer background images with pure-Go linear and radial gradient rasterization and external image layers (`background_image.go`, `gradient.go`). Missing image skipped. Tests `TestBackgroundImageLayoutPaints`, `phase79_test.go` |
| `outline` / `outline-width` / `outline-style` / `outline-color` / `outline-offset` | Implemented | Stroke outside the border edge; does not affect layout size. solid/dashed/dotted. Tests `TestOutlineParse`, `TestOutlineStroke` |
| `box-shadow` / `-webkit-box-shadow` | Implemented | Multi-layer box-shadows with inset layers, offset fill, spread expansion, and blur approximation (`box_shadow.go`). Does not change layout size. Tests `TestBoxShadowParse`, `TestBoxShadowPaints`, `TestBoxShadowBlurPaints`, `phase79_test.go` |
| `opacity` | Implemented | Parsed in `applyRestProps`; paint via PDF ExtGState (`SetOpacity`). Nested opacities multiply. Accepts `filter: opacity()`. |
| `accent-color` | Implemented | Parsed `style_properties.go`; inherited. Fill color for form controls (`widgetValueColor` `layout.go`). Tests `widget_color_test.go`, `phase79_test.go` |

### 2.5 Table subset

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `border-collapse` (`collapse|separate`) | Implemented | `collapse` resolves adjacent borders and suppresses border-spacing (`emitCollapsedRowGrid`). Tests `TestBorderSpacing`, `TestTableLayout` |
| `border-spacing` | Implemented | `style.go`; used by `tableSpacing` (suppressed when collapse); test `TestBorderSpacing` |
| `caption-side` | Implemented | `top` (default) above the grid; `bottom` below; `left`/`right` sit beside the grid (caption shrink-to-fit capped at 40% of table width, grid gets the rest). Tests `TestCaptionSideParse`, `TestCaptionSideBottom`, `TestCaptionSideLeft`, `TestCaptionSideRight` |
| `table-layout` (`auto|fixed`) | Implemented | Consumed when `fixed` and table width is definite (`layout_tables.go:45`). Column used widths from hints, leftover split evenly; content max-content ignored (`sizeFixedTableColumns` `layout_measure.go:830`). `auto` remains the default path. Test `TestTableLayoutFixedIgnoresContentMax` |

### 2.6 Print / paged media

| `page-break-before/after/inside` and `break-before/after/inside` | Implemented | Same apply arms (`applyPageBreakProps` `style_properties.go:1467`). Stored as `style.PageBreak*` `always` / `avoid` / unset. Paginator honors them as canvas-Y flow shifts: `avoidInside` `paint_flow.go:488`, `beforeAlways` `paint_flow.go:822`, `afterBreaks` `paint_flow.go:1037`. Tests `TestPageBreakParsing`, `TestPageBreakBeforeAlways`, `TestPageBreakInsideAvoid`. Alias table below. |
| `orphans`, `widows` | Implemented | CSS `orphans` / `widows` **parsed** (integer ≥1, inherit, initial 2) in `applyRestProps`; Fragmentation Rule 3 enforced when line boxes are countable (`paint_flow.go` `orphansWidows`). Geometric short-block **heuristic** remains as fallback when line counts are unavailable (fixture-30). See fixture-37. |
| `@page` unnamed `margin` / `size` | Implemented | Last unnamed `@page` still fills `Stylesheet.Page`. Convert applies that box to every page (`applyCSSPageMargins`). |
| `@page :first` | Partial | Parsed onto `Pages` with `Sel ":first"`. Page 1 can use a different **margin**. Size stays unnamed-only (one page size per document). `:first` wins over `:left`/`:right` on page 1. Proof: `TestParsePageSelectors`, `TestPageFirstMargins`, `TestPageFirstWinsOverLeftRight`. |
| `@page :left` / `:right` | Partial | Margins applied. Honest LTR print: page 1 is `:right` (recto), even pages `:left`, odd pages `:right`. Not a duplex sheet; `break-before: left` still aliases to `always`. Size unnamed-only. Proof: `TestPageLeftRightMargins`. |
| `page` (`auto` / ident) | Implemented | Used value stored on `ResolvedStyle.PageName`. Unspecified/`auto` keeps the parent used name. A sibling whose used name changes gets `break-before: always`. `@page ident { margin }` applies on pages that overlap a box with that name. Link and outline destinations use the same named-page, side, and first-page cascade after page names are recorded. Size unnamed-only. Proof: `TestPageNameInherits`, `TestPageNameBreak`, `TestPageNamedMargins`, `TestPageMarginsSharePaintCascade`. |
| `@page` margin boxes (`@top-center` and friends) | Partial (lite) | Parses `@top-left/center/right` and `@bottom-left/center/right` quoted `content` strings (`css/page_margin.go`). Unnamed `@page` boxes fill empty CLI header/footer slots. Occupied CLI slots and `--header-html` win. `counter()` / `running()` drop. Proof: `TestParsePageMarginBoxes`, `TestPageMarginBoxes`, `TestPageMarginBoxesCLIWins`. |

**Break aliases (54.2).** `page-break-*` and `break-*` share one store. The PDF writer has no left/right or even/odd page side (duplex is out of scope, §5). `break-before: left` does **not** force a left page; it aliases to page `always`.

| Specified value | `break-before` / `page-break-before` and `*-after` | `break-inside` / `page-break-inside` |
|-----------------|-----------------------------------------------------|--------------------------------------|
| `always` | `always` (`style_properties.go:1486` and `:1499`) | `always` (`style_properties.go:1513`) |
| `page` | `always` | `always` |
| `column` | `always` (new page, which also starts a new multicol line) | ignored (not in the inside switch) |
| `left`, `right` | `always` | ignored |
| `avoid` | `avoid` (`style_properties.go:1488` and `:1501`) | `avoid` (`style_properties.go:1515`) |
| `avoid-page` | `avoid` | `avoid` |
| `avoid-column` | ignored (not mapped to page `avoid`; wiki reference lists) | ignored (`style_properties.go:1511`) |
| `recto`, `verso`, other | ignored | ignored |

### Feature checklist (page geometry, tables, pagination)

| Feature | Status | Notes / verified by |
|---------|--------|---------------------|
| Page size (A4/Letter/…), landscape, margins (mm) | Implemented | `settings.ParsePageSize` (`settings/pagesize.go:39`), `convert.pageGeometry` (`convert.go:138`) |
| `colspan` | Yes | `colSpan`; test `TestTableColspan` |
| `rowspan` | Yes / Implemented | column occupancy + height growth (`placeTableCells`, `growRowspanRows`); tests `TestTableRowspan*` |
| `border-collapse` | Implemented | see §2.5 |
| Pagination | Fragment + whole-op + phase-18 polish | rect-type ops (fill/stroke/line) split at page boundaries; text/images/links move wholly (line-level) (`paint_flow.go`); `page-break-before/after: always`, `page-break-inside: avoid`, table rows never split; **`<thead>` / `table-header-group` repeat** on continuation pages (`repeatTableHeaders`, fixture-23); CSS `orphans`/`widows` parsed + Rule 3 when line boxes exist (heuristic fallback; fixtures 30/37); `--zoom` forwarded; smart-shrinking re-layouts. `Result.Locations` for outlines/links. Break aliases in §2.6. See "Pagination" note below. |
| Floats / absolute positioning | Float lite + absolute/fixed/sticky lite | float/`clear` lite (§2.2); relative/absolute/fixed lite; sticky = print page scrollport + overflow@0 (§2.2; fixture-31) |
| Flexbox / Grid | Partial | Stage A flex + Stage B grid (areas/dense/`minmax`) + Stage C lite (§2.7 / §2.8). Paths: `flex.go`, `grid.go`, `style.go`; fixtures 25/28/32-35; plan `plans/0.2.0/phases/subplans-tier-2/flex-grid-full.md`. **Not** Bootstrap/Tailwind / Chrome layout-test parity |
| Multicol | Partial | Report lite: `column-count`/`column-width`/`columns`, `column-gap` (normal to 1em), `column-span:none\|all`, `column-fill:balance\|auto`; column boxes do not straddle pages (§2.9; `multicol.go`; fixture-39) |
| Transforms (static 2D) | Implemented | `transform` + `transform-origin` paint CTM; stacking + abs/fixed CB; sibling flow unchanged. No animation timelines; no 3D; 2D image filter (opacity, blur, grayscale, invert, adjustments) on raster images + CSS `opacity()` on elements; no CSS shader/SVG filter composition. Fixture-40; `transform.go`, `filter.go` |
| JavaScript | No | `<script>` stripped at load; no engine. `--enable-javascript` is an **unknown option** (Policy A) |
| Image-mode text | TTF outline raster | same Liberation faces as PDF; pure-Go coverage AA (`internal/imageout/ttfraster.go`); 5×7 bitmap only if an op has no font |

**Pagination (phases 5 + 18).** Box-aware fragmentation: rect-type ops crossing a page boundary are split; text, images and links move wholly (line-level). `page-break-before/after: always` and `page-break-inside: avoid` via canvas-Y flow shifts (`paint_flow.go`); table rows never split. **Table headers repeat** across pages (`repeatTableHeaders`; fixture-23). **`--zoom`** is forwarded to `layout.Options.Zoom` (`convert.go`; `TestZoom`). **Smart-shrinking** detects over-wide content and **re-layouts** with an effective zoom (`TestRunPDFSmartShrinking`). CSS `orphans`/`widows` are parsed (initial 2) and Fragmentation Rule 3 is applied when line boxes are available; the geometric short-block heuristic remains for edge cases (fixtures 30/37). Break value aliases (`left`/`right`/`page`/`column` -> `always`, `avoid-column` ignored) are in §2.6; they do not create even/odd pages. `Result.Locations` carries element boxes for outlines/links.

### 2.7 Flexbox (Stage A - print CSS subset)

Evidence: `internal/layout/flex.go`, `style_cascade.go` (`applyRestProps`) and `style_properties.go` / `style_values.go` (`parseFlexShorthand`); fixtures 25/28/32/33; `flex_test.go`. Status uses the §2 legend (Implemented / Partial / Not implemented). Checklist form: **[x]** Implemented · **[~]** Partial · **[ ]** Missing.

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `display: flex` / `inline-flex` | [x] Implemented | Routed to `buildFlex`; fixtures 25/28/32 |
| `flex-direction` (`row` | `column` | `row-reverse` | `column-reverse`) | [x] Implemented | `style.go`; row/column + reverse paths in `flex.go`; `TestFlexRowReverse` |
| `flex-wrap` (`nowrap` | `wrap` | `wrap-reverse`) | [x] Implemented | Multi-line wrap; wrap-reverse reverses line order |
| `flex-flow` | [x] Implemented | Shorthand sets `flex-direction` and `flex-wrap`. Test `TestFlexFlowShorthand` |
| `justify-content` (`flex-start` | `flex-end` | `center` | `space-between` | `space-around` | `space-evenly`) | [x] Implemented | Row + column (definite height); `TestFlexSpaceEvenly` |
| `align-items` (`stretch` | `flex-start` | `center` | `flex-end`) | [x] Implemented | Row cross-axis stretch sizes auto-height items to the flex line (fixture-33); start/center/end honored; `TestFlexAlignItemsStretchRow` |
| `align-self` | [x] Implemented | Overrides container; stretch follows same rules; `TestFlexAlignSelf` |
| `align-content` (multi-line) | [x] Implemented | Distributes free cross space when container height is definite and wrap produced ≥2 lines. `stretch` grows each line's cross size; auto-height items with `align-items`/`align-self` stretch fill the grown line. Height:auto packs at start. Row wrap only. `TestAlignContentStretch` |
| `place-content` / `place-items` / `place-self` | [x] Implemented | Shorthands expand to align-* and justify-*. Test `TestPlaceShorthands` |
| `gap` / `row-gap` / `column-gap` | [x] Implemented | Independent longhands; shorthand fills both when longhands unset (`flexGaps`) |
| `flex` shorthand (`none` \| `auto` \| grow/shrink/basis) | [x] Implemented | `parseFlexShorthand` |
| `flex-grow` / `flex-shrink` / `flex-basis` | [x] Implemented | Length/%/auto basis; post grow/shrink min/max-width clamp; column grow/shrink when height definite |
| Content-based min-size floor | [x] Partial polish | `flexMinMainSize` / `flexClampMainWidths`: content-based `min-width:auto` + `%` min re-resolve on definite containers; overflow non-visible -> auto min 0 (Flexbox §4.5 lite). Deep multi-pass intrinsic still out |
| `order` | [x] Implemented | Stable sort before place |
| Percentage basis cyclic sizing | [x] Partial | Definite CB: `%` vs content main size; indefinite/cyclic -> treat as `auto` (content) (fixture-33 / `TestFlexBasisPercent*`) |
| Nested percentage / intrinsic flex iterations | [x] Partial polish | Definite-item `%` children re-resolve against used main size; indefinite CB `%` -> auto/content; not full Flexbox intrinsic passes |

### 2.8 CSS Grid (Stage B + Stage C lite - print CSS subset)

Evidence: `internal/layout/grid.go`, `style.go`; fixtures 28/32/34/35; `grid_test.go`. **Not** full Grid L1 / L3 / Chrome parity.

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `display: grid` / `inline-grid` | [x] Implemented | Routed to `buildGrid`. `inline-grid` is inline-level (`isInlineChild`) and stays in the paragraph IFC; `display:grid` still block-breaks. `TestInlineGridIsInlineLevel` |
| `display: subgrid` | [~] Partial | `inheritSubgridFromParent` copy-inherits parent template columns (and unspecified gaps); tracks re-resolve against the subgrid's own content box. Joint Resolve Intrinsic / full subgrid L1 out of scope |
| `grid` / `grid-template` | [x] Implemented | Shorthands expand to areas, columns, and rows. Test `TestGridTemplateShorthand` |
| `grid-template-columns` | [x] Implemented | Lengths, `fr`, `repeat(N, ...)`, `minmax(...)`; gap subtracted before `fr` distribute |
| `grid-template-rows` | [x] Implemented | Consumed when height definite; fixed mins on auto-height; fixture-32 |
| `minmax()` track sizing | [x] Implemented | Lengths / `%` (definite) / `fr` / `auto` / `min-content` / `max-content` subset; `fr` keeps min floors (fixture-35) |
| `gap` / `row-gap` / `column-gap` | [x] Implemented | Independent (`gridGaps`); `TestGridRowGapVsColumnGap` asserts the row gap is at least 8pt while column gap stays distinct |
| `grid-column` / `grid-column-start` / `grid-column-end` / `span N` | [x] Implemented | Line numbers + span; 2D occupancy |
| `grid-row` / `grid-row-start` / `grid-row-end` / `span N` | [x] Implemented | Row span + stretch into spanned tracks; `TestGridRowSpan*` |
| Auto-flow placement (row / column) | [x] Implemented | Sparse row default; column major via `grid-auto-flow: column` |
| `grid-auto-flow: dense` | [x] Implemented | `dense` / `row dense` / `column dense` hole-fill (fixture-34) |
| `grid-template-areas` / `grid-area` names | [x] Implemented | Named areas + lite line form; areas can extend auto tracks (fixture-34) |
| `justify-items` / `align-items` | [x] Implemented | Default stretch; start/center/end; stretch sizes item to grid area |
| `justify-self` / `align-self` | [x] Implemented | Overrides container; stretch fills area |
| Masonry | [~] Partial | `grid-template-rows: masonry` packs items into the shortest column (`emitMasonryItems`). Not full CSS Grid L3 masonry / Chrome parity |
| Intrinsic / nested % track cycles | [x] Partial | Measure-pass lite for min/max-content track mins; cyclic `%` -> auto when CB indefinite |

### 2.9 CSS Multi-column (report lite)

Evidence: `internal/layout/multicol.go`, `style_cascade.go` (`applyRestProps`) and `style_properties.go`; fixture-39; `multicol_test.go`. **Not** full Multicol L1 / L2 / Chrome balancing with floats.

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `column-count` (`auto` \| integer ≥1) | [x] Implemented | Establishes multicol when ≠ auto; `TestMulticolParseProps` |
| `column-width` (`auto` \| `<length>`) | [x] Implemented | Used count/width per Multicol §3.3; `TestUsedColumnCountWidth` |
| `columns` shorthand | [x] Implemented | `parseColumnsShorthand` |
| `column-gap` (`normal` \| `<length>`) | [x] Implemented | Multicol: `normal` → 1em; flex/grid still treat unset/normal as 0 gap |
| `column-span` (`none` \| `all`) | [x] Implemented | Mid-flow spanner; preceding columns balance - fixture-39 / `TestMulticolColumnSpanAll` |
| Nested multicol (2 levels) | [x] Implemented | Outer / inner geometry isolated - `TestMulticolNestedTwoLevels` |
| Column box pagination | [x] Implemented | Column boxes do not cross page boundaries; new multicol line on next page - `TestMulticolLinesDoNotStraddlePages` |
| `break-*: column \| avoid-column` | [~] Partial | `column` on before/after aliases to page `always` (`applyBreakBeforeProps` `style_properties.go:1486`). `avoid-column` is **ignored** (not page `avoid`). See §2.6 alias table. |
| `column-rule` / `column-rule-width` / `column-rule-style` / `column-rule-color` | [x] Implemented | Border-style subset (`solid` / `dashed` / `dotted` / `none`); width `thin` / `medium` / `thick` plus lengths; color `currentColor`. Vertical rule centered in `column-gap`; gap 0 or `none` paints nothing. No column-axis (horizontal) rule. `TestColumnRuleParse`, `TestColumnRulePaints` |
| L2 integer spans, overflow columns | [ ] Missing | Deferred (see `plans/0.2.0/phases/tier-2-pending-3/multicol.md` out of scope) |

## 3. Supported units

Status legend as in §2; resolution sites: `LengthToPt`
`internal/css/container.go:113` (`ex`/`ch` as 0.5em at lines 133-134),
`lengthBox` `style_values.go:598` (width/height/min/max), `marginLen`
`style_values.go:564` (margins/padding/letter-spacing), parse gate
`ParseLength` `values.go:108`.

| Unit | Status | Notes |
|------|--------|-------|
| `px` | Implemented | 1 px = 0.75 pt (96 dpi reference) |
| `pt` | Implemented | |
| `mm`, `cm`, `in` | Implemented | |
| `pc` | Implemented | `style.go:654, 689` |
| `em` | Implemented | relative to element font-size (font-size, margins, lengths) |
| `rem` | Implemented | 16 px reference (`style.go:593, 657, 694`) |
| `%` | Implemented | containing block for box/margins; parent font-size for `font-size` |
| `vw`, `vh` | Partial | resolved for width/height/min/max (`lengthBox` `style.go:662-663`) only; ignored for margins/padding/font-size |
| `ex` | Partial | Resolved as 0.5em (`LengthToPt` `internal/css/container.go:133-134`, `exChToEmFactor`). Not font-metric x-height. |
| `ch` | Partial | Layout uses the default Liberation face U+0030 DIGIT ZERO advance at the element's font-size (`lengthToPt` / `GlyphAdvancePoints`). Falls back to 0.5em when the face is missing or the advance is 0. Media/container queries still use `LengthToPt` 0.5em. Test `TestChUsesZeroGlyphAdvance`. |
| `calc()` | Partial | Three-token subset only: `calc(A + B)`, `calc(A - B)`, `calc(A * N)` (`calcLength` `style_values.go:763`), used from `lengthBox` / `marginLen`. Longer or nested calc stays invalid so a fallback can win. |
| `clamp()` | Partial | `clamp(min, pref, max)` via `clampLength`; no longer dropped from cascade. Nested calc inside clamp out. Test `TestClampLength`. `color-mix(` / `light-dark(` / `oklch(` still excluded. |
| `vmin`, `vmax` | Partial | Layout `vminVmaxPt` (`style_values.go:668`) on `lengthBox` / `marginLen`. `css.ParseLength` still rejects them; used values are parsed in layout. Test `TestVminVmax`. |
| `dpi`-style | Not implemented | rejected at parse |

## 4. Supported selector syntax (cascade)

Status legend as in §2; evidence in `internal/css/css.go`.

| Selector | Status | Notes / verified by |
|----------|--------|---------------------|
| Element (`h1`, `p`, …), class (`.foo`), ID (`#bar`) | Implemented | `parseCompound` `css.go:444`; matching `Match` `css.go:518`; test `css_test.go::TestMatch` |
| Universal (`*`) | Implemented | `css.go:456-459` |
| Descendant (`div p`), child (`ul > li`) | Implemented | combinators `css.go:356-362`; matching `css.go:528-543` |
| Sibling (`a + b`, `a ~ b`) | Implemented | next-sibling `+` and subsequent-sibling `~` (`css.Match`); test `TestSiblingCombinators` |
| Attribute (`[href]`, `[href="…"]`) | Partial | presence, exact `=`, word `~=`, substring `*=`, prefix `^=`, suffix `$=`, dash `|=`; ASCII `i` flag on valued selectors (`[attr=value i]`); no `s` flag. Tests `TestAttrWordAndSubstring`, `TestAttrPrefixSuffixDash`, `TestAttrIFlag` |
| `:first-child`, `:last-child`, `:nth-child(n)` | Implemented | `odd`/`even`/`an+b`/integer; tests `TestMatch`, `TestNthChildZebraSheet` |
| `:first-of-type`, `:last-of-type`, `:nth-of-type()`, `:nth-last-of-type()` | Implemented | 1-based index among same-tag siblings; `odd`/`even`/`an+b`/integer via `parseNthArg`/`matchNth`; invalid an+b never matches. Tests `TestFirstOfType`, `TestLastOfType`, `TestNthOfType`, `TestNthLastOfType` |
| `:link`, `:visited` | Partial | Print semantics: match any `a` with non-empty `href` (no visit history; `:visited` ≡ `:link`). Specificity counts as a class-level pseudo. Proof: `TestLinkVisitedPseudos`, `TestLinkPseudoColor` |
| `:hover`, `:active`, `:focus` | Not implemented (accepted, never match) | Parsed onto the compound but `matchPseudo` returns false so `a:hover` does not degrade to bare `a` |
| `::before` / `::after` | Partial / Implemented | `MatchPseudo` plus generated content (`pseudo_content.go`): quoted strings and `attr()`. Host-element rules do not apply to the host |
| `!important` | Implemented | `css.go:664-688`; separate cascade layer `style.go:221-247`; test `css_test.go::TestParseImportant` |
| Specificity (ID > class > element), inline `style` wins, `!important` overrides | Implemented | `Specificity` `css.go:578`; inline style priority `style.go:233-239`; test `css_test.go::TestSpecificity` |
| `@media print` / `screen` filtering | Implemented | `MediaMatches` (`css/media.go`); cascade `style.go`; convert `Media: "print"` (PDF) or `"screen"` (Image); only `print` and `screen` are evaluated (all other media types and unsupported feature queries evaluate to false); tests `TestParseMedia`, `TestMediaMatches*` |
| `@media` feature queries (`(min-width: …)`) | Partial | size features + orientation vs viewport; unknown features → false; `TestMediaMatchesSizeFeatures` |
| `:has()` | Partial | Relative selectors inside `:has(...)`; descendant/child/sibling + simple compounds; no forgiving-selector list / complex chrome edge cases. `has.go`; fixture-41 |
| `:not()` | Implemented | `appendFunctionalPseudo` (`css.go:1058`); match `matchNone` (`css.go:1459`). Argument list is strict (empty items fail). Specificity of the most specific argument (`css.go:1744`). Tests `has_test.go` |
| `:is()` | Implemented | Strict selector-list arguments; nested `:is` allowed; `::` in args rejected. Match any argument. Specificity of the most specific argument. Tests `TestParseIs`, `TestIsPseudo`, `TestIsSpecificity` |
| `:where()` | Implemented | Same matching as `:is()`; specificity contribution 0. Test `TestWherePseudo` |
| `:root` | Implemented | Matches the document element (`<html>`), not the synthetic `#document` wrapper (`matchPseudo` `css.go:1437`; `isRootElement` `css.go:1473`). Test `TestRootPseudo` |
| `var()` / `--*` custom properties | Partial | `--*` inherit then overlay (`mergeCustomProps` `style_cascade.go:18`); `var()` expanded before apply (`resolveRawVars` `style_cascade.go:45`; `ResolveCustomProps` `values.go:514`). Cycles resolve empty. Tests `cssvar_font_test.go`, `TestResolveCustomProps*` |
| `@container` | Partial | Size queries only (`inline-size`/`width` + `and`/`or`/`not`); named containers; two-pass style after used inline size. No style/scroll-state queries; no `cq*` units. `internal/css/container.go`; fixture-42 |
| `container-type` / `container-name` / `container` | Implemented | Parsed `applyContainerProps` (`style_properties.go:1296`). Size containers measured in `layout/container.go:43`. `container-type` honors `normal`/`size`/`inline-size` for size `@container` queries. Fixture-42 |
| `@page` | Partial | Unnamed `margin`/`size` on every page. `:first` / `:left` / `:right` override **margin** (LTR page 1 is `:right`; `:first` wins on page 1). `page: ident` used-value inherit plus sibling break; named `@page` margin on pages that overlap that name. Size unnamed-only. Margin boxes: unnamed quoted `@top-*` / `@bottom-*` map to CLI HF empty slots. See §2.6. |
| `@font-face` | Partial | Parsed; `MergeFontFaces` loads TTF/OTF/WOFF1 via `FetchSub` (local **and** `https://`) under the same ACL + `NetworkPolicy` on PDF and image paths. `.woff2` / `.eot` / `data:` skipped. See §5 |
| `@import` | Partial | Parsed onto `Stylesheet.Imports`; `CollectSheets` fetches under the same ACL as `<link>`, depth cap 8, cycle skip, failed fetch skipped. Media prelude uses `MediaMatches`. Tests `TestParseImport`, `TestImportStylesheet` |

## 5. Explicitly unsupported (MVP)

| Feature | Handling |
|---------|----------|
| JavaScript / `<script>` / DOM APIs | **Stripped at load.** No JS engine. `--enable-javascript` and other JS flags are **unknown options** (Policy A) |
| Full CSS Grid / full Flexbox | Stage A/B print CSS subset **shipped** (§2.7 / §2.8); Stage C lite + flex min-size polish + Partial subgrid/masonry span (`tier-2-pending-3/flex-grid-remaining.md`). **Not** Bootstrap/Tailwind / Chrome layout-test parity |
| `transform`, `filter`, `animation`, `transition` | Partial / out of scope | **Static 2D** `transform` + `transform-origin` Implemented (translate/scale/rotate/matrix/skew*; paint CTM; stacking + abs/fixed CB). Sibling flow unchanged. **`filter`:** 2D image filter (opacity, blur, grayscale, invert, adjustments) on raster images (`filter.go`) + CSS `opacity()` on elements; no CSS shader/SVG filter composition. **`animation`/`transition`/`@keyframes`:** parse-ignored (static cascaded value only; no timelines). **3D / perspective:** permanent non-goal. Fixture-40; `transform.go` / `transform_test.go`, `filter.go` |
| `background-image` / gradients | **Implemented** multi-layer `url(...)` and pure-Go linear/radial gradient rasterization (`background_image.go`, `gradient.go`) |
| `@font-face` (remote / WOFF2) | **Partial:** local **and `https://`** TTF/OTF/WOFF1 via `FetchSub` (same ACL + `NetworkPolicy` as other subresources) on PDF/image paths. **`.woff2` / `.eot` / `data:`** skipped. Missing faces fall back to registry / Liberation |
| Custom XSLT TOC (`--xsl-style-sheet`) | Not implemented (no XSLT in stdlib); Go templates instead (Phase 6) |
| SVG-as-`<img>` / SVG presentation | **Implemented** SVG-as-`<img>` rasterization via `internal/svg`; 5 CSS properties (`fill`, `stroke`, `stroke-width`, `fill-opacity`, `stroke-opacity`) parsed in style; remaining 53 SVG presentation properties are unsupported |
| Masking / clipping | **Implemented** `overflow-clip` for descendant box clipping (`overflow_clip.go`); `clip-path` and CSS `mask-*` properties are unsupported |
| CSS Regions & Exclusions | **Not implemented** (`flow-from`, `flow-into`, `flow-tolerance`, `region-fragment`, `wrap-after`, `wrap-before`, `wrap-flow`, `wrap-inside`, `wrap-through` are permanent non-goals for print PDF) |
| WebP, AVIF | Not implemented; broken-image placeholder or skip |
| Fixed CSS headers/footers via `position: fixed` alone | Prefer CLI `--header-*` / `--footer-*` for repeating chrome; CSS `fixed` lite paints on every page but is not a full running-element model |
| Complex-script shaping (Indic, Arabic, CJK) | **Type0/CID Identity-H** for BMP Unicode (CJK with a capable face); **Arabic OT** via `go-text/typesetting` when the face has GSUB (+ presentation-form `ShapeText` fallback); Hangul needs a Hangul face. `writing-mode` vertical keywords inherit and rotate glyphs (`RotateDeg == -90`); block/line layout stays horizontal. **Indic Partial** (OT when face/cmap allow; not production-claimed). Optional OT **`halt`/`palt`** for CJK punctuation via `ShapeTextFont` FontFeatures |
| PDF version (1.4 / 1.7 / 2.0) | **Supported:** PDF 1.4 is default; PDF 1.7 and PDF 2.0 are opt-in via `--pdf-version 1.7` / `--pdf-version 2.0` or library field `Document.PDFVersion`. Emits `%PDF-1.7` (trailer `/ID`, Info with UTF-16BE + BOM strings, non-claiming XMP Metadata stream) or `%PDF-2.0` (trailer `/ID`, UTF-8 document strings, non-claiming XMP with `dc:format`, `pdf:Producer`, dates). PDF 2.0 output is a **version**, not a conformance claim |
| PDF/A-3a, PDF/UA-1 (ISO 19005-3 / ISO 14289-1) | **Supported:** Opt-in via `--pdf-profile a3a-ua1` / library field `Document.PDFProfile` (`"a3a-ua1"`, `"a3a"`, `"ua1"`). Implies PDF 1.7. Emits claiming XMP metadata (`pdfaid:part=3`, `pdfaid:conformance=A`, `pdfuaid:part=1`), sRGB OutputIntent, `/DefaultRGB`, and full logical structure tree (`H1`..`H6`, `P`, `Table` > `TR` > `TH`/`TD`, `L` > `LI` > `LBody` > `Link`, `Figure` + `alt`, `/Artifact /Pagination`) |
| PDF 2.0 (ISO 32000-2) | **Shipped as opt-in version** (#32): `--pdf-version 2.0` / `Document.PDFVersion = "2.0"`. Version alone is **not** a PDF/A or PDF/UA claim |
| PDF/A-4, PDF/UA-2 (PDF 2.0 conformance profiles) | **Supported** (#33): Opt-in via `--pdf-profile a4-ua2` / `Document.PDFProfile` (`"a4-ua2"`, `"a4"`, `"ua2"`). Implies PDF 2.0. Emits claiming XMP (`pdfaid:part=4`, `pdfaid:rev=2020`, `pdfuaid:part=2`, `pdfuaid:rev=2024`), sRGB+Gray OutputIntent / Default* ICCBased, structure `/Namespace`, `ListNumbering` on lists, structure destinations on internal links, and full logical structure tree (`L` > `LI` > `LBody` > `Link`) |
| PDF encryption, duplex, AcroForm | Out of scope (not in original wkhtmltopdf either) |

### 5.1 Deferred niche and draft families (94 properties - Not implemented)

The following 94 properties are intentionally left **unsupported** (Not implemented). They have no print PDF consumer and are out of scope for the print PDF engine. Declarations are parsed as valid property names where recognized and then ignored with graceful degrade, never claimed as Implemented. No Implemented claims are made for any of the 94.

| Family | Count | Status | Note | Examples (not exhaustive) |
|--------|-------|--------|------|---------------------------|
| Draft corner-shape CSS (34 properties: corner, corner-block-*, corner-inline-*, etc.) | 34 | Not implemented | Draft corner-shape CSS; not in print PDF engine. Left unsupported. `corner-*` is not `border-radius` (see §2.1). | `corner`, `corner-shape`, `corner-block-start-shape`, `corner-inline-end-shape`, `corner-top-left-shape`, `corner-bottom-right-shape`, etc. |
| Ruby/MathML/rhythmic niche (33 properties: block-ellipsis, block-step-*, box-snap, ruby-*, math-*, etc.) | 33 | Not implemented | Ruby/MathML/rhythmic niche; not implemented for print PDF. Left unsupported. | `block-ellipsis`, `block-step-*`, `box-snap`, `ruby-align`, `ruby-position`, `math-depth`, `math-style`, `line-snap`, etc. |
| Draft gap/row-rule decorations (27 properties: row-rule*, rule*) | 27 | Not implemented | Draft gap/row-rule decorations; no print consumer. Left unsupported. `row-rule*` and `rule*` are not `column-rule` (see §2.9). | `row-rule`, `row-rule-break`, `row-rule-color`, `row-rule-style`, `row-rule-width`, `rule`, `rule-break`, `rule-color`, `rule-style`, `rule-width`, etc. |

Total: 94 properties across three families. All are Not implemented and left unsupported with no print PDF consumer.

### 5.2 Phase 83 hard-defer categories (87 properties - Not implemented)

The following 87 properties across three categories are intentionally left **unsupported** (Not implemented) as hard-deferred or permanent non-goals for print PDF:

| Category | Count | Status | Description and scope |
|----------|-------|--------|-----------------------|
| SVG presentation & geometry (`B_svg_presentation`) | 53 | Not implemented | Remaining SVG presentation and geometry properties. SVG-as-`<img>` is implemented via `internal/svg` rasterizer; 5 CSS properties (`fill`, `stroke`, `stroke-width`, `fill-opacity`, `stroke-opacity`) are parsed in style; remaining 53 SVG presentation properties are unsupported (`alignment-baseline`, `baseline-shift`, `color-interpolation`, `cx`, `cy`, `d`, `dominant-baseline`, `fill-break`, `fill-color`, `fill-image`, `fill-origin`, `fill-position`, `fill-repeat`, `fill-rule`, `fill-size`, `glyph-orientation-vertical`, `image-rendering`, `marker`, `marker-end`, `marker-mid`, `marker-side`, `marker-start`, `paint-order`, `path-length`, `r`, `rx`, `ry`, `shape-rendering`, `stop-color`, `stop-opacity`, `stroke-align`, `stroke-alignment`, `stroke-break`, `stroke-color`, `stroke-dash-corner`, `stroke-dash-justify`, `stroke-dashadjust`, `stroke-dasharray`, `stroke-dashcorner`, `stroke-dashoffset`, `stroke-image`, `stroke-linecap`, `stroke-linejoin`, `stroke-miterlimit`, `stroke-origin`, `stroke-position`, `stroke-repeat`, `stroke-size`, `text-anchor`, `text-rendering`, `vector-effect`, `x`, `y`). |
| Mask, clip, and filter effects (`B_mask_clip_filter_effects`) | 25 | Not implemented | Masking, clipping, and filter primitives. `overflow-clip` is implemented for descendant box clipping (`overflow_clip.go`); 2D image filter (opacity, blur, grayscale, invert, adjustments) on raster images + CSS `opacity()` on elements are implemented (`filter.go`); `clip-path`, CSS `mask-*` properties, `backdrop-filter`, and CSS shader/SVG filter composition are unsupported (`backdrop-filter`, `clip`, `clip-path`, `clip-rule`, `color-interpolation-filters`, `flood-color`, `flood-opacity`, `lighting-color`, `mask`, `mask-border`, `mask-border-mode`, `mask-border-outset`, `mask-border-repeat`, `mask-border-slice`, `mask-border-source`, `mask-border-width`, `mask-clip`, `mask-composite`, `mask-image`, `mask-mode`, `mask-origin`, `mask-position`, `mask-repeat`, `mask-size`, `mask-type`). |
| CSS Regions & Exclusions (`B_regions_exclusions`) | 9 | Not implemented | CSS Regions and Exclusions are permanent non-goals for print PDF (`flow-from`, `flow-into`, `flow-tolerance`, `region-fragment`, `wrap-after`, `wrap-before`, `wrap-flow`, `wrap-inside`, `wrap-through`). |

Total: 87 properties across three hard-defer categories. All remain Not implemented.

### 5.3 Vendor-prefix aliases (-webkit-) - Phase 69 and Phase 82

This section is the contract for vendor-prefixed alias handling. Alias mechanism is
`normalizeVendorPrefix` at `internal/layout/style_cascade.go:913` called from
`applyStyleProp`, plus `display` value remaps at `internal/layout/style_properties.go:81`
and value remaps in `remapWebkitValue` at `internal/layout/style_cascade.go:1000`.

- **Total tracked:** 70 `-webkit-` alias names.
- **Implemented:** 28 aliases that map to an already Implemented unprefixed base.
- **Unsupported:** 42 aliases whose bases are not Implemented or are print-noop. These are
  not Implemented and degrade gracefully (ignored declaration).

Do not treat Unsupported aliases as Implemented. Tests for Implemented aliases live in
`internal/layout/style_cascade_test.go:188` (`TestWebkitPrefixAliases`).

#### 5.3.1 Implemented vendor aliases (28)

28 = 22 from Phase 69 + 6 new in Phase 82 slice A. Each entry has a prefixed-name
test and a consumer in layout or paint (the unprefixed base is already Implemented).

| Vendor alias | Canonical property | Notes / base |
|--------------|--------------------|--------------|
| `-webkit-box-sizing` | `box-sizing` | `normalizeVendorPrefix` `style_cascade.go:913` |
| `-webkit-border-radius` | `border-radius` | same |
| `-webkit-border-top-left-radius` | `border-top-left-radius` | same |
| `-webkit-border-top-right-radius` | `border-top-right-radius` | same |
| `-webkit-border-bottom-left-radius` | `border-bottom-left-radius` | same |
| `-webkit-border-bottom-right-radius` | `border-bottom-right-radius` | same |
| `-webkit-transform` | `transform` | same |
| `-webkit-transform-origin` | `transform-origin` | same |
| `-webkit-flex` | `flex` | flex shorthand |
| `-webkit-flex-basis` | `flex-basis` | same |
| `-webkit-flex-direction` | `flex-direction` | same |
| `-webkit-flex-flow` | `flex-flow` | same |
| `-webkit-flex-grow` | `flex-grow` | same |
| `-webkit-flex-shrink` | `flex-shrink` | same |
| `-webkit-flex-wrap` | `flex-wrap` | same |
| `-webkit-justify-content` | `justify-content` | same |
| `-webkit-align-content` | `align-content` | same |
| `-webkit-align-items` | `align-items` | same |
| `-webkit-align-self` | `align-self` | same |
| `-webkit-order` | `order` | same |
| `-webkit-box-shadow` | `box-shadow` | alias to `box-shadow` (see `box-shadow` row in 2.4) |
| `-webkit-filter` | `filter` | alias to `filter` |
| `-webkit-box-align` | `align-items` | **New in Phase 82 slice A.** Value remap `start`->`flex-start`, `end`->`flex-end`, `center`->`center`, `stretch`->`stretch`, `baseline`->`baseline` via `remapWebkitValue` `style_cascade.go:1000` |
| `-webkit-box-flex` | `flex-grow` | **New in Phase 82 slice A.** Direct value pass-through to `flex-grow` |
| `-webkit-box-ordinal-group` | `order` | **New in Phase 82 slice A.** Value remap `N` -> `N-1` (1-based group to 0-based order) via `remapWebkitValue` `style_cascade.go:1000` |
| `-webkit-box-orient` | `flex-direction` | **New in Phase 82 slice A.** Value remap `horizontal`/`inline-axis`->`row`, `vertical`/`block-axis`->`column` via `remapWebkitValue` |
| `-webkit-box-pack` | `justify-content` | **New in Phase 82 slice A.** Value remap `start`->`flex-start`, `end`->`flex-end`, `center`->`center`, `justify`->`space-between` via `remapWebkitValue` |
| `-webkit-text-fill-color` | `color` | **New in Phase 82 slice A.** Alias to `color` |

Display value aliases (also Phase 82 slice A, at `internal/layout/style_properties.go:81`):

| Specified value | Used value |
|-----------------|------------|
| `display: -webkit-box` | `display: flex` |
| `display: -webkit-inline-box` | `display: inline-flex` |

#### 5.3.2 Still Unsupported vendor aliases (42)

42 aliases remain **Unsupported**. They are not Implemented. Each group notes the
blocking base. Do not claim Implemented for any of these 42.

**Group B - 3 background longhands (wait Phase 80):** bases `background-clip`,
`background-origin`, `background-size` are not yet Implemented. Aliases stay
Unsupported until those bases ship.

| Vendor alias | Blocking base | Reason |
|--------------|---------------|--------|
| `-webkit-background-clip` | `background-clip` | wait Phase 80 |
| `-webkit-background-origin` | `background-origin` | wait Phase 80 |
| `-webkit-background-size` | `background-size` | wait Phase 80 |

**Group C - 14 mask family (wait Phase 83 hard defer):** bases `mask` and
`mask-border` families are hard-deferred. No alias flips until bases are
Implemented.

| Vendor alias | Blocking base | Reason |
|--------------|---------------|--------|
| `-webkit-mask` | `mask` | wait Phase 83 hard defer |
| `-webkit-mask-image` | `mask-image` | wait Phase 83 hard defer |
| `-webkit-mask-size` | `mask-size` | wait Phase 83 hard defer |
| `-webkit-mask-repeat` | `mask-repeat` | wait Phase 83 hard defer |
| `-webkit-mask-position` | `mask-position` | wait Phase 83 hard defer |
| `-webkit-mask-origin` | `mask-origin` | wait Phase 83 hard defer |
| `-webkit-mask-clip` | `mask-clip` | wait Phase 83 hard defer |
| `-webkit-mask-composite` | `mask-composite` | wait Phase 83 hard defer |
| `-webkit-mask-box-image` | `mask-border` | wait Phase 83 hard defer |
| `-webkit-mask-box-image-source` | `mask-border-source` | wait Phase 83 hard defer |
| `-webkit-mask-box-image-slice` | `mask-border-slice` | wait Phase 83 hard defer |
| `-webkit-mask-box-image-width` | `mask-border-width` | wait Phase 83 hard defer |
| `-webkit-mask-box-image-outset` | `mask-border-outset` | wait Phase 83 hard defer |
| `-webkit-mask-box-image-repeat` | `mask-border-repeat` | wait Phase 83 hard defer |

**Group D - 20 animation / transition / 3D / UI print-noop:** bases have no print
PDF consumer and stay skipped. Aliases stay Unsupported (permanent non-goal for
print).

| Vendor alias | Blocking base | Reason |
|--------------|---------------|--------|
| `-webkit-animation` | `animation` | print-noop (no timelines) |
| `-webkit-animation-delay` | `animation-delay` | print-noop |
| `-webkit-animation-direction` | `animation-direction` | print-noop |
| `-webkit-animation-duration` | `animation-duration` | print-noop |
| `-webkit-animation-fill-mode` | `animation-fill-mode` | print-noop |
| `-webkit-animation-iteration-count` | `animation-iteration-count` | print-noop |
| `-webkit-animation-name` | `animation-name` | print-noop |
| `-webkit-animation-play-state` | `animation-play-state` | print-noop |
| `-webkit-animation-timing-function` | `animation-timing-function` | print-noop |
| `-webkit-transition` | `transition` | print-noop |
| `-webkit-transition-delay` | `transition-delay` | print-noop |
| `-webkit-transition-duration` | `transition-duration` | print-noop |
| `-webkit-transition-property` | `transition-property` | print-noop |
| `-webkit-transition-timing-function` | `transition-timing-function` | print-noop |
| `-webkit-backface-visibility` | `backface-visibility` | print-noop (3D) |
| `-webkit-perspective` | `perspective` | print-noop (3D) |
| `-webkit-perspective-origin` | `perspective-origin` | print-noop (3D) |
| `-webkit-transform-style` | `transform-style` | print-noop (3D) |
| `-webkit-appearance` | `appearance` | print-noop (UI) |
| `-webkit-user-select` | `user-select` | print-noop (UI) |

**Group E - 5 WebKit-native with no print consumer:** bases are WebKit extensions
with no consumer in the print engine. Left Unsupported unless a base plus consumer
lands.

| Vendor alias | Blocking base | Reason |
|--------------|---------------|--------|
| `-webkit-line-clamp` | `line-clamp` | WebKit-native, no print consumer |
| `-webkit-text-size-adjust` | `text-size-adjust` | WebKit-native, no print consumer |
| `-webkit-text-stroke` | `text-stroke` | WebKit-native, no print consumer |
| `-webkit-text-stroke-color` | `text-stroke-color` | WebKit-native, no print consumer |
| `-webkit-text-stroke-width` | `text-stroke-width` | WebKit-native, no print consumer |

### 5.4 Phase 84 print-noop categories (155 properties - Not implemented)

The following 155 properties across six categories are intentionally left **unsupported** (Not implemented) as print-noop non-goals. Static print PDF has no animation time loop, no interactive scroll viewport, no pointer/caret chrome, no motion or anchor timelines, no aural speech synthesis, and no 3D scene graph. Declarations are recognized where valid CSS syntax appears and dropped without error; none are claimed as Implemented.

| Category | Count | Status | Description and scope |
|----------|-------|--------|-----------------------|
| Time, animation, and transition (`A_time_animation_transition`) | 45 | Not implemented | Static print PDF has no animation time loop, transition timeline, trigger activation, or view-transition engine (`animation`, `animation-composition`, `animation-delay`, `animation-delay-end`, `animation-delay-start`, `animation-direction`, `animation-duration`, `animation-fill-mode`, `animation-iteration-count`, `animation-name`, `animation-play-state`, `animation-range`, `animation-range-center`, `animation-range-end`, `animation-range-start`, `animation-timeline`, `animation-timing-function`, `animation-trigger`, `event-trigger`, `event-trigger-name`, `event-trigger-source`, `image-animation`, `pointer-timeline`, `pointer-timeline-axis`, `pointer-timeline-name`, `timeline-trigger`, `timeline-trigger-activation-range`, `timeline-trigger-activation-range-end`, `timeline-trigger-activation-range-start`, `timeline-trigger-active-range`, `timeline-trigger-active-range-end`, `timeline-trigger-active-range-start`, `timeline-trigger-name`, `timeline-trigger-source`, `transition`, `transition-behavior`, `transition-delay`, `transition-duration`, `transition-property`, `transition-timing-function`, `trigger-scope`, `view-transition-class`, `view-transition-group`, `view-transition-name`, `view-transition-scope`). |
| Scroll snap, overscroll, and scrollbar (`A_scroll_snap_overscroll`) | 41 | Not implemented | Static print PDF has no scroll viewport, scroll snapping, scroll margin/padding offsets, scroll timeline drivers, or scrollbar gutter/color/width chrome (`overscroll-behavior`, `overscroll-behavior-block`, `overscroll-behavior-inline`, `overscroll-behavior-x`, `overscroll-behavior-y`, `scroll-axis-lock`, `scroll-behavior`, `scroll-initial-target`, `scroll-margin`, `scroll-margin-block`, `scroll-margin-block-end`, `scroll-margin-block-start`, `scroll-margin-bottom`, `scroll-margin-inline`, `scroll-margin-inline-end`, `scroll-margin-inline-start`, `scroll-margin-left`, `scroll-margin-right`, `scroll-margin-top`, `scroll-marker-group`, `scroll-padding`, `scroll-padding-block`, `scroll-padding-block-end`, `scroll-padding-block-start`, `scroll-padding-bottom`, `scroll-padding-inline`, `scroll-padding-inline-end`, `scroll-padding-inline-start`, `scroll-padding-left`, `scroll-padding-right`, `scroll-padding-top`, `scroll-snap-align`, `scroll-snap-stop`, `scroll-snap-type`, `scroll-target-group`, `scroll-timeline`, `scroll-timeline-axis`, `scroll-timeline-name`, `scrollbar-color`, `scrollbar-gutter`, `scrollbar-width`). |
| Pointer, caret, and form UI (`A_pointer_form_ui`) | 25 | Not implemented | Static print PDF has no mouse pointer, cursor styling, caret animation/color/shape, spatial navigation, dynamic field sizing, input security masking, touch gestures, user text selection, or window dragging (`appearance`, `caret`, `caret-animation`, `caret-color`, `caret-shape`, `cursor`, `field-sizing`, `input-security`, `interactivity`, `interest-delay`, `interest-delay-end`, `interest-delay-start`, `nav-down`, `nav-left`, `nav-right`, `nav-up`, `pointer-events`, `resize`, `slider-orientation`, `spatial-navigation-action`, `spatial-navigation-contain`, `spatial-navigation-function`, `touch-action`, `user-select`, `window-drag`). |
| Anchor positioning, offset motion, and view timelines (`A_anchor_timeline_motion`) | 21 | Not implemented | Interactive anchor positioning, motion path offset rotations/distances, view timelines, scroll anchoring, and paint invalidation hints have no print PDF consumer (`anchor-name`, `anchor-scope`, `offset`, `offset-anchor`, `offset-distance`, `offset-path`, `offset-position`, `offset-rotate`, `overflow-anchor`, `position-anchor`, `position-area`, `position-try`, `position-try-fallbacks`, `position-try-order`, `position-visibility`, `timeline-scope`, `view-timeline`, `view-timeline-axis`, `view-timeline-inset`, `view-timeline-name`, `will-change`). |
| Speech and aural (`A_speech_aural`) | 19 | Not implemented | Visual print PDF has no aural speech synthesis, sound cues, pauses, rests, or speech voice properties (`cue`, `cue-after`, `cue-before`, `pause`, `pause-after`, `pause-before`, `rest`, `rest-after`, `rest-before`, `speak`, `speak-as`, `voice-balance`, `voice-duration`, `voice-family`, `voice-pitch`, `voice-range`, `voice-rate`, `voice-stress`, `voice-volume`). |
| 3D transforms (`A_3d_transforms`) | 4 | Not implemented | Static 2D affine transforms (`transform`, `transform-origin`) are Implemented for paint CTM; 3D transform matrices, perspective projection, perspective origins, and backface culling are permanent non-goals for print PDF (`backface-visibility`, `perspective`, `perspective-origin`, `transform-style`). |

Total: 155 properties across six categories. All remain Not implemented.

## 6. Security policy (frozen defaults)

| Rule | Value |
|------|-------|
| Local file access | **Blocked by default** (`--allow-local-files` opt-in; `--allow` path allowlist walk) |
| Untrusted HTML | **Not supported** - same warning as upstream `docs/status.md`; use with HTML you control only |
| Remote URL fetch | `net/http` defaults: connect + response timeouts, redirect limit. Compatible mode allows `http://localhost` / RFC1918; Restricted (`--restrict-network`) does not. `file://` is gated by the local-file ACL |
| SSRF posture | No automatic form submission; POST only via explicit `--post` flags; no cookies auto-forwarded from site contexts |

## 7. CLI flag support matrix (Phase 9.1)

Extracted from every `add(...)` call in `internal/cli/flags.go`. Status is
**ground truth, not intent**: each flag's dotted setting was traced from the
`Set` surface (`internal/settings/reflect.go`) to its consumers in
`internal/convert`, `internal/load`, `internal/imageout` and
`internal/layout`.

**Policy A:** flags with no engine consumer are **not** accepted no-ops.
They fail parse with `unknown option` (`TestStubFlagsRemoved`,
`TestUnknownFlagErrors`).

- **Supported** - parsed, wired into settings, and the setting is consumed
  by the PDF/image pipeline; exercised end-to-end in `internal/convert`
  tests (incl. `TestGoldenCorpusAllFixtures`), `internal/cli` parse tests,
  or the CLI smoke run.
- **Partial** - accepted and stored, but only part of the upstream
  behavior is honored (note column says which).
- **Ignored** - accepted and stored, never consumed; no effect on output.
- **Rejected** - not registered; `--<name>` → `unknown option`.

### 7.1 Documentation flags

| Flag | Mode | Status |
|------|------|--------|
| `--help`, `--version`, `--license`, `--extended-help` | Both | Supported (handled by the parser; prints and exits 0) |
| `--man`, `--html` | Both | **Rejected** (`unknown option`) |

### 7.2 Global page/PDF flags

| Flag | Mode | Status |
|------|------|--------|
| `--quiet` | Both | Supported (suppresses progress; `TestRunPDFQuiet`) |
| `--log-level` | Both | **Rejected** (`unknown option`) |
| `--collate`, `--copies` | PDF | Supported (`TestRunPDFCopiesCollate`, `TestRunPDFCopiesNonCollate`) |
| `--orientation` | PDF | Supported (page geometry swap) |
| `--page-size` | PDF | Supported (A4/Letter/…; golden runner) |
| `--grayscale` | PDF | **Supported** (sets `Global.Grayscale`; `convert` → `doc.SetGrayscale`; `TestGrayscaleSetsConvertField`) |
| `--lowquality` | PDF | **Rejected** (`unknown option`) |
| `--title` | PDF | Supported (PDF /Title info) |
| `--margin-top/bottom/left/right` | PDF | Supported (page geometry + golden runner) |
| `--dpi` | PDF | **Rejected** (`unknown option`) |
| `--page-width`, `--page-height` | PDF | Supported (`pageGeometry` override) |
| `--pdf-version` | PDF | Supported (PDF 1.4 default; 1.7 / 2.0 opt-in; invalid values error) |
| `--pdf-profile` | PDF | Supported (PDF/A-3a, PDF/UA-1, dual 1.7 profiles; PDF/A-4, PDF/UA-2, dual 2.0 profiles) |
| `--image-quality` | Both | **Rejected** (`unknown option`). Image mode uses `--quality` |
| `--image-dpi` | PDF | **Rejected** (`unknown option`) |
| `--no-pdf-compression` | PDF | Supported (uncompressed streams; used by tests) |
| `--use-xserver` | Both | **Rejected** (`unknown option`) |
| `--cookie-jar` | Both | **Rejected** (`unknown option`) |
| `--read-args-from-stdin` | Both | **Rejected** (`unknown option`) |

### 7.3 Pagination & smart shrinking

| Flag | Mode | Status |
|------|------|--------|
| `--page-offset` | PDF | Supported (header/footer and TOC page numbers) |
| `--enable-smart-shrinking`, `--disable-smart-shrinking` | PDF | Supported (re-layout with zoom; `TestRunPDFSmartShrinking`). There is **no** bare `--smart-shrinking` flag |

### 7.4 Outline

| Flag | Mode | Status |
|------|------|--------|
| `--outline` | PDF | Supported (`TestOutlineWiring`, `TestOutlineDisabled`) |
| `--outline-depth` | PDF | Supported (outline tree truncation) |
| `--dump-outline` | PDF | Supported (wkhtmltopdf XML to stdout) |
| `--dump-default-toc-xsl` | PDF | Supported (built-in template description to stdout) |

### 7.5 Web & load (page-scoped)

| Flag | Mode | Status |
|------|------|--------|
| `--enable-javascript`, `--disable-javascript` | Both | **Rejected** (`unknown option`; no JS engine) |
| `--allow-local-files`, `--no-allow-local-files` | Both | Supported (local-file ACL; security policy §6 + golden runner) |
| `--allow` | PDF | Supported (ACL allow-prefix list). **Not registered** in image mode |
| `--restrict-network` | Both | Supported (`RestrictedNetworkPolicy`: block private destinations and cross-host redirects) |
| `--allow-host` | Both | Supported (exact or `*.example.com` host allowlist; exact entries may skip the private-IP check) |
| `--background`, `--no-background` | Both | Supported (paint gate; golden runner sets it on) |
| `--enable-plugins`, `--disable-plugins` | Both | **Rejected** (`unknown option`) |
| `--default-encoding` | Both | **Rejected** (`unknown option`) |
| `--minimum-font-size` | Both | **Rejected** (`unknown option`) |
| `--user-style-sheet` | Both | **Rejected** (`unknown option`) |
| `--print-media-type`, `--no-print-media-type` | Both | **Supported** (consumed via `ResolveMedia`; PDF default remains `print`) |
| `--simplify-dom`, `--no-simplify-dom` | Both | Supported (opt-in chrome-strip; landmarks-only `SimplifyChromeCSS`; default off; `TestSimplifyDOMOnHidesChrome`) |
| `--simplify-dom-profile` | Both | Supported (`mediawiki` adds MW selectors; empty = landmarks only; `TestSimplifyDOMMediaWikiProfile`) |
| `--print-link-underline` | Both | Supported (opt-in underline `a[href]` after cascade; default off; `TestPrintLinkUnderlineOptIn`) |
| `--media-type` | Both | **Supported** (consumed via `ResolveMedia`; PDF default remains `print`) |
| `--javascript-delay` | Both | **Rejected** (`unknown option`) |
| `--window-status`, `--run-script` | Both | **Rejected** (`unknown option`) |
| `--zoom` | Both | Supported (operator layout scale → `layout.Options.Zoom`; not stylesheet emulation) |
| `--stop-slow-scripts`, `--no-stop-slow-scripts` | Both | **Rejected** (`unknown option`) |
| `--debug-javascript`, `--no-debug-javascript` | Both | **Rejected** (`unknown option`) |
| `--load-error-handling` | Both | Supported (abort/skip/ignore in the loader) |
| `--load-media-error-handling` | Both | **Rejected** (`unknown option`) |
| `--proxy` | Both | Partial (global proxy wired into the HTTP transport; object-level `load.proxy` is stored but not applied) |
| `--username`, `--password` | Both | Supported (HTTP basic auth) |
| `--custom-header-propagation`, `--no-custom-header-propagation` | Both | **Rejected** (`unknown option`) |
| `--timeout` | Both | Supported (HTTP response timeout) |
| `--external-links`, `--no-external-links` | PDF | **Supported** (`--no-external-links` honored via `stripLinkURIs`; default on) |
| `--internal-links`, `--no-internal-links` | PDF | Partial (body `#` fragment GoTo via layout `OpLinkURI` + `applyInternalLinks`; HTML HF `#id` → body `AddLinkDest`. Geometry caveats - runs without paint boxes still skipped) |
| `--resolve-relative-links`, `--keep-relative-links` | PDF | Supported (`resolveRelativeLinkURIs`; relative `href` resolution vs keep-as-written) |
| `--font-path` | Both | Supported (extra font search directories for registry discovery) |
| `--use-system-fonts` | Both | Supported (opt-in system font dirs; default off for determinism) |
| `--produce-forms` | PDF | **Rejected** (`unknown option`) |

### 7.6 Pair flags (two values)

| Flag | Mode | Status |
|------|------|--------|
| `--cookie <name> <value>` | Both | Supported (HTTP cookies) |
| `--custom-header <name> <value>` | Both | Supported (HTTP headers) |
| `--post <name> <value>` | Both | Supported (POST form bodies) |
| `--replace <name> <value>` | PDF | Supported (header/footer substitution) |

### 7.7 Header & footer

| Flag | Mode | Status |
|------|------|--------|
| `--header-left/center/right`, `--footer-left/center/right` | PDF | Supported (text HFs; `TestTextHeaderFooter`) |
| `--header-font-size`, `--footer-font-size` | PDF | Supported |
| `--header-spacing`, `--footer-spacing` | PDF | Supported (band measurement) |
| `--header-line`, `--footer-line` | PDF | Supported (separator line) |
| `--header-font-name`, `--footer-font-name` | PDF | Partial (stored; every font renders as the embedded Liberation Sans) |
| `--header-html`, `--footer-html` | PDF | **Partial** nested HTML HF: child layout (body CSS subset, flex/grid/images, local `@font-face` via shared registry/ACL); clipped to margin band; URI + `#id` GoTo to **body** destinations only (HF-tree ids are not destinations). URL values; raw markup rejected. Tests: `TestHTMLHeader*`, `hf_links_test.go` |

### 7.8 TOC objects

| Flag | Mode | Status |
|------|------|--------|
| `--xsl-style-sheet` | PDF | Partial (accepted; warning emitted, built-in Go template used instead - no XSLT in stdlib) |
| `--toc-header-text` | PDF | Supported (`TestTOC`) |
| `--toc-text-size-shrink` | PDF | Supported |
| `--disable-toc-links` | PDF | Supported |
| `--disable-dotted-lines` | PDF | Supported |
| `--toc-level-indentation` | PDF | Supported |
| `--toc-forward-links` | PDF | Supported (`TestTOC`) |
| `--toc-back-links` | PDF | Supported (`TestTOC`) |

### 7.9 Image mode (`gowkhtmltoimage`)

| Flag | Mode | Status |
|------|------|--------|
| `--width`, `--height` | Image | Supported (viewport/min canvas; `TestImageFlags`) |
| `--crop-x`, `--crop-y`, `--crop-w`, `--crop-h` | Image | Supported (output crop) |
| `--format` | Image | Supported (explicit png/jpg vs output-extension sniffing) |
| `--quality` | Image | Supported (JPEG encoder) |
| `--transparent` | Image | Supported (PNG alpha; JPEG falls back to white with a warning) |
| `--smart-width`, `--no-smart-width` | Image | Supported (grow-to-fit viewport) |

Short flags: `-q` quiet, `-g` grayscale, `-O` orientation, `-s` page-size,
`-T`/`-B`/`-R` margins, `-c` copies, `-t` title. **`-L` is `--license`**,
not margin-left. `-h` help, `-V` version, `-E` extended-help.

---

## Amendment process

Any change to this matrix = a plan amendment (Phase 0.5 review), recorded in
`plans/0.1.0/00-canonical-pure-go-rewrite.md`. Phase 4 closure goldens must map to
this matrix row-by-row.
