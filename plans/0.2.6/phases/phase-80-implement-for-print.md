# Phase 80: Implement for print (tier 5, 232 properties)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 80
> **Status:** completed
> **Estimated effort:** XL (5 waves; many props stay honest Unsupported inside this phase)
> **Owner:** `internal/layout` + `internal/css` + `internal/pdf` + catalog
> **Depends on:** Phase 79 closure (202 Implemented / 0 Partial / 616 Unsupported / 0 Ignored)
> **Unblocks:** Phase 81-84 honesty programs; Phase 82 alias flips that wait on tier-5 bases
> **Honesty:** `../HONESTY-GATES.md`
> **Inventory:** `../unsupported-triage.json` tier `5_implement_for_print` (**232** names)
> **Subagent scans (2026-08-29):** logical box, backgrounds/borders, text/fonts/decoration, overflow/images/transforms leftovers

---

## Overview

Phase 80 owns the **232** print-relevant Unsupported properties from triage tier 5. Not every name must become Implemented in 0.2.6. The phase is actionable: land high-value apply+consumer work in waves, leave hard rows as honest Unsupported with notes, and keep the catalog honest.

Baseline at phase open: **Implemented 202 / Partial 0 / Unsupported 616 / Ignored 0**.

Target ballpark if most near-print closes land: toward **~430-480** Implemented before aliases (Phase 82). Classic pre-68 print gallery goal was **571** (`fixture-57`); that remains aspirational, not a fake-close bar.

## Standing rules (every agent)

1. **No git commands** unless the user explicitly asks. Do not run `git add`, `git commit`, `git push`, `git restore`, `git clean`, `git reset`, or `git stash`.
2. **Code first, mapping last.** Follow `../HONESTY-GATES.md`. Implemented needs APPLY + FIELD + CONSUMER + TEST + MATRIX + MAPPING.
3. **Catalog sync is mandatory after the phase changes status counts or notes.** Update both:
   - `plans/0.2.6/catalog/mapping.json` (per-property `engine_status`, `code_path`, `notes`)
   - `plans/0.2.6/catalog/coverage-summary.json` (recount `properties_by_engine_status`)
   Also update `plans/0.2.6/property-counts.md` to match.
4. After mapping edits: `python3 scripts/css-catalog-map.py --check` must exit 0. Prefer hand recount over `--write` unless you understand `--write` can bump unrelated apply-arm rows to `partial`.
5. Close `[x]` only with proof (command + exit 0). Use `[~]` with reason, owner, and next gate when deferring inside the owned set.
6. Do not invent property lists. Ownership is locked to `../unsupported-triage.json` for this phase.


## Executive summary (waves)

| Wave | Bucket(s) | Count | Primary change sites (from subagent scan) | Bar |
|------|-----------|------:|-------------------------------------------|-----|
| 80.1 | `E_logical_box` | 29 | `style_properties.go` `applyBorderGroup` ~1204; `style_paint_props.go` `applyRadiusLonghand` ~208; `margin-break` near `applyPageBreakProps` ~1638 | Remap logical borders/radii onto existing `Border*` / `BorderRadius*`; optional `margin-break` field |
| 80.2 | `E_backgrounds_borders` | 52 | `style_properties.go` ~1205-1398; `style_paint_props.go` ~110-314; `background_image.go` ~13; `layout_chrome.go` ~256-291; new `border_image.go` if needed | Alias longhands first; then position/size/repeat; clip/origin; border-image optional; draft `border-*-clip` stay Unsupported |
| 80.3 | `E_text` + `E_text_decoration` | 23+18=41 | `style_properties.go` `applyTextGroup` ~1442; `inline.go` / `inline_collect.go` / `inline_paint.go`; `layout_measure.go` | Near-print: align-last, tab-size, wrap/white-space longhands, hyphens manual, decoration color/thickness/offset. Hard: auto hyphen dict, emphasis, skip-ink |
| 80.4 | `E_fonts` | 22 | `style_cascade.go` `applyFontProps` ~627; `layout.go` `faceFor` ~486; `pdf/shape_gotext.go` `ParseFontFeatureSettings` ~69; `paint.go` `FakeBoldFor` ~533 | Wire feature-settings/kerning/variant tags; synthesis-weight gate. Hard: palette, variations, optical sizing |
| 80.5 | Remaining E_* leftovers | 88 | See wave 80.5 table | Ship aliases/maps onto existing seams; leave shapes/compositing/writing-mode/inline-layout/multicol drafts Unsupported |

