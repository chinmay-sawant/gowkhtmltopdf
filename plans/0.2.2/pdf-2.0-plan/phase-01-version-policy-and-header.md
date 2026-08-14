# Phase 1 — Version Policy and Header

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** not started
> **Estimated effort:** 2–4 days
> **Depends on:** [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31) / [../pdf-1.7-plan/](../pdf-1.7-plan/) `WriterPolicy` type (or land that type first, then add `PDF20` here)
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

- [ ] Reuse the #31 `PDFVersion` / `WriterPolicy` type — do not add a second enum
- [ ] Implement `PDF20` on that type; policy methods own header spelling (`"2.0"`)
- [ ] `PDF17` follows the 1.7 plan (working emit, not a reserved reject) once that plan has landed
- [ ] Callers do not format version strings
- [ ] Test: table of versions → header strings; unknown/unwired versions error

### 1.2 Document construction

- [ ] `NewDocument()` keeps today's 1.4 behavior — `internal/pdf/pdf.go`
- [ ] `NewDocumentWithPolicy(WriterPolicy) (*Document, error)` stores the policy and rejects unsupported versions before any object is allocated
- [ ] `Document` does not grow a public `Version string` that callers can mutate after pages exist
- [ ] Test: `NewDocument` output still starts with `%PDF-1.4` and the binary comment `%\xe2\xe3\xcf\xd3`
- [ ] Test: `NewDocumentWithPolicy({PDF20})` output starts with `%PDF-2.0` and the same binary comment

### 1.3 Header emission

- [ ] `writePDFHeader` takes the document (or the policy), not the package constant — `pdf.go`
- [ ] Existing `TestWriteHeaderAndTrailer` (or equivalent) still passes for the default document
- [ ] New test: 2.0 header + binary comment + xref still points at real `n` objects
- [ ] `ParseSemantic` reports `Version == "2.0"` for a 2.0 document — `semantic.go` already has `SemanticDoc.Version`

### 1.4 Default isolation

- [ ] No production caller of `NewDocument()` is changed in this phase (`internal/convert` still uses 1.4)
- [ ] `go test ./internal/pdf` — existing 1.4 tests green
- [ ] Grep/test: the string `%PDF-2.0` does not appear in default-path goldens

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
