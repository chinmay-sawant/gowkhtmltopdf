# Chromium Flexbox interaction plan for v0.2.7

## Status

The Chromium source map and the Go-side inventory are complete. Cases 21-40
now have source-shaped fixtures, focused evidence, and final manifest results:
twenty completed cases.

The first converted interaction is the column flex container with centered
auto-width children. Its Go regression test checks the child center against the
container center instead of only checking that the child moved away from the
left edge. The same interaction also has a raw inline HTML PNG regression with
a fixed 500px viewport. Its Chromium source entry is an explicit adaptation of
the broader `flex-align.html` test, not a line-for-line port of one browser
case.

The first broader conversion wave now has direct layout tests for flex sizing,
main-axis alignment, auto margins, wrapping, reverse flow, and multiline
`align-content`. The sizing tests also exposed and fixed max-width
redistribution and fractional grow-factor behavior in the shared row algorithm.

The first conversion wave covers the first 10 manifest cases with five
parallel two-case workers. Cases 1 through 10 now have passing direct layout
tests. Case 4 records Chrome's fractional constrained-shrink geometry instead
of rounding the two item sizes to different integers. The second wave has now
converted cases 11 through 20 into source-faithful fixtures with named focused
tests, Chromium rectangle probes where needed, and completed manifest status.
The individual case fixtures and focused tests are now the canonical evidence;
the former consolidated review HTML and PDF have been removed. The 21-40
closure is recorded in the shared phase-wise checklist and its two child
ledgers.

## Execution ledgers

The remaining cases are split into three ten-case ledgers so each batch has a
small owner set, a clear validation gate, and a separate closure decision:

- [Phase-wise checklist for cases 21-40](04-cases-21-40-phase-wise-checklist.md):
  cross-phase execution ledger and evidence rules for the next twenty cases.
- [Cases 11-20](01-cases-11-20.md): wrapping, alignment, margins, flex basis,
  direction, and automatic minimum size.
- [Cases 21-30](02-cases-21-30.md): wrapping and gaps, aspect ratio, writing
  modes, definite percentages, absolute positioning, print fragmentation, and
  column auto margins.
- [Cases 31-40](03-cases-31-40.md): reverse wrapping, fractional factors,
  minimum size, intrinsic sizing, border-box sizing, replaced elements, gaps,
  overflow, and min-content.

Each ledger owns only its case range. The parent plan owns the shared evidence
rules and the final interaction-coverage gate.

## Definition of done

- The 40 cases remain tied to exact Chromium source paths.
- Every case has a static HTML input and one named expected behavior.
- Direct layout cases use Go box geometry assertions.
- Print cases use the repository's PDF page and semantic checks.
- Reference cases compare the same input in Chromium and the Go renderer.
- A later coverage check reports which interaction families have no case.

## Evidence boundary

The local `chromium/` directory is a shallow, blob-filtered source checkout.
It contains the Blink Flexbox implementation, the legacy Flexbox tests, and
the external WPT Flexbox tests. This run did not build Chromium.

Four read-only workers inspected separate slices. They found that Chromium
C++ tests cannot be copied into Go because they inspect Blink fragments,
constraint spaces, lifecycle state, and scroll state. Static HTML and CSS
inputs can be reused. JavaScript-generated matrices need fixed cases. Reftests
need an explicit reference comparison.

## Case allocation

| Go target | Count | First owner | Gate |
| --- | ---: | --- | --- |
| `layout-unit` | 24 | `internal/layout` | Direct box positions and sizes |
| `chrome-reference` | 15 | Chromium plus the Go CLI | Matching selected geometry or pixels |
| `golden-fixture` | 1 | `internal/convert` | Page bounds and semantic PDF checks |

## Phase 0. Source map and scope

- [x] Use a shallow Chromium checkout with only the Blink Flexbox and test
  paths needed for this study.
- [x] Select exactly 40 high-value cases.
- [x] Record the source path, interaction family, expected result, and Go
  target in `test/chrome/manifest.json`.
- [x] Keep the checkout and the local decision trail out of the repository.

Proof uses `python3 scripts/generate_chrome_flex_cases.py` and
`go test ./test/chrome`.

## Phase 1. Porting scaffold

