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
| `ul`, `ol`, `li` | Simple lists: bullet / decimal markers |
| `table`, `thead`, `tbody`, `tfoot`, `tr`, `th`, `td` | Table subset; see §4 |
| `img` | Replaced element; PNG/JPEG/GIF decode (stdlib `image/*`) |
| `a` | Hyperlink (`href`); internal anchors (Phase 6) |
| `strong`, `em`, `b`, `i`, `u`, `small` | Inline text styling |
| `pre`, `code` | Monospace block / inline (no tab expansion guarantee) |
| `blockquote` | Block with indent margins |
| `header`, `footer`, `main`, `section`, `article`, `aside`, `nav` | Treated as `div` (semantic aliases) |

## 2. Supported CSS properties

Property groups for Phase 4 implementation. Values outside the listed syntax
are **ignored** (declaration dropped, not an error).

### 2.1 Box model

- `margin` / `margin-top|right|bottom|left`
- `padding` / `padding-top|right|bottom|left`
- `border` / `border-top|right|bottom|left`, `border-width`, `border-style`
  (`solid|dashed|dotted|none`), `border-color`
- `width`, `height`, `min-width`, `min-height`, `max-width`, `max-height`
- `box-sizing` (`content-box|border-box`)
- `overflow` (`visible|hidden` only; no scrollbars)

### 2.2 Display & flow

- `display`: `block`, `inline`, `inline-block`, `none`, `table`,
  `table-row`, `table-cell`, `table-header-group`, `table-row-group`,
  `table-footer-group`
- `float` (`left|right`): **best-effort** — MVP may implement simple
  `img`/`div` floats; mark per-property status in Phase 4 goldens
- `position`: `static` and `relative` only (offset ignored if no box shift)
- `clear` (`left|right|both`) — only if floats implemented

### 2.3 Text & fonts

- `font-family` (named families + generic `serif|sans-serif|monospace`)
- `font-size` (see §3 units; `small|medium|large` keywords allowed)
- `font-weight` (`normal|bold|100–900`)
- `font-style` (`normal|italic|oblique`)
- `text-align` (`left|right|center|justify`)
- `text-decoration` (`none|underline|line-through`)
- `text-indent`
- `line-height` (number, length, `normal`)
- `letter-spacing`, `word-spacing`
- `text-transform` (`none|uppercase|lowercase|capitalize`)
- `vertical-align` (baseline, `top|middle|bottom` for inline boxes)
- `white-space` (`normal|nowrap|pre|pre-wrap`)

### 2.4 Color & background

- `color`
- `background-color`
- `background` (shorthand → color only; background-image deferred to §5)
- `opacity` (applied to text/box paint)

### 2.5 Table subset

- `border-collapse` (`collapse|separate`)
- `border-spacing`
- `caption-side` (ignored; caption renders above if present)
- `table-layout` (`auto|fixed`)

### 2.6 Print / paged media

- `page-break-before` (`auto|always|avoid`)
- `page-break-after` (`auto|always|avoid`)
- `page-break-inside` (`auto|avoid`)
- `orphans`, `widows` (simple heuristics, Phase 5)

## 3. Supported units

| Unit | Notes |
|------|-------|
| `px` | 1 px = 1/96 in (CSS reference) |
| `pt` | 1 pt = 1/72 in |
| `mm` | 1 mm = 1/25.4 in |
| `cm` | 1 cm = 1/2.54 in |
| `in` | inch |
| `em` | relative to element font-size |
| `%` | relative to containing block (or inherited font-size for `font-size`) |

`rem`, `vw`, `vh`, `ch`, `ex`, `vmin`, `vmax`, `dpi`-style units:
**unsupported** (ignored).

## 4. Supported selector syntax (cascade)

- Element selectors (`h1`, `p`, …)
- Class selectors (`.foo`)
- ID selectors (`#bar`)
- Descendant combinator (`div p`)
- Child combinator (`ul > li`)
- Attribute selectors (`[href]`, `[href="…"]` exact match only)
- Pseudo-classes: `:first-child`, `:last-child`, `:nth-child(n)`
- `:link`, `:visited`, `:hover`, `:active`, `:focus`: accepted and ignored
  (no interaction states in print)
- Pseudo-elements `::before` / `::after`: **unsupported** in MVP
- `!important` honored with specificity rules

Specificity: ID > class/attribute/pseudo-class > element. Inline `style`
wins over stylesheet rules (except `!important`). `@media print` rules apply
when `printMediaType` is on (default behavior, Phase 5).

## 5. Explicitly unsupported (MVP)

| Feature | Handling |
|---------|----------|
| JavaScript / `<script>` / DOM APIs | **Stripped at load.** `--enable-javascript` accepted but ignored with warning (Phase 1) |
| CSS Grid, Flexbox (`display:flex/grid`) | Declaration ignored → element falls back to `block` |
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
