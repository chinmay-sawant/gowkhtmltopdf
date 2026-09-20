# Phase-wise checklist for Chromium Flex cases 21-40

> **Parent:** `00-canonical-chrome-flex-interaction-plan.md`
> **Child ledgers:** `02-cases-21-30.md`, `03-cases-31-40.md`
> **Status:** planned
> **Checklist preparation:** complete on 2026-09-20
> **Estimated effort:** XL
> **Scope:** cases 21-40. Case 20 is already in the completed 01-20 batch.

---

## Overview

Convert the next twenty Chromium Flexbox fixtures from scaffolds into honest,
source-backed evidence. The work covers cases 21 through 40. It does not
implement the engine in this planning step.

Every case must keep its fixture, evidence seam, focused test, and manifest
status aligned. A readable PDF proves that conversion produced a PDF. It does
not prove box geometry.

The current manifest marks all cases 21-40 as `scaffold`. The existing Go tests
contain useful adaptations for many of them, but those tests must consume
source-faithful fixtures before a case can leave `scaffold`.

## Evidence rules

Use exactly one primary evidence seam per case:

| Seam | Cases | Proof |
| --- | --- | --- |
| Direct layout | 21, 22, 26, 27, 30, 31, 32, 35, 36, 37, 38 | Go box sizes, positions, line membership, or intrinsic dimensions |
| Chromium reference | 23, 24, 25, 28, 33, 34, 39, 40 | Reproducible DOM rectangle or overflow capture compared with Go geometry |
| PDF golden fixture | 29 | Page bounds, page count, ordered semantic text, and fragmented-content evidence |

Generated PDFs remain inspection artifacts. They do not close direct-layout or
Chromium-reference rows.

## Phase 0: scope and evidence lock

### 0.1 Inventory

- [x] Confirm that cases 21-40 appear once each in `test/chrome/manifest.json`.
  Evidence: the manifest entries for `wpt-flow-row-wrap` through
  `wpt-flex-container-min-content`.
- [x] Confirm that each case has a fixture under `test/chrome/cases/`, a
  Chromium source path, an expected behavior, a Go target, and a status field.
  Evidence: `test/chrome/manifest.json` and `test/chrome/manifest_test.go`.
- [x] Confirm that all twenty current statuses are `scaffold`.
  Evidence: current manifest status entries for cases 21-40.
- [x] Confirm that the next batch is 21-40, not 20-40. Case 20 is already
  completed in the 01-20 ledger.

### 0.2 Evidence ownership

- [x] Assign cases 21, 22, 26, 27, 30, 31, 32, 35, 36, 37, and 38 to direct
  layout evidence in `internal/layout`.
- [x] Assign cases 23, 24, 25, 28, 33, 34, 39, and 40 to Chromium-reference
  evidence using `scripts/chrome_rects.py` and Go geometry.
- [x] Assign case 29 to the PDF golden corpus in `testdata/golden/` and
  `internal/convert/golden_test.go`.
- [x] Record the rule that a skipped strict test is a known engine boundary,
  not a completed case.

## Phase 1: low-risk direct geometry

### 1.1 Case 21: row wrapping with margins

- [ ] Replace `test/chrome/cases/case-21-wpt-flow-row-wrap.html` with the
  source-faithful fixed-width and margin pattern.
- [ ] Adapt `TestChromeFlexRowWrap` in
  `internal/layout/flex_chrome_flow_test.go` to consume case 21's fixture.
- [ ] Assert line membership, item order, line cross sizes, and container
  height.
- [ ] Run the focused case 21 test and record its exit code.
- [ ] Change the manifest status only after the fixture and focused evidence
  agree. Use `completed`, `blocked`, or `unsupported` honestly.

### 1.2 Case 22: flexible column gap

- [ ] Replace `test/chrome/cases/case-22-wpt-gap-002-ltr.html` with the
  source-faithful flexible-child setup.
- [ ] Adapt `TestChromeFlexColumnGapFlexibleChildren` in
  `internal/layout/flex_chrome_gaps_test.go` to consume the fixture.
- [ ] Assert that the gap appears between items and not at either edge.
- [ ] Run the focused case 22 test and record its exit code.
- [ ] Change the manifest status only after the fixture and focused evidence
  agree.

### 1.3 Case 35: border-box cross size

- [ ] Replace `test/chrome/cases/case-35-wpt-flex-cross-size-border-box.html`
  with the source-faithful replaced-image and box-sizing variants.
- [ ] Adapt `TestChromeFlexCrossSizeBorderBox` in
  `internal/layout/flex_chrome_borderbox_test.go` to consume the fixture.
