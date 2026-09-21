# Phase 94.5: Fixtures 40-49

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** not started
> **Estimated effort:** M
> **Depends on:** 94.0 and 94.4
> **Unblocks:** 94.6

---

## Overview

File `internal/convert/output_pos_40_49_test.go`. Ten PDFs. Fixtures 43, 47, and 49 include an image anchor. Fixture 43's envelope is 5 pages, so one anchor goes on page 5.

## Checklist

- [ ] 94.5.1 `output/fixture-40-transform-badge.pdf`. If the badge text is rotated, the reader must have found it through `Tm` in 94.0.2.3. `TestOutputPosFixture40`.
- [ ] 94.5.2 `output/fixture-41-has-selector.pdf`. `TestOutputPosFixture41`.
- [ ] 94.5.3 `output/fixture-42-container-inline-size.pdf`. `TestOutputPosFixture42`.
- [ ] 94.5.4 `output/fixture-43-complex-dossier.pdf`. Anchor on page 1 and page 5, plus one image lower-left. `TestOutputPosFixture43`.
- [ ] 94.5.5 `output/fixture-44-receipt.pdf`. `TestOutputPosFixture44`.
- [ ] 94.5.6 `output/fixture-45-purchase-order.pdf`. `TestOutputPosFixture45`.
- [ ] 94.5.7 `output/fixture-46-contract.pdf`. `TestOutputPosFixture46`.
- [ ] 94.5.8 `output/fixture-47-certificate.pdf`. Three text anchors plus one image lower-left. `TestOutputPosFixture47`.
- [ ] 94.5.9 `output/fixture-48-shipping-document.pdf`. `TestOutputPosFixture48`.
- [ ] 94.5.10 `output/fixture-49-night-train-poster.pdf`. Three text anchors plus one image lower-left. `TestOutputPosFixture49`.
- [ ] 94.5.11 `wc -l internal/convert/output_pos_40_49_test.go` is at or under 2000. Record the count.

Proof for each fixture row: `go test ./internal/convert -run 'TestOutputPosFixtureNN$' -count=1` exit 0.
