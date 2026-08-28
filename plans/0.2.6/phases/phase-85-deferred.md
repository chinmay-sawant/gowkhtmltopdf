# Phase 85: Consolidated Deferred & Unsupported Ledger (440 Properties)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 85
> **Status:** [x] completed (Consolidated Ledger & Policy Reference)
> **Total Properties Ledgered:** **440** (All remaining unsupported CSS properties)
> **Engine Philosophy:** Pure-Go static print-to-PDF / image engine (no JavaScript, no DOM runtime, no time loop, no interactive display server)
> **Honesty Rule:** Every unsupported property is explicitly accounted for with concrete technical reasons.

---

## Executive Overview

This document is the canonical ledger and architectural rationale for all **440 unsupported and deferred CSS properties** in `gowkhtmltopdf` v0.2.6.

In accordance with our strict engine constraints (pure Go, standard library font shaping, deterministic PDF generation, zero CGO, zero JavaScript runtime), features that require runtime event loops, time progression, continuous display refresh, user input tracking, hardware audio synthesis, or unstandardized draft specifications are permanently or conditionally deferred.

## Category Breakdown

| Category | Count | Primary Spec | Core Technical Reason |
|---|---:|---|---|
| Animations, Transitions & View Timelines | 68 | CSS / WHATWG | Static print document has no runtime event loop or draft spec support |
| Interactive UI, Forms, Cursors & Pointer Events | 24 | CSS / WHATWG | Static print document has no runtime event loop or draft spec support |
| Interactive Scrolling, Snapping & Scrollbars | 34 | CSS / WHATWG | Static print document has no runtime event loop or draft spec support |
| Speech & Aural Audio Synthesis | 19 | CSS / WHATWG | Static print document has no runtime event loop or draft spec support |
| DOM Anchor Positioning & Motion Offset | 13 | CSS / WHATWG | Static print document has no runtime event loop or draft spec support |
| Complex SVG Masking, Clip Paths & Vector Geometry | 55 | CSS / WHATWG | Static print document has no runtime event loop or draft spec support |
| Advanced Border/Corner/Gap Drafts (CSS Borders 4 & CSS Gaps 1) | 88 | CSS / WHATWG | Static print document has no runtime event loop or draft spec support |
| Vendor-Prefixed Legacy Aliases & Experimental Drafts | 139 | CSS / WHATWG | Static print document has no runtime event loop or draft spec support |
| **Total** | **440** | | |

## Category Details & Rationale

### Animations, Transitions & View Timelines (68 properties)

> **Technical Rationale:** Static print PDF captures a single immutable frame at render time. Keyframe animations, cubic bezier transitions, and view-transition pseudoelements require a continuous frame loop (60Hz timer) and active DOM layout invalidation which does not exist in a static print document.

