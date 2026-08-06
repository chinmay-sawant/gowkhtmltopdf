# gowkhtmltopdf - HTML/CSS Compatibility Matrix (MVP Allowlist)

> **Parent:** `plans/00-canonical-pure-go-rewrite.md` (Phase 0.1); post-MVP updates under `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** living contract - amendments go through plan review  
> **Target:** controlled report/invoice HTML → PDF. **Not** a browser.  
> **Last honesty audit:** 2026-08-05 · base commit `f5bb754` · Tier 2 phases 17–20 core on master (#16/#17) · fidelity guide: [fidelity.md](fidelity.md)  
> **Phase 21 note:** arbitrary-website / “decent print” work does **not** expand this matrix. CSS remains a **report subset** (Partial flex/grid/position; many properties Not implemented). No new Implemented rows until code + tests ship — see [fidelity.md § Arbitrary websites](fidelity.md#arbitrary-websites-phase-21).

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
| `ul`, `ol`, `li` | Simple lists: bullet markers. `ol` renders bullets too (decimal markers not implemented; always `•`, `layout.go:220`) |
| `table`, `thead`, `tbody`, `tfoot`, `tr`, `th`, `td` | Table subset; see §4/§2.5 (colspan yes, rowspan no) |
| `img` | Replaced element; **PNG/JPEG/SVG subset**. JPEG is DCTDecode pass-through. PNG decoded to DeviceRGB; alpha soft-mask when present. **SVG** rasterized via `internal/svg` (rect/circle/path subset → PNG). Layout uses a fixed 96 dpi CSS px→pt map. `web.images=false` skips fetch/paint. |
| `a` | Hyperlink (`href`) for `http/https/mailto` external URI annotations; body `#id` / `#name` **GoTo** via `applyInternalLinks` (fixture-24). HTML header/footer **external URI** and **fragment GoTo** (`#id` → body destinations via `AddLinkDest`, copies-aware) are carried onto body pages |
| `strong`, `em`, `b`, `i`, `u`, `small` | `b`/`strong` → bold face; `em`/`i` → italic face (Liberation family, §2.3); `u` underline; `small` smaller; fake stroke bold only if a bold face is missing |
| `pre`, `code` | `pre` honors `white-space: pre`; `code` has no monospace rule - single embedded font for all families (§2.3) |
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
| `display: table-caption`, `table-column(-group)` | Not implemented | parsed; no caption/column model in `buildTable` - `<caption>` does not render |
| `float` (`left\|right`) | Implemented (lite) | out-of-flow pack to side; stacks on same side; simple exclusion for following in-flow content; float inside `td` packs in cell BFC; in-flow `table` always clears below floats (no shrink-beside); `float` on `table-cell`/`table-row` blockifies (CSS2.1 §9.7); tests `TestFloatLeftRightClear`, `TestFloatInsideTableCell`, `TestTableClearsFloat`, fixture-22 / 29 / 38 |
| `clear` (`left\|right\|both`) | Implemented (lite) | advances past named float bottoms (`float.go`); test `TestFloatLeftRightClear` |
| `position` (`static\|relative\|absolute\|fixed\|sticky`) | Partial | static in-flow; `relative`/`absolute`/`fixed` lite via `buildAbsolute` / `buildFixed` / `applyRelativeOffset` (fixtures 26/28). `sticky` = print-scoped clamp (page content box = scrollport; `sticky.go`, fixture-31, `TestSticky*`) plus overflow-box scrollport at offset 0 |
| `position: sticky` | Partial (print + overflow@0) | Default scrollport = page content box (`contentH`); clamps `top`/`bottom`/`left`/`right` within containing block; continuation-page clones where CB intersects. Inside `overflow:auto\|scroll\|hidden\|clip`, that box is the scrollport at **scroll offset 0** (PDF has no scroll; no page clones). **Not** `position:fixed`. Path: `sticky.go` / `applyStickyPrint`; fixture-31; `TestSticky*` / `TestStickyOverflow*` |

