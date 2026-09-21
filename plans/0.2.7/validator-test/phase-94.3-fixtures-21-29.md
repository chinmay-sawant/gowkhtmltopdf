# Phase 94.3: Fixtures 21-29

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** not started
> **Estimated effort:** M
> **Depends on:** 94.0 and 94.1
> **Unblocks:** 94.4 and the fixture 21 half of 94.8

---

## Overview

File `internal/convert/output_pos_21_29_test.go`. Ten PDFs. Fixture 29 is two files, so the decade stops at 29 and fixture 30 starts the next file. That keeps this file at 10.

Layout tests already cover pieces of fixture 21, 23, and 28 in `internal/layout/requested_fixture_regression_test.go`. Leave those tests alone. This phase adds the PDF-space band on the committed sample.

## Checklist

- [ ] 94.3.1 `output/fixture-21-detailed-report.pdf`. Anchors on page 1 and on the last page. Keep the three strings. Phase 94.8 reuses this table. `TestOutputPosFixture21`.
- [ ] 94.3.2 `output/fixture-22-float-invoice-chrome.pdf`. `TestOutputPosFixture22`.
- [ ] 94.3.3 `output/fixture-23-thead-repeat.pdf`. Include one anchor that only exists on a continuation page, so a missing repeated header fails this band. `TestOutputPosFixture23`.
- [ ] 94.3.4 `output/fixture-24-internal-anchors.pdf`. Envelope is 2 pages. Text anchors only. `TestOutputPosFixture24`.
- [ ] 94.3.5 `output/fixture-25-flex-row.pdf`. `TestOutputPosFixture25`.
- [ ] 94.3.6 `output/fixture-26-position-lite.pdf`. `TestOutputPosFixture26`.
- [ ] 94.3.7 `output/fixture-27-cjk-fontpath.pdf`. Anchors are ASCII labels, not the CJK runs. `TestOutputPosFixture27`.
- [ ] 94.3.8 `output/fixture-28-flex-wrap-grid-fixed.pdf`. Envelope is 2 pages. `TestOutputPosFixture28`.
- [ ] 94.3.9 `output/fixture-29-float-beside-table.pdf`. `TestOutputPosFixture29Float`.
- [ ] 94.3.10 `output/fixture-29-wpt-break-nested-float-print.pdf`. Envelope is 3 pages. `TestOutputPosFixture29Wpt`. The run regex is `TestOutputPosFixture29Wpt$` so it does not run the float test.
- [ ] 94.3.11 `wc -l internal/convert/output_pos_21_29_test.go` is at or under 2000. Record the count.

Proof for each row: `go test ./internal/convert -run '<TestName>$' -count=1` exit 0.
