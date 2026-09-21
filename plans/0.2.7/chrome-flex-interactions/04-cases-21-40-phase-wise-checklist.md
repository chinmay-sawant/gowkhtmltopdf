# Phase-wise checklist for Chromium Flex cases 21-40

> Parent: `00-canonical-chrome-flex-interaction-plan.md`
> Child ledgers: `02-cases-21-30.md`, `03-cases-31-40.md`
> Status: complete
> Completed: 2026-09-20

## Closure rule

Every row is `[x]` only after the fixture, focused assertion, manifest entry,
and relevant validation passed on the final tree. Cases 34 and 40 validate the
engine's deterministic intrinsic-width contract. Their focused tests avoid
font-metric-only height comparisons.

## Case ledger

| Case | Evidence seam | Manifest result | Proof |
| --- | --- | --- | --- |
| 21 | Direct layout | `completed` | `TestChromeFixtureCase21RowWrapMargins` validates margin-bearing line cross size and four item rectangles. |
| 22 | Direct layout | `completed` | `TestChromeFixtureCase22ColumnGap` validates three items and both edge gaps. |
| 23 | Chromium reference | `completed` | `TestChromeFixtureCase23AspectRatioFeedback` validates nested 200 by 50 and 100 by 50 flex boxes. |
| 24 | Chromium reference | `completed` | `TestChromeFixtureCase24WritingModeMatrix` builds all eight logical-axis branches and validates the row-wrap reference rectangle. |
| 25 | Chromium reference | `completed` | `TestChromeFixtureCase25PercentageAbsoluteChild` validates the item, marker, and percentage absolute fill. |
| 26 | Direct layout | `completed` | `TestChromeFixtureCase26MinHeightPercentage` validates the definite min-height basis and percentage child. |
| 27 | Direct layout | `completed` | `TestChromeFixtureCase27ColumnPercentageHeight` validates the definite column percentage height. |
| 28 | Chromium reference | `completed` | `TestChromeFixtureCase28AspectRatioMinimumWidth` validates the transferred replaced-element minimum. |
| 29 | PDF golden | `completed` | `TestGoldenCorpusAllFixtures/fixture-29-wpt-break-nested-float-print.html` validates three pages and ordered semantic text; `TestChromeFixtureCase29NestedFloatLeft` pins the float to the left content edge. |
| 30 | Direct layout | `completed` | `TestChromeFixtureCase30ColumnAutoMargins` validates horizontal and vertical-writing auto-margin geometry. |
| 31 | Direct layout | `completed` | `TestChromeFixtureCase31ColumnReverseWrap` validates each named item after reverse-line placement. |
| 32 | Direct layout | `completed` | `TestChromeFixtureCase32FractionalFactors` validates fractional grow and shrink behavior across row, column, and vertical branches. |
| 33 | Chromium reference | `completed` | `TestChromeFixtureCase33ReplacedInputs` validates text, range, button, and deferred `calc()` widths. |
| 34 | Chromium reference | `completed` | `TestChromeFixtureCase34MaxContentContribution` validates row and column intrinsic width contributions. |
| 35 | Direct layout | `completed` | `TestChromeFixtureCase35BorderBoxReplacedCrossSize` validates both box-sizing frames and the 1px replaced child. |
| 36 | Direct layout | `completed` | `TestChromeFixtureCase36MaxWidthBaseSize` validates clamp and shrink-mode redistribution. |
| 37 | Direct layout | `completed` | `TestChromeFixtureCase37ReplacedRatioPrecision` validates the intrinsic SVG dimensions and offset. |
| 38 | Direct layout | `completed` | `TestChromeFixtureCase38WrappedGapGeometry` validates six items, two rows, and both gaps. |
| 39 | Chromium reference | `completed` | `TestChromeFixtureCase39VerticalOverflow` validates the negative overflow rectangle. |
| 40 | Chromium reference | `completed` | `TestChromeFixtureCase40MinContentContribution` validates the deterministic min-content width and item placement. |

## Phase 0: scope and ownership

- [x] Confirmed cases 21-40 occur exactly once in `test/chrome/manifest.json`.
- [x] Confirmed every case has a source path, fixture, Go target, expected behavior, and completed status.
- [x] Confirmed every fixture starts with a doctype and carries the manifest status marker.
- [x] Assigned direct layout, Chromium reference, or PDF golden evidence to every case.
- [x] Kept case 29 on the PDF golden seam because semantic pagination is its contract.
- [x] Used `scripts/chrome_rects.py` for the Chromium-reference measurements.

## Phase 1: fixtures and focused tests