### 2.3 Text & fonts

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `font-family` (named + generic) | Partial | parsed + inherited; embedded Liberation Sans family (R/B/I/BI) plus **font registry** (`--font-path`, optional `--use-system-fonts`) and local `@font-face` TTF/OTF/WOFF1 on **PDF and image** paths (see §4 / §5). Named families resolve as named; missing faces fall through the author’s comma stack, then Liberation; only CSS generics (`serif`/`sans-serif`/`monospace`) expand to Liberation |
| `writing-mode` (`horizontal-tb\|vertical-rl\|vertical-lr`) | Partial (horizontal only) | Parsed; `vertical-rl`/`vertical-lr` lay out as normal horizontal block (vertical lite paint removed for leanness — ponytail Phase 4). Full CSS vertical typesetting is out of scope |
| `font-size` | Implemented | `style.go` `fontSize` (px/pt/em/%/rem/in/cm/mm/pc + keywords); `%`/`em` resolve against parent; test `TestFontSizeEmInherit` |
| `font-weight` (`normal\|bold\|100-900`) | Implemented | ≥700 selects Liberation Sans **Bold** (or BoldItalic); fake stroke bold only if a bold face is missing; tests `TestRealBoldFaceOps`, `TestBoldFaceInInvoicePDF` |
| `font-style` (`italic\|oblique`) | Implemented | selects Liberation Sans Italic / BoldItalic (`pdf.FaceSet.Resolve`); test `TestRealBoldFaceOps` |
| `text-align` (`left\|right\|center\|justify`) | Implemented (justify lite) | left/right/center; `justify` distributes leftover space between word items on non-final lines (`inline.go`); test `TestTextAlignJustify` |
| `text-decoration` (`none\|underline\|line-through`) | Implemented | drawn in `inline.go`; test `TestBoldUnderline` |
| `text-indent` | Not implemented | parsed, never consumed |
| `line-height` (number, length, `normal`) | Implemented | consumed in line metrics; test `TestMarginCollapse` |
| `letter-spacing` | Implemented | consumed in run width |
| `word-spacing` | Not implemented | absent from `applyRestProps` |
| `text-transform` | Not implemented | absent from `applyRestProps` |
| `vertical-align` (`baseline\|top\|middle\|bottom`) | Partial | table cells: top/middle/bottom offset within row (`emitCell`); inline replaced: top/middle/bottom vs baseline; test `TestTableCellVerticalAlignMiddle` |
| `white-space` (`normal\|nowrap\|pre\|pre-wrap`) | Partial | normal/nowrap/pre implemented; `pre-wrap`/`pre-line` collapse to `pre`; test `TestWhiteSpacePre` |

### 2.4 Color & background

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `color` | Implemented | `style.go:402-405`; consumed `inline.go:152-155`; test `TestCascadeAndInline` |
| `background-color` | Implemented | `style.go:406-409`; painted `layout.go:234-237, 504-507, 531-534` (gated by `Background`); tests `TestBackgroundFill`, `TestRunPDFStyleTableImage` |
| `background` (shorthand) | Not implemented | absent from `applyRestProps` (only `background-color`) |
| `opacity` | Partial | Parsed in `applyRestProps`; paint via PDF ExtGState (`SetOpacity`). Nested opacities multiply. Also accepts `filter: opacity()`; other filter functions ignored (permanent print non-goal for blur/shadow). |

