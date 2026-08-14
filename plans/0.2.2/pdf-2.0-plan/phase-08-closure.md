# Phase 8 — Closure

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** not started
> **Estimated effort:** 1 day after 1–7
> **Depends on:** Phases 1–7
> **Unblocks:** issue #32 can be closed when evidence is attached; #33 may start later

---

## Overview

Close the #32 ledger only with current evidence. Do not mark #29 or
#33 done. Do not treat a green `make test` as a PDF/A claim.

---

## Phase 8 checklist

### 8.1 Required checks

- [ ] `make lint` → record outcome
- [ ] `make test` → record outcome
- [ ] 1.4 default proof: a convert/golden test without `--pdf-version` still starts with `%PDF-1.4`
- [ ] 2.0 opt-in proof: the phase-6 needle test is green

### 8.2 Ledger hygiene

- [ ] Every in-scope row in phases 1–7 is `[x]` only with evidence, or `[~]` with a pointer
- [ ] Parent `00-canonical-pdf-20-plan.md` success-criteria boxes match reality
- [ ] `plans/0.2.2/README.md` status line updated
- [ ] No leftover gocorepdfengine / Zerodha / `engine/` paths in this folder

### 8.3 Issue boundary

- [ ] #32 description / this plan still lists #33 work as out of scope
- [ ] No catalog `/MarkInfo`, `/StructTreeRoot`, `/OutputIntents`, or `pdfaid` in default or 2.0 fixtures
- [ ] #31 shared policy: `PDF17` is either implemented by the sibling plan or still reserved/rejected — not a silent 1.7 header

### 8.4 What this does **not** close

- [ ] Epic #29 remains open until #31 and #33 are decided or done
- [ ] PDF 1.7 feature completeness remains #31
- [ ] PDF/A-4 and PDF/UA-2 remain #33

---

## Done when

Lint and test are recorded, 1.4 is still the default, 2.0 is opt-in
and proven, and the docs/plan no longer describe another product.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 1–7 | Issue #32 evidence; later #33 can consume the 2.0 path |
