# Phase 14 - Images in PDF: Robust Path

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** complete (2026-08-04) - PNG/JPEG path + docs + `web.images` gate  

> **Estimated effort:** 1–2 weeks  
> **Depends on:** Phase 3 image XObjects; fixtures 07, 20  
> **Unblocks:** logo-heavy invoices; marketing-ish pages with static images  
> **Tier:** 1 #4 · **Constraint:** stdlib `image`, `image/png`, `image/jpeg` only

---

## Overview

MVP already embeds PNG/JPEG. This phase hardens the **path for logos and image grids**: sizing, missing resources, decoding edge cases, and tests - so report templates can rely on images without silent weirdness.

## Executive Summary

| Today | Status |
|-------|--------|
| PNG/JPEG XObject | Shipped; GIF/WebP/SVG/AVIF unsupported |
| fixture-07 logo, fixture-20 grid | Golden `images: true` |
| Intrinsic dims in layout | CSS width/height interactions covered |
| Missing `src` | Skip without crash |

---

## Phase 14 checklist

### 14.1 Decode & embed audit

- [x] Audit `internal/pdf/images.go` + load path for img bytes (JPEG DCT pass-through, PNG Flate/RGB)
- [x] Document supported formats: PNG, JPEG only (matrix §1 / §5)
- [x] JPEG pass-through vs re-encode path: document quality/DPI knobs ignored for PDF (matrix §1 img + §7.2)
- [x] PNG with alpha: soft-mask `/SMask` when present (`AddPNGImage`)
- [x] Large image guard: loader `MaxBodySize` default 100 MiB (matrix §1 note)

### 14.2 Layout replaced-element robustness

- [x] `<img>` with CSS `width`/`height` only (`layout_test` image sizing)
- [x] `<img>` with intrinsic size only
- [x] `<img>` with both (CSS wins per defined rule - document rule) - code path exists; rule could be clearer in matrix
- [~] `max-width` / `min-width` on images regression (explicit test still thin) - deferred polish
- [x] Broken/missing file: placeholder or skip **without crash** (`Images` callback error → no paint)
- [x] Zero-byte / corrupt image: skip + warn (`TestAddInvalidImage`)
- [x] Path: `internal/layout/layout.go`, `internal/load`, `internal/pdf/images.go`

### 14.3 Logo & grid fixtures

- [x] fixture-07: logo present in PDF (`/Subtype /Image`) - golden `images: true`
- [x] fixture-20: multi-image grid all embedded or documented skips - golden `images: true`
- [~] Optional new fixture: remote vs local logo under ACL (local blocked by default) - covered by existing ACL docs; skip new fixture
- [x] Table cell containing logo sizes reasonably (fixture-07 letterhead pattern)

### 14.4 CLI / settings

- [x] `web.images` false: images not painted (`TestRunPDFWebImagesFalse` via `Global.Web.Images`)
- [x] image DPI / quality settings: documented ignored for PDF honestly (matrix)

### 14.5 Docs

- [x] Matrix §1 `img` row + §5 unsupported formats
- [x] Fidelity: “Images in PDF” section
- [x] Library docs: how to allow local logo path (ACL pair in library-api / integration-security)

### 14.6 Closure gates

- [x] `make lint` → green
- [x] `make test` → green
- [x] Parent Phase 14 checked
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
