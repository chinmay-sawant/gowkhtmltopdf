# Phase 18 - Pagination Polish (Table Header Repeat & Smarter Breaks)

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** done (core) on `master` via #16; **docs polish pending**  
> **Estimated effort:** 3–6 weeks  
> **Depends on:** Phase 5 pagination; table layout  
> **Unblocks:** long invoices/statements; Tier 2 #8  
> **Tier:** 2 #8 · **Constraint:** stdlib-only

---

## Overview

MVP pagination fragments boxes, honors `page-break-*`, and keeps table rows
intact. This phase adds **`<thead>` repeat** on continued pages, simple
orphan/widow / keep-with-next heuristics, and wires `--zoom` /
smart-shrinking re-layout.

## Executive Summary

| Feature | MVP | Target | Status (2026-08-05) |
|---------|-----|--------|---------------------|
| Table rows unsplit | Yes | Keep | **Shipped** |
| `thead` repeat | No | Yes | **Shipped** (`fixture-23`) |
| orphans/widows | No | Simple heuristics | **Shipped** (heuristics; CSS props not parsed) |
| `--zoom` in convert | Partial | Wired | **Shipped** |
| Smart-shrinking re-layout | Warn only | Subset | **Shipped** via `Options.Zoom` |

---

## Phase 18 checklist

### 18.1 Table header repeat

- [x] Detect `thead` / `table-header-group` rows
- [x] On page break inside `tbody`, re-emit header row(s) at top of next page content box
- [x] Header height reserved so body rows do not overlap
- [x] Multi-header-row `thead` supported
- [x] Nested tables: each table repeats only its own thead (documented in code/README)
- [x] Test: multi-page table fixture asserts header text appears on pages 2+
- [x] Path: `internal/layout/paint.go` (`repeatTableHeaders`)

### 18.2 Smarter page breaks

- [x] Improve `page-break-inside: avoid` for blocks taller than page (fallback documented)
- [x] Simple orphans/widows: keep N lines of paragraph together when cheap
- [x] Avoid breaking immediately after heading (optional heuristic)
- [ ] Matrix §2.6: still says `orphans`/`widows` “Not implemented” — update to Partial / heuristic note
- [ ] Fidelity MVP-gap row still says “thead repeat no” — **stale; fix**

### 18.3 Zoom & smart-shrinking

- [x] Wire `layout.Options.Zoom` through `internal/convert` render path
- [x] Test: `--zoom` / smart-shrinking re-layout path exercised in convert
- [x] Smart-shrinking: scale-to-fit re-layout via `Options.Zoom`
- [x] Record decision in README deferred table

### 18.4 Fixtures

- [x] Multi-page table with thead (`fixture-23-thead-repeat.html`)
- [ ] Long paragraph orphans fixture (optional quality)
- [x] Golden page-count envelopes updated

### 18.5 Docs

- [x] README deferred “table header repeat” → shipped
- [ ] CLI / library docs mention thead repeat behavior explicitly
- [ ] Compatibility-matrix pagination paragraph still describes pre-phase-18 state — **pending refresh**

### 18.6 Closure gates

- [x] `make lint` →
- [x] `make test` →
- [x] Parent Phase 18 core checked
- [ ] Remaining: matrix/fidelity/CLI honesty (see Pending)
- [x] Next: Phase 20 / 17 / 19 were parallel Tier 2; product next is **Phase 21** or doc sync

---

## Pending (after #17)

> **Execution subplan:** [`subplans-tier-2/phase-18-pending.md`](subplans-tier-2/phase-18-pending.md)  
> **Shared doc honesty:** [`subplans-tier-2/00-shared-doc-honesty.md`](subplans-tier-2/00-shared-doc-honesty.md)

| Item | Notes |
|------|--------|
| Matrix §2.6 orphans/widows | Heuristics exist; CSS `orphans`/`widows` properties still absent |
| Fidelity + matrix pagination blurbs | Still claim no thead repeat / old zoom story |
| CLI docs thead repeat | README mentions; dedicated CLI/library note optional |
| Optional orphans fixture | Nice-to-have, not blocking |

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
- CSS `orphans` / `widows` property parsing (heuristics only unless amended)
