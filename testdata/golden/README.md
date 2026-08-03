# testdata/golden — Golden Fixture Corpus

> **Parent:** `plans/00-canonical-pure-go-rewrite.md` (Phase 0.3)
> **Purpose:** deterministic HTML in → PDF out, measured against pass criteria.

## Layout

```
testdata/golden/
  README.md
  fixture-01-simple-invoice.html     # single page, minimal CSS
  fixture-02-table-heavy-invoice.html  # wide table, borders, many rows
  fixture-03-multi-page-invoice.html   # >1 page, page-break usage
  out/                  # generated PDFs (gitignored)
```

Golden comparison is **structural + content**, not pixel-diff (Phase 0.3
decision; revisit image diffing in Phase 4 closure if cheap).

## Fixture inventory

| Fixture | What it exercises | Allowed matrix surface |
|---------|-------------------|------------------------|
| 01 | Simple invoice: `div`, `h1`, `p`, `table`, `span`, `img` logo | §1 tags, box model, fonts, colors, `text-align` |
| 02 | Table-heavy: long `table`, `thead`/`tbody`, borders, `th`/`td` | §2.5 table subset, borders, `font-size`, row spanning intent |
| 03 | Multi-page: `page-break-before/after/inside` on sections | §2.6 paged media, pagination, repeated table header intent |

Each fixture: ~1–3 pages of server-generated HTML, **no JS**, no external
assets (images inline as data URI or local file in `assets/` if ever added).

## Pass criteria (MVP)

A fixture passes when the generated PDF:

1. **Structure:** page count matches expected range per fixture (01: 1, 02: 1–2,
   03: ≥2). Verified via PDF page tree traversal (Phase 3 tests).
2. **Content:** all fixture text strings present in extracted text, in order
   (text extraction from content streams, Phase 3/9).
3. **Geometry:** key box positions within tolerance of expected layout
   (Phase 4 golden tests; tolerance ±1 px @ 96 dpi initially).
4. **No regression:** output is byte-deterministic for identical input and
   settings (PDF writer determinism gate, Phase 3).

## How to regenerate

```sh
make golden-update   # writes testdata/golden/out/*.pdf from fixtures
```

Golden *source* (HTML) is committed; golden *output* (PDF) is generated and
reviewed at each phase gate, then archived on release only.

## Tolerance policy

- Text content: exact string match.
- Geometry: ±1 px @ 96 dpi; documented in each layout test.
- PDF bytes: deterministic per build (no timestamps by default; `--dpi`-ish
  metadata kept stable).