| Property | Spec | Engine Status |
|---|---|---|
| `-webkit-animation` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-animation-delay` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-animation-direction` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-animation-duration` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-animation-fill-mode` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-animation-iteration-count` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-animation-name` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-animation-play-state` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-animation-timing-function` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-transition` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-transition-delay` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-transition-duration` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-transition-property` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-transition-timing-function` | compat.spec.whatwg.org | `unsupported` |
| `animation` | css-animations-1 | `unsupported` |
| `animation-composition` | css-animations-2 | `unsupported` |
| `animation-delay` | css-animations-2 | `unsupported` |
| `animation-delay-end` | css-animations-2 | `unsupported` |
| `animation-delay-start` | css-animations-2 | `unsupported` |
| `animation-direction` | css-animations-1 | `unsupported` |
| `animation-duration` | css-animations-2 | `unsupported` |
| `animation-fill-mode` | css-animations-1 | `unsupported` |
| `animation-iteration-count` | css-animations-1 | `unsupported` |
| `animation-name` | css-animations-1 | `unsupported` |
| `animation-play-state` | css-animations-1 | `unsupported` |
| `animation-range` | scroll-animations-1 | `unsupported` |
| `animation-range-center` | pointer-animations-1 | `unsupported` |
| `animation-range-end` | scroll-animations-1 | `unsupported` |
| `animation-range-start` | scroll-animations-1 | `unsupported` |
| `animation-timeline` | css-animations-2 | `unsupported` |
| `animation-timing-function` | css-animations-1 | `unsupported` |
| `animation-trigger` | animation-triggers-1 | `unsupported` |
| `caret-animation` | css-ui-4 | `unsupported` |
| `event-trigger` | animation-triggers-1 | `unsupported` |
| `event-trigger-name` | animation-triggers-1 | `unsupported` |
| `event-trigger-source` | animation-triggers-1 | `unsupported` |
| `image-animation` | css-image-animation-1 | `unsupported` |
| `pointer-timeline` | pointer-animations-1 | `unsupported` |
| `pointer-timeline-axis` | pointer-animations-1 | `unsupported` |
| `pointer-timeline-name` | pointer-animations-1 | `unsupported` |
| `scroll-timeline` | scroll-animations-1 | `unsupported` |
| `scroll-timeline-axis` | scroll-animations-1 | `unsupported` |
| `scroll-timeline-name` | scroll-animations-1 | `unsupported` |
| `timeline-scope` | scroll-animations-1 | `unsupported` |
| `timeline-trigger` | animation-triggers-1 | `unsupported` |
| `timeline-trigger-activation-range` | animation-triggers-1 | `unsupported` |
| `timeline-trigger-activation-range-end` | animation-triggers-1 | `unsupported` |
| `timeline-trigger-activation-range-start` | animation-triggers-1 | `unsupported` |
| `timeline-trigger-active-range` | animation-triggers-1 | `unsupported` |
| `timeline-trigger-active-range-end` | animation-triggers-1 | `unsupported` |
| `timeline-trigger-active-range-start` | animation-triggers-1 | `unsupported` |
| `timeline-trigger-name` | animation-triggers-1 | `unsupported` |
| `timeline-trigger-source` | animation-triggers-1 | `unsupported` |
| `transition` | css-transitions-1 | `unsupported` |
| `transition-behavior` | css-transitions-2 | `unsupported` |
| `transition-delay` | css-transitions-1 | `unsupported` |
| `transition-duration` | css-transitions-1 | `unsupported` |
| `transition-property` | css-transitions-1 | `unsupported` |
| `transition-timing-function` | css-transitions-1 | `unsupported` |
| `trigger-scope` | animation-triggers-1 | `unsupported` |
| `view-timeline` | scroll-animations-1 | `unsupported` |
| `view-timeline-axis` | scroll-animations-1 | `unsupported` |
| `view-timeline-inset` | scroll-animations-1 | `unsupported` |
| `view-timeline-name` | scroll-animations-1 | `unsupported` |
| `view-transition-class` | css-view-transitions-2 | `unsupported` |
| `view-transition-group` | css-view-transitions-2 | `unsupported` |
| `view-transition-name` | css-view-transitions-2 | `unsupported` |
| `view-transition-scope` | css-view-transitions-2 | `unsupported` |

### Interactive UI, Forms, Cursors & Pointer Events (24 properties)

> **Technical Rationale:** Interactive mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome are client-side UI concepts irrelevant to printed paper or static raster images.

| Property | Spec | Engine Status |
|---|---|---|
| `-webkit-appearance` | css-ui-4 | `unsupported` |
| `-webkit-transform-style` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-user-select` | css-ui-4 | `unsupported` |
| `appearance` | css-ui-4 | `unsupported` |
| `caret` | css-ui-4 | `unsupported` |
| `caret-color` | css-ui-4 | `unsupported` |
| `caret-shape` | css-ui-4 | `unsupported` |
| `cursor` | css-ui-4 | `unsupported` |
| `field-sizing` | css-forms-1 | `unsupported` |
| `input-security` | css-forms-1 | `unsupported` |
| `interactivity` | css-ui-4 | `unsupported` |
| `interest-delay` | css-ui-4 | `unsupported` |
| `interest-delay-end` | css-ui-4 | `unsupported` |
| `interest-delay-start` | css-ui-4 | `unsupported` |
| `nav-down` | css-ui-4 | `unsupported` |
| `nav-left` | css-ui-4 | `unsupported` |
| `nav-right` | css-ui-4 | `unsupported` |
| `nav-up` | css-ui-4 | `unsupported` |
| `pointer-events` | svg2-draft | `unsupported` |
| `resize` | css-ui-4 | `unsupported` |
| `touch-action` | compat.spec.whatwg.org | `unsupported` |
| `user-select` | css-ui-4 | `unsupported` |
| `window-drag` | css-ui-4 | `unsupported` |
| `word-space-transform` | css-text-4 | `unsupported` |

