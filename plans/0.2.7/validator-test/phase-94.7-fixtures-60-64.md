# Phase 94.7: Fixtures 60-64

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** in progress
> **Estimated effort:** M
> **Depends on:** 94.0, and a written font-path answer in 94.7.0
> **Unblocks:** 94.9

---

## Overview

This phase uses one test file per PDF under `internal/convert/fixturetests`. The files open
the exact PDF samples and resolve the matching HTML fixtures by exact
basename. Four fixtures include local images, and every multi-page fixture
pins a first-page, continuation-page, or final-page record selected from the
independent PDF inspection.

## Checklist

- [x] 94.7.0 The fresh test request matches the samples command: A4 defaults, backgrounds enabled, local files enabled, `--font-path /usr/share/fonts/truetype/droid` when present, and `--font-path testdata/fonts` when present. The test helper also attaches fixture 36 header and footer companions when present. The implementation uses `requestForFixture`, which is the same effective configuration as `make samples`.
- [x] 94.7.1 `output/fixture-60-implemented-props-a.pdf`. `TestOutputFixture60ImplementedPropsA` pins three ASCII text records and one image. Shape: 8 pages, 859 text runs, 1,831 strokes, 472 fills, 120 images.
- [x] 94.7.2 `output/fixture-61-implemented-props-b.pdf`. `TestOutputFixture61ImplementedPropsB` pins three text records and one image. Shape: 8 pages, 989 text runs, 4,356 strokes, 488 fills, 8 images.
- [x] 94.7.3 `output/fixture-62-implemented-props-c.pdf`. `TestOutputFixture62ImplementedPropsC` pins three text records and one image. Shape: 8 pages, 890 text runs, 4,478 strokes, 500 fills, 7 images.
- [x] 94.7.4 `output/fixture-63-page-level-demos.pdf`. `TestOutputFixture63PageLevelDemos` pins the first-page dark-scheme title and needle plus the later split-box text. Shape: 7 pages, 35 text runs, 1,495 strokes, 18 fills, 0 images.
- [ ] 94.7.5 `output/fixture-64-next-72-props.pdf`. `TestOutputFixture64Next72Props` pins three ASCII text records and one image, including `NEXT-72-PROPS`; it does not use the CJK `汉A汉A汉` string. Shape: 8 pages, 1,002 committed text runs, 2,206 strokes, 302 fills, 6 images. The fresh conversion currently has 1,003 text runs and reports 1,576 to 1,586 text-stream differences, including non-breaking-space, soft-hyphen, and fallback-run changes.
- [x] 94.7.6 The five tests are split per PDF and remain far below the 2,000-line limit. Fixture tests 60-63 pass; fixture 64 remains open because the committed and fresh operation streams do not match.

Proof: `GOCACHE=/tmp/gowkhtmltopdf-fixture-test-cache-21-64 go test ./internal/convert/fixturetests -run '^TestOutputFixture(60|61|62|63|64)' -count=1` reaches fixture 64 and exits nonzero on the recorded differences.
