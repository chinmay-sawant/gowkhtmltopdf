# Phase 8 — Docs and Closure

> **Parent:** `plans/0.2.2/pdf-1.7-compliance-plan/00-canonical-pdf-17-compliance-plan.md`
> **Status:** completed
> **Estimated effort:** 1–2 days
> **Depends on:** Phase 7 evidence
> **Unblocks:** honest 1.7 compliance claim

---

## Overview

Close this ledger only with phase-7 evidence. Do not close #29, #32,
or #33. Do not call unclaimed 1.7 “PDF/A” or “accessible.”

---

## Phase 8 checklist

### 8.1 Docs

- [x] `README.md` / overview: 1.4 default; 1.7 opt-in version; A-3a+UA-1 opt-in **profile**
- [x] `documentation/compatibility-matrix.md`: separate rows for version vs PDF/A-3a vs PDF/UA-1 vs A-4/UA-2
- [x] `documentation/deferred.md`: A-4 / UA-2 still #33; A-3b/A-3u not product modes unless shipped
- [x] `documentation/cli.md` and `library-api.md` document `--pdf-profile` / setter
- [x] `architecture/09-pdf-writer.md` notes claiming XMP, OutputIntent, structure
- [x] `make claim-scan` (or equivalent) passes

### 8.2 Required checks

- [x] `make lint` → record outcome (clean, 0 lint warnings)
- [x] `make test` → record outcome (all packages pass, 100% tests green)
- [x] Unclaimed 1.4 still `%PDF-1.4` without `pdfaid`
- [x] Dual path proven by phase-7 needles

### 8.3 Ledger

- [x] Phases 1–7 rows `[x]` only with evidence, or `[~]` with pointer
- [x] Parent success-criteria boxes match reality
- [x] `plans/0.2.2/README.md` lists this plan’s status

### 8.4 What this does not close

- [x] Epic #29 stays open
- [x] PDF 2.0 stays #32
- [x] PDF/A-4 + PDF/UA-2 stay #33

---

## Done when

Docs state the 1.7 highest pair as PDF/A-3a + PDF/UA-1, only after
validators (or recorded skip) back it, and default output is still
unclaimed 1.4.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 7 | User-facing claim |