### Interactive Scrolling, Snapping & Scrollbars (34 properties)

> **Technical Rationale:** A paginated document fragments blocks across physical pages (A4, Letter) rather than providing an interactive scrolling canvas. Scrollbars, scroll snap points, overscroll behavior, and scroll gutter geometry do not apply to printed pages.

| Property | Spec | Engine Status |
|---|---|---|
| `overflow-anchor` | css-scroll-anchoring-1 | `unsupported` |
| `overscroll-behavior` | css-overscroll-1 | `unsupported` |
| `overscroll-behavior-block` | css-overscroll-1 | `unsupported` |
| `overscroll-behavior-inline` | css-overscroll-1 | `unsupported` |
| `overscroll-behavior-x` | css-overscroll-1 | `unsupported` |
| `overscroll-behavior-y` | css-overscroll-1 | `unsupported` |
| `scroll-axis-lock` | css-overflow-5 | `unsupported` |
| `scroll-behavior` | css-overflow-3 | `unsupported` |
| `scroll-initial-target` | css-scroll-snap-2 | `unsupported` |
| `scroll-margin-block` | css-scroll-snap-1 | `unsupported` |
| `scroll-margin-block-end` | css-scroll-snap-1 | `unsupported` |
| `scroll-margin-block-start` | css-scroll-snap-1 | `unsupported` |
| `scroll-margin-inline` | css-scroll-snap-1 | `unsupported` |
| `scroll-margin-inline-end` | css-scroll-snap-1 | `unsupported` |
| `scroll-margin-inline-start` | css-scroll-snap-1 | `unsupported` |
| `scroll-marker-group` | css-overflow-5 | `unsupported` |
| `scroll-padding` | css-scroll-snap-1 | `unsupported` |
| `scroll-padding-block` | css-scroll-snap-1 | `unsupported` |
| `scroll-padding-block-end` | css-scroll-snap-1 | `unsupported` |
| `scroll-padding-block-start` | css-scroll-snap-1 | `unsupported` |
| `scroll-padding-bottom` | css-scroll-snap-1 | `unsupported` |
| `scroll-padding-inline` | css-scroll-snap-1 | `unsupported` |
| `scroll-padding-inline-end` | css-scroll-snap-1 | `unsupported` |
| `scroll-padding-inline-start` | css-scroll-snap-1 | `unsupported` |
| `scroll-padding-left` | css-scroll-snap-1 | `unsupported` |
| `scroll-padding-right` | css-scroll-snap-1 | `unsupported` |
| `scroll-padding-top` | css-scroll-snap-1 | `unsupported` |
| `scroll-snap-align` | css-scroll-snap-1 | `unsupported` |
| `scroll-snap-stop` | css-scroll-snap-1 | `unsupported` |
| `scroll-snap-type` | css-scroll-snap-1 | `unsupported` |
| `scroll-target-group` | css-overflow-5 | `unsupported` |
| `scrollbar-color` | css-scrollbars-1 | `unsupported` |
| `scrollbar-gutter` | css-overflow-3 | `unsupported` |
| `scrollbar-width` | css-scrollbars-1 | `unsupported` |

### Speech & Aural Audio Synthesis (19 properties)

> **Technical Rationale:** Speech synthesis properties (voice volume, rate, pitch, aural cues) target text-to-speech audio engines and screen readers, which are outside the scope of a visual PDF/raster document generator.

