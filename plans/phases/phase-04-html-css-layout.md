# Phase 04 — HTML Parser + CSS Subset Layout

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** not started  
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
- [ ] Tokenizer for HTML subset (do not use full browser error recovery unless needed)
- [ ] Tree construction into DOM nodes
- [ ] Encoding: UTF-8 primary; honor meta charset + defaultEncoding
- [ ] Strip `script`; ignore `iframe`/`object`/`embed` for MVP
- [ ] Collect: stylesheets (link + style), images, base href
- [ ] Tests: malformed tags still produce usable tree for common cases

### 4.2 CSS parse (`internal/css`)
- [ ] Stylesheet parser: rules, selectors, declarations
- [ ] Selectors: `*`, type, `.class`, `#id`, descendant, child (optional)
- [ ] Specificity + source order + important (basic)
- [ ] Inheritance list (color, font-*, line-height, text-align, …)
- [ ] Value parse: colors (#rgb/#rrggbb/named subset), lengths, percentages, keywords
- [ ] `@media print` / `screen` filtering
- [ ] User stylesheet + inline style + element style attribute

### 4.3 Used style
- [ ] Style resolution per element against DOM
- [ ] Default UA stylesheet for headings, lists, tables, `a`, `p`, `body`

### 4.4 Layout (`internal/layout`)
- [ ] Containing blocks, viewport width from page content box
- [ ] Block formatting context: margins, padding, borders, width/height auto
- [ ] Margin collapsing (document subset)
- [ ] Inline formatting: line boxes, text wrapping, basic vertical-align
- [ ] Tables: rows/cols, border-collapse separate (collapse optional later), colspan (rowspan later)
- [ ] Images: intrinsic size, max-width, object sizing subset
- [ ] Overflow: visible default; clip optional
- [ ] Floats / absolute: `[~]` after MVP if corpus needs
- [ ] Flex/grid: out of MVP allowlist

### 4.5 Display list & paint
- [ ] IR: DrawRect, DrawBorder, DrawText, DrawImage, DrawLine
- [ ] Map IR → `internal/pdf` content streams
- [ ] Text runs with font selection (family list → embedded fonts)
- [ ] Debug: optional box-outline mode for tests

### 4.6 Integration
- [ ] `convert` path: load HTML → layout → single long canvas OR paginated (Phase 5)
- [ ] Early: single continuous page then paginate, **or** paginate during layout — document choice in design note

### 4.7 Golden corpus
- [ ] Fixture: simple invoice
- [ ] Fixture: nested tables / line items
- [ ] Fixture: image logo + header block
- [ ] Fixture: long text wrap
- [ ] Record visual or geometric assertions

### 4.8 Closure
- [ ] Compatibility matrix updated: each CSS property status
- [ ] `make test` / `make lint` pass
- [ ] Performance note: layout time for largest golden fixture (command + ms)

---

## Dependencies

```
Phase 0 allowlist ──► HTML/CSS scope
Phase 2 loader    ──► bytes + subresources
Phase 3 pdf       ──► paint backend
```

## Design notes (fill during implementation)

- [ ] Write short design note: box tree ownership, reflow triggers (static for MVP)
- [ ] Font matching algorithm (family, weight, style)

## Risks

| Risk | Mitigation |
|------|------------|
| Scope creep into “real CSS” | Allowlist gate; reject PRs adding props without fixture |
| Table layout complexity | Port only auto table algorithm subset |
| Text metrics differ from WebKit | Accept; golden against ourselves not Qt |
