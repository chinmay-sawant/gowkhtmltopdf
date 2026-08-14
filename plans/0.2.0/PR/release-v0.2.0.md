## gowkhtmltopdf v0.2.0 - print CSS subset, multi-font, and a faster report engine

Second public release of **gowkhtmltopdf**: still a **no-cgo**, **no Qt/WebKit**, **no browser** clean-room work-alike of the [wkhtmltopdf](https://wkhtmltopdf.org/) CLI, aimed at **controlled server-generated documents** (invoices, tables, multi-page reports, headers/footers, TOC, PDF outlines).

Since [v0.1.0](https://github.com/chinmay-sawant/gowkhtmltopdf/releases/tag/v0.1.0) the engine closed **Tier 1** (report quality) and **Tier 2** (leave wkhtmltopdf for most report jobs): real multi-face fonts, Type0/CJK/Arabic paths, flex/grid/float/sticky as a **print subset**, repeating table headers, typed library requests, and a measured speedup versus wkhtmltopdf 0.12.6.1.

**Not** a full browser-print engine. Prefer this when you want a static Go binary and MIT licensing for **HTML you control**. Prefer Chrome headless / upstream wkhtmltopdf when you need arbitrary-page or JavaScript fidelity. This release does **not** claim Chrome or Wikipedia visual parity.

- **License:** [MIT](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/LICENSE) - Copyright (c) 2026 Chinmay Sawant
- **Version source:** [`VERSION`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/VERSION) (`0.2.0`)
- **Site:** https://chinmay-sawant.github.io/gowkhtmltopdf/
- **Compare:** https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.1.0...v0.2.0
- **Engine PRs:** [#7](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/7)–[#34](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/34)
- **Wrap-up PRs:** [#36](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/36) release prep · [#37](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/37) docs site · [#38](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/38) CONTRIBUTING

---

### Highlights

| Area | What you get vs 0.1.0 |
|------|------------------------|
| **CLI** | Same `gowkhtmltopdf` + `gowkhtmltoimage` grammar; mode-invalid flags now fail at parse; `--font-path` / `--use-system-fonts`; opt-in `--simplify-dom` |
| **Library** | Prefer `RunPDF` / `PDFRequest` and `RunImage` / `ImageRequest`. `ConvertHTML` one-shot helper. Compatibility `Converter` kept. Settings cloned so later mutation cannot change an in-flight job |
| **Layout** | Report-subset flex, grid, float, `inline-block`, `box-sizing`, print-scoped sticky, repeating `<thead>`, CSS orphans/widows, nested HTML headers/footers |
| **Fonts** | Liberation Sans/Serif/Mono (R/B/I/BI) + DejaVu fallback; Type0/CID for non-Latin; `@font-face` TTF/OTF/WOFF1; OpenType GSUB via allowlisted `go-text/typesetting` |
| **Image mode** | TrueType outline raster with coverage AA (the 0.1.0 5×7 bitmap font is gone) |
| **PDF** | Unique multi-image XObjects; JPEG DCT pass-through; PNG alpha soft-mask; SVG-as-`<img>` via allowlisted `tdewolff/canvas` |
| **Performance** | Faster than wkhtmltopdf 0.12.6.1 at every tested size on the 2026-08-14 snapshot (about **16×** at 2 pages, **1.6×** at 500 pages) |
| **Security defaults** | Local files still **blocked** unless opted in; `<script>` stripped; JS CLI flags are **unknown options** (not silent no-ops) |
| **Ops** | `CGO_ENABLED=0` static builds; golangci-lint v1.64.8; `v*` tags publish 12 binaries + `SHA256SUMS` |
| **Site** | Docs, Issue Dossier (1,329 upstream issues classified), Showcase, Benchmarks |

---

### Install / build

Cross-platform binaries are attached to this release (`gowkhtmltopdf` and `gowkhtmltoimage` for linux / windows / darwin × amd64 / arm64) plus `SHA256SUMS`.

From source (Go 1.26+):

```sh
git clone https://github.com/chinmay-sawant/gowkhtmltopdf.git
cd gowkhtmltopdf
git checkout v0.2.0
CGO_ENABLED=0 go build -ldflags "-X gowkhtmltopdf/internal/cli.Version=0.2.0" \
  -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
CGO_ENABLED=0 go build -ldflags "-X gowkhtmltopdf/internal/cli.Version=0.2.0" \
  -o bin/gowkhtmltoimage ./cmd/gowkhtmltoimage
```

Or `make build`. Convert a committed invoice fixture:

```sh
./bin/gowkhtmltopdf --enable-local-file-access \
  testdata/golden/fixture-01-simple-invoice.html /tmp/invoice.pdf
```

Library (typed API):

```go
var out bytes.Buffer
err := gowkhtmltopdf.RunPDF(ctx, &gowkhtmltopdf.PDFRequest{
    Objects: []*gowkhtmltopdf.ObjectSettings{
        gowkhtmltopdf.NewObjectSettings().SetBody(
            []byte(`<html><body><h1>Invoice</h1></body></html>`), ""),
    },
    Output: &out,
})
```

Sample PDFs live under [`output/`](https://github.com/chinmay-sawant/gowkhtmltopdf/tree/v0.2.0/output) (`make samples`).

---

### What landed since v0.1.0

#### Rendering and typography

- Liberation **Sans / Serif / Mono** Regular, Bold, Italic, BoldItalic, plus **DejaVu Sans** as Unicode fallback. CSS `font-weight` / `font-style` pick real faces; fake bold only if a face is missing.
- Image mode (`gowkhtmltoimage`) rasterizes TrueType outlines with coverage AA (2× supersample, stable baselines) instead of the MVP 5×7 bitmap font. Metrics match PDF/layout.
- `--font-path DIR` (repeatable, depth 2) and `--use-system-fonts`. Type0 / CID Identity-H for runes above U+00FF. Mixed Latin + CJK keeps Liberation for Latin.
- Local and HTTPS `@font-face` for TTF / TrueType-outlined OTF / WOFF1 (PDF **and** image). WOFF2 / `data:` / EOT are skipped on purpose.
- Shaping without cgo HarfBuzz: OpenType GSUB when the face has it (`go-text/typesetting`); Arabic presentation-form + Lam-Alef fallback; optional `halt` / `palt`. Indic remains Partial. `writing-mode: vertical-*` is parsed but lays out **horizontal**.
- Without a capable face, CJK is still tofu. CI ships only a tiny OFL Hangul subset for smoke, not a full CJK family.

#### CSS and invoice / report layout

- Selectors: `[attr]`, `[attr=value]`, `:first-child` / `:last-child` / `:nth-child(odd|even|an+b)`, sibling `+` / `~`, plus `:has()` (simple compounds).
- **Float lite:** `float: left|right` + `clear` for logo/meta chrome. Real `display: inline-block`. Simple `text-align: justify`. Table-cell `vertical-align` top/middle/bottom.
- `box-sizing: content-box` (now the default) and `border-box`. **Migration:** explicit `width` + padding without `box-sizing` grows vs 0.1.0; add `box-sizing: border-box` to keep the old visual size.
- **Flex Stage A** (report subset): `flex` / `inline-flex`, direction including reverse, wrap, grow/shrink/basis/order, gap, justify including `space-around` / `space-evenly`, align-self, stretch, cyclic `%` → auto. Not Flexbox L1 / Chrome parity.
- **Grid Stage B + Stage C lite:** columns/rows, `fr`, `repeat`, `minmax`, span, named areas, dense packing, copy-inherit subgrid (no shared-track sizing), one-axis masonry. Not Grid L1/L3 complete.
- `position: relative | absolute | fixed` lite. `position: sticky` is **print-scoped** (page content box is the scrollport; overflow boxes at offset 0). Not browser sticky scroll.
- Repeating `<thead>` / `table-header-group` (and leading all-`<th>` rows). CSS `orphans` / `widows` plus a geometric fallback. Nested HTML headers/footers as child documents.
- Report-lite extras: multicol (`column-count` / `width` / `gap` / `span` / `fill`), static 2D `transform` + `opacity`, size-only `@container`, print `@media` subset, HTML entity decoding, CSS `background` color token.
- `letter-spacing`, `text-transform`, and `border-radius` survive into the PDF so original static templates need fewer renderer-specific workarounds.

#### Layout and paint correctness

- Block backgrounds and borders paint **under** text. `tr` backgrounds show through transparent cells. `rgba()` fills composite against white.
- Multi-image pages use unique XObjects (`I0`, `I1`, …; headers/footers `HFI0`, …). JPEG bytes pass through as DCTDecode; PNG alpha becomes a soft-mask. `web.images=false` skips image fetch/paint.
- Nested tables keep document order; `%` widths resolve against the **parent** containing block; colspan contributes across spanned columns; cell height is measured at the **final** column width.
- Tables: empty/padding-only rows collapsed; per-row border-collapse (no phantom empty bands); rowspan cite cells with `<br>` spread vertically; continuation-page fragments seal under repeated thead.
- Long tokens / URLs honor `overflow-wrap` / `word-break` (with inheritance) and emergency wrap. Float tails that fit one full-width line clear below the float.
- Link underlines coalesce on a line and skip bare URL strings in reference lists.
- Pagination: `page-break-before: always` lands at next-page top; multi-section reports paginate 1:1 (50 sections are 50 pages, not 43); `preferSplitOverBlank` for short `page-break-inside: avoid` boxes; **document-global gap packing that interleaved body and reference text is gone**.
- `display: flex` / `grid` restored after lint adoption. Sticky chrome no longer clones like `position: fixed`. Dashed and `border-left` segments no longer stretch into solid stubs.

#### Library and CLI

- New `ConvertHTML(ctx, html, global)` one-shot helper (in-memory HTML → PDF bytes). Does **not** relax the local-file ACL.
- Typed `PDFRequest` / `RunPDF` and `ImageRequest` / `RunImage`, plus `PdfGlobalOptions`. Existing `Converter` / string settings stay.
- `GuessURL` accepts an `inline:` prefix.
- Settings, inline HTML, headers/footers, and allowlists are copied at ownership boundaries.
- Context cancellation is carried through layout, header/footer paint, and rasterization.
- PDF-only / image-only flags are rejected at CLI parse. Restricted network dial pinning plus matching flags.
- `--dump-outline` needs a distinct outline sink; multiplexed outline XML + PDF on the same stdout is rejected.
- Opt-in `--simplify-dom` / `--no-simplify-dom` chrome-strip (default **off**) for exploratory URL / Phase 21 work. Not applied to invoice HTML unless asked.

#### Performance (snapshot, not an SLA)

Fresh generic CLI **0.2.0** versus installed **wkhtmltopdf 0.12.6.1 (patched Qt)** on Linux amd64, 13th Gen Intel Core i7-13700HX. Same report fixture, `--quiet --enable-local-file-access`, median of three process runs after one warmup. Source: [`documentation/performance.md`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/performance.md).

| Pages | gowkhtmltopdf | wkhtmltopdf | Faster by |
|------:|--------------:|------------:|----------:|
| 2 | 16 ms | 254 ms | **16×** |
| 10 | 30 ms | 278 ms | **9.4×** |
| 100 | 184 ms | 530 ms | **2.9×** |
| 500 | 1.045 s | 1.641 s | **1.6×** |

Faster at every tested size. Peak RSS is **lower through 100 pages** and **higher from 200 pages** on this generic path. Reproduce with `make bench-cli-compare` and `make bench`.

Page islands (`convert.NewBenchmarkPDFRequest`) are an **internal benchmark opt-in**, not a user CLI or library mode. Do not quote island-era RSS as the current product claim.

#### Documentation site, dossier, and samples

- Product site: https://chinmay-sawant.github.io/gowkhtmltopdf/ — Overview, Getting Started, sidebar docs (`/#/documentation/cli` and siblings), Issue Dossier, Showcase, Benchmarks. Light/dark theme, command palette (`⌘K` / `Ctrl+K`).
- **Issue dossier:** all **1,329** open `wkhtmltopdf/wkhtmltopdf` issues classified against this engine (implemented / partial / not implemented) with filterable cards and cited code paths. Verdicts are a starting point, not a formal audit. Site copy at cut: **451 / 285 / 593**.
- **Showcase:** page-flipping gallery of committed `output/` PDFs, Open PDF / View template links, keyboard lightbox.
- New golden fixtures **21–56**, including the FY2024 detailed report, float-lite invoice chrome, thead / flex / CJK / sticky / nested-HF cases, business docs 44–48, print-story posters and storybooks 49–55, and the 20-page architecture diagram (56).
- User docs rewritten from a source scan: fidelity guide, fonts guide, performance snapshots, comparison with [SebastiaanKlippert/go-wkhtmltopdf](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/comparison-with-others/sebastiaanklippert-go-wkhtmltopdf.md), 2026 landscape note, and package-level notes under `documentation/architecture/`.
- Implementation ledgers split into [`plans/0.1.0/`](https://github.com/chinmay-sawant/gowkhtmltopdf/tree/v0.2.0/plans/0.1.0) (MVP) and [`plans/0.2.0/`](https://github.com/chinmay-sawant/gowkhtmltopdf/tree/v0.2.0/plans/0.2.0) (post-MVP).
- `CONTRIBUTING.md` is now the GitHub-recognized contributing guide.

---

### Documentation

| Doc | Link |
|------|------|
| **Site** | https://chinmay-sawant.github.io/gowkhtmltopdf/ |
| **Overview** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/overview.md |
| **Getting started** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/getting-started.md |
| **Architecture** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/architecture.md |
| **Architecture deep-dives** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/architecture/README.md |
| **CLI** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/cli.md |
| **Library API** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/library-api.md |
| **Fidelity** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/fidelity.md |
| **Fonts** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/fonts.md |
| **Performance** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/performance.md |
| **Samples** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/samples.md |
| **Compatibility matrix** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/compatibility-matrix.md |
| **Deferred** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/deferred.md |
| **Threat model** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/THREAT-MODEL.md |
| **Integration security** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/integration-security.md |
| **Comparison (go-wkhtmltopdf wrapper)** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/comparison-with-others/sebastiaanklippert-go-wkhtmltopdf.md |
| **Contributing** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/CONTRIBUTING.md |
| **Changelog** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/CHANGELOG.md#020-2026-08-14 |
| **Post-MVP ledger** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/plans/0.2.0/10-canonical-post-mvp-roadmap.md |

---

### Known limitations (honest 0.2.0)

- **Not a browser.** Flex/grid/float/sticky are a **report subset**, not full CSS3. Arbitrary websites (Wikipedia chrome, marketing SPAs) are exploratory, not a pass criterion.
- **No JavaScript.** `<script>` is stripped. `--enable-javascript` and related flags are **unknown options**.
- **Fonts.** No bundled Noto CJK. WOFF2 unsupported. Indic is Partial. Vertical writing-mode is not implemented.
- **PDF versions / compliance.** No PDF 1.7 / 2.0 / UA-2 / A-4 / encryption / AcroForm. Tickets [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)–[#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33) remain open.
- Full deferred list: [documentation/deferred.md](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/documentation/deferred.md).

---

### Breaking changes / migration

| Item | Migration |
|------|-----------|
| `box-sizing` default is now `content-box` | Add `box-sizing: border-box` if you relied on 0.1.0’s effective border-box sizing |
| JS / unused wkhtml flags | Flags that were silent no-ops in 0.1.0 may now be **unknown options** |
| Outline + PDF on the same stdout | Write outline and PDF to distinct sinks |
| Invalid global PDF options | Surface as validation / preflight errors (`OnError` when set) |
| Direct modules | `go-text/typesetting` and `tdewolff/canvas` are now allowlisted; still `CGO_ENABLED=0` |

Ordinary local golden → PDF converts need no API change. Re-run `make samples` / the golden suite after upgrading if you snapshot PDFs.

---

### Verify

```sh
make test
make lint
make golden
CGO_ENABLED=0 go build ./cmd/gowkhtmltopdf ./cmd/gowkhtmltoimage
```

Optional: `make bench-cli-compare`.

**Full changelog for 0.2.0:** [CHANGELOG.md](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.0/CHANGELOG.md#020-2026-08-14)

---

## What's Changed

* feat(render): multi-font, TTF image raster, CSS selectors, layout fidelity by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/7
* fix(layout/pdf): backgrounds under text; unique multi-image XObjects by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/8
* fix(layout): tr row backgrounds, rgba composite, nested table document order by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/9
* fix(layout): logo gap, nested table width, block margins, letterhead padding by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/10
* docs: comparison with SebastiaanKlippert/go-wkhtmltopdf by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/11
* samples: add fixture-21 detailed report by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/12
* docs: fidelity guide + phase checklist reconcile by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/13
* docs(plans): mark phase 10 complete in post-MVP ledger by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/14
* feat(layout): close Tier 1 remainder (float lite, images, ConvertHTML) by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/15
* feat(tier-2): phases 18→20→17→19 + visual QA (flex/CJK/HF/grid) by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/16
* feat(tier-2-pending): entities, flex/grid, stdlib CJK/Arabic, PDF subset fidelity by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/17
* feat(tier-2-pending-2): sticky print, image @font-face, shaping, HF polish by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/18
* feat(layout): close flex-grid Stage A/B and Stage C lite by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/19
* feat(tier-2): close pending-3 leftovers and kick off Phase 21 by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/20
* fix(layout): print overflow, tables, refs, and gap-pack overlap fix by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/21
* feat(architecture): deepen conversion contracts and validation by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/22
* perf(benchmarks): add conversion matrix and stable table pagination by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/23
* perf(layout): optimize rendering and pagination correctness by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/24
* fix(layout): restore flex/grid and pagination after lint adoption by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/25
* perf(engine): integrate architecture and performance refactors by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/26
* feat(docs): add wkhtmltopdf issue dossier site with showcase by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/27
* feat(showcase): site polish, docs IA, and golden print fixtures 49–53 by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/28
* fix(renderer): improve original template rendering fidelity by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/34
* chore(release): harden architecture, layout fidelity, docs, and showcase by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/36
* chore(frontend): documentation site UI/UX overhaul and plans versioning by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/37
* chore: surface CONTRIBUTING.md and exclude JS from language stats by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/38

**Full Changelog**: https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.1.0...v0.2.0