| Property | Spec | Engine Status |
|---|---|---|
| `cue` | css-speech-1 | `unsupported` |
| `cue-after` | css-speech-1 | `unsupported` |
| `cue-before` | css-speech-1 | `unsupported` |
| `pause` | css-speech-1 | `unsupported` |
| `pause-after` | css-speech-1 | `unsupported` |
| `pause-before` | css-speech-1 | `unsupported` |
| `rest` | css-speech-1 | `unsupported` |
| `rest-after` | css-speech-1 | `unsupported` |
| `rest-before` | css-speech-1 | `unsupported` |
| `speak` | css-speech-1 | `unsupported` |
| `speak-as` | css-speech-1 | `unsupported` |
| `voice-balance` | css-speech-1 | `unsupported` |
| `voice-duration` | css-speech-1 | `unsupported` |
| `voice-family` | css-speech-1 | `unsupported` |
| `voice-pitch` | css-speech-1 | `unsupported` |
| `voice-range` | css-speech-1 | `unsupported` |
| `voice-rate` | css-speech-1 | `unsupported` |
| `voice-stress` | css-speech-1 | `unsupported` |
| `voice-volume` | css-speech-1 | `unsupported` |

### DOM Anchor Positioning & Motion Offset (13 properties)

> **Technical Rationale:** Anchor positioning and motion paths dynamically tether floating UI widgets (tooltips, popovers, dropdowns) to moving anchor nodes in a live interactive viewport. Print layout resolves via static block/inline/flex/grid formatting contexts.

| Property | Spec | Engine Status |
|---|---|---|
| `anchor-name` | css-anchor-position-1 | `unsupported` |
| `anchor-scope` | css-anchor-position-1 | `unsupported` |
| `offset-anchor` | motion-1 | `unsupported` |
| `offset-distance` | motion-1 | `unsupported` |
| `offset-path` | motion-1 | `unsupported` |
| `offset-position` | motion-1 | `unsupported` |
| `offset-rotate` | motion-1 | `unsupported` |
| `position-anchor` | css-anchor-position-1 | `unsupported` |
| `position-area` | css-anchor-position-1 | `unsupported` |
| `position-try` | css-anchor-position-1 | `unsupported` |
| `position-try-fallbacks` | css-anchor-position-1 | `unsupported` |
| `position-try-order` | css-anchor-position-1 | `unsupported` |
| `position-visibility` | css-anchor-position-1 | `unsupported` |

### Complex SVG Masking, Clip Paths & Vector Geometry (55 properties)

> **Technical Rationale:** Advanced arbitrary vector masking (alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, SVG filter effects) require a full-fledged SVG composition pipeline beyond our print-focused engine.

