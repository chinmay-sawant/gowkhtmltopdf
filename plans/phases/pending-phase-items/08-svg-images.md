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
- [ ] Decode `image/svg+xml` / `.svg` `<img src>` into PNG via subset rasterizer
- [ ] Support rect/circle/path/fill basics sufficient for wiki wordmark smoke
- [ ] Tests: tiny SVG fixture paints non-empty image op
- [ ] Status → done when wiki logo or fixture SVG appears in PDF

### 8.2 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Smoke note for Ana logo presence

---

## Out of scope (narrow)

- Full SVG 2 / CSS-in-SVG / SMIL animation
- External `<use href>` network fan-out beyond existing loader ACL
