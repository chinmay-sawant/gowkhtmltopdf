# Phase 94.6: Fixtures 50-59

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** not started
> **Estimated effort:** M
> **Depends on:** 94.0 and 94.5
> **Unblocks:** 94.7 and the fixture 56 half of 94.8

---

## Overview

File `internal/convert/output_pos_50_59_test.go`. Ten PDFs. Fixture 56 is the 21-page architecture diagram. Its anchor table is reused by phase 94.8, so pick strings that still exist in the version and compliance renders. Fixtures 50, 51, 53, and 54 include an image anchor. Fixture 59 includes one too.

Fixture 56 and fixture 57 are large. One targeted `go test` of a single function is the mid-phase proof. Do not run `make test` here.

## Checklist

- [ ] 94.6.1 `output/fixture-50-letter-template.pdf`. Text anchors plus one image lower-left. `TestOutputPosFixture50`.
- [ ] 94.6.2 `output/fixture-51-asteria-storybook.pdf`. Envelope is 4 pages. Anchor on page 1 and page 4, plus one image. `TestOutputPosFixture51`.
- [ ] 94.6.3 `output/fixture-52-airline-boarding-pass.pdf`. `TestOutputPosFixture52`.
- [ ] 94.6.4 `output/fixture-53-asteria-observatory-poster.pdf`. Text anchors plus one image. `TestOutputPosFixture53`.
- [ ] 94.6.5 `output/fixture-54-ember-harbor-storybook.pdf`. Envelope is 4 pages. Anchor on page 1 and page 4, plus one image. `TestOutputPosFixture54`.
- [ ] 94.6.6 `output/fixture-55-lantern-cooperative-report.pdf`. Envelope is 3 pages. `TestOutputPosFixture55`.
- [ ] 94.6.7 `output/fixture-56-architecture-diagram.pdf`. Envelope is 21 pages. Anchor on page 1, one middle page, and page 21. Phase 94.8 reuses this table. `TestOutputPosFixture56`.
- [ ] 94.6.8 `output/fixture-57-vanguard-telemetry-audit.pdf`. Envelope is 9 pages. Anchor on page 1 and page 9. Avoid small-caps words. `TestOutputPosFixture57`.
- [ ] 94.6.9 `output/fixture-58-unsupported-worklist-audit.pdf`. Envelope is 9 pages. `TestOutputPosFixture58`.
- [ ] 94.6.10 `output/fixture-59-apex-digital-landing.pdf`. Envelope is 7 to 9 pages. Anchor on the first page and the last page, plus one image. `TestOutputPosFixture59`.
- [ ] 94.6.11 `wc -l internal/convert/output_pos_50_59_test.go` is at or under 2000. Record the count. If the file crosses 1600 before fixture 59 is added, split 57-59 into `output_pos_57_59_test.go` and say so in this row. Do not edit the allowlist.

Proof for each fixture row: `go test ./internal/convert -run 'TestOutputPosFixtureNN$' -count=1` exit 0.
