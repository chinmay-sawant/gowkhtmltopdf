# Phase 3 — Fonts, Images, and Content Gates

> **Parent:** `plans/0.2.2/pdf-1.7-plan/00-canonical-pdf-17-plan.md`
> **Status:** not started
> **Estimated effort:** 2–4 days
> **Depends on:** Phase 1; phase 2 recommended so finalize is version-aware
> **Unblocks:** phase 6 fixtures that include text and images
> **Spec:** ISO 32000-1 §2.1, §6, clause 9, §7.4, §11, §14.2 — [SPEC-NOTES.md](SPEC-NOTES.md)

---

## Overview

This repository already embeds subset TrueType, Type0/CID, JPEG, PNG,
ExtGState opacity, links, and outlines. Those objects are 1.4-legal
and therefore 1.7-legal (§6).

Phase 3 only asks: is each object the current writer emits still legal
and honest on a PDF 1.7 file? If a combination is not implemented,
reject it at the policy seam before `Write`.

Layout paint stays version-agnostic. Version decisions belong in
`ensureFont`, image embed, and `finalize`, behind `WriterPolicy`.

Do not add JPEG 2000, CFF/OTTO, ProcSet, object streams, or optional
content. §2.1 does not require them.

---

## Executive Summary

| Path | Today | 1.7 action |
|------|-------|------------|
| Simple WinAnsi font | 1.4 legal | keep |
| Type0 + Identity-H + CIDToGIDMap | 1.4 legal | keep |
| JPEG `/DCTDecode`, PNG Flate + `/SMask` | 1.4 legal | keep |
| ExtGState `/ca` opacity | 1.4 legal | keep |
| CFF / `OTTO` | rejected | stay rejected |
| Unimplemented 1.5–1.7 (object streams, OCG, AES, JPEG2000) | n/a | do not emit; error if a caller asks |
| New paint operators | n/a | none |

---

## Phase 3 checklist

### 3.1 Font emit under 1.7

- [ ] `ensureFont` / `emitSimple` / `emitType0` take policy implicitly via `Document` — `fonttype0.go`
- [ ] 1.7 documents still subset, still use Identity-H, still emit `ToUnicode`
- [ ] No second subsetter; CFF/`OTTO` still errors
- [ ] Test: existing Type0 / mixed-Latin tests pass when the document policy is `PDF17`
- [ ] Test: 1.4 font cache keys do not collide with 1.7 keys if emit bytes diverge (include version in the cache key if they do)

### 3.2 Image emit under 1.7

- [ ] JPEG pass-through and PNG Flate + SMask work on a 1.7 document — `images.go`
- [ ] Color space stays DeviceRGB / DeviceGray (no ICC rewrite; that is #33)
- [ ] Size caps (`validateEmbeddedImage`) unchanged
- [ ] Test: existing JPEG/PNG tests pass with `PDF17`

### 3.3 Content stream and resources

- [ ] `Content` operators stay as they are — `content.go`
- [ ] Page `/Resources` still built at finalize from used fonts/images/ExtGState
- [ ] Do not start emitting `/ProcSet` (obsolete since 1.4, §14.2; not a 1.7 gate)
- [ ] Test: `TestRichDocStructure` (or a 1.7 clone) covers graphics, text, link, outline, image on `%PDF-1.7`

### 3.4 Feature gates

- [ ] `WriterPolicy` (or `validatePolicy` on `Document`) lists combinations this writer will not emit
- [ ] Asking for encryption, forms, signatures, object streams, or a standards profile on this path returns a typed error **before** `Write` produces bytes
- [ ] Test: negative cases return `errors.Is` to a documented sentinel; no partial file on `WriteTo` when validation fails in `finalize`

### 3.5 Image mode untouched

- [ ] `internal/imageout` still compiles against the font/shaping surface only
- [ ] No image-mode setting for PDF version
- [ ] Test: `go test ./internal/imageout` green

---

## Explicitly out of scope

- Replacing layout tables/text/images
- ICCBased rewrite, DefaultRGB, OutputIntent (#33)
- Tagged BDC/EMC (#33)
- WOFF2 / CFF / JPEG 2000

---

## Done when

The same paint API produces a 1.7 file whose fonts, images, links, and
outlines match the 1.4 objects structurally, and unimplemented
features fail closed.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 `Document` policy | Phase 5 convert jobs with real content |
| Existing font/image tests | Phase 6 goldens |