### 2.5 Table subset

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `border-collapse` (`collapse\|separate`) | Not implemented (separate only) | parsed (`style.go:446-449`), never consumed - `buildTable` always uses the separate model with `BorderSpacing` (`layout.go:450, 500`) |
| `border-spacing` | Implemented | `style.go:450-451`; used `layout.go:450-455, 500` |
| `caption-side` | Not implemented | absent from `applyRestProps`; `<caption>` elements are not rendered (see `display: table-caption`) |
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
| `colspan` | Yes | `colSpan` `layout.go:618-622`; test `TestTableColspan` |
| `rowspan` | No | attribute ignored - only `colspan` is read |
| `border-collapse` | Separate only | see §2.5 |
| Pagination | Fragment + whole-op + phase-18 polish | rect-type ops (fill/stroke/line) split at page boundaries; text/images/links move wholly (line-level) (`paint.go`); `page-break-before/after: always`, `page-break-inside: avoid`, table rows never split; **`<thead>` / `table-header-group` repeat** on continuation pages (`repeatTableHeaders`, fixture-23); CSS `orphans`/`widows` parsed + Rule 3 when line boxes exist (heuristic fallback; fixtures 30/37); `--zoom` forwarded; smart-shrinking re-layouts. `Result.Locations` for outlines/links. See "Pagination" note below. |
| Floats / absolute positioning | Float lite + absolute/fixed/sticky lite | float/`clear` lite (§2.2); relative/absolute/fixed lite; sticky = print page scrollport + overflow@0 (§2.2; fixture-31) |
| Flexbox / Grid | Partial | Stage A flex + Stage B grid (areas/dense/`minmax`) + Stage C lite — §2.7 / §2.8. Paths: `flex.go`, `grid.go`, `style.go`; fixtures 25/28/32–35; plan `plans/phases/subplans-tier-2/flex-grid-full.md`. **Not** Bootstrap/Tailwind / Chrome layout-test parity |
| Multicol | Partial | Report lite: `column-count`/`column-width`/`columns`, `column-gap` (normal→1em), `column-span:none\|all`, `column-fill:balance\|auto`; column boxes do not straddle pages — §2.9; `multicol.go`; fixture-39 |
| Transforms (static 2D) | Partial | `transform` + `transform-origin` paint CTM; stacking + abs/fixed CB; sibling flow unchanged. No animation timelines; no 3D; `filter` only `opacity()`. Fixture-40; `transform.go` |
| JavaScript | No | stripped at load; `--enable-javascript` accepted + warning (Phase 1) |
| Image-mode text | TTF outline raster | same Liberation faces as PDF; pure-Go coverage AA (`internal/imageout/ttfraster.go`); 5×7 bitmap only if an op has no font |

**Pagination (phases 5 + 18).** Box-aware fragmentation: rect-type ops crossing a page boundary are split; text, images and links move wholly (line-level). `page-break-before/after: always` and `page-break-inside: avoid` via canvas-Y flow shifts; table rows never split. **Table headers repeat** across pages (`repeatTableHeaders` in `paint.go`; fixture-23). **`--zoom`** is forwarded to `layout.Options.Zoom` (`convert.go`; `TestZoom`). **Smart-shrinking** detects over-wide content and **re-layouts** with an effective zoom (`TestRunPDFSmartShrinking`). CSS `orphans`/`widows` are parsed (initial 2) and Fragmentation Rule 3 is applied when line boxes are available; the geometric short-block heuristic remains for edge cases (fixtures 30/37). `Result.Locations` carries element boxes for outlines/links.

### 2.7 Flexbox (Stage A — report subset)

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

### 2.8 CSS Grid (Stage B + Stage C lite — report subset)

Evidence: `internal/layout/grid.go`, `style.go`; fixtures 28/32/34/35; `grid_test.go`. **Not** full Grid L1 / L3 / Chrome parity.

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `display: grid` / `inline-grid` | [x] Implemented | Routed to `buildGrid`; nested grids OK |
| `display: subgrid` | [~] Not implemented | Treated as ordinary `display: grid` (no parent template inherit). Joint Resolve Intrinsic / subgrid L1 out of scope — ponytail Phase 4 cut |
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
| Masonry | [~] Not implemented | Lone `masonry` keyword stripped → dense auto-flow only (no shortest-stack pack). CSS Grid L3 masonry out of scope — ponytail Phase 4 cut |
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
| `column-rule*`, L2 integer spans, overflow columns | [ ] Missing | Deferred (see `plans/phases/tier-2-pending-3/multicol.md` out of scope) |

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
| `::before` / `::after` | Not implemented | dropped with the other pseudo-classes |
| `!important` | Implemented | `css.go:664-688`; separate cascade layer `style.go:221-247`; test `css_test.go::TestParseImportant` |
| Specificity (ID > class > element), inline `style` wins, `!important` overrides | Implemented | `Specificity` `css.go:578`; inline style priority `style.go:233-239`; test `css_test.go::TestSpecificity` |
| `@media print` / `screen` filtering | Implemented | `MediaMatches` (`css/media.go`); cascade `style.go`; convert `Media: "print"`; tests `TestParseMedia`, `TestMediaMatches*` |
| `@media` feature queries (`(min-width: …)`) | Partial | size features + orientation vs viewport; unknown features → false; `TestMediaMatchesSizeFeatures` |
| `:has()` | Partial | Relative selectors inside `:has(...)`; descendant/child/sibling + simple compounds; no forgiving-selector list / complex chrome edge cases. `has.go`; fixture-41 |
| `@container` / `container-type` | Partial | Size queries only (`inline-size`/`width` + `and`/`or`/`not`); named containers; two-pass style after used inline size. No style/scroll-state queries; no `cq*` units. `container.go`; fixture-42 |
| `@page` | Not implemented | `@page` blocks skipped gracefully at parse |
| `@font-face` | Partial | Parsed; `MergeFontFaces` loads **local TTF/OTF/WOFF1** via `FetchSub` ACL for **PDF and image** paths. WOFF2 / remote network `src` skipped by policy. See §5 |

