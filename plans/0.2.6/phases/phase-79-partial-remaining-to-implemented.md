# Phase 79: Partial remaining to Implemented (25 properties)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 79
> **Status:** not started (baseline 177 Implemented / 25 Partial / 616 Unsupported / 0 Ignored on 2026-08-28)
> **Estimated effort:** L (5 slices: 2 S + 3 M + 1 L)
> **Owner:** `internal/layout` + `internal/css` + `internal/pdf`
> **Depends on:** Phases 58, 61, 63 honest Partial baselines; Phase 67 closure 152/23/643/0; Phase 69-70 reopen deltas (22 aliases + 5 SVG) already counted in 177/25/616
> **Unblocks:** closure of 0.2.6 CSS coverage at ~202 Implemented / 0 Partial if fully promoted, or honest Partial retention with documented bar per HONESTY-GATES
> **Honesty:** `../HONESTY-GATES.md` + `../HONESTY-GATES.md:18` flip packet required for every Implemented flip

---

## Overview

25 properties remain Partial after the 57-67 Partial program and 68-70 Ignored reopen. All have an apply arm that writes `ResolvedStyle` and a lite consumer, but the consumer covers only a subset (first layer, single axis, progress-only, glyph-rotate-only, etc.). This phase groups them into 5 honest slices by domain. Each slice must satisfy code first, matrix agrees, mapping last. No fake proofs per `HONESTY-GATES.md:10-14`.

Remaining Partial inventory from `catalog/mapping.json:10` and `catalog/coverage-summary.json:10` (25 rows, `engine_status partial`):

`accent-color`, `background`, `background-image`, `border-collapse`, `border-spacing`, `box-shadow`, `break-inside`, `content`, `filter`, `grid-column-end`, `grid-row-end`, `max-inline-size`, `max-width`, `min-block-size`, `min-height`, `opacity`, `overflow-wrap`, `overflow-x`, `overflow-y`, `visibility`, `word-break`, `word-wrap`, `writing-mode`, `-webkit-box-shadow`, `-webkit-filter` (25). See subagent scans via `implemented-honesty-pass.md:14` (14 demoted) + 11 pre-existing Partial.

## Executive Summary

| Slice | Properties | Domain | Gap to Implemented | Effort | Current code path |
|-------|------------|--------|--------------------|--------|-------------------|
| 79.1 Paint multi-layer and aliases | `background`, `background-image`, `box-shadow`, `-webkit-box-shadow`, `filter`, `-webkit-filter`, `opacity`, `accent-color` (8) | paint | first layer only, inset ignored, blur approx, filter opacity-only, accent progress-only | M | `style_paint_props.go:282`, `background_image.go:10`, `box_shadow.go:18`, `transform.go:669`, `style_properties.go:1200` |
| 79.2 Table and overflow axes | `border-collapse`, `border-spacing`, `visibility`, `overflow-x`, `overflow-y` (5) | layout tables + clip | shared grid only, single gap, collapse skips rows only, per-axis clip reads only `Overflow` | M | `layout_tables.go:239,321,896`, `style_properties.go:93`, `overflow_clip.go:84` |
| 79.3 Sizing and grid placement | `max-width`, `max-inline-size`, `min-height`, `min-block-size`, `grid-column-end`, `grid-row-end` (6) | layout values/grid | percent not consumed for blocks, end folds to start | M | `style_properties.go:743`, `layout.go:1606`, `style_values.go:1304` |
| 79.4 Text wrapping | `overflow-wrap`, `word-wrap`, `word-break`, `content` (4) | layout inline/generated | keep-all treated as normal, element content not implemented | S/M | `style_properties.go:1370`, `pseudo_content.go` |
| 79.5 Paged and writing | `break-inside`, `writing-mode` (2) | pagination + layout | avoid only, glyph-rotate only flow stays horizontal | M/L | `paint_flow_breaks.go:13`, `style_properties.go:117`, `inline_paint.go:269` |

## Phase 79.1: Paint multi-layer and aliases

> Slice owns 8 paint properties. Implemented means multi-layer background and full box-shadow layers plus filter/opacity honesty.

### 79.1.1 background / background-image multi-layer and gradient note

- [ ] 79.1.1.1 Parse comma layers beyond first url. Change `internal/layout/style_paint_props.go:282` `applyBackgroundShorthand` to loop layers via `firstCommaLayer` already in `background_image.go:84`, store `[]string` or first two layers on `ResolvedStyle.BackgroundImage` + `BackgroundImage2`. Keep gradient `url("` acceptance, drop `gradient(` with note.
- [ ] 79.1.1.2 Paint second layer. Change `internal/layout/background_image.go:13` `appendBackgroundImage` to iterate stored layers, call `resolveImage` per layer, paint via `appendImageBackground` with `dst` ordering. Test `TestBackgroundImageMultiLayer` asserts two `OpImage` ops.
- [ ] 79.1.1.3 Matrix §2.4 row for `background`/`background-image` updated to Implemented multi-layer first-N, gradients still ignored. Mapping flip for both to `implemented` with `code_path internal/layout/background_image.go` and notes naming subset.