- [x] Generate one static HTML scaffold for each mapped case.
- [x] Validate the count, unique IDs, safe fixture paths, doctype, and source
  paths when the checkout is present.
- [x] Strengthen the column plus centered cross-axis regression in
  `internal/layout/flex_test.go` with an exact geometry assertion.
- [x] Retarget the existing alignment inventory slot to the column plus center
  input and add the raw HTML PNG regression in `internal/imageout`.
- [x] Replace the first 20 scaffolds with reviewed cases that preserve the
  Chromium behavior without browser-only JavaScript. Cases 21 through 40 are
  now covered by the completed 21-40 execution ledgers.

The scaffold is an inventory check. It is not an engine pass.

## Phase 2. Direct layout-unit cases

Port the 24 `layout-unit` cases in groups that share one layout decision.

- [x] Convert the sizing group with grow, scaled shrink, max-width freezing,
  and fractional grow-factor geometry in
  `internal/layout/flex_chrome_sizing_test.go`.
- [x] Convert the alignment group with `justify-content` and row auto margins
  in `internal/layout/flex_chrome_alignment_test.go`.
- [x] Convert the flow group with row and column wrapping, reverse flow, and
  multiline `align-content` in `internal/layout/flex_chrome_flow_test.go`.

1. Flex basis, grow, shrink, min, and max constraints.
2. Main-axis alignment, cross-axis alignment, and auto margins.
3. Direction, reverse flow, wrapping, and align-content.
4. Gaps, intrinsic sizing, aspect ratio, and replaced elements.
5. Definite percentages and border-box cross sizes.

Each case ends with a direct `internal/layout` geometry assertion. A case does
not move to the next group until its input, expected geometry, and targeted Go
test pass.

### Historical exploratory wave context

The following exploratory results remain useful as background for future
waves, but they are not the current manifest status. The active first-10
ledger is recorded in the status section above and in `test/chrome/manifest.json`.

Ten workers, one property area each, disjoint test files under
`internal/layout/flex_chrome_*.go`.

| Property area | Cases | Outcome |
| --- | --- | --- |
| alignment | legacy-multiline-align-self, wpt-align-items-stretch, wpt-auto-margins-column | horizontal coverage passes; baseline, stretch-with-margins, and column cross auto-margin branches skip on named gaps |
| sizing | wpt-flex-basis-011, wpt-flex-factor-less-than-one, wpt-flex-base-size-max-width | row fractional grow passes; percentage basis under stretch, fractional height grow/shrink, and post-shrink regrow skip |
| flow + print | legacy-multiline, legacy-multiline-align-content-column, wpt-column-reverse-multiline, wpt-break-nested-float-print | wrap-reverse and row-reverse pass; column wrap cases skip; the print case remains a separate pending case |
| gaps | wpt-gap-002-ltr, blink-gap-decorations-basic | both pass on current geometry |
| intrinsic sizing | legacy-columns-auto-size, wpt-flex-container-max-content, wpt-flex-container-min-content | content and padding paths pass; margins, `min-height:0`, and min-content keywords skip |
| aspect ratio | wpt-aspect-ratio-cross-size-002, wpt-flex-minimum-width-aspect | transferred minimum passes; nested cross-size feedback skips |
| replaced + abspos | blink-replaced-aspect-ratio-precision, wpt-flex-item-compressible, wpt-flex-item-percentage-abspos | replaced sizing passes; abspos percentage height skips |
| definite percentages | legacy-definite-main-size, wpt-definite-sizes-002, wpt-percentage-heights-005 | all three skip; containing-block height is not threaded to children |
| border-box + min-size | wpt-flex-cross-size-border-box, wpt-min-size-auto-overflow-clip, blink-scrollbars-row-reverse-vrl | border-box passes; overflow-clip min-size and vertical writing skip |
| writing modes + direction + baseline | legacy-flex-align-vertical-writing, legacy-flex-flow-orientations, legacy-flex-flow, legacy-flex-flow-auto-margins, legacy-flex-align-baseline, wpt-rtl-flow-reverse, wpt-writing-mode-006 | horizontal-tb branches pass; vertical, rtl, and baseline branches skip |

