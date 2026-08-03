# Phase 05 — Print Pagination & Multi-Object Assembly

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** complete (2026-08-03)  
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
- [x] Compute content box from page size + margins (UnitReal → points) — `convert.pageGeometry` (`internal/convert/convert.go:265`), margins mm→pt via `mmToPt` (`convert.go:22,192-195`); A4 − 10 mm golden geometry asserted in `golden_test.go:63-66,147-155`
- [x] Orientation swap width/height — landscape swap in `pageGeometry` (`convert.go:278-280`)
- [x] Custom page width/height when both set — explicit `size.width/height` win over named size (`convert.go:270-272`)
- [x] DPI/zoom: define CSS px ↔ pt mapping (document: 96 CSS px/inch default) — mapping defined as 1 px = 0.75 pt (`layout/style.go` `pxToPt`; documented in `documentation/compatibility-matrix.md` §3); zoom application tracked under 5.3

### 5.2 Fragmentation
- [x] Split block flow into page-sized fragments — `paginateOps` (`paint.go:115`); rect-type ops crossing a boundary split by `Paint` (`paint.go:50-62`); test `TestBoundaryFillSplit` (`layout_test.go:849`)
- [x] Honor `page-break-before: always|avoid`, `page-break-after`, `page-break-inside: avoid` on allowlisted elements — `beforeAlways` (`paint.go:203`), `afterBreaks` (`paint.go:236`), `avoidInside` (`paint.go:179`); tests `TestPageBreakBeforeAlways` (`layout_test.go:809`), `TestPageBreakInsideAvoid` (`layout_test.go:828`), parsing `TestPageBreakParsing` (`layout_test.go:753`)
- [x] Table row avoid-break best-effort — `rowsIntact` (`paint.go:290`); rows never split; test `TestTableRowNoSplit` (`layout_test.go:893`)
- [x] Do not split DrawText mid-glyph (line-level breaks) — text/bullet ops snap wholly to the next page (`paint.go:118-129`); only rect-type ops are splittable (`isSplittable` `paint.go:107-109`)
- [x] Track element → (pageIndex, rect) map for links/headings (`elementLocation` parity structure) — `populateLocations` (`paint.go:341`), `Result.Locations` (`layout.go:52`), `ElementLocation` (`layout.go:58`); test `TestElementLocations` (`layout_test.go:180`)

### 5.3 Print media
- [x] Apply `@media print` styles when `printMediaType` true — convert always passes `Media: "print"` (`convert.go:210`); stylesheet media filter `linkStylesheet` (`convert.go:341-350`); test `TestRunPDFScreenOnlyStylesheetExcluded` (`convert_test.go:132`)
- [~] Zoom factor scales layout — `layout.Options.Zoom` + `zoomScale` implemented (`layout.go:33,114`; test `TestZoom` `layout_test.go:726`), CLI `--zoom` accepted (`cli/flags.go:225`, `cli_test.go:247`), but `renderObject` does not forward `ZoomFactor` into `layout.Layout` yet → PDF-path zoom is a no-op until wired
- [~] Smart-shrinking: implement simple scale-to-width **or** mark unsupported with warning — chose the warning path: over-width detection via op extents (`measuredWidth` `convert.go:250-261`) + warning (`convert.go:218-229`); scale-to-width re-layout with `Options.Zoom` still TODO (`convert.go:221-225`); `TestRunPDFSmartShrinking` (`convert_test.go:419`) asserts the warning-only behavior

### 5.4 Multi-object print (`internal/convert`)
- [x] Phase machine mirroring: load → count pages → (TOC later) → links → HF → print — `RunPDFContext` reports Loading pages / Counting pages / Resolving links / Printing pages / Done (`convert.go:62-73`); `TestRunPDFProgress` (`convert_test.go:349`)
- [x] Ordered objects: each contributes pages to one Document — per-object `renderObject` loop appends pages, ranges recorded (`convert.go:58-68`); `TestRunPDFThreeObjects` (`convert_test.go:333`), `TestRunPDFMultiPage` (`convert_test.go:86`)
- [ ] `includeInOutline` / `pagesCount` flags — deferred; not referenced in `renderObject` (`convert.go:166-242`); `pageOffset` likewise not implemented
- [x] Cover: no HF (enforced in settings already) — `IsCover` set by CLI (`cli/cli.go:148`); no HF drawing exists in `renderObject`
- [x] Copies loop + collate vs non-collate (`printDocument` / `spoolTo` logic) — `materializeCopies` (`convert.go:131`) via `Document.DuplicatePage` (`pdf.go:170`); non-collate = `ReorderPages` permutation (`convert.go:149-163`, `pdf.go:140`); tests `TestRunPDFCopiesCollate` / `TestRunPDFCopiesNonCollate` (`convert_test.go:289,311`)
- [x] Grayscale paint mode — `doc.SetGrayscale(cmd.Global.Grayscale)` (`convert.go:80`; `pdf.go:58`)
- [x] Compression flag at write time — `doc.SetCompression(cmd.Global.UseCompression)` (`convert.go:79`; `pdf.go:55`)