| Property | Spec | Engine Status |
|---|---|---|
| `-webkit-mask` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-box-image` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-box-image-outset` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-box-image-repeat` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-box-image-slice` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-box-image-source` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-box-image-width` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-clip` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-composite` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-image` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-origin` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-position` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-repeat` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-mask-size` | compat.spec.whatwg.org | `unsupported` |
| `cx` | svg2-draft | `unsupported` |
| `cy` | svg2-draft | `unsupported` |
| `d` | svg2-draft | `unsupported` |
| `flood-color` | filter-effects-1 | `unsupported` |
| `flood-opacity` | filter-effects-1 | `unsupported` |
| `lighting-color` | filter-effects-1 | `unsupported` |
| `marker` | svg2-draft | `unsupported` |
| `marker-end` | svg2-draft | `unsupported` |
| `marker-mid` | svg2-draft | `unsupported` |
| `marker-start` | svg2-draft | `unsupported` |
| `mask` | css-masking-1 | `unsupported` |
| `mask-border` | css-masking-1 | `unsupported` |
| `mask-border-mode` | css-masking-1 | `unsupported` |
| `mask-border-outset` | css-masking-1 | `unsupported` |
| `mask-border-repeat` | css-masking-1 | `unsupported` |
| `mask-border-slice` | css-masking-1 | `unsupported` |
| `mask-border-source` | css-masking-1 | `unsupported` |
| `mask-border-width` | css-masking-1 | `unsupported` |
| `mask-clip` | css-masking-1 | `unsupported` |
| `mask-composite` | css-masking-1 | `unsupported` |
| `mask-image` | css-masking-1 | `unsupported` |
| `mask-mode` | css-masking-1 | `unsupported` |
| `mask-origin` | css-masking-1 | `unsupported` |
| `mask-position` | css-masking-1 | `unsupported` |
| `mask-repeat` | css-masking-1 | `unsupported` |
| `mask-size` | css-masking-1 | `unsupported` |
| `mask-type` | css-masking-1 | `unsupported` |
| `paint-order` | svg2-draft | `unsupported` |
| `path-length` | svg2-draft | `unsupported` |
| `r` | svg2-draft | `unsupported` |
| `rx` | svg2-draft | `unsupported` |
| `ry` | svg2-draft | `unsupported` |
| `shape-image-threshold` | css-shapes-1 | `unsupported` |
| `shape-margin` | css-shapes-1 | `unsupported` |
| `shape-outside` | css-shapes-1 | `unsupported` |
| `stop-color` | svg2-draft | `unsupported` |
| `stop-opacity` | svg2-draft | `unsupported` |
| `text-rendering` | svg2-draft | `unsupported` |
| `vector-effect` | svg2-draft | `unsupported` |
| `x` | svg2-draft | `unsupported` |
| `y` | svg2-draft | `unsupported` |

### Advanced Border/Corner/Gap Drafts (CSS Borders 4 & CSS Gaps 1) (88 properties)

> **Technical Rationale:** CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) that are not yet widely implemented in modern web browsers.

| Property | Spec | Engine Status |
|---|---|---|
| `border-block-clip` | css-borders-4 | `unsupported` |
| `border-block-end-clip` | css-borders-4 | `unsupported` |
| `border-block-start-clip` | css-borders-4 | `unsupported` |
| `border-bottom-clip` | css-borders-4 | `unsupported` |
| `border-clip` | css-borders-4 | `unsupported` |
| `border-inline-clip` | css-borders-4 | `unsupported` |
| `border-inline-end-clip` | css-borders-4 | `unsupported` |
| `border-inline-start-clip` | css-borders-4 | `unsupported` |
| `border-left-clip` | css-borders-4 | `unsupported` |
| `border-limit` | css-borders-4 | `unsupported` |
| `border-right-clip` | css-borders-4 | `unsupported` |
| `border-shape` | css-borders-4 | `unsupported` |
| `border-top-clip` | css-borders-4 | `unsupported` |
| `column-rule-break` | css-gaps-1 | `unsupported` |
| `column-rule-inset` | css-gaps-1 | `unsupported` |
| `column-rule-inset-cap` | css-gaps-1 | `unsupported` |
| `column-rule-inset-cap-end` | css-gaps-1 | `unsupported` |
| `column-rule-inset-cap-start` | css-gaps-1 | `unsupported` |
| `column-rule-inset-end` | css-gaps-1 | `unsupported` |
| `column-rule-inset-junction` | css-gaps-1 | `unsupported` |
| `column-rule-inset-junction-end` | css-gaps-1 | `unsupported` |
| `column-rule-inset-junction-start` | css-gaps-1 | `unsupported` |
| `column-rule-inset-start` | css-gaps-1 | `unsupported` |
| `column-rule-visibility-items` | css-gaps-1 | `unsupported` |
| `corner` | css-borders-4 | `unsupported` |
| `corner-block-end` | css-borders-4 | `unsupported` |
| `corner-block-end-shape` | css-borders-4 | `unsupported` |
| `corner-block-start` | css-borders-4 | `unsupported` |
| `corner-block-start-shape` | css-borders-4 | `unsupported` |
| `corner-bottom` | css-borders-4 | `unsupported` |
| `corner-bottom-left` | css-borders-4 | `unsupported` |
| `corner-bottom-left-shape` | css-borders-4 | `unsupported` |
| `corner-bottom-right` | css-borders-4 | `unsupported` |
| `corner-bottom-right-shape` | css-borders-4 | `unsupported` |
| `corner-bottom-shape` | css-borders-4 | `unsupported` |
| `corner-end-end` | css-borders-4 | `unsupported` |
| `corner-end-end-shape` | css-borders-4 | `unsupported` |
| `corner-end-start` | css-borders-4 | `unsupported` |
| `corner-end-start-shape` | css-borders-4 | `unsupported` |
| `corner-inline-end` | css-borders-4 | `unsupported` |
| `corner-inline-end-shape` | css-borders-4 | `unsupported` |
| `corner-inline-start` | css-borders-4 | `unsupported` |
| `corner-inline-start-shape` | css-borders-4 | `unsupported` |
| `corner-left` | css-borders-4 | `unsupported` |
| `corner-left-shape` | css-borders-4 | `unsupported` |
| `corner-right` | css-borders-4 | `unsupported` |
| `corner-right-shape` | css-borders-4 | `unsupported` |
| `corner-shape` | css-borders-4 | `unsupported` |
| `corner-start-end` | css-borders-4 | `unsupported` |
| `corner-start-end-shape` | css-borders-4 | `unsupported` |
| `corner-start-start` | css-borders-4 | `unsupported` |
| `corner-start-start-shape` | css-borders-4 | `unsupported` |
| `corner-top` | css-borders-4 | `unsupported` |
| `corner-top-left` | css-borders-4 | `unsupported` |
| `corner-top-left-shape` | css-borders-4 | `unsupported` |
| `corner-top-right` | css-borders-4 | `unsupported` |
| `corner-top-right-shape` | css-borders-4 | `unsupported` |
| `corner-top-shape` | css-borders-4 | `unsupported` |
| `grid-column-gap` | css-gaps-1 | `unsupported` |
| `grid-gap` | css-gaps-1 | `unsupported` |
| `grid-row-gap` | css-gaps-1 | `unsupported` |
| `row-rule` | css-gaps-1 | `unsupported` |
| `row-rule-break` | css-gaps-1 | `unsupported` |
| `row-rule-color` | css-gaps-1 | `unsupported` |
| `row-rule-inset` | css-gaps-1 | `unsupported` |
| `row-rule-inset-cap` | css-gaps-1 | `unsupported` |
| `row-rule-inset-cap-end` | css-gaps-1 | `unsupported` |
| `row-rule-inset-cap-start` | css-gaps-1 | `unsupported` |
| `row-rule-inset-end` | css-gaps-1 | `unsupported` |
| `row-rule-inset-junction` | css-gaps-1 | `unsupported` |
| `row-rule-inset-junction-end` | css-gaps-1 | `unsupported` |
| `row-rule-inset-junction-start` | css-gaps-1 | `unsupported` |
| `row-rule-inset-start` | css-gaps-1 | `unsupported` |
| `row-rule-style` | css-gaps-1 | `unsupported` |
| `row-rule-visibility-items` | css-gaps-1 | `unsupported` |
| `row-rule-width` | css-gaps-1 | `unsupported` |
| `rule` | css-gaps-1 | `unsupported` |
| `rule-break` | css-gaps-1 | `unsupported` |
| `rule-color` | css-gaps-1 | `unsupported` |
| `rule-inset` | css-gaps-1 | `unsupported` |
| `rule-inset-cap` | css-gaps-1 | `unsupported` |
| `rule-inset-end` | css-gaps-1 | `unsupported` |
| `rule-inset-junction` | css-gaps-1 | `unsupported` |
| `rule-inset-start` | css-gaps-1 | `unsupported` |
| `rule-overlap` | css-gaps-1 | `unsupported` |
| `rule-style` | css-gaps-1 | `unsupported` |
| `rule-visibility-items` | css-gaps-1 | `unsupported` |
| `rule-width` | css-gaps-1 | `unsupported` |

### Vendor-Prefixed Legacy Aliases & Experimental Drafts (139 properties)

> **Technical Rationale:** Deprecated vendor-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage.

| Property | Spec | Engine Status |
|---|---|---|
| `-webkit-backface-visibility` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-background-clip` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-background-origin` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-background-size` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-box-align` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-box-flex` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-box-ordinal-group` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-box-orient` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-box-pack` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-line-clamp` | css-overflow-4 | `unsupported` |
| `-webkit-perspective` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-perspective-origin` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-text-fill-color` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-text-size-adjust` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-text-stroke` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-text-stroke-color` | compat.spec.whatwg.org | `unsupported` |
| `-webkit-text-stroke-width` | compat.spec.whatwg.org | `unsupported` |
| `all` | css-cascade-5 | `unsupported` |
| `aspect-ratio` | css-sizing-4 | `unsupported` |
| `backdrop-filter` | filter-effects-2 | `unsupported` |
| `background-tbd` | css-backgrounds-4 | `unsupported` |
| `baseline-shift` | css-inline-3 | `unsupported` |
| `baseline-source` | css-inline-3 | `unsupported` |
| `block-ellipsis` | css-overflow-4 | `unsupported` |
| `block-step` | css-rhythm-1 | `unsupported` |
| `block-step-align` | css-rhythm-1 | `unsupported` |
| `block-step-insert` | css-rhythm-1 | `unsupported` |
| `block-step-round` | css-rhythm-1 | `unsupported` |
| `block-step-size` | css-rhythm-1 | `unsupported` |
| `border-boundary` | css-round-display-1 | `unsupported` |
| `box-snap` | css-line-grid-1 | `unsupported` |
| `column-height` | css-multicol-2 | `unsupported` |
| `column-wrap` | css-multicol-2 | `unsupported` |
| `continue` | css-overflow-4 | `unsupported` |
| `copy-into` | css-gcpm-4 | `unsupported` |
| `counter-set` | css-lists-3 | `unsupported` |
| `empty-cells` | css-tables-3 | `unsupported` |
| `fill-break` | fill-stroke-3 | `unsupported` |
| `fill-color` | fill-stroke-3 | `unsupported` |
| `fill-image` | fill-stroke-3 | `unsupported` |
| `fill-origin` | fill-stroke-3 | `unsupported` |
| `fill-position` | fill-stroke-3 | `unsupported` |
| `fill-repeat` | fill-stroke-3 | `unsupported` |
| `fill-size` | fill-stroke-3 | `unsupported` |
| `flex-line-count` | css-flexbox-2 | `unsupported` |
| `float-defer` | css-page-floats-3 | `unsupported` |
| `float-offset` | css-page-floats-3 | `unsupported` |
| `float-reference` | css-page-floats-3 | `unsupported` |
| `flow-from` | css-regions-1 | `unsupported` |
| `flow-into` | css-regions-1 | `unsupported` |
| `flow-tolerance` | css-grid-3 | `unsupported` |
| `font-feature-settings` | css-fonts-4 | `unsupported` |
| `font-kerning` | css-fonts-4 | `unsupported` |
| `font-size-adjust` | css-fonts-5 | `unsupported` |
| `font-stretch` | css-fonts-4 | `unsupported` |
| `font-synthesis` | css-fonts-4 | `unsupported` |
| `font-synthesis-position` | css-fonts-4 | `unsupported` |
| `font-synthesis-small-caps` | css-fonts-4 | `unsupported` |
| `font-synthesis-style` | css-fonts-4 | `unsupported` |
| `font-synthesis-weight` | css-fonts-4 | `unsupported` |
| `font-variant` | css-fonts-4 | `unsupported` |
| `font-variant-alternates` | css-fonts-4 | `unsupported` |
| `font-variant-caps` | css-fonts-4 | `unsupported` |
| `font-variant-east-asian` | css-fonts-4 | `unsupported` |
| `font-variant-emoji` | css-fonts-4 | `unsupported` |
| `font-variant-ligatures` | css-fonts-4 | `unsupported` |
| `font-variant-numeric` | css-fonts-4 | `unsupported` |
| `font-variant-position` | css-fonts-4 | `unsupported` |
| `font-width` | css-fonts-4 | `unsupported` |
| `frame-sizing` | css-sizing-4 | `unsupported` |
| `glyph-orientation-vertical` | css-writing-modes-4 | `unsupported` |
| `grid-auto-columns` | css-grid-2 | `unsupported` |
| `grid-auto-rows` | css-grid-2 | `unsupported` |
| `hanging-punctuation` | css-text-4 | `unsupported` |
| `hyphenate-limit-chars` | css-text-4 | `unsupported` |
| `hyphenate-limit-last` | css-text-4 | `unsupported` |
| `hyphenate-limit-lines` | css-text-4 | `unsupported` |
| `hyphenate-limit-zone` | css-text-4 | `unsupported` |
| `image-rendering` | css-images-3 | `unsupported` |
| `initial-letter` | css-inline-3 | `unsupported` |
| `initial-letter-align` | css-inline-3 | `unsupported` |
| `initial-letter-wrap` | css-inline-3 | `unsupported` |
| `inline-sizing` | css-inline-3 | `unsupported` |
| `interpolate-size` | css-values-5 | `unsupported` |
| `line-fit-edge` | css-inline-3 | `unsupported` |
| `line-grid` | css-line-grid-1 | `unsupported` |
| `line-height-step` | css-rhythm-1 | `unsupported` |
| `line-padding` | css-text-4 | `unsupported` |
| `line-snap` | css-line-grid-1 | `unsupported` |
| `link-parameters` | css-link-params-1 | `unsupported` |
| `marker-side` | css-lists-3 | `unsupported` |
| `math-depth` | mathml-core | `unsupported` |
| `math-shift` | mathml-core | `unsupported` |
| `math-style` | mathml-core | `unsupported` |
| `min-intrinsic-sizing` | css-sizing-4 | `unsupported` |
| `object-fit` | css-images-4 | `unsupported` |
| `object-position` | css-images-3 | `unsupported` |
| `offset` | motion-1 | `unsupported` |
| `overflow-block` | css-overflow-3 | `unsupported` |
| `overflow-inline` | css-overflow-3 | `unsupported` |
| `overlay` | css-position-4 | `unsupported` |
| `reading-flow` | css-display-4 | `unsupported` |
| `reading-order` | css-display-4 | `unsupported` |
| `region-fragment` | css-regions-1 | `unsupported` |
| `shape-inside` | css-shapes-2 | `unsupported` |
| `shape-padding` | css-shapes-2 | `unsupported` |
| `slider-orientation` | css-forms-1 | `unsupported` |
| `spatial-navigation-action` | css-spatial-nav-1 | `unsupported` |
| `spatial-navigation-contain` | css-spatial-nav-1 | `unsupported` |
| `spatial-navigation-function` | css-spatial-nav-1 | `unsupported` |
| `stroke-align` | fill-stroke-3 | `unsupported` |
| `stroke-alignment` | strokes | `unsupported` |
| `stroke-break` | fill-stroke-3 | `unsupported` |
| `stroke-color` | fill-stroke-3 | `unsupported` |
| `stroke-dash-corner` | fill-stroke-3 | `unsupported` |
| `stroke-dash-justify` | fill-stroke-3 | `unsupported` |
| `stroke-dashadjust` | strokes | `unsupported` |
| `stroke-dashcorner` | strokes | `unsupported` |
| `stroke-image` | fill-stroke-3 | `unsupported` |
| `stroke-origin` | fill-stroke-3 | `unsupported` |
| `stroke-position` | fill-stroke-3 | `unsupported` |
| `stroke-repeat` | fill-stroke-3 | `unsupported` |
| `stroke-size` | fill-stroke-3 | `unsupported` |
| `text-autospace` | css-text-4 | `unsupported` |
| `text-box` | css-inline-3 | `unsupported` |
| `text-box-edge` | css-inline-3 | `unsupported` |
| `text-box-trim` | css-inline-3 | `unsupported` |
| `text-fit` | css-text-5 | `unsupported` |
| `text-group-align` | css-text-4 | `unsupported` |
| `text-size-adjust` | css-size-adjust-1 | `unsupported` |
| `text-spacing` | css-text-4 | `unsupported` |
| `text-spacing-trim` | css-text-4 | `unsupported` |
| `will-change` | css-will-change-1 | `unsupported` |
| `wrap-after` | css-text-4 | `unsupported` |
| `wrap-before` | css-text-4 | `unsupported` |
| `wrap-flow` | css-exclusions-1 | `unsupported` |
| `wrap-inside` | css-text-4 | `unsupported` |
| `wrap-through` | css-exclusions-1 | `unsupported` |
| `zoom` | css-viewport | `unsupported` |