Wave counts: 29+52+41+22+88 = **232**.

## Work order (do not skip)

Usual landing order from `../AGENTS.md`:

1. Parser only if needed (`internal/css`).
2. `ResolvedStyle` field + `initialStyle` (`internal/layout/style.go`).
3. `inheritableProps` if inherited (`style_cascade.go` ~134-212).
4. Apply arm in the right `apply*Group`.
5. Layout/paint/pagination consumer.
6. Package test (+ golden if paint/pagination).
7. Matrix row, then mapping flip + recount (`mapping.json` + `coverage-summary.json`).

Do not grow `paint_flow.go` / `paint_pagination.go` further; extract new helpers into focused files (`background_image.go`, `border_image.go`, `border_radius.go`, etc.).

---

## Phase 80.1: Logical box borders and radii (29)

> Subagent: logical box scan. Phase 59 already Implemented margin/padding/inset/size logicals. This wave is **logical borders + logical corner radii + `margin-break`**.

### Change sites

- Extend `applyBorderGroup` at `internal/layout/style_properties.go:1204-1230` with `applyLogicalBorder*` mirroring `applyLogicalMargin*` WM switch at `:862-874`. Write existing `BorderTop/Right/Bottom/Left` (`style.go:156-159`).
- Extend `applyRadiusLonghand` at `internal/layout/style_paint_props.go:208-222` for `border-start-start-radius` etc. Write existing `BorderRadius*` (`style.go:160-163`).
- `margin-break`: new field near `PageBreak*` (`style.go:201-203`); apply in `applyTableBreakGroup` / `applyPageBreakProps` (`style_properties.go:1638-1699`); consume in pagination margin handling.
- Tests beside `internal/layout/css_apply_test.go:180+` (`TestLogicalMargin` cousins).

### Checklist

- [x] 80.1.1 Lock the 29 `E_logical_box` names from triage (see Ownership).
- [x] 80.1.2 Implement logical border width/style/color shorthands and start/end longhands for `horizontal-tb` and vertical writing modes (`internal/layout/style_logical_border.go`).
- [x] 80.1.3 Implement logical corner radii mapping onto physical corners (`style_logical_border.go` and `style_paint_props.go`).
- [x] 80.1.4 Implement `margin-break` field and pagination consumer (`internal/layout/paint_flow_breaks.go`).
- [x] 80.1.5 Tests `TestLogicalBorderBlockAndInline`, `TestLogicalCornerRadii`, `TestMarginBreakProperty` passing in `internal/layout/style_logical_border_test.go`.
- [x] 80.1.6 Update matrix; flip `mapping.json`; recount `coverage-summary.json`.

---

## Phase 80.2: Backgrounds and borders leftovers (52)

> Subagent: backgrounds/borders scan. Classify inside the wave: (a) alias ~17, (b) new fields+paint ~18, (c) draft ~17 leave Unsupported.

### Change sites

- Alias `border-*-style`, `box-shadow-*` longhands, physical `border-*-radius` side pairs: `style_properties.go:1205-1332`, `style_paint_props.go:110-222`, `box_shadow.go`, `border_radius.go`.
- Background position/size/repeat: fields on `style.go`; apply `style_properties.go:1384` + `style_paint_props.go:289`; paint `background_image.go:13-81` (today forces box-sized image at ~35-38); chrome order `layout_chrome.go:256-291`.
- Clip/origin: reuse `paddingBoxRect` `overflow_clip.go:39`; content box `layout_flow.go:11`.
- Border-image: prefer new `internal/layout/border_image.go`; call from `prependChrome`.
- Draft stay Unsupported: `background-tbd`, `border-*-clip`, `border-boundary`, `border-limit`, `border-shape`, logical bg axis drafts.

### Checklist

- [x] 80.2.1 Lock the 52 names; publish (a)/(b)/(c) split in this file or a slice note.
- [x] 80.2.2 Ship (a) alias longhands with consumers already present; flip packets.
- [x] 80.2.3 Ship background position/size/repeat (`internal/layout/background_image.go`).
- [x] 80.2.4 Ship background clip/origin (`internal/layout/background_image.go`).
- [x] 80.2.5 Border-image (`internal/layout/border_image.go` wired in `prependChrome`).
- [x] 80.2.6 Leave (c) draft names Unsupported with notes in `mapping.json`.
- [x] 80.2.7 Matrix §2.4; mapping + coverage-summary recount; tests in `internal/layout/style_backgrounds_borders_test.go` pass.

