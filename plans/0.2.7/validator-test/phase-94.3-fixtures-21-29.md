# Phase 94.3: Fixtures 21-29

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** complete
> **Estimated effort:** M
> **Depends on:** 94.0 and 94.1
> **Unblocks:** 94.4 and the fixture 21 half of 94.8

---

## Overview

This phase uses one test file per PDF under `internal/convert/fixturetests/`. There are ten
PDFs because fixture 29 has two distinct body fixtures. Each test compares the
full committed and fresh page-operation records, then pins inspected PDF-space
anchors and the measured operation shape.

Layout tests already cover pieces of fixture 21, 23, and 28 in `internal/layout/requested_fixture_regression_test.go`. Leave those tests alone. This phase adds the PDF-space band on the committed sample.

## Checklist

- [x] 94.3.1 `output/fixture-21-detailed-report.pdf`. `TestOutputFixture21DetailedReport`. Shape: 4 pages, 267 text runs, 411 strokes, 118 fills, 0 images.
- [x] 94.3.2 `output/fixture-22-float-invoice-chrome.pdf`. `TestOutputFixture22FloatInvoiceChrome`. Shape: 1 page, 19 text runs, 40 strokes, 6 fills, 0 images.
- [x] 94.3.3 `output/fixture-23-thead-repeat.pdf`. `TestOutputFixture23TheadRepeat` includes continuation-page text. Shape: 2 pages, 249 text runs, 563 strokes, 8 fills, 0 images.
- [x] 94.3.4 `output/fixture-24-internal-anchors.pdf`. `TestOutputFixture24InternalAnchors`. Shape: 2 pages, 10 text runs, 2 strokes, 0 fills, 0 images.
- [x] 94.3.5 `output/fixture-25-flex-row.pdf`. `TestOutputFixture25FlexRow`. Shape: 1 page, 4 text runs, 4 strokes, 3 fills, 0 images.
- [x] 94.3.6 `output/fixture-26-position-lite.pdf`. `TestOutputFixture26PositionLite`. Shape: 1 page, 4 text runs, 320 strokes, 4 fills, 0 images.
- [x] 94.3.7 `output/fixture-27-cjk-fontpath.pdf`. `TestOutputFixture27CJKFontpath` uses ASCII anchors because the low-level parser does not return stable CJK text bytes. Shape: 1 page, 24 text runs, 0 strokes, 0 fills, 0 images.
- [x] 94.3.8 `output/fixture-28-flex-wrap-grid-fixed.pdf`. `TestOutputFixture28FlexWrapGridFixed`. Shape: 2 pages, 14 text runs, 8 strokes, 10 fills, 0 images.
- [x] 94.3.9 `output/fixture-29-float-beside-table.pdf`. `TestOutputFixture29FloatBesideTable`. Shape: 1 page, 28 text runs, 34 strokes, 3 fills, 0 images.
- [x] 94.3.10 `output/fixture-29-wpt-break-nested-float-print.pdf`. `TestOutputFixture29WPTBreakNestedFloatPrint`. Shape: 3 pages, 1 text run, 0 strokes, 3 fills, 0 images.
- [x] 94.3.11 All ten targeted fixture tests passed in the 21-29 batch. The tests are split per PDF, so no generated test file approaches the 2,000-line limit.

Proof: `GOCACHE=/tmp/gowkhtmltopdf-fixture-test-cache-21-64 go test ./internal/convert/fixturetests -run '^TestOutputFixture(2[1-9])' -count=1` exited 0.
