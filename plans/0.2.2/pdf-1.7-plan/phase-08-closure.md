# Phase 8 — Closure

> **Parent:** `plans/0.2.2/pdf-1.7-plan/00-canonical-pdf-17-plan.md`
> **Status:** completed
> **Estimated effort:** 1 day after 1–7
> **Depends on:** Phases 1–7
> **Unblocks:** issue #31 can be closed when evidence is attached; #32 may extend the same policy

---

## Overview

Close the #31 ledger only with current evidence. Do not mark #29, #32,
or #33 done. Do not treat a green `make test` as a PDF/A or PDF 2.0
claim.

---

## Phase 8 checklist

### 8.1 Required checks

- [x] `make lint` → record outcome: PASSED cleanly with zero errors across all packages
- [x] `make test` → record outcome: PASSED cleanly (all unit, golden, and integration tests passed)
- [x] 1.4 default proof: a convert/golden test without `--pdf-version` still starts with `%PDF-1.4` (`internal/convert/golden_test.go:737-754`, `internal/convert/convert_test.go:733-748`)
- [x] 1.7 opt-in proof: the phase-6 needle test is green (`internal/convert/golden_test.go:683-735`, `TestConvertPDF17Needles`)

### 8.2 Ledger hygiene

- [x] Every in-scope row in phases 1–7 is `[x]` only with concrete evidence
- [x] Parent `00-canonical-pdf-17-plan.md` success-criteria boxes match reality (`00-canonical-pdf-17-plan.md`)
- [x] `plans/0.2.2/README.md` status line updated (`plans/0.2.2/README.md`)
- [x] `WriterPolicy` is the single type; the 2.0 plan extends it rather than redefining it (`internal/pdf/policy.go:45-53`)

### 8.3 Issue boundary

- [x] #31 / this plan still lists #32 and #33 work as out of scope
- [x] No catalog `/MarkInfo`, `/StructTreeRoot`, `/OutputIntents`, or `pdfaid` in default or 1.7 fixtures (`internal/pdf/policy_test.go:498-534`, `internal/convert/golden_test.go:723-734`)
- [x] No UTF-8 text strings on the 1.7 path (UTF-16BE only) (`internal/pdf/pdf.go:1036-1076`)
- [x] `PDF20` is still reserved/rejected unless #32 has landed (`internal/pdf/policy.go:56-58`)

### 8.4 What this does **not** close

- [x] Epic #29 remains open until #32 and #33 are decided or done (explicitly documented as separate scope)
- [x] PDF 2.0 feature completeness remains #32 (explicitly deferred)
- [x] PDF/A-4 and PDF/UA-2 remain #33 (explicitly deferred)

---

## Done when

Lint and test are recorded, 1.4 is still the default, 1.7 is opt-in
and proven, and the sibling 2.0 plan can extend this policy without
inventing a second type.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 1–7 | Issue #31 evidence; #32 consumes `WriterPolicy` |
