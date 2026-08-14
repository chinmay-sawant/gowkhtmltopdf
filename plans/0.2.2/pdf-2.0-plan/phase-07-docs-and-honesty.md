# Phase 7 — Documentation and Honesty

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** not started
> **Estimated effort:** 1–2 days
> **Depends on:** Phase 6 (do not document unproven behavior)
> **Unblocks:** phase 8

---

## Overview

User-facing docs currently say the writer is PDF 1.4 only. After phases
1–6, they must say:

- default output is still PDF 1.4
- PDF 2.0 is an **opt-in version**
- PDF 2.0 is **not** PDF/A-4 or PDF/UA-2

Update docs only where the claim is backed by phase-6 tests.

---

## Executive Summary

| Doc | Today | Target |
|-----|-------|--------|
| `README.md` | PDF 1.4 output | 1.4 default; 2.0 selectable |
| `documentation/architecture.md` | PDF 1.4 writer | version policy + default |
| `documentation/architecture/09-pdf-writer.md` | 1.4-only deep dive | policy, header, `/ID`, XMP, what is not claimed |
| `documentation/compatibility-matrix.md` | encryption/PDF/A out of scope | version row separate from conformance row |
| `documentation/library-api.md` / `cli.md` | no version flag | document `--pdf-version` / setting |
| `documentation/deferred.md` | PDF/A deferred | still deferred; 2.0 version is not a conformance claim |
| `internal/pdf` package comment | "PDF 1.4 writer" | version-aware writer, default 1.4 |

---

## Phase 7 checklist

### 7.1 Architecture

- [ ] `documentation/architecture.md` package map: writer is version-aware, default 1.4
- [ ] Pipeline diagram still shows one `internal/pdf` sink (not a second engine)
- [ ] `documentation/architecture/09-pdf-writer.md` records `WriterPolicy`, header, trailer `/ID`, Info+XMP, classic xref
- [ ] `documentation/architecture/README.md` table row updated
- [ ] `documentation/architecture/08-convert-pipeline.md` notes where the policy is applied (`NewDocument*`)

### 7.2 Compatibility and deferred

- [ ] `documentation/compatibility-matrix.md`: PDF **version** (1.4 default, 2.0 opt-in) is a separate row from PDF/A, encryption, AcroForm
- [ ] `documentation/deferred.md`: PDF/A, PDF/UA, encryption stay deferred / pointed at #33
- [ ] No sentence that "PDF 2.0" implies archival or accessibility conformance

### 7.3 User surfaces

- [ ] `README.md` output claim matches tests
- [ ] `documentation/cli.md` documents `--pdf-version`
- [ ] `documentation/library-api.md` documents the setting on the preferred `PDFRequest` path
- [ ] `documentation/overview.md` pipeline line no longer says "PDF 1.4 write" as the only possibility

### 7.4 In-tree comments

- [ ] `internal/pdf` package comment (`pdf.go` / `doc.go`) describes default 1.4 + opt-in 2.0
- [ ] Producer / `Version` comments do not claim a single hard-coded 1.4 header

### 7.5 Claim scan

- [ ] `make claim-scan` (or the repo's equivalent honesty check) passes
- [ ] This plan's feature matrix in `00-canonical-pdf-20-plan.md` is updated to match shipped behavior

---

## Explicitly out of scope

- Frontend marketing pages beyond what `claim-scan` already covers
- Rewriting comparison essays except a one-line version note if they assert "1.4 only" as a current product fact
- #33 design docs

---

## Done when

A reader of README + compatibility matrix can tell 1.4 default, 2.0
opt-in, and "not PDF/A" apart, and those sentences match tests.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 6 evidence | Phase 8 closure |
