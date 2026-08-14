# Tier 2 Pending-3 — Static 2D CSS transforms (print)

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Status:** done  
> **Estimated effort:** 1–3 weeks  
> **Constraint:** stdlib-only  
> **Spec:** [CSS Transforms Level 1](https://www.w3.org/TR/css-transforms-1/)

---

## Overview

Ship **static computed** 2D transforms for PDF paint. Ignore animation/
transition timelines (take end-state / cascaded value only). Filters beyond
simple opacity are permanent print-engine non-goals; 3D transforms are
permanent non-goals.

Existing `Op.RotateDeg` serves vertical-rl only — kept independent of CSS
transform CTM (CSS wraps the op; glyph RotateDeg still applies in text space).

---

## Executive Summary

| Feature | Stance |
|---------|--------|
| `transform` 2D | `translate*`, `scale*`, `rotate`, `matrix`, `skewX/Y` |
| `transform-origin` | lengths + `%` |
| Layout geometry of siblings | **Unchanged** (paint-time / CB effects only) |
| Abs/fixed CB | Transformed ancestor becomes containing block |
| Stacking | Creates stacking context (like positioned z-index 0) |
| `filter` | `opacity()` via ExtGState; blur/shadow permanent non-goal |
| Animations / transitions | Parse-ignore; static cascaded value only |
| 3D / perspective | **Permanent non-goal** |

---

## Phase 1: Design

### 1.1 Evidence

- [x] Confirm no CSS `transform` in `applyRestProps` (pre-change; now handled)
- [x] Cite `Op.RotateDeg` / vertical path as existing CTM hint (`layout.go` / `paint.go` `drawText`)
- [x] Decide PDF emission: content-stream `cm` / Q-q wrappers (vector CTM via `pdfCTMFromCSS`)

### 1.2 Side effects checklist

- [x] `transform != none` → stacking context; paint order via existing `sortPaintIndices` (`pushZ`)
- [x] Establishes containing block for absolute/fixed descendants (padding box; fixed→absolute under transform)
- [x] Does not change sibling flow boxes
- [x] Overflow: no clip — transformed ink may paint outside the page box; page size unchanged (print policy)

---

## Phase 2: Parse & style

### 2.1 Parsing

- [x] Parse `transform` function list + `transform-origin` (`transform.go` + `applyRestProps`)
- [x] Store matrix on `ResolvedStyle`; bake origin + compose on ops after layout
- [x] `@keyframes` / `transition` / `animation`: skip `@keyframes` blocks in `css.Parse`; ignore animation/transition props — no timeline
- [x] Path: `style.go`, `internal/layout/transform.go`; `@keyframes` skip in `internal/css/css.go`
- [x] Proof: parse unit tests for translate/rotate/scale/matrix (`TestParseTransform*`)

---

## Phase 3: Paint & containing blocks

### 3.1 Paint

- [x] Apply CTM around affected ops (text, fill, image; link annots transform canvas rect)
- [x] Nested transforms compose (post-multiply order per L1)
- [x] Path: `paint.go` / `transform.go`; vertical-rl `RotateDeg` unchanged
- [x] Proof: `TestTransformRotateBadge` — siblings unmoved; PDF shows `cm`

### 3.2 Containing block

- [x] Abspos child of transformed ancestor uses transformed padding box as CB
- [x] Proof: `TestTransformAbsposContainingBlockScale`

### 3.3 Filters & permanence

- [x] `opacity` CSS + `filter: opacity()` via ExtGState (`TestOpacityExtGState`)
- [x] `blur()` / `drop-shadow()` / SVG filters — **permanent out of product scope for print engine** (completed boundary)
- [x] 3D / `perspective` / `matrix3d` — **permanent out of product scope** (parse reject; completed boundary)
- [x] Animations / transitions timelines — **permanent out of product scope**; static cascaded value only (completed boundary)

---

## Phase 4: Fixtures & closure

### 4.1 Tests

- [x] Unit: rotate, translate, nested scale+translate
- [x] Unit: abspos CB under transform
- [x] Golden `fixture-40-transform-badge.html` (+ `fixturePageBounds`)
- [x] `@keyframes` present → static cascaded value only (`TestTransformKeyframesStaticCascaded`)

### 4.2 Docs

- [x] Matrix: Transforms → Partial (static 2D); animations/3D/blur permanent non-goals; opacity Partial
- [x] Flip phase-17 transforms row: static 2D `[x]`
- [x] `make lint` / transform + golden tests recorded below

### 4.3 Closure gates

- [x] `make lint` — pass (`go vet ./...` + gofmt clean) 2026-08-05
- [x] `go test ./internal/layout/ -run 'TestParseTransform|TestTransform|TestOpacity'` — pass
- [x] `go test ./internal/convert/ -run 'TestGoldenCorpusAllFixtures'` — pass (incl. fixture-40)
- [x] `go test ./internal/layout/` full package — pass
- [x] `go test ./internal/css/` — pass (`@keyframes` skip)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Position absolute/fixed + z-index lite | CB + stacking |
| Paint display list | CTM wrappers |

---

## Out of scope (permanent product boundaries)

- CSS Animations / Transitions timelines (static value only)
- 3D transforms / `preserve-3d`
- Full Filter Effects / SVG filter graphs (beyond `opacity()`)
- Matching Chrome compositor layers
