# gowkhtmltopdf — HTML/CSS Compatibility Matrix (MVP Allowlist)

> **Parent:** `plans/00-canonical-pure-go-rewrite.md` (Phase 0.1)
> **Status:** frozen for MVP; amendments go through plan review
> **Target:** controlled report/invoice HTML → PDF. **Not** a browser.

This document is the **contract** the layout engine (Phase 4) is allowed to
implement. Anything not listed here is *unsupported*; unsupported input must
degrade gracefully (ignored declaration / skipped node / documented error),
never crash.

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
| `img` | Replaced element; **PNG/JPEG only** (intrinsic dims `layout.go:352`; GIF not detected → skipped) |
| `a` | Hyperlink (`href`) for `http/https/mailto` targets (`inline.go:226,305`); internal anchors deferred to Phase 6 |
| `strong`, `em`, `b`, `i`, `u`, `small` | `b`/`strong` bold (fake bold), `u` underline, `small` smaller; `em`/`i` parse italic but render upright (no italic font yet, §2.3) |
| `pre`, `code` | `pre` honors `white-space: pre`; `code` has no monospace rule — single embedded font for all families (§2.3) |
| `blockquote` | Block-level only — no indent margins (UA rule `style.go:714-717`) |
| `header`, `footer`, `main`, `section`, `article`, `aside`, `nav` | Treated as `div` (semantic aliases) |

## 2. Supported CSS properties

Status legend (verified against `internal/layout/style.go` `applyRestProps` +
`uaRules`, `internal/css/css.go`, `internal/layout/layout.go`, `inline.go`,
`paint.go`, and the tests in `internal/layout/layout_test.go` /
`internal/convert/golden_test.go` as of Phase 4):

- **Implemented** — parsed and consumed by layout.
- **Partial** — parsed and used in a subset of the declared cases; the rest
  degrades silently.
- **Not implemented** — parsed and dropped, or not parsed at all; the
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
| `box-sizing` (`content-box\|border-box`) | Not implemented | absent from `applyRestProps` (`style.go:291-458`). Default behavior is mixed: auto width is border-box-ish, explicit width is content-box (`layout.go:176-191`) |
| `overflow` (`visible\|hidden`) | Not implemented | absent from `applyRestProps`; no clipping anywhere |

### 2.2 Display & flow

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `display` (`block\|inline\|none\|list-item\|table\|table-row\|table-cell\|table-row-group\|table-header-group\|table-footer-group`) | Implemented | `style.go:295-301`; consumed `layout.go:158-171, 243-261`; `none` test `TestDisplayNone`; tables test `TestTableLayout` |
| `display: inline-block` | Partial | parsed, but degrades to block layout (`layout.go:255` only treats `inline` as inline); `<img>` gets `inline-block` via UA rule and works only because `img` is special-cased |
| `display: table-caption`, `table-column(-group)` | Not implemented | parsed (`style.go:299`); no caption/column model in `buildTable` (`layout.go:382`) — `<caption>` does not render |
| `float` (`left\|right`) | Not implemented | parsed (`style.go:306-310`); only consumer is `layout.go:255`, which treats a floated element as plain in-flow inline — no float positioning. `layout.go:8` declares floats out of scope |
| `clear` (`left\|right\|both`) | Not implemented | parsed (`style.go:311-315`), never consumed |
| `position` (`static\|relative`) | Not implemented | parsed (`style.go:302-305`), never consumed — everything is `static`; `relative` produces no shift |
| `position: fixed` / `absolute` | Not implemented | ignored → `static` (consistent with §5) |

### 2.3 Text & fonts

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `font-family` (named + generic) | Partial | parsed + inherited (`style.go:261-265`, `css.go:899`); layout renders every run with the single embedded Liberation Sans (`inline.go:154`) — no family selection |
| `font-size` | Implemented | `style.go:258-260`, `fontSize` `style.go:561` (px/pt/em/%/rem/in/cm/mm/pc + keywords); `%`/`em` resolve against parent; test `TestFontSizeEmInherit` |
| `font-weight` (`normal\|bold\|100-900`) | Implemented | `style.go:266-281`; ≥700 renders fake bold via stroke text render mode (`paint.go:130-133`); test `TestBoldUnderline` |
| `font-style` (`italic\|oblique`) | Not implemented | parsed (`style.go:282-284`) into `FontItalic`, never consumed by layout/paint (no slant) |
| `text-align` (`left\|right\|center\|justify`) | Partial | `style.go:412-420`; consumed `inline.go:118-126` for left/right/center; `justify` parses but renders as left |
| `text-decoration` (`none\|underline\|line-through`) | Implemented | `style.go:433-441`; drawn `inline.go:156-161`; test `TestBoldUnderline` |
| `text-indent` | Not implemented | parsed (`style.go:444-445`), never consumed |
| `line-height` (number, length, `normal`) | Implemented | `style.go:410-411`, `lineHeight` `style.go:607`; consumed `inline.go:262-267`; test `TestMarginCollapse` |
| `letter-spacing` | Implemented | `style.go:442-443`, consumed in run width `inline.go:258` |
| `word-spacing` | Not implemented | absent from `applyRestProps` |
| `text-transform` | Not implemented | absent from `applyRestProps` |
| `vertical-align` (`baseline\|top\|middle\|bottom`) | Not implemented | parsed (`style.go:421-425`), never consumed by line layout |
| `white-space` (`normal\|nowrap\|pre\|pre-wrap`) | Partial | `style.go:426-432`; normal/nowrap/pre implemented (`inline.go:62, 188-207`); `pre-wrap`/`pre-line` collapse to `pre`; test `TestWhiteSpacePre` |