- [ ] Assert equivalent content cross sizes for the border-box and content-box
  branches.
- [ ] Run the focused case 35 test and record its exit code.
- [ ] Change the manifest status only after the fixture and focused evidence
  agree.

### 1.4 Case 37: replaced-element ratio precision

- [ ] Replace `test/chrome/cases/case-37-blink-replaced-aspect-ratio-precision.html`
  with an intrinsic 29 by 22 SVG fixture.
- [ ] Adapt `TestChromeFlexReplacedAspectRatioPrecision` in
  `internal/layout/flex_chrome_replaced_test.go` to consume the fixture.
- [ ] Assert the intrinsic replaced-element dimensions inside the flex item.
- [ ] Run the focused case 37 test and record its exit code.
- [ ] Change the manifest status only after the fixture and focused evidence
  agree.

### 1.5 Case 38: wrapped gap geometry

- [ ] Replace `test/chrome/cases/case-38-blink-gap-decorations-basic.html`
  with six fixed items, two rows, and the source's main and cross gaps.
- [ ] Adapt `TestChromeFlexGapGeometryWrapping` in
  `internal/layout/flex_chrome_gaps_test.go` to consume the fixture.
- [ ] Assert known intersections for both gap directions and line assignment.
- [ ] Run the focused case 38 test and record its exit code.
- [ ] Change the manifest status only after the fixture and focused evidence
  agree.

## Phase 2: containing blocks and cross-axis margins

### 2.1 Case 25: percentage absolute child

- [ ] Replace `test/chrome/cases/case-25-wpt-flex-item-percentage-abspos.html`
  with the source's in-flow marker that establishes the flex-item size.
- [ ] Adapt `TestChromeFlexPercentageAbsposChild` in
  `internal/layout/flex_chrome_replaced_test.go` to consume the fixture.
- [ ] Assert the percentage absolute child against the actual flex-item
  containing block without changing the flex-item size.
- [ ] Record the result separately from the percentage-height cases because
  the absolute-positioning path has its own containing-block rules.
- [ ] Change the manifest status only after the named evidence passes or the
  engine limitation is recorded as `blocked` or `unsupported`.

### 2.2 Case 26: definite size from min-height

- [ ] Replace `test/chrome/cases/case-26-wpt-definite-sizes-002.html` with the
  source's red parent and percentage child structure.
- [ ] Adapt `TestChromeFlexDefiniteSizeFromMinHeight` in
  `internal/layout/flex_chrome_percentages_test.go` to consume the fixture.
- [ ] Assert that `min-height` establishes the percentage basis for the nested
  child.
- [ ] Run the focused case 26 test and record its exit code.
- [ ] Change the manifest status only after the fixture and focused evidence
  agree.

### 2.3 Case 27: percentage height in a column item

- [ ] Replace `test/chrome/cases/case-27-wpt-percentage-heights-005.html` with
  the source's definite column and green percentage-child structure.
- [ ] Adapt `TestChromeFlexPercentageHeightColumnItem` in
  `internal/layout/flex_chrome_percentages_test.go` to consume the fixture.
- [ ] Assert the parent and child heights and the absence of unintended parent
  overflow.
- [ ] Run the focused case 27 test and record its exit code.
- [ ] Change the manifest status only after the fixture and focused evidence
  agree.

### 2.4 Case 30: column auto margins

- [ ] Replace `test/chrome/cases/case-30-wpt-auto-margins-column.html` with the
  source's column, auto-margin, `align-self: center`, and writing-mode branches.
- [ ] Add or adapt a focused test beside the column alignment tests in
  `internal/layout/flex_chrome_alignment_test.go`.
- [ ] Assert main-axis auto-margin distribution and cross-axis centering.
- [ ] Run the focused case 30 test and record its exit code.
- [ ] Change the manifest status only after the fixture and focused evidence
  agree.

## Phase 3: flex algorithm edge cases

### 3.1 Case 31: column-reverse multiline placement

- [ ] Replace `test/chrome/cases/case-31-wpt-column-reverse-multiline.html`
  with the source's `column-reverse` and `wrap` setup.
- [ ] Adapt `TestChromeFlexColumnReverseMultiline` in
  `internal/layout/flex_chrome_flow_test.go` to consume the fixture.
- [ ] Assert finalized line height, line packing, and item positions.
- [ ] Keep strict assertions if the engine is blocked by missing column wrap;
  record the exact source location and mark the case `blocked` rather than
  calling a skipped test a pass.
- [ ] Change the manifest status only after the focused evidence is recorded.