Twelve cases are fully blocked and keep strict assertions with `file:line`
skip reasons: wpt-flex-basis-011, legacy-multiline-align-content-column,
wpt-column-reverse-multiline, wpt-flex-container-min-content,
wpt-aspect-ratio-cross-size-002, legacy-definite-main-size,
wpt-definite-sizes-002, wpt-percentage-heights-005,
wpt-min-size-auto-overflow-clip, blink-scrollbars-row-reverse-vrl,
wpt-rtl-flow-reverse, wpt-flex-base-size-max-width.

Engine gap ledger, owner `internal/layout`:

1. Containing-block height is not threaded to children: `layout.go:2051`,
   `layout.go:2303`, `layout.go:2083`; abspos path `layout.go:2346`. Blocks the
   definite-percentage cases and the abspos fill height.
2. Flex fractional factors: grow heights divide by `growSum` (`flex.go:1649`);
   shrink sums below one are not clamped (`flex.go:1070`, `flex.go:1657`);
   regrow after a shrink-mode freeze is missing (`flex.go:830`).
3. Percentage flex-basis under stretch: `flex.go:1277`, `flex.go:1170`,
   `flex.go:1460`.
4. Column wrap is not implemented: `flex.go:1528`, `flex.go:1616`,
   `flex.go:1567`.
5. Writing modes are ignored: `flex.go:84`.
6. `direction: rtl` is ignored: `flex.go:84`.
7. Reverse flows pack at main-start: `flex.go:306`, `flex.go:1101`,
   `flex.go:1539`.
8. Margins in flex: dropped from placement (`flex.go:1272`, `flex.go:1294`,
   `flex.go:1765`), wrong free-space base (`flex.go:1049`, `flex.go:1397`),
   column main margins (`flex.go:1436`, `flex.go:1777`), column cross auto
   margins (`flex.go:1808`, `flex.go:1833`), stretch not subtracting margins
   (`flex.go:1170`, `flex.go:1230`, `flex.go:1407`).
9. Baseline alignment is not implemented: `flex.go:1342`, `flex.go:1808`.
10. `min-height:0` does not override the content floor: `flex.go:1503`,
    `flex.go:1569`.
11. Intrinsic sizing: max-content omits item margins (`flex.go:657`);
    `min-content`/`max-content` keywords are unsupported
    (`style_properties.go:577`, `layout_measure.go:452`).
12. Aspect-ratio height is resolved after item flow: `flex.go:97`,
    `flex.go:981`, `flex.go:717`.
13. Automatic minimum with `overflow: clip`: `flex.go:615`, `flex.go:747`.

Phase 5 coverage gate stays open: the families still missing verified coverage
are vertical writing modes, RTL plus wrap or reverse, column wrap, baseline,
intrinsic keywords, percentage flex-basis, definite-height percentages, and
automatic minimum size with overflow clip.

### Focused pagination regressions retained (2026-09-18)

The consolidated review artifact has been removed. Its useful pagination
observations remain as self-contained focused tests, each with its own inline
HTML and CSS input. Six engine fixes landed, each with a focused regression
test:

1. Inline SVG ignored max-width and max-height. `usedInlineSVGSize`
   (`layout_svg.go`) now clamps through the same ratio-preserving helpers as
   `<img>`, and the SVG measure arm (`layout_measure.go`) feeds the clamped
   width to the flex minimum.
2. The orphan-row seal rewrote a definite-height flex item's fill to the last
   ink baseline plus padding. `tightenLastRowChrome` now skips fills owned by
   flex item boxes (`paint_pagination_seal.go`).
3. A wrapped flex row's backgrounds kept their pre-snap Y after a page break
   while its text snapped forward. The flow shift now moves same-line flex
   siblings with their text (`paint_flow_index.go`), scoped to genuine leaf
   flex items.
4. Deferred chrome entries from sibling sections interleaved at one insertion
   index, overlapping box op ranges; range-based overflow clips then
   deactivated the previous section's fills. Chrome now splices in document
   order (`layout_chrome.go`), keeping sibling ranges disjoint.
5. Column cross-axis auto margins were erased by `buildColumnItems`;
   `alignColumnItem` now distributes free cross space to auto margins and
   stretch is disabled for auto cross margins (`flex.go`).
