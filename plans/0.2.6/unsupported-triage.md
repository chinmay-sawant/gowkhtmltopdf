# Unsupported property triage (616)

Source: `catalog/mapping.json` (engine_status=unsupported) plus `ignored-inventory.json` buckets.
Generated for the post-Phase-79 baseline: Implemented 202 / Unsupported 616 / Ignored 0.

## Tier rollup

| Tier | Count | What to do |
|------|------:|------------|
| 1 Skip (print noop) | 155 | Leave unsupported. Animation, speech, pointer UI, scroll snap, 3D, anchor/timelines. |
| 2 Hard defer | 87 | Later program. SVG presentation, mask/clip/filter, CSS regions/exclusions. |
| 3 Alias when base done | 48 | `-webkit-*` etc. Only after unprefixed twin is Implemented. |
| 4 Niche / draft | 94 | Ruby/MathML, corner-shape, gap decorations, rhythmic sizing. Skip for 0.2.6 print push. |
| 5 Implement for print | 232 | The real work list. |
| 6 Manual review | 0 | Should be 0 after this pass. |
| **Total** | **616** | |

## Target math

- Implemented now: **202**
- If all tier 5 lands: **434**
- If tier 5 + vendor aliases: **482**
- Classic pre-68 print goal in the ledger was **571** (fixture-57 gallery).
- A 520-550 landing is realistic if we take most of tier 5 and leave tiers 1/2/4 alone.

## Buckets

| Bucket | Tier | Count |
|--------|------|------:|
| `B_svg_presentation` | 2_hard_defer | 53 |
| `E_backgrounds_borders` | 5_implement_for_print | 52 |
| `C_vendor_prefix_aliases` | 3_alias_when_base_done | 48 |
| `A_time_animation_transition` | 1_skip_print_noop | 45 |
| `A_scroll_snap_overscroll` | 1_skip_print_noop | 41 |
| `D_draft_corner_shape` | 4_niche_or_draft | 34 |
| `D_math_ruby_niche` | 4_niche_or_draft | 33 |
| `E_logical_box` | 5_implement_for_print | 29 |
| `D_draft_gap_decorations` | 4_niche_or_draft | 27 |
| `A_pointer_form_ui` | 1_skip_print_noop | 25 |
| `B_mask_clip_filter_effects` | 2_hard_defer | 25 |
| `E_text` | 5_implement_for_print | 23 |
| `E_fonts` | 5_implement_for_print | 22 |
| `A_anchor_timeline_motion` | 1_skip_print_noop | 21 |
| `A_speech_aural` | 1_skip_print_noop | 19 |
| `E_text_decoration` | 5_implement_for_print | 18 |
| `E_overflow` | 5_implement_for_print | 16 |
| `E_multicol` | 5_implement_for_print | 11 |
| `B_regions_exclusions` | 2_hard_defer | 9 |
| `E_inline_layout` | 5_implement_for_print | 8 |
| `E_shapes` | 5_implement_for_print | 8 |
| `E_box_sizing` | 5_implement_for_print | 6 |
| `E_paged_media` | 5_implement_for_print | 6 |
| `E_color` | 5_implement_for_print | 5 |
| `E_grid` | 5_implement_for_print | 5 |
| `E_images` | 5_implement_for_print | 5 |
| `A_3d_transforms` | 1_skip_print_noop | 4 |
| `E_transforms_2d` | 5_implement_for_print | 4 |
| `E_writing_modes` | 5_implement_for_print | 4 |
| `E_compositing` | 5_implement_for_print | 3 |
| `E_containment` | 5_implement_for_print | 2 |
| `E_box_model` | 5_implement_for_print | 1 |
| `E_cascade` | 5_implement_for_print | 1 |
| `E_fragmentation` | 5_implement_for_print | 1 |
| `E_lists` | 5_implement_for_print | 1 |
| `E_table` | 5_implement_for_print | 1 |

## Recommended next waves (tier 5 only)

1. `E_logical_box` (29) - map to existing physical box fields.
2. `E_backgrounds_borders` (52) - background longhands, border-image, radii.
3. `E_text` (23) + `E_text_decoration` (18).
4. `E_fonts` (22).
5. Smaller closes: overflow (16), images (5), 2D transform individuals (4), writing-mode leftovers (4), grid leftovers (5).

Full property lists live in `unsupported-triage.json`.

## Tier 5 property lists

### `E_backgrounds_borders` (52)

- `background-attachment`
- `background-clip`
- `background-origin`
- `background-position`
- `background-position-block`
- `background-position-inline`
- `background-position-x`
- `background-position-y`
- `background-repeat`
- `background-repeat-block`
- `background-repeat-inline`
- `background-repeat-x`
- `background-repeat-y`
- `background-size`
- `background-tbd`
- `border-block-clip`
- `border-block-end-clip`
- `border-block-end-radius`
- `border-block-start-clip`
- `border-block-start-radius`
- `border-bottom-clip`
- `border-bottom-radius`
- `border-bottom-style`
- `border-boundary`
- `border-clip`
- `border-image`
- `border-image-outset`
- `border-image-repeat`
- `border-image-slice`
- `border-image-source`
- `border-image-width`
- `border-inline-clip`
- `border-inline-end-clip`
- `border-inline-end-radius`
- `border-inline-start-clip`
- `border-inline-start-radius`
- `border-left-clip`
- `border-left-radius`
- `border-left-style`
- `border-limit`
- `border-right-clip`
- `border-right-radius`
- `border-right-style`
- `border-shape`
- `border-top-clip`
- `border-top-radius`
- `border-top-style`
- `box-shadow-blur`
- `box-shadow-color`
- `box-shadow-offset`
- `box-shadow-position`
- `box-shadow-spread`

