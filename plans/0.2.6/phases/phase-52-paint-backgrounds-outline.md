# Phase 52: Paint, backgrounds, outline, overflow clip

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 52
> **Status:** complete (background, outline, radius, overflow, and shadow proofs plus full gates verified 2026-08-28)
> **Estimated effort:** 1-2 weeks
> **Owner:** `internal/layout` paint/chrome, `internal/convert` image fetch
> **Depends on:** Phase 51 not strictly required
> **Unblocks:** richer report chrome without extra HTML `<img>` hacks

---

## Overview

`background` shorthand keeps the first supported color and image layer. `background-image` stores a URL and paints it at the box origin through the shared image path.

`border-radius` shorthand and longhands support the documented circular and elliptical subset.

`overflow` also clips the supported paint operations for hidden, clip, auto, and scroll values.

Outline is a common focus/section ring in templates. No field.

Reuse existing image decode and paint ops. Do not invent a second image pipeline.

Watch the 2000-line file cap. If `paint.go` or `layout_chrome.go` is near the limit, extract a new file in the same package.

## Goals

- First-layer `background-image: url(...)` paints
- Outline stroke
- Radius longhands
- Overflow clip lite

## Checklist

### 52.1 background-image

- [x] 52.1.1 Store image URL. Proof: `TestBackgroundImageParse`.
- [x] 52.1.2 Fetch via img path; missing skipped. Proof: `TestBackgroundImageLayoutPaints`.
- [x] 52.1.3 First layer, no-repeat at box origin. Proof: layout unit tests (no extra golden).
- [x] 52.1.4 Matrix Partial. Mapping `--write`.

### 52.2 outline

- [x] 52.2.1 Outline stroke outside the border edge. Proof: `TestOutlineStroke`.
- [x] 52.2.2 Outline does not affect layout size. Proof: same test.
- [x] 52.2.3 Matrix outline row.

### 52.3 radius longhands

- [x] 52.3.1 Radius longhands. Proof: `TestRadiusLonghand`.
- [x] 52.3.2 Elliptical `/` syntax. Shorthand `r / s` and longhands `10pt / 5pt` or `10pt 5pt`. Paint uses unequal rx/ry Bezier arcs. Percent slash still uniform. Proof: `TestRadiusSlash`, `TestRadiusEllipticalLonghand`.

### 52.4 overflow clip

- [x] 52.4.1 Overflow clip for hidden/clip/auto/scroll. Proof: `TestOverflowClip`.
- [x] 52.4.2 Sticky overflow tests green. Proof: `TestStickyOverflow*`.
- [x] 52.4.3 Matrix overflow row updated.

### 52.5 box-shadow optional

- [x] 52.5.1 Lite un-inset `box-shadow` offset fill plus stacked expanding-rect blur. Inset and spread ignored. Layout size unchanged. Proof: `TestBoxShadowParse`, `TestBoxShadowPaints`, `TestBoxShadowBlurPaints`.

### 52.6 gates

- [x] 52.6.1 Mapping `--check`. `make lint`/`test`/`golden` green 2026-08-28.

## Dependencies

Image resolve used by `<img>`. Rounded fill already in `layout_chrome.go`.

## Evidence

Golden with CSS background. Outline unit test. Sticky overflow tests still pass.


## Agent guard

Read `../HONESTY-GATES.md` before any mapping flip. This phase is marked complete only for the **print subset documented in the matrix**. Do not broaden to Chrome-complete or re-fake Implemented rows with empty `code_path`.

## Out of scope

Gradients as a closer for this phase. Multiple background layers. Filter blur. `clip-path`. Mix-blend.

## Handoff

Next is Phase 53.