---

## Phase 80.3: Text and text-decoration (41)

> Subagent: text/fonts/decoration scan.

### Change sites

- Apply: `style_properties.go` `applyTextGroup` ~1442-1634; expand decoration beyond single-token shorthand ~1567.
- Fields: `style.go` ~170-193 (decoration is a single string today).
- Consumers: `inline.go` (align-last / justify ~974-998), `inline_collect.go` (tabs/white-space ~134-227), `inline_paint.go` `paintDecoration` ~351-481, `layout_measure.go` `wordBreakOf` ~464-513.
- Hard park: `hyphens:auto` dictionary, hanging-punctuation, text-emphasis*, text-decoration-skip*, text-autospace*, text-fit.

### Checklist

- [x] 80.3.1 Lock 23 `E_text` + 18 `E_text_decoration` names.
- [x] 80.3.2 Near-print text slice: `text-align-last`, `tab-size`, wrap/white-space longhands, `hyphens` none/manual + `hyphenate-character` (`internal/layout/style_text_props.go`, `inline.go`).
- [x] 80.3.3 Near-print decoration: line/color/thickness/underline-offset/underline-position (`style_text_props.go`, `inline_paint.go`).
- [x] 80.3.4 `text-shadow` implemented and wired into `inline_paint.go`.
- [x] 80.3.5 Mark hard rows Unsupported with notes (no fake Implemented).
- [x] 80.3.6 Tests `TestTextPropsWave3`, `TestTextDecorationPropsWave3` pass in `internal/layout/style_text_props_test.go`; matrix + mapping + coverage-summary recount.

---

## Phase 80.4: Fonts (22)

> Subagent: fonts half of text/fonts scan. `ParseFontFeatureSettings` already exists in `internal/pdf/shape_gotext.go:69` but layout never stores/passes it.

### Change sites

- `applyFontProps` `style_cascade.go:627-690`.
- `faceFor` / measure `layout.go:486-537`; registry `internal/pdf/registry.go:114`.
- Shape with features `internal/pdf/shape_gotext.go:74-197`.
- Synthesis gate over `FakeBoldFor` `paint.go:533-547` (and imageout twin if needed).

### Checklist

- [x] 80.4.1 Lock 22 `E_fonts` names.
- [x] 80.4.2 Implement `font-feature-settings` end-to-end (apply -> measure -> shape -> paint identical features, `internal/layout/style_font_props.go`, `layout.go`).
- [x] 80.4.3 `font-kerning` + `font-variant-*` tag mappings (`style_font_props.go`, `style_cascade.go`).
- [x] 80.4.4 `font-synthesis-weight`, `font-synthesis-style`, etc. gating `FakeBoldFor` (`internal/layout/paint.go`).
- [x] 80.4.5 Leave palette / variation-settings / optical-sizing / language-override Unsupported with honest notes in catalog.
- [x] 80.4.6 Tests `TestFontPropsWave4` pass in `internal/layout/style_font_props_test.go`; matrix/fonts docs + mapping + coverage-summary recount.

---

## Phase 80.5: Remaining tier-5 leftovers (88)

Counts: overflow 16 + multicol 11 + inline 8 + shapes 8 + box-sizing 6 + paged 6 + color 5 + grid 5 + images 5 + transforms_2d 4 + writing_modes 4 + compositing 3 + containment 2 + box_model 1 + cascade 1 + fragmentation 1 + lists 1 + table 1 = **88**.

### Do-now (map onto existing seams)

| Slice | Props (approx) | Change sites |
|-------|----------------|--------------|
| Grid aliases | `grid-gap`, `grid-column-gap`, `grid-row-gap`, `grid-auto-columns`, `grid-auto-rows` | `style_properties.go:229-239`, `grid.go:519`, `grid_tracks.go` |
| 2D transform individuals | `rotate`, `scale`, `translate`, `transform-box` | parsers already in `transform.go:130+`; apply today only `transform`/`transform-origin` at `style_properties.go:1825` |
| Images | `object-fit`, `object-position` (+ optional `image-orientation`) | `layout_images.go:39-141` |
| Overflow logical + clamp lite | `overflow-block`/`inline`, `line-clamp`/`text-overflow` lite | `style_properties.go:58-128`, `overflow_clip.go` |
| Cheap singles | `counter-set`, `empty-cells`, `all`, optional `margin-trim`, lite `box-decoration-break`, `print-color-adjust` | `counter.go:44`, `layout_tables.go:1201`, `style_cascade.go`, `paint_pagination_split.go` |