### `E_logical_box` (29)

- `border-block`
- `border-block-color`
- `border-block-end`
- `border-block-end-color`
- `border-block-end-style`
- `border-block-end-width`
- `border-block-start`
- `border-block-start-color`
- `border-block-start-style`
- `border-block-start-width`
- `border-block-style`
- `border-block-width`
- `border-end-end-radius`
- `border-end-start-radius`
- `border-inline`
- `border-inline-color`
- `border-inline-end`
- `border-inline-end-color`
- `border-inline-end-style`
- `border-inline-end-width`
- `border-inline-start`
- `border-inline-start-color`
- `border-inline-start-style`
- `border-inline-start-width`
- `border-inline-style`
- `border-inline-width`
- `border-start-end-radius`
- `border-start-start-radius`
- `margin-break`

### `E_text` (23)

- `hanging-punctuation`
- `hyphenate-character`
- `hyphenate-limit-chars`
- `hyphenate-limit-last`
- `hyphenate-limit-lines`
- `hyphenate-limit-zone`
- `hyphens`
- `line-break`
- `tab-size`
- `text-align-all`
- `text-align-last`
- `text-autospace`
- `text-fit`
- `text-group-align`
- `text-justify`
- `text-spacing`
- `text-spacing-trim`
- `text-wrap`
- `text-wrap-mode`
- `text-wrap-style`
- `white-space-collapse`
- `white-space-trim`
- `word-space-transform`

### `E_fonts` (22)

- `font-feature-settings`
- `font-kerning`
- `font-language-override`
- `font-optical-sizing`
- `font-palette`
- `font-size-adjust`
- `font-stretch`
- `font-synthesis`
- `font-synthesis-position`
- `font-synthesis-small-caps`
- `font-synthesis-style`
- `font-synthesis-weight`
- `font-variant`
- `font-variant-alternates`
- `font-variant-caps`
- `font-variant-east-asian`
- `font-variant-emoji`
- `font-variant-ligatures`
- `font-variant-numeric`
- `font-variant-position`
- `font-variation-settings`
- `font-width`

### `E_text_decoration` (18)

- `text-decoration-color`
- `text-decoration-inset`
- `text-decoration-line`
- `text-decoration-skip`
- `text-decoration-skip-box`
- `text-decoration-skip-ink`
- `text-decoration-skip-self`
- `text-decoration-skip-spaces`
- `text-decoration-style`
- `text-decoration-thickness`
- `text-emphasis`
- `text-emphasis-color`
- `text-emphasis-position`
- `text-emphasis-skip`
- `text-emphasis-style`
- `text-shadow`
- `text-underline-offset`
- `text-underline-position`

### `E_overflow` (16)

- `line-clamp`
- `max-lines`
- `overflow-block`
- `overflow-clip-margin`
- `overflow-clip-margin-block`
- `overflow-clip-margin-block-end`
- `overflow-clip-margin-block-start`
- `overflow-clip-margin-bottom`
- `overflow-clip-margin-inline`
- `overflow-clip-margin-inline-end`
- `overflow-clip-margin-inline-start`
- `overflow-clip-margin-left`
- `overflow-clip-margin-right`
- `overflow-clip-margin-top`
- `overflow-inline`
- `text-overflow`

### `E_multicol` (11)

- `column-rule-break`
- `column-rule-inset`
- `column-rule-inset-cap`
- `column-rule-inset-cap-end`
- `column-rule-inset-cap-start`
- `column-rule-inset-end`
- `column-rule-inset-junction`
- `column-rule-inset-junction-end`
- `column-rule-inset-junction-start`
- `column-rule-inset-start`
- `column-rule-visibility-items`

### `E_inline_layout` (8)

- `baseline-source`
- `initial-letter`
- `initial-letter-align`
- `initial-letter-wrap`
- `inline-sizing`
- `text-box`
- `text-box-edge`
- `text-box-trim`

### `E_shapes` (8)

- `float-defer`
- `float-offset`
- `float-reference`
- `shape-image-threshold`
- `shape-inside`
- `shape-margin`
- `shape-outside`
- `shape-padding`

### `E_box_sizing` (6)

- `aspect-ratio`
- `contain-intrinsic-block-size`
- `contain-intrinsic-height`
- `contain-intrinsic-inline-size`
- `contain-intrinsic-size`
- `contain-intrinsic-width`

### `E_paged_media` (6)

- `bookmark-label`
- `bookmark-level`
- `bookmark-state`
- `footnote-display`
- `footnote-policy`
- `string-set`

### `E_color` (5)

- `color-adjust`
- `color-scheme`
- `dynamic-range-limit`
- `forced-color-adjust`
- `print-color-adjust`

### `E_grid` (5)

- `grid-auto-columns`
- `grid-auto-rows`
- `grid-column-gap`
- `grid-gap`
- `grid-row-gap`

### `E_images` (5)

- `image-orientation`
- `image-resolution`
- `object-fit`
- `object-position`
- `object-view-box`

### `E_transforms_2d` (4)

- `rotate`
- `scale`
- `transform-box`
- `translate`

### `E_writing_modes` (4)

- `direction`
- `text-combine-upright`
- `text-orientation`
- `unicode-bidi`

### `E_compositing` (3)

- `background-blend-mode`
- `isolation`
- `mix-blend-mode`

### `E_containment` (2)

- `contain`
- `content-visibility`

### `E_box_model` (1)

- `margin-trim`

### `E_cascade` (1)

- `all`

### `E_fragmentation` (1)

- `box-decoration-break`

### `E_lists` (1)

- `counter-set`

### `E_table` (1)

- `empty-cells`

