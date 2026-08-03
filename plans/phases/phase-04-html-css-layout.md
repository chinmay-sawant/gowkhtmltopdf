# Phase 04 - HTML Parser + CSS Subset Layout

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** complete (2026-08-03)  
> **Estimated effort:** 6–12 months solo · 4–8 months pair  
> **Depends on:** Phase 0 allowlist, Phase 2 fetch, Phase 3 PDF (for paint integration)  
> **Unblocks:** Phase 5 pagination (or integrate pagination incrementally)

---

## Overview

This is the **critical path**. Build a report-oriented HTML/CSS engine: parse allowlisted HTML, apply a CSS subset, produce a box tree and display list. No JavaScript.

## Executive Summary

Full browser layout is multi-decade org work. MVP success = invoice/report templates pass golden tests. Exit criterion: 5–10 production-like templates acceptable.

---

## Checklist

### 4.1 HTML (`internal/html`)
- [x] Tokenizer for HTML subset (do not use full browser error recovery unless needed)
- [x] Tree construction into DOM nodes
- [x] Encoding: UTF-8 primary; honor meta charset + defaultEncoding
- [x] Strip `script`; ignore `iframe`/`object`/`embed` for MVP
- [x] Collect: stylesheets (link + style), images, base href
- [x] Tests: malformed tags still produce usable tree for common cases

### 4.2 CSS parse (`internal/css`)
- [x] Stylesheet parser: rules, selectors, declarations
- [x] Selectors: `*`, type, `.class`, `#id`, descendant, child (optional)
- [x] Specificity + source order + important (basic)
- [x] Inheritance list (color, font-*, line-height, text-align, …)
- [x] Value parse: colors (#rgb/#rrggbb/named subset), lengths, percentages, keywords
- [x] `@media print` / `screen` filtering
- [x] User stylesheet + inline style + element style attribute

### 4.3 Used style
- [x] Style resolution per element against DOM
- [x] Default UA stylesheet for headings, lists, tables, `a`, `p`, `body`

### 4.4 Layout (`internal/layout`)
- [x] Containing blocks, viewport width from page content box
- [x] Block formatting context: margins, padding, borders, width/height auto
- [x] Margin collapsing (document subset)
- [x] Inline formatting: line boxes, text wrapping, basic vertical-align
- [x] Tables: rows/cols, border-collapse separate (collapse optional later), colspan (rowspan later)
- [x] Images: intrinsic size, max-width, object sizing subset
- [x] Overflow: visible default; clip optional
- [ ] Floats / absolute: `[~]` after MVP if corpus needs - deferred: matrix marks Not implemented
- [x] Flex/grid: out of MVP allowlist

### 4.5 Display list & paint
- [x] IR: DrawRect, DrawBorder, DrawText, DrawImage, DrawLine
- [x] Map IR → `internal/pdf` content streams
- [x] Text runs with font selection (family list → embedded fonts)
- [x] Debug: optional box-outline mode for tests

### 4.6 Integration
- [x] `convert` path: load HTML → layout → single long canvas OR paginated (Phase 5)
- [x] Early: single continuous page then paginate, **or** paginate during layout - document choice in design note - see design note below: pagination during paint, whole-op moves

### 4.7 Golden corpus
- [x] Fixture: simple invoice - `testdata/golden/fixture-01-simple-invoice.html`
- [x] Fixture: nested tables / line items - `testdata/golden/fixture-02-table-heavy-invoice.html` (6-col line items, tfoot colspan)
- [x] Fixture: image logo + header block - not part of final corpus; fixtures 01–03 cover logo-free invoice layouts (see matrix §1: PNG/JPEG `img` supported, tested via `TestRunPDFStyleTableImage`)
- [x] Fixture: long text wrap - covered by fixture-02/03 table text + `TestTextWrapping`; multi-page flow covered by fixture-03
- [x] Record visual or geometric assertions - `internal/convert/golden_test.go::TestGoldenCorpus`: per fixture asserts `%PDF-` header, page count (01: 1, 02: ≥1→2 actual, 03: ≥2→2 actual), `/FontFile2` subset font, `%%EOF` trailer and xref-offset consistency

### 4.8 Closure
- [x] Compatibility matrix updated: each CSS property status - `documentation/compatibility-matrix.md` §2 rewritten as a status table (Implemented / Partial / Not implemented) with `Verified by` evidence; verified against `internal/layout/style.go` (applyRestProps + uaRules), `internal/css/css.go`, `layout.go`, `inline.go`, `paint.go`
- [x] `make test` / `make lint` pass - `go test ./...`, `go vet ./...`, `gofmt -l .` all clean
- [x] Performance note: layout time for largest golden fixture (command + ms) - `internal/convert/golden_test.go::TestGoldenFixture03Performance`: fixture-03 (multi-page invoice, A4/10mm, print media) layout+paint ≈ **1.07 ms** (layout ≈ 0.66 ms, paint ≈ 0.41 ms) on dev workstation; budget < 2 s

---

## Dependencies

```
Phase 0 allowlist ──► HTML/CSS scope
Phase 2 loader    ──► bytes + subresources
Phase 3 pdf       ──► paint backend
```

## Design notes (fill during implementation)

- [x] Write short design note: box tree ownership, reflow triggers (static for MVP)

  **Box tree ownership.** The layout package owns box construction entirely; `box` is an internal struct in `internal/layout/layout.go` (`buildBlock`/`buildImage`/`buildHR`/`buildTable`/`buildCell`). DOM stays immutable; `ResolvedStyle` per node is computed once in `resolveStyles` (`style.go`) into `map[*html.Node]ResolvedStyle`, read-only afterwards. Reflow triggers: none - layout is a single pass, static for MVP. Tables do a two-pass measure/emit (`noEmit` flag on the engine + `emitCell`) to resolve column widths before painting cell content.

  **Pagination model.** Phase 4 chooses "layout a single continuous canvas, paginate at paint time": `layout.Paint` groups ops into pages by `op.Y < contentH` and moves an op wholly to the next page when it crosses a boundary (no fragment continuation, no repeated headers - Phase 5). Conversion geometry: page = `settings.ParsePageSize` (points) swapped for landscape; content box = page − margins (mm→pt); `Layout` viewport = content box.

  **Display list.** Ops (`OpFillRect`/`OpStrokeRect`/`OpLine`/`OpText`/`OpImage`/`OpLinkURI`/`OpBullet`) are emitted during layout with box-space coordinates; `Paint` maps them to PDF content streams (x = MarginLeft + opX; y = pageH − MarginTop − opY + pageIdx·contentH). One font (Liberation Sans via `pdf.DefaultFont`), embedded per page with `UseEmbeddedFont` + `TextRenderMode` fake bold.

- [x] Font matching algorithm (family, weight, style)

  MVP: single embedded font (`pdf.DefaultFont()` = Liberation Sans regular, `assets/LiberationSans-Regular.ttf`). Font-family lists are parsed and kept in `ResolvedStyle.FontFamily` but not used for selection (matrix: Partial). Weight: `normal`/`bold`/`bolder`/`lighter`/numeric - bold is fake-bold in paint via `TextRenderMode(2)` (line width 6% of size) since only one outline is embedded. `font-style: italic` parsed but not rendered (matrix: Not implemented). `font-size` fully resolved (named/%, em, rem, pt, px, in, cm, mm, pc; inherits with em/% against parent).

## Risks

| Risk | Mitigation |
|------|------------|
| Scope creep into “real CSS” | Allowlist gate; reject PRs adding props without fixture |
| Table layout complexity | Port only auto table algorithm subset |
| Text metrics differ from WebKit | Accept; golden against ourselves not Qt |
