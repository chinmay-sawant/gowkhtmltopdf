# Phase 1 — Version Policy and Header

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** completed (2026-08-15)
> **Estimated effort:** 2–4 days
> **Depends on:** [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31) / [../pdf-1.7-plan/](../pdf-1.7-plan/) `WriterPolicy` type (landed before this phase)
> **Unblocks:** phases 2–5

---

## Overview

Today the writer has a package-level `const Version = "1.4"` and
`writePDFHeader` always prints it. There is no way to build a document
that is not PDF 1.4.

This phase introduces one version policy on `Document` and makes the
header follow that policy. Default `NewDocument()` stays 1.4 so every
existing test keeps its header.

Do not invent a second writer, a `ModePDF20` flag soup, or version
checks inside `Content` methods.

---

## Executive Summary

| Surface | Today | Target |
|---------|-------|--------|
| `const Version` | Process-wide `"1.4"` used by header and Producer | Header/Producer read the document policy; a compatibility alias may remain for 1.4 tests |
| `NewDocument()` | Implicit 1.4 | Still 1.4 |
| Policy type | none | `PDFVersion` + `WriterPolicy` in `internal/pdf` |
| `writePDFHeader` | `%%PDF-%s` with `Version` | `%%PDF-%s` with `doc.policy.HeaderVersion()` |
| Unknown version | n/a | `NewDocumentWithPolicy` returns an error |

---

## Phase 1 checklist

### 1.1 Policy type

- [x] Reuse the #31 `PDFVersion` / `WriterPolicy` type — do not add a second enum (already from #31)
- [x] Implement `PDF20` on that type; policy methods own header spelling (`"2.0"`) — `PDF20` const now supported; `HeaderVersion()` returns `versionToken20`; `ErrReservedPDF20` sentinel removed
- [x] `PDF17` follows the 1.7 plan (working emit, not a reserved reject) once that plan has landed (already from #31)
- [x] Callers do not format version strings — header uses `policy.HeaderVersion()`, Producer uses `policy.ProducerVersion()` (Info dict + XMP); package-level `const Version = "1.4"` retained only as a 1.4 test alias, no emit site uses it
- [x] Test: table of versions → header strings; unknown/unwired versions error — `TestPolicyHeaderAndProducerStrings` (1.4/1.7/2.0), `TestPolicyValidation` (PDFVersion(-1)/PDFVersion(99) → `ErrUnsupportedPDFVersion`)

### 1.2 Document construction

- [x] `NewDocument()` keeps today's 1.4 behavior — `internal/pdf/pdf.go` (already from #31; verified by `TestDefaultNewDocumentAsserts14`)
- [x] `NewDocumentWithPolicy(WriterPolicy) (*Document, error)` stores the policy and rejects unsupported versions before any object is allocated — now also accepts `PDF20`; rejects unknown versions with `ErrUnsupportedPDFVersion`
- [x] `Document` does not grow a public `Version string` that callers can mutate after pages exist (already from #31; unchanged)
- [x] Test: `NewDocument` output still starts with `%PDF-1.4` and the binary comment `%\xe2\xe3\xcf\xd3` — `TestDefaultNewDocumentAsserts14` (+ new negative assertion that `%PDF-2.0` never appears in default output)
- [x] Test: `NewDocumentWithPolicy({PDF20})` output starts with `%PDF-2.0` and the same binary comment — new `TestPDF20HeaderEmissionAndSemantic`

### 1.3 Header emission

- [x] `writePDFHeader` takes the document (or the policy), not the package constant — `pdf.go` (already from #31: `writePDFHeader(out, d.policy)`)
- [x] Existing `TestWriteHeaderAndTrailer` (or equivalent) still passes for the default document
- [x] New test: 2.0 header + binary comment + xref still points at real `n` objects — `TestPDF20HeaderEmissionAndSemantic` (xref entries verified via `parseXrefEntries`)
- [x] `ParseSemantic` reports `Version == "2.0"` for a 2.0 document — `semantic.go` derives the version from the header line (`%PDF-` prefix trim), no hard-coding; verified in `TestPDF20HeaderEmissionAndSemantic`

### 1.4 Default isolation

- [x] No production caller of `NewDocument()` is changed in this phase (`internal/convert` still uses 1.4)
- [x] `go test ./internal/pdf` — existing 1.4 tests green (136 tests pass, 0 fail; `go vet ./internal/pdf` clean)
- [x] Grep/test: the string `%PDF-2.0` does not appear in default-path goldens — negative assertion added to `TestDefaultNewDocumentAsserts14`; all default-path tests assert `%PDF-1.4`

---

## Explicitly out of scope

- Catalog `/Version`, trailer `/ID`, XMP (phase 2)
- Settings / CLI / library (phase 4)
- Convert wiring (phase 5)
- UTF-8 string encoding (phase 2)
- PDF/A or tagged PDF

---

## Done when

A unit test can build a one-page PDF 2.0 whose header is `%PDF-2.0`,
and every existing `NewDocument()` test still sees `%PDF-1.4`.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Current `writePDFHeader` / `NewDocument` | Phase 2 catalog/trailer; phase 5 convert |
