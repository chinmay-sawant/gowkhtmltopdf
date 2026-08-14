# Phase 12 - Typography: Real Bold (and Italic) Faces

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** complete (2026-08-04) - Liberation Sans R/B/I/BI + FaceSet.Resolve
> **Estimated effort:** 2–4 weeks  
> **Depends on:** Phase 3 fonts/PDF writer; Phase 10 matrix language  
> **Unblocks:** Phase 13 spacing; Phase 15 image-mode metrics; Phase 19 multi-font  
> **Tier:** 1 #1 · **Constraint:** embed OFL fonts in-repo; no FreeType/cgo

---

## Overview

Replace **fake bold** (PDF text render mode 2 stroke) and upright “italic” with **real TrueType faces**. Introduce a small font registry so CSS `font-weight` / `font-style` select faces. Still Latin-first; CJK is Phase 19.

## Executive Summary

| Evidence | Location |
|----------|----------|
| Only Regular TTF bundled | `internal/pdf/assets/LiberationSans-Regular.ttf` |
| Fake bold paint | `internal/layout/paint.go` `TextRenderMode(2)` |
| HF also fake-bold | `internal/convert/hf.go` |
| Italic parsed, unused | matrix §2.3; `FontItalic` |
| Layout always one face | `internal/layout/inline.go` metrics |

---

## Phase 12 checklist

### 12.1 Assets (stdlib embed only)

- [x] Obtain Liberation Sans **Bold** (and ideally **Italic**, **BoldItalic**) under SIL OFL / same license as Regular
- [x] Place under `internal/pdf/assets/` (or `internal/fonts/assets/`)
- [x] Update `assets.go` `//go:embed` patterns
- [x] LICENSE/NOTICE note for additional faces if required
- [x] Verify files load via `pdf.LoadFont` (or current loader) in tests

### 12.2 Font registry

- [x] Define face key: family + weight bucket (400/700) + style (normal/italic)
- [x] Register bundled faces at init: `Liberation Sans` / `sans-serif` defaults
- [x] API: `ResolveFace(familyList, weight, italic) (*Font, error)` with fallback chain
- [x] Unit tests: bold request → bold face; missing italic → regular (documented), not crash
- [x] Keep single-module, no network font download in this phase

### 12.3 PDF multi-face embed

- [x] `ensureFont` supports multiple faces per document (`F0`, `F1`, …)
- [x] Content stream font switches per run style
- [x] Subset + ToUnicode still valid per face
- [x] Test: PDF contains ≥2 `/FontFile2` or distinct BaseFont entries for bold sample
- [x] Path: `internal/pdf/{fonts.go,fontpdf.go,content.go}`

### 12.4 Layout measurement

- [x] Inline layout measures with **selected face** advances, not only Regular
- [x] Bold headings wider than regular same size (test)
- [x] `<b>`, `<strong>`, `font-weight: bold|700+` select bold
- [x] `<i>`, `<em>`, `font-style: italic|oblique` select italic when face present
- [x] Nested bold+italic selects BoldItalic when available
- [x] Path: `internal/layout/{inline.go,style.go,paint.go}`

### 12.5 Kill default fake-bold

- [x] When bold face available, paint with fill mode 0 (no stroke fake)
- [x] When bold face missing, keep fake-bold as fallback and document
- [x] Same rule for HF text bold (`internal/convert/hf.go`)
- [x] Test asserting fake-bold path not used for default report fixtures

### 12.6 CSS font-family partial wiring

- [x] Honor generic families: `sans-serif` → Liberation Sans; optional `serif`/`monospace` only if faces bundled - else fallback + document
- [x] Named family match case-insensitive against registry
- [x] Unknown family → default sans (no crash)
- [x] Matrix §2.3 update: font-family Partial → improved notes

### 12.7 Fixtures & samples

- [x] Extend fixture-18 (typography) or add fixture-21: bold/italic/strong/em matrix
- [x] `make samples` regenerates PDF; visual check headings are real bold
- [x] Golden structure tests still pass

### 12.8 Docs

- [x] `compatibility-matrix.md` §2.3: real bold/italic status
- [x] `fidelity.md` / samples note
- [x] README feature line: “real bold faces” when shipped
- [x] Explicit: still no OpenType shaping / kerning claim

### 12.9 Closure gates

- [x] `make lint` →
- [x] `make test` →
- [x] At least two PDF faces validated end-to-end
- [x] Parent Phase 12 rows checked
- [x] Next: **Phase 13** (spacing) in parallel-capable with 14

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| PDF font embed (phase 3) | Image mode raster (15), i18n (19) |
| Style font-weight/style parse | Layout consumers |

---

## Out of scope

- CJK / CID fonts (phase 19)
- System font discovery (phase 19)
- Full kerning / GSUB/GPOS
- Image-mode face paint (phase 15 uses registry from here)

## Evidence (reconcile 2026-08-04)

- Assets: `internal/pdf/assets/LiberationSans-{Regular,Bold,Italic,BoldItalic}.ttf` + `assets.go` embed
- Registry: `internal/pdf/faces.go` `FaceSet.Resolve(weight, italic)`
- Layout: `faceFor` in `layout.go`; measure + paint use selected face (`inline.go`)
- Fake bold only when face missing: `paint.go` `fakeBold := op.Bold && !op.Font.Bold()`
- Tests: `TestRealBoldFaceOps`, `faces_test.go`, matrix §2.3 Implemented for weight/style
- Closure: `make test` / `make lint` green on master