6. Pagination deskewed container frames from their box geometry: crossing
   snaps half-moved boxes and chrome, and `stretchPaginatedChrome` amplified
   the gap into 9pt rail overshoots. The fix defers snaps for avoid boxes that
   keep together, moves straddling wrapped flex lines wholly to the next page,
   and realigns a box's own frame rules after pagination
   (`paint_pagination_fixpoint.go`, `paint_pagination_chrome.go`).

The historical exploratory skips above are retained as background notes. The
active case ledger records cases 11 through 20 as completed, with focused
layout tests and Chromium rectangle probes as the gates. Cases 21 through 40
have final `completed` or `blocked` statuses backed by focused tests and
reference captures. The golden corpus no longer includes a consolidated
Chrome Flex review fixture.

### Case 5 deferred-chrome leak (2026-09-20)

Case `legacy-definite-main-size` resolved its percentage child correctly at the
box level but painted it twice. `flexMinCrossMainSize` (`flex.go`) measured the
item's children through `layoutCell` with emission enabled, so `prependChrome`
recorded the child's background and outline in `e.deferredChrome`. The
following `e.ops = e.ops[:start]` truncation dropped the emitted ops but not
the deferred entry, and `finalizeChrome` (`layout_chrome.go`) spliced the stale
chrome back at its old index with the measurement geometry (50% of the flex
container, 150pt, at the untransformed container origin). The measurement now
sets `e.noEmit` around `layoutCell`, the same contract as `measureCellHeight`
(`layout_tables.go`) and `measureMulticolChildHeight` (`multicol.go`).
Regression: `TestChromeCaseLegacyDefiniteMainSizePaintsChildOnce` in
`internal/layout/flex_chrome_cases_05_06_test.go` counts the display-list green
fills (want one, 300x125). The fixture comments that claimed an unresolved
percentage were corrected.

### Case 13 inline-block sizing and marker seal (2026-09-20)

Case `legacy-flex-flow-auto-margins` already matched Chromium for every flex
child position, but three engine defects changed the picture:

1. `inlineBlockAvail` counted a specified-width blockish child's horizontal
   margins twice: `measureCellMinMax` includes them through
   `specifiedBlockOuterWidth`, and `nestedBlockHChrome` added them again.
   `nestedBlockHChrome` now skips children consumed by that measure arm, using
   the shared `countedBySpecifiedBlockMeasure` predicate that `measureElement`
   also uses.
2. `flowChildren` added the trailing child's bottom margin only when the
   parent had bottom padding or border. BFC roots (`establishesBFC`: inline-
   block, flow-root, overflow other than visible, floats, cells) do not let
   that margin collapse out, so the condition now includes them.
3. The orphan-row strip zeroed the vertical-rl reverse-flow marker's fill
   because its center sat below the last title ink. `stripOrphanRowOp` now
   requires a row-shaped fill (`stripRowAspect`, width at least 4x height)
   before stripping, so small square markers survive.

Regression: `TestChromeFlexCase13AutoMarginsFixture` asserts the five wrapper
sizes (120x105, 120x105, 120x105, 105x120, 105x120), 15x15 markers, and five
blue marker fills after paint. Stale "Blocked" notes in
`flex_chrome_writing_test.go` were removed. Remaining pre-existing gaps: an
inline-block's own vertical margins are dropped from the line box, the
`translateY(56pt)` is not scaled by smart shrink, and the strip pass still
"removes" horizontal rules by zeroing Width, which the painter clamps to a
1pt hairline.

### Case 15 panel clearance and stale gap claim (2026-09-20)

