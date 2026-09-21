# Phase 94.2: Fixtures 11-20

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** not started
> **Estimated effort:** M
> **Depends on:** 94.0 and 94.1
> **Unblocks:** 94.3

---

## Overview

File `internal/convert/output_pos_11_20_test.go`. Ten fixtures. Fixture 20 has an image anchor. Multi-page fixtures put anchors on different pages.

## Checklist

- [ ] 94.2.1 `output/fixture-11-long-text-wrap.pdf`. Envelope starts at 3 pages. Anchor on the first page and the last page. `TestOutputPosFixture11`.
- [ ] 94.2.2 `output/fixture-12-lists.pdf`. `TestOutputPosFixture12`.
- [ ] 94.2.3 `output/fixture-13-pre-code-block.pdf`. `TestOutputPosFixture13`.
- [ ] 94.2.4 `output/fixture-14-colorful-report.pdf`. `TestOutputPosFixture14`.
- [ ] 94.2.5 `output/fixture-15-bulleted-requirements.pdf`. `TestOutputPosFixture15`.
- [ ] 94.2.6 `output/fixture-16-invoice-with-css.pdf`. `TestOutputPosFixture16`.
- [ ] 94.2.7 `output/fixture-17-cover-and-content.pdf`. Envelope is 2 pages. One anchor on the cover, one on the content page. `TestOutputPosFixture17`.
- [ ] 94.2.8 `output/fixture-18-typography.pdf`. `TestOutputPosFixture18`.
- [ ] 94.2.9 `output/fixture-19-margin-and-sizing.pdf`. `TestOutputPosFixture19`.
- [ ] 94.2.10 `output/fixture-20-image-grid.pdf`. Three text anchors plus one image lower-left. `TestOutputPosFixture20`.
- [ ] 94.2.11 `wc -l internal/convert/output_pos_11_20_test.go` is at or under 2000. Record the count.

Proof for each fixture row: `go test ./internal/convert -run 'TestOutputPosFixtureNN$' -count=1` exit 0.
