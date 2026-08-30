# Phase 85: Consolidated Deferred CSS Ledger (440 Properties)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 85
> **Status:** [x] completed (Consolidated Ledger & Architectural Rationale)
> **Estimated effort:** M (documentation, classification, catalog audit)

---

## Overview

This phase is the canonical execution ledger and architectural rationale for all **440 unsupported and deferred CSS properties** in `gowkhtmltopdf` v0.2.6.

Static print PDF has no continuous runtime event loop, no display server or 60Hz screen refresh, no mouse/pointer interaction, no aural text-to-speech engine, and no active DOM repositioning. Properties that require these capabilities or depend on experimental, non-standard draft specifications are intentionally deferred (`[~]`) under clear owner boundaries and technical justifications.

## Executive Summary

| Phase / Category | Properties | Primary Specs | Reason |
|---|---:|---|---|
| Phase 85.1: Animations, Transitions & View Timelines | 68 | css-animations, css-transitions, view-transitions | Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. |
| Phase 85.2: Interactive UI, Forms, Cursors & Pointer Events | 24 | css-ui, html form controls | Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. |
| Phase 85.3: Interactive Scrolling, Snapping & Scrollbars | 34 | css-scroll-snap, css-overflow | Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. |
| Phase 85.4: Speech & Aural Audio Synthesis | 19 | css-speech | Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. |
| Phase 85.5: DOM Anchor Positioning & Motion Offset | 13 | css-anchor-position, motion-path | Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. |
| Phase 85.6: Complex SVG Masking, Clip Paths & Vector Geometry | 55 | css-masking, svg2, css-shapes | Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. |
| Phase 85.7: Experimental Border/Corner/Gap Drafts (CSS Borders 4 & CSS Gaps 1) | 88 | css-borders-4, css-gaps-1 | CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. |
| Phase 85.8: Vendor-Prefixed Legacy Aliases & Niche Drafts | 139 | compat.spec.whatwg.org, draft specs | Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. |
| **Total** | **440** | | |

## Phase 85.1: Animations, Transitions & View Timelines (68 properties)

> **Technical Policy:** Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop.

