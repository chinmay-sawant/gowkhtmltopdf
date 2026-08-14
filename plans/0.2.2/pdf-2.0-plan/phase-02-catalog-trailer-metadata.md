# Phase 2 — Catalog, Trailer, and Metadata

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** completed (2026-08-15)
> **Estimated effort:** 3–5 days
> **Depends on:** Phase 1 (policy + header)
> **Unblocks:** phases 3, 6

---

## Overview

PDF 2.0 support is more than a header. ISO 32000-2 still allows a
classic xref; this plan keeps that (determinism, existing tests). The
2.0-specific document shell is:

1. Trailer file identifier `/ID`
2. Honest metadata: keep Info (deprecated, widely consumed) and add a
   **non-claiming** XMP Metadata stream
3. Optional catalog `/Version /2.0` only if we decide the header is not
   enough; do not emit a catalog version that disagrees with the header

Do not add OutputIntent, ICC, `pdfaid`, `pdfuaid`, MarkInfo, or a
structure tree. Those belong to #33.

---

## Executive Summary

| Object | 1.4 today | 2.0 target |
|--------|-----------|------------|
| Trailer | `/Size /Root /Info` | same + `/ID [ <a> <b> ]` |
| `/ID` bytes | none | two deterministic hex strings; same input + `SetCreationTime` → same ID |
| Info | Title/Producer/dates, Latin-1 | still present; string encoding follows policy (UTF-8 on 2.0) |
| Metadata stream | none | `/Type /Metadata /Subtype /XML` on the catalog |
| XMP claims | n/a | `dc:format`, `pdf:Producer`, dates — **no** `pdfaid` / `pdfuaid` |
| xref | classic table | classic table (no xref stream) |

**What #31 already provided (verified against the landed 1.7 code):**
`writePDFTrailer` already emits a deterministic `/ID` for
`Version >= PDF17` (`computeTrailerID` hashes the policy header version,
the injectable creation time, sorted Info, page sizes, and object
dicts — MD5, no `math/rand`/`time.Now`); `finalize` already allocates
the `/Metadata` stream for `Version >= PDF17` (`buildXMPMetadata` emits
the non-claiming packet with `dc:format`, `pdf:Producer`,
`xmp:CreateDate`/`ModifyDate`); `catalogDict` is already the single
builder with outlines finalized before it; and `encodeTextString` was
already version-aware. PDF 2.0 therefore inherits all of that; the only
2.0 code delta was the text-string branch (UTF-8 instead of UTF-16BE)
plus tests.

---

## Phase 2 checklist

### 2.1 Trailer `/ID`

