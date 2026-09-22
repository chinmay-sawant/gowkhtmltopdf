# Phase 94.1: Fixtures 01-10

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** complete; fixtures 01-10 verified
> **Estimated effort:** M
> **Depends on:** 94.0
> **Unblocks:** 94.2, and the confidence that the helper works on a real sample

---

## Overview

First ten committed samples. Each fixture gets its own test file under
`internal/convert/fixturetests`, named after the complete PDF basename. Measure
each authored operation from the committed PDF,
then require the fresh `requestForFixture` conversion to match. Fixture 07
also gets one image anchor.

The path under `output/` is `fixture-NN-<slug>.pdf`. The golden HTML has the same basename with `.html`.

## Checklist

- [x] 94.1.1 `output/fixture-01-simple-invoice.pdf`. Test `TestOutputFixture01SimpleInvoice` pins measured text and rule operations. Proof: `go test ./internal/convert/fixturetests -run 'TestOutputFixture01SimpleInvoice$' -count=1` exit 0.
- [x] 94.1.2 `output/fixture-02-table-heavy-invoice.pdf`. Test `TestOutputFixture02TableHeavyInvoice` pins measured text, table fills, a cell border, page geometry, and operation counts. Proof: `go test ./internal/convert/fixturetests -run 'TestOutputFixture02TableHeavyInvoice$' -count=1` exit 0.
- [x] 94.1.3 `output/fixture-03-multi-page-invoice.pdf`. Test `TestOutputFixture03MultiPageInvoice` pins page 1, page 2, page 3, and page 4 anchors plus the table header. Proof: `go test ./internal/convert/fixturetests -run 'TestOutputFixture03MultiPageInvoice$' -count=1` exit 0.
- [x] 94.1.4 `output/fixture-04-two-column-layout.pdf`. Test `TestOutputFixture04TwoColumnLayout` pins both column fills, their border, and column headings. Proof: `go test ./internal/convert/fixturetests -run 'TestOutputFixture04TwoColumnLayout$' -count=1` exit 0.
- [x] 94.1.5 `output/fixture-05-linked-stylesheet.pdf`. Test `TestOutputFixture05LinkedStylesheet` pins linked-stylesheet colors, notice fill, table fill, and border. Proof: `go test ./internal/convert/fixturetests -run 'TestOutputFixture05LinkedStylesheet$' -count=1` exit 0.
- [x] 94.1.6 `output/fixture-06-external-link.pdf`. Test `TestOutputFixture06ExternalLink` pins link text styling and table geometry; URI rectangles remain covered by the golden runner. Proof: `go test ./internal/convert/fixturetests -run 'TestOutputFixture06ExternalLink$' -count=1` exit 0.
- [x] 94.1.7 `output/fixture-07-image-logo.pdf`. Test `TestOutputFixture07ImageLogo` pins both image placements plus the letterhead and signature rules. Proof: `go test ./internal/convert/fixturetests -run 'TestOutputFixture07ImageLogo$' -count=1` exit 0.
- [x] 94.1.8 `output/fixture-08-forced-page-breaks.pdf`. Test `TestOutputFixture08ForcedPageBreaks` pins the intro and all four forced section pages. Proof: `go test ./internal/convert/fixturetests -run 'TestOutputFixture08ForcedPageBreaks$' -count=1` exit 0.
- [x] 94.1.9 `output/fixture-09-multi-section-doc.pdf`. Test `TestOutputFixture09MultiSectionDoc` pins the first-page report and second-page continuation. Proof: `go test ./internal/convert/fixturetests -run 'TestOutputFixture09MultiSectionDoc$' -count=1` exit 0.
- [x] 94.1.10 `output/fixture-10-table-colspan.pdf`. Test `TestOutputFixture10TableColspan` pins both colspan header fills, nested-table text, and table geometry. Proof: `go test ./internal/convert/fixturetests -run 'TestOutputFixture10TableColspan$' -count=1` exit 0.
- [x] 94.1.11 All ten fixture test files are at or under 2000 lines. Current counts: 64, 67, 70, 66, 68, 64, 61, 65, 63, and 64.

Each completed row records its actual fixture test name and proof command. Close
the row only after that command exits 0.
