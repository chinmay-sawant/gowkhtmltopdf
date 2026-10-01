## v0.2.7

Eighth public release of **gowkhtmltopdf**: a **pure-Go**, **no-cgo**, **no Qt/WebKit**, **no browser** HTML template engine that turns structured HTML and templates into multi-page PDFs and images.

**v0.2.6** shipped 354 implemented CSS properties and the browser WASM adapter. **v0.2.7** adds the next 72-name print CSS wave, 40 Chromium flex cases, and public packages that parse, style, and lay out HTML without writing a PDF. The catalog is now **407 implemented / 0 partial / 411 unsupported** of 818 W3C webref properties. 53 of the 72 new names have code that reads the declaration and code that places or paints it. 19 stay unsupported on purpose.

Default output is still **unclaimed PDF 1.4**. `--pdf-version` / `Document.PDFVersion` is a version header, **not** a conformance claim. The claim is `--pdf-profile` / `Document.PDFProfile`. Python and the C ABI keep the v0.2.5 contract, stamped `0.2.7`.

- **License:** [MIT](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/LICENSE) - Copyright (c) 2026 Chinmay Sawant
- **Version source:** [`VERSION`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/VERSION) (`0.2.7`)
- **Site:** https://chinmay-sawant.github.io/gowkhtmltopdf/
- **Live demo:** https://chinmay-sawant.github.io/gowkhtmltopdf/#/live-demo
- **Compare:** https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.2.6...v0.2.7
- **Tree this note covers:** `e974d30` (84 commits, 1,213 files, +44,837 / -1,952 since `v0.2.6`) plus the 0.2.7 stamp
- **PRs:** [#74](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/74), [#76](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/76), [#77](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/77), [#78](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/78), [#79](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/79), [#81](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/81)

---

### Highlights

| Area | What ships in v0.2.7 |
|------|----------------------|
| **Print CSS** | 407 implemented / 0 partial / 411 unsupported of 818 (`plans/0.2.6/catalog/coverage-summary.json`, counts block, generated 2026-09-17). The gain is 53 of the 72 next-72 names. Fixture-64 paints them. |
| **Layout without a PDF** | `html.Parse`, `css.Apply`, and `layout.Lay` are public. `screen.Render` returns a PNG and each element's rectangle. `markup.Parse` returns a detached copy of the tree. `Document` and `ImageDocument` still write the PDF and the image. |
| **Flex cases** | 40 HTML cases from Chromium and WPT sources, with Go box checks or PDF checks under `test/chrome/`. |
| **Link underlines** | Author `text-decoration: none` is what gets painted. A forced underline is only `--print-link-underline` (default off). |
| **Checks** | PDF operation checks for the numbered fixtures, plus 65 visual reference PDFs under `output/validated/`. The pinned full-corpus pixel run and its CI job stay open. |
| **Allocation** | Warm bytes for one pass of the 40 flex cases fell from 64.3 MB to 28.0 MB. The 2026-09-13 500-page snapshot was not remeasured. |

PDF 1.7 / 2.0 and the PDF/A + PDF/UA profiles from earlier releases are unchanged.

---

### Install / build

Cross-platform Go binaries are attached to this release (`gowkhtmltopdf` and `gowkhtmltoimage` for linux / windows / darwin on amd64 and arm64), plus `SHA256SUMS`. The tag workflow also publishes the version-stamped WASM artifact and the Python wheels.

Install the CLIs with Go 1.26+:

```sh
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.7
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltoimage@v0.2.7
gowkhtmltopdf --version
```

Library pin:

```sh
go get github.com/chinmay-sawant/gowkhtmltopdf@v0.2.7
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

### What landed in v0.2.7

#### 1. Next-72 print CSS (PR [#76](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/76))

Batches 87.1-87.8 closed on 2026-09-17. Catalog math: 354 + 53 = 407 implemented. Partial stayed 0. Unsupported fell from 464 to 411.

Implemented in this wave:

- **Fonts (21):** `font-feature-settings`, `font-kerning`, `font-variant` and its longhands, `font-synthesis` and its longhands, `font-stretch`, `font-width`, `font-size-adjust`. `font-optical-sizing` and `font-variation-settings` build an instance when the face has an `fvar` table. A static face ignores them. `font-palette` selects a COLR+CPAL palette and paints the first palette color as a solid fill.
- **Text and inline (15):** `text-box`, `text-box-edge`, `text-box-trim`, `text-autospace`, `text-spacing`, `text-spacing-trim`, `text-group-align`, `hanging-punctuation`, `hyphenate-limit-chars`, `hyphenate-limit-last`, `hyphenate-limit-lines`, `hyphenate-limit-zone`, `initial-letter`, `initial-letter-align`, `initial-letter-wrap`.
- **Grid, columns, images, overflow, shapes, floats, counters (17):** `grid-gap`, `grid-row-gap`, `grid-column-gap`, `grid-auto-columns`, `grid-auto-rows`, `column-height`, `column-wrap`, `aspect-ratio`, `object-fit`, `object-position`, `overflow-block`, `overflow-inline`, `shape-outside`, `shape-margin`, `float-offset`, `float-reference`, `counter-set`.

`testdata/golden/fixture-64-next-72-props.html` is the audit page. The needle is `NEXT-72-PROPS`.

Still unsupported, on purpose (19):

- 14 Borders-4 / Round Display drafts: `border-clip` and the physical and logical clip longhands, `border-limit`, `border-shape`, `border-boundary`.
- `text-fit`, `shape-inside`, `shape-padding`, `shape-image-threshold`, `float-defer`.

The per-property rows are `documentation/compatibility-matrix.md` sections 2.3 and 5.5, and `plans/0.2.7/next-72-properties.json`.

#### 2. Parse, style, and lay out without a PDF (PR [#81](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/81))

A second Go module can use the same pipeline the PDF writer uses, and stop before pagination.

- `html.Parse` (`html/parse.go`) keeps the engine tree. `Find` returns an element by id.
- `css.Parse` parses one stylesheet. `css.Apply` (`css/css.go`) collects `<style>` from that document and appends any extra sheets. Linked stylesheets and images are not fetched.
- `layout.Lay` (`layout/layout.go`) places that document and paints one image. Boxes are CSS pixels, y down, origin at the top left. Pagination and PDF writing are not called.
- `screen.Render` (`screen/screen.go`) is those three calls plus a PNG encode. The frame also carries each element's rectangle, so a host can tell which element a click landed on. The media type for this call is `screen`.
- `markup.Parse` (`markup/markup.go`) returns a detached copy. `css.Apply` does not accept that copy. Script text stays text. Nothing in these packages runs it.

`Document.PDF` / `ImageDocument` are unchanged. `LibraryVersion` stays `0.12.7-dev` (the wkhtmltopdf settings id, not this release).

#### 3. Flex interaction cases and paint fixes (PR [#76](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/76))

`test/chrome/` holds 40 cases with source-shaped HTML. Cases 1-20 and 21-40 are closed in `plans/0.2.7/chrome-flex-interactions/`. Direct layout cases assert box positions. Print cases use PDF page checks.

Fixes that came out of those cases:

- Centered auto-width items in a column flex container line up on the container center.
- A definite stretched cross size is the used size.
- Stale deferred chrome is dropped from flex measurement. Container chrome counts in the flex intrinsic width.
- A specified height, and a transformed flex frame, stay out of the chrome stretch.
- A measure pass does not leave floats behind.
- Vertical-writing column items center on the cross axis.
- Transparent borders are not painted. Dashed and dotted border fragments are centered.
- A repeated `thead` band from an earlier page does not stay open on the continuation page.
- A range input paints a thumb. An input button paints its label.

Warm allocated bytes for one pass of all 40 cases fell from 64.3 MB to 28.0 MB (70,091 allocs to 63,699). The plan is `plans/performance/2026-09-21/00-canonical-chrome-case-alloc-reduction.md`. Wall time on that small harness moved around with the host, so this release does not claim a speedup there. The 500-page numbers in the README are still the 2026-09-13 capture of the 0.2.6 binary.

#### 4. Link decoration (PR [#74](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/74))

`paintDecoration` used to draw an underline under every `a[href]`, including when the author set `text-decoration: none`. The underline now comes from the resolved style. Operators who want the old forced rule pass `--print-link-underline`. That flag stays off unless set.

#### 5. Fixture checks, samples, and the site (PRs [#76](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/76), [#77](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/77), [#78](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/78), [#79](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/79))

- PDF operation checks cover page boxes, text, strokes, fills, and image placements for the numbered fixtures (`internal/pdf/page_ops.go`, `internal/convert/fixturetests/`).
- `output/validated/` holds 65 reference PDFs (fixtures 01-64, with two body files for fixture 29). `TestVisualReferenceInventory` rejects a missing or extra file. `TestVisualGoldenFixtureCorpus` renders each page with Ghostscript 10.08.0 (`png16m`, 150 DPI, smoothing 4) and compares pixels. The test skips unless `GOWKHTMLTOPDF_VISUAL_GS` is set. The pinned full run and the CI job are still open (`plans/0.2.7/validator-test/95-canonical-pixel-regression-tests.md`).
- Equal-priority Unicode aliases in the font cmap choose the lowest code point.
- `documentation/ghostscript-licensing.md` says when an AGPL Ghostscript install matters. The PDF engine does not start Ghostscript. Only the optional visual test does.
- The showcase lists fixtures 59-64 and `fixture-29-wpt-break-nested-float-print`. Fixtures 57 and 58 stay as PDFs under `output/` and are not on the gallery. Fixture 64's card title is `font-feature-settings` through `counter-set`.
- The compatibility page names all 72 next-72 properties: 53 implemented, 19 not implemented.

#### 6. What this release does not include

- Next-100 (`plans/0.2.7/88-canonical-0.2.7-next-100.md`, fixture-66) is written as a plan. No next-100 property batch shipped.
- No new CLI flag and no new direct dependency. The allowlist is still `go-text/typesetting` and `tdewolff/canvas`.
- No JavaScript. No Chrome or WebKit print parity claim.
- The 2026-09-13 benchmark table in the README is a measurement of the 0.2.6 binary (`8aab63a`). It is not a 0.2.7 remeasure.

---

### Documentation

| Doc | Link |
|------|------|
| **Site** | https://chinmay-sawant.github.io/gowkhtmltopdf/ |
| **Overview** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/documentation/overview.md |
| **Getting started** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/documentation/getting-started.md |
| **CLI** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/documentation/cli.md |
| **Library API** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/documentation/library-api.md |
| **Python** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/documentation/python.md |
| **Browser WASM** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/documentation/wasm.md |
| **Compatibility matrix** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/documentation/compatibility-matrix.md |
| **Ghostscript licensing** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/documentation/ghostscript-licensing.md |
| **Deferred work** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/documentation/deferred.md |
| **Changelog** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/CHANGELOG.md#027-2026-10-01 |
| **0.2.7 plans** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.7/plans/0.2.7/README.md |

---

### Known limits (honest)

- 411 of 818 webref properties stay unsupported. That includes the 19 next-72 names listed above, plus the print no-ops from 0.2.6 (animation, scroll UI, pointer UI, speech, 3D). WOFF2 is still rejected. Fonts load from TTF, OTF, and WOFF1 under the configured ACL.
- `html` / `css` / `layout` / `screen` do not fetch linked stylesheets or images, do not paginate, and do not write a PDF.
- The browser adapter still accepts inline HTML only. Local files, arbitrary remote resources, and document JavaScript stay off.
- The C ABI is still a one-shot inline-HTML interface. Python file and URL sources still raise `NotImplementedError`.
- Python wheels ship for Linux x86_64 / aarch64 (`manylinux_2_28`), macOS arm64, and Windows x86_64 when the `v0.2.7` tag is pushed. Intel macOS wheels are not built.
- Pixel comparison of all 65 references is present and skips without `GOWKHTMLTOPDF_VISUAL_GS`. It is not a required CI gate yet.
