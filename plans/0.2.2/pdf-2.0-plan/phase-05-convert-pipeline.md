# Phase 5 — Convert Pipeline Wiring

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** completed (2026-08-15)
> **Estimated effort:** 2–3 days
> **Depends on:** Phases 1–4
> **Unblocks:** phases 6–7 end-to-end proof

---

## Overview

`convert.Run` always does `doc: pdf.NewDocument()` today
(`internal/convert/convert.go`). Layout then paints into that
document. Assembly sets Title, Producer, compression, grayscale, and
creation time (`pdf_pipeline.go` `assembleDocument`).

This phase is the only convert change: construct the document with the
request's version policy. Layout, TOC, headers/footers, copies, and
links stay as they are. They already speak the paint API.

Image jobs do not touch this path.

---

## Executive Summary

| Call site | Today | Target |
|-----------|-------|--------|
| `convert.Run` document | `pdf.NewDocument()` | policy from `req.Global`; 1.4 still uses the default constructor or an explicit `PDF14` policy |
| `assembleDocument` | `SetInfo("Producer", "gowkhtmltopdf")` (writer overwrites with `Version`) | leave producer to the writer policy; do not hard-code `1.4` in convert |
| TOC scratch `pdf.NewDocument()` in `toc.go` | 1.4 scratch | same version as the job, or remain 1.4 if the scratch file is never user-visible — document the choice |
| `layout.PaintContext` | paints into `*pdf.Document` | unchanged |

---

## Phase 5 checklist

### 5.1 Document construction

- [x] Map `settings.PdfGlobal` version → `pdf.WriterPolicy` in **one** helper (convert or settings adapter). No `if global == "2.0"` in layout
  (`convert.PolicyForGlobal` `internal/convert/convert.go:229-260`; `"2.0"`
  case → `pdf.WriterPolicy{Version: pdf.PDF20}`; layout has no version
  branches)
- [x] `convert.Run` uses that helper when creating `runContext.doc` — `internal/convert/convert.go`
  (`convert.go:359-364`: `policyForGlobal(req.Global)` →
  `pdf.NewDocumentWithPolicy(policy)`)
- [x] Invalid version never reaches `render.Run` (phase 4 sentinel is enough)
  (`Request.Validate` → `PolicyForGlobal` errors on garbage with
  `settings.ErrInvalidPDFVersion` before `render.Run`; covered by
  `TestPDFVersionNegativeValidation`)
- [x] Default / empty setting → 1.4 document
  (`PolicyForGlobal` `""` → `PDF14`; `TestConvertPDFVersion` step 1 asserts
  `%PDF-1.4` for the unset path)
- [x] Test: `Run` / `RunPDF` on a tiny HTML with version 2.0 writes `%PDF-2.0`
  (`convert_test.go` `TestConvertPDFVersion` step 4 — green;
  `api_test.go` `TestPDFVersionAPI` step 3 — green)
- [x] Test: same HTML without the setting writes `%PDF-1.4`
  (`TestConvertPDFVersion` step 1; `TestPDFVersionAPI` step 4 — both green)

### 5.2 Assemble

- [x] `assembleDocument` does not fight the writer on Producer/version — `pdf_pipeline.go`
  (`pdf_pipeline.go:145` sets the bare producer; the writer policy owns the
  version suffix — 2.0 output carries `/Producer (gowkhtmltopdf 2.0)` and
  `<pdf:Producer>gowkhtmltopdf 2.0</pdf:Producer>`, asserted in
  `TestConvertPDF20GoldenNeedles`)
- [x] Title, compression, grayscale, `SetCreationTime(run.req.now())` still applied
  (`pdf_pipeline.go:133-148` unchanged and verified under 2.0 by the golden
  needle test; compression/grayscale exercised by `internal/pdf/pdf20_test.go`)
- [x] Copies / reorder / HF still operate on the same `*pdf.Document`
  (unchanged pipeline; HF under 2.0 proven by
  `TestConvertPDF20MultiPageTOCHF`)

### 5.3 Scratch documents

- [x] Inventory every `pdf.NewDocument()` outside tests (`convert.go`, `toc.go`, …)
  (production call sites: `convert.go:364` and the TOC scratch
  `toc.go:163`; both construct via `NewDocumentWithPolicy`. Benchmarks/tests
  keep `NewDocument()`)
- [x] User-visible PDFs honor the request version
  (`Run` document carries the request policy; asserted end-to-end)
- [x] Internal scratch docs are either the same policy or documented as non-emitted
  (TOC scratch uses `doc.Policy()` — `toc.go:242/261` `paintCount(ctx,
  doc.Policy(), …)`; never written to the sink)
- [x] Test: TOC + outline + links job with `--pdf-version 2.0` still opens (structural parse)
  (`golden_test.go` `TestConvertPDF20MultiPageTOCHF`: `%PDF-2.0`, >= 4
  pages, outlines, HF text, `ParseSemantic` version + page count — green)

### 5.4 Layout isolation

- [x] `internal/layout` does not import version types
  (verified by inspection; layout speaks only the paint API)
- [x] `PaintContext` is not branched on PDF version
  (no version references in `internal/layout`)
- [x] Test: `go test ./internal/layout` green (no new layout fixtures required)
  (ran 2026-08-15: `ok`, no changes to the package)

---

## Explicitly out of scope

- Page-island / benchmark path changes
- New TOC or HF features
- Golden corpus refresh beyond what phase 6 lists
- Claiming PDF/A because the header is 2.0

---

## Done when

`gowkhtmltopdf --pdf-version 2.0 in.html out.pdf` (and `RunPDF` with
the same setting) emits a 2.0 file through the normal HTML pipeline,
and the unset path is still 1.4.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 1–4 | Phase 6 end-to-end goldens |