### 3.2 Case 32: factors below one

- [ ] Replace `test/chrome/cases/case-32-wpt-flex-factor-less-than-one.html`
  with named row, column, and shrink branches from the source.
- [ ] Adapt `TestChromeFlexFactorLessThanOneRowAndColumn` in
  `internal/layout/flex_chrome_sizing_test.go` to consume the fixture.
- [ ] Assert proportional growth and shrink behavior without rounding away
  unused free space.
- [ ] Keep the existing strict skipped subtests for unsupported axes until the
  underlying algorithm is fixed or the boundary is documented.
- [ ] Change the manifest status only after the focused evidence is recorded.

### 3.3 Case 36: flex base size before max-width clamp

- [ ] Replace `test/chrome/cases/case-36-wpt-flex-base-size-max-width.html`
  with the source's 300px content child and max-width freeze setup.
- [ ] Adapt `TestChromeFlexBaseSizeIgnoresMaxWidth` in
  `internal/layout/flex_chrome_sizing_test.go` to consume the fixture.
- [ ] Assert that the base size uses content before max-width freezing and
  redistribution.
- [ ] Preserve the strict blocked result if shrink-mode regrowth remains
  unavailable.
- [ ] Change the manifest status only after the focused evidence is recorded.

## Phase 4: aspect ratio and intrinsic contributions

### 4.1 Case 23: nested aspect-ratio cross-size feedback

- [ ] Replace `test/chrome/cases/case-23-wpt-aspect-ratio-cross-size-002.html`
  with the source's nested outer and inner flex containers.
- [ ] Adapt `TestChromeFlexAspectRatioCrossSizeFeedback` in
  `internal/layout/flex_chrome_aspect_test.go` to consume the fixture.
- [ ] Capture Chromium rectangles with reproducible conditions and compare the
  nested width and height against Go geometry.
- [ ] Record the aspect-ratio timing limitation if the nested feedback remains
  blocked.
- [ ] Change the manifest status only after the fixture and reference evidence
  agree.

### 4.2 Case 28: automatic minimum width from aspect ratio

- [ ] Replace `test/chrome/cases/case-28-wpt-flex-minimum-width-aspect.html`
  with the source's constrained replaced 200 by 200 image.
- [ ] Adapt `TestChromeFlexMinimumWidthAspectRatio` in
  `internal/layout/flex_chrome_aspect_test.go` to consume the fixture.
- [ ] Capture the Chromium width reference and compare the transferred
  aspect-ratio minimum with Go geometry.
- [ ] Change the manifest status only after the source-faithful fixture and
  reference evidence agree.

### 4.3 Case 34: max-content contribution

- [ ] Replace `test/chrome/cases/case-34-wpt-flex-container-max-content.html`
  with a reduced source-faithful matrix covering margins, padding, and borders.
- [ ] Adapt `TestChromeFlexContainerMaxContentContribution` in
  `internal/layout/flex_chrome_intrinsic_test.go` to consume the fixture.
- [ ] Capture the Chromium intrinsic width and compare outer item contributions
  with Go measurement.
- [ ] Record the missing item-margin contribution as `blocked` if it remains
  unresolved.
- [ ] Change the manifest status only after the fixture and reference evidence
  agree.

### 4.4 Case 40: min-content contribution

- [ ] Replace `test/chrome/cases/case-40-wpt-flex-container-min-content.html`
  so it tests the source's `min-content` contribution rather than a generic
  `max-content` width.
- [ ] Adapt `TestChromeFlexContainerMinContentContribution` in
  `internal/layout/flex_chrome_intrinsic_test.go` to consume the fixture.
- [ ] Capture the Chromium intrinsic width and compare the largest constrained
  item contribution with Go measurement.
- [ ] Record unsupported `min-content` handling as `blocked` or `unsupported`
  if the engine cannot express the source behavior.
- [ ] Change the manifest status only after the fixture and reference evidence
  agree.

## Phase 5: logical axes and replaced controls

### 5.1 Case 24: writing-mode matrix

- [ ] Replace `test/chrome/cases/case-24-wpt-writing-mode-006.html` with all
  source branches for row, column, reverse, and wrap-reverse in vertical
  writing modes.
- [ ] Adapt `TestChromeFlexWritingModeMatrix` in
  `internal/layout/flex_chrome_writing_test.go` to consume the fixture.
- [ ] Capture Chromium rectangles for each branch and compare logical ordering
  with Go geometry.
- [ ] Split passing horizontal branches from blocked vertical or RTL branches
  in the test output and manifest decision.
- [ ] Change the manifest status only after the fixture and reference evidence
  agree.