## 5. Explicitly unsupported (MVP)

| Feature | Handling |
|---------|----------|
| JavaScript / `<script>` / DOM APIs | **Stripped at load.** `--enable-javascript` accepted but ignored with warning (Phase 1) |
| Full CSS Grid / full Flexbox | Stage A/B report subset **shipped** (§2.7 / §2.8); Stage C lite + flex min-size polish + Partial subgrid/masonry span (`tier-2-pending-3/flex-grid-remaining.md`). **Not** Bootstrap/Tailwind / Chrome layout-test parity |
| `position: sticky` continuous scroll | Overflow boxes are sticky scrollports at **offset 0** only (PDF has no scroll). Page content box remains the print scrollport when no overflow ancestor. No scroll-offset > 0 animation |
| `transform`, `filter`, `animation`, `transition` | Partial / out of scope | **Static 2D** `transform` + `transform-origin` Implemented (translate/scale/rotate/matrix/skew*; paint CTM; stacking + abs/fixed CB). Sibling flow unchanged. Overflow: no clip — ink may paint outside page box. **`filter`:** only `opacity()` (see `opacity` row); blur/drop-shadow/SVG filters = permanent print-engine non-goal. **`animation`/`transition`/`@keyframes`:** parse-ignored (static cascaded value only; no timelines). **3D / perspective:** permanent non-goal. Fixture-40; `transform.go` / `transform_test.go` |
| `background-image` / gradients | Ignored (Phase 3+ candidate) |
| `@font-face` (remote / WOFF2) | **Partial:** local **and https** TTF/OTF/WOFF1 via `FetchSub` ACL on PDF/image paths. **WOFF2** still skipped (Brotli/WOFF2 decode). **data:** skipped. Missing faces fall back to registry / Liberation |
| Custom XSLT TOC (`--xsl-style-sheet`) | Not implemented (no XSLT in stdlib); Go templates instead (Phase 6) |
| WebP, SVG-as-`img`, AVIF | Not decodable by stdlib; broken-image placeholder or skip |
| Fixed CSS headers/footers via `position: fixed` alone | Prefer CLI `--header-*` / `--footer-*` for repeating chrome; CSS `fixed` lite paints on every page but is not a full running-element model |
| Complex-script shaping (Indic, Arabic, CJK) | **Type0/CID Identity-H** for BMP Unicode (CJK with a capable face); **Arabic OT** via `go-text/typesetting` when the face has GSUB (+ presentation-form `ShapeText` fallback); Hangul needs a Hangul face; **vertical-rl** lite (rotated CJK). **Indic Partial** (OT when face/cmap allow; not production-claimed). Optional OT **`halt`/`palt`** for CJK punctuation via `ShapeTextFont` FontFeatures. No CGO HarfBuzz (pure-Go port inside typesetting only) |
| PDF encryption, PDF/A, duplex, AcroForm | Out of scope (not in original wkhtmltopdf either) |

