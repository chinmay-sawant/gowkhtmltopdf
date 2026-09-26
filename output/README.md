# Sample outputs

Committed viewer-smoke artifacts produced by `make samples` (and optional URL
smoke). Regenerate anytime:

```sh
make samples
```

These are **not** golden byte baselines. CI uses `make golden` / structure
assertions, not binary PDF equality against this folder. See
[`documentation/samples.md`](../documentation/samples.md).

## Validated visual references

`validated/` holds one reference PDF for every numbered body fixture from 01
through 64. There are 65 PDFs because fixture 29 has two body HTML files.
Fixture 36's header and footer files belong to its single body PDF.

`TestVisualReferenceInventory` checks that the directory has exactly these
65 PDFs (`internal/convert/fixturetests/pixel_regression_corpus_test.go:18-56`).
`TestVisualGoldenFixtureCorpus` converts each body fixture with the shared
fixture settings and compares every rendered page
(`internal/convert/fixturetests/pixel_regression_corpus_test.go:59-96`). The
tests read `validated/` and never create or replace files there
(`internal/convert/fixturetests/pixel_regression_test.go:36-59`).

To approve a reference, generate a candidate PDF under `output/` or a temporary
directory with the same fixture and font settings used by the test helper
(`internal/convert/fixturetests/fixture_helpers_test.go:78-112`). The fixture
01 candidate command is:

```sh
go run ./cmd/gowkhtmltopdf --allow-local-files \
  --font-path /usr/share/fonts/truetype/droid \
  --font-path testdata/fonts \
  -o output/fixture-01-simple-invoice-candidate.pdf \
  testdata/golden/fixture-01-simple-invoice.html
```

Render candidates with Ghostscript 10.08.0, `png16m`, 150 dots per inch
(DPI), and text and graphics smoothing level 4. Review every page before
copying an accepted candidate into `validated/`. Tests never approve or copy
references. The comparator checks page count and dimensions, then every RGB
pixel. Mismatches report the fixture, page, changed-pixel count, largest
channel difference, and diff path
(`internal/convert/fixturetests/pixel_regression_test.go:77-100`,
`internal/convert/fixturetests/pixel_regression_test.go:198-218`,
`internal/convert/fixturetests/pixel_regression_test.go:221-290`).

Run the full fixture comparison with the pinned Ghostscript binary:

```sh
GOWKHTMLTOPDF_VISUAL_GS=/path/to/gs \
  go test ./internal/convert/fixturetests \
  -run '^TestVisualGoldenFixtureCorpus$' -count=1
```

The full comparison converts each fixture unless you run the separate fixture
01 saved-candidate check. Set `GOWKHTMLTOPDF_VISUAL_CANDIDATE` to compare a
saved candidate with the fixture 01 reference:

```sh
GOWKHTMLTOPDF_VISUAL_GS=/path/to/gs \
  GOWKHTMLTOPDF_VISUAL_CANDIDATE=output/fixture-01-simple-invoice-candidate.pdf \
  go test ./internal/convert/fixturetests \
  -run '^TestVisualGoldenFixture01SavedCandidate$' -count=1
```

The full visual comparison and candidate checks skip in ordinary `make test`
runs unless `GOWKHTMLTOPDF_VISUAL_GS` is set. The inventory check always runs.
Set `GOWKHTMLTOPDF_VISUAL_DIFF_DIR` to keep the mutation diff for inspection
after the test exits
(`internal/convert/fixturetests/pixel_regression_cases_test.go:141-192`).

`make samples` rewrites `fixture-*.pdf`, `fixture-*.png`,
`showcase-*.pdf`, `architecture-diagram.pdf` (via
`go run ./testdata/golden/api`), the four `pdf-1.7/` /
`pdf-1.7-compliance/` / `pdf-2.0/` / `pdf-2.0-compliance/` smokes
(fixture-21 and fixture-56), and (when the network works)
`wiki-ana-de-armas.pdf`.

## Fixture PDFs (`fixture-01` ... `fixture-64`)

Each file is `testdata/golden/<same-basename>.html` converted with
`--enable-local-file-access`. Companion `*-header.html` / `*-footer.html`
files are skipped as bodies; fixture-36 attaches them as HTML header/footer.

