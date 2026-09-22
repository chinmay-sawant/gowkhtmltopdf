# 95 - Pixel-level PDF visual regression tests (v0.2.7)

> **Parent:** `plans/0.2.7/README.md`
> **Status:** proposed; this ledger contains plans only
> **Estimated effort:** L
> **Owner:** `internal/convert/fixturetests`, sample-output tooling, and CI
> **Depends on:** one fixed PDF-to-image tool and manually approved fixture PDFs
> **Unblocks:** a whole-page visual regression check for fixtures 01-64
> **Number:** 95. Phase 94's page-operation checks remain until this ledger has replaced them.

---

## Overview

Replace the operation-by-operation fixture checks in phase 94 with comparisons
of the rendered pages. For each fixture, the test will convert the HTML again,
use Ghostscript, a PDF-to-image renderer, to turn the fresh PDF and its
approved reference PDF into page images, then compare every red, green, and
blue (RGB) pixel using the same pinned Ghostscript build.

Keep approved reference PDFs under `output/validated/`. Tests may read files
there but must never create or update them. A person reviews the candidate PDF
and its diff, then copies an approved candidate into that directory by hand.
The existing `make samples` cleanup only removes top-level
`output/fixture-*.pdf` files, so it does not reach `output/validated/`
(`Makefile:181-200`).

## Executive summary

The current suite extracts text, strokes, filled boxes, image placements, and
page boxes from each PDF. It compares those records with numeric tolerances
(`internal/pdf/page_ops.go:62-80`,
`internal/convert/fixturetests/output_ops_test.go:57-110`). This catches many
layout changes, but it does not compare the final page image. Comparing page
images checks the visible result and shows which region changed.

Do not compare PDF files byte for byte. The gate is visual: render both PDFs
with the same tool, settings, and fonts, then compare page count, image size,
and every RGB pixel. Require exact equality. If the test fails, report the
fixture and page, count changed pixels, and save a diff image under ignored
temporary output. Do not loosen the comparison to make an unexplained change
pass.

Pixel comparison does not name a changed HTML element and does not check
invisible PDF data such as link targets or embedded-font structure. Keep
`make golden` for PDF structure, page envelopes, fonts, images, URI
annotations, and ordered text (`internal/convert/golden_test.go:484-489`,
`internal/convert/golden_test.go:518-545`). Keep the focused layout tests in
`internal/layout/requested_fixture_regression_test.go`.

The scope is all numbered fixtures 01-64, including both fixture 29 PDFs.
That is 65 reference PDFs. Version and compliance copies, Python samples, and
wkhtmltopdf comparison PDFs are outside this phase.

## Phase 1: Set the comparison contract

### 95.0 Pin and prove PDF rasterization