## 6. Security policy (frozen defaults)

| Rule | Value |
|------|-------|
| Local file access | **Blocked by default** (`--enable-local-file-access` opt-in; `--allow` path allowlist walk) |
| Untrusted HTML | **Not supported** - same warning as upstream `docs/status.md`; use with HTML you control only |
| Remote URL fetch | `net/http` defaults: connect + response timeouts, redirect limit, `blockLocalFileAccess` covers `file://` and localhost refs |
| SSRF posture | No automatic form submission; POST only via explicit `--post` flags; no cookies auto-forwarded from site contexts |

## 7. CLI flag support matrix (Phase 9.1)

Extracted from every `add(...)` call in `internal/cli/flags.go` (98 flags;
each bool flag also accepts `--no-<flag>`). Status is **ground truth, not
intent**: each flag's dotted setting was traced from the `Set` surface
(`internal/settings/reflect.go`) to its consumers in `internal/convert`,
`internal/load`, `internal/imageout` and `internal/layout`. Flags whose
setting has no consumer are marked **Ignored**, even when upstream
wkhtmltopdf honors them.

- **Supported** - parsed, wired into settings, and the setting is consumed
  by the PDF/image pipeline; exercised end-to-end in `internal/convert`
  tests (incl. `TestGoldenCorpusAllFixtures`), `internal/cli` parse tests,
  or the CLI smoke run.
- **Partial** - accepted and stored, but only part of the upstream
  behavior is honored (note column says which).
- **Ignored** - accepted and stored, never consumed; no effect on output
  (mostly `web.*` / `load.*` JS-era settings).
- **Error** - rejected. No flag in the table is rejected at parse, but
  `--<unknown>` → `unknown option`, invalid enum/length values → error, and
  a bogus `--page-size` fails at conversion time. `--bogus-flag` is
  exercised by `TestUnknownFlagErrors`.

### 7.1 Documentation flags

| Flag | Mode | Status |
|------|------|--------|
| `--help`, `--version`, `--license`, `--extended-help` | Both | Supported (handled by the parser; prints and exits 0) |
| `--man`, `--html` | Both | Ignored (parsed, stored on the command, never consumed) |

### 7.2 Global page/PDF flags

| Flag | Mode | Status |
|------|------|--------|
| `--quiet` | Both | Supported (suppresses progress; `TestRunPDFQuiet`) |
| `--log-level` | Both | Ignored (stored; the pipeline has no level filtering) |
| `--collate`, `--copies` | PDF | Supported (`TestRunPDFCopiesCollate`, `TestRunPDFCopiesNonCollate`) |
| `--orientation` | PDF | Supported (page geometry swap) |
| `--page-size` | PDF | Supported (A4/Letter/…; golden runner) |
| `--grayscale` | PDF | Ignored (wired to `colormode`→`ColorMode`, which no pipeline code reads; the PDF grayscale switch is fed by the library-side `grayscale` key only) |
| `--lowquality` | PDF | Ignored (stored, no consumer) |
| `--title` | PDF | Supported (PDF /Title info) |
| `--margin-top/bottom/left/right` | PDF | Supported (page geometry + golden runner) |
| `--dpi` | PDF | Ignored (layout uses a fixed 96 dpi reference) |
| `--page-width`, `--page-height` | PDF | Supported (`pageGeometry` override) |
| `--image-quality` | Both | Ignored for PDF (JPEG pass-through; no re-encode). Image mode uses `--quality` |
| `--image-dpi` | PDF | Ignored (stored; paint size comes from layout CSS px @ 96 dpi) |
| `--no-pdf-compression` | PDF | Supported (uncompressed streams; used by tests) |
| `--use-xserver` | Both | Ignored (stored, no consumer) |
| `--cookie-jar` | Both | Ignored (loader always uses an in-memory jar) |
| `--read-args-from-stdin` | Both | Ignored (argv-only parsing; `-` output is covered, not stdin args) |

### 7.3 Pagination & smart shrinking