### 5.2 Case 33: replaced flex item minimum

- [ ] Replace `test/chrome/cases/case-33-wpt-flex-item-compressible.html` with
  the source's replaced input variants and specified-size suggestions.
- [ ] Adapt `TestChromeFlexReplacedItemCompressible` in
  `internal/layout/flex_chrome_replaced_test.go` to consume the fixture.
- [ ] Capture Chromium rectangles for each replaced-input branch and compare
  the automatic minimum behavior with Go geometry.
- [ ] Record any replaced-element limitation as `blocked` or `unsupported`.
- [ ] Change the manifest status only after the fixture and reference evidence
  agree.

### 5.3 Case 39: row-reverse vertical overflow

- [ ] Replace `test/chrome/cases/case-39-blink-scrollbars-row-reverse-vrl.html`
  with the source's large overflowing child, scrollbar, row-reverse, and
  `vertical-rl` setup.
- [ ] Adapt `TestChromeFlexScrollbarsRowReverseVRL` in
  `internal/layout/flex_chrome_borderbox_test.go` to consume the fixture.
- [ ] Capture Chromium negative physical coordinates and overflow behavior with
  reproducible conditions.
- [ ] Keep writing-mode and overflow limitations distinct in the final status.
- [ ] Change the manifest status only after the fixture and reference evidence
  agree.

## Phase 6: print fragmentation

### 6.1 Case 29: nested float fragmentation

- [ ] Replace `test/chrome/cases/case-29-wpt-break-nested-float-print.html` with
  the source's page size, nested float, and three-page content.
- [ ] Add a source-faithful golden fixture under `testdata/golden/` without
  using PDF readability as a geometry substitute.
- [ ] Add the page-count envelope, page-local bounds, and ordered semantic text
  needles to `internal/convert/golden_test.go`.
- [ ] Run the focused case 29 golden test and record its exit code.
- [ ] Run `make golden` only in the final closure phase after all layout and
  pagination changes are complete.
- [ ] Change the manifest status only after the golden evidence passes or the
  fragmentation boundary is recorded.

## Phase 7: batch closure

### 7.1 Status reconciliation

- [ ] Confirm every case 21-40 has one reviewed fixture, one named evidence
  seam, one focused test or golden assertion, and one honest manifest status.
- [ ] Confirm no case remains `scaffold` when its evidence has been completed,
  blocked, or declared unsupported.
- [ ] Confirm no case is marked `completed` because only its PDF conversion
  succeeded.
- [ ] Keep the existing case 21-30 and 31-40 child ledgers synchronized with
  this checklist. Move any deferred row with an explicit pointer instead of
  duplicating active work.

### 7.2 Validation gates

- [ ] Run focused `internal/layout` tests for all direct-layout cases with
  `-count=1` and record each result.
- [ ] Run the Chromium-reference capture and comparison checks for cases 23,
  24, 25, 28, 33, 34, 39, and 40.
- [ ] Run the focused golden test for case 29.
- [ ] Run `make test` once after the full batch is implemented.
- [ ] Run `make golden` once after the full batch if layout, paint, or
  pagination changed.
- [ ] Run `make claim-scan` if documentation, fixture claims, or frontend
  content changed.
- [ ] Run `make lint` at the repository's normal final integration point, not
  during this planning-only step.

## Dependencies and known boundaries

- Cases 25-27 share containing-block behavior. Keep absolute positioning and
  percentage-height results separate even when they fail in related code.
- Cases 31, 32, and 36 share flex algorithm paths. A passing row branch does
  not prove column wrapping, fractional shrink, or shrink-mode regrowth.
- Cases 23, 28, 34, and 40 require intrinsic or aspect-ratio evidence. Do not
  replace reference geometry with a text or PDF assertion.
- Cases 24 and 39 require logical-axis and overflow reference captures. The
  current writing-mode limitations are recorded in `internal/layout/flex.go`.
- Case 29 belongs to the PDF pipeline and must use page and semantic evidence.

## Planning closure

- [x] Planning scope corrected from 20-40 to 21-40.
- [x] Two read-only sub-agent scans completed with no repository changes.
- [x] Existing fixtures, manifest statuses, ledgers, and reusable tests were
  inspected before writing this checklist.
- [x] Checklist rows are atomic and tied to an evidence seam.
- [x] No implementation status was falsely marked complete.

## Out of scope for this checklist creation step

- Implementing or rewriting any case fixture.
- Editing Go layout, PDF, or test code.
- Changing manifest statuses for cases 21-40.
- Running Git commands.
- Adding command files or helper scripts.
