# Phase 06 — Headers/Footers, TOC, Outlines, Links

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** not started  
> **Estimated effort:** 2–4 months solo  
> **Depends on:** Phase 5 pagination + element locations  
> **Unblocks:** Feature-complete report MVP

---

## Overview

Port outline extraction, PDF bookmarks, text/HTML headers & footers, TOC generation (without XSLT2), and link annotations — matching `outline.cc` + HF paths in `pdfconverter.cc`.

---

## Checklist

### 6.1 Outline (`internal/outline`)
- [ ] Collect headings h1–h6 (extend to h9 if needed)
- [ ] Sort by (page, y, x) using element location map
- [ ] Level-stack tree heuristic (non-monotonic headings)
- [ ] Assign synthetic anchors `__WKANCHOR_*` (base36) for TOC back/forward
- [ ] Emit PDF `/Outlines` up to `outlineDepth`
- [ ] `dumpOutline` XML with namespace `http://wkhtmltopdf.org/outline` (parity optional)

### 6.2 Text headers/footers
- [ ] Draw left/center/right in margin band
- [ ] Font name/size (map to embedded fonts; fallback default)
- [ ] Separator line
- [ ] Spacing (mm → pt)
- [ ] `fillParms` / `hfreplace` for all documented placeholders + `--replace`
- [ ] Section/subsection from outline HF cache (levels 0–2)
- [ ] Auto margin when top/bottom = -1: reserve max HF height across objects

### 6.3 HTML headers/footers
- [ ] Load `header.htmlUrl` / `footer.htmlUrl` via loader
- [ ] Inject params (query or in-document replace)
- [ ] Measure height (layout HF as single page, measure body)
- [ ] Composite onto each page after body spool
- [ ] Reject raw HTML strings that look like markup not URLs (`looksLikeHtmlAndNotAUrl`)

### 6.4 TOC
- [ ] Object type `isTableOfContent`
- [ ] Generate TOC HTML from outline via **Go templates** (default look from `tocstylesheet.cc` semantics)
- [ ] Settings: caption, dotted lines, indentation, fontScale, forwardLinks
- [ ] Re-layout TOC; if page count changes, re-number (fixed-point loop with max iterations)
- [ ] `[~]` Custom `--xsl-style-sheet`: unsupported — error or ignore with warning

### 6.5 Links
- [ ] Internal: same-document anchors + cross-object URL map (`urlToPageObj`)
- [ ] External: URI annotations
- [ ] Flags: useLocalLinks, useExternalLinks, resolveRelativeLinks

### 6.6 Forms
- [ ] `[~]` `produceForms` deferred — optional post-MVP AcroForm text/checkbox only

### 6.7 Tests
- [ ] Heading outline structure test
- [ ] Placeholder substitution table
- [ ] Multi-chapter + TOC page count
- [ ] Internal link click dest (structural PDF test)

### 6.8 Closure
- [ ] MVP feature set complete for reports
- [ ] `make test` / `make lint`
- [ ] Parent ledger Phase 6 rows closed with proof

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
