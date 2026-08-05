# testdata/golden - Golden Fixture Corpus

> **Parent:** `plans/00-canonical-pure-go-rewrite.md` (Phase 0.3, extended 9.1)
> **Purpose:** deterministic HTML in → PDF out, measured against pass criteria.

## Layout

```
testdata/golden/
  README.md
  logo.png                       # relative asset for fixture-07 (160x48 PNG)
  style-05.css                   # relative stylesheet for fixtures 05/06
  fixture-01-simple-invoice.html       # single page, minimal CSS
  fixture-02-table-heavy-invoice.html  # wide table, borders, many rows
  fixture-03-multi-page-invoice.html   # >1 page, page-break usage
  fixture-04-*.html .. fixture-21-*.html   # phase-9.1 corpus + detailed report
  out/                  # generated PDFs (gitignored)
```

Golden comparison is **structural + content**, not pixel-diff (Phase 0.3
decision; revisit image diffing in Phase 4 closure if cheap).

## Fixture inventory

Every fixture carries a comment header naming it and stating what it
proves. Page envelopes are pinned in `internal/convert/golden_test.go`
(`fixturePageBounds`); a fixture that moves out of its envelope fails
`make golden`.

| Fixture | What it exercises | Pages |
|---------|-------------------|-------|
| 01 | Simple invoice: `div`, `h1`, `p`, `table`, `span`, bold total | 1 |
| 02 | Table-heavy: 15-row table, thead/tbody/tfoot, colspan subtotal, borders | 2 |
| 03 | Multi-page statement: `page-break-before/after/inside`, outline headings | ≥2 |
| 04 | Two-column layout via fixed table cells (float lite also available; see fixture-22) | 1 |
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
| 16 | Full invoice: letterhead, bill-to, 24 line items, tfoot totals | 3 |
| 17 | Cover-style first page + `page-break-before` content page | 2 |
| 18 | Typography: h1–h6, strong/em/u/s/small, blockquote, code | 1 |
| 19 | Box model: fixed/min/max widths, margins, padding, borders | 1 |
| 20 | Image grid: four data: URI PNGs at intrinsic sizes | 1 |
| 21 | Detailed multi-section ops report (KPIs, WPs, invoice extract, REQs) | ≥3 |
| 22 | Float lite invoice chrome: float left/right + clear, inline-block badge, border-box | 1 |
| 23 | Multi-page table with repeating `<thead>` on continuation pages (phase 18) | ≥2 |
| 24 | Same-document `<a href="#id">` → PDF GoTo annotations (phase 20) | 2 |
| 25 | Partial flex row: justify-content, gap, flex-grow (phase 17) | 1 |
| 26 | Position lite: relative offsets + absolute overlay (phase 17) | 1 |
| 27 | CJK/Unicode sample (pair with `--font-path` for real glyphs; phase 19) | 1 |
| 28 | flex-wrap, CSS grid lite, position:fixed stamp | 2 |
| 29 | Float beside table: float:right infobox + wrapping prose + clear (phase 17 quality) | 1 |
| 30 | Orphans/keep-with-next heuristic sample (phase 18; geometric fallback) | ≥2 |
| 31 | Print-scoped `position: sticky` (page content box = scrollport; phase 17 sticky-print) | ≥2 |
| 32 | Flex Stage A + Grid Stage B lite (`flex-grid-full.md`): reverse, space-evenly, gaps, column flex, template-rows, row span | 1 |
| 33 | Flex `%` basis cyclic honesty: definite vs indefinite main size (cyclic → auto) | 1 |
| 34 | Grid `template-areas` / `grid-area` names + `grid-auto-flow: dense` | 1 |
| 35 | Grid `minmax` / intrinsic measure lite + subgrid copy-inherit + masonry pack | 1 |
| 33 | Flex % basis: definite resolve + indefinite/cyclic → auto (`flex-grid-full.md` §2.2) | 1 |
| 34 | Grid Stage B areas + dense (`flex-grid-full.md`): `grid-template-areas` / `grid-area`, implied tracks, `grid-auto-flow: dense` | 1 |
| 35 | Grid Stage B/C: `minmax()`+`fr` floors, cyclic height %, `display:subgrid` inherit, `grid-template-rows:masonry` packing | 1 |
| 37 | CSS `orphans`/`widows` parse + Rule 3 keep-together (tier-2-pending-3) | ≥2 |
| 38 | Float inside `td`: icon float + wrap + clear; table after float clears below (tier-2-pending-3) | 1 |
| 39 | CSS multicol lite: `column-count`/`gap`/`span`/`fill`; column boxes stay on one page (tier-2-pending-3) | ≥2 |
| 40 | Static 2D CSS `transform` badge (rotate) + abspos CB under transform (tier-2-pending-3) | 1 |
| 41 | `:has()` relational selector: article footnote border, `tr:has(td.neg)` row highlight (tier-2-pending-3) | 1 |

## Pass criteria (MVP)

A fixture passes when the generated PDF:

1. **Structure:** page count matches the expected envelope per fixture
   (table above; verified via `TestGoldenCorpusAllFixtures`).
2. **Content:** all fixture text strings present in extracted text, in order
   (text extraction from content streams, Phase 3/9).
3. **Geometry:** key box positions within tolerance of expected layout
   (Phase 4 golden tests; tolerance ±1 px @ 96 dpi initially).
4. **No regression:** output is byte-deterministic for identical input and
   settings (PDF writer determinism gate, Phase 3).

## How to run

```sh
make golden    # runs TestGoldenCorpus + TestGoldenCorpusAllFixtures
make golden-update   # writes testdata/golden/out/*.pdf from fixtures (stub)
```

Golden *source* (HTML/CSS/PNG) is committed; golden *output* (PDF) is
generated and reviewed at each phase gate, then archived on release only.

## Tolerance policy

- Text content: exact string match.
- Geometry: ±1 px @ 96 dpi; documented in each layout test.
- PDF bytes: deterministic per build (no timestamps by default; `--dpi`-ish
  metadata kept stable).
