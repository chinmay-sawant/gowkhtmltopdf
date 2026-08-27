# Phase 52: Paint, backgrounds, outline, overflow clip

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 52
> **Status:** in progress (background-image, outline, radius longhands, overflow clip landed; box-shadow `[~]`)
> **Estimated effort:** 1-2 weeks
> **Owner:** `internal/layout` paint/chrome, `internal/convert` image fetch
> **Depends on:** Phase 51 not strictly required
> **Unblocks:** richer report chrome without extra HTML `<img>` hacks

---

## Overview

`background` shorthand takes the first color token only (`firstBackgroundColor`, `style_properties.go:965`). `background-image` has no field. Logos and watermarks in CSS never show.

`border-radius` shorthand is Partial. Longhands `border-top-left-radius` etc. parse then ignore.

`overflow` is a sticky scrollport keyword. It does not clip paint (`style.go:128`).

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

- [ ] 52.1.1 Store image URL from `background-image` / `background` shorthand on `ResolvedStyle` (or a small paint-only struct). `url()` only. Gradients ignored this phase. Proof: `go test ./internal/layout -run TestBackgroundImageParse`.
- [ ] 52.1.2 Fetch via the same ACL path as `<img>`. Missing image: skip paint, do not fail convert. Proof: convert test with local PNG.
- [ ] 52.1.3 Paint first layer, `background-repeat: no-repeat` default acceptable, position 0 0, size auto. `background-size` / `repeat` / `position` Partial rows if only a subset lands. Proof: golden fixture with a CSS background logo. Add `fixturePageBounds`.
- [ ] 52.1.4 Matrix §2.4 / §5: `background-image` Partial. Mapping flip.

### 52.2 outline

- [x] 52.2.1 Outline stroke outside the border edge. Proof: `TestOutlineStroke`.
- [x] 52.2.2 Outline does not affect layout size. Proof: same test.
- [x] 52.2.3 Matrix outline row.

### 52.3 radius longhands

- [ ] 52.3.1 `border-top-left-radius` and the other three, including `%`. Proof: extend `rounded_border_test.go`.
- [ ] 52.3.2 Elliptical `/` syntax may stay Partial. Document.

### 52.4 overflow clip

- [x] 52.4.1 Overflow clip for hidden/clip/auto/scroll. Proof: `TestOverflowClip`.
- [x] 52.4.2 Sticky overflow tests green. Proof: `TestStickyOverflow*`.
- [x] 52.4.3 Matrix overflow row updated.

### 52.5 box-shadow optional

- [~] 52.5.1 `box-shadow` not this session. Next gate: a named report fixture that needs a single un-inset shadow.

### 52.6 gates

- [ ] 52.6.1 Mapping flips. `make lint`, `make test`, `make golden`. Record tails.

## Dependencies

Image resolve used by `<img>`. Rounded fill already in `layout_chrome.go`.

## Evidence

Golden with CSS background. Outline unit test. Sticky overflow tests still pass.

## Out of scope

Gradients as a closer for this phase. Multiple background layers. Filter blur. `clip-path`. Mix-blend.

## Handoff

Next is Phase 53.
