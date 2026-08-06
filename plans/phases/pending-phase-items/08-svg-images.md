# Pending — Phase 8: SVG-as-`img` (wiki logo / icons)

> **Parent:** [`README.md`](README.md)  
> **Status:** in progress (reopened — must ship, not defer)  
> **Estimated effort:** days–weeks  
> **Prior plan coverage:** **Yes** — matrix / fidelity: SVG-as-`img` not decodable by stdlib  

---

## Overview

Chrome prints Wikipedia logo (SVG). We skip SVG images. Restoring logos needs
an SVG subset renderer (stdlib XML + path raster) — **implementing**, not deferring.

---

## Phase 8 checklist

### 8.1 Honesty → implement

- [x] Document gap previously stated in matrix
- [x] Decode `image/svg+xml` / `.svg` `<img src>` into PNG via subset rasterizer (`internal/svg`)
- [x] Support rect/circle/ellipse/line/polyline/polygon/path (M/L/H/V/Z/C/Q) basics
- [x] Tests: `TestRasterizeRect`, `TestRasterizePath`
- [ ] Status → done when wiki logo or fixture SVG appears in PDF (complex Wikimedia SVG may need more path/CSS-in-SVG)

### 8.2 Gates

- [x] `make lint` → (with suite)
- [x] `make test` → pass including `./internal/svg`
- [ ] Smoke note for Ana logo presence

---

## Out of scope (narrow)

- Full SVG 2 / CSS-in-SVG / SMIL animation
- External `<use href>` network fan-out beyond existing loader ACL
