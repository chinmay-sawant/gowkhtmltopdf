# Deferred features and workload priority

This page is the product-facing inventory of work that is **partial**,
**not implemented**, or **not planned**. The live post-MVP ledger is
[`plans/0.2.0/10-canonical-post-mvp-roadmap.md`](../plans/0.2.0/10-canonical-post-mvp-roadmap.md).
Leftover print CSS is tracked in
[`plans/0.2.6/48-canonical-0.2.6-css-coverage.md`](../plans/0.2.6/48-canonical-0.2.6-css-coverage.md)
(phases 48-53 landed; 54-56 closing). Fidelity language and degrade rules:
[fidelity.md](fidelity.md). The normative per-property contract is
[compatibility-matrix.md](compatibility-matrix.md).

Status here is checked against current code (CLI flag registration, load,
layout, and PDF write). Stale “accepted + warning” wording elsewhere loses
to this table when they disagree.

---

## Workload priority

The dominant practical workload for an HTML-to-PDF tool is backend-generated
business documents, not arbitrary public websites:

```text
application data
  -> server-side HTML template
  -> HTML/CSS
  -> PDF
```

Typical documents are invoices, receipts, reports, statements, purchase
orders, contracts, certificates, and shipping documents. Integrations in
Python, PHP, Ruby, and Go commonly expose both HTML/string/file input and
URL input. The template path is usually the primary path because it avoids
an HTTP loopback, authentication and cookie problems, and makes assets and
testing more predictable.

URL input is still valuable when it points to a server-rendered internal
page, such as `/orders/123/print`: it reuses an existing web view, CSS,
data loading, and localisation. It should not be conflated with a
client-rendered SPA URL. A dossier URL whose initial HTML is an empty root
element, with React constructing the document in JavaScript, is a
lower-frequency browser-rendering workload and is not representative of the
core invoice or report path.

Highest-impact order for gowkhtmltopdf:

1. **HTML strings** (in-memory / library `InlineHTML` and similar)
2. **Local HTML template files**
3. **Server-rendered internal URLs**
4. **JavaScript-heavy SPA URLs**

The core product should prioritise tables, pagination, headers and footers,
images, fonts, CSS layout, page breaks, and repeated sections. SPA execution
should remain a separate capability rather than the definition of the main
HTML-to-PDF workload.

String input via the library is the first-class path for (1). CLI stdin
HTML (`-` as the page input) is **not** implemented today — see the table.
Do not treat a public SPA URL as the acceptance bar for report work.

---

## Deferred inventory