### 2.4 Color & background

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `color` | Implemented | `style.go:402-405`; consumed `inline.go:152-155`; test `TestCascadeAndInline` |
| `background-color` | Implemented | `style.go:406-409`; painted `layout.go:234-237, 504-507, 531-534` (gated by `Background`); tests `TestBackgroundFill`, `TestRunPDFStyleTableImage` |
| `background` (shorthand) | Not implemented | absent from `applyRestProps` (only `background-color`) |
| `opacity` | Not implemented | absent from `applyRestProps`; the alpha channel only carries `rgba()`/`#rrggbbaa` background alpha |

### 2.5 Table subset

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `border-collapse` (`collapse\|separate`) | Not implemented (separate only) | parsed (`style.go:446-449`), never consumed — `buildTable` always uses the separate model with `BorderSpacing` (`layout.go:450, 500`) |
| `border-spacing` | Implemented | `style.go:450-451`; used `layout.go:450-455, 500` |
| `caption-side` | Not implemented | absent from `applyRestProps`; `<caption>` elements are not rendered (see `display: table-caption`) |
| `table-layout` (`auto\|fixed`) | Not implemented (auto only) | parsed (`style.go:452-455`), never consumed |

### 2.6 Print / paged media

| Property | Status | Notes / verified by |
|----------|--------|---------------------|
| `page-break-before/after/inside` (`auto\|always\|avoid`) | Implemented (print pipeline) | parsed into `style.PageBreak*`; honored as canvas-Y flow shifts by the phase-5 paginator — `beforeAlways` `paint.go:203`, `afterBreaks` `paint.go:236`, `avoidInside` `paint.go:179`; tests `TestPageBreakParsing`, `TestPageBreakBeforeAlways`, `TestPageBreakInsideAvoid` |
| `orphans`, `widows` | Not implemented | absent from `applyRestProps` |

### Feature checklist (page geometry, tables, pagination)

| Feature | Status | Notes / verified by |
|---------|--------|---------------------|
| Page size (A4/Letter/…), landscape, margins (mm) | Implemented | `settings.ParsePageSize` (`settings/pagesize.go:39`), `convert.pageGeometry` (`convert.go:138`) |
| `colspan` | Yes | `colSpan` `layout.go:618-622`; test `TestTableColspan` |
| `rowspan` | No | attribute ignored — only `colspan` is read |
| `border-collapse` | Separate only | see §2.5 |
| Pagination | Fragment + whole-op (phase 5) | rect-type ops (fill/stroke/line) split at page boundaries; text/images/links move wholly (line-level) (`paint.go:107-150`); `page-break-before/after: always`, `page-break-inside: avoid`, table rows never split (`paint.go:179-336`); element → (page, rect) map in `Result.Locations` for Phase 6. See "Pagination (phase 5)" note below. |
| Floats / absolute positioning | No | see §2.2; degrade to in-flow layout |
| Flexbox / Grid | No | `display:flex|grid` not in the allowlist → ignored, element keeps the initial `inline` display (`style.go:295-301`; see §5) |
| JavaScript | No | stripped at load; `--enable-javascript` accepted + warning (Phase 1) |

**Pagination (phase 5).** Implemented 2026-08-03 (`internal/layout/paint.go`): fragmentation is box-aware — rect-type ops crossing a page boundary are split at the boundary, while text, images and links move wholly to the next page (text is already line-level, so glyphs never split). `page-break-before/after: always` and `page-break-inside: avoid` are honored via canvas-Y flow shifts (`shiftFlowY` `paint.go:156`), and table rows never split (`rowsIntact` `paint.go:290`). `Result.Locations` (`paint.go:341`) carries element boxes (page + rect) for Phase 6 outline/TOC/links. Remaining partials: `--zoom` is accepted and `layout.Options.Zoom` works (`TestZoom` `layout_test.go:726`), but the convert pipeline does not forward it yet; smart-shrinking detects over-wide content and warns without re-layout (`convert.go:218-229`); table-header repeat across pages not implemented; orphan/widow control not implemented.

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
| `ex`, `ch` | Not implemented | accepted by the parser (`css.go:731`) but never resolved by `style.go` — declaration dropped |
| `vmin`, `vmax`, `dpi`-style | Not implemented | rejected at parse |

