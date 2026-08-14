# Phase 1 — Version Policy and Header

> **Parent:** `plans/0.2.2/pdf-1.7-plan/00-canonical-pdf-17-plan.md`
> **Status:** not started
> **Estimated effort:** 2–4 days
> **Depends on:** nothing in this ledger; this phase **owns** `WriterPolicy` for [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32) as well
> **Unblocks:** phases 2–5; 2.0 plan phase 1
> **Spec:** ISO 32000-1 §7.5.2 (header), §6 (version designations) — [SPEC-NOTES.md](SPEC-NOTES.md)

---

## Overview

Today the writer has a package-level `const Version = "1.4"` and
`writePDFHeader` always prints it. There is no way to build a document
that is not PDF 1.4.

This phase introduces one version policy on `Document` and makes the
header follow that policy. Default `NewDocument()` stays 1.4 so every
existing test keeps its header.

ISO 32000-1 §7.5.2: first line is `%PDF-` + `1.N` (N = 0–7). Readers
must accept `%PDF-1.7`. A binary comment of ≥4 bytes ≥128 follows when
the file has binary data (we already emit this).

Do not invent a second writer or version checks inside `Content`.

---

## Executive Summary

| Surface | Today | Target |
|---------|-------|--------|
| `const Version` | Process-wide `"1.4"` used by header and Producer | Header/Producer read the document policy; a compatibility alias may remain for 1.4 tests |
| `NewDocument()` | Implicit 1.4 | Still 1.4 |
| Policy type | none | `PDFVersion` + `WriterPolicy` in `internal/pdf` |
| `writePDFHeader` | `%%PDF-%s` with `Version` | `%%PDF-%s` with `doc.policy.HeaderVersion()` |
| `PDF20` | n/a | Reserved; `NewDocumentWithPolicy` errors until #32 |
| Unknown version | n/a | constructor returns an error |

---

## Phase 1 checklist

### 1.1 Policy type

- [ ] Add `PDFVersion` with `PDF14`, `PDF17`, and reserved `PDF20` in `internal/pdf` (new file or `pdf.go` — one home)
- [ ] Add `WriterPolicy` with a `Version` field; zero value is `PDF14`
- [ ] Policy methods own header spelling (`"1.4"` / `"1.7"`). Callers do not format version strings
- [ ] `PDF20` is rejected with a clear error that names issue #32 — do not silently emit `%PDF-2.0`
- [ ] Test: table of versions → header strings; unknown/unwired versions error

### 1.2 Document construction

- [ ] `NewDocument()` keeps today's 1.4 behavior — `internal/pdf/pdf.go`
- [ ] `NewDocumentWithPolicy(WriterPolicy) (*Document, error)` stores the policy and rejects unsupported versions before any object is allocated
- [ ] `Document` does not grow a public `Version string` that callers can mutate after pages exist
- [ ] Test: `NewDocument` output still starts with `%PDF-1.4` and the binary comment `%\xe2\xe3\xcf\xd3`
- [ ] Test: `NewDocumentWithPolicy({PDF17})` output starts with `%PDF-1.7` and the same binary comment

### 1.3 Header emission

- [ ] `writePDFHeader` takes the document (or the policy), not the package constant — `pdf.go`
- [ ] Existing `TestWriteHeaderAndTrailer` (or equivalent) still passes for the default document
- [ ] New test: 1.7 header + binary comment + xref still points at real `n` objects
- [ ] `ParseSemantic` reports `Version == "1.7"` for a 1.7 document — `semantic.go` already has `SemanticDoc.Version`

### 1.4 Default isolation

- [ ] No production caller of `NewDocument()` is changed in this phase (`internal/convert` still uses 1.4)
- [ ] `go test ./internal/pdf` — existing 1.4 tests green
- [ ] Grep/test: the string `%PDF-1.7` does not appear in default-path goldens

---

## Explicitly out of scope

- Catalog `/Version`, trailer `/ID`, XMP, UTF-16BE (phase 2)
- Settings / CLI / library (phase 4)
- Convert wiring (phase 5)
- PDF 2.0 header (#32)

---

## Done when

A unit test can build a one-page PDF 1.7 whose header is `%PDF-1.7`,
and every existing `NewDocument()` test still sees `%PDF-1.4`.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Current `writePDFHeader` / `NewDocument` | Phase 2 catalog/trailer; phase 5 convert; #32 policy type |
