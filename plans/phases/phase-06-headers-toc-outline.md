# Phase 06 — Headers/Footers, TOC, Outlines, Links

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** complete (2026-08-03)  
> **Estimated effort:** 2–4 months solo  
> **Depends on:** Phase 5 pagination + element locations  
> **Unblocks:** Feature-complete report MVP

---

## Overview

Port outline extraction, PDF bookmarks, text/HTML headers & footers, TOC generation (without XSLT2), and link annotations — matching `outline.cc` + HF paths in `pdfconverter.cc`.

---

## Checklist

### 6.1 Outline (`internal/outline`)
- [x] Collect headings h1–h6 (extend to h9 if needed) — `outline.CollectHeadings` (h1–h6, whitespace-collapsed titles)
- [x] Sort by (page, y, x) using element location map — sort via `Result.Locations` (`outline.BuildTree`)
- [x] Level-stack tree heuristic (non-monotonic headings) — level-stack with clamp (level jump >+1 clamps to previous+1)
- [x] Assign synthetic anchors `__WKANCHOR_*` (base36) for TOC back/forward — `outline.AssignAnchors`, stable
- [x] Emit PDF `/Outlines` up to `outlineDepth` — wired in convert (per-object UseOutline/IncludeInOutline gates, ExcludeFromOutline selectors via css.Match)
- [x] `dumpOutline` XML with namespace `http://wkhtmltopdf.org/outline` (parity optional) — `outline.DumpOutlineXML` (1-based pages, link/backLink attrs)

### 6.2 Text headers/footers
- [x] Draw left/center/right in margin band — hf.go, per-page margin-band content
- [x] Font name/size (map to embedded fonts; fallback default) — default font, FontSize honored (default 12pt)
- [x] Separator line — HeaderFooter.Line
- [x] Spacing (mm → pt) — HeaderFooter.Spacing
- [x] `fillParms` / `hfreplace` for all documented placeholders + `--replace` — `[page]`(+PageOffset), `[topage]`, `[frompage]`, `[date]`, `[time]`, `[title]`, `[doctitle]`, `[webpage]`, `[section]`/`[subsection]`; unknown tokens pass through; replace merged from global+object header/footer maps
- [x] Section/subsection from outline HF cache (levels 0–2) — first/last heading at-or-before the page
- [x] Auto margin when top/bottom = -1: reserve max HF height across objects — measured HF height (font line box or HTML HF layout) + spacing applied to margins before body layout

### 6.3 HTML headers/footers
- [x] Load `header.htmlUrl` / `footer.htmlUrl` via loader — loader.FetchSub with object base, pre-loaded/cached per object
- [x] Inject params (query or in-document replace) — placeholder injection; per-page re-layout when placeholders present
- [x] Measure height (layout HF as single page, measure body) — layout at content width, height from layout
- [x] Composite onto each page after body spool — ops redrawn per page into the margin band (clip to band), images embedded per page
- [x] Reject raw HTML strings that look like markup not URLs (`looksLikeHtmlAndNotAUrl`) — warns + ignores

### 6.4 TOC
- [x] Object type `isTableOfContent` — `IsTableOfContent` objects render generated TOC HTML (caption h1, per-entry div with dotted leader + page number)
- [x] Generate TOC HTML from outline via **Go templates** (default look from `tocstylesheet.cc` semantics)
- [x] Settings: caption, dotted lines, indentation, fontScale, forwardLinks — CaptionText/Indentation/FontScale/DottedLines/ForwardLinks/BackLinks honored
- [x] Re-layout TOC; if page count changes, re-number (fixed-point loop with max iterations) — fixed-point, max 2 iterations; TOC pages prepended via ReorderPages so `[page]`, outline dests and TOC numbers include TOC pages
- [x] `[~]` Custom `--xsl-style-sheet`: unsupported — error or ignore with warning — warns + ignores

### 6.5 Links
- [x] Internal: same-document anchors + cross-object URL map (`urlToPageObj`) — MVP: TOC forward/back links (block entries + heading boxes → AddLinkDest); arbitrary inline `<a href="#x">` source rects skipped (inline elements produce no boxes; documented TODO); cross-object map deferred
- [x] External: URI annotations — layout OpLinkURI → AddLinkURI (verified end-to-end)
- [x] Flags: useLocalLinks, useExternalLinks, resolveRelativeLinks — ExternalLinks gate neutralizes OpLinkURI ops (OpKind sentinel); LocalLinks read; resolveRelativeLinks deferred

### 6.6 Forms
- [ ] `[~]` `produceForms` deferred — optional post-MVP AcroForm text/checkbox only — deferred as planned

### 6.7 Tests
- [x] Heading outline structure test — TestOutlineTreeNesting, TestOutlineTreeSortAndClamp, TestOutlineTreeLevelStackAcrossPages, TestOutlineDepth, TestOutlineExclude, TestOutlineCollectTitlesAndAnchors
- [x] Placeholder substitution table — TestTextHeaderFooter, TestPlaceholderReplace, TestSectionSubsectionPlaceholder, TestFromPagePlaceholder
- [x] Multi-chapter + TOC page count — TestTOC (caption + chapter titles in output, ≥2 pages)
- [x] Internal link click dest (structural PDF test) — TestInternalLinkDest, TestOutlineWiring

### 6.8 Closure
- [x] MVP feature set complete for reports
- [x] `make test` / `make lint` — `go test ./...` / `go vet ./...` / `gofmt -l .` all clean (2026-08-03)
- [x] Parent ledger Phase 6 rows closed with proof — ledger row updated

---

## Design notes (filled 2026-08-03)

- **Outline tree is a pure `outline.Node` tree** (canvas coords); convert maps it to `pdf.Outline` where geometry + `PageRef` (new `pdf.Document.PageRef(idx)`) live — internal/outline stays unit-testable without pdf/layout coupling.
- **Body pages painted once, then `ReorderPages` moves TOC pages to the front** — no scratch docs, no re-paint; annotations/outline/HF passes all use final indices afterwards.
- **External-link gating neutralizes ops in place** (sentinel OpKind Paint ignores) instead of filtering — removing ops would corrupt box-tree op indices and crash pagination.
- **Auto margin measures text via font line box, HTML HF via its layout height**, pre-loading/caching HTML HFs in the loading loop (upstream loads headers before rendering).
- **Known limitations**: dump-outline XML pages are body-relative (no TOC offset); TOC fixed-point may drift if re-layout changes the count; HTML HF links not carried to body pages; `[topage]` ignores copies (HF baked before duplication); `[subject]` expands empty (no setting field); inline `<a href="#x">` source-rect links not emitted.

---

## Dependencies

| Component | Need |
|-----------|------|
| element locations | Phase 5 |
| layout HTML HF | Phase 4 |
| PDF annots/outline | Phase 3 |

## Upstream refs

- `outline.cc` — tree, dump, printOutline, fillHeaderFooterParms  
- `pdfconverter.cc` — endPage, loadHeaders, loadTocs, findLinks, fillParms  
- `tocstylesheet.cc` — default TOC visual rules  
