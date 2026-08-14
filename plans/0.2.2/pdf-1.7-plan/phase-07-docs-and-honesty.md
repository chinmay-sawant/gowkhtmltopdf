# Phase 7 — Documentation and Honesty

> **Parent:** `plans/0.2.2/pdf-1.7-plan/00-canonical-pdf-17-plan.md`
> **Status:** completed
> **Estimated effort:** 1–2 days
> **Depends on:** Phase 6 (do not document unproven behavior)
> **Unblocks:** phase 8

---

## Overview

User-facing docs currently say the writer is PDF 1.4 only. After phases
1–6, they must say:

- default output is still PDF 1.4
- PDF 1.7 is an **opt-in version** (ISO 32000-1)
- PDF 1.7 is **not** PDF 2.0, PDF/A-4, or PDF/UA-2
- 1.7 Unicode strings are UTF-16BE, not UTF-8

Update docs only where the claim is backed by phase-6 tests.

---

## Executive Summary

| Doc | Today | Target |
|-----|-------|--------|
| `README.md` | PDF 1.4 output | 1.4 default; 1.7 selectable |
| `documentation/architecture.md` | PDF 1.4 writer | version policy + default |
| `documentation/architecture/09-pdf-writer.md` | 1.4-only deep dive | policy, header, `/ID`, XMP, UTF-16BE, what is not claimed |
| `documentation/compatibility-matrix.md` | encryption/PDF/A out of scope | version row separate from conformance row |
| `documentation/library-api.md` / `cli.md` | no version flag | document `--pdf-version` |
| `documentation/deferred.md` | PDF/A deferred | still deferred; 1.7 version is not a conformance claim |
| `internal/pdf` package comment | "PDF 1.4 writer" | version-aware writer, default 1.4 |

---

## Phase 7 checklist

### 7.1 Architecture

- [x] `documentation/architecture.md` package map: writer is version-aware, default 1.4 (`documentation/architecture.md:38-42, 60-64, 116-126`)
- [x] Pipeline diagram still shows one `internal/pdf` sink (not a second engine) (`documentation/architecture.md:60-64`)
- [x] `documentation/architecture/09-pdf-writer.md` records `WriterPolicy`, header, trailer `/ID`, Info+XMP, UTF-16BE, classic xref (`documentation/architecture/09-pdf-writer.md:15-35, 78-86, 250-272, 355-360, 490-502`)
- [x] `documentation/architecture/README.md` table row updated (`documentation/architecture/README.md:82, 164-170`)
- [x] `documentation/architecture/08-convert-pipeline.md` notes where the policy is applied (`NewDocument*`) (`documentation/architecture/08-convert-pipeline.md:42-45, 230-232, 336-340`)

### 7.2 Compatibility and deferred

- [x] `documentation/compatibility-matrix.md`: PDF **version** (1.4 default, 1.7 opt-in) is a separate row from PDF 2.0, PDF/A, encryption, AcroForm (`documentation/compatibility-matrix.md:257-261, 314`)
- [x] `documentation/deferred.md`: PDF/A, PDF/UA, encryption stay deferred / pointed at #33; PDF 2.0 pointed at #32 (`documentation/deferred.md:74-76`)
- [x] No sentence that "PDF 1.7" implies archival or accessibility conformance (`documentation/compatibility-matrix.md:259-260`, `documentation/deferred.md:75`)

### 7.3 User surfaces

- [x] `README.md` output claim matches tests (`README.md:24`)
- [x] `documentation/cli.md` documents `--pdf-version` (`documentation/cli.md:89-91, 275`)
- [x] `documentation/library-api.md` documents the setting on the preferred `PDFRequest` path (`documentation/library-api.md:652, 709`)
- [x] `documentation/overview.md` pipeline line no longer says "PDF 1.4 write" as the only possibility (`documentation/overview.md:96, 113, 116`)

### 7.4 In-tree comments

- [x] `internal/pdf` package comment (`pdf.go` / `doc.go`) describes default 1.4 + opt-in 1.7 (`internal/pdf/doc.go:1-4`, `internal/pdf/pdf.go:1-5`)
- [x] Producer / `Version` comments do not claim a single hard-coded 1.4 header (`internal/pdf/pdf.go:27-28`, `internal/pdf/policy.go:88-105`)

### 7.5 Claim scan

- [x] `make claim-scan` (or the repo's equivalent honesty check) passes (clean zero claims violation)
- [x] This plan's feature matrix in `00-canonical-pdf-17-plan.md` is updated to match shipped behavior (`00-canonical-pdf-17-plan.md`)

---

## Explicitly out of scope

- Frontend marketing pages beyond what `claim-scan` already covers
- Rewriting comparison essays except a one-line version note if they assert "1.4 only" as a current product fact
- #32 / #33 design docs

---

## Done when

A reader of README + compatibility matrix can tell 1.4 default, 1.7
opt-in, "not PDF 2.0", and "not PDF/A" apart, and those sentences
match tests.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 6 evidence | Phase 8 closure |
