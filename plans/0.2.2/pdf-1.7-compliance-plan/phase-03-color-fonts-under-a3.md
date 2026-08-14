# Phase 3 — Color and Fonts under PDF/A-3

> **Parent:** `plans/0.2.2/pdf-1.7-compliance-plan/00-canonical-pdf-17-compliance-plan.md`
> **Status:** not started
> **Estimated effort:** 3–5 days
> **Depends on:** Phase 2 (ICC object exists)
> **Unblocks:** phase 7 A-3 color/font gates

---

## Overview

The writer already embeds subset TTF / Type0 with ToUnicode and paints
DeviceRGB JPEG/PNG. Under PDF/A-3, uncalibrated DeviceRGB is a
typical veraPDF failure. Fonts must be embedded; a missing embed must
fail the job, not produce a claiming file with a standard-14 name.

Do not add a second subsetter.

---

## Phase 3 checklist

### 3.1 Images and page color

- [ ] Under A-3 / dual, image XObjects use ICCBased (or DefaultRGB on the page) tied to the phase-2 sRGB profile — `images.go`
- [ ] JPEG `/DCTDecode` and PNG Flate + SMask still work (A-3 allows transparency)
- [ ] Grayscale path uses ICC gray or DefaultGray
- [ ] Unclaimed 1.4 / 1.7 images stay DeviceRGB
- [ ] Test: existing JPEG/PNG tests pass unclaimed; new A-3 cases assert ICCBased / DefaultRGB

### 3.2 Fonts

- [ ] A-3 / UA-1 / dual: every text font used on a page is embedded (`FontFile2`)
- [ ] Every text font has `ToUnicode` (needed for A-3u/A-3a and UA-1)
- [ ] If a face cannot be subset/embedded, `Write` / convert returns a typed error **before** a claiming PDF exists
- [ ] Test: existing Type0 tests pass under `PDF17` + A-3a policy
- [ ] Test: negative — force an unembeddable condition if one exists; otherwise document that all shipped faces embed

### 3.3 ExtGState / opacity

- [ ] Soft-mask + `/ca` remain allowed (A-3 ≠ A-1)
- [ ] Test: PNG-with-alpha fixture under A-3 still produces `/SMask`

---

## Explicitly out of scope

- CMYK / DeviceN / spot
- JPEG 2000
- Structure tags (phase 4)

---

## Done when

An A-3 policy document with text + JPEG + PNG-alpha has ICC-managed
color, embedded fonts, ToUnicode, and unclaimed goldens are untouched.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 2 ICC | Phase 7 |
