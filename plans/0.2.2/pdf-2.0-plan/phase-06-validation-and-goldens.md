# Phase 6 — Validation and Goldens

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** completed (2026-08-15)
> **Estimated effort:** 3–5 days
> **Depends on:** Phases 1–5
> **Unblocks:** phases 7–8

---

## Overview

#32 requires structural fixtures, golden outputs, and an independent
parser or validator. This repo already has:

- Package tests in `internal/pdf/*_test.go` (header, xref, determinism,
  rich structure)
- `ParseSemantic` (`semantic.go`) — in-tree view of **our** files
- Convert goldens under `testdata/golden/` (mostly 1.4 envelopes)

Use those. Do not add veraPDF as a required gate here (A-4 / UA-2 is
#33). An optional external parser is welcome if it is skippable when
missing.

Do not replace the 1.4 golden corpus. Add 2.0 needles beside it.

---

## Executive Summary

| Gate | What it proves |
|------|----------------|
| Unit (writer) | Header, xref, `/ID`, Metadata, Info, fonts, images on 2.0 |
| Semantic parse | `ParseSemantic(data).Version == "2.0"` and page text/annots still extract |
| Convert golden needles | One small HTML fixture emits `%PDF-2.0`, `/ID`, `/Metadata`, and not `pdfaid` |
| 1.4 regression | Existing goldens and `TestDeterministicOutput` unchanged |
| Optional parser | If present, opens the 2.0 fixture; if absent, test skips |

---

## Phase 6 checklist

### 6.1 Writer unit tests

- [x] 2.0 header + binary comment
  (`policy_test.go` `TestPDF20HeaderEmissionAndSemantic` asserts
  `%PDF-2.0\n%\xe2\xe3\xcf\xd3\n` — landed in phase 2/3)
- [x] 2.0 xref offsets
  (`pdf20_test.go` `TestPDF20XrefOffsets` — every xref entry points at its
  `N 0 obj` header; classic table only)
- [x] 2.0 trailer `/ID` present; 1.4 trailer unchanged
  (`pdf20_test.go` `TestPDF20TrailerID` — two 32-char hex entries; 1.4
  trailer has no `/ID`)
- [x] 2.0 catalog `/Metadata` present; bytes contain no `pdfaid` / `pdfuaid`
  (`pdf20_test.go` `TestPDF20CatalogAndMetadataStream` + `TestPDF20RichDocument`)
- [x] 2.0 determinism (two writes, equal bytes)
  (`pdf20_test.go` `TestPDF20TrailerID` step 2 and `TestPDF20RichDocument`
  determinism check)
- [x] 2.0 rich document: text + image + link + outline
  (`pdf20_test.go` `TestPDF20RichDocument` — JPEG+PNG, URI + Dest links,
  outline hierarchy)
- [x] Short-writer contract still fails closed on 2.0
  (`pdf20_test.go` `TestPDF20ShortWriterContract` — `Write`/`WriteTo` on a
  short writer surface `io.ErrShortWrite`; added this phase)
- [x] Default `NewDocument` tests still assert `%PDF-1.4`
  (`policy_test.go` `TestDefaultNewDocumentAsserts14` — header, no
  `/Metadata`, no trailer `/ID`)

### 6.2 Semantic parser

- [x] `ParseSemantic` accepts a 2.0 file this package emits — `semantic.go` / `semantic_*_test.go`
  (`pdf20_test.go` `TestPDF20RichDocument` and
  `semantic_converted_test.go` `TestSemanticPDF20OracleConvertedFixtures` —
  both green)
- [x] `SemanticDoc.Version` is `"2.0"`
  (asserted in both tests above; `ParseSemantic` reads the header token)
- [x] Page text, images, and annots still populate
  (`TestSemanticPDF20OracleConvertedFixtures`: fixture-01 text needles,
  fixture-06 URI annots, fixture-07 image XObject, fixture-24 internal
  destinations — all under a 2.0 conversion)
- [x] A 1.4 file still reports `"1.4"`
  (`semantic_converted_test.go` `TestSemanticPDFOracleConvertedFixtures`
  asserts `doc.Version == "1.4"`; `policy_test.go:972` asserts the same —
  both green)

### 6.3 Convert / golden needles

- [x] Add one small committed HTML fixture (or reuse a tiny existing one) converted with version 2.0
  (reused `testdata/golden/fixture-01-simple-invoice.html` — no new fixture,
  no corpus regeneration)
- [x] Needle assertions: `%PDF-2.0`, trailer `/ID`, `/Metadata`, Producer contains `2.0`, absence of `pdfaid`
  (`golden_test.go` `TestConvertPDF20GoldenNeedles` — header + binary
  comment, `/ID` regex, `/Type /Metadata /Subtype /XML`,
  `/Producer (gowkhtmltopdf 2.0)`, and no `pdfaid`/`pdfuaid`)
- [x] Same fixture without the setting still matches the 1.4 envelope
  (`TestConvertPDF20GoldenNeedles` step 2 asserts `%PDF-1.4` and no
  `%PDF-2.0` byte; the existing 1.4 goldens keep proving the rest)
- [x] Do not pixel-diff PDFs as the default gate
  (no pixel comparison added anywhere)
- [x] TOC + HF job covered by a structural test (page count + header version), not a new visual corpus
  (`golden_test.go` `TestConvertPDF20MultiPageTOCHF` — `%PDF-2.0`, >= 4
  pages, outlines, HF text, `ParseSemantic` version + page count)

### 6.4 Negative tests

- [x] Unsupported version string never produces a file that claims 2.0
  (`convert_test.go` `TestPDFVersionNegativeValidation` — garbage versions
  error with `ErrInvalidPDFVersion` and 0 bytes written)
- [x] 2.0 + a not-implemented combination (if any API exists) errors before `Write`
  (policy matrix now carries PDF20 rows: encryption/forms/signatures/
  object-streams/A-4/UA-2 all fail `NewDocumentWithPolicy` +
  `policy.Validate()` with their sentinels — `TestPDFVersionNegativeValidation`)
- [x] Image mode has no version claim
  (`TestPDFVersionNegativeValidation` `image_mode_no_version_claim` asserts
  PNG output with no `%PDF-` bytes; `cli_test.go` rejects `--pdf-version`
  for image mode)

### 6.5 Optional independent parser

- [x] Document one optional command (e.g. `qpdf --check` or `mutool info`) in the phase notes or a test helper
  (`golden_test.go` `TestOptionalPDFValidation` — `qpdf --check` and
  `mutool info`/`mutool clean`; now also validates a 2.0 conversion)
- [x] Test skips when the binary is missing; does not fail CI
  (`exec.LookPath` both binaries; `t.Skip` when neither is installed —
  observed skip in this environment)
- [x] Do **not** add veraPDF flavour `4` / `ua2` as a #32 gate
  (no veraPDF dependency; compliance profiles remain #33)

---

## Explicitly out of scope

- Refreshing every `testdata/golden` PDF
- `make samples` regeneration as a required gate (manual viewer check is enough)
- Compliance profiles

---

## Done when

A 2.0 HTML conversion is proven by unit + semantic + needle tests, and
the 1.4 corpus is still green.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 1–5 | Phase 7 documentation claims |
