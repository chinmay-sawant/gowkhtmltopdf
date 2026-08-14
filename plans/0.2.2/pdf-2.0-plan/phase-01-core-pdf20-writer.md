# Phase 1 — Core PDF 2.0 Writer

**Status:** ✅ COMPLETED — all items implemented and tested  
**Depends on:** None  
**Base plan refs:** §4 (`doc`, `write`, `page`), §5 Phase 0–1/5–6 (minimal), §6.1–6.3, §6.6–6.7, §11 Phase A

---

## Goal

Ship a minimal **untagged** PDF 2.0 file: valid header, catalog, pages tree, one page, simple content stream, one font reference, classic xref, trailer with `/ID`.

---

## Package layout (scaffold this phase)

- [x] `engine/doc/` — object ID allocator, document builder, final assembly
- [x] `engine/write/` — dict/stream/xref/trailer byte encoding
- [x] `engine/page/` — pages tree + single page object
- [x] `engine/content/` — minimal text operators only
- [x] `go.mod` + package smoke test

---

## Checklist — document shell

### Header
- [x] Emit `%PDF-2.0`
- [x] Emit binary comment line (bytes ≥ 128)

### Catalog (`/Type /Catalog`)
- [x] `/Type /Catalog`
- [x] `/Pages` → pages root

### Pages tree
- [x] `/Type /Pages`
- [x] `/Kids [ <page> 0 R ]`
- [x] `/Count 1`

### Page
- [x] `/Type /Page`
- [x] `/Parent` pages root
- [x] `/MediaBox [0 0 width height]` (e.g. A4)
- [x] `/Contents` content stream ref
- [x] `/Resources /Font` with at least one font name

### Content stream
- [x] Stream object with `/Length`
- [x] Simple text operators (e.g. `BT` … `ET`) drawing one line
- [x] Optional: `/Filter /FlateDecode` (can defer to later)

### Font (placeholder only)
- [x] Simple Type1 standard font dict is OK for this phase (e.g. Helvetica)
- [x] Do **not** require embedding yet (phase 3)

### XRef + trailer
- [x] Classic `xref` with correct byte offsets
- [x] `trailer` with `/Size`, `/Root`, `/ID [ <file-id> <file-id> ]`
- [x] Optional non-A `/Info` with `/CreationDate` / `/ModDate` (allowed until phase 4)
- [x] `startxref` + `%%EOF`

### Mode hooks (stubs only)
- [x] Document mode flags exist: `ModePDF20` (always on), stubs for `ModePDFA4`, `ModePDFUA2`, `ModeEmbedFonts`
- [x] No A-4 / UA-2 objects yet

---

## Acceptance criteria

- [x] Generated file opens in at least one PDF viewer
- [x] Header starts with `%PDF-2.0`
- [x] Unit test: every object ID in xref resolves to a real `n` offset
- [x] Unit test: trailer `/Root` points at catalog
- [x] Unit test: `/ID` present with two entries

---

## Explicitly out of scope

- [ ] Tables, multi-page, images
- [ ] Font embedding / subsetting
- [ ] XMP, OutputIntent, ICC
- [ ] Structure tree / BDC-EMC
- [ ] veraPDF (not required until phase 4)

---

## Done when

One API (e.g. `GenerateMinimalPDF`) returns `[]byte` for a single-page “Hello” PDF 2.0 with valid xref.
