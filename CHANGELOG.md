# Changelog

All notable changes to gowkhtmltopdf are recorded here. This project follows
semantic versioning; `VERSION` holds the current release and is stamped into
binaries at build time (see README "Versioning").

## Unreleased

### Fixed (print / layout)

- **Long tokens / URLs:** honor `overflow-wrap` / `word-break` (with inheritance);
  emergency wrap when a token exceeds the line so text does not paint past the
  page edge; soft breaks at URL punctuation.
- **Float tails:** short remaining text that fits one full-width line clears
  below the float instead of orphaning beside it.
- **Captions:** `overflow-wrap: break-word` no longer mid-breaks words that fit
  the next full line.
- **Tables:** per-row border-collapse grid bound to row op ranges (no phantom
  empty bands across page breaks); empty/padding-only rows collapsed; leading
  all-`<th>` rows repeat as headers; multi-cite nowrap min-content; rowspan
  cells with `<br>` cites spread vertically across the cell height.
- **Pagination:** `preferSplitOverBlank` for short `page-break-inside: avoid`
  boxes (dense reference lists); **disabled** document-global gap packing that
  crushed line spacing and interleaved body text.
- **Links:** coalesce same-href underlines on a line; clamp stroke weight; skip
  underlines on bare URL strings in reference lists.
- **Sample:** regenerate `output/wiki-ana-de-armas.pdf` (live Wikipedia smoke).

### Tests

- Layout regressions for overflow wrap, thead-from-th, ref gaps, table empty
  rows / multi-cite / rowspan cites / continuation borders, underline coalesce,
  and body-line packing (no overlap).

## 0.1.0 (2026-08-03)

First MVP release: a pure-Go, stdlib-only HTML-to-PDF engine with a
wkhtmltopdf-compatible CLI and library API.

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
