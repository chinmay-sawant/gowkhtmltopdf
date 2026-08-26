# Changelog

All notable changes to gowkhtmltopdf are recorded here. This project follows
semantic versioning; `VERSION` holds the current release and is stamped into
binaries at build time (see README "Versioning").

## 0.2.5 (2026-08-26)

### Added

- **Python bindings (in-process, [#35](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/35)):** new `bindings/c` frozen C ABI v1 (`GOWKHTMLTOPDF_VERSION 0.2.5`, `GOWKHTMLTOPDF_ABI_VERSION 1`, 7 status codes, `abi_version`+`struct_size` gate, borrowed inputs, `malloc`/`free` ownership) and `bindings/c/include/gowkhtmltopdf.h` as committed header. `CGO_ENABLED=1 -buildmode=c-shared` exports `gowkhtmltopdf_html_to_pdf` / `html_to_image` over `Document.WritePDF` / `ImageDocument.WriteImage` with timeout/context.
- **Python package `gowkhtmltopdf` on PyPI:** `bindings/python` (`pyproject.toml`, `src` layout, `setup.py` `VERSION` shim, `py.typed`, zero runtime deps, `requires-python >=3.8`) with `ctypes` loader (`_lib.py` search `GOWKHTMLTOPDF_LIBRARY_PATH` then wheel then `dist/`, `RTLD_LOCAL`, pinned `argtypes`, `ABI_VERSION` check, keepalive, `GIL` released during render), `Document`/`ImageDocument` snake_case parity plus `PDFOptions`/`ImageOptions`, `convert_html_to_pdf` / `convert_file_to_pdf` (file parent `file://` base for linked CSS) / `convert_html_to_image`, and typed errors with sentinels. `make c-shared`, `make python-binding-test`, and `scripts/build_cshared_for_wheel.sh` for wheels.
- **Wheels and publish:** `tool.cibuildwheel` `manylinux_2_28` `x86_64`+`aarch64` (+ `macos-13/14`, `windows-2019`) in `.github/workflows/publish-pypi.yml` (tag `v*` + `workflow_dispatch`, `id-token: write` Trusted Publishing, `check` with `scripts/check_versions.sh` + `twine check --strict` before `pypa/gh-action-pypi-publish`).
- **Docs:** new `documentation/python.md` (install, both snippet styles, options/mapping, errors 0-6, timeouts, security, ABI stability, self-build, platforms) plus `README.md` teaser and `documentation/README.md` / `getting-started.md` / `deferred.md` updates.

### Changed

- Bumped `VERSION` / `internal/cli.Version` / `bindings/python` / `bindings/c/include/gowkhtmltopdf.h` to `0.2.5` and aligned `scripts/check_versions.sh` single source.
- `Makefile` now has `BINDINGS_VERSION_LDFLAGS`, guarded `c-shared` / `bindings-clean` / `check-versions` / `python-binding-test` targets; `.golangci.yml` excludes `bindings` (cgo glue via `go vet`); `.gitignore` ignores `dist/` and `bindings/**/*.so/.dylib/.dll`; `ci.yml` adds purity guard and `build-shared` + `python-binding` jobs.

## 0.2.4 (2026-08-18)

### Breaking

- Replaced the public wkhtml-shaped `Converter`, dotted settings, and typed
  request wrappers with `Document` / `ImageDocument`, explicit `Content`
  sources, validation, writer-first methods, and `PDF` / `Image` helpers.
- Redesigned both command binaries around `-o/--output`, explicit
  `--html`/`--url` sources, positional page files, and `--cover`/`--toc`.
  The old `page`/`cover`/`toc` object grammar is no longer accepted.
- Added [the 0.2.4 migration guide](documentation/MIGRATION-0.2.4.md).

### Changed

- External comparison paths are frozen as `bench-cli-compare` for
  wkhtmltopdf and `bench` for WeasyPrint/Puppeteer; optional host tools are
  skipped with explicit evidence.
- The benchmark and sample commands now use the native 0.2.4 CLI grammar.

## 0.2.3 (2026-08-15)

Same engine as [0.2.2](#022-2026-08-15). Module path is
`github.com/chinmay-sawant/gowkhtmltopdf` so `go install` works.

```sh
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.3
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltoimage@v0.2.3
```

PRs: [#50](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/50),
[#51](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/51).
Compare:
[v0.2.2...v0.2.3](https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.2.2...v0.2.3).

### Changed

- **Module path ([#51](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/51)):**
  `go.mod` is `github.com/chinmay-sawant/gowkhtmltopdf`. Library import and
  `go install` / `go get` use that path. Nested stub modules keep `frontend/`,
  `output/`, and `docs/` out of the parent module zip.

## 0.2.2 (2026-08-15)

PDF version and conformance-profile release after [0.2.1](#021-2026-08-14).
Default output remains **unclaimed PDF 1.4**. `--pdf-version` `1.4` / `1.7` /
`2.0` is a version header, **not** a conformance claim. The claim is
`--pdf-profile` / `WithPDFProfile`. Encryption, AcroForm, and JavaScript
remain out of scope.

PRs: [#44](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/44),
[#45](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/45),
[#46](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/46),
[#47](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/47),
[#48](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/48),
[#49](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/49).
Compare:
[v0.2.1...v0.2.2](https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.2.1...v0.2.2).

### Added

- **PDF 1.7 profiles ([#45](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/45)):**
  Opt-in PDF/A-3a, PDF/UA-1, and dual `a3a-ua1` via `--pdf-profile` /
  `WithPDFProfile` (implies PDF 1.7). Claiming XMP (`pdfaid:part=3`,
  `pdfaid:conformance=A`, `pdfuaid:part=1`), sRGB OutputIntent, `/DefaultRGB`,
  MarkInfo, and a logical structure tree. Multi-page structure elements emit
  MCR dictionaries; CIDFontType2 `/FontName` matches parent `/BaseFont`.
- **PDF 2.0 and 2.0 profiles ([#46](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/46)):**
  Opt-in `%PDF-2.0` via `--pdf-version 2.0` / `WithPDFVersion("2.0")`
  (trailer `/ID`, UTF-8 document strings, non-claiming XMP). Opt-in PDF/A-4,
  PDF/UA-2, and dual `a4-ua2` via `--pdf-profile` / `WithPDFProfile` (implies
  PDF 2.0). Claiming XMP (`pdfaid:part=4` / `rev=2020`, `pdfuaid:part=2` /
  `rev=2024`), sRGB+Gray OutputIntent, structure `/Namespace`, ListNumbering,
  and dual named destinations (`/D` page + `/SD` structure).

### Changed

- **Profile Get and sentinels ([#47](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/47)):**
  `Get("pdfprofile")` returns the canonical token (`a3a-ua1` →
  `PDF/A-3a+PDF/UA-1`; `ua1` → `PDF/UA-1`; `a4-ua2` → `PDF/A-4+PDF/UA-2`;
  and so on). Profile + wrong-version conflicts use unified
  `ErrConformanceRequiresPDF17` / `ErrConformanceRequiresPDF20` (with
  `ErrProfileRequiresPDF17` / `ErrProfileRequiresPDF20` aliases).
  `ErrProfilePDF20Unsupported` remains defined for source compatibility but
  is never returned — 2.0 profiles are supported.

### Fixed

- **Tagged PDF wiring ([#45](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/45),
  [#47](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/47)):**
  Structure-tree and Arlington CIDFontType2 FontName issues on the 1.7 path;
  cloned-page MCIDs, link/outline `/SD` identity, single `/Document`, and
  header/footer isolation from the body tree. List tags nest
  `L` → `LI` → `LBody` → `Link` (inline links inside `<li>` are no longer
  siblings of `LI`).

## 0.2.1 (2026-08-14)

Contracts, print layout, and verification release. Tightens embedder public
APIs, unifies error/panic policies, enhances table and float layout handling,
and adds parser fuzzing and CI integration.

### Added

- **Library & Settings:** Unified `NetworkPolicy` definition between root
  `api` and `internal/load`. Added `EnableLocalFileAccess()` helpers on
  `PDFRequest`, `ImageRequest`, `Converter`, `GlobalSettings`, and `ObjectSettings`.
  Fluent builder `PdfGlobalOptions` stores options without panicking and validates
  at request boundary (`ValidatePDF` / `RunPDF`). Safe nil-receiver handling on `AddHTML`.
- **Flow & Tables:** Consecutive same-side float vertical stacking when
  overlapping; margin collapse through empty block elements; `flex-shrink`
  assertion testing; content-based grid row height calculations.
  Deterministic table continuation edge rendering for multi-page collapsed tables.
- **Verification & Fuzzing:** Continuous fuzz targets `FuzzParseHTML`,
  `FuzzParseCSS`, and `FuzzConvertHTML`. Added `master` branch to GitHub Actions CI workflow.

## 0.2.0 (2026-08-14)

Post-MVP release after [0.1.0](#010-2026-08-03). Still a **no-cgo, no-browser**
HTML template engine (HTML→PDF / HTML→image) for **structured templates and
documents**: invoices, receipts, certificates, storybooks, posters, boarding
passes, letters, statements, tables, and multi-page documents with headers,
footers, TOC, and PDF outlines. **Tier 1** (template quality) and **Tier 2**
(leave wkhtmltopdf for most template jobs) are closed. This is **not** Chrome
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
- **Print CSS for templates:** attribute / `:nth-child` / sibling selectors; float
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
- Flex/grid/float/sticky are a **print CSS subset**, not full CSS3.
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
