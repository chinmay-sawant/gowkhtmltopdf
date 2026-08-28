# Phase 84: Skip print noop (tier 1, 155 properties)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 84
> **Status:** not started (honest Unsupported notes; no engine promotions)
> **Estimated effort:** L (docs/catalog; six buckets)
> **Owner:** catalog policy (+ matrix/deferred)
> **Depends on:** Phases 72-77 bucket honesty (this phase is the tier-1 rollup for remaining triage names)
> **Unblocks:** clean closure language for print non-goals
> **Honesty:** `../HONESTY-GATES.md`
> **Inventory:** `../unsupported-triage.json` tier `1_skip_print_noop` (**155** names)
> **Subagent scan (2026-08-29):** skip/niche pattern from phases 72-77

---

## Overview

Static print PDF has no time loop, no interactive scroll viewport, no pointer/caret chrome, no aural synthesis, no 3D scene graph, and no anchor/view timelines. All **155** tier-1 names stay `engine_status: unsupported` with honest notes. This phase documents that policy; it does not implement these properties.

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


## Ownership buckets (155)

| Bucket | Count | Why skip |
|--------|------:|----------|
| `A_time_animation_transition` | 45 | No animation/transition/view-transition timeline |
| `A_scroll_snap_overscroll` | 41 | No interactive scroll viewport |
| `A_pointer_form_ui` | 25 | No pointer/caret/form chrome |
| `A_anchor_timeline_motion` | 21 | Anchor / offset / view timelines |
| `A_speech_aural` | 19 | No aural speech synthesis |
| `A_3d_transforms` | 4 | 2D transform only (`perspective*`, `transform-style`, `backface-visibility`) |
| **Total** | **155** | |

## Work order

1. Lock 155 names from triage.
2. Replace mass-revert stub notes with per-bucket prose in `mapping.json`.
3. Keep `unsupported` + empty `code_path`.
4. Align matrix § transforms / deferred: 2D transform stays Implemented only where code already has consumers; 3D stays Unsupported.
5. `--check`; recount only if counts change.

## Checklist

- [ ] 84.1.1 Ownership locked to 155 names across the six `A_*` buckets.
- [ ] 84.2.1 Policy: print-noop / non-goal; do not implement in 0.2.6.
- [ ] 84.2.2 Mapping notes updated per bucket for all 155; status remains unsupported.
- [ ] 84.2.3 Scripted verify count 155 unsupported.
- [ ] 84.2.4 Matrix/deferred language matches (animation, scroll, pointer, speech, 3D, anchor timelines).
- [ ] 84.2.5 No new apply arms pretending to support these names.

### Catalog and gate close

- [ ] CATALOG.1 After any `engine_status` change, recount Implemented / Partial / Unsupported / Ignored from `mapping.json` with a `Counter` on `engine_status`.
- [ ] CATALOG.2 Write the same counts into `catalog/coverage-summary.json` `counts.properties_by_engine_status` and into `property-counts.md`.
- [ ] CATALOG.3 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] CATALOG.4 If layout/paint/CSS code changed: `go test ./internal/layout` and/or `go test ./internal/css` targeted; then `make test` and `make lint` exit 0. If paint/pagination changed: `make golden` exit 0.
- [ ] CATALOG.5 If matrix/docs claims changed: `make claim-scan` exit 0.
- [ ] CATALOG.6 No git commands were run unless the user explicitly asked.


## Forbidden proofs

- `TestParseTransformNoneAnd3DRejected` as 3D Implemented
- Animation-ignored tests as `animation-*` Implemented
- PDF tagging as `speak` / `voice-*`
- Sticky/overflow tests as `scroll-*` Implemented
- Layout variable named `cursor` as CSS `cursor`
- Flipping any of the 155 to Implemented
- Git commands without explicit user request

## Note templates

- animation/transition: `Static print PDF: no animation/transition timeline. Left unsupported.`
- scroll: `Print PDF has no interactive scroll viewport. Left unsupported.`
- pointer/form UI: `Print PDF has no pointer/caret/form chrome. Left unsupported.`
- anchor/timelines: `Anchor positioning / view timelines not supported for print. Left unsupported.`
- speech/aural: `Visual print PDF has no aural speech synthesis. Left unsupported.`
- 3D: `2D transform only; perspective/transform-style/backface-visibility unsupported.`

## Ownership (155)

### `A_time_animation_transition` (45)

```
animation, animation-composition, animation-delay, animation-delay-end
animation-delay-start, animation-direction, animation-duration, animation-fill-mode
animation-iteration-count, animation-name, animation-play-state, animation-range
animation-range-center, animation-range-end, animation-range-start, animation-timeline
animation-timing-function, animation-trigger, event-trigger, event-trigger-name
event-trigger-source, image-animation, pointer-timeline, pointer-timeline-axis
pointer-timeline-name, timeline-trigger, timeline-trigger-activation-range
timeline-trigger-activation-range-end, timeline-trigger-activation-range-start
timeline-trigger-active-range, timeline-trigger-active-range-end
timeline-trigger-active-range-start, timeline-trigger-name, timeline-trigger-source
transition, transition-behavior, transition-delay, transition-duration
transition-property, transition-timing-function, trigger-scope, view-transition-class
view-transition-group, view-transition-name, view-transition-scope
```

### `A_scroll_snap_overscroll` (41)

```
overscroll-behavior, overscroll-behavior-block, overscroll-behavior-inline
overscroll-behavior-x, overscroll-behavior-y, scroll-axis-lock, scroll-behavior
scroll-initial-target, scroll-margin, scroll-margin-block, scroll-margin-block-end
scroll-margin-block-start, scroll-margin-bottom, scroll-margin-inline
scroll-margin-inline-end, scroll-margin-inline-start, scroll-margin-left
scroll-margin-right, scroll-margin-top, scroll-marker-group, scroll-padding
scroll-padding-block, scroll-padding-block-end, scroll-padding-block-start
scroll-padding-bottom, scroll-padding-inline, scroll-padding-inline-end
scroll-padding-inline-start, scroll-padding-left, scroll-padding-right
scroll-padding-top, scroll-snap-align, scroll-snap-stop, scroll-snap-type
scroll-target-group, scroll-timeline, scroll-timeline-axis, scroll-timeline-name
scrollbar-color, scrollbar-gutter, scrollbar-width
```

### `A_pointer_form_ui` (25)

```
appearance, caret, caret-animation, caret-color, caret-shape, cursor, field-sizing
input-security, interactivity, interest-delay, interest-delay-end, interest-delay-start
nav-down, nav-left, nav-right, nav-up, pointer-events, resize, slider-orientation
spatial-navigation-action, spatial-navigation-contain, spatial-navigation-function
touch-action, user-select, window-drag
```

### `A_anchor_timeline_motion` (21)

```
anchor-name, anchor-scope, offset, offset-anchor, offset-distance, offset-path
offset-position, offset-rotate, overflow-anchor, position-anchor, position-area
position-try, position-try-fallbacks, position-try-order, position-visibility
timeline-scope, view-timeline, view-timeline-axis, view-timeline-inset
view-timeline-name, will-change
```

### `A_speech_aural` (19)

```
cue, cue-after, cue-before, pause, pause-after, pause-before, rest, rest-after
rest-before, speak, speak-as, voice-balance, voice-duration, voice-family, voice-pitch
voice-range, voice-rate, voice-stress, voice-volume
```

### `A_3d_transforms` (4)

```
backface-visibility, perspective, perspective-origin, transform-style
```


