# Phase 3 — Fonts, Images, and Content Gates

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** not started
> **Estimated effort:** 2–4 days
> **Depends on:** Phase 1; phase 2 recommended so finalize is version-aware
> **Unblocks:** phase 6 fixtures that include text and images

---

## Overview

This repository already embeds subset TrueType, Type0/CID, JPEG, PNG,
ExtGState opacity, links, and outlines. The old gocorepdfengine plan
rebuilt all of that under `engine/font` and `engine/layout`. **Do not.**

Phase 3 only asks: is each object the current writer emits still legal
and honest on a PDF 2.0 file? If a combination is not implemented,
reject it at the policy seam before `Write`.

Layout paint (`Content.TextShow`, `AddJPEGImage`, …) stays
version-agnostic. Version decisions belong in `ensureFont`, image
embed, and `finalize`, behind `WriterPolicy`.

---

## Executive Summary

| Path | Today | 2.0 action |
|------|-------|------------|
| Simple WinAnsi font | 1.4 legal | keep |
| Type0 + Identity-H + CIDToGIDMap | 1.4 legal | keep |
| JPEG `/DCTDecode`, PNG Flate + `/SMask` | 1.4 legal | keep |
| ExtGState `/ca` opacity | 1.4 legal | keep |
| Unimplemented 2.0-only (associated files, AES, forms) | n/a | do not emit; error if a caller asks |
| New paint operators | n/a | none |

---

## Phase 3 checklist

### 3.1 Font emit under 2.0

- [ ] `ensureFont` / `emitSimple` / `emitType0` take policy implicitly via `Document` — `fonttype0.go`
- [ ] 2.0 documents still subset, still use Identity-H, still emit `ToUnicode`
- [ ] No second subsetter or Liberation download path
- [ ] Test: existing Type0 / mixed-Latin tests pass when the document policy is `PDF20`
- [ ] Test: 1.4 font cache keys do not collide with 2.0 keys if encoding ever differs (include version in the cache key if emit bytes diverge)

### 3.2 Image emit under 2.0

- [ ] JPEG pass-through and PNG Flate + SMask work on a 2.0 document — `images.go`
- [ ] Color space stays DeviceRGB / DeviceGray (no ICC rewrite here; that is #33)
- [ ] Size caps (`validateEmbeddedImage`) unchanged
- [ ] Test: existing JPEG/PNG tests pass with `PDF20`

### 3.3 Content stream and resources

- [ ] `Content` operators stay as they are — `content.go`
- [ ] Page `/Resources` still built at finalize from used fonts/images/ExtGState
- [ ] Do not start emitting `/ProcSet` (unnecessary and unhelpful on 2.0)
- [ ] Test: `TestRichDocStructure` (or a 2.0 clone) covers graphics, text, link, outline, image on `%PDF-2.0`

### 3.4 Feature gates

- [ ] `WriterPolicy` (or a `validatePolicy` on `Document`) lists combinations this writer will not emit
- [ ] Asking for encryption, forms, signatures, or a standards profile on this path returns a typed error **before** `Write` produces bytes
- [ ] Test: negative cases return `errors.Is` to a documented sentinel; no partial file on `WriteTo` when validation fails in `finalize`

### 3.5 Image mode untouched

- [ ] `internal/imageout` still compiles against the font/shaping surface only
- [ ] No image-mode setting for PDF version
- [ ] Test: `go test ./internal/imageout` green

---

## Explicitly out of scope

- Replacing layout tables/text/images (already in `internal/layout`)
- ICCBased rewrite, DefaultRGB, OutputIntent (#33)
- Tagged BDC/EMC around paint ops (#33)
- WOFF2 / CFF / new font formats

---

## Done when

The same paint API produces a 2.0 file whose fonts, images, links, and
outlines match the 1.4 objects structurally, and unimplemented features
fail closed.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 `Document` policy | Phase 5 convert jobs with real content |
| Existing font/image tests | Phase 6 goldens |