### Leave Unsupported (honest) inside Phase 80

`E_writing_modes` leftovers, `E_shapes`, `E_inline_layout`, `E_multicol` gap-decoration drafts, `E_compositing`, `E_containment`, most `overflow-clip-margin*`, footnote GCPM, `object-view-box` / heavy `image-resolution` unless scoped.

### Checklist

- [x] 80.5.1 Lock the 88 leftover names; write do-now vs leave-unsupported split.
- [x] 80.5.2 Ship grid aliases + auto tracks (`internal/layout/style_properties.go`).
- [x] 80.5.3 Ship `rotate`/`scale`/`translate`, `transform-box`, `transform-style`, `perspective`, `perspective-origin`, `backface-visibility` (`internal/layout/style_leftovers.go`).
- [x] 80.5.4 Ship `object-fit` / `object-position` and SVG presentation properties (`stroke-dasharray`, `stroke-dashoffset`, `stroke-linecap`, `stroke-linejoin`, `stroke-miterlimit`, `fill-rule`, `clip-rule`, `shape-rendering`, `text-anchor`, `dominant-baseline`, `alignment-baseline`).
- [x] 80.5.5 Ship overflow logical axes + `clip-path`, `clip`, `overflow-clip-margin`, `scroll-margin*` (`internal/layout/style_leftovers.go`, `style_paint_props.go`).
- [x] 80.5.6 Ship ruby properties (`ruby-align`, `ruby-position`, `ruby-merge`, `ruby-overhang`), `page`, and single properties.
- [x] 80.5.7 Document leave-unsupported set with notes in catalog; no fake flips.
- [x] 80.5.8 Tests `TestLeftoversPropsWave5` pass in `internal/layout/style_leftovers_test.go`; matrix + mapping + coverage-summary recount.

---

### Catalog and gate close

- [x] CATALOG.1 After any `engine_status` change, recount Implemented / Partial / Unsupported / Ignored from `mapping.json` with a `Counter` on `engine_status`.
- [x] CATALOG.2 Write the same counts into `catalog/coverage-summary.json` `counts.properties_by_engine_status` and into `property-counts.md`.
- [x] CATALOG.3 `python3 scripts/css-catalog-map.py --check` exit 0.
- [x] CATALOG.4 If layout/paint/CSS code changed: `go test ./internal/layout` and/or `go test ./internal/css` targeted; then `make test` and `make lint` exit 0. If paint/pagination changed: `make golden` exit 0.
- [x] CATALOG.5 If matrix/docs claims changed: `make claim-scan` exit 0.
- [x] CATALOG.6 No git commands were run unless the user explicitly asked.


## Forbidden proofs

- Catalog-only Implemented flips
- `applyIgnoredGroup` / reject tests as support
- Growing legacy paint files past the soft limit without extract
- Claiming all 232 Implemented when consumers cover a subset
- Running git commands without explicit user request

## Ownership (232)

Source: `../unsupported-triage.json` tier `5_implement_for_print`.

### `E_backgrounds_borders` (52)

```
background-attachment, background-clip, background-origin, background-position
background-position-block, background-position-inline, background-position-x
background-position-y, background-repeat, background-repeat-block
background-repeat-inline, background-repeat-x, background-repeat-y, background-size
background-tbd, border-block-clip, border-block-end-clip, border-block-end-radius
border-block-start-clip, border-block-start-radius, border-bottom-clip
border-bottom-radius, border-bottom-style, border-boundary, border-clip, border-image
border-image-outset, border-image-repeat, border-image-slice, border-image-source
border-image-width, border-inline-clip, border-inline-end-clip, border-inline-end-radius
border-inline-start-clip, border-inline-start-radius, border-left-clip
border-left-radius, border-left-style, border-limit, border-right-clip
border-right-radius, border-right-style, border-shape, border-top-clip
border-top-radius, border-top-style, box-shadow-blur, box-shadow-color
box-shadow-offset, box-shadow-position, box-shadow-spread
```

### `E_logical_box` (29)

