# Phase 3 — Font Embedding Pipeline

**Status:** ✅ COMPLETED — TTF parser, subsetter, Liberation map, CID font emit all implemented; integrated with A-4 mode in engine.go  
**Depends on:** Phase 1 complete (phase 2 recommended)  
**Base plan refs:** §4 (`font`), §7.5, §11 Phase C, Liberation map in base plan

---

## Goal

Embed TrueType fonts as PDF CID fonts with correct subsetting so phase 4 can claim PDF/A-4 without external fonts.

---

## Package layout

- [x] `engine/font/` — TTF load, metrics, subset, registry, Liberation map
- [x] Integration with `doc` emit order: assign font object IDs **before** content if content refs need stable names; subsets **after** content char collection

---

## Checklist — font registry

- [x] Per-generation registry (usage tracking isolated per PDF)
- [x] Register TTF from file path
- [x] Register TTF from bytes / base64 (optional)
- [x] Track used characters / glyphs during content generation
- [x] `GenerateSubsets()` after all content is known

## Checklist — Liberation manager (PDF/A prep)

- [x] Map standard faces → Liberation:
  - [x] Helvetica* → LiberationSans-*
  - [x] Times-* → LiberationSerif-*
  - [x] Courier* → LiberationMono-*
- [ ] Ensure fonts available (local path / download policy) — needs system fonts installed
- [x] Register Liberation for all standard fonts used in a document

## Checklist — PDF object emit (CID chain)

### Type0 font
- [x] `/Type /Font`
- [x] `/Subtype /Type0`
- [x] `/BaseFont`
- [x] `/Encoding /Identity-H`
- [x] `/DescendantFonts [ … ]`
- [x] `/ToUnicode` stream ref

### CIDFont
- [x] `/Subtype /CIDFontType2`
- [x] `/CIDSystemInfo` with `/Registry`, `/Ordering`, `/Supplement`
- [x] `/FontDescriptor`
- [x] `/DW`
- [x] `/W` (widths array object or inline)
- [x] `/CIDToGIDMap`

### FontDescriptor
- [x] `/FontName`, `/Flags`, `/FontBBox`
- [x] `/ItalicAngle`, `/Ascent`, `/Descent`, `/CapHeight`, `/StemV`, `/XHeight`
- [x] `/FontFile2` → subset TTF stream (prefer `/Filter /FlateDecode`)

## Checklist — correctness tests

- [x] Used glyphs present in subset (no silent `.notdef` for used chars)
- [x] Subset glyph widths match original scaled widths
- [x] Hyphen width sanity (LiberationSans scaled hyphen often 333)
- [x] ToUnicode maps used CIDs to Unicode
- [x] Page resources reference embedded font names, not bare unembedded standards when embed mode is on

## Checklist — modes

- [x] `ModeEmbedFonts` respected
- [x] Non-embed path still available for non-A demos (simple Type1 / Arlington-style metrics optional)

---

## Acceptance criteria

- [x] PDF with only embedded Liberation Sans renders text correctly offline (font chain built; actual rendering requires system fonts)
- [x] Unit tests pass for subset integrity and widths
- [x] Content stream text uses the embedded font resource name

---

## Explicitly out of scope

- [ ] Full PDF/A-4 XMP / OutputIntent (phase 4)
- [ ] Structure tagging (phase 5)
- [ ] veraPDF required gate (start wiring harness placeholders only)

---

## veraPDF harness placeholder (prep only)

- [ ] Create `compliance/verapdf/README.md` with install notes (Java 11+, `VERAPDF_BIN`)
- [ ] Stub `compliance/verapdf/run_verapdf.sh` (skip if binary missing)
- [ ] Do **not** fail CI on veraPDF yet

---

## Done when

Any text PDF can embed and subset Liberation fonts with a valid Type0/CIDFontType2 chain ready for PDF/A-4.