- [x] 2.0 trailer includes `/ID` with two byte strings — `writePDFTrailer` in `pdf.go` (already from #31: gated `Version >= PDF17`, so `PDF20` inherits); verified by `TestPDF20TrailerID` (`internal/pdf/pdf20_test.go`)
- [x] ID is deterministic: same pages, info, creation time, and version produce the same pair — `computeTrailerID`; `TestPDF20TrailerID` asserts two independent writes are byte-identical
- [x] ID is not `math/rand` or `time.Now` (creation time is already injectable) — `computeTrailerID` uses `doc.creationTime` with the fixed 2000-01-01 fallback; determinism test guards it
- [x] 1.4 trailer stays bit-compatible with today's tests unless an explicit 1.4 `/ID` is approved (default: do not add `/ID` to 1.4 in this phase) — gate unchanged; `TestPDF20TrailerID` part 4 asserts no `/ID` on 1.4; existing `TestTrailerIDBehavior` / `TestDefaultNewDocumentAsserts14` keep passing
- [x] Test: two `Write`s of the same 2.0 document are byte-identical (`TestDeterministicOutput` equivalent) — `TestPDF20TrailerID` (+ deterministic double-build in `TestPDF20RichDocument`)
- [x] Test: 2.0 trailer parse finds `/ID` with two entries — `TestPDF20TrailerID` regex `<32hex> <32hex>` with equal entries; also asserts the 2.0 ID differs from the 1.7 ID of the same content (version participates)

### 2.2 Catalog

- [x] `catalogDict` stays the single builder — `pdf.go` (already from #31; unchanged builder)
- [x] 2.0 catalog still has `/Type /Catalog` and `/Pages` — `TestPDF20CatalogAndMetadataStream` regex match
- [x] Outlines still finalize **before** the catalog (existing invariant) — `finalize` calls `finalizeOutlines` before `d.setDict(catalogRef, ...)`; `TestPDF20CatalogAndMetadataStream` asserts `/Outlines N 0 R` and `/PageMode /UseOutlines`
- [x] If catalog `/Version` is emitted, it matches the header; if not emitted, document that the header is the sole version — **decided: not emitted**; code comment added on `catalogDict` ("the file header is the single version authority", matching the #31 1.7 sibling), test asserts the 2.0 catalog contains no `/Version`
- [x] Test: catalog `/Root` still resolves; outlines still present when set — `TestPDF20CatalogAndMetadataStream`, `TestPDF20XrefOffsets`

### 2.3 Info dictionary (kept)

- [x] 2.0 still emits an Info dict and trailer `/Info` (deprecated ≠ removed) — `infoDict` unconditional; `TestPDF20InfoAndOutlineUTF8` and `TestPDF20HeaderEmissionAndSemantic` assert `/Producer (gowkhtmltopdf 2.0)`
- [x] Producer string uses the document version (`gowkhtmltopdf 2.0`), not a leftover `1.4` constant — `infoDict` uses `d.policy.ProducerVersion()` (already from #31); XMP uses the same method
- [x] 1.4 Producer remains `gowkhtmltopdf 1.4` — existing `TestInfoDict`, `TestDefaultNewDocumentAsserts14`, `TestPDF17InfoAndOutlineUnicodeUTF16BE` part 2 (suite green)
- [x] 2.0 Info / outline title strings use UTF-8 when the policy says so; 1.4 `pdfString` Latin-1 fold is unchanged — `encodeTextString` now branches: `< PDF17` → `pdfString`; `PDF17` → UTF-16BE hex; `>= PDF20` → `encodeUTF8Hex` (BOM `EF BB BF` + UTF-8, ISO 32000-2); 1.4 path untouched (`TestPDFStringLatin1NotUTF8` still green)
- [x] Test: `TestInfoDict` still passes on 1.4 — suite green
- [x] Test: a 2.0 title containing U+2014 (em dash) does not become `?` or a WinAnsi fold — `TestPDF20InfoAndOutlineUTF8` asserts `/Title <EFBBBF` with `E28094` and rejects `?` / `\227` folds; outline title asserted too

### 2.4 Non-claiming XMP

- [x] 2.0 `finalize` allocates a Metadata stream object and sets catalog `/Metadata` — `finalize` + `catalogDict` (already from #31, gate `Version >= PDF17`); `TestPDF20CatalogAndMetadataStream` resolves the catalog `/Metadata N 0 R` to its object
- [x] Packet is well-formed XMP (`xpacket` begin/end, `dc:format = application/pdf`, `pdf:Producer`, create/modify dates from `SetCreationTime`) — same test asserts all of these against the pinned `SetCreationTime` (2026-08-14T15:30:00Z)
- [x] Packet does **not** contain `pdfaid`, `pdfuaid`, or `pdfaExtension` — negative loop in `TestPDF20CatalogAndMetadataStream`; claim keys are only emitted under `IsPDFA3`/`IsPDFUA1`, which require PDF 1.7
- [x] 1.4 documents still have no `/Metadata` — gate unchanged; existing `TestDefaultNewDocumentAsserts14` asserts the 1.4 catalog has no `/Metadata`
- [x] Test: 2.0 catalog references a `/Type /Metadata /Subtype /XML` stream — `TestPDF20CatalogAndMetadataStream`
- [x] Test: negative — `pdfaid` / `pdfuaid` absent from the file bytes — negative loop in `TestPDF20CatalogAndMetadataStream`

### 2.5 Classic xref stays

- [x] No object streams, no `/Type /XRef`, no `/Filter` on the xref — classic table only; `TestPDF20XrefOffsets` asserts `xref\n0 ` + free entry and rejects `/Type /XRef` / `/ObjStm`
- [x] `TestXrefOffsets` (every `n` entry points at `N 0 obj`) passes for 2.0 — `TestPDF20XrefOffsets` (multi-page, fonts, image, annots, outlines) via `parseXrefEntries`; original 1.4 `TestXrefOffsets` still passes
- [x] Empty document still fails (`errPDFNoPages`) on both versions — `TestEmptyDocFails` (1.4) and new `TestPDF20EmptyDocFails` (2.0, also asserts 0 bytes written)

---

## Explicitly out of scope

- PDF/A-4 OutputIntent / ICC / omitting Info (#33)
- Structure tree, BDC/EMC, ParentTree (#33)
- Settings / convert wiring (phases 4–5)
- Replacing `pdfString` on the 1.4 path

---

## Done when

A hand-built 2.0 document (no convert job yet) has `%PDF-2.0`, a
classic xref, trailer `/ID`, Info, and a non-claiming Metadata stream,
and 1.4 fixtures are unchanged.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 policy + header | Phase 3 gates; phase 6 goldens |
