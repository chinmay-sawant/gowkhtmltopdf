# Phase 94.6: Fixtures 50-59

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** in progress
> **Estimated effort:** M
> **Depends on:** 94.0 and 94.5
> **Unblocks:** 94.7 and the fixture 56 half of 94.8

---

## Overview

This phase uses one test file per PDF under `internal/convert`. The tests open
the exact top-level samples, convert their exact matching HTML fixtures, and
compare every parsed page operation before checking measured anchors.

Fixtures 56 and 57 are large. Fixture 56 currently exposes six exact text
token differences between its committed sample and fresh conversion, so its
row stays open. No comparison normalization was added.

## Checklist

- [x] 94.6.1 `output/fixture-50-letter-template.pdf`. `TestOutputFixture50LetterTemplate` pins text and an image. Shape: 1 page, 20 text runs, 1 stroke, 3 fills, 2 images.
- [x] 94.6.2 `output/fixture-51-asteria-storybook.pdf`. `TestOutputFixture51AsteriaStorybook` pins first-page and last-page text plus an image. Shape: 4 pages, 26 text runs, 1 stroke, 9 fills, 3 images.
- [x] 94.6.3 `output/fixture-52-airline-boarding-pass.pdf`. `TestOutputFixture52AirlineBoardingPass`. Shape: 1 page, 150 text runs, 356 strokes, 17 fills, 0 images.
- [x] 94.6.4 `output/fixture-53-asteria-observatory-poster.pdf`. `TestOutputFixture53AsteriaObservatoryPoster` pins text and an image. Shape: 1 page, 7 text runs, 1 stroke, 3 fills, 1 image.
- [x] 94.6.5 `output/fixture-54-ember-harbor-storybook.pdf`. `TestOutputFixture54EmberHarborStorybook` pins first-page and last-page text plus an image. Shape: 4 pages, 31 text runs, 1 stroke, 9 fills, 3 images.
- [x] 94.6.6 `output/fixture-55-lantern-cooperative-report.pdf`. `TestOutputFixture55LanternCooperativeReport`. Shape: 3 pages, 118 text runs, 164 strokes, 39 fills, 0 images.
- [ ] 94.6.7 `output/fixture-56-architecture-diagram.pdf`. `TestOutputFixture56ArchitectureDiagram` has the required page 1, page 19, and page 21 anchors and shape record: 21 pages, 1,920 text runs, 2,206 strokes, 10,834 fills, 9 images. The full comparison currently reports six text-token differences: soft hyphens in the date and non-breaking spaces in `html.go`, `832 lines`, `25 000`, and `1 000 000`.
- [x] 94.6.8 `output/fixture-57-vanguard-telemetry-audit.pdf`. `TestOutputFixture57VanguardTelemetryAudit`. Shape: 9 pages, 924 text runs, 3,198 strokes, 949 fills, 413 images.
- [x] 94.6.9 `output/fixture-58-unsupported-worklist-audit.pdf`. `TestOutputFixture58UnsupportedWorklistAudit`. Shape: 9 pages, 995 text runs, 3,728 strokes, 969 fills, 216 images.
- [x] 94.6.10 `output/fixture-59-apex-digital-landing.pdf`. `TestOutputFixture59ApexDigitalLanding` pins first-page and last-page text plus an image. Shape: 9 pages, 66 text runs, 57 strokes, 135 fills, 9 images.
- [x] 94.6.11 All files are split per PDF and remain far below the 2,000-line limit. The six passing fixture rows were verified in the 50-59 batch; fixture 57-59 pass independently of the open fixture 56 row.

Proof: `GOCACHE=/tmp/gowkhtmltopdf-fixture-test-cache-21-64 go test ./internal/convert -run '^TestOutputFixture(5[0-9])' -count=1` reaches the fixture 56 comparison and exits nonzero on the recorded differences.