| Flag | Mode | Status |
|------|------|--------|
| `--page-offset` | PDF | Supported (header/footer and TOC page numbers) |
| `--smart-shrinking` | PDF | Supported (re-layout with zoom; `TestRunPDFSmartShrinking`) |
| `--enable-smart-shrinking`, `--disable-smart-shrinking` | PDF | Supported (same key) |

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
| `--enable-javascript`, `--disable-javascript` | Both | Ignored (JS stripped at load; no engine - the warning stub `load.WarnJSStubs` is not wired into the pipeline) |
| `--enable-local-file-access`, `--disable-local-file-access` | Both | Supported (local-file ACL; security policy §6 + golden runner) |
| `--allow` | PDF | Supported (ACL allow-prefix list) |
| `--background`, `--no-background` | Both | Supported (paint gate; golden runner sets it on) |
| `--enable-plugins`, `--disable-plugins` | Both | Ignored (stored, no consumer) |
| `--default-encoding` | Both | Ignored (UTF-8 assumed throughout) |
| `--minimum-font-size` | Both | Ignored (stored, no consumer) |
| `--user-style-sheet` | Both | Ignored (stored, no consumer) |
| `--print-media-type`, `--no-print-media-type` | Both | Ignored (layout always runs with `Media: "print"`; the flag cannot change that) |
| `--simplify-dom`, `--no-simplify-dom` | Both | Supported (opt-in chrome-strip; landmarks-only `SimplifyChromeCSS`; default off; `TestSimplifyDOMOnHidesChrome`) |
| `--simplify-dom-profile` | Both | Supported (`mediawiki` adds MW selectors; empty = landmarks only; `TestSimplifyDOMMediaWikiProfile`) |
| `--print-link-underline` | Both | Supported (opt-in underline `a[href]` after cascade; default off; `TestPrintLinkUnderlineOptIn`) |
| `--media-type` | Both | Ignored (same) |
| `--javascript-delay` | Both | Ignored (`WaitJSDelay` is never invoked) |
| `--window-status`, `--run-script` | Both | Ignored (stored; warning stub not wired) |
| `--zoom` | Both | Supported (operator layout scale → `layout.Options.Zoom`; not stylesheet emulation) |
| `--stop-slow-scripts`, `--no-stop-slow-scripts` | Both | Ignored (no JS) |
| `--debug-javascript`, `--no-debug-javascript` | Both | Ignored (no JS) |
| `--load-error-handling` | Both | Supported (abort/skip/ignore in the loader) |
| `--load-media-error-handling` | Both | Ignored (both appliers are no-ops) |
| `--proxy` | Both | Partial (global proxy wired into the HTTP transport; object-level `load.proxy` is stored but not applied) |
| `--username`, `--password` | Both | Supported (HTTP basic auth) |
| `--custom-header-propagation`, `--no-custom-header-propagation` | Both | Ignored (`RepeatCustomHeaders` stored, never read) |
| `--timeout` | Both | Supported (HTTP response timeout) |
| `--external-links`, `--no-external-links` | PDF | Partial (the `stripLinkURIs` path exists, but `applyObjectDefaults` ORs the default on, so the off state is unreachable from the CLI - documented quirk in `convert.go`) |
| `--internal-links`, `--no-internal-links` | PDF | Partial (body `#` fragment GoTo via layout `OpLinkURI` + `applyInternalLinks`; HTML HF `#id` → body `AddLinkDest`. Geometry caveats — runs without paint boxes still skipped) |
| `--resolve-relative-links`, `--keep-relative-links` | PDF | Supported (`resolveRelativeLinkURIs`; relative `href` resolution vs keep-as-written) |
| `--font-path` | Both | Supported (extra font search directories for registry discovery) |
| `--use-system-fonts` | Both | Supported (opt-in system font dirs; default off for determinism) |
| `--produce-forms` | PDF | Ignored (no AcroForm support in the PDF writer) |

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

Short flags: `-q -g -O -s -T -B -L -R -c -t` alias the long forms in §7.2.

---

## Amendment process

Any change to this matrix = a plan amendment (Phase 0.5 review), recorded in
`plans/00-canonical-pure-go-rewrite.md`. Phase 4 closure goldens must map to
this matrix row-by-row.