| File | Source fixture |
|------|----------------|
| `fixture-01-simple-invoice.pdf` | Simple invoice |
| `fixture-02-table-heavy-invoice.pdf` | Table-heavy invoice |
| `fixture-03-multi-page-invoice.pdf` | Multi-page statement |
| `fixture-04-two-column-layout.pdf` | Two-column table layout |
| `fixture-05-linked-stylesheet.pdf` | Linked `style-05.css` |
| `fixture-06-external-link.pdf` | External URI annotations |
| `fixture-07-image-logo.pdf` | Relative + data: URI images |
| `fixture-08-forced-page-breaks.pdf` | Forced page breaks (5 pages) |
| `fixture-09-multi-section-doc.pdf` | Long multi-section report |
| `fixture-10-table-colspan.pdf` | Nested tables / colspan |
| `fixture-11-long-text-wrap.pdf` | Justified prose wrap |
| `fixture-12-lists.pdf` | Nested lists |
| `fixture-13-pre-code-block.pdf` | `pre` / `code` |
| `fixture-14-colorful-report.pdf` | Backgrounds, rgba, borders |
| `fixture-15-bulleted-requirements.pdf` | Requirements list |
| `fixture-16-invoice-with-css.pdf` | Full CSS invoice (1–2 pages) |
| `fixture-17-cover-and-content.pdf` | Cover + content |
| `fixture-18-typography.pdf` | Heading / inline styles |
| `fixture-19-margin-and-sizing.pdf` | Box model |
| `fixture-20-image-grid.pdf` | Data-URI image grid |
| `fixture-21-detailed-report.pdf` | Detailed ops report |
| `fixture-22-float-invoice-chrome.pdf` | Float lite chrome |
| `fixture-23-thead-repeat.pdf` | Repeating `<thead>` |
| `fixture-24-internal-anchors.pdf` | Same-document GoTo |
| `fixture-25-flex-row.pdf` | Flex row lite |
| `fixture-26-position-lite.pdf` | Relative / absolute lite |
| `fixture-27-cjk-fontpath.pdf` | CJK / `--font-path` |
| `fixture-28-flex-wrap-grid-fixed.pdf` | Flex-wrap + grid + fixed stamp |
| `fixture-29-float-beside-table.pdf` | Float beside table |
| `fixture-29-wpt-break-nested-float-print.pdf` | Nested float print break |
| `fixture-30-orphans-heuristic.pdf` | Orphans heuristic |
| `fixture-31-sticky-top.pdf` | Print-scoped sticky |
| `fixture-32-flex-grid-full.pdf` | Flex/grid stage sample |
| `fixture-33-flex-cyclic-basis.pdf` | Flex `%` cyclic basis |
| `fixture-34-grid-areas-dense.pdf` | Grid areas + dense |
| `fixture-35-grid-minmax-intrinsic.pdf` | Grid minmax / subgrid / masonry lite |
| `fixture-36-hf-nested-flex.pdf` | Nested HTML header/footer |
| `fixture-37-orphans-css.pdf` | CSS orphans / widows |
| `fixture-38-float-inside-td.pdf` | Float inside `td` |
| `fixture-39-multicol-article.pdf` | Multicol lite |
| `fixture-40-transform-badge.pdf` | Static 2D transform |
| `fixture-41-has-selector.pdf` | `:has()` |
| `fixture-42-container-inline-size.pdf` | `@container` size lite |
| `fixture-43-complex-dossier.pdf` | Five-page dossier |
| `fixture-44-receipt.pdf` | Receipt |
| `fixture-45-purchase-order.pdf` | Purchase order |
| `fixture-46-contract.pdf` | Contract |
| `fixture-47-certificate.pdf` | Certificate |
| `fixture-48-shipping-document.pdf` | Shipping document |
| `fixture-49-night-train-poster.pdf` | Illustrated poster |
| `fixture-50-letter-template.pdf` | Stationery letter |
| `fixture-51-asteria-storybook.pdf` | Asteria storybook (4 pages) |
| `fixture-52-airline-boarding-pass.pdf` | Boarding pass |
| `fixture-53-asteria-observatory-poster.pdf` | Observatory poster |
| `fixture-54-ember-harbor-storybook.pdf` | Ember Harbor storybook (4 pages) |
| `fixture-55-lantern-cooperative-report.pdf` | Northline operations brief |
| `fixture-56-architecture-diagram.pdf` | Architecture diagram (21 pages) |
| `fixture-57-vanguard-telemetry-audit.pdf` | Vanguard Telemetry Audit (356 implemented CSS properties) |
| `fixture-58-unsupported-worklist-audit.pdf` | Unsupported CSS worklist audit (462 unsupported CSS properties) |
| `fixture-59-apex-digital-landing.pdf` | Apex Digital landing page |
| `fixture-60-implemented-props-a.pdf` | Implemented CSS properties, group A |
| `fixture-61-implemented-props-b.pdf` | Implemented CSS properties, group B |
| `fixture-62-implemented-props-c.pdf` | Implemented CSS properties, group C |
| `fixture-63-page-level-demos.pdf` | Page-level CSS demos |
| `fixture-64-next-72-props.pdf` | Next-72 CSS properties |

