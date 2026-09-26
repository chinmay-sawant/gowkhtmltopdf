# Phase 94.5: Fixtures 40-49

> **Parent:** [94-canonical-output-position-tests.md](94-canonical-output-position-tests.md)
> **Status:** complete
> **Estimated effort:** M
> **Depends on:** 94.0 and 94.4
> **Unblocks:** 94.6

---

## Overview

This phase uses one test file per PDF under `internal/convert/fixturetests`. Fixtures 43,
47, and 49 include image anchors. Fixture 43 spans five pages, so its test
pins both the first-page and page-5 authored records.

## Checklist

- [x] 94.5.1 `output/fixture-40-transform-badge.pdf`. `TestOutputFixture40TransformBadge` pins the transformed text record. Shape: 1 page, 6 text runs, 152 strokes, 5 fills, 0 images.
- [x] 94.5.2 `output/fixture-41-has-selector.pdf`. `TestOutputFixture41HasSelector`. Shape: 1 page, 13 text runs, 23 strokes, 4 fills, 0 images.
- [x] 94.5.3 `output/fixture-42-container-inline-size.pdf`. `TestOutputFixture42ContainerInlineSize`. Shape: 1 page, 8 text runs, 8 strokes, 1 fill, 0 images.
- [x] 94.5.4 `output/fixture-43-complex-dossier.pdf`. `TestOutputFixture43ComplexDossier` pins page 1, page 5, and an image. Shape: 5 pages, 372 text runs, 361 strokes, 118 fills, 6 images.
- [x] 94.5.5 `output/fixture-44-receipt.pdf`. `TestOutputFixture44Receipt`. Shape: 1 page, 18 text runs, 19 strokes, 0 fills, 0 images.
- [x] 94.5.6 `output/fixture-45-purchase-order.pdf`. `TestOutputFixture45PurchaseOrder`. Shape: 1 page, 41 text runs, 70 strokes, 6 fills, 0 images.
- [x] 94.5.7 `output/fixture-46-contract.pdf`. `TestOutputFixture46Contract`. Shape: 1 page, 34 text runs, 11 strokes, 1 fill, 0 images.
- [x] 94.5.8 `output/fixture-47-certificate.pdf`. `TestOutputFixture47Certificate` pins text and image records. Shape: 1 page, 16 text runs, 12 strokes, 0 fills, 1 image.
- [x] 94.5.9 `output/fixture-48-shipping-document.pdf`. `TestOutputFixture48ShippingDocument`. Shape: 1 page, 37 text runs, 359 strokes, 5 fills, 0 images.
- [x] 94.5.10 `output/fixture-49-night-train-poster.pdf`. `TestOutputFixture49NightTrainPoster` pins text and image records. Shape: 1 page, 7 text runs, 1 stroke, 3 fills, 1 image.
- [x] 94.5.11 All ten targeted fixture tests passed in the 40-49 batch. The tests are split per PDF, so no generated test file approaches the 2,000-line limit.

Proof: `GOCACHE=/tmp/gowkhtmltopdf-fixture-test-cache-21-64 go test ./internal/convert/fixturetests -run '^TestOutputFixture(4[0-9])' -count=1` exited 0.
