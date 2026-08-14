# Phase 2 — Catalog, Trailer, and Metadata

> **Parent:** `plans/0.2.2/pdf-1.7-plan/00-canonical-pdf-17-plan.md`
> **Status:** completed
> **Estimated effort:** 3–5 days
> **Depends on:** Phase 1 (policy + header)
> **Unblocks:** phases 3, 6
> **Spec:** ISO 32000-1 §7.5.5 / Table 15, §7.7.2 / Table 28, §7.9.2.2, §14.3, §14.4 — [SPEC-NOTES.md](SPEC-NOTES.md)

---

## Overview

PDF 1.7 support is more than a header. ISO 32000-1 still allows a
classic xref; this plan keeps that. The 1.7 document shell is:

1. Trailer file identifier `/ID` — optional, but §14.4 says it
   **should** be used. Table 15 NOTE 2: absence can break workflows
   that need unique files.
2. Info dictionary stays **first-class** (unlike PDF 2.0, which
   deprecates it). Values that are not PDFDocEncoding use **UTF-16BE
   with BOM `FE FF`** (§7.9.2.2). Not UTF-8.
3. Non-claiming XMP Metadata stream — preferred metadata path since
   1.4 (§14.3.1 NOTE), still absent from this writer.
4. Do **not** emit catalog `/Version` when the header is already
   `1.7` (Table 28: catalog Version is used only if **later** than
   the header). Do **not** emit `/Extensions`.

Do not add OutputIntent, ICC, `pdfaid`, `pdfuaid`, MarkInfo, or a
structure tree. Those belong to #33.

---

## Executive Summary

| Object | 1.4 today | 1.7 target |
|--------|-----------|------------|
| Trailer | `/Size /Root /Info` | same + `/ID [ <a> <b> ]` |
| `/ID` bytes | none | two deterministic hex strings; first write: both equal (§14.4) |
| Info | Title/Producer/dates, Latin-1 | still present; UTF-16BE + BOM when a value is not PDFDocEncoding |
| Metadata stream | none | `/Type /Metadata /Subtype /XML` on the catalog |
| XMP claims | n/a | `dc:format`, `pdf:Producer`, dates — **no** `pdfaid` / `pdfuaid` |
| Catalog `/Version` | none | omit |
| xref | classic table | classic table (no xref stream) |

---

## Phase 2 checklist

### 2.1 Trailer `/ID`

- [x] 1.7 trailer includes `/ID` with two byte strings — `writePDFTrailer` in `internal/pdf/pdf.go:507-526`
- [x] On first write both identifiers are equal (ISO 32000-1 §14.4) — `internal/pdf/pdf.go:511-512`
- [x] ID is deterministic: same pages, info, creation time, and version produce the same pair (goldens; spec allows non-reproducible IDs, we cannot) — `internal/pdf/pdf.go:453-488`
- [x] ID is not `math/rand` or wall-clock `time.Now` (creation time is already injectable) — `internal/pdf/pdf.go:456-465`
- [x] 1.4 trailer stays bit-compatible with today's tests (default: do not add `/ID` to 1.4 in this phase) — `internal/pdf/pdf.go:507-508`
- [x] Test: two `Write`s of the same 1.7 document are byte-identical (`internal/pdf/policy_test.go:252-278`)
- [x] Test: 1.7 trailer parse finds `/ID` with two equal entries (`internal/pdf/policy_test.go:226-250`)

### 2.2 Catalog

- [x] `catalogDict` stays the single builder — `internal/pdf/pdf.go:616-632`
- [x] 1.7 catalog still has `/Type /Catalog` and `/Pages` — `internal/pdf/pdf.go:618-620`
- [x] Outlines still finalize **before** the catalog (existing invariant) — `internal/pdf/pdf.go:565-578`
- [x] No catalog `/Version` when the header is `%PDF-1.7` — `internal/pdf/pdf.go:616-632`
- [x] No catalog `/Extensions` — `internal/pdf/pdf.go:616-632`
- [x] Test: catalog `/Root` still resolves; outlines still present when set (`internal/pdf/policy_test.go:280-332`)

### 2.3 Info dictionary (kept, first-class)

- [x] 1.7 still emits an Info dict and trailer `/Info` — `internal/pdf/pdf.go:509-510, 710-738`
- [x] Producer string uses the document version (`gowkhtmltopdf 1.7`) — `internal/pdf/pdf.go:722`
- [x] 1.4 Producer remains `gowkhtmltopdf 1.4` — `internal/pdf/policy.go:102-104`
- [x] 1.7 Info / outline title strings that need Unicode use UTF-16BE + BOM (`FE FF`); Latin-1-only values may stay PDFDocEncoding / current `pdfString` — `internal/pdf/pdf.go:1036-1076`
- [x] 1.4 `pdfString` Latin-1 fold is unchanged — `internal/pdf/pdf.go:1039-1041`
- [x] Test: `TestInfoDict` still passes on 1.4 (`internal/pdf/pdf_test.go:420-438`)
- [x] Test: a 1.7 title containing U+2014 (em dash) is UTF-16BE, not `?` and not UTF-8 (`internal/pdf/policy_test.go:372-428`)

### 2.4 Non-claiming XMP

- [x] 1.7 `finalize` allocates a Metadata stream object and sets catalog `/Metadata` — `internal/pdf/pdf.go:580-592, 629-631`
- [x] Packet is well-formed XMP (`xpacket` begin/end, `dc:format = application/pdf`, `pdf:Producer`, create/modify dates from `SetCreationTime`) — `internal/pdf/pdf.go:1107-1132`
- [x] Packet does **not** contain `pdfaid`, `pdfuaid`, or `pdfaExtension` — `internal/pdf/pdf.go:1107-1132`
- [x] 1.4 documents still have no `/Metadata` — `internal/pdf/pdf.go:580`
- [x] Test: 1.7 catalog references a `/Type /Metadata /Subtype /XML` stream (`internal/pdf/policy_test.go:430-496`)
- [x] Test: negative — `pdfaid` / `pdfuaid` absent from the file bytes (`internal/pdf/policy_test.go:498-534`)

### 2.5 Classic xref stays

- [x] No object streams, no `/Type /XRef`, no `/Filter` on the xref — `internal/pdf/pdf.go:490-534`
- [x] `TestXrefOffsets` passes for 1.7 (`internal/pdf/policy_test.go:584-678`)
- [x] Empty document still fails (`errPDFNoPages`) on both versions (`internal/pdf/policy_test.go:536-582`)

---

## Explicitly out of scope

- PDF/A-4 OutputIntent / ICC / omitting Info (#33)
- Structure tree, BDC/EMC, ParentTree (#33)
- UTF-8 text strings (#32)
- Settings / convert wiring (phases 4–5)

---

## Done when

A hand-built 1.7 document (no convert job yet) has `%PDF-1.7`, a
classic xref, trailer `/ID`, Info (UTF-16BE when needed), and a
non-claiming Metadata stream, and 1.4 fixtures are unchanged.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 policy + header | Phase 3 gates; phase 6 goldens |