Case `legacy-multiline-align-self` had no engine defect: all fourteen
align-self children painted with matching geometry, and the fixture's claim
that the dark-blue end-aligned child was a zero-height line was stale. The
reported problem was the missing gap above the case: the absolutely positioned
description panel wraps to three lines under the engine's default font
(Liberation Sans is wider than Chromium's serif), so the panel reached the
flexbox that the fixture clears with `translateY(56pt)`. The stale sentence
was replaced with a shorter truthful one ("All fourteen children paint with
their expected geometry."), which keeps the panel at two lines in both engines
and restores a 12.05pt gap (Chromium 7.25pt). Regression:
`TestChromeCase15DescriptionPanelClearance` asserts the panel bottom stays at
least 4pt above the flexbox's painted top and fails on the pre-fix fixture
with -4.88pt.

The same review reconfirmed the suite-wide transform gap: absolute
`translate` lengths are applied unscaled under smart-shrink zoom (case 15 is
+4.93pt normalized), and the case sweep found case-05's panel overlapping its
case top by about 1pt under the opaque panel. Both are pre-existing and out of
scope here.

### Case 17 stretched cross size is definite for contents (2026-09-20)

Case `wpt-flex-basis-011` rendered its two `flex: 1 0 100%` items
content-sized instead of filling the nested column. CSS Flexbox 9.4 step 5
makes a stretched flex item's used cross size (the line's cross size) definite
for its contents, so the column's content-sized height becomes the percentage
basis and each item resolves to it, overflowing by its borders. The engine
excluded intrinsic flex containers from the forced stretch whenever the row's
cross size was indefinite; `buildRowItems` now stretches them like any other
item and the unused `definiteCross` plumbing was removed. The WPT reftest
contract is preserved: Go renders the source and the `height: 100%` reference
identically. Regression: `TestChromeFlexPercentageBasisIndefiniteColumn` now
asserts a 44pt column with 46pt items (the definite branch stays 100pt/102pt),
and the fixture, manifest, and case-17 ledger rows were updated.

### Case 20 body margin reset and 1:1 page setup (2026-09-20)

Case `wpt-min-size-auto-overflow-clip` renders the automatic minimum size
correctly (the green child stays 150px wide and overflows the 100px
container), but the page did not line up with the Chromium reference: the
description panel sat at (18,18) instead of (12,12) and the smart-shrink zoom
was 0.910 instead of 1. Cause: the fixture was one of five (16-20) that never
added the `html, body { margin: 0; padding: 0 }` reset, so the body's 8px UA
margin shifted the absolutely positioned panel's containing block by 6pt, and
the panel's 560pt width pushed the measured content (586pt) past the A4
content area (538.6pt), triggering the smart shrink. The fixture now carries
the body reset plus `@page { margin: 0 }`: the page area becomes 595.28pt, the
shrink does not fire, and every size matches the Chromium reference 1:1 (panel
12,12; container at y=56; green 112.5x37.5pt). Cases 16-19 still lack both
lines and show the same 6pt panel offset and 0.910 zoom; the same `@page`
rule would make them 1:1 as well.

The engine still resolves an absolutely positioned element's containing block
against a static ancestor's content box (`flowAbsCB` in `layout_flow.go`)
instead of the initial containing block, which is why the body margin reached
the panel. Changing that is a broader fix with its own blast radius; the
fixtures keep the body margin at zero for now.

## Phase 3. Chrome-reference cases

Port the 15 cases that need a browser reference or a larger rewrite.

- Expand JavaScript-generated matrices into a small set of named static cases.
- Replace `offset*` assertions with the geometry that the Go layout result can
  expose.
- Use Puppeteer to capture Chromium reference output for the same HTML.
- Record unsupported features as explicit skips or rewrite notes. Do not turn
  an unsupported feature into a false pass.

The existing `scripts/puppeteer_print.js` path can provide a PDF reference.
The future reference runner should also collect browser box rectangles for
cases where PDF coordinates are too indirect.

## Phase 4. Print fragmentation case

Port the nested flex fragmentation case as a golden fixture. Pin page count,
ordered text, and the location of the repeated content. Add a raster crop
check only if the structural checks cannot detect a regression.

## Phase 5. Interaction coverage gate

Create a small interaction registry from the manifest categories. The registry
will report missing pairs such as direction plus alignment, alignment plus
automatic sizing, wrapping plus gaps, and aspect ratio plus percentage size.

The registry will not request every mathematical pair of CSS properties. It
will cover shared layout decisions and require a named case for each selected
branch.

## Validation commands

```sh
python3 scripts/generate_chrome_flex_cases.py
go test ./test/chrome
go test ./internal/layout -run '^TestFlex'
make golden
```

Run `make test`, `make claim-scan`, and `make lint` after the first converted
case group lands. Do not use `go test ./...` without the repository's test
concurrency limits.

## Non-goals

- Building Chromium locally.
- Porting Blink's C++ test harness.
- Claiming full Chrome or WebKit parity.
- Generating tens of thousands of pairwise fixtures.