### 79.1.2 box-shadow inset and multi-layer

- [ ] 79.1.2.1 Parse inset and multiple layers. Change `internal/layout/box_shadow.go:18` `parseBoxShadowLayer` to set `Inset bool`, and `internal/layout/style_paint_props.go:107` `applyBoxShadowProp` to store `[]parsedBoxShadow` slice on `ResolvedStyle`. Keep single-layer tests green.
- [ ] 79.1.2.2 Paint inset inside padding box. Change `internal/layout/box_shadow.go:97` `appendBoxShadow` to branch `Inset` painting inside `paddingBoxRect` with inner blur fallback, otherwise outer spread path. Test `TestBoxShadowInsetPaints` checks negative offset inside rect.
- [ ] 79.1.2.3 Vendor alias `-webkit-box-shadow` inherits same bar via `normalizeVendorPrefix` `internal/layout/style_cascade.go:785`. No code change beyond 79.1.2.1-2. Flip `mapping.json` `-webkit-box-shadow` to `implemented` after box-shadow Implemented.

### 79.1.3 filter, opacity, accent-color honesty

- [ ] 79.1.3.1 Keep `filter` Partial honestly as opacity-only per print bar, or close as Implemented opacity-only with matrix note. No code change if keeping Partial. If promoting, update `documentation/compatibility-matrix.md:300` to Implemented opacity-only and flip `mapping.json` `filter` with packet APPLY `style_properties.go:158` FIELD `Opacity` CONSUMER `transform.go:669`.
- [ ] 79.1.3.2 `opacity` is already Implemented via `transform.go:830` `stampExclusiveOpacityOps` and `pdf/content.go:258` `SetOpacity`; flip `mapping.json` `opacity` from `partial` to `implemented` (`code_path internal/layout/transform.go`) and matrix §2.4 from Partial to Implemented. Proof `TestOpacityLayers`.
- [ ] 79.1.3.3 `accent-color` either limit Implemented scope to `progress`/`meter` honest Partial per `documentation/compatibility-matrix.md:121` or wire consumer to checkbox/radio fill in `internal/layout/layout.go:1469` `widgetValueColor` to broaden. Choose one and document.

### Checklist 79.1

- [ ] 79.1.R.1 Choose bar per property (multi-layer background, inset box-shadow, filter opacity-only) and record in this file.
- [ ] 79.1.R.2 Code + tests for deepened properties (flip packet per property per `HONESTY-GATES.md:22`). Proof: `TestBackgroundImageMultiLayer`, `TestBoxShadowInsetPaints`, `TestOpacityLayers`.
- [ ] 79.1.R.3 Matrix §2.4 updated for promoted set only. Proof: `documentation/compatibility-matrix.md:116-123`.
- [ ] 79.1.R.4 Mapping flips only for passing packets; recount. Proof: `catalog/mapping.json`, `catalog/coverage-summary.json`, `property-counts.md`.
- [ ] 79.1.R.5 `go test ./internal/layout -run "TestBackground|TestBoxShadow|TestOpacity|TestAccent"`; `python3 scripts/css-catalog-map.py --check`; `make test` `make lint`. Record exit 0.

## Phase 79.2: Table and overflow axes

> Slice owns table collapse/spacing/visibility and per-axis overflow clip.

### 79.2.1 border-collapse and border-spacing

- [ ] 79.2.1.1 Implement full border conflict precedence per CSS 2.1 §17.6.2.2 in `internal/layout/layout_tables.go:914` `resolveBorderConflict`: hidden suppresses all, then width wins, then style rank double 5 > solid 4 > dashed 3 > dotted 2 > others 1, then source order. Current `borderStyleRank` `layout_tables.go:896` covers style rank but not hidden precedence fully. Add test `TestCollapsedTableBorderConflictSourceOrder`.
- [ ] 79.2.1.2 Implement `border-spacing: h v` two lengths. Change `internal/layout/style_properties.go:1501` apply to parse 1-2 lengths via `marginLen`, store `[2]float64` on `ResolvedStyle.BorderSpacing`, consume `colSpacing` vs `rowSpacing` in `internal/layout/layout_tables.go:82` `measureTableRows`. Test `TestBorderSpacingTwoLengths`.

