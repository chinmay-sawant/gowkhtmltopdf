# Phase 05 — Print Pagination & Multi-Object Assembly

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** not started  
> **Estimated effort:** 2–4 months solo  
> **Depends on:** Phase 4 layout, Phase 3 PDF  
> **Unblocks:** Phase 6 (page numbers, outlines), production multi-page PDFs

---

## Overview

Fragment laid-out content into print pages (the role of `QWebPrinter` in upstream), then assemble multiple input objects (cover/pages) into one PDF with copies/collate.

## Executive Summary

wkhtmltopdf multi-doc “merge” is one continuous print session, not PDF concatenation (`pdfconverter.cc` printDocument). Match that model.

---

## Checklist

### 5.1 Page geometry
- [ ] Compute content box from page size + margins (UnitReal → points)
- [ ] Orientation swap width/height
- [ ] Custom page width/height when both set
- [ ] DPI/zoom: define CSS px ↔ pt mapping (document: 96 CSS px/inch default)

### 5.2 Fragmentation
- [ ] Split block flow into page-sized fragments
- [ ] Honor `page-break-before: always|avoid`, `page-break-after`, `page-break-inside: avoid` on allowlisted elements
- [ ] Table row avoid-break best-effort
- [ ] Do not split DrawText mid-glyph (line-level breaks)
- [ ] Track element → (pageIndex, rect) map for links/headings (`elementLocation` parity structure)

### 5.3 Print media
- [ ] Apply `@media print` styles when `printMediaType` true
- [ ] Zoom factor scales layout
- [ ] Smart-shrinking: implement simple scale-to-width **or** mark unsupported with warning

### 5.4 Multi-object print (`internal/convert`)
- [ ] Phase machine mirroring: load → count pages → (TOC later) → links → HF → print
- [ ] Ordered objects: each contributes pages to one Document
- [ ] `includeInOutline` / `pagesCount` flags
- [ ] Cover: no HF (enforced in settings already)
- [ ] Copies loop + collate vs non-collate (`printDocument` / `spoolTo` logic)
- [ ] Grayscale paint mode
- [ ] Compression flag at write time

### 5.5 Output
- [ ] Write to path, stdout `-`, or memory buffer
- [ ] Progress phases + percentages for CLI

### 5.6 Tests
- [ ] 3-page forced breaks fixture
- [ ] Multi-object two HTML files → single PDF page count
- [ ] copies=2 collate vs not (page order assertions)
- [ ] Margin/page-size golden geometry

### 5.7 Closure
- [ ] `make test` / `make lint`
- [ ] End-to-end: `gowkhtmltopdf page.html out.pdf` works for corpus

---

## Dependencies

| Upstream behavior | File |
|-------------------|------|
| printDocument / spoolPage | `pdfconverter.cc` |
| createPrinter margins | `pdfconverter.cc` pagesLoaded |
| QWebPrinter role | replaced by this phase |

## Risks

- Table header repeat across pages: upstream needed Qt patches; MVP may omit (`[~]`).