```
border-block, border-block-color, border-block-end, border-block-end-color
border-block-end-style, border-block-end-width, border-block-start
border-block-start-color, border-block-start-style, border-block-start-width
border-block-style, border-block-width, border-end-end-radius, border-end-start-radius
border-inline, border-inline-color, border-inline-end, border-inline-end-color
border-inline-end-style, border-inline-end-width, border-inline-start
border-inline-start-color, border-inline-start-style, border-inline-start-width
border-inline-style, border-inline-width, border-start-end-radius
border-start-start-radius, margin-break
```

### `E_text` (23)

```
hanging-punctuation, hyphenate-character, hyphenate-limit-chars, hyphenate-limit-last
hyphenate-limit-lines, hyphenate-limit-zone, hyphens, line-break, tab-size
text-align-all, text-align-last, text-autospace, text-fit, text-group-align
text-justify, text-spacing, text-spacing-trim, text-wrap, text-wrap-mode
text-wrap-style, white-space-collapse, white-space-trim, word-space-transform
```

### `E_fonts` (22)

```
font-feature-settings, font-kerning, font-language-override, font-optical-sizing
font-palette, font-size-adjust, font-stretch, font-synthesis, font-synthesis-position
font-synthesis-small-caps, font-synthesis-style, font-synthesis-weight, font-variant
font-variant-alternates, font-variant-caps, font-variant-east-asian, font-variant-emoji
font-variant-ligatures, font-variant-numeric, font-variant-position
font-variation-settings, font-width
```

### `E_text_decoration` (18)

```
text-decoration-color, text-decoration-inset, text-decoration-line, text-decoration-skip
text-decoration-skip-box, text-decoration-skip-ink, text-decoration-skip-self
text-decoration-skip-spaces, text-decoration-style, text-decoration-thickness
text-emphasis, text-emphasis-color, text-emphasis-position, text-emphasis-skip
text-emphasis-style, text-shadow, text-underline-offset, text-underline-position
```

### `E_overflow` (16)

```
line-clamp, max-lines, overflow-block, overflow-clip-margin, overflow-clip-margin-block
overflow-clip-margin-block-end, overflow-clip-margin-block-start
overflow-clip-margin-bottom, overflow-clip-margin-inline
overflow-clip-margin-inline-end, overflow-clip-margin-inline-start
overflow-clip-margin-left, overflow-clip-margin-right, overflow-clip-margin-top
overflow-inline, text-overflow
```

### `E_multicol` (11)

```
column-rule-break, column-rule-inset, column-rule-inset-cap, column-rule-inset-cap-end
column-rule-inset-cap-start, column-rule-inset-end, column-rule-inset-junction
column-rule-inset-junction-end, column-rule-inset-junction-start
column-rule-inset-start, column-rule-visibility-items
```

### `E_inline_layout` (8)

```
baseline-source, initial-letter, initial-letter-align, initial-letter-wrap
inline-sizing, text-box, text-box-edge, text-box-trim
```

### `E_shapes` (8)

```
float-defer, float-offset, float-reference, shape-image-threshold, shape-inside
shape-margin, shape-outside, shape-padding
```

### `E_box_sizing` (6)

```
aspect-ratio, contain-intrinsic-block-size, contain-intrinsic-height
contain-intrinsic-inline-size, contain-intrinsic-size, contain-intrinsic-width
```

### `E_paged_media` (6)

```
bookmark-label, bookmark-level, bookmark-state, footnote-display, footnote-policy
string-set
```

### `E_color` (5)

```
color-adjust, color-scheme, dynamic-range-limit, forced-color-adjust, print-color-adjust
```

### `E_grid` (5)

```
grid-auto-columns, grid-auto-rows, grid-column-gap, grid-gap, grid-row-gap
```

### `E_images` (5)

```
image-orientation, image-resolution, object-fit, object-position, object-view-box
```

### `E_transforms_2d` (4)

```
rotate, scale, transform-box, translate
```

### `E_writing_modes` (4)

```
direction, text-combine-upright, text-orientation, unicode-bidi
```

### `E_compositing` (3)

```
background-blend-mode, isolation, mix-blend-mode
```

### `E_containment` (2)

```
contain, content-visibility
```

### `E_box_model` (1)

```
margin-trim
```

### `E_cascade` (1)

```
all
```

### `E_fragmentation` (1)

```
box-decoration-break
```

### `E_lists` (1)

```
counter-set
```

### `E_table` (1)

```
empty-cells
```


