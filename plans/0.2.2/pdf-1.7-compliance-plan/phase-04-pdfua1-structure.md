# Phase 4 — PDF/UA-1 Structure (Writer)

> **Parent:** `plans/0.2.2/pdf-1.7-compliance-plan/00-canonical-pdf-17-compliance-plan.md`
> **Status:** completed
> **Estimated effort:** 1–2 weeks
> **Depends on:** Phase 1; phase 2 if dual XMP is already there
> **Unblocks:** phase 5
> **Spec:** ISO 14289-1; ISO 32000-1 §14.7–14.8; [SPEC-NOTES.md](SPEC-NOTES.md) §4

---

## Overview

Build the tagged-PDF objects **inside `internal/pdf`**, with a small
API layout can call. Use ISO 32000-1 standard structure types. Do
**not** emit PDF 2.0 `/Namespace` (`http://iso.org/pdf2/ssn`) — that
is UA-2 / #33.

When the profile is UA-1 or dual, untagged paint is a bug: either
layout marked it or finalize fails.

---

## Executive Summary

| Object | Target |
|--------|--------|
| XMP | `pdfuaid:part` = 1; if dual, UA extension schema next to `pdfaid` |
| Catalog | `/Lang`, `/MarkInfo << /Marked true >>`, `/ViewerPreferences << /DisplayDocTitle true >>`, `/StructTreeRoot` |
| StructTreeRoot | `/K` → Document, `/ParentTree` |
| Marked content | `/S << /MCID n >> BDC` … `EMC` |
| Artifacts | `/Artifact << /Type /Pagination … >>` (API ready; HF wiring is phase 5) |
| ParentTree | Nums: page key → array indexed by MCID |
| Page | `/StructParents` when the page has MCIDs; `/Tabs /S` when annots exist |

---

## Phase 4 checklist

### 4.1 XMP UA claim

- [x] UA-1 / dual: `pdfuaid:part` = 1
- [x] Dual: PDF/A extension schema registers `pdfuaid`
- [x] Unclaimed 1.7: still no `pdfuaid`
- [x] Test: negative — no `pdfuaid:part` = 2

### 4.2 Catalog UA keys

- [x] `/Lang` from document language (default `en-US` until phase 5 reads HTML `lang`)
- [x] `/MarkInfo << /Marked true >>`
- [x] `/ViewerPreferences << /DisplayDocTitle true >>`
- [x] Dual / UA-1 with empty title → error (UA-1 requires a display title)
- [x] Test: catalog keys present only when the profile needs them

### 4.3 Structure API

- [x] Types for StructElem, MCID allocation, ParentTree — one module, small exported surface for layout
- [x] Standard types at minimum: `Document`, `H1`, `P`, `Table`, `TR`, `TH`, `TD`, `Figure`, `Link`, `L`, `LI`
- [x] `RoleMap` only if we emit a non-standard name
- [x] No PDF 2.0 Namespace object
- [x] Structure methods no-op when the profile is unclaimed

### 4.4 Marked content + ParentTree

- [x] `Content` helpers for BDC/EMC with MCID
- [x] Per-page MCID from 0
- [x] ParentTree leaf is the element that owns the MCID (TD/TH, not TR)
- [x] Page `/StructParents` only when MCIDs exist
- [x] Link StructElem + `/OBJR` + page `/Tabs /S` for a writer-level link fixture
- [x] Test: one hand-built page (H1 + P + Figure+Alt + table + link) serializes a consistent tree

### 4.5 Finalize order

- [x] Reserve StructTreeRoot before catalog (same rule as outlines)
- [x] Emit Namespace-free tree: StructElems → ParentTree → StructTreeRoot → catalog

---

## Explicitly out of scope

- Mapping HTML (phase 5)
- veraPDF (phase 7)
- PDF/UA-2 namespaces (#33)

---

## Done when

A writer-only fixture (no HTML) is a tagged 1.7 file with UA-1 catalog
keys and a ParentTree that matches MCIDs.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1; phase 2 XMP if dual | Phase 5 |
