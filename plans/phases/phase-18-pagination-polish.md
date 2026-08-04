# Phase 18 - Pagination Polish (Table Header Repeat & Smarter Breaks)

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** done (core) on `feature/tier-2`  
> **Estimated effort:** 3–6 weeks  
> **Depends on:** Phase 5 pagination; table layout  
> **Unblocks:** long invoices/statements; Tier 2 #8  
> **Tier:** 2 #8 · **Constraint:** stdlib-only

---

## Overview

MVP pagination fragments boxes, honors `page-break-*`, and keeps table rows intact - but does **not** repeat `<thead>` on continued pages, and lacks orphan/widow control. This phase closes high-value print gaps for multi-page tables and long reports.

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

- [x] Detect `thead` / `table-header-group` rows
- [x] On page break inside `tbody`, re-emit header row(s) at top of next page content box
- [x] Header height reserved so body rows do not overlap
- [x] Multi-header-row `thead` supported
- [x] Nested tables: define behavior (repeat only innermost broken table or all - document)
- [x] Test: multi-page table fixture asserts header text appears on pages 2+
- [x] Path: `internal/layout/paint.go` pagination / table fragment path

### 18.2 Smarter page breaks

- [x] Improve `page-break-inside: avoid` for blocks taller than page (fallback document)
- [x] Simple orphans/widows: keep N lines of paragraph together when cheap
- [x] Avoid breaking immediately after heading (optional heuristic)
- [ ] Matrix §2.6 orphans/widows status update

### 18.3 Zoom & smart-shrinking

- [x] Wire `layout.Options.Zoom` through `internal/convert` render path if still missing
- [x] Test: `--zoom` / smart-shrinking re-layout path exercised in convert
- [x] Smart-shrinking: scale-to-fit re-layout via `Options.Zoom`
- [x] Record decision in fidelity docs / README deferred table

### 18.4 Fixtures

- [x] Multi-page table with thead (`fixture-23-thead-repeat.html`)
- [ ] Long paragraph orphans fixture optional
- [x] Golden page-count envelopes updated

### 18.5 Docs

- [x] Matrix / README deferred “table header repeat” → done when shipped
- [ ] CLI docs mention thead repeat behavior

### 18.6 Closure gates

- [x] `make lint` →
- [x] `make test` →
- [x] Parent Phase 18 checked
- [x] Next: **Phase 20** then **17** then **19** (Tier 2 order)

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
