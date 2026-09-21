# Phase 94.1: Fixtures 01-10

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** not started
> **Estimated effort:** M
> **Depends on:** 94.0
> **Unblocks:** 94.2, and the confidence that the helper works on a real sample

---

## Overview

First ten committed samples. File `internal/convert/output_pos_01_10_test.go`. Ten test functions. Measure each band from the committed PDF, then require the fresh `requestForFixture` conversion to match. Fixture 07 also gets one image anchor.

The path under `output/` is `fixture-NN-<slug>.pdf`. The golden HTML has the same basename with `.html`.

## Checklist

- [ ] 94.1.1 `output/fixture-01-simple-invoice.pdf`. Three ASCII text anchors. Test `TestOutputPosFixture01`. Proof: `go test ./internal/convert -run 'TestOutputPosFixture01$' -count=1` exit 0.
- [ ] 94.1.2 `output/fixture-02-table-heavy-invoice.pdf`. `TestOutputPosFixture02`. Same proof shape with `TestOutputPosFixture02$`.
- [ ] 94.1.3 `output/fixture-03-multi-page-invoice.pdf`. Anchors on different pages, including the last page. `TestOutputPosFixture03`.
- [ ] 94.1.4 `output/fixture-04-two-column-layout.pdf`. `TestOutputPosFixture04`.
- [ ] 94.1.5 `output/fixture-05-linked-stylesheet.pdf`. `TestOutputPosFixture05`.
- [ ] 94.1.6 `output/fixture-06-external-link.pdf`. Text anchors only. Link rectangles stay out of scope. `TestOutputPosFixture06`.
- [ ] 94.1.7 `output/fixture-07-image-logo.pdf`. Three text anchors plus one image lower-left. `images: true` in `fixturePageBounds`. `TestOutputPosFixture07`.
- [ ] 94.1.8 `output/fixture-08-forced-page-breaks.pdf`. Envelope is 5 pages. Put an anchor on page 1 and an anchor on page 5. `TestOutputPosFixture08`.
- [ ] 94.1.9 `output/fixture-09-multi-section-doc.pdf`. `TestOutputPosFixture09`.
- [ ] 94.1.10 `output/fixture-10-table-colspan.pdf`. `TestOutputPosFixture10`.
- [ ] 94.1.11 `wc -l internal/convert/output_pos_01_10_test.go` is at or under 2000. Record the count here.

Each row's proof is `go test ./internal/convert -run 'TestOutputPosFixtureNN$' -count=1` with that row's number, exit 0. Close the row only after that command exits 0.
