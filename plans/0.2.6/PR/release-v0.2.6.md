## v0.2.6

Seventh public release of **gowkhtmltopdf**: a **pure-Go**, **no-cgo**, **no Qt/WebKit**, **no browser** HTML template engine that turns structured HTML and templates into multi-page PDFs and images.

**v0.2.5** shipped the in-process Python path. **v0.2.6** is the print CSS coverage release. The engine now implements **354 of the 818 W3C webref CSS properties**, with **0 partial** states and the remaining 464 tracked as unsupported. It also adds the **browser WASM adapter** for PDF, PNG, and JPEG output, recovers most of the warm-path time and allocation lost during the CSS work, and fixes table continuation, border joins, form widgets, list markers, and vertical text.

Default output is still **unclaimed PDF 1.4**. `--pdf-version` / `Document.PDFVersion` is a version header, **not** a conformance claim. The claim is `--pdf-profile` / `Document.PDFProfile`. The Python and C ABI surfaces keep their v0.2.5 contract, stamped `0.2.6`.

- **License:** [MIT](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/LICENSE) - Copyright (c) 2026 Chinmay Sawant
- **Version source:** [`VERSION`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/VERSION) (`0.2.6`)
- **Site:** https://chinmay-sawant.github.io/gowkhtmltopdf/
- **Live demo:** https://chinmay-sawant.github.io/gowkhtmltopdf/#/live-demo
- **Compare:** https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.2.5...v0.2.6
- **PRs:** [#62](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/62), [#64](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/64), [#65](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/65), [#66](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/66), [#67](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/67), [#68](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/68), [#69](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/69), [#70](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/70), [#71](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/71), [#72](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/72)

---

### Highlights

| Area | What ships in v0.2.6 |
|------|----------------------|
| **Print CSS coverage** | 354 implemented / 0 partial / 464 unsupported of 818 webref properties (`plans/0.2.6/catalog/coverage-summary.json`, generated 2026-09-12). The catalog, compatibility matrix, and apply arms are checked against each other. |
| **Backgrounds and borders** | Multi-layer background images with `background-size` (`contain`, `cover`, explicit lengths), `background-position`, `background-repeat`, `background-clip`, and `background-origin`; multi-layer `box-shadow` including `inset`; `border-image`; logical border longhands and logical corner radii; 1-4 value border shorthands; correct joins where border widths differ. |
| **Text, transforms, and compositing** | `text-shadow`, `text-decoration-thickness`, `text-decoration-color`, `text-underline-offset`, `text-align-last`, `tab-size`, manual `hyphens`; individual `translate` / `rotate` / `scale` / `transform-box`; `mix-blend-mode` and `isolation` run as real element transparency groups in PDF (Form XObjects with `/Group /S /Transparency /I true`) and PNG (group buffers). |
| **Browser WASM** | Inline HTML to PDF, PNG, or JPEG in the browser. This release attaches `gowkhtmltopdf_0.2.6_wasm.wasm`, `wasm_exec_0.2.6.js`, `wasm_sample_0.2.6.html`, and `wasm_manifest_0.2.6.json`. No local files, no arbitrary remote resources, no document JavaScript. |
| **Tables and paged media** | Header rows repeat on continuation pages, `contain: size` cells measure as one placeholder, page-break seals ignore transformed chrome, and fixture-63 pins `break-before` / `break-after` / `break-inside` behavior. |
| **Performance** | Current Snapshot N (2026-09-13): CLI 13 ms vs wkhtmltopdf 258 ms at 2 pages (19.68x) and 573 ms vs 1.718 s at 500 pages (3.00x), with lower peak RSS at every tested size. The warm 500-page path moved from the 1,228.72 ms recovery baseline to 576.33 ms in the same-source phase-6 capture (2.13x). |
| **Golden corpus** | Fixtures 57-63 added: implemented-props galleries (60-62), the unsupported worklist audit (58), the Apex landing page (59), and page-level demos (63). The corpus is 63 numbered fixtures plus 3 named harness pages. |

PDF 1.7 / 2.0 and the PDF/A + PDF/UA profiles from earlier releases are unchanged.

---

### Showcase - browser demo

Live demo: https://chinmay-sawant.github.io/gowkhtmltopdf/#/live-demo  
WASM guide: [`documentation/wasm.md`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/documentation/wasm.md)

Paste inline HTML, convert to a PDF preview, and download any page as PNG or JPEG. Multi-page image exports arrive as a ZIP built in the browser. PNG keeps transparency with a checkerboard preview; JPEG composites onto white. The conversion runs in a worker through the same Go layout and paint code, compiled to `GOOS=js GOARCH=wasm`, with no server round trip.

---

### Install / build

Cross-platform Go binaries are attached to this release (`gowkhtmltopdf` and `gowkhtmltoimage` for linux / windows / darwin on amd64 and arm64), plus `SHA256SUMS` and the four WASM artifacts.

Install the CLIs with Go 1.26+:

```sh
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.6
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltoimage@v0.2.6
gowkhtmltopdf --version
```

Library pin:

```sh
go get github.com/chinmay-sawant/gowkhtmltopdf@v0.2.6
```

Python (in-process; wheels are published from this tag):

```sh
pip install gowkhtmltopdf
```

```python
from gowkhtmltopdf import PDFOptions, convert_html_to_pdf

pdf_bytes = convert_html_to_pdf(
    b"<html><body><h1>Invoice #42</h1></body></html>",
    options=PDFOptions(page_size="A4"),
)
```

Default Go builds stay `CGO_ENABLED=0`. The shared library and Python wheels rebuild with `CGO_ENABLED=1 make c-shared` and a C toolchain.

---

### What landed in v0.2.6

#### 1. Print CSS coverage (PRs [#62](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/62), [#65](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/65), [#70](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/70))

The catalog program mapped all 818 names in the webref CSS list onto the engine and closed every "partial" state:

- 354 implemented, 0 partial, 464 unsupported, 0 ignored (`plans/0.2.6/catalog/coverage-summary.json`); 258 apply arms are mapped to code.
- 24 previously demoted properties returned with real layout, paint, or shaping consumers. That includes `contain` / `contain-intrinsic-*` / `content-visibility`, print-color-adjust and forced-color-adjust, image orientation and resolution, `unicode-bidi`, `text-orientation`, `text-combine-upright`, and `font-language-override`.
- `mix-blend-mode` and `isolation` moved to implemented with element-level transparency groups shared by the PDF and PNG writers.
- 28 common `-webkit-*` aliases remap to their standard properties (`-webkit-box-sizing`, `-webkit-text-fill-color`, `-webkit-box-shadow`, `-webkit-border-radius`, and more).
- `documentation/compatibility-matrix.md`, `catalog/mapping.json`, and the fixture evidence move together; `scripts/css-catalog-map.py --check` guards drift.

#### 2. Browser WASM adapter (PRs [#68](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/68), [#69](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/69), [#71](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/71), [#72](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/72))

- `bindings/wasm` exports one bridge function, `gowkhtmltopdfWASM(jsonRequest, progressCallback)`, returning PDF, PNG, or JPEG bytes plus width, height, MIME type, and version metadata.
- The request accepts inline HTML, mode, page size, orientation, dimensions, padding, and quality. It has no file or URL fields, and the render runs with a network policy that allows zero schemes, local files off, and system fonts off.
- Limits are explicit: 4 MiB HTML, 32 MiB output, 4096 px per image axis, 60 seconds per conversion.
- `make wasm` builds the version-stamped artifact into the site; `make wasm-test` runs the contract tests, the frontend lint / build / tests, and a Chrome smoke test for PDF, PNG, and JPEG. CI runs the same target.
- The `/live-demo` route replaces the earlier `/wasm` route and adds paginated previews and image downloads.

#### 3. Performance and memory (PR [#70](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/70))

- Style resolution is memoized with a declared-property mask, `ResolvedStyle` records are interned through generated fingerprint code, grid border ops batch into `OpGridRun`, paint-range checks are O(1), and pagination skips census walks.
- The PDF writer keeps parallel flate workers, a bounded parsed-font cache, lazy per-face parsing, and releases page raw buffers after stream materialization.
- Image output gained a direct final-resolution raster threshold, a streaming filter-none PNG writer, strip-window raster reuse, a bounded supersample cache, pooled encode buffers, and glyph scratch pools.
- Committed Snapshot M medians: 695.42 ms internal / 698.79 ms public library / 0.70 s CLI for 500 pages, with B/op at or below target. Allocated bytes per 500-page PDF fell from 321.10 MB (Snapshot K) to 163.02 MB (Snapshot M). The image 500-tile figure fell from 94.97 MB to 27.21 MB B/op, and PDF allocations dropped 94.5 percent across the memory wave.
- Current README capture (2026-09-13): `Document.WritePDF` at 5.50 ms for 2 pages and 554.56 ms for 500 pages; Python at 3.52 ms and 504.93 ms.

#### 4. Layout, tables, and correctness (PRs [#64](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/64), [#65](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/65), [#66](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/66), [#71](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/71))

- `thead` rows repeat on every continuation page and table bodies no longer overlap the repeated band.
- Table cells with `contain: size` or `content-visibility: hidden` measure as one intrinsic placeholder instead of walking hidden descendants.
- Border shorthands expand 1-4 values per side, and corners where border widths differ join with miter-shortened rails.
- `input[type=checkbox]` and `input[type=radio]` paint native control faces with tick or dot, auto size, and accent color.
- The `list-style` shorthand expands at cascade time, so author rules beat the UA disc marker.
- Vertical writing modes paint upright text runs without rotation, and `container-type` / `container-name` resolve CSS-wide keywords.
- Opacity no longer compounds on restamped transforms and filtered images.
- `Document.Validate` now accepts negative top and bottom margins as the auto header/footer sentinel, matching the CLI. Left and right margins still require non-negative finite values.

#### 5. Image output (PR [#70](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/70))

- The JPEG encoder converts NRGBA canvases to full-resolution YCbCr before `jpeg.Encode`, restoring the byte-exact output of the RGBA path. A parity suite covers 13 shapes including partial-MCU edges. Corpus allocations dropped 99.48 percent (101,272,253 to 526,832) and time 6935 ms to 5248 ms.
- Oversized clipped images scale only the visible strip window on the same sampling grid, so pixels stay identical while the cache stops rebuilding a 65.6 MiB canvas per strip. fixture-49: 1334.1 to 43.4 MB B/op and 1395 to 151 ms. fixture-53: 1329.2 to 38.2 MB and 1486 to 151 ms. The PNG corpus fell from 3.65 GB to 0.94 GB B/op.
- Quality clamps to 1-100 for JPEG; PNG ignores it. `--transparent` applies to PNG only, while transparent JPEG warnings composite onto white.

#### 6. Golden corpus, samples, and dossier (PRs [#62](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/62), [#65](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/65), [#66](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/66), [#70](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/70))

- fixture-57 gallery proves all implemented properties render; fixture-58 proves the unsupported worklist parses and degrades without crashing.
- fixture-59 (Apex digital landing) exercises hero, auto-fit feature / pricing / gallery grids, CSS variables, and local assets.
- fixtures 60-62 slice the implemented-props audit into three galleries; fixture-63 pins page-level `break-*` demos and inert footnote display.
- `make samples` regenerated 63 fixture PDFs/PNGs plus the 1.7 and 2.0 profile smokes; `make screenshots` rebuilt 223 PNGs and 223 WebP thumbnails.
- The issue dossier was re-verified against 0.2.6: 547 implemented / 339 partial / 443 not-implemented out of 1,329 wkhtmltopdf verdicts.

#### 7. Docs, site, and tooling (PRs [#62](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/62), [#67](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/67), [#68](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/68), [#69](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/69), [#70](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/70))

- New `documentation/benchmarks.md`; `performance.md` re-based to the 2026-09-13 capture; architecture pages, `cli.md`, `compatibility-matrix.md`, `fonts.md`, and `samples.md` refreshed.
- The site moved Getting Started under `/documentation/getting-started`, sorted the compatibility table by support tier, consolidated the benchmarks page, and rebuilt `docs/`.
- `make lint` now also runs a file-size gate (`scripts/check-file-size.sh`, three allowlisted files).
- No public Go API or CLI flag changes. No new dependencies; the direct module allowlist still holds two entries.

---

### Documentation

| Doc | Link |
|------|------|
| **Site** | https://chinmay-sawant.github.io/gowkhtmltopdf/ |
| **Overview** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/documentation/overview.md |
| **Getting started** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/documentation/getting-started.md |
| **CLI** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/documentation/cli.md |
| **Library API** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/documentation/library-api.md |
| **Python** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/documentation/python.md |
| **Browser WASM** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/documentation/wasm.md |
| **Compatibility matrix** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/documentation/compatibility-matrix.md |
| **Benchmarks** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/documentation/benchmarks.md |
| **Deferred work** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/documentation/deferred.md |
| **Changelog** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/CHANGELOG.md#026-2026-09-13 |
| **0.2.6 plans** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.6/plans/0.2.6/README.md |

---

### Known limits (honest)

- Not a browser. There is no Chrome, WebKit, or Qt print parity claim, no JavaScript execution, and no cgo on the default build path. URL sources render on a best-effort basis; live-site output is exploratory.
- 464 of 818 webref properties stay unsupported on purpose: 155 print no-ops (animation, transitions, scroll snap, pointer UI), 87 hard defers, 48 alias-when-base-done rows, and 94 niche or draft names. Permanent non-goals include 3D transforms, filter blur and drop-shadow, full flex / grid / subgrid intrinsic layout, CJK font bundling, and WOFF2.
- WOFF2 fonts are rejected with an explicit error. Fonts load from TTF, OTF, and WOFF1 under the configured ACL; no HarfBuzz and no variable-font instancing.
- Wave 2 performance stays open. The goal to halve the remaining public library PDF time and allocation is not met: 624.49 ms / 121.02 MB B/op against 366.74 ms / 81.51 MB targets (`plans/0.2.6/perf-improve/wave-2-50pct/phase-wise-checklist.md`). The 250-tile image time also finished slightly above target.
- The browser adapter accepts inline HTML only. Local files, arbitrary remote CSS / images / fonts, and system fonts are not available, and the caps are 4 MiB HTML, 32 MiB output, 4096 px per image axis, and 60 seconds. No native-versus-WASM byte-identity claim.
- The C ABI is a one-shot inline-HTML interface with no document handle. `GwkPdfOptions` has no cover, TOC, header/footer, outline, font-path, or system-font fields, so it keeps engine defaults. Python file and URL sources raise `NotImplementedError`.
- Python wheels ship for Linux x86_64 / aarch64 (`manylinux_2_28`), macOS arm64, and Windows x86_64. Intel macOS wheels are not built. Rebuilding the shared library needs `CGO_ENABLED=1` and a C toolchain.
