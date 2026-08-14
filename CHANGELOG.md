# Changelog

All notable changes to gowkhtmltopdf are recorded here. This project follows
semantic versioning; `VERSION` holds the current release and is stamped into
binaries at build time (see README "Versioning").

## Unreleased

## 0.2.0 (2026-08-14)

Post-MVP release after [0.1.0](#010-2026-08-03). Still a **no-cgo, no-browser**
HTML→PDF / HTML→image engine for **controlled templates** (invoices, tables,
headers/footers, TOC, outlines). **Tier 1** (report quality) and **Tier 2**
(leave wkhtmltopdf for most report jobs) are closed. This is **not** Chrome
print parity.

PRs: [#7](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/7)–[#34](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/34)
(engine, docs, samples), [#36](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/36)
(release prep), [#37](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/37)
(docs site), [#38](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/38)
(CONTRIBUTING). Compare:
[v0.1.0...v0.2.0](https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.1.0...v0.2.0).

### Added

- **Typography:** Liberation Sans/Serif/Mono Regular/Bold/Italic/BoldItalic
  plus DejaVu Sans Unicode fallback. CSS `font-weight` / `font-style` select
  real faces. Image mode rasterizes TTF outlines (coverage AA) instead of the
  0.1.0 5×7 bitmap font.
- **Fonts / Unicode:** `--font-path` / `--use-system-fonts`; Type0/CID
  Identity-H for runes above U+00FF; local/HTTPS `@font-face` TTF/OTF/WOFF1
  (PDF and image). OpenType GSUB via allowlisted `go-text/typesetting` when
  the face has it; Arabic presentation-form + Lam-Alef fallback. No bundled
  Noto CJK — pass a capable face or you still get tofu.
- **CSS for reports:** attribute / `:nth-child` / sibling selectors; float
  lite; real `inline-block`; `box-sizing`; `text-align: justify`; cell
  `vertical-align`; flex Stage A; grid Stage B + Stage C lite (named areas,
  dense, copy-inherit subgrid, one-axis masonry); `position`
  relative/absolute/fixed lite; print-scoped sticky; repeating `<thead>`;
  CSS `orphans`/`widows`; nested HTML headers/footers; multicol lite;
  static 2D transforms + opacity; `:has()`; size-only `@container`; print
  `@media` subset; HTML entity decoding; SVG-as-`<img>` via allowlisted
  `tdewolff/canvas`.
- **Library:** `ConvertHTML`; typed `PDFRequest` / `RunPDF` and
  `ImageRequest` / `RunImage` (string `Converter` API kept); `GuessURL`
  `inline:` prefix; `web.images=false` skips image XObjects; settings
  cloned at ownership boundaries; context cancellation through layout,
  HF paint, and raster.
- **CLI / ops:** mode-invalid flags fail at parse; Restricted dial pinning;
  opt-in `--simplify-dom` chrome-strip (default off); `v*` tag workflow
  publishes linux/windows/darwin × amd64/arm64 binaries + `SHA256SUMS`.
- **Documentation site:** https://chinmay-sawant.github.io/gowkhtmltopdf/
  — Overview, Getting Started, sidebar docs, Issue Dossier (1,329 open
  wkhtmltopdf issues classified), Showcase gallery, Benchmarks tab.
- **Docs:** fidelity guide, fonts guide, performance snapshots,
  comparison with SebastiaanKlippert/go-wkhtmltopdf, 2026 landscape note,
  architecture deep-dives, `CONTRIBUTING.md`, versioned `plans/0.1.0/` and
  `plans/0.2.0/`.
- **Samples:** fixtures 21–56 (detailed report, float chrome, thead, flex,
  CJK, sticky, nested HF, posters, storybooks, boarding pass, architecture
  diagram). Regenerated `output/` PDFs and showcase thumbs.
- **Benchmarks:** `make bench` / `make bench-cli-compare`. 2026-08-14
  generic CLI vs wkhtmltopdf 0.12.6.1: **16x** at 2 pages, **1.6x** at
  500 pages. Faster at every tested size. See
  `documentation/performance.md`.

### Changed

- `box-sizing: content-box` is now the default. Explicit `width` + padding
  without `box-sizing` grows vs 0.1.0; add `box-sizing: border-box` to keep
  the old visual size.
- Direct modules are the allowlisted `go-text/typesetting` and
  `tdewolff/canvas` (0.1.0 was stdlib-only). Still `CGO_ENABLED=0`.
- User docs rewritten from source; root README slimmed to landing +
  quick start + index.
- `CONTRIBUTIONS.md` renamed to `CONTRIBUTING.md` so GitHub surfaces it.

### Fixed

- Backgrounds and borders paint **under** text; unique multi-image
  XObjects; `tr` row backgrounds; `rgba()` composite; nested-table document
  order and `%` widths vs the parent containing block.
- Table cell height at final column width; empty-row collapse; per-row
  border-collapse; rowspan cite alignment; continuation-page table chrome.
- Long tokens / URLs honor `overflow-wrap` / `word-break`; emergency wrap;
  float tails clear; caption wrap; link-underline coalesce (skip bare URLs
  in ref lists).
- Pagination: `page-break-before:always` lands at next-page top; multi-section
  reports paginate 1:1; `preferSplitOverBlank` for short avoid-boxes;
  **disabled** document-global gap packing that interleaved body and
  reference text.
- `display:flex` / `grid` restored after lint adoption; sticky chrome no
  longer clones like `fixed`; dashed / left borders no longer stretch into
  solid stubs.
- Original-template typography: `letter-spacing`, `text-transform`, and
  `border-radius` survive into the PDF.

### Known limitations

- **Not a browser.** No JavaScript (`<script>` stripped; JS flags are
  unknown options). No Chrome / Wikipedia visual parity.
- Flex/grid/float/sticky are a **report subset**, not full CSS3.
- CJK/Arabic **Partial** (operator-supplied faces + OT when present).
  `writing-mode: vertical-*` is parsed but lays out horizontal. No WOFF2.
- No AcroForm, no PDF encryption / PDF/A / 1.7 / 2.0 / UA (tickets
  #29–#33 remain open).
- Full list: [`documentation/deferred.md`](documentation/deferred.md).

## 0.1.0 (2026-08-03)

First MVP release: a pure-Go HTML-to-PDF engine (no cgo, no browser) with a
wkhtmltopdf-compatible CLI and library API. Direct module dependencies are
the allowlisted `github.com/go-text/typesetting` and `github.com/tdewolff/canvas`
modules recorded in `go.mod`.

### Added

- **Settings + CLI (phases 0-1):** wkhtmltopdf-compatible flag surface  - 
  global options, `page`/`cover`/`toc` object grammar, `-V`/`--version`,
  exit-code mapping; dotted-name settings model with defaults matching
  pdfsettings.cc.
- **Resource loader (phase 2):** URL guess, HTTP(S) with auth/cookies/POST,
  local-file ACL (deny by default) and `--allow` list, `data:`/`inline:`
  sources.
- **PDF writer (phase 3):** object model, xref, content streams, Flate
  compression, TTF subset embedding, JPEG/PNG images, link annotations,
  outlines; PDF/A, encryption and AcroForm intentionally out of scope.
- **HTML + CSS layout (phase 4):** stdlib HTML parser, CSS subset with
  cascade/important, block/inline boxes, tables (colspan), lists, images,
  colors/backgrounds/borders, `page-break-*`.
- **Pagination (phase 5):** fragment-level pagination, multi-object
  assembly, copies/collate, TOC page reordering, smart-shrinking
  over-width warning.
- **Headers/footers, TOC, outline, links (phase 6):** text headers/footers
  with `[page]`/`[frompage]`/`[title]`/… placeholders, table of contents,
  PDF outline, internal/external links.
- **Image converter (phase 7):** `gowkhtmltoimage` PNG/JPEG raster output
  (bitmap-font text, no anti-aliasing - see limits).
- **Library API (phase 8):** `gowkhtmltopdf` root package  - 
  `NewConverter`/`AddObject`/`Convert`/`Output`,
  `NewImageConverter`, settings `Set`/`Get`.
- **Release gates (phase 9):** 10-page table-report performance budget
  test, GitHub Actions CI (test + lint + `CGO_ENABLED=0` builds), `VERSION`
  file with ldflags stamping, this changelog, README rewrite.

### Known limitations

- No JavaScript/WebKit features; no floats/positioned layout, flex or grid;
  subset CSS selectors only.
- Single embedded font (Liberation Sans subset) for all font families; no
  CJK/complex-script shaping; italics render upright, `ol` markers are
  bullets.
- Image mode renders text with a 5x7 bitmap font - no anti-aliasing, no
  glyph rasterizer.
- No XSLT TOC stylesheets, no AcroForm forms, no SVG/BMP output, no SOCKS5
  proxy, no PDF encryption/PDF-A.
- Full list: README → "Deferred / not planned".