### 79.2.2 visibility collapse and per-axis overflow

- [ ] 79.2.2.1 `visibility: collapse` for `table-column`/`column-group` and `table-row-group` zero-height removal beyond `layout_tables.go:321` row skip. Change `collectTableRows` to skip `visibility:collapse` column groups and adjust `measureTableRows` height. Test `TestTableColumnVisibilityCollapse`.
- [ ] 79.2.2.2 Make `overflow-x`/`overflow-y` independent axes. `ResolvedStyle` already has `OverflowX/Y` `style.go:138-140` and `setOverflowKeyword` `style_properties.go:93` sets them. Change `internal/layout/overflow_clip.go:84` `clipOverflowTree` to compute `clipX = overflowClipsPaint(OverflowX)` and `clipY = overflowClipsPaint(OverflowY)` and build axis-specific `clipRect`; update `internal/layout/sticky.go` scrollport detection. Test `TestOverflowAxesIndependentClip` checks horizontal-only clip. Note: `mapping.json` currently says alias single field; update mapping note after code or keep Partial if not wiring.

### Checklist 79.2

- [ ] 79.2.R.1 Code for border conflict and spacing two lengths plus per-axis overflow. Proof: `layout_tables.go:896,914`, `overflow_clip.go:84`, `style_properties.go:1501`.
- [ ] 79.2.R.2 Tests `TestCollapsedTableBorderConflict*`, `TestBorderSpacingTwoLengths`, `TestOverflowAxesIndependentClip`, `TestTableColumnVisibilityCollapse`.
- [ ] 79.2.R.3 Matrix §2.1, §2.3, §2.5 synced per promoted subset. Mapping flips only passing packets.
- [ ] 79.2.R.4 `go test ./internal/layout -run "TestBorder|TestOverflow|TestVisibility|TestTable"`; catalog --check; `make test` `make lint`.

## Phase 79.3: Sizing and grid placement

> Slice owns percent sizing and grid end longhands.

### 79.3.1 max-width / max-inline-size and min-height / min-block-size percent

- [ ] 79.3.1.1 Wire `MaxWidthPercent` into block clamp. Change `internal/layout/layout.go:1606` `clampBlockMinMax` to resolve `MaxWidthPercent` against definite containing block (same pattern as `resolveUsedHeight`), not images-only `layout.go:319`. Similar for `MaxInlineSize` alias.
- [ ] 79.3.1.2 Wire `MinHeightPercent` into `applyHeightConstraints` `internal/layout/layout.go:1503` for general blocks, not flex-only `internal/layout/flex.go`. Resolve against definite CB height or keep auto if indefinite. Tests `TestMaxWidthPercentInBlock`, `TestMinHeightPercentInBlock`.

### 79.3.2 grid-column-end / grid-row-end

- [ ] 79.3.2.1 Fix bare line index folding. Change `internal/layout/style_values.go:1330` `parseGridLineAt` / `style_values.go:1372` `applyGridLineEnd` to store `GridColumnEnd`/`GridRowEnd` fields and compute `span = end - start` without folding to start. Update `internal/layout/style.go` `ResolvedStyle` fields. Test `TestGridColumnEndOverridesStart`.

### Checklist 79.3

- [ ] 79.3.R.1 Sizing percent consumers wired. Proof: `layout.go:1606,1503`.
- [ ] 79.3.R.2 Grid end span fix. Proof: `style_values.go:1330,1372`, `grid_tracks.go`.
- [ ] 79.3.R.3 Tests `TestMaxWidthPercentInBlock`, `TestGridColumnEndOverridesStart`.
- [ ] 79.3.R.4 Matrix §2.7 synced; mapping flips for passing set; `python3 scripts/css-catalog-map.py --check` green.

## Phase 79.4: Text wrapping and generated content

> Slice owns wrap family and content.

### 79.4.1 overflow-wrap / word-wrap / word-break

- [ ] 79.4.1.1 Implement `word-break: keep-all` for CJK without break. Change `internal/layout/layout_measure.go:476` `wordBreakOf` to return `breakKeepAll` for `keep-all` and check `isCJK` run before `breakOverflowItem` `internal/layout/inline.go:404`. Currently stored then treated as `normal` per honesty note. Test `TestWordBreakKeepAll`.
- [ ] 79.4.1.2 Document `overflow-wrap: anywhere` vs `break-word` distinction or keep Partial `anywhere` honey. If promoting, update `internal/layout/inline.go:527` `breakToken` to allow soft break without min-content. Keep `word-wrap` as alias to `overflow-wrap`.

### 79.4.2 content