## Image smokes

| File | How it is produced |
|------|--------------------|
| `fixture-01-simple-invoice.png` | `gowkhtmltoimage` on fixture-01 |
| `fixture-21-detailed-report.png` | `examples/image` on fixture-21 (`--width 1024`) |

## Showcase and live URL

| File | Description |
|------|-------------|
| `showcase-toc-hf-outline.pdf` | TOC + text headers/footers + outline on fixture-16 |
| `wiki-ana-de-armas.pdf` | Live Wikipedia **raw** smoke from `make samples` (no `--simplify-dom`; `--use-system-fonts --zoom 0.666667`; needs network; **soft-fail** if offline). Not a chrome-stripped "pretty" print - see [`documentation/cli.md`](../documentation/cli.md#url-mode-chrome-strip-simplify-dom). |

## Library-API architecture diagram

| File | Note |
|------|------|
| `architecture-diagram.pdf` | Regenerated by `make samples` via `go run ./testdata/golden/api` from `testdata/golden/api/architecture-diagram.html` (5 pages). Distinct from `fixture-56-architecture-diagram.pdf` and from the corpus fixture `testdata/golden/architecture-diagram.html`. This is the only PDF the generator writes; nothing is written under `testdata/golden/`. |

## Python-API samples

Regenerated by `make samples-python` (requires `CGO_ENABLED=1 make c-shared`).
`make python-api` is the smaller subset (architecture + inline + architecture
compliance only).

| File / dir | Note |
|------------|------|
| `python/fixture-*.pdf` | Every golden body fixture via `convert_file_to_pdf` (skip `*-header`/`*-footer`). Fixture-36 is body-only: v1 ABI has no header/footer HTML. Most are gitignored; **fixture-55** and **fixture-56** stay committed as smoke exceptions. Regenerate with `make samples-python`. |
| `python/architecture-diagram.pdf` | `testdata/golden/python_api/architecture-diagram.html` through `Document.pdf()`. |
| `python/invoice-inline.pdf` | Inline HTML Document sample (`generate_inline.py`). |
| `python/pdf-1.7/` | `pdf_version="1.7"` smokes: architecture-diagram + fixture-21 + fixture-56 (unclaimed). |
| `python/pdf-1.7-compliance/` | `pdf_profile="a3a-ua1"` for the same three basenames. |
| `python/pdf-2.0/` | `pdf_version="2.0"` smokes (unclaimed). |
| `python/pdf-2.0-compliance/` | `pdf_profile="a4-ua2"` for the same three basenames. |

## Version and compliance sample folders

Part of `make samples` (0.2.2 flags). Same two fixtures as the
1.7 / 2.0 smoke set:

| Dir | How produced | Files |
|-----|--------------|-------|
| `pdf-1.7/` | `--pdf-version 1.7` (unclaimed) | `fixture-21-detailed-report.pdf`, `fixture-56-architecture-diagram.pdf` |
| `pdf-1.7-compliance/` | `--pdf-profile a3a-ua1` | same basenames |
| `pdf-2.0/` | `--pdf-version 2.0` (unclaimed) | same basenames |
| `pdf-2.0-compliance/` | `--pdf-profile a4-ua2` | same basenames |

These are **artifacts**, not golden byte baselines. A version flag is not a
conformance claim; only the `-compliance/` dirs use a profile.

## Extra artifacts (not `make samples`)

These may be present from earlier manual or comparison runs. `make samples`
does **not** regenerate them:

| File / dir | Note |
|------------|------|
| `font-examples.pdf` | Manual leftover from `testdata/golden/font-examples.html` (fonts not bundled; `--font-path` driven). Not a `make samples` output. |
| `chrome_ana.pdf` | Manual Chrome comparison artifact, if present. |
| `wkhtmltopdf/` | Side-by-side wkhtmltopdf fixture PDFs and benchmark notes; not overwritten by `make samples`. |
