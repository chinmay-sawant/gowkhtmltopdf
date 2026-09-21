# Phase 87.1: Quick wins (aliases, object, overflow, counters, aspect-ratio, grid-auto)

> **Parent:** `../87-canonical-0.2.7-next-72.md`
> **Status:** done
> **Estimated effort:** S–M (11 properties)
> **Owner:** `internal/layout`
> **Depends on:** none
> **Unblocks:** 87.2+
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

Ship the low-risk print wins first: legacy gap aliases, logical overflow aliases,
object-fit/position, counter-set, aspect-ratio, and honesty for grid-auto tracks
that already have apply arms.

## Properties (11)

| Cluster | Names | Effort | Seam |
|---------|-------|--------|------|
| Grid aliases | `grid-gap`, `grid-row-gap`, `grid-column-gap` | S | Alias into existing `gap` / `row-gap` / `column-gap` (`style_properties.go:244-265` today; **extract** before adding if file would grow) |
| Grid auto | `grid-auto-columns`, `grid-auto-rows` | S–M | Apply already `style_properties.go:611-614`; consumers `grid.go:61,195-228`, `grid_tracks.go:20-25`. Prove + deepen indefinite auto-rows if needed, then flip |
| Overflow logical | `overflow-block`, `overflow-inline` | S | New `style_overflow_logical.go` → write `OverflowX`/`OverflowY` via writing-mode; clip already in `overflow_clip.go:109-140` |
| Object | `object-fit`, `object-position` | M | Extend `style_image_adjust_props.go`; paint/size in `layout_images.go:54-99,413-455` (optional extract `object_fit.go`) |
| Counters | `counter-set` | S–M | Apply beside reset/increment in `style_paint_props.go:375-382`; `counter.go` walk order reset → **set** → increment |
| Sizing | `aspect-ratio` | M | New apply group or containment-adjacent file; used-size helpers shared with `resolveUsedWidth` / `resolveContentHeight` (`grid.go:717-765`) and replaced path |

Count: 3+2+2+2+1+1 = 11.

## Architecture

- Do **not** grow `style_properties.go` (2171 allowlisted). For gap aliases: extract `applyGapProps` into `style_gap_props.go` first, then add three alias cases.
- Logical overflow: no new `OverflowBlock` fields required; map to physical axes (same pattern as logical margin).
- Object-fit: deep helper behind `paintReplacedImage` / `usedImageSize`; keep call sites thin.
- Counter-set: one `applySet` on the counter map; do not invent a second counter engine.

## Checklist

### 87.1.1 scope lock

- [x] 87.1.1.1 Confirm the 11 names above against `../next-72-properties.json`. Proof: list in `_proof-87.1.md`.

### 87.1.2 gap aliases

- [x] 87.1.2.1 Extract gap apply into `style_gap_props.go` (or equivalent) so `style_properties.go` does not gain net lines. Register on `styleGroups`. Proof: `style_properties.go` 2171→2120; dispatch stays in `applyFlexGroup`.
- [x] 87.1.2.2 Map `grid-gap`→`gap`, `grid-row-gap`→`row-gap`, `grid-column-gap`→`column-gap`. Tests: `TestGridGapAliasesMatchGap`, existing gap geometry tests stay green.

### 87.1.3 grid-auto honesty

- [x] 87.1.3.1 Package tests proving `grid-auto-columns` / `grid-auto-rows` affect implicit tracks (`TestGridAutoColumns`, `TestGridAutoRows`).
- [x] 87.1.3.2 If indefinite row path ignores auto-rows, deepen `grid_tracks.go` only; do not grow `layout.go`. Proof: `gridAutoFixedPt` + `lockRows` in `resolveGridRows`.
- [x] 87.1.3.3 Flip packet + matrix note for both when consumers proven. Proof: `_proof-87.1.md`.

### 87.1.4 overflow-block / overflow-inline

- [x] 87.1.4.1 Add `style_overflow_logical.go` apply arms; horizontal-tb block→Y inline→X; vertical writing-mode swaps. Proof: `writing-mode` early in `restShorthandProps`.
- [x] 87.1.4.2 Test `TestOverflowBlockInlineMapToAxes` (clip / non-visible axis). Flip mapping after proof.

### 87.1.5 object-fit / object-position

- [x] 87.1.5.1 Fields on `ResolvedStyle`; apply in `applyImageAdjustProps`.
- [x] 87.1.5.2 Consumer: cover/contain/none/scale-down + position offset in image paint. Tests: `TestObjectFitCover`, `TestObjectPositionRightBottom`.
- [x] 87.1.5.3 If `layout_images.go` would approach 2k lines, extract `object_fit.go`. Flip mapping last. Proof: extracted `object_fit.go`.

### 87.1.6 counter-set

- [x] 87.1.6.1 Parse via existing counter-list helper; store set ops.
- [x] 87.1.6.2 Walk: reset → set → increment in `counter.go`. Test `TestCounterSetBeforeIncrement`. Flip mapping.

### 87.1.7 aspect-ratio

- [x] 87.1.7.1 Apply + field; prefer new `style_aspect_ratio_props.go` over growing box group in `style_properties.go`.
- [x] 87.1.7.2 Consumer for definite width→height (and inverse) on blocks/replaced; share one helper. Test `TestAspectRatioOneToOne`. Flip mapping.

### 87.1.R batch gate (package only)

- [x] 87.1.R.1 `go test ./internal/layout -run 'TestGridGap|TestGridAuto|TestOverflowBlock|TestObjectFit|TestObjectPosition|TestCounterSet|TestAspectRatio' -count=1` exit 0.
- [x] 87.1.R.2 `python3 scripts/css-catalog-map.py --check` if apply arms added.
- [x] 87.1.R.3 Mapping + matrix updated only for properties with flip packets. **Do not** run `make test` / `make lint` here.

## Out of scope

Font features, shapes, border-clip drafts, initial-letter, full multicol-2 wrap.
