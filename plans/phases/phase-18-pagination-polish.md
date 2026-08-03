# Phase 18 — Pagination Polish (Table Header Repeat & Smarter Breaks)

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** not started  
> **Estimated effort:** 3–6 weeks  
> **Depends on:** Phase 5 pagination; table layout  
> **Unblocks:** long invoices/statements; Tier 2 #8  
> **Tier:** 2 #8 · **Constraint:** stdlib-only

---

## Overview

MVP pagination fragments boxes, honors `page-break-*`, and keeps table rows intact—but does **not** repeat `<thead>` on continued pages, and lacks orphan/widow control. This phase closes high-value print gaps for multi-page tables and long reports.

## Executive Summary

| Feature | MVP | Target |
|---------|-----|--------|
| Table rows unsplit | Yes | Keep |
| `thead` repeat | No | Yes |
| orphans/widows | No | Simple heuristics |
| `--zoom` in convert | Partial | Wired |
| Smart-shrinking re-layout | Warn only | Document or implement subset |

---

## Phase 18 checklist

### 18.1 Table header repeat

- [ ] Detect `thead` / `table-header-group` rows
- [ ] On page break inside `tbody`, re-emit header row(s) at top of next page content box
- [ ] Header height reserved so body rows do not overlap
- [ ] Multi-header-row `thead` supported
- [ ] Nested tables: define behavior (repeat only innermost broken table or all—document)
- [ ] Test: multi-page table fixture asserts header text appears on pages 2+
- [ ] Path: `internal/layout/paint.go` pagination / table fragment path

### 18.2 Smarter page breaks

- [ ] Improve `page-break-inside: avoid` for blocks taller than page (fallback document)
- [ ] Simple orphans/widows: keep N lines of paragraph together when cheap
- [ ] Avoid breaking immediately after heading (optional heuristic)
- [ ] Matrix §2.6 orphans/widows status update

### 18.3 Zoom & smart-shrinking

- [ ] Wire `layout.Options.Zoom` through `internal/convert` render path if still missing
- [ ] Test: `--zoom 0.75` changes content scale / page count band
- [ ] Smart-shrinking: either implement scale-to-fit pass **or** document permanent warn-only with matrix honesty
- [ ] Record decision in fidelity docs

### 18.4 Fixtures

- [ ] Multi-page table with thead (extend fixture-02/03 or new fixture-22)
- [ ] Long paragraph orphans fixture optional
- [ ] Golden page-count envelopes updated

### 18.5 Docs

- [ ] Matrix pagination section
- [ ] README deferred “table header repeat” → done when shipped
- [ ] CLI docs mention thead repeat behavior

### 18.6 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Parent Phase 18 checked
- [ ] Next: **Phase 19** or **20**

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 5 fragmenter | Long report quality |
| Table layout | Header geometry |

---

## Out of scope

- Full CSS Paged Media Level 3
- Named pages / running elements (beyond existing HF)
- Footnote regions