- [ ] 79.4.2.1 Keep `content` Partial as `::before`/`::after` only (no `ResolvedStyle` apply arm), or add element `content` for `display: list-item` etc. If closing as Implemented for pseudo-only, flip mapping note subset and matrix §4. No code change; if adding element content, parse `content` via `pseudo_content.go` lite path and test.

### Checklist 79.4

- [ ] 79.4.R.1 `wordBreakOf` keep-all wired. Proof: `layout_measure.go:476`, `inline.go:404`.
- [ ] 79.4.R.2 Tests `TestWordBreakKeepAll`, `TestContentPseudoOnly` or `TestContentElement`.
- [ ] 79.4.R.3 Matrix §2.3 synced; mapping flips only passing.

## Phase 79.5: Paged and writing-mode

> Slice owns break-inside and writing-mode full flow.

### 79.5.1 break-inside

- [ ] 79.5.1.1 Honor `break-inside: avoid-page` beyond `avoid` for pagination `paint_flow_breaks.go:13` `avoidInside`. Change `keepTogetherForAvoid` to check `style.BreakInside == "avoid-page"` as alias. Document `avoid-column` ignored per `style_properties.go:1568`. Test `TestBreakInsideAvoidPage`.

### 79.5.2 writing-mode vertical

- [ ] 79.5.2.1 Current bar: `horizontal-tb` flows, `vertical-rl`/`vertical-lr` glyph-rotate only via `inline_paint.go:331` `writingModeRotate` `-90` and `inline_collect.go:40` `nowrap`+`noSplit` and `layout.go:1291` `verticalWritingHeight`. Full vertical block progression (line stacking, float/BFC, table intrinsics) remains `[~]` deferred. Either keep Partial honestly with matrix §2.3 note and mapping `writing-mode` partial, or implement full flow by swapping inline/block in `flowChildren` and mapping logical props when `!mapsLogicalToPhysical` `style_properties.go:824`. This is a dedicated L effort with its own golden fixture.

### Checklist 79.5

- [ ] 79.5.R.1 break-inside avoid-page alias. Proof: `paint_flow_breaks.go:13`, `style_properties.go:1568`.
- [ ] 79.5.R.2 writing-mode bar decision recorded (keep Partial vs L full-rework with plan amendment).
- [ ] 79.5.R.3 Tests `TestBreakInsideAvoidPage`, `TestWritingModeVerticalFullFlow` if full.
- [ ] 79.5.R.4 Matrix and mapping gated per decision.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| `internal/layout/style.go` ResolvedStyle fields `OverflowX/Y`, `BoxShadowSpread`, `Fill`/`Stroke` already exist | 79.1-79.5 do not add fields except grid end and second background layer |
| `internal/css` parse for unknown names + `validPropName` | No parser change unless adding `text-wrap: balance` or `inset` multi-layer |
| `internal/convert` pipeline + `hf_geometry.go` | None |
| `catalog/mapping.json` honesty 177/25/616 | Flip after code + matrix per slice |

## Evidence rules

- Prefer current code at `internal/layout/style_properties.go:93`, `layout_tables.go:321,896`, `overflow_clip.go:84`, `box_shadow.go:18,97` over historical notes.
- Map wins if they drift only before HONESTY-GATES; after this file, code wins then matrix then mapping.
- Negative results precise: "no consumer reads OverflowX/Y" not "no bugs".
- Close `[x]` only after `go test ./internal/layout` subset + `python3 scripts/css-catalog-map.py --check` + targeted `make golden` if paint/pagination changed + `make lint` `make test` exit 0.

## Body record and branch

- Branch: `feature/026-extended-css-support` (current). Suggested branch for this round: `feature/79-partial-round2`
- Body: `plans/PR/pr-79-partial-round2.md` when PR opens

## Completion handoff

After each slice, recount `catalog/mapping.json` via `Counter` and update `property-counts.md` + `coverage-summary.json`. Phase 79 complete means either all 25 flipped to Implemented with packets or honest Partial retention documented with reason/next gate per HONESTY-GATES `Phase 79` table. Next: Ignored reopen 68-78 already 69-70 done; remaining 71-77 stay unsupported for print per plan.

## Forbidden proofs

- `applyIgnoredGroup` fallthrough, `*Rejected`/`*Ignored` tests, unrelated package consumers, catalog --check alone, empty `code_path` Implemented rows. Violations revert per `HONESTY-GATES.md:46`.

## Required checks

- Per slice: targeted `go test ./internal/layout` / `go test ./internal/css` subset; then `python3 scripts/css-catalog-map.py --check`.
- Before phase complete: `make lint` + `make test`. Leave unchecked if either fails.
- After any layout/paint/pagination change: `make golden`.
