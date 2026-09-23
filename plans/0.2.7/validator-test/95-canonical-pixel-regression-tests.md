# 95 - Pixel-level PDF visual regression tests (v0.2.7)

> **Parent:** `plans/0.2.7/README.md`
> **Status:** In progress. The fixture 01 Go comparator pilot passes.
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

- [x] Pin the local comparator to Ghostscript 10.08.0 and document `png16m`, 150 DPI, and text and graphics smoothing level 4 (`internal/convert/fixturetests/pixel_regression_test.go:16-20`, `internal/convert/fixturetests/pixel_regression_test.go:77-100`, `output/README.md:14-53`).
- [ ] Install the pinned Ghostscript build in a separate CI visual-test job. Current CI has no renderer setup (`.github/workflows/ci.yml:9-34`).
- [x] Render the approved and fresh fixture 01 PDFs with those settings. The opt-in comparison passed with equal page counts, image sizes, and RGB pixels (`GOWKHTMLTOPDF_VISUAL_GS=/tmp/gowkhtmltopdf-ghostscript-install/bin/gs GOCACHE=/tmp/gowkhtmltopdf-gocache go test ./internal/convert/fixturetests -run '^TestVisualGoldenFixture01' -count=1 -v`).
- [x] Change one pixel and one page dimension in comparator tests. Both cases fail and name the fixture and page (`internal/convert/fixturetests/pixel_regression_cases_test.go:27-72`).

### 95.1 Establish the manual baseline rule

- [x] Create `output/validated/` for approved references and document that tests only read it. The test fails when the requested PDF is missing (`internal/convert/fixturetests/pixel_regression_test.go:36-59`, `internal/convert/fixturetests/pixel_regression_cases_test.go:74-82`, `output/README.md:14-53`).
- [x] Document the fixture 01 candidate command and manual review, copy, and rerun steps (`output/README.md:21-53`). The test itself does not write into `output/validated/` (`internal/convert/fixturetests/pixel_regression_test.go:48-59`, `internal/convert/fixturetests/pixel_regression_test.go:158-195`).
- [ ] Compare the reference directory with the full expected 65-PDF fixture inventory so extra and missing entries fail. Only fixture 01 has an approved reference so far.

## Phase 2: Build and pilot the visual check

### 95.2 Implement the page raster comparator

- [x] Add a helper that checks the pinned Ghostscript version, rasterizes every page to PNG, and decodes the images from temporary storage (`internal/convert/fixturetests/pixel_regression_test.go:62-147`).
- [x] Compare page count and dimensions before exact RGB comparison. A mismatch reports the fixture, page, changed-pixel count, maximum channel difference, and saved diff path (`internal/convert/fixturetests/pixel_regression_test.go:198-290`, `internal/convert/fixturetests/pixel_regression_test.go:350-382`).
- [x] Cover identical images, a changed pixel, dimensions, page count, missing reference, and Ghostscript failure (`internal/convert/fixturetests/pixel_regression_cases_test.go:27-101`). The focused package tests pass.

### 95.3 Prove fixture 01 end to end

- [x] Review and place the fixture 01 reference at `output/validated/fixture-01-simple-invoice.pdf`. The current candidate at `output/fixture-01-simple-invoice-candidate.pdf` is a one-page PDF; its generation command is in `output/README.md:21-39`.
- [x] Convert the fixture through the shared request settings, rasterize both PDFs, and pass on exact page-pixel equality (`internal/convert/fixturetests/pixel_regression_cases_test.go:103-128`). The command is recorded under 95.0.
- [x] Read the saved CLI candidate from `output/fixture-01-simple-invoice-candidate.pdf` and compare it with the approved reference. It passes at exact pixel equality (`output/README.md:52-67`, `internal/convert/fixturetests/pixel_regression_cases_test.go:194-213`).
- [x] Change the title color only in the test's temporary fixture copy. The comparator detects the mismatch and writes a diff, which was inspected at `/tmp/gowkhtmltopdf-visual-fixture01-mutation-diff/fixture-01-simple-invoice-page-01-diff.png` (`internal/convert/fixturetests/pixel_regression_cases_test.go:130-180`). The approved PDF remains unchanged.

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

- [ ] Add `make visual-golden` and a CI job with the pinned Ghostscript build. No `visual-golden` Make target exists yet. Keep it separate from `make test` so ordinary tests do not gain an undeclared system dependency.
- [x] Update `plans/0.2.7/README.md`, `plans/README.md`, `output/README.md`, and the knowledge base with the approval flow, rendering settings, and current status.
- [x] Rerun `GOCACHE=/tmp/gowkhtmltopdf-gocache make test` on the final Go tree. It exits 0, including `internal/convert/fixturetests`.
- [x] Rerun `GOCACHE=/tmp/gowkhtmltopdf-gocache make golden`. It exits 0 and all structural fixture checks pass.
- [x] Rerun `GOCACHE=/tmp/gowkhtmltopdf-gocache make claim-scan` after the documentation updates. It exits 0 with `claim-scan: clean`.
- [ ] Pass `make lint` before closing phase 95. The 2026-09-23 run exits 2 on existing `paralleltest` findings in `output_fixture_*.go` and one `wsl` finding in `internal/pdf/shape_gotext.go:532`. The new pixel test files have no lint findings.

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
- The Go pilot fixes Ghostscript 10.08.0, the PNG device, resolution, and smoothing settings. The comparator checks page count and dimensions, then exact RGB values and writes a diff for mismatches (`internal/convert/fixturetests/pixel_regression_test.go:16-20`, `internal/convert/fixturetests/pixel_regression_test.go:77-100`, `internal/convert/fixturetests/pixel_regression_test.go:198-290`).
- Fresh fixture conversion, the saved CLI candidate, and the temporary color-mutation check pass with pinned Ghostscript. The mutation writes an inspected diff. The approved directory has only fixture 01; no Make target or CI job exists yet (`internal/convert/fixturetests/pixel_regression_cases_test.go:103-180`, `internal/convert/fixturetests/pixel_regression_cases_test.go:194-213`, `output/validated/`).
- Final local gates: `make test`, `make golden`, and `make claim-scan` pass. `make lint` exits 2 on existing `paralleltest` findings in `output_fixture_*.go` and one `wsl` finding in `internal/pdf/shape_gotext.go:532`. The two new pixel-test files report no lint findings.

## Out of scope

- Exact PDF-byte comparison, Chrome or wkhtmltopdf parity, Python output, and version/compliance variants.
- Automatically updating, approving, or deleting files under `output/validated/`.
