# Phase 5 — Convert Pipeline Wiring

> **Parent:** `plans/0.2.2/pdf-1.7-plan/00-canonical-pdf-17-plan.md`
> **Status:** not started
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
links stay as they are.

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

- [ ] Map `settings.PdfGlobal` version → `pdf.WriterPolicy` in **one** helper (convert or settings adapter). No `if global == "1.7"` in layout
- [ ] `convert.Run` uses that helper when creating `runContext.doc` — `internal/convert/convert.go`
- [ ] Invalid version never reaches `render.Run` (phase 4 sentinel is enough)
- [ ] Default / empty setting → 1.4 document
- [ ] Test: `Run` / `RunPDF` on a tiny HTML with version 1.7 writes `%PDF-1.7`
- [ ] Test: same HTML without the setting writes `%PDF-1.4`

### 5.2 Assemble

- [ ] `assembleDocument` does not fight the writer on Producer/version — `pdf_pipeline.go`
- [ ] Title, compression, grayscale, `SetCreationTime(run.req.now())` still applied
- [ ] Copies / reorder / HF still operate on the same `*pdf.Document`

### 5.3 Scratch documents

- [ ] Inventory every `pdf.NewDocument()` outside tests (`convert.go`, `toc.go`, …)
- [ ] User-visible PDFs honor the request version
- [ ] Internal scratch docs are either the same policy or documented as non-emitted
- [ ] Test: TOC + outline + links job with `--pdf-version 1.7` still opens (structural parse)

### 5.4 Layout isolation

- [ ] `internal/layout` does not import version types
- [ ] `PaintContext` is not branched on PDF version
- [ ] Test: `go test ./internal/layout` green (no new layout fixtures required)

---

## Explicitly out of scope

- Page-island / benchmark path changes
- New TOC or HF features
- Golden corpus refresh beyond what phase 6 lists
- Claiming PDF/A or PDF 2.0 because the header is 1.7

---

## Done when

`gowkhtmltopdf --pdf-version 1.7 in.html out.pdf` (and `RunPDF` with
the same setting) emits a 1.7 file through the normal HTML pipeline,
and the unset path is still 1.4.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 1–4 | Phase 6 end-to-end goldens |
