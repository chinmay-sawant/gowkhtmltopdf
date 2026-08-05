# Tier 2 Pending-3 — Static 2D CSS transforms (print)

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Status:** not started  
> **Estimated effort:** 1–3 weeks  
> **Constraint:** stdlib-only  
> **Spec:** [CSS Transforms Level 1](https://www.w3.org/TR/css-transforms-1/)

---

## Overview

Ship **static computed** 2D transforms for PDF paint. Ignore animation/
transition timelines (take end-state / cascaded value only). Filters beyond
simple opacity are deferred; 3D transforms are non-goals.

Existing `Op.RotateDeg` serves vertical-rl only — generalize carefully.

---

## Executive Summary

| Feature | Stance |
|---------|--------|
| `transform` 2D | `translate*`, `scale*`, `rotate`, `matrix`, `skewX/Y` |
| `transform-origin` | lengths + `%` |
| Layout geometry of siblings | **Unchanged** (paint-time / CB effects only) |
| Abs/fixed CB | Transformed ancestor becomes containing block |
| Stacking | Creates stacking context (like positioned z-index 0) |
| `filter` | Optional later: opacity / grayscale; defer blur/shadow |
| Animations / transitions | Parse-ignore or static end value only |
| 3D / perspective | **Non-goal** |

---

## Phase 1: Design

### 1.1 Evidence

- [ ] Confirm no CSS `transform` in `applyRestProps`
- [ ] Cite `Op.RotateDeg` / vertical path as existing CTM hint
- [ ] Decide PDF emission: content-stream `cm` / Q-q wrappers vs rasterize (prefer vector CTM)

### 1.2 Side effects checklist

- [ ] `transform != none` → stacking context; paint order via existing `sortPaintIndices` or equivalent
- [ ] Establishes containing block for absolute/fixed descendants
- [ ] Does not change sibling flow boxes
- [ ] Overflow: document clip-vs-expand policy when transform pushes outside page

---

## Phase 2: Parse & style

### 2.1 Parsing

- [ ] Parse `transform` function list + `transform-origin`
- [ ] Store matrix (or deferred compose) on `ResolvedStyle` / box
- [ ] `@keyframes` / `transition` / `animation`: ignore for layout; no timeline
- [ ] Path: `style.go`, optional `internal/css` transform parser
- [ ] Proof: parse unit tests for translate/rotate/scale/matrix

---

## Phase 3: Paint & containing blocks

### 3.1 Paint

- [ ] Apply CTM around affected ops (text, fill, image, links source rects)
- [ ] Nested transforms compose (post-multiply order per L1)
- [ ] Path: `paint.go` / draw helpers; keep vertical-rl working
- [ ] Proof: `TestTransformRotateBadge` — siblings unmoved; PDF shows rotation

### 3.2 Containing block

- [ ] Abspos child of transformed ancestor uses transformed padding box as CB
- [ ] Proof: unit layout test for abspos under `transform: scale(2)`

### 3.3 Filters (optional slice)

- [~] `opacity()` via ExtGState — only if cheap
- [~] `grayscale()` / `brightness()` — defer unless needed
- [~] `blur()` / `drop-shadow()` / SVG filters — deferred

---

## Phase 4: Fixtures & closure

### 4.1 Tests

- [ ] Unit: rotate, translate, nested scale+translate
- [ ] Unit: abspos CB under transform
- [ ] Optional golden `fixture-40-transform-badge.html`
- [ ] `@keyframes` present → static cascaded value only

### 4.2 Docs

- [ ] Matrix: Transforms → Partial (static 2D); animations Not implemented
- [ ] Flip phase-17 transforms row: static 2D `[x]`; filters/animations remain `[~]`
- [ ] `make lint` / `make test` recorded

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Position absolute/fixed + z-index lite | CB + stacking |
| Paint display list | CTM wrappers |

---

## Out of scope

- CSS Animations / Transitions timelines
- 3D transforms / `preserve-3d`
- Full Filter Effects / SVG filter graphs
- Matching Chrome compositor layers