### 5.5 Output
- [x] Write to path, stdout `-`, or memory buffer — path / `-` → stdout dispatch (`convert.go:95-109`); tests `TestRunPDFOutputStdout` (`convert_test.go:142`), `TestWriteToMemoryBuffer` (`pdf_test.go:291`); image-mode memory path is phase 7
- [x] Progress phases + percentages for CLI — `RunPDFContext` callback + log lines, 0-100 `percent` (`convert.go:35-73,113-118`); `TestRunPDFProgress` (`convert_test.go:349`), quiet gating `TestRunPDFQuiet` (`convert_test.go:384`), cancel `TestRunPDFContextCancel` (`convert_test.go:396`)

### 5.6 Tests
- [x] 3-page forced breaks fixture — `TestPageBreakBeforeAlways` (`layout_test.go:809`) + golden multi-page invoice fixture-03 (min 2 pages, `golden_test.go:31`, `TestGoldenCorpus` `golden_test.go:102`)
- [x] Multi-object two HTML files → single PDF page count — `TestRunPDFThreeObjects` (`convert_test.go:333`), `TestRunPDFMultiPage` (`convert_test.go:86`)
- [x] copies=2 collate vs not (page order assertions) — `TestRunPDFCopiesCollate` (`convert_test.go:289`, order A B A B), `TestRunPDFCopiesNonCollate` (`convert_test.go:311`, order A A B B)
- [x] Margin/page-size golden geometry — A4 + 10 mm margins exercised in golden corpus (`golden_test.go:63-66,147-155`), `TestRunPDFSinglePageA4` (`convert_test.go:72`)

### 5.7 Closure
- [x] `make test` / `make lint` — `go test ./...` all packages ok, `go vet ./...` exit 0, `gofmt -l .` empty (verified 2026-08-03; Makefile targets `Makefile:5-10`)
- [x] End-to-end: `gowkhtmltopdf page.html out.pdf` works for corpus — `TestGoldenCorpus` runs all golden fixtures through the full `RunPDF` pipeline (`golden_test.go:102`)

---

## Design notes (filled 2026-08-03)

1. **Whole-op snap + canvas-Y shift fragmentation model.** Every op is assigned a page by its top edge; text, images and links that would straddle a boundary snap wholly to the next page (text is already line-level, so glyphs never split). Rect-type ops (fill/stroke/line) crossing a boundary are split in place by `Paint` into two ops (`paint.go:50-62,107-150`).
2. **Break policies are flow shifts, not page assignments.** `page-break-before/after: always`, `page-break-inside: avoid` and table-row no-split are implemented as canvas-Y displacements (`shiftFlowY` `paint.go:156`), iterated to a fixpoint over the element box tree; final pages are derived from the shifted Y positions (`paginateOps` `paint.go:115-151`).
3. **Element-location map feeds phase 6.** `populateLocations` fills `Result.Locations` with (element box, page, x, y, w, h) in document order (`paint.go:341-360`, `layout.go:52-58`); phase 6 outline/TOC/links consume this instead of re-walking layout.
4. **Copies/collate are a finalize-time page permutation.** `materializeCopies` duplicates page objects via `Document.DuplicatePage` (no content-stream copying) and collation order is produced by a `/Kids` permutation through `Document.ReorderPages` (`convert.go:131-163`, `pdf.go:140,170`).

## Dependencies

| Upstream behavior | File |
|-------------------|------|
| printDocument / spoolPage | `pdfconverter.cc` |
| createPrinter margins | `pdfconverter.cc` pagesLoaded |
| QWebPrinter role | replaced by this phase |

## Risks

- Table header repeat across pages: upstream needed Qt patches; MVP may omit (`[~]`).