| Item | Status / reason | Next gate |
|------|-----------------|-----------|
| JavaScript / `--enable-javascript` | Flags **not registered** (unknown option). `<script>` is stripped at load; scripts are **not** executed. Related stubs (`--javascript-delay`, `--run-script`, `--window-status`, `--debug-javascript`, `--enable-plugins`) are also unknown. | Phase 22 |
| Full CSS / Chrome print parity | Not a product goal for this PDF engine based on HTML templates (without any wrappers). | Phase 23 deferred |
| Full flex / grid / subgrid / masonry | **Partial** print CSS subset (fixtures 25/28/32–35). Joint subgrid intrinsic sizing and CSS Grid L3 masonry are out. | — |
| Chrome sticky scroll parity | Print-scoped sticky: page content box is the scrollport; overflow boxes are scrollports at **offset 0**. No continuous scroll, no Chrome pixel match. | Non-goal |
| CJK / complex scripts | Type0/CID + `--font-path`; Arabic OpenType (GSUB) with presentation-form fallback; Indic **Partial**. `writing-mode: vertical-rl` / `vertical-lr` are parsed but **not** implemented (horizontal layout only). | No CGO HarfBuzz |
| AcroForm / `--enable-forms` | No form model in the PDF writer. `--produce-forms` is not a working forms path. | Intermediate roadmap |
| XSLT TOC (`--xsl-style-sheet`) | Flag is accepted and **warns**; the built-in Go-template TOC is used. | Not planned |
| SVG image **output** (`--format svg`) | Image mode encodes PNG/JPEG only. | Not planned |
| BMP output | No demand; PNG/JPEG cover `image/*`. | Not planned |
| SOCKS5 proxy | `parseProxy` accepts `http` / `https` only. | Not planned |
| PDF 1.7 + PDF/A-3a / PDF/UA-1 | **Shipped in 0.2.2** (#31, [#45](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/45)/[#47](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/47)): opt-in `--pdf-version 1.7` / `Document.PDFVersion = "1.7"` is a version, not a claim. Profiles `--pdf-profile a3a-ua1` / `a3a` / `ua1` (or `Document.PDFProfile`) imply 1.7. Tagged lists nest `L` → `LI` → `LBody` → `Link`. | #31 done |
| PDF 2.0 (ISO 32000-2) | **Shipped in 0.2.2** (#32, [#46](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/46)): opt-in version via `--pdf-version 2.0` / `Document.PDFVersion = "2.0"` — header, trailer `/ID`, UTF-8 document strings, non-claiming XMP. Version alone is **not** a PDF/A or PDF/UA claim. | #32 done |
| PDF/A-4 / PDF/UA-2 (PDF 2.0 conformance profiles) | **Shipped in 0.2.2** (#33, [#46](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/46)/[#47](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/47)): opt-in via `--pdf-profile a4-ua2` / `Document.PDFProfile` (also `a4`, `ua2`). Implies PDF 2.0. Emits claiming XMP (`pdfaid:part=4`, `pdfuaid:part=2`), OutputIntent, structure namespaces, and full tagging. | #33 done |
| PDF encryption / AcroForm / signatures | Out of scope; no writer support (rejected on every version, incl. 2.0). | Not planned |
| C ABI (`gowkhtmltopdf_*` c-shared exports) | **Shipped in 0.2.5**: opt-in c-shared exports under `bindings/c` (`-buildmode=c-shared`) power the Python bindings; see [python.md](python.md). Default Go builds stay `CGO_ENABLED=0`; cgo lives only in the isolated shared-library target. | Python bindings track, `plans/0.2.5/` |
| `--read-args-from-stdin` | **Not implemented.** The flag is not a working batch loop (rejected / unused). | Not planned |
| Stdin HTML input (`-`) | **Not implemented.** CLI parse stores `Page: "-"`, but `load.GuessURL("-")` falls through to **`http://-`**. Library callers should pass inline HTML; do not document CLI `-` as stdin. | Document honestly; not a hidden feature |
| WOFF2 / `data:` `@font-face` | Skipped (WOFF2 needs Brotli, not allowlisted; `data:` src rejected). Local TTF/OTF/WOFF1 under ACL works. | No Brotli module |
| `[subject]` placeholder | Expands **empty** (no subject setting field). | Not planned |
| HTML header / footer | **Partial** nested child layout (body CSS subset, flex/grid/images, local `@font-face`), clipped to the reserved margin band. Not a browser nested browsing context; no CSS running elements. | Browser HF out |
| `:hover` / `:focus` / `:active` | Parsed onto the compound; `matchPseudo` **never matches** (print has no pointer/focus). | — |
| `table-layout: fixed` | **Partial** (fixed lite): consumed when `fixed` and table width is definite (`layout_tables.go:45`). Content max-content ignored. | Matrix §2.5 |
| `@page size` | **Consumed** via `applyCSSPageMargins` (EXT-04); unnamed `@page margin` likewise. | #EXT-04 done |
| `background-image` / gradients | **Partial** first `url(...)` layer (`background_image.go`); gradients still ignored. | Phase 52 |
| Overflow clip | **Partial**: `hidden`/`clip`/`auto`/`scroll` clip descendant paint to the padding box (`overflow_clip.go`). Sticky still uses overflow as scrollport at offset 0. | Phase 52 |
| Leftover print CSS | Active ledger: [`plans/0.2.6/48-canonical-0.2.6-css-coverage.md`](../plans/0.2.6/48-canonical-0.2.6-css-coverage.md). Do not treat `plans/0.2.0/phases/pending-phase-items/` as the live CSS list. | 0.2.6 leftovers: float wrap, duplex size, GCPM |
| `box-shadow` | **Partial** un-inset offset fill plus lite stacked-rect blur (`box_shadow.go`). Inset and spread ignored. | 52.5 `[x]` |
| `list-style-image` | Paints via the img fetch path; type marker fallback. | 53.4 `[x]` |
| `@page :first` / `:left` / `:right` | Margins applied. LTR page 1 is `:right`; `:first` wins on page 1. Size unnamed-only. | 54.1.2 `[x]` |
| `page: ident` | Used-value inherit; sibling name change breaks; named `@page` margin on overlapping pages. No per-page size. | 54.1.3 `[x]` lite |
| `@page` margin boxes (`@top-center`) | Unnamed quoted `@top-*` / `@bottom-*` fill empty CLI header/footer slots. `running()` / corners out. | 54.3 `[x]` lite |

---

## How to use this list

- A row marked **Phase 22** or **Phase 23** is a ledger item, not a shipped
  date. Phase 23 (open-web / browser competition) stays deferred unless the
  roadmap is amended.
- **Not planned** means there is no active design; a future amendment would
  be required.
- **Partial** means a report-shaped subset exists; Chrome/layout-test
  parity is still out.

When behavior claims change, update this table together with
[fidelity.md](fidelity.md) and [compatibility-matrix.md](compatibility-matrix.md).
