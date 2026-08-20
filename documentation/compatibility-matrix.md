# gowkhtmltopdf - HTML/CSS Compatibility Matrix (MVP Allowlist)

> **Parent:** `plans/0.1.0/00-canonical-pure-go-rewrite.md` (Phase 0.1); post-MVP updates under `plans/0.2.0/10-canonical-post-mvp-roadmap.md`  
> **Status:** living contract - amendments go through plan review  
> **Target:** authored HTML templates → PDF. **Not** a browser.  
> **Last honesty audit:** 2026-08-13 · fidelity guide: [fidelity.md](fidelity.md)  
> **Phase 21 note:** arbitrary-website / “decent print” work does **not** expand this matrix. CSS remains a **print CSS subset** (Partial flex/grid/position; many properties Not implemented). No new Implemented rows until code + tests ship — see [fidelity.md § Arbitrary websites](fidelity.md#arbitrary-websites-phase-21).

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

Status legend (verified against `internal/layout/style.go` `applyRestProps` +
`uaRules`, `internal/css/css.go`, `internal/layout/layout.go`, `inline.go`,
`paint.go`, and the tests in `internal/layout/layout_test.go` /
`internal/convert/golden_test.go` as of Phase 4):

- **Implemented** - parsed and consumed by layout.
- **Partial** - parsed and used in a subset of the declared cases; the rest
  degrades silently.
- **Not implemented** - parsed and dropped, or not parsed at all; the
  declaration is ignored (graceful, per the contract).

### 2.1 Box model

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `margin` / `margin-top\|right\|bottom\|left` | Implemented | `style.go:340-349` (`setFour`, `marginLen`); sibling margin collapsing `layout.go:263`; tests `TestBlockWidthsAndMargins`, `TestMarginCollapse` |
| `padding` / `padding-top\|right\|bottom\|left` | Implemented | `style.go:350-359`; test `TestPaddingBorderBox` |
| `border` / `border-top\|right\|bottom\|left` | Implemented | `style.go:360-379` (`parseBorder` `style.go:521`); styles `solid\|dashed\|dotted\|none`, width + color; test `TestPaddingBorderBox` |
| `border-width`, `border-style`, `border-color` | Implemented | `style.go:380-401`; `thin\|medium\|thick` widths `style.go:546` |
| `width`, `height` | Implemented | `style.go:316-323`; consumed in `layout.go:176-191` (block) and `layout.go:315-320` (images) |
| `min-width`, `min-height`, `max-width`, `max-height` | Implemented | `style.go:324-339`; enforced `layout.go:181-186, 321-328`; `%` resolves against viewport approximation |
| `box-sizing` (`content-box\|border-box`) | Implemented | parsed `style.go`; default `content-box` (specified width is content width); `border-box` makes width include padding+border (`layout.go` `buildBlock`); test `TestBoxSizingBorderBox` |
| `overflow` (`visible\|hidden\|auto\|scroll\|clip`) | Partial | Parsed for sticky scrollport selection (`style.go`). **Not** general paint clipping. Sticky inside `auto\|scroll\|hidden\|clip` clamps to that box at scroll offset 0 (`sticky.go`) |

### 2.2 Display & flow

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `display` (`block\|inline\|none\|list-item\|table\|table-row\|table-cell\|table-row-group\|table-header-group\|table-footer-group\|flex\|inline-flex\|grid\|inline-grid`) | Implemented / Partial | Core display values Implemented; `flex`/`inline-flex`/`grid`/`inline-grid` accepted in `style.go` and routed to Stage A flex / Stage B grid lite (§2.7 / §2.8). `none` test `TestDisplayNone`; tables `TestTableLayout`; flex/grid fixtures 25/28/32 |
| `display: inline-block` | Implemented (lite) | atomic inline box with width/height/margins; shrink-to-fit when width auto; test `TestInlineBlockBesideText` |
| `display: table-caption` | Implemented | `<caption>` and `display: table-caption` render **above** the table (`buildTableCaption`) |
| `display: table-column`, `table-column-group` | Not implemented | parsed; no column model in `buildTable` |
| `float` (`left\|right`) | Implemented (lite) | out-of-flow pack to side; stacks on same side; simple exclusion for following in-flow content; float inside `td` packs in cell BFC; in-flow `table` always clears below floats (no shrink-beside); `float` on `table-cell`/`table-row` blockifies (CSS2.1 §9.7); tests `TestFloatLeftRightClear`, `TestFloatInsideTableCell`, `TestTableClearsFloat`, fixture-22 / 29 / 38 |
| `clear` (`left\|right\|both`) | Implemented (lite) | advances past named float bottoms (`float.go`); test `TestFloatLeftRightClear` |
| `position` (`static\|relative\|absolute\|fixed\|sticky`) | Partial | static in-flow; `relative`/`absolute`/`fixed` lite via `buildAbsolute` / `buildFixed` / `applyRelativeOffset` (fixtures 26/28). `sticky` = print-scoped clamp (page content box = scrollport; `sticky.go`, fixture-31, `TestSticky*`) plus overflow-box scrollport at offset 0 |
| `position: sticky` | Partial (print + overflow@0) | Default scrollport = page content box (`contentH`); clamps `top`/`bottom`/`left`/`right` within the containing block; natural fragment only, with no fixed-style continuation-page clones. Inside `overflow:auto\|scroll\|hidden\|clip`, that box is the scrollport at **scroll offset 0** (PDF has no scroll; no page clones). **Not** `position:fixed`. Path: `sticky.go` / `applyStickyPrint`; fixture-31; `TestSticky*` / `TestStickyOverflow*` |

### 2.3 Text & fonts

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `font-family` (named + generic) | Partial | parsed + inherited; `FontResolver`: exact registry/`@font-face`/`--font-path` match per token, then author stack, then CSS generics (`serif`/`sans-serif`/`monospace`/`system-ui`) → Liberation/DejaVu, then Liberation Sans terminal default. Legacy names (Georgia/Arial/…) are **not** aliased to Liberation unless supplied. See [fonts.md](fonts.md) |
| `writing-mode` (`horizontal-tb\|vertical-rl\|vertical-lr`) | Partial (horizontal only) | Parsed; `vertical-rl` / `vertical-lr` still lay out as a normal horizontal block. Full CSS vertical typesetting is out of scope |
| `font-size` | Implemented | `style.go` `fontSize` (px/pt/em/%/rem/in/cm/mm/pc + keywords); `%`/`em` resolve against parent; test `TestFontSizeEmInherit` |
| `font-weight` (`normal\|bold\|100-900`) | Implemented | ≥700 selects Liberation Sans **Bold** (or BoldItalic); fake stroke bold only if a bold face is missing; tests `TestRealBoldFaceOps`, `TestBoldFaceInInvoicePDF` |
| `font-style` (`italic\|oblique`) | Implemented | selects Liberation Sans Italic / BoldItalic (`pdf.FaceSet.Resolve`); test `TestRealBoldFaceOps` |
| `text-align` (`left\|right\|center\|justify`) | Implemented (justify lite) | left/right/center; `justify` distributes leftover space between word items on non-final lines (`inline.go`); test `TestTextAlignJustify` |
| `text-decoration` (`none\|underline\|line-through`) | Implemented | drawn in `inline.go`; test `TestBoldUnderline` |
| `text-indent` | Not implemented | parsed, never consumed |
| `line-height` (number, length, `normal`) | Implemented | consumed in line metrics; test `TestMarginCollapse` |
| `letter-spacing` | Implemented | consumed in run width |
| `word-spacing` | Not implemented | absent from `applyRestProps` |
| `text-transform` | Implemented | `none` \| `uppercase` \| `lowercase` \| `capitalize` (`setTextTransformValue`; applied at measure and paint) |
| `vertical-align` (`baseline\|top\|middle\|bottom`) | Partial | table cells: top/middle/bottom offset within row (`emitCell`); inline replaced: top/middle/bottom vs baseline; test `TestTableCellVerticalAlignMiddle` |
| `white-space` (`normal\|nowrap\|pre\|pre-wrap`) | Partial | normal/nowrap/pre implemented; `pre-wrap`/`pre-line` collapse to `pre`; test `TestWhiteSpacePre` |

### 2.4 Color & background

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `color` | Implemented | `style.go:402-405`; consumed `inline.go:152-155`; test `TestCascadeAndInline` |
| `background-color` | Implemented | `style.go:406-409`; painted `layout.go:234-237, 504-507, 531-534` (gated by `Background`); tests `TestBackgroundFill`, `TestRunPDFStyleTableImage` |
| `background` (shorthand) | Partial | Color token only (`firstBackgroundColor`); `url(...)` / gradients ignored |
| `opacity` | Partial | Parsed in `applyRestProps`; paint via PDF ExtGState (`SetOpacity`). Nested opacities multiply. Also accepts `filter: opacity()`; other filter functions ignored (permanent print non-goal for blur/shadow). |

### 2.5 Table subset

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `border-collapse` (`collapse\|separate`) | Partial | `collapse` ≈ `border-spacing: 0` plus a collapsed grid emitter (`emitCollapsedRowGrid`); not a full CSS collapse engine |
| `border-spacing` | Implemented | `style.go`; used by `tableSpacing` (suppressed when collapse); test `TestBorderSpacing` |
| `caption-side` | Not implemented | not consumed; captions always paint above the table |
| `table-layout` (`auto\|fixed`) | Not implemented (auto only) | parsed (`style.go:452-455`), never consumed |

### 2.6 Print / paged media

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `page-break-before/after/inside` (`auto\|always\|avoid`) | Implemented (print pipeline) | parsed into `style.PageBreak*`; honored as canvas-Y flow shifts by the phase-5 paginator - `beforeAlways` `paint.go:203`, `afterBreaks` `paint.go:236`, `avoidInside` `paint.go:179`; tests `TestPageBreakParsing`, `TestPageBreakBeforeAlways`, `TestPageBreakInsideAvoid` |
| `orphans`, `widows` | Partial | CSS `orphans` / `widows` **parsed** (integer ≥1, inherit, initial 2) in `applyRestProps`; Fragmentation Rule 3 enforced when line boxes are countable (`paint.go` `orphansWidows`). Geometric short-block **heuristic** remains as fallback when line counts are unavailable (fixture-30). See fixture-37. |

### Feature checklist (page geometry, tables, pagination)

| Feature | Status | Notes / verified by |
|---------|--------|---------------------|
| Page size (A4/Letter/…), landscape, margins (mm) | Implemented | `settings.ParsePageSize` (`settings/pagesize.go:39`), `convert.pageGeometry` (`convert.go:138`) |
| `colspan` | Yes | `colSpan`; test `TestTableColspan` |
| `rowspan` | Yes / Implemented | column occupancy + height growth (`placeTableCells`, `growRowspanRows`); tests `TestTableRowspan*` |
| `border-collapse` | Partial | see §2.5 |
| Pagination | Fragment + whole-op + phase-18 polish | rect-type ops (fill/stroke/line) split at page boundaries; text/images/links move wholly (line-level) (`paint.go`); `page-break-before/after: always`, `page-break-inside: avoid`, table rows never split; **`<thead>` / `table-header-group` repeat** on continuation pages (`repeatTableHeaders`, fixture-23); CSS `orphans`/`widows` parsed + Rule 3 when line boxes exist (heuristic fallback; fixtures 30/37); `--zoom` forwarded; smart-shrinking re-layouts. `Result.Locations` for outlines/links. See "Pagination" note below. |
| Floats / absolute positioning | Float lite + absolute/fixed/sticky lite | float/`clear` lite (§2.2); relative/absolute/fixed lite; sticky = print page scrollport + overflow@0 (§2.2; fixture-31) |
| Flexbox / Grid | Partial | Stage A flex + Stage B grid (areas/dense/`minmax`) + Stage C lite — §2.7 / §2.8. Paths: `flex.go`, `grid.go`, `style.go`; fixtures 25/28/32–35; plan `plans/0.2.0/phases/subplans-tier-2/flex-grid-full.md`. **Not** Bootstrap/Tailwind / Chrome layout-test parity |
| Multicol | Partial | Report lite: `column-count`/`column-width`/`columns`, `column-gap` (normal→1em), `column-span:none\|all`, `column-fill:balance\|auto`; column boxes do not straddle pages — §2.9; `multicol.go`; fixture-39 |
| Transforms (static 2D) | Partial | `transform` + `transform-origin` paint CTM; stacking + abs/fixed CB; sibling flow unchanged. No animation timelines; no 3D; `filter` only `opacity()`. Fixture-40; `transform.go` |
| JavaScript | No | `<script>` stripped at load; no engine. `--enable-javascript` is an **unknown option** (Policy A) |
| Image-mode text | TTF outline raster | same Liberation faces as PDF; pure-Go coverage AA (`internal/imageout/ttfraster.go`); 5×7 bitmap only if an op has no font |

**Pagination (phases 5 + 18).** Box-aware fragmentation: rect-type ops crossing a page boundary are split; text, images and links move wholly (line-level). `page-break-before/after: always` and `page-break-inside: avoid` via canvas-Y flow shifts; table rows never split. **Table headers repeat** across pages (`repeatTableHeaders` in `paint.go`; fixture-23). **`--zoom`** is forwarded to `layout.Options.Zoom` (`convert.go`; `TestZoom`). **Smart-shrinking** detects over-wide content and **re-layouts** with an effective zoom (`TestRunPDFSmartShrinking`). CSS `orphans`/`widows` are parsed (initial 2) and Fragmentation Rule 3 is applied when line boxes are available; the geometric short-block heuristic remains for edge cases (fixtures 30/37). `Result.Locations` carries element boxes for outlines/links.

### 2.7 Flexbox (Stage A — print CSS subset)

Evidence: `internal/layout/flex.go`, `style.go` (`applyRestProps` + `parseFlexShorthand`); fixtures 25/28/32/33; `flex_test.go`. Status uses the §2 legend (Implemented / Partial / Not implemented). Checklist form: **[x]** Implemented · **[~]** Partial · **[ ]** Missing.

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `display: flex` / `inline-flex` | [x] Implemented | Routed to `buildFlex`; fixtures 25/28/32 |
| `flex-direction` (`row` \| `column` \| `row-reverse` \| `column-reverse`) | [x] Implemented | `style.go`; row/column + reverse paths in `flex.go`; `TestFlexRowReverse` |
| `flex-wrap` (`nowrap` \| `wrap` \| `wrap-reverse`) | [x] Implemented | Multi-line wrap; wrap-reverse reverses line order |
| `justify-content` (`flex-start` \| `flex-end` \| `center` \| `space-between` \| `space-around` \| `space-evenly`) | [x] Implemented | Row + column (definite height); `TestFlexSpaceEvenly` |
| `align-items` (`stretch` \| `flex-start` \| `center` \| `flex-end`) | [x] Implemented | Row cross-axis stretch sizes auto-height items to the flex line (fixture-33); start/center/end honored; `TestFlexAlignItemsStretchRow` |
| `align-self` | [x] Implemented | Overrides container; stretch follows same rules; `TestFlexAlignSelf` |
| `align-content` (multi-line) | [~] Partial | Distributes free cross space when container height is definite and wrap produced ≥2 lines; `stretch` = pack at start |
| `gap` / `row-gap` / `column-gap` | [x] Implemented | Independent longhands; shorthand fills both when longhands unset (`flexGaps`) |
| `flex` shorthand (`none` \| `auto` \| grow/shrink/basis) | [x] Implemented | `parseFlexShorthand` |
| `flex-grow` / `flex-shrink` / `flex-basis` | [x] Implemented | Length/%/auto basis; post grow/shrink min/max-width clamp; column grow/shrink when height definite |
| Content-based min-size floor | [x] Partial polish | `flexMinMainSize` / `flexClampMainWidths` — content-based `min-width:auto` + `%` min re-resolve on definite containers; overflow non-visible → auto min 0 (Flexbox §4.5 lite). Deep multi-pass intrinsic still out |
| `order` | [x] Implemented | Stable sort before place |
| Percentage basis cyclic sizing | [x] Partial | Definite CB: `%` vs content main size; indefinite/cyclic → treat as `auto` (content) — fixture-33 / `TestFlexBasisPercent*` |
| Nested percentage / intrinsic flex iterations | [x] Partial polish | Definite-item `%` children re-resolve against used main size; indefinite CB `%` → auto/content; not full Flexbox intrinsic passes |

### 2.8 CSS Grid (Stage B + Stage C lite — print CSS subset)

Evidence: `internal/layout/grid.go`, `style.go`; fixtures 28/32/34/35; `grid_test.go`. **Not** full Grid L1 / L3 / Chrome parity.

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `display: grid` / `inline-grid` | [x] Implemented | Routed to `buildGrid`; nested grids OK |
| `display: subgrid` | [~] Partial | `inheritSubgridFromParent` copy-inherits parent template columns (and unspecified gaps); tracks re-resolve against the subgrid's own content box. Joint Resolve Intrinsic / full subgrid L1 out of scope |
| `grid-template-columns` | [x] Implemented | Lengths, `fr`, `repeat(N, …)`, `minmax(...)`; gap subtracted before `fr` distribute |
| `grid-template-rows` | [x] Implemented | Consumed when height definite; fixed mins on auto-height; fixture-32 |
| `minmax()` track sizing | [x] Implemented | Lengths / `%` (definite) / `fr` / `auto` / `min-content` / `max-content` subset; `fr` keeps min floors — fixture-35 |
| `gap` / `row-gap` / `column-gap` | [x] Implemented | Independent (`gridGaps`); `TestGridIndependentGaps` |
| `grid-column` / `grid-column-start` / `grid-column-end` / `span N` | [x] Implemented | Line numbers + span; 2D occupancy |
| `grid-row` / `grid-row-start` / `grid-row-end` / `span N` | [x] Implemented | Row span + stretch into spanned tracks; `TestGridRowSpan*` |
| Auto-flow placement (row / column) | [x] Implemented | Sparse row default; column major via `grid-auto-flow: column` |
| `grid-auto-flow: dense` | [x] Implemented | `dense` / `row dense` / `column dense` hole-fill — fixture-34 |
| `grid-template-areas` / `grid-area` names | [x] Implemented | Named areas + lite line form; areas can extend auto tracks — fixture-34 |
| `justify-items` / `align-items` | [x] Implemented | Default stretch; start/center/end; stretch sizes item to grid area |
| `justify-self` / `align-self` | [x] Implemented | Overrides container; stretch fills area |
| Masonry | [~] Partial | `grid-template-rows: masonry` packs items into the shortest column (`emitMasonryItems`). Not full CSS Grid L3 masonry / Chrome parity |
| Intrinsic / nested % track cycles | [x] Partial | Measure-pass lite for min/max-content track mins; cyclic `%` → auto when CB indefinite |

### 2.9 CSS Multi-column (report lite)

Evidence: `internal/layout/multicol.go`, `style.go` (`applyRestProps`); fixture-39; `multicol_test.go`. **Not** full Multicol L1 / L2 / Chrome balancing with floats.

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `column-count` (`auto` \| integer ≥1) | [x] Implemented | Establishes multicol when ≠ auto; `TestMulticolParseProps` |
| `column-width` (`auto` \| `<length>`) | [x] Implemented | Used count/width per Multicol §3.3; `TestUsedColumnCountWidth` |
| `columns` shorthand | [x] Implemented | `parseColumnsShorthand` |
| `column-gap` (`normal` \| `<length>`) | [x] Implemented | Multicol: `normal` → 1em; flex/grid still treat unset/normal as 0 gap |
| `column-span` (`none` \| `all`) | [x] Implemented | Mid-flow spanner; preceding columns balance — fixture-39 / `TestMulticolColumnSpanAll` |
| `column-fill` (`balance` \| `auto`) | [x] Implemented | Balance packs to equal stacks; auto fills to page/definite height |
| Column box pagination | [x] Implemented | Column boxes do not cross page boundaries; new multicol line on next page — `TestMulticolLinesDoNotStraddlePages` |
| `break-*: column \| avoid-column` | [~] Partial | Aliased to page `always`/`avoid` (starts new multicol line via page break) |
| `column-rule*`, L2 integer spans, overflow columns | [ ] Missing | Deferred (see `plans/0.2.0/phases/tier-2-pending-3/multicol.md` out of scope) |

## 3. Supported units

Status legend as in §2; resolution sites: `fontSize` `style.go:561`,
`lengthBox` `style.go:634` (width/height/min/max), `marginLen`
`style.go:671` (margins/padding/letter-spacing), parse gate `css.go:706`.

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
| `ex`, `ch` | Not implemented | accepted by the parser (`css.go:731`) but never resolved by `style.go` - declaration dropped |
| `vmin`, `vmax`, `dpi`-style | Not implemented | rejected at parse |

## 4. Supported selector syntax (cascade)

Status legend as in §2; evidence in `internal/css/css.go`.

| Selector | Status | Notes / verified by |
|----------|--------|---------------------|
| Element (`h1`, `p`, …), class (`.foo`), ID (`#bar`) | Implemented | `parseCompound` `css.go:444`; matching `Match` `css.go:518`; test `css_test.go::TestMatch` |
| Universal (`*`) | Implemented | `css.go:456-459` |
| Descendant (`div p`), child (`ul > li`) | Implemented | combinators `css.go:356-362`; matching `css.go:528-543` |
| Sibling (`a + b`, `a ~ b`) | Implemented | next-sibling `+` and subsequent-sibling `~` (`css.Match`); test `TestSiblingCombinators` |
| Attribute (`[href]`, `[href="…"]`) | Partial | presence, exact `=`, word `~=`, substring `*=`, prefix `^=`, suffix `$=`, dash `|=`; `TestAttrWordAndSubstring`, `TestAttrPrefixSuffixDash` |
| `:first-child`, `:last-child`, `:nth-child(n)` | Implemented | `odd`/`even`/`an+b`/integer; tests `TestMatch`, `TestNthChildZebraSheet` |
| `:link`, `:visited` | Partial | Print semantics: match any `a` with non-empty `href` (no visit history; `:visited` ≡ `:link`). Specificity counts as a class-level pseudo. Proof: `TestLinkVisitedPseudos`, `TestLinkPseudoColor` |
| `:hover`, `:active`, `:focus` | Not implemented (accepted, never match) | Parsed onto the compound but `matchPseudo` returns false so `a:hover` does not degrade to bare `a` |
| `::before` / `::after` | Partial / Implemented | `MatchPseudo` plus generated content (`pseudo_content.go`): quoted strings and `attr()`. Host-element rules do not apply to the host |
| `!important` | Implemented | `css.go:664-688`; separate cascade layer `style.go:221-247`; test `css_test.go::TestParseImportant` |
| Specificity (ID > class > element), inline `style` wins, `!important` overrides | Implemented | `Specificity` `css.go:578`; inline style priority `style.go:233-239`; test `css_test.go::TestSpecificity` |
| `@media print` / `screen` filtering | Implemented | `MediaMatches` (`css/media.go`); cascade `style.go`; convert `Media: "print"` (PDF) or `"screen"` (Image); only `print` and `screen` are evaluated (all other media types and unsupported feature queries evaluate to false); tests `TestParseMedia`, `TestMediaMatches*` |
| `@media` feature queries (`(min-width: …)`) | Partial | size features + orientation vs viewport; unknown features → false; `TestMediaMatchesSizeFeatures` |
| `:has()` | Partial | Relative selectors inside `:has(...)`; descendant/child/sibling + simple compounds; no forgiving-selector list / complex chrome edge cases. `has.go`; fixture-41 |
| `@container` / `container-type` | Partial | Size queries only (`inline-size`/`width` + `and`/`or`/`not`); named containers; two-pass style after used inline size. No style/scroll-state queries; no `cq*` units. `container.go`; fixture-42 |
| `@page` | Implemented | `@page { margin }` and `@page { size }` applied to page geometry (`applyCSSPageMargins`) |
| `@font-face` | Partial | Parsed; `MergeFontFaces` loads TTF/OTF/WOFF1 via `FetchSub` (local **and** `https://`) under the same ACL + `NetworkPolicy` on PDF and image paths. `.woff2` / `.eot` / `data:` skipped. See §5 |

## 5. Explicitly unsupported (MVP)

| Feature | Handling |
|---------|----------|
| JavaScript / `<script>` / DOM APIs | **Stripped at load.** No JS engine. `--enable-javascript` and other JS flags are **unknown options** (Policy A) |
| Full CSS Grid / full Flexbox | Stage A/B print CSS subset **shipped** (§2.7 / §2.8); Stage C lite + flex min-size polish + Partial subgrid/masonry span (`tier-2-pending-3/flex-grid-remaining.md`). **Not** Bootstrap/Tailwind / Chrome layout-test parity |
| `position: sticky` continuous scroll | Overflow boxes are sticky scrollports at **offset 0** only (PDF has no scroll). Page content box remains the print scrollport when no overflow ancestor. No scroll-offset > 0 animation |
| `transform`, `filter`, `animation`, `transition` | Partial / out of scope | **Static 2D** `transform` + `transform-origin` Implemented (translate/scale/rotate/matrix/skew*; paint CTM; stacking + abs/fixed CB). Sibling flow unchanged. Overflow: no clip — ink may paint outside page box. **`filter`:** only `opacity()` (see `opacity` row); blur/drop-shadow/SVG filters = permanent print-engine non-goal. **`animation`/`transition`/`@keyframes`:** parse-ignored (static cascaded value only; no timelines). **3D / perspective:** permanent non-goal. Fixture-40; `transform.go` / `transform_test.go` |
| `background-image` / gradients | Ignored (Phase 3+ candidate) |
| `@font-face` (remote / WOFF2) | **Partial:** local **and `https://`** TTF/OTF/WOFF1/WOFF2 via `FetchSub` (same ACL + `NetworkPolicy`) on PDF/image paths. **`.eot` / `data:`** skipped. Missing faces fall back to registry / Liberation |
| Custom XSLT TOC (`--xsl-style-sheet`) | Not implemented (no XSLT in stdlib); Go templates instead (Phase 6) |
| SVG-as-`<img>` | **Implemented** (raster via `internal/svg`) |
| WebP, AVIF | Not implemented; broken-image placeholder or skip |
| Fixed CSS headers/footers via `position: fixed` alone | Prefer CLI `--header-*` / `--footer-*` for repeating chrome; CSS `fixed` lite paints on every page but is not a full running-element model |
| Complex-script shaping (Indic, Arabic, CJK) | **Type0/CID Identity-H** for BMP Unicode (CJK with a capable face); **Arabic OT** via `go-text/typesetting` when the face has GSUB (+ presentation-form `ShapeText` fallback); Hangul needs a Hangul face. `writing-mode` vertical keywords are parsed; **layout stays horizontal**. **Indic Partial** (OT when face/cmap allow; not production-claimed). Optional OT **`halt`/`palt`** for CJK punctuation via `ShapeTextFont` FontFeatures |
| PDF version (1.4 / 1.7 / 2.0) | **Supported:** PDF 1.4 is default; PDF 1.7 and PDF 2.0 are opt-in via `--pdf-version 1.7` / `--pdf-version 2.0` or library API `WithPDFVersion`. Emits `%PDF-1.7` (trailer `/ID`, Info with UTF-16BE + BOM strings, non-claiming XMP Metadata stream) or `%PDF-2.0` (trailer `/ID`, UTF-8 document strings, non-claiming XMP with `dc:format`, `pdf:Producer`, dates). PDF 2.0 output is a **version**, not a conformance claim |
| PDF/A-3a, PDF/UA-1 (ISO 19005-3 / ISO 14289-1) | **Supported:** Opt-in via `--pdf-profile a3a-ua1` / `WithPDFProfile("a3a-ua1")`, `"a3a"`, `"ua1"`. Implies PDF 1.7. Emits claiming XMP metadata (`pdfaid:part=3`, `pdfaid:conformance=A`, `pdfuaid:part=1`), sRGB OutputIntent, `/DefaultRGB`, and full logical structure tree (`H1`..`H6`, `P`, `Table` > `TR` > `TH`/`TD`, `L` > `LI` > `LBody` > `Link`, `Figure` + `alt`, `/Artifact /Pagination`) |
| PDF 2.0 (ISO 32000-2) | **Shipped as opt-in version** (#32): `--pdf-version 2.0` / `WithPDFVersion("2.0")`. Version alone is **not** a PDF/A or PDF/UA claim |
| PDF/A-4, PDF/UA-2 (PDF 2.0 conformance profiles) | **Supported** (#33): Opt-in via `--pdf-profile a4-ua2` / `WithPDFProfile("a4-ua2")`, `"a4"`, `"ua2"`. Implies PDF 2.0. Emits claiming XMP (`pdfaid:part=4`, `pdfaid:rev=2020`, `pdfuaid:part=2`, `pdfuaid:rev=2024`), sRGB+Gray OutputIntent / Default* ICCBased, structure `/Namespace`, `ListNumbering` on lists, structure destinations on internal links, and full logical structure tree (`L` > `LI` > `LBody` > `Link`) |
| PDF encryption, duplex, AcroForm | Out of scope (not in original wkhtmltopdf either) |

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
| `--internal-links`, `--no-internal-links` | PDF | Partial (body `#` fragment GoTo via layout `OpLinkURI` + `applyInternalLinks`; HTML HF `#id` → body `AddLinkDest`. Geometry caveats — runs without paint boxes still skipped) |
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
| `--header-font-name`, `--footer-font-name` | PDF | Partial (`resolveHFFont` uses the shared registry / `FontResolver` when the named family is available; otherwise Liberation Sans) |
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
