# Phase 94.4: Fixtures 30-39

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** not started
> **Estimated effort:** M
> **Depends on:** 94.0 and 94.3
> **Unblocks:** 94.5

---

## Overview

File `internal/convert/output_pos_30_39_test.go`. Ten PDFs. Fixture 36 is the one that attaches `fixture-36-header.html` and `fixture-36-footer.html`. The fresh conversion must go through `requestForFixture`, which already calls `attachHFCompanions`. An anchor in the header and an anchor in the body are both required, so a conversion that drops the header fails.

## Checklist

- [ ] 94.4.1 `output/fixture-30-orphans-heuristic.pdf`. `TestOutputPosFixture30`.
- [ ] 94.4.2 `output/fixture-31-sticky-top.pdf`. `TestOutputPosFixture31`.
- [ ] 94.4.3 `output/fixture-32-flex-grid-full.pdf`. `TestOutputPosFixture32`.
- [ ] 94.4.4 `output/fixture-33-flex-cyclic-basis.pdf`. `TestOutputPosFixture33`.
- [ ] 94.4.5 `output/fixture-34-grid-areas-dense.pdf`. `TestOutputPosFixture34`.
- [ ] 94.4.6 `output/fixture-35-grid-minmax-intrinsic.pdf`. `TestOutputPosFixture35`.
- [ ] 94.4.7 `output/fixture-36-hf-nested-flex.pdf`. One anchor from the header HTML, one from the body, one from the footer HTML, plus one image lower-left. `TestOutputPosFixture36`.
- [ ] 94.4.8 `output/fixture-37-orphans-css.pdf`. `TestOutputPosFixture37`.
- [ ] 94.4.9 `output/fixture-38-float-inside-td.pdf`. `TestOutputPosFixture38`.
- [ ] 94.4.10 `output/fixture-39-multicol-article.pdf`. `TestOutputPosFixture39`.
- [ ] 94.4.11 `wc -l internal/convert/output_pos_30_39_test.go` is at or under 2000. Record the count.

Proof for each fixture row: `go test ./internal/convert -run 'TestOutputPosFixtureNN$' -count=1` exit 0.
