# Phase 2 — Catalog, Trailer, and Metadata

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** not started
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

---

## Phase 2 checklist

### 2.1 Trailer `/ID`

- [ ] 2.0 trailer includes `/ID` with two byte strings — `writePDFTrailer` in `pdf.go`
- [ ] ID is deterministic: same pages, info, creation time, and version produce the same pair
- [ ] ID is not `math/rand` or `time.Now` (creation time is already injectable)
- [ ] 1.4 trailer stays bit-compatible with today's tests unless an explicit 1.4 `/ID` is approved (default: do not add `/ID` to 1.4 in this phase)
- [ ] Test: two `Write`s of the same 2.0 document are byte-identical (`TestDeterministicOutput` equivalent)
- [ ] Test: 2.0 trailer parse finds `/ID` with two entries

### 2.2 Catalog

- [ ] `catalogDict` stays the single builder — `pdf.go`
- [ ] 2.0 catalog still has `/Type /Catalog` and `/Pages`
- [ ] Outlines still finalize **before** the catalog (existing invariant)
- [ ] If catalog `/Version` is emitted, it matches the header; if not emitted, document that the header is the sole version
- [ ] Test: catalog `/Root` still resolves; outlines still present when set

### 2.3 Info dictionary (kept)

- [ ] 2.0 still emits an Info dict and trailer `/Info` (deprecated ≠ removed)
- [ ] Producer string uses the document version (`gowkhtmltopdf 2.0`), not a leftover `1.4` constant — `infoDict`
- [ ] 1.4 Producer remains `gowkhtmltopdf 1.4`
- [ ] 2.0 Info / outline title strings use UTF-8 when the policy says so; 1.4 `pdfString` Latin-1 fold is unchanged
- [ ] Test: `TestInfoDict` still passes on 1.4
- [ ] Test: a 2.0 title containing U+2014 (em dash) does not become `?` or a WinAnsi fold

### 2.4 Non-claiming XMP

- [ ] 2.0 `finalize` allocates a Metadata stream object and sets catalog `/Metadata`
- [ ] Packet is well-formed XMP (`xpacket` begin/end, `dc:format = application/pdf`, `pdf:Producer`, create/modify dates from `SetCreationTime`)
- [ ] Packet does **not** contain `pdfaid`, `pdfuaid`, or `pdfaExtension`
- [ ] 1.4 documents still have no `/Metadata`
- [ ] Test: 2.0 catalog references a `/Type /Metadata /Subtype /XML` stream
- [ ] Test: negative — `pdfaid` / `pdfuaid` absent from the file bytes

### 2.5 Classic xref stays

- [ ] No object streams, no `/Type /XRef`, no `/Filter` on the xref
- [ ] `TestXrefOffsets` (every `n` entry points at `N 0 obj`) passes for 2.0
- [ ] Empty document still fails (`errPDFNoPages`) on both versions

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
