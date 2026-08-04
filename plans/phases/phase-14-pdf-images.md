# Phase 14 - Images in PDF: Robust Path

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** partial (2026-08-04) - PNG/JPEG path + golden fixtures solid; audit/docs polish remain  

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

- [x] Audit `internal/pdf/images.go` + load path for img bytes (JPEG DCT pass-through, PNG Flate/RGB)
- [x] Document supported formats: PNG, JPEG only (matrix §1 / §5)
- [ ] JPEG pass-through vs re-encode path: document quality/DPI knobs state (knobs still best-effort/ignored)
- [ ] PNG with alpha: document how alpha is represented (or flattened)
- [ ] Large image guard: max decode size / memory note (align with load MaxBodySize)

### 14.2 Layout replaced-element robustness

- [x] `<img>` with CSS `width`/`height` only (`layout_test` image sizing)
- [x] `<img>` with intrinsic size only
- [x] `<img>` with both (CSS wins per defined rule - document rule) - code path exists; rule could be clearer in matrix
- [ ] `max-width` / `min-width` on images regression (explicit test still thin)
- [x] Broken/missing file: placeholder or skip **without crash** (`Images` callback error → no paint)
- [x] Zero-byte / corrupt image: skip + warn (`TestAddInvalidImage`)
- [x] Path: `internal/layout/layout.go`, `internal/load`, `internal/pdf/images.go`

### 14.3 Logo & grid fixtures

- [x] fixture-07: logo present in PDF (`/Subtype /Image`) - golden `images: true`
- [x] fixture-20: multi-image grid all embedded or documented skips - golden `images: true`
- [ ] Optional new fixture: remote vs local logo under ACL (local blocked by default)
- [x] Table cell containing logo sizes reasonably (fixture-07 letterhead pattern)

### 14.4 CLI / settings

- [ ] `web.images` false: images not painted (test)
- [ ] image DPI / quality settings: implement or document ignored honestly

### 14.5 Docs

- [x] Matrix §1 `img` row + §5 unsupported formats
- [ ] Fidelity: “solid path for logos/grids” after tests green (fidelity.md pending phase 10)
- [x] Library docs: how to allow local logo path (ACL pair in library-api / integration-security)

### 14.6 Closure gates

- [x] `make lint` → green (2026-08-04 reconcile)
- [x] `make test` → green
- [ ] Parent Phase 14 checked (remaining audit/docs items above)
- [x] Next: **Phase 15** (image mode raster) - **shipped**; or **16** (invoice CSS)

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