## 4. Supported selector syntax (cascade)

Status legend as in §2; evidence in `internal/css/css.go`.

| Selector | Status | Notes / verified by |
|----------|--------|---------------------|
| Element (`h1`, `p`, …), class (`.foo`), ID (`#bar`) | Implemented | `parseCompound` `css.go:444`; matching `Match` `css.go:518`; test `css_test.go::TestMatch` |
| Universal (`*`) | Implemented | `css.go:456-459` |
| Descendant (`div p`), child (`ul > li`) | Implemented | combinators `css.go:356-362`; matching `css.go:528-543` |
| Sibling (`a + b`, `a ~ b`) | Partial | parsed (`css.go:357-360`) but matched as descendant (`css.go:534` default branch) |
| Attribute (`[href]`, `[href="…"]`) | Not implemented | dropped from the compound during tokenizing (`css.go:396-405`); `[x]` alone becomes `*` |
| `:first-child`, `:last-child`, `:nth-child(n)` | Not implemented | dropped during tokenizing (`css.go:406-421`) — note `tbody tr:nth-child(even)` in fixture-02 matches **all** rows |
| `:link`, `:visited`, `:hover`, `:active`, `:focus` | Not implemented (accepted, ignored) | dropped with the other pseudo-classes; no interaction states in print |
| `::before` / `::after` | Not implemented | dropped with the other pseudo-classes |
| `!important` | Implemented | `css.go:664-688`; separate cascade layer `style.go:221-247`; test `css_test.go::TestParseImportant` |
| Specificity (ID > class > element), inline `style` wins, `!important` overrides | Implemented | `Specificity` `css.go:578`; inline style priority `style.go:233-239`; test `css_test.go::TestSpecificity` |
| `@media print` / `screen` filtering | Implemented | `mediaType` `css.go:186`; applied per rule `style.go:212-214`; convert passes `Media: "print"` (`convert.go:115`); test `css_test.go::TestParseMedia` |
| `@media` feature queries (`(min-width: …)`) | Not implemented | only the media type substring is considered |
| `@page`, `@font-face` | Not implemented | block skipped gracefully (`css.go:85-95`) |

## 5. Explicitly unsupported (MVP)

| Feature | Handling |
|---------|----------|
| JavaScript / `<script>` / DOM APIs | **Stripped at load.** `--enable-javascript` accepted but ignored with warning (Phase 1) |
| CSS Grid, Flexbox (`display:flex/grid`) | Declaration ignored → element keeps the initial `inline` display (`style.go:295-301` accepts only the listed values) |
| `position: fixed` / `absolute` | Ignored → `static` |
| `transform`, `filter`, `animation`, `transition` | Ignored |
| `background-image` / gradients | Ignored (Phase 3+ candidate) |
| `@font-face` (remote or local) | Ignored; font-family falls back to defaults |
| Custom XSLT TOC (`--xsl-style-sheet`) | Not implemented (no XSLT in stdlib); Go templates instead (Phase 6) |
| WebP, SVG-as-`img`, AVIF | Not decodable by stdlib; broken-image placeholder or skip |
| `position: fixed` headers/footers | Use CLI `--header-*` / `--footer-*` flags instead (Phase 6) |
| RTL / complex-script shaping (Arabic, Devanagari, CJK) | Latin-first; non-Latin best-effort via embedded fonts (Phase 3); documented limit |
| PDF encryption, PDF/A, duplex, AcroForm | Out of scope (not in original wkhtmltopdf either) |

## 6. Security policy (frozen defaults)

| Rule | Value |
|------|-------|
| Local file access | **Blocked by default** (`--enable-local-file-access` opt-in; `--allow` path allowlist walk) |
| Untrusted HTML | **Not supported** — same warning as upstream `docs/status.md`; use with HTML you control only |
| Remote URL fetch | `net/http` defaults: connect + response timeouts, redirect limit, `blockLocalFileAccess` covers `file://` and localhost refs |
| SSRF posture | No automatic form submission; POST only via explicit `--post` flags; no cookies auto-forwarded from site contexts |

---

## Amendment process

Any change to this matrix = a plan amendment (Phase 0.5 review), recorded in
`plans/00-canonical-pure-go-rewrite.md`. Phase 4 closure goldens must map to
this matrix row-by-row.
