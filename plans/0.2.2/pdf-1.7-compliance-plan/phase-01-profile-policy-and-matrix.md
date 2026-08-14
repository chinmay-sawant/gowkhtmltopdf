# Phase 1 — Profile Policy and Matrix

> **Parent:** `plans/0.2.2/pdf-1.7-compliance-plan/00-canonical-pdf-17-compliance-plan.md`
> **Status:** completed
> **Estimated effort:** 2–3 days
> **Depends on:** #31 `WriterPolicy` (`internal/pdf/policy.go`)
> **Unblocks:** phases 2–6

---

## Overview

Today any non-empty `ConformanceProfile` returns
`ErrConformanceProfilesUnsupported` and the message names #33. That
lumps 1.7 profiles (A-3, UA-1) with 2.0 profiles (A-4, UA-2).

This phase names the 1.7 profiles, accepts them on `PDF17` only, and
writes the in-repo matrix. It does **not** emit claiming XMP yet.

---

## Executive Summary

| Surface | Today | Target |
|---------|-------|--------|
| `ConformanceProfile` | any string → error (#33) | typed or allowlisted: empty, `PDF/A-3a`, `PDF/UA-1`, dual |
| `PDF17` + empty profile | ok (unclaimed 1.7) | unchanged |
| `PDF17` + A-3a / UA-1 / dual | error | `Validate()` succeeds; later phases emit |
| `PDF14` + any profile | error | still error (A-3/UA-1 require 1.7) |
| `PDF20` + anything | `ErrReservedPDF20` (historical — sentinel removed in 0.2.2; see note below) | unchanged at the time; superseded by #32 |
| A-4 / UA-2 strings | error (#33) | still error, message stays on #33 |

> **Superseded:** the `PDF20` row describes the state before PDF 2.0
> landed. Since 0.2.2 (issue #32, `plans/0.2.2/pdf-2.0-plan/`), `PDF20` is
> a valid version and `ErrReservedPDF20` no longer exists in code; 1.7-era
> profiles on a 2.0 document still error, but via the version gate
> (`ErrConformanceRequiresPDF17`) or the #33 deferral sentinel
> (`pdf.ErrConformanceProfilesUnsupported`), not via a reserved-version
> sentinel.

---

## Phase 1 checklist

### 1.1 Profile names

- [x] Record the headline dual name and the two singles in `policy.go` (constants or a small `Conformance` type)
- [x] Dual implies both A-3a and UA-1 (one policy value, not two independent booleans that can drift)
- [x] Test: table of strings → accepted / rejected, including aliases you refuse (`pdfa-3a` vs `PDF/A-3a` — pick one spelling)

### 1.2 Validate rules

- [x] Empty profile + `PDF14` / `PDF17` still ok
- [x] Accepted 1.7 profile + `PDF17` ok
- [x] Accepted 1.7 profile + `PDF14` → typed error (“requires PDF 1.7”)
- [x] `PDF/A-4`, `PDF/UA-2`, `PDF/A-1b`, unknown → still error; A-4/UA-2 text still cites #33
- [x] Encryption / forms / signatures / object streams still fail even with a profile
- [x] Test: existing `policy_test.go` rows that used `"PDF/UA-1"` as a **negative** case are updated — UA-1 on PDF17 is now valid

### 1.3 Matrix

- [x] Copy or link the feature matrix from the parent ledger into `documentation/` **or** keep it only in the ledger until phase 8
- [x] Matrix states emitted / accepted / validated for A-3a, UA-1, and dual
- [x] Matrix says A-3b / A-3u are not product modes in this ledger

---

## Explicitly out of scope

- Writing pdfaid into XMP (phase 2)
- Structure tree (phase 4)
- Settings / CLI (phase 6)

---

## Done when

`NewDocumentWithPolicy({PDF17, ConformanceProfile: dual})` returns a
document, and `{PDF17, "PDF/A-4"}` still errors.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| `internal/pdf/policy.go` | Phases 2–6 |
