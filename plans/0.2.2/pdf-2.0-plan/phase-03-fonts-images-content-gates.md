# Phase 3 — Fonts, Images, and Content Gates

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** completed (2026-08-15)
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

**What #31 already provided (verified against the landed 1.7 code):**
`ensureFont`/`emitSimple`/`emitType0` already take the policy implicitly
via `Document`; `WriterPolicy.Validate()` already rejects
encryption/forms/signatures/object-streams before any byte is written;
`buildPageResources` already assembles `/Resources` at finalize; the
gate tests (`TestFeatureGatesFailClosed`) already existed with a
"no partial file on `WriteTo`" shape. The 2.0 deltas here were: the
`/ProcSet` omission on 2.0 pages, a more honest `#33` deferral sentinel
for 2.0-era conformance profiles, and PDF20-policy tests for every
existing 1.7 test family (fonts, images, xref, rich document).

---

## Phase 3 checklist

### 3.1 Font emit under 2.0

- [x] `ensureFont` / `emitSimple` / `emitType0` take policy implicitly via `Document` — `fonttype0.go` (already from #31; unchanged in this phase)
- [x] 2.0 documents still subset, still use Identity-H, still emit `ToUnicode` — `TestType0CJKEmbeddingPDF20` builds under `WriterPolicy{Version: PDF20}` and asserts `/Subtype /Type0`, `/CIDFontType2`, `/Encoding /Identity-H`, `/ToUnicode`, `begincmap`/`beginbfchar`, and the CID hex `Tj`
- [x] No second subsetter or Liberation download path — none added; `fonts.go` / `subset.go` / `fonttype0.go` untouched (grep-verified: they read no `policy.Version`)
- [x] Test: existing Type0 / mixed-Latin tests pass when the document policy is `PDF20` — `TestType0CJKEmbeddingPDF20` + `TestType0MixedLatinFallbackPDF20` (F0_u Type0 sibling, Liberation Latin fallback, `(Hello )` simple run, `<4E2D6587>` CIDs)
- [x] Test: 1.4 font cache keys do not collide with 2.0 keys if encoding ever differs (include version in the cache key if emit bytes diverge) — emit bytes do **not** diverge: subset keys (`v%d|%x|%s|%s` in `unionFontRunes`) carry no version, and the font/image emit paths read no policy. Version-independence is therefore by design and documented here; `TestPDF20FontCacheKeysVersionIndependent` asserts a 1.4 and a 2.0 document with the same face+rune set produce the same cache key and byte-identical `FontFile2` streams

### 3.2 Image emit under 2.0

- [x] JPEG pass-through and PNG Flate + SMask work on a 2.0 document — `images.go` (unchanged); `TestJPEGAndPNGImagesPDF20` asserts `/Filter /DCTDecode`, `/SMask`, `/J1 Do`, `/P1 Do` under `%PDF-2.0`
- [x] Color space stays DeviceRGB / DeviceGray (no ICC rewrite here; that is #33) — same test asserts `/ColorSpace /DeviceRGB` and the absence of `/ICCBased` / `/OutputIntents`; `TestGrayscaleJPEGFoldPDF20` asserts the grayscale fold keeps `/DeviceGray`
- [x] Size caps (`validateEmbeddedImage`) unchanged — function untouched; still covered by `TestValidateEmbeddedImageCapsPDF17` (caps are version-independent)
- [x] Test: existing JPEG/PNG tests pass with `PDF20` — `TestJPEGAndPNGImagesPDF20`, `TestGrayscaleJPEGFoldPDF20` (plus 1.4/1.7 suites still green)

### 3.3 Content stream and resources

- [x] `Content` operators stay as they are — `content.go` untouched (zero-line diff)
- [x] Page `/Resources` still built at finalize from used fonts/images/ExtGState — `buildPageResources` still called from `finalizePage`, now version-aware (`d.policy.Version` parameter)
- [x] Do not start emitting `/ProcSet` (unnecessary and unhelpful on 2.0) — `buildPageResources` omits the `/ProcSet` entry on PDF 2.0 pages (`/ProcSet` was removed in ISO 32000-2); 1.4/1.7 pages keep it, so `TestPDF17RichDocument` and 1.4 bit-compat tests stay green. `TestPDF20RichDocument` and `TestJPEGAndPNGImagesPDF20` assert the 2.0 resources dict contains no `/ProcSet`
- [x] Test: `TestRichDocStructure` (or a 2.0 clone) covers graphics, text, link, outline, image on `%PDF-2.0` — `TestPDF20RichDocument`: fills/strokes, ExtGState opacity, text, JPEG + PNG, URI + GoTo annots, outline hierarchy, `ParseSemantic` version `"2.0"`, and a byte-identical second build

### 3.4 Feature gates

- [x] `WriterPolicy` (or a `validatePolicy` on `Document`) lists combinations this writer will not emit — `WriterPolicy.Validate()` in `policy.go` (already from #31); refined so a 2.0-era profile on a PDF 2.0 policy fails with `ErrConformanceProfilesUnsupported` (the #33 deferral sentinel) instead of the 1.7-era `ErrConformanceRequiresPDF17`
- [x] Asking for encryption, forms, signatures, or a standards profile on this path returns a typed error **before** `Write` produces bytes — `NewDocumentWithPolicy` and `finalize` both call `Validate()` before any object byte is emitted
- [x] Test: negative cases return `errors.Is` to a documented sentinel; no partial file on `WriteTo` when validation fails in `finalize` — `TestFeatureGatesFailClosed` extended with PDF20 rows (forms, signatures, object streams, `PDF/A-4`, `PDF/UA-2`); each row asserts `errors.Is` from `NewDocumentWithPolicy`, `doc.Validate`, `doc.WriteTo` (0 bytes), and `doc.Write` (empty buffer). `TestPolicyValidation` updated to the #33 deferral sentinel for PDF20 + `PDF/A-4` / `PDF/UA-2`

### 3.5 Image mode untouched

- [x] `internal/imageout` still compiles against the font/shaping surface only — no changes to `internal/imageout`
- [x] No image-mode setting for PDF version — none added
- [x] Test: `go test ./internal/imageout` green — `ok gowkhtmltopdf/internal/imageout`

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
