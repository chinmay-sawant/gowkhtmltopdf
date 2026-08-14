# Phase 7 — Documentation and Honesty

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** completed (2026-08-15)
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

- [x] `documentation/architecture.md` package map: writer is version-aware, default 1.4
  (row now reads "PDF writer (default 1.4, opt-in 1.7 / 2.0 via
  `WriterPolicy`…"; the PDF-writer section's table lists `%PDF-1.4` /
  `%PDF-1.7` / `%PDF-2.0` headers, trailer `/ID` on 1.7+2.0, Info string
  encoding per version, non-claiming XMP on 1.7+2.0, and a new "Catalog
  `/Version` — not emitted" row)
- [x] Pipeline diagram still shows one `internal/pdf` sink (not a second engine)
  (the diagram in `architecture.md` keeps a single `internal/pdf` branch
  labeled `(1.4 default / 1.7 & 2.0 opt-in)`; no engine split)
- [x] `documentation/architecture/09-pdf-writer.md` records `WriterPolicy`, header, trailer `/ID`, Info+XMP, classic xref
  (§1 version-policy bullet: `WriterPolicy` with `PDF14` default, `PDF17` /
  `PDF20` opt-in, header spelling, classic xref on all versions, `/ID` on
  1.7 and 2.0, catalog `/Version` deliberately not emitted; §4.2 finalize
  step 6: Info encoding per version + XMP `/Metadata` on 1.7 and 2.0;
  §4.3: header and trailer rows updated; §6 decision table row updated;
  §10 limitation bullet rewritten to "PDF 2.0 shipped as opt-in version,
  not a conformance claim")
- [x] `documentation/architecture/README.md` table row updated
  (package-map row + §5 "PDF mode" bullet now say 1.4 default, 1.7 / 2.0
  opt-in, UTF-8 text strings on 2.0, non-claiming XMP on 1.7 and 2.0, and
  "PDF 2.0 is a version, not PDF/A-4 or PDF/UA-2"; diagram line updated)
- [x] `documentation/architecture/08-convert-pipeline.md` notes where the policy is applied (`NewDocument*`)
  (§4.1 flow now shows `policy := PolicyForGlobal(req.Global)` at
  `convert.go:229-260` feeding the single
  `pdf.NewDocumentWithPolicy(policy)` construction; §1 and §6 already
  named `NewDocumentWithPolicy`)

### 7.2 Compatibility and deferred

- [x] `documentation/compatibility-matrix.md`: PDF **version** (1.4 default, 2.0 opt-in) is a separate row from PDF/A, encryption, AcroForm
  (§5 rows now: "PDF version (1.4 / 1.7 / 2.0)" — supported, with 2.0
  described as a version not a conformance claim; "PDF 2.0 (ISO 32000-2)"
  row changed from Deferred to "Shipped as opt-in version (#32)"; PDF/A-4
  / PDF/UA-2 stay "Deferred to #33" with
  `pdf.ErrConformanceProfilesUnsupported`; CLI flag row updated to
  "1.7 / 2.0 opt-in; invalid values error")
- [x] `documentation/deferred.md`: PDF/A, PDF/UA, encryption stay deferred / pointed at #33
  (PDF 2.0 row now "Shipped (0.2.2, #32): opt-in version … not a
  conformance claim"; PDF/A-4 / PDF/UA-2 row stays deferred to #33;
  encryption/forms/signatures row now notes rejection on every version
  incl. 2.0)
- [x] No sentence that "PDF 2.0" implies archival or accessibility conformance
  (every edited sentence that names 2.0 pairs it with "a version, **not**
  PDF/A-4 or PDF/UA-2 (#33)"; no doc links 2.0 to archival/accessibility)

### 7.3 User surfaces

- [x] `README.md` output claim matches tests
  (feature table row now: "PDF 1.4 / PDF 1.7 / PDF 2.0 output — Yes —
  PDF 1.4 default; opt-in 1.7 / 2.0 via `--pdf-version` / `WithPDFVersion`.
  PDF 2.0 is a **version**, not PDF/A-4 or PDF/UA-2")
- [x] `documentation/cli.md` documents `--pdf-version`
  (example block gains a `--pdf-version 2.0` line labeled "opt-in version
  — NOT PDF/A-4 or PDF/UA-2"; flag table row updated with the 2.0 emit
  details and the invalid-value error)
- [x] `documentation/library-api.md` documents the setting on the preferred `PDFRequest` path
  (builder example now uses `WithPDFVersion("2.0")` with the "1.4 default /
  1.7 / 2.0 — 2.0 is a version, not PDF/A-4 / PDF/UA-2" comment; the
  `pdfversion` dotted-key row updated likewise)
- [x] `documentation/overview.md` pipeline line no longer says "PDF 1.4 write" as the only possibility
  (pipeline line and both binary/library table rows now read
  "1.4 default / 1.7 & 2.0 opt-in")

### 7.4 In-tree comments

- [x] `internal/pdf` package comment (`pdf.go` / `doc.go`) describes default 1.4 + opt-in 2.0
  (`doc.go` now: "version-aware PDF document generator (PDF 1.4 default,
  opt-in PDF 1.7 and 2.0) … Emitting PDF 2.0 is a version choice, not a
  PDF/A-4 or PDF/UA-2 conformance claim (#33)"; `pdf.go` package comment
  likewise, and its "stdlib-only" wording was corrected to name the one
  allowlisted shaping exception so it cannot be read as a zero-dependency
  claim)
- [x] Producer / `Version` comments do not claim a single hard-coded 1.4 header
  (`policy.go` `HeaderVersion()` and `ProducerVersion()` comments now list
  "1.4", "1.7", "2.0"; `const Version = "1.4"` stays but is commented as
  the **default** legacy header, which is accurate)

### 7.5 Claim scan

- [x] `make claim-scan` (or the repo's equivalent honesty check) passes
  (ran 2026-08-15: `claim-scan: clean`, exit 0 — no forbidden phrase in
  README.md, doc.go, documentation/*.md, frontend content, or help.go)
- [x] This plan's feature matrix in `00-canonical-pdf-20-plan.md` is updated to match shipped behavior
  (catalog `/Version` row → "not emitted; header is the sole version
  authority" with the decision recorded; XMP row keeps "without A-4/UA-2
  claims"; Info/trailer/header rows already matched and are now annotated
  with their tests; success criteria all [x] with evidence)

---

## Explicitly out of scope

- Frontend marketing pages beyond what `claim-scan` already covers
- Rewriting comparison essays except a one-line version note if they assert "1.4 only" as a current product fact
- #33 design docs

(Applied as allowed: `sebastiaanklippert-go-wkhtmltopdf.md` rendering row and
`landscape-2026.md/.html` score card got one-line version corrections — the
former asserted "PDF 1.4" as the only output, the latter "no PDF/A" which
contradicts the shipped 1.7 profile; frontend pages were left untouched.)

---

## Done when

A reader of README + compatibility matrix can tell 1.4 default, 2.0
opt-in, and "not PDF/A" apart, and those sentences match tests.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 6 evidence | Phase 8 closure |
