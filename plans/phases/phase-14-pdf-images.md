# Phase 14 - Images in PDF: Robust Path

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** not started  
> **Estimated effort:** 1–2 weeks  
> **Depends on:** Phase 3 image XObjects; fixtures 07, 20  
> **Unblocks:** logo-heavy invoices; marketing-ish pages with static images  
> **Tier:** 1 #4 · **Constraint:** stdlib `image`, `image/png`, `image/jpeg` only

---

## Overview

MVP already embeds PNG/JPEG. This phase hardens the **path for logos and image grids**: sizing, missing resources, decoding edge cases, and tests - so report templates can rely on images without silent weirdness.

## Executive Summary

| Today | Gap |
|-------|-----|
| PNG/JPEG XObject | GIF not detected; WebP/SVG/AVIF unsupported |
| fixture-07 logo, fixture-20 grid | Need explicit robustness tests |
| Intrinsic dims in layout | CSS width/height interactions need coverage |
| Missing `src` | Behavior must be defined |

---

## Phase 14 checklist

### 14.1 Decode & embed audit

- [ ] Audit `internal/pdf/images.go` + load path for img bytes
- [ ] Document supported formats: PNG, JPEG only (matrix already says; keep honest)
- [ ] JPEG pass-through vs re-encode path: document quality/DPI knobs state
- [ ] PNG with alpha: document how alpha is represented (or flattened)
- [ ] Large image guard: max decode size / memory note (align with load MaxBodySize)

### 14.2 Layout replaced-element robustness

- [ ] `<img>` with CSS `width`/`height` only
- [ ] `<img>` with intrinsic size only
- [ ] `<img>` with both (CSS wins per defined rule - document rule)
- [ ] `max-width` / `min-width` on images regression
- [ ] Broken/missing file: placeholder or skip **without crash**; log warn
- [ ] Zero-byte / corrupt image: skip + warn
- [ ] Path: `internal/layout/layout.go`, `internal/load`, `internal/pdf/images.go`

### 14.3 Logo & grid fixtures

- [ ] fixture-07: logo present in PDF (`/Subtype /Image`) - strengthen assert if weak
- [ ] fixture-20: multi-image grid all embedded or documented skips
- [ ] Optional new fixture: remote vs local logo under ACL (local blocked by default)
- [ ] Table cell containing logo sizes reasonably

### 14.4 CLI / settings

- [ ] `web.images` false: images not painted (test)
- [ ] image DPI / quality settings: implement or document ignored honestly

### 14.5 Docs

- [ ] Matrix §1 `img` row + §5 unsupported formats
- [ ] Fidelity: “solid path for logos/grids” after tests green
- [ ] Library docs: how to allow local logo path

### 14.6 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Parent Phase 14 checked
- [ ] Next: **Phase 15** (image mode raster) or **16** (invoice CSS)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Loader ACL + PDF images | Phase 21 marketing images |

---

## Out of scope

- SVG-as-img, WebP, AVIF (no stdlib decoder)
- `background-image` / CSS gradients (later if ever)
- Image-mode screenshot quality (phase 15)
