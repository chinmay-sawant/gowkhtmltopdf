# Phase 7 — veraPDF and Goldens

> **Parent:** `plans/0.2.2/pdf-1.7-compliance-plan/00-canonical-pdf-17-compliance-plan.md`
> **Status:** not started
> **Estimated effort:** 3–5 days
> **Depends on:** Phases 2–6
> **Unblocks:** phase 8 docs

---

## Overview

Proof is independent validation, not our own XMP string. Use
**veraPDF** flavours that match 1.7:

| Claim | Flavour |
|-------|---------|
| PDF/A-3a | `-f 3a` |
| PDF/UA-1 | `-f ua1` |

Do **not** run `-f 4` or `-f ua2` as a pass gate (those are #33).

If `verapdf` is missing, tests skip. Do not add a Go module for it.

Keep 1.4 / unclaimed 1.7 goldens. Add a small dual HTML fixture.

---

## Fixture matrix

| Fixture | Description | `-f 3a` | `-f ua1` |
|---------|-------------|---------|----------|
| `minimal-text` | title + H1 + P, Liberation | PASS | PASS |
| `table-simple` | TH + TD | PASS | PASS |
| `figure-alt` | img with alt | PASS | PASS |
| `link-annot` | URI link + Tabs /S | PASS | PASS |
| `hf-artifact` | header/footer present | PASS | PASS |

---

## Phase 7 checklist

### 7.1 Harness

- [ ] Document install (`VERAPDF_BIN` or `PATH`)
- [ ] Helper script or Go test that invokes `-f 3a` and `-f ua1`
- [ ] Skip when the binary is absent; do not fail CI
- [ ] Record the veraPDF version when the test runs

### 7.2 Positive

- [ ] Generate each matrix row through `convert` / `RunPDF` with the dual profile
- [ ] Needles: `%PDF-1.7`, `pdfaid:part`, `pdfaid:conformance`, `pdfuaid:part`, `/OutputIntents`, `/StructTreeRoot`
- [ ] veraPDF both flavours PASS when the binary exists

### 7.3 Negative / isolation

- [ ] Unclaimed 1.7 fixture does **not** contain `pdfaid` / `pdfuaid`
- [ ] Default 1.4 goldens unchanged
- [ ] Dual fixture bytes do not contain `pdfaid:part>4` or `pdfuaid:part>2` wait — assert part is `3` and `1`
- [ ] Do not treat veraPDF `-f 4` PASS as a #31/#this-plan success

### 7.4 Structure check (optional)

- [ ] If easy: assert ParentTree cell ownership in-process (no Python required)
- [ ] Table MCID owned by TD/TH, not TR

---

## Explicitly out of scope

- PAC / Adobe Accessibility Checker as a required gate
- Regenerating the whole `testdata/golden` corpus

---

## Done when

The dual HTML fixture is in-tree, needles pass, and veraPDF `3a`+`ua1`
either PASS or skip with the reason recorded.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 2–6 | Phase 8 claims |