- [~] `-webkit-animation` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-animation-delay` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-animation-direction` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-animation-duration` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-animation-fill-mode` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-animation-iteration-count` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-animation-name` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-animation-play-state` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-animation-timing-function` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-transition` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-transition-delay` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-transition-duration` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-transition-property` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `-webkit-transition-timing-function` (compat.spec.whatwg.org): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation` (css-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-composition` (css-animations-2): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-delay` (css-animations-2): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-delay-end` (css-animations-2): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-delay-start` (css-animations-2): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-direction` (css-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-duration` (css-animations-2): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-fill-mode` (css-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-iteration-count` (css-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-name` (css-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-play-state` (css-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-range` (scroll-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-range-center` (pointer-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-range-end` (scroll-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-range-start` (scroll-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-timeline` (css-animations-2): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-timing-function` (css-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `animation-trigger` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `caret-animation` (css-ui-4): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `event-trigger` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `event-trigger-name` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `event-trigger-source` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `image-animation` (css-image-animation-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `pointer-timeline` (pointer-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `pointer-timeline-axis` (pointer-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `pointer-timeline-name` (pointer-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `scroll-timeline` (scroll-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `scroll-timeline-axis` (scroll-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `scroll-timeline-name` (scroll-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `timeline-scope` (scroll-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `timeline-trigger` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `timeline-trigger-activation-range` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `timeline-trigger-activation-range-end` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `timeline-trigger-activation-range-start` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `timeline-trigger-active-range` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `timeline-trigger-active-range-end` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `timeline-trigger-active-range-start` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `timeline-trigger-name` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `timeline-trigger-source` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `transition` (css-transitions-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `transition-behavior` (css-transitions-2): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `transition-delay` (css-transitions-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `transition-duration` (css-transitions-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `transition-property` (css-transitions-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `transition-timing-function` (css-transitions-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `trigger-scope` (animation-triggers-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `view-timeline` (scroll-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `view-timeline-axis` (scroll-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `view-timeline-inset` (scroll-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `view-timeline-name` (scroll-animations-1): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `view-transition-class` (css-view-transitions-2): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `view-transition-group` (css-view-transitions-2): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `view-transition-name` (css-view-transitions-2): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.
- [~] `view-transition-scope` (css-view-transitions-2): Deferred. Static print PDF captures a single immutable frame; no 60Hz continuous frame timer or DOM layout invalidation loop. Owner: catalog/deferred.

## Phase 85.2: Interactive UI, Forms, Cursors & Pointer Events (24 properties)

> **Technical Policy:** Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images.

- [~] `-webkit-appearance` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `-webkit-transform-style` (compat.spec.whatwg.org): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `-webkit-user-select` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `appearance` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `caret` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `caret-color` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `caret-shape` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `cursor` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `field-sizing` (css-forms-1): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `input-security` (css-forms-1): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `interactivity` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `interest-delay` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `interest-delay-end` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `interest-delay-start` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `nav-down` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `nav-left` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `nav-right` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `nav-up` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `pointer-events` (svg2-draft): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `resize` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `touch-action` (compat.spec.whatwg.org): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `user-select` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `window-drag` (css-ui-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.
- [~] `word-space-transform` (css-text-4): Deferred. Mouse pointers, text editing carets, touch drag actions, user selection highlights, and form appearance chrome do not apply to static printed paper or raster images. Owner: catalog/deferred.

## Phase 85.3: Interactive Scrolling, Snapping & Scrollbars (34 properties)

> **Technical Policy:** Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport.

- [~] `overflow-anchor` (css-scroll-anchoring-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `overscroll-behavior` (css-overscroll-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `overscroll-behavior-block` (css-overscroll-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `overscroll-behavior-inline` (css-overscroll-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `overscroll-behavior-x` (css-overscroll-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `overscroll-behavior-y` (css-overscroll-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-axis-lock` (css-overflow-5): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-behavior` (css-overflow-3): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-initial-target` (css-scroll-snap-2): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-margin-block` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-margin-block-end` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-margin-block-start` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-margin-inline` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-margin-inline-end` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-margin-inline-start` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-marker-group` (css-overflow-5): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-padding` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-padding-block` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-padding-block-end` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-padding-block-start` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-padding-bottom` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-padding-inline` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-padding-inline-end` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-padding-inline-start` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-padding-left` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-padding-right` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-padding-top` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-snap-align` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-snap-stop` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-snap-type` (css-scroll-snap-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scroll-target-group` (css-overflow-5): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scrollbar-color` (css-scrollbars-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scrollbar-gutter` (css-overflow-3): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.
- [~] `scrollbar-width` (css-scrollbars-1): Deferred. Document is paginated across physical pages (A4, Letter) rather than displayed in an interactive scrolling viewport. Owner: catalog/deferred.

## Phase 85.4: Speech & Aural Audio Synthesis (19 properties)

> **Technical Policy:** Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output.

- [~] `cue` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `cue-after` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `cue-before` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `pause` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `pause-after` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `pause-before` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `rest` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `rest-after` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `rest-before` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `speak` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `speak-as` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `voice-balance` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `voice-duration` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `voice-family` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `voice-pitch` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `voice-range` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `voice-rate` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `voice-stress` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.
- [~] `voice-volume` (css-speech-1): Deferred. Speech synthesis and aural properties target text-to-speech audio engines and screen readers, which are outside the scope of visual PDF output. Owner: catalog/deferred.

## Phase 85.5: DOM Anchor Positioning & Motion Offset (13 properties)

> **Technical Policy:** Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout.

- [~] `anchor-name` (css-anchor-position-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `anchor-scope` (css-anchor-position-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `offset-anchor` (motion-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `offset-distance` (motion-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `offset-path` (motion-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `offset-position` (motion-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `offset-rotate` (motion-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `position-anchor` (css-anchor-position-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `position-area` (css-anchor-position-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `position-try` (css-anchor-position-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `position-try-fallbacks` (css-anchor-position-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `position-try-order` (css-anchor-position-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.
- [~] `position-visibility` (css-anchor-position-1): Deferred. Dynamic anchor positioning and motion paths tether floating UI widgets to moving anchor nodes in a live interactive viewport. Print uses static layout. Owner: catalog/deferred.

## Phase 85.6: Complex SVG Masking, Clip Paths & Vector Geometry (55 properties)

> **Technical Policy:** Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline.

- [~] `-webkit-mask` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-box-image` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-box-image-outset` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-box-image-repeat` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-box-image-slice` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-box-image-source` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-box-image-width` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-clip` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-composite` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-image` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-origin` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-position` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-repeat` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `-webkit-mask-size` (compat.spec.whatwg.org): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `cx` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `cy` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `d` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `flood-color` (filter-effects-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `flood-opacity` (filter-effects-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `lighting-color` (filter-effects-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `marker` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `marker-end` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `marker-mid` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `marker-start` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-border` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-border-mode` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-border-outset` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-border-repeat` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-border-slice` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-border-source` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-border-width` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-clip` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-composite` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-image` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-mode` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-origin` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-position` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-repeat` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-size` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `mask-type` (css-masking-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `paint-order` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `path-length` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `r` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `rx` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `ry` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `shape-image-threshold` (css-shapes-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `shape-margin` (css-shapes-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `shape-outside` (css-shapes-1): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `stop-color` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `stop-opacity` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `text-rendering` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `vector-effect` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `x` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.
- [~] `y` (svg2-draft): Deferred. Arbitrary vector alpha masks, SVG clipPath references, CSS shape-outside polygon exclusions, and SVG lighting filters require a full SVG composition pipeline. Owner: catalog/deferred.

## Phase 85.7: Experimental Border/Corner/Gap Drafts (CSS Borders 4 & CSS Gaps 1) (88 properties)

> **Technical Policy:** CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers.

- [~] `border-block-clip` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-block-end-clip` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-block-start-clip` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-bottom-clip` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-clip` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-inline-clip` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-inline-end-clip` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-inline-start-clip` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-left-clip` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-limit` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-right-clip` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `border-top-clip` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `column-rule-break` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `column-rule-inset` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `column-rule-inset-cap` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `column-rule-inset-cap-end` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `column-rule-inset-cap-start` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `column-rule-inset-end` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `column-rule-inset-junction` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `column-rule-inset-junction-end` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `column-rule-inset-junction-start` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `column-rule-inset-start` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `column-rule-visibility-items` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-block-end` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-block-end-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-block-start` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-block-start-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-bottom` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-bottom-left` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-bottom-left-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-bottom-right` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-bottom-right-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-bottom-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-end-end` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-end-end-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-end-start` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-end-start-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-inline-end` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-inline-end-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-inline-start` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-inline-start-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-left` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-left-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-right` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-right-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-start-end` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-start-end-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-start-start` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-start-start-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-top` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-top-left` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-top-left-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-top-right` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-top-right-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `corner-top-shape` (css-borders-4): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `grid-column-gap` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `grid-gap` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `grid-row-gap` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-break` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-color` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-inset` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-inset-cap` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-inset-cap-end` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-inset-cap-start` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-inset-end` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-inset-junction` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-inset-junction-end` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-inset-junction-start` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-inset-start` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-style` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-visibility-items` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `row-rule-width` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule-break` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule-color` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule-inset` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule-inset-cap` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule-inset-end` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule-inset-junction` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule-inset-start` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule-overlap` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule-style` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule-visibility-items` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.
- [~] `rule-width` (css-gaps-1): Deferred. CSS Borders 4 and CSS Gaps 1 contain experimental draft specifications (corner-shape, border-limit, non-standard rule gaps) not standardized across browsers. Owner: catalog/deferred.

## Phase 85.8: Vendor-Prefixed Legacy Aliases & Niche Drafts (139 properties)

> **Technical Policy:** Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage.

- [~] `-webkit-backface-visibility` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-background-clip` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-background-origin` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-background-size` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-box-align` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-box-flex` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-box-ordinal-group` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-box-orient` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-box-pack` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-line-clamp` (css-overflow-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-perspective` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-perspective-origin` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-text-fill-color` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-text-size-adjust` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-text-stroke` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-text-stroke-color` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `-webkit-text-stroke-width` (compat.spec.whatwg.org): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `all` (css-cascade-5): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `aspect-ratio` (css-sizing-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `backdrop-filter` (filter-effects-2): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `background-tbd` (css-backgrounds-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `baseline-shift` (css-inline-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `baseline-source` (css-inline-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `block-ellipsis` (css-overflow-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `block-step` (css-rhythm-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `block-step-align` (css-rhythm-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `block-step-insert` (css-rhythm-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `block-step-round` (css-rhythm-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `block-step-size` (css-rhythm-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `border-boundary` (css-round-display-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `box-snap` (css-line-grid-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `column-height` (css-multicol-2): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `column-wrap` (css-multicol-2): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `continue` (css-overflow-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `copy-into` (css-gcpm-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `counter-set` (css-lists-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `empty-cells` (css-tables-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `fill-break` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `fill-color` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `fill-image` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `fill-origin` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `fill-position` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `fill-repeat` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `fill-size` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `flex-line-count` (css-flexbox-2): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `float-defer` (css-page-floats-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `float-offset` (css-page-floats-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `float-reference` (css-page-floats-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `flow-from` (css-regions-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `flow-into` (css-regions-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `flow-tolerance` (css-grid-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-feature-settings` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-kerning` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-size-adjust` (css-fonts-5): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-stretch` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-synthesis` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-synthesis-position` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-synthesis-small-caps` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-synthesis-style` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-synthesis-weight` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-variant` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-variant-alternates` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-variant-caps` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-variant-east-asian` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-variant-emoji` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-variant-ligatures` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-variant-numeric` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-variant-position` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `font-width` (css-fonts-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `frame-sizing` (css-sizing-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `glyph-orientation-vertical` (css-writing-modes-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `grid-auto-columns` (css-grid-2): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `grid-auto-rows` (css-grid-2): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `hanging-punctuation` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `hyphenate-limit-chars` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `hyphenate-limit-last` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `hyphenate-limit-lines` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `hyphenate-limit-zone` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `image-rendering` (css-images-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `initial-letter` (css-inline-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `initial-letter-align` (css-inline-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `initial-letter-wrap` (css-inline-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `inline-sizing` (css-inline-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `interpolate-size` (css-values-5): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `line-fit-edge` (css-inline-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `line-grid` (css-line-grid-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `line-height-step` (css-rhythm-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `line-padding` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `line-snap` (css-line-grid-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `link-parameters` (css-link-params-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `marker-side` (css-lists-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `math-depth` (mathml-core): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `math-shift` (mathml-core): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `math-style` (mathml-core): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `min-intrinsic-sizing` (css-sizing-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `object-fit` (css-images-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `object-position` (css-images-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `offset` (motion-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `overflow-block` (css-overflow-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `overflow-inline` (css-overflow-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `overlay` (css-position-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `reading-flow` (css-display-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `reading-order` (css-display-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `region-fragment` (css-regions-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `shape-inside` (css-shapes-2): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `shape-padding` (css-shapes-2): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `slider-orientation` (css-forms-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `spatial-navigation-action` (css-spatial-nav-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `spatial-navigation-contain` (css-spatial-nav-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `spatial-navigation-function` (css-spatial-nav-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-align` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-alignment` (strokes): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-break` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-color` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-dash-corner` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-dash-justify` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-dashadjust` (strokes): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-dashcorner` (strokes): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-image` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-origin` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-position` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-repeat` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `stroke-size` (fill-stroke-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `text-autospace` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `text-box` (css-inline-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `text-box-edge` (css-inline-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `text-box-trim` (css-inline-3): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `text-fit` (css-text-5): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `text-group-align` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `text-size-adjust` (css-size-adjust-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `text-spacing` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `text-spacing-trim` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `will-change` (css-will-change-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `wrap-after` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `wrap-before` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `wrap-flow` (css-exclusions-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `wrap-inside` (css-text-4): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `wrap-through` (css-exclusions-1): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.
- [~] `zoom` (css-viewport): Deferred. Deprecated browser-prefixed properties (-webkit-*, -moz-*, -ms-*) and niche draft features with minimal real-world print document usage. Owner: catalog/deferred.

## Dependencies & Verification Gates

- [x] **85.9.1 Catalog Synchronization:** All 440 properties recorded with `engine_status: unsupported` in `plans/0.2.6/catalog/mapping.json`.
- [x] **85.9.2 Summary Reconciliation:** `plans/0.2.6/catalog/coverage-summary.json` reflects 378 Implemented / 0 Partial / 440 Unsupported / 0 Ignored.
- [x] **85.9.3 Tooling Gate:** `python3 scripts/css-catalog-map.py --check` exits 0 with all apply arms mapped.
- [x] **85.9.4 Test Gate:** `make test` exits 0 (all unit and integration packages pass).
- [x] **85.9.5 Golden Corpus Gate:** `make golden` exits 0 (all 61 fixtures pass structural and semantic checks).
- [x] **85.9.6 Claim Scan Gate:** `make claim-scan` exits 0 with zero forbidden claims.