- [x] Replaced cases 21-40 scaffolds with source-shaped fixtures.
- [x] Added fixture-backed tests in `internal/layout/flex_chrome_cases_21_40_fixture_test.go`.
- [x] Added the case 29 golden fixture and its page-count and semantic-text envelope.
- [x] Implemented shared fixes for margins, aspect-ratio feedback, logical wrapping, fractional factors, replaced sizing, calc widths, intrinsic widths, and max-width redistribution.

## Phase 2: status reconciliation

- [x] Set every case 21-40 manifest entry to `completed`.
- [x] Synchronized every case fixture's `Port status` marker with the manifest.
- [x] Confirmed no case 21-40 remains `scaffold`.
- [x] Confirmed no case 21-40 remains deferred in the checklist.
- [x] Confirmed the target checklist contains no open or partial rows.

## Phase 3: validation record

- [x] Ran the strict 21-40 layout suite with `-count=1`; all named tests passed with no skips.
- [x] Ran Chromium rectangle captures for the reference fixtures; exit code 0.
- [x] Ran the focused case 29 golden test; exit code 0 with exactly three pages and ordered semantic text.
- [x] Ran the manifest and 40-case PDF output tests; exit code 0.
- [x] Ran `make test`; exit code 0 on the final tree.
- [x] Ran `make golden`; exit code 0 on the final tree.
- [x] Ran `make claim-scan`; exit code 0 on the final tree.
- [x] Ran `make lint`; exit code 0 on the final tree.

## Evidence boundary

- [x] Confirmed all twenty cases have a source-backed fixture, named assertion, and completed manifest result.
- [x] Confirmed all twenty case rows are closed with `[x]`.

## Case 29 follow-up: float alignment (2026-09-21)

- [x] Reproduced the gap against the Puppeteer print reference: Chromium paints the green float at x=36..180pt, the Go PDF painted it at x=180..324pt on all three pages.
- [x] Traced the ghost float to the `flexItemBaseHeight` noEmit measure build and fixed it with the measure float-state quarantine in `internal/layout`.
- [x] Added `TestChromeFixtureCase29NestedFloatLeft`; the focused test passes with `-count=1`.
- [x] Re-ran the case 29 PDF comparison: all three pages match Chromium's green rect, and pages 2 and 3 are pixel-identical at 110 DPI.
- [x] Ran `make test`, `make golden`, `make claim-scan`, and `make size-check`; exit code 0.
- [x] Ran golangci-lint on the tree; exit code 0 (the temporary `zz_probe_case30_test.go` from the parallel case 30 session was excluded, and that file is not part of this change).

## Case 35 follow-up: transparent border paint (2026-09-21)

- [x] Reproduced the case 35 picture difference against the Puppeteer print reference: Chromium painted only the description panel, while the Go PDF added two solid black 8pt frames around the `border: 8pt solid transparent` `.case` boxes (8 extra stroke rows, pixel delta 3.34%).
- [x] Traced the root cause to the border color parse paths: `border.Color` is `[3]float64` and `parseBorder`, `setFourBorderColor`, and `setBorderColor` dropped the source alpha, so `transparent` painted opaque black.
- [x] Added `Transparent bool` to the `border` struct, set it from the parsed alpha in `parseBorder` and via the new `parseUsedColorAlpha` in `setFourBorderColor` / `setBorderColor`, and gated the border paint emitters `borderOpsSides`, `roundedBorderOps`, `roundedAccentBorderOps`, `emitBorders`, `emitThumbImageBottomSeparator`, and `inlineBorderVisible`. Layout width is unchanged.
- [x] Added `TestTransparentBorderPaintsNothing` (`internal/layout/style_backgrounds_borders_test.go`); it failed with 4 black ops before the fix, and it pins that the transparent border keeps its 8pt width while an opaque control still paints.
- [x] Restored the missing WPT `1x1-green.png` src on both case 35 `<img>` children, matching the original WPT source and the case 28 asset convention.
- [x] Re-ran the Puppeteer comparison after the fix: the black frames are gone, Go drawing rows dropped 13 to 5, both engines paint the two 0.75pt images, and the pixel delta fell from 3.34% to 0.82% (under the 1.00% threshold). Remaining deltas are the pre-existing page-size, font, and 0.25pt round-off differences.
- [x] Regenerated `test/chrome/pdf/case-35-wpt-flex-cross-size-border-box.pdf`.
- [x] Ran `make test`, `make golden`, and `make lint`; exit code 0 on the final tree.

Known same-class gaps left open (pre-existing, not regressed by this change): `hr` borders (`buildHR`), collapsed-table borders, and `column-rule-color` store `[3]float64` colors without the transparency flag and still paint a transparent color.
