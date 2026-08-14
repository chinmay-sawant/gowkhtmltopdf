# Phase 2 — PDF/A-3 Archival Objects

> **Parent:** `plans/0.2.2/pdf-1.7-compliance-plan/00-canonical-pdf-17-compliance-plan.md`
> **Status:** completed
> **Estimated effort:** 4–6 days
> **Depends on:** Phase 1
> **Unblocks:** phases 3, 7 (A-3 half)
> **Spec:** ISO 19005-3; [SPEC-NOTES.md](SPEC-NOTES.md) §3

---

## Overview

Unclaimed 1.7 already has a Metadata stream without `pdfaid`. PDF/A-3
requires a **claiming** packet plus an OutputIntent and an ICC profile.

Do not omit Info (1.7 still allows it). Keep Info consistent with XMP.
Do not add structure tags here (that is A-3a’s UA half, phase 4).

---

## Executive Summary

| Object | Unclaimed 1.7 | A-3a / dual |
|--------|---------------|-------------|
| XMP `pdfaid:part` | absent | `3` |
| XMP `pdfaid:conformance` | absent | `A` |
| OutputIntent | absent | `/S /GTS_PDFA1` + DestOutputProfile |
| ICC | absent | sRGB (and gray if we emit DefaultGray) |
| Trailer `/ID` | present | present |
| Info | present | present, dates aligned with XMP |

---

## Phase 2 checklist

### 2.1 Claiming XMP

- [x] When the profile includes PDF/A-3, the Metadata stream contains `pdfaid:part` = 3 and `pdfaid:conformance` = A
- [x] Unclaimed 1.7 XMP still has **no** `pdfaid`
- [x] Keep `dc:format`, `pdf:Producer`, create/modify dates from `SetCreationTime`
- [x] Test: byte / XML parse of the Metadata stream

### 2.2 OutputIntent + ICC

- [x] Embed a compact sRGB ICC (process-once, not per page)
- [x] Catalog `/OutputIntents` array with `/Type /OutputIntent`, `/S /GTS_PDFA1`, `/DestOutputProfile`
- [x] Object IDs reserved before catalog write (same finalize discipline as outlines)
- [x] Unclaimed 1.7 still has no `/OutputIntents`
- [x] Test: catalog references a real ICC stream (`/N 3`)

### 2.3 Info vs XMP

- [x] Info `ModDate` / `CreationDate` match the XMP dates
- [x] Producer in Info matches `pdf:Producer`
- [x] Title, if set, appears in both Info `/Title` and `dc:title`

### 2.4 Negative

- [x] Profile set but finalize cannot build ICC → error, no claiming file
- [x] Test: unclaimed 1.7 goldens unchanged

---

## Explicitly out of scope

- ICCBased rewrite of images (phase 3)
- `pdfuaid` (phase 4)
- veraPDF (phase 7)

---

## Done when

A hand-built A-3a policy document has claiming XMP and an OutputIntent,
and unclaimed 1.7 fixtures are unchanged.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 profile | Phase 3 color; phase 7 A-3 fixture |