- [ ] Choose one Ghostscript version for local and CI use. Record the exact version and fixed PDF-to-PNG options in the test documentation. The official [Ghostscript FAQ](https://ghostscript.com/faq/) documents PDF page output with the `png16m` device and a fixed resolution. Existing CI runs `make test` without installing a PDF renderer (`.github/workflows/ci.yml:9-34`), so add this requirement only to the visual-test job.
- [ ] Render the approved and fresh fixture 01 PDFs at the same resolution in dots per inch (DPI), color format, and edge-smoothing setting. Confirm equal page counts and image dimensions before comparing pixels.
- [ ] Add a small comparator check that changes one pixel and one image dimension. Prove both cases fail and name the page in the error.

### 95.1 Establish the manual baseline rule

- [ ] Create `output/validated/` as the only home for approved reference PDFs. Add a README entry that says tests only read this directory and no target creates or replaces its files.
- [ ] Define the approval steps: regenerate a candidate under `output/`, inspect it beside the diff image, manually copy the accepted PDF to `output/validated/`, then rerun the visual check.
- [ ] Make missing or extra fixture references fail. Never bootstrap, refresh, or delete a validated PDF from a test or Makefile target.

## Phase 2: Build and pilot the visual check

### 95.2 Implement the page raster comparator

- [ ] Add a test helper that invokes the chosen Ghostscript build on a PDF and returns one RGB image per page. Keep page images in a temporary or ignored directory.
- [ ] Compare every pixel exactly after checking page count and per-page dimensions. On mismatch, report the fixture, page, changed-pixel count, and largest color-channel difference; write a diff or overlay image for review.
- [ ] Add unit coverage for identical images, one changed pixel, changed page dimensions, changed page count, a missing reference, and rasterizer failure.

### 95.3 Prove fixture 01 end to end

- [ ] Manually review and place the fixture 01 reference at `output/validated/fixture-01-simple-invoice.pdf`.
- [ ] Convert `testdata/golden/fixture-01-simple-invoice.html` through the same request settings used for the existing sample (`internal/convert/fixturetests/fixture_helpers_test.go:78-112`), rasterize both PDFs, and pass on exact page-pixel equality.
- [ ] Introduce a local-only visual change to a candidate, confirm the test fails, inspect the generated diff, then discard that candidate without changing the approved PDF.

## Phase 3: Cover all fixture PDFs

### 95.4 Migrate fixtures 01-20

- [ ] Manually review and approve one PDF for each fixture 01-20. Do not copy candidates into `output/validated/` until the rendered comparison and visual review are complete.
- [ ] Run the visual check across fixtures 01-20. Confirm missing references and changed pixels fail.

### 95.5 Migrate fixtures 21-40

- [ ] Manually review and approve the fixture PDFs 21-40, including both fixture 29 files and fixture 36's HTML header/footer rendering.
- [ ] Run the visual check across this group using the same fixture request and font settings as the source corpus.

### 95.6 Migrate fixtures 41-64

- [ ] Manually review and approve the fixture PDFs 41-64, including local-font fixtures and the multi-page audit fixtures.
- [ ] Run the visual check across fixtures 41-64 and confirm the baseline inventory has exactly one entry for each of the 65 numbered PDFs.

## Phase 4: Retire operation checks and close the change

### 95.7 Replace the old fixture assertions

- [ ] After all 65 visual references pass, remove the per-fixture output-operation tests and their assertion helpers from `internal/convert/fixturetests/`.
- [ ] Search for remaining `ParsePageOps` callers. Remove `internal/pdf/page_ops.go` and its tests only if the visual migration leaves no separate use for that reader.
- [ ] Keep `make golden` and layout geometry tests. The raster check adds visual evidence; it does not replace PDF structure, text, links, or layout-unit contracts.

### 95.8 Add the CI gate and finish documentation

- [ ] Add `make visual-golden` and a CI job with the pinned Ghostscript build. Run the visual gate separately from `make test` so ordinary Go tests do not silently gain an undeclared system dependency.
- [ ] Update `plans/0.2.7/README.md`, `plans/README.md`, and the knowledge base with the manual approval flow, rendering settings, and current status.
- [ ] Run `make test`, `make golden`, `make visual-golden`, `make claim-scan`, and `make lint`. Record the final results here before closing the ledger.

## Dependencies

- 95.0 sets the fixed raster format and comparison behavior before fixture approvals.
- 95.1 sets the protected reference location before any baseline is copied.
- 95.2 and 95.3 prove the comparator with fixture 01 before the full corpus work.
- 95.4-95.6 cover every fixture before 95.7 removes the existing checks.
- 95.8 starts after the fixture suite and manual baseline inventory pass.

## Sources and current evidence

- The old tests compare parsed drawing records field by field and allow numeric tolerances (`internal/convert/fixturetests/output_ops_test.go:14-23`, `internal/convert/fixturetests/output_ops_test.go:57-110`); the former tolerance values are recorded in phase 94 (`plans/0.2.7/validator-test/94-canonical-output-position-tests.md:50-53`).
- `ParsePageOps` records text, strokes, fills, images, and page boxes, not a raster image (`internal/pdf/page_ops.go:62-80`). Its fill boxes are path bounds (`internal/pdf/page_ops.go:44-51`).
- The fixture helper uses the public conversion request and mirrors the sample settings, including font paths and header/footer companions (`internal/convert/fixturetests/fixture_helpers_test.go:57-112`).
- `make samples` rewrites the top-level fixture samples, while `output/README.md` says they are viewer-smoke artifacts rather than byte baselines (`Makefile:181-200`, `output/README.md:3-18`).
- The structural corpus checks PDF validity, page bounds, embedded fonts, images, URI annotations, and ordered text (`internal/convert/golden_test.go:484-545`).

## Out of scope

- Exact PDF-byte comparison, Chrome or wkhtmltopdf parity, Python output, and version/compliance variants.
- Automatically updating, approving, or deleting files under `output/validated/`.
