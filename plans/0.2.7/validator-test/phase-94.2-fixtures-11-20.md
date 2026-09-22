# Phase 94.2: Fixtures 11-20

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** complete
> **Estimated effort:** M
> **Depends on:** 94.0 and 94.1
> **Unblocks:** 94.3

---

## Overview

Ten fixture-specific files under `internal/convert/fixturetests`. Fixture 20 has four
image anchors. Multi-page fixtures put anchors on different pages. Each test
compares the committed PDF with a fresh conversion, then pins measured text,
color, rule, fill, or image operations for the feature the fixture claims to
prove.

## Checklist

- [x] 94.2.1 `output/fixture-11-long-text-wrap.pdf`. Three pages with anchors on the first and last pages. `TestOutputFixture11LongTextWrap` in `fixturetests/output_fixture_11_long_text_wrap_test.go`.
- [x] 94.2.2 `output/fixture-12-lists.pdf`. Nested-list and ordered-marker anchors. `TestOutputFixture12Lists` in `fixturetests/output_fixture_12_lists_test.go`.
- [x] 94.2.3 `output/fixture-13-pre-code-block.pdf`. Monospace pre text, inline code, fill, and border anchors. `TestOutputFixture13PreCodeBlock` in `fixturetests/output_fixture_13_pre_code_block_test.go`.
- [x] 94.2.4 `output/fixture-14-colorful-report.pdf`. Banner, KPI header fill, and horizontal-rule anchors. `TestOutputFixture14ColorfulReport` in `fixturetests/output_fixture_14_colorful_report_test.go`.
- [x] 94.2.5 `output/fixture-15-bulleted-requirements.pdf`. Requirement text, italic label, and acceptance-table anchors. `TestOutputFixture15BulletedRequirements` in `fixturetests/output_fixture_15_bulleted_requirements_test.go`.
- [x] 94.2.6 `output/fixture-16-invoice-with-css.pdf`. Two-page invoice, table header, and page-two footer anchors. `TestOutputFixture16InvoiceWithCSS` in `fixturetests/output_fixture_16_invoice_with_css_test.go`.
- [x] 94.2.7 `output/fixture-17-cover-and-content.pdf`. Two-page cover/content anchors and cover rule. `TestOutputFixture17CoverAndContent` in `fixturetests/output_fixture_17_cover_and_content_test.go`.
- [x] 94.2.8 `output/fixture-18-typography.pdf`. Heading, italic, small text, and blockquote-rule anchors. `TestOutputFixture18Typography` in `fixturetests/output_fixture_18_typography_test.go`.
- [x] 94.2.9 `output/fixture-19-margin-and-sizing.pdf`. Fixed-box fill and border-box rule anchors. `TestOutputFixture19MarginAndSizing` in `fixturetests/output_fixture_19_margin_and_sizing_test.go`.
- [x] 94.2.10 `output/fixture-20-image-grid.pdf`. Three text anchors and all four image placements. `TestOutputFixture20ImageGrid` in `fixturetests/output_fixture_20_image_grid_test.go`.
- [x] 94.2.11 Ten test files total 569 lines, all under the 2000-line file limit.

Proof for each fixture row: `go test ./internal/convert/fixturetests -run '^TestOutputFixtureNN' -count=1` exit 0. The batch proof also passed with `go test ./internal/convert/fixturetests -run 'TestOutputFixture(1[1-9]|20)' -count=1`.

## Measured record

Counts below come from `pdf.ParsePageOps` on the committed samples. Locations
were measured with `scripts/inspect_pdf_ops.py`; PDF coordinates use the
bottom-left origin.

| Fixture | Pages | Text ops | Strokes | Fills | Images |
|---------|------:|---------:|--------:|------:|-------:|
| 11 long text wrap | 3 | 1698 | 0 | 0 | 0 |
| 12 lists | 1 | 50 | 0 | 0 | 0 |
| 13 pre/code block | 1 | 39 | 12 | 3 | 0 |
| 14 colorful report | 1 | 30 | 50 | 23 | 0 |
| 15 bulleted requirements | 1 | 49 | 27 | 2 | 0 |
| 16 invoice with CSS | 2 | 175 | 333 | 78 | 0 |
| 17 cover and content | 2 | 21 | 3 | 0 | 0 |
| 18 typography | 1 | 35 | 5 | 0 | 0 |
| 19 margin and sizing | 1 | 24 | 42 | 4 | 0 |
| 20 image grid | 1 | 8 | 16 | 1 | 4 |
