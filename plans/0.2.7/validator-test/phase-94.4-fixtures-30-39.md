# Phase 94.4: Fixtures 30-39

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** complete
> **Estimated effort:** M
> **Depends on:** 94.0 and 94.3
> **Unblocks:** 94.5

---

## Overview

This phase uses one test file per PDF under `internal/convert/fixturetests`. Fixture 36
resolves `fixture-36-header.html` and `fixture-36-footer.html` through
`requestForFixture` and `attachHFCompanions`; its test pins header, body,
footer, and image evidence.

## Checklist

- [x] 94.4.1 `output/fixture-30-orphans-heuristic.pdf`. `TestOutputFixture30OrphansHeuristic`. Shape: 3 pages, 82 text runs, 0 strokes, 0 fills, 0 images.
- [x] 94.4.2 `output/fixture-31-sticky-top.pdf`. `TestOutputFixture31StickyTop`. Shape: 2 pages, 108 text runs, 332 strokes, 41 fills, 0 images.
- [x] 94.4.3 `output/fixture-32-flex-grid-full.pdf`. `TestOutputFixture32FlexGridFull`. Shape: 1 page, 21 text runs, 48 strokes, 15 fills, 0 images.
- [x] 94.4.4 `output/fixture-33-flex-cyclic-basis.pdf`. `TestOutputFixture33FlexCyclicBasis`. Shape: 1 page, 23 text runs, 13 strokes, 9 fills, 0 images.
- [x] 94.4.5 `output/fixture-34-grid-areas-dense.pdf`. `TestOutputFixture34GridAreasDense`. Shape: 1 page, 21 text runs, 52 strokes, 13 fills, 0 images.
- [x] 94.4.6 `output/fixture-35-grid-minmax-intrinsic.pdf`. `TestOutputFixture35GridMinmaxIntrinsic`. Shape: 1 page, 26 text runs, 60 strokes, 18 fills, 0 images.
- [x] 94.4.7 `output/fixture-36-hf-nested-flex.pdf`. `TestOutputFixture36HFNestedFlex` pins header, body, footer, and image records. Shape: 1 page, 16 text runs, 2 strokes, 0 fills, 1 image.
- [x] 94.4.8 `output/fixture-37-orphans-css.pdf`. `TestOutputFixture37OrphansCSS`. Shape: 3 pages, 49 text runs, 0 strokes, 0 fills, 0 images.
- [x] 94.4.9 `output/fixture-38-float-inside-td.pdf`. `TestOutputFixture38FloatInsideTD`. Shape: 1 page, 18 text runs, 7 strokes, 2 fills, 0 images.
- [x] 94.4.10 `output/fixture-39-multicol-article.pdf`. `TestOutputFixture39MulticolArticle`. Shape: 3 pages, 215 text runs, 0 strokes, 0 fills, 0 images.
- [x] 94.4.11 All ten targeted fixture tests passed in the 30-39 batch. The tests are split per PDF, so no generated test file approaches the 2,000-line limit.

Proof: `GOCACHE=/tmp/gowkhtmltopdf-fixture-test-cache-21-64 go test ./internal/convert/fixturetests -run '^TestOutputFixture(3[0-9])' -count=1` exited 0.
