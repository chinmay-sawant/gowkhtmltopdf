# Samples and golden fixtures

This page is the operator map for the HTML corpus, the committed PDFs under
[`output/`](../output/), and the two Makefile targets that keep them honest.

Related:

- Fixture inventory and pass criteria: [`testdata/golden/README.md`](../testdata/golden/README.md)
- Product claims and degrade rules: [fidelity.md](fidelity.md)
- URL recipes (raw smoke vs chrome-strip): [cli.md — URL mode](cli.md#url-mode--chrome-strip---simplify-dom)
- Contributor setup, visual QA, PR expectations: [`CONTRIBUTING.md`](../CONTRIBUTING.md)

---

## What `make golden` proves

```sh
make golden
# equivalent: go test ./internal/convert/ -run 'TestGoldenCorpus' -v
```

That target runs `TestGoldenCorpus` (fixtures 01–03 plus the fixture-03
layout/paint budget) **and** `TestGoldenCorpusAllFixtures` (every
`testdata/golden/*.html` body fixture). A fixture passes when the generated
PDF has:

1. **Structure** — `%PDF-` header, `%%EOF` trailer, reachable `xref`, and an
   embedded `/FontFile2` subset.
2. **Page envelope** — page count inside the per-fixture bounds in
   `internal/convert/golden_test.go` (`fixturePageBounds`). A missing map
   key is a fail. `maxPages == 0` means “no upper bound”.
3. **Needles** — selected fixtures assert ordered extracted text via
   `pdf.ParseSemantic` (01 Invoice / total, 06 Partner Handbook, 07 Nordwind,
   24 Internal link report, 54 Ember Harbor, 55 Northline).
4. **Features** — `images` / `uris` flags require `/Subtype /Image` or
   `/S /URI`.

Golden comparison is **structural and semantic**. It is **not** a binary PDF
equality check and not a ±1 px image-diff of every box. Font subsets, stream
filters, and wall-clock metadata can change across rebuilds. Viewer quality
(overlap, table chrome, underline noise) is a **visual** check — see
[Visual QA](#visual-qa-checklist) below.

`make golden-update` is deliberately narrow: one body-fixture basename, an
explicit `GOLDEN_APPROVE=1` acknowledgement, and a write only to the
gitignored `testdata/golden/out/` directory. It never rewrites committed
HTML/CSS/PNG fixtures.

```sh
make golden-update GOLDEN_FIXTURE=fixture-01-simple-invoice.html GOLDEN_APPROVE=1
```

---

## Golden HTML corpus (`testdata/golden/`)

**Body fixtures:** `fixture-01` … `fixture-56`. Companion files matching
`*-header.html` / `*-footer.html` are **not** converted as bodies.

When `fixture-NN-header.html` and/or `fixture-NN-footer.html` sit beside a
body fixture, both `make golden` (`attachHFCompanions`) and `make samples`
set `--header-html` / `--footer-html` and auto (`-1`) margins. Today that
applies to **fixture-36** (`fixture-36-header.html`, `fixture-36-footer.html`).

### Extra HTML (golden-tested, not `make samples`)

These live next to the numbered fixtures and **are** walked by
`TestGoldenCorpusAllFixtures`, but `make samples` only converts
`fixture-*.html`. They are **not** rewritten into `output/` unless they
happen to share a `fixture-*` name.

| File | Role | Envelope |
|------|------|----------|
| [`font-examples.html`](../testdata/golden/font-examples.html) | Font-path showcase (faces not bundled; `--font-path` / `fontpath`) | 1–30 pages |
| [`complex-css.html`](../testdata/golden/complex-css.html) | Large CSS surface / catalog stress (needle `Alexandria`) | 1–40 pages |
| [`architecture-diagram.html`](../testdata/golden/architecture-diagram.html) | Corpus fixture for the 5-page library-API architecture (needle `Architecture`). Distinct from `testdata/golden/api/architecture-diagram.html` and from fixture-56. | 1–12 pages |

`testdata/golden/api/` holds the library-API architecture generator
(`go run ./testdata/golden/api`). It is not part of the `*.html` walk at the
corpus root. `make samples` runs it to refresh `output/architecture-diagram.pdf`
only — it does not rewrite the corpus fixture above or write a PDF under
`testdata/golden/`.

### Fixture groups

| Group | Fixtures | Theme |
|-------|----------|-------|
| Core reports | 01–21 | Invoices, tables, page breaks, linked CSS, images, typography, box model |
| Layout lite | 22–31 | Float chrome, thead repeat, internal GoTo, flex/position lite, orphans heuristic, sticky print |
| Flex / grid / CSS lite | 32–42 | Flex/grid stages, nested HTML HF, CSS orphans/widows, float-in-`td`, multicol, transform, `:has()`, `@container` |
| Business documents | 43–48 | Dossier, receipt, PO, contract, certificate, shipping |
| Illustrated print | 49–54 | Poster, letter, storybooks, boarding pass, observatory poster, Ember Harbor |
| Long-form | 55–56 | Operations brief, 20-page architecture diagram |

### Fixture inventory

Page envelopes come from `fixturePageBounds`. “What it exercises” matches the
fixture header comments and [`testdata/golden/README.md`](../testdata/golden/README.md).

| Fixture | What it exercises | Pages |
|---------|-------------------|-------|
| 01 | Simple invoice: `div`, `h1`, `p`, `table`, `span`, bold total | 1 |
| 02 | Table-heavy: 15-row table, thead/tbody/tfoot, colspan subtotal, borders | 1–2 |
| 03 | Multi-page statement: `page-break-before/after/inside`, outline headings | ≥2 |
| 04 | Two-column layout via fixed table cells (float lite also in fixture-22) | 1 |
| 05 | Linked relative stylesheet (`style-05.css`) driving all styling | 1 |
| 06 | External `<a href>` → PDF URI annotations + linked stylesheet | 1 |
| 07 | Relative `logo.png` + data: URI PNG, intrinsic sizes | 1 |
| 08 | Forced page breaks: before/inside/after + margin-gap convergence | 5 |
| 09 | Long multi-section report, natural pagination, outline fodder | ≥2 |
| 10 | Nested tables, `colspan` (2 and 4), th, tfoot totals | 1 |
| 11 | Paragraph-heavy justified prose, line-level pagination | ≥3 |
| 12 | Nested ul/ol/li with indentation and list-item styles | 1 |
| 13 | `pre` white-space preservation, `code` runs, log excerpts | 1 |
| 14 | Background colors, rgba() alpha, colored borders, hr | 1 |
| 15 | Markdown-ish bulleted requirements, nested lists, anchors | 1–2 |
| 16 | Full invoice: letterhead, bill-to, 24 line items, tfoot totals | 1–2 |
| 17 | Cover-style first page + `page-break-before` content page | 2 |
| 18 | Typography: h1–h6, strong/em/u/s/small, blockquote, code | 1 |
| 19 | Box model: fixed/min/max widths, margins, padding, borders | 1 |
| 20 | Image grid: four data: URI PNGs at intrinsic sizes | 1 |
| 21 | Detailed multi-section ops report (KPIs, WPs, invoice extract, REQs) | ≥3 |
| 22 | Float lite invoice chrome: float left/right + clear, inline-block badge, border-box | 1 |
| 23 | Multi-page table with repeating `<thead>` on continuation pages | ≥2 |
| 24 | Same-document `<a href="#id">` → PDF GoTo annotations | 2 |
| 25 | Partial flex row: justify-content, gap, flex-grow | 1 |
| 26 | Position lite: relative offsets + absolute overlay | 1 |
| 27 | CJK/Unicode sample (pair with `--font-path` for real glyphs) | 1 |
| 28 | flex-wrap, CSS grid lite, position:fixed stamp | 2 |
| 29 | Float beside table: float:right infobox + wrapping prose + clear | 1 |
| 30 | Orphans/keep-with-next heuristic sample (geometric fallback) | ≥2 |
| 31 | Print-scoped `position: sticky` (page content box = scrollport) | ≥2 |
| 32 | Flex Stage A + Grid Stage B lite: reverse, space-evenly, gaps, column flex, template-rows, row span | 1 |
| 33 | Flex `%` basis cyclic honesty: definite vs indefinite main size (cyclic → auto) | 1 |
| 34 | Grid `template-areas` / `grid-area` names + `grid-auto-flow: dense` | 1 |
| 35 | Grid `minmax` / intrinsic measure lite + subgrid copy-inherit + masonry pack | 1 |
| 36 | Nested HTML HF: flex header + image + `#target` GoTo; placeholder footer (companions `fixture-36-header.html` / `fixture-36-footer.html`) | 1 |
| 37 | CSS `orphans`/`widows` parse + Rule 3 keep-together | ≥2 |
| 38 | Float inside `td`: icon float + wrap + clear; table after float clears below | 1 |
| 39 | CSS multicol lite: `column-count`/`gap`/`span`/`fill`; column boxes stay on one page | ≥2 |
| 40 | Static 2D CSS `transform` badge (rotate) + abspos CB under transform | 1 |
| 41 | `:has()` relational selector: article footnote border, `tr:has(td.neg)` row highlight | 1 |
| 42 | `@container` size lite: named `inline-size` + unnamed `width` queries | 1 |
| 43 | Self-contained five-page dossier: embedded images, flex/grid cards, floats, multicol prose, anchors, `:has()`, repeating tables, code, positioning, forms, print styling | 5 |
| 44 | Receipt: compact paid transaction with line items, tax, and total | 1 |
| 45 | Purchase order: supplier/delivery blocks, item table, terms, and approvals | 1 |
| 46 | Contract: numbered clauses, obligations, commercial terms, and signatures | 1 |
| 47 | Certificate: ornamental background image, centered award composition, and signatures | 1 |
| 48 | Shipping document: addresses, package table, tracking, and handling instructions | 1 |
| 49 | Night train poster: generated art, free generic fonts, layered composition | 1 |
| 50 | Letter template: local mark, stationery hierarchy, quote, and signature | 1 |
| 51 | Original Asteria storybook: page illustrations, live text, and page breaks | 4 |
| 52 | Airline boarding pass: e-ticket itinerary, multi-column stubs, mono barcodes | 1 |
| 53 | Asteria poster variant: shared theme with a different illustration and copy | 1 |
| 54 | Ember Harbor storybook: cover + three chapter pages, shared print theme, local illustrations | 4 |
| 55 | Self-contained operations brief: inline CSS, status cards, route table, action plan, and page breaks | 3 |
| 56 | Architecture diagram: hero, pipeline strip, TOC, 10 domain sections (modern semantic tags and CSS with graceful-degrade fallbacks), linked `fixture-56-architecture-diagram.css` | 20 |

Supporting assets in the same directory: `logo.png`, `certificate-background.jpg`,
`style-05.css`, `theme-print-stories.css`, `fixture-56-architecture-diagram.css`,
and generated artwork under `assets/`.

---

## Vendored web fixtures (`testdata/web/`)

Static HTML used by Phase 21 acceptance tests. `make test` does **not** hit
the live network.

| File | Role |
|------|------|
| `testdata/web/wiki-like-article.html` | Wiki-like article (title **Ana de Armas**); nav chrome present but `display:none` |
| `testdata/web/marketing-landing.html` | Marketing landing with hero + primary CTA |

```sh
go test ./internal/convert -run 'TestWeb(Wiki|Marketing)FixtureAcceptance' -count=1
```

Live Wikipedia remains optional smoke (`make samples`), not a CI gate. See
[fidelity.md — Arbitrary websites](fidelity.md#arbitrary-websites-phase-21)
and [cli.md — URL mode](cli.md#url-mode--chrome-strip---simplify-dom).

---

## Committed samples (`output/`)

Regenerate:

```sh
make samples
```

`make samples` rebuilds the binaries via `go run` and:

1. Deletes regenerable `output/fixture-*.pdf`, `output/fixture-*.png`, and
   `output/showcase-*.pdf`.
2. Converts every `testdata/golden/fixture-*.html` **except** `*-header.html`
   / `*-footer.html` to `output/<basename>.pdf`, attaching HF companions when
   present. Optional `--font-path` is added when
   `/usr/share/fonts/truetype/droid` or `testdata/fonts` exists (fixture-27
   and named families).
3. Writes `output/showcase-toc-hf-outline.pdf` from fixture-16 with `--toc`,
   text headers/footers, and `--outline`.
4. Writes two PNGs:
   - `output/fixture-01-simple-invoice.png` via `gowkhtmltoimage`
   - `output/fixture-21-detailed-report.png` via `examples/image`
5. Runs `go run ./testdata/golden/api` to refresh
   `output/architecture-diagram.pdf` only. It does **not** rewrite
   `testdata/golden/architecture-diagram.html` or write a PDF under
   `testdata/golden/api/`.
6. Writes version / compliance smokes (0.2.2) for
   `fixture-21-detailed-report` and `fixture-56-architecture-diagram` into
   `output/pdf-1.7/` (`--pdf-version 1.7`), `output/pdf-1.7-compliance/`
   (`--pdf-profile a3a-ua1`), `output/pdf-2.0/` (`--pdf-version 2.0`), and
   `output/pdf-2.0-compliance/` (`--pdf-profile a4-ua2`).
7. Optionally refreshes `output/wiki-ana-de-armas.pdf` from the live
   Wikipedia URL **without** `--simplify-dom`, with `--use-system-fonts` and
   `--zoom 0.666667`. Network is required; **soft-fail** if the fetch fails
   so offline hosts still get fixture samples.

### Opt-in metric-alias sample (`make samples-metric-aliases`)

Separate from `make samples`. Regenerates only:

`output/fixture-55-lantern-cooperative-report-metric-aliases.pdf`

with `--use-system-fonts --use-metric-font-aliases` (Gelasio/Cousine when
those faces are discoverable). Does **not** rewrite the default
`output/fixture-55-lantern-cooperative-report.pdf`.

```sh
make samples-metric-aliases
```

`font-examples.html` and `complex-css.html` are **not** part of the fixture
loop. Leftover or manual PDFs that may sit in `output/`
(`font-examples.pdf`, `chrome_ana.pdf`, `wkhtmltopdf/`) are documented in
[`output/README.md`](../output/README.md).

These files are **viewer smoke artifacts**, not golden byte masters. CI
asserts structure through `make golden`, not binary equality against
`output/`.

### Version and compliance folders

Part of `make samples`. Opt-in PDF version and profile flags write four
sibling dirs (same two fixtures: `fixture-21-detailed-report`,
`fixture-56-architecture-diagram`). Inventory: [`output/README.md`](../output/README.md).

| Dir | Produced with | Meaning |
|-----|---------------|---------|
| [`output/pdf-1.7/`](../output/pdf-1.7/) | `--pdf-version 1.7` | Unclaimed PDF 1.7 |
| [`output/pdf-1.7-compliance/`](../output/pdf-1.7-compliance/) | `--pdf-profile a3a-ua1` | PDF/A-3a + PDF/UA-1 |
| [`output/pdf-2.0/`](../output/pdf-2.0/) | `--pdf-version 2.0` | Unclaimed PDF 2.0 |
| [`output/pdf-2.0-compliance/`](../output/pdf-2.0-compliance/) | `--pdf-profile a4-ua2` | PDF/A-4 + PDF/UA-2 |

These are **artifacts**, not golden byte baselines. A version flag is not a
conformance claim; only the `-compliance/` dirs were written with a profile.

### Optional live smoke (also via `make samples` — not `make test`)

The Wikipedia artifact is an **operator recipe**, not a CSS-fidelity claim.
`--use-system-fonts` supplies IPA/Unicode fallback; `--zoom 0.666667`
densifies author `p { font-size: 12pt }` toward ~8pt. Other optional flags
for a separate “decent print” attempt: `--print-link-underline`,
`--simplify-dom --simplify-dom-profile=mediawiki`. Do **not** bake
`--simplify-dom` into this smoke file.

Manual equivalent:

```sh
./bin/gowkhtmltopdf --use-system-fonts --zoom 0.666667 \
  'https://en.wikipedia.org/wiki/Ana_de_Armas' output/wiki-ana-de-armas.pdf
```

Full URL-mode recipes (raw vs decent-print):
[cli.md — URL mode](cli.md#url-mode--chrome-strip---simplify-dom). Open the
PDF and judge layout honestly against the Phase 21 “decent print” bar. Do
**not** gate CI on this; commit `output/wiki-*.pdf` only when intentionally
updating the smoke artifact.

---

## Visual QA checklist

Structure tests (`make golden`) do not catch overlapping text, broken table
chrome, or underline noise. After changing `internal/layout` (or pagination
paint):

1. Rebuild: `make build`
2. Regenerate if needed: `make samples` (network for the wiki smoke)
3. Open `output/wiki-ana-de-armas.pdf` or a focused fixture PDF in a real
   viewer

Check:

- [ ] Body paragraphs are sequential — no interleaved or overpainted lines
- [ ] No huge empty bands between short `page-break-inside: avoid` list items
- [ ] Multi-page tables: continuous outer borders on continuation strips; Ref
      cites in rowspan cells readable (not crushed together)
- [ ] Reference lists: bare URLs not a dense underline forest; titles still linked
- [ ] Floats: text clears beside/below without orphan fragments mid-column
- [ ] Nested HTML HF (fixture-36): header/footer stay inside the reserved
      margin band; `#target` from the header reaches the body destination

Focused visual proofs already in tests (2026-08-12):

| Fixture | Proof | Verdict |
|---------|-------|---------|
| 21 | `TestFixture21ParagraphAfterForcedBreakStaysContiguous` | contiguous paragraph |
| 23 | `TestFixture23RepeatedHeaderHasNoVisualGap` | thead band, no gap |
| 28 | `TestFixture28FlexWrapGridItemsStayInFirstPageLayout` | labels on page 1 |
| 43 | `TestFixture43CardsAndTheadDoNotOverlap` | cards + thead |
| 55 | semantic needle `Northline` + crop test | masthead text present |

Contributor setup and PR expectations: [`CONTRIBUTING.md`](../CONTRIBUTING.md).

---

## Makefile targets

| Target | Action |
|--------|--------|
| `make test` | `go test ./...` |
| `make lint` | `golangci-lint run` (all linters via `.golangci.yml`), then `npm run lint` in `frontend/` |
| `make build` | `bin/gowkhtmltopdf`, `bin/gowkhtmltoimage` (version stamped from `VERSION`) |
| `make fmt` | `gofmt -w .` |
| `make golden` | `TestGoldenCorpus*` — structure, page envelopes, needles (not binary PDF equality) |
| `make golden-update GOLDEN_FIXTURE=fixture-NN-name.html GOLDEN_APPROVE=1` | Generate one explicitly approved fixture PDF under ignored `testdata/golden/out/`; never rewrites fixture sources |
| `make samples` | Refresh `output/fixture-*.pdf`, `showcase-toc-hf-outline.pdf`, the two PNGs, `output/pdf-{1.7,2.0}{,-compliance}/` (fixture-21 and fixture-56), and the optional live wiki PDF |
| `make clean` | Remove `testdata/golden/out` |
| `make claim-scan` | Fail if forbidden over-claim phrases appear in user-facing docs |

---

## Expected quality bar

- Latin report HTML: readable text, sane letter-spacing, multi-page tables
- TOC/outline showcase: navigable bookmarks, page headers/footers
- Tier 2 print CSS subset: float/flex/grid lite, thead repeat, Type0/CJK with a
  capable face via `--font-path`
- Wikipedia-class pages: may open as multi-page PDFs but **open-web layout
  parity is not a pass criterion** — Phase 21 / fidelity smoke only

What is in scope, Partial, or deferred: [fidelity.md](fidelity.md),
[compatibility-matrix.md](compatibility-matrix.md),
[deferred.md](deferred.md).
