# Phase 20 - Headers/Footers & Links Edge Cases

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** done (core) on `master` via #16; **one HF HTML-link leftover**  
> **Estimated effort:** 2–4 weeks  
> **Depends on:** Phase 6 HF/TOC/links MVP  
> **Unblocks:** Tier 2 #9 polish  
> **Tier:** 2 #9 · **Constraint:** stdlib-only

---

## Overview

Close known **edge gaps** listed in the README deferred table for
headers/footers, anchors, and relative links - without redesigning the Phase 6
model.

## Executive Summary

| Gap (README) | Target | Status (2026-08-05) |
|--------------|--------|---------------------|
| Inline `#anchor` source rects | Best-effort boxes for inlines | **Shipped** (`fixture-24`) |
| `resolveRelativeLinks` | Implement flag behavior | **Shipped** |
| HTML HF links on body pages | Carry URI annotations if feasible | **Partial** (external URI carried; HF fragment GoTo limited) |
| `[topage]` with copies | Correct when copies > 1 | **Shipped** |
| `dump-outline` TOC offset | Include TOC pages | **Shipped** |
| Cross-object URL map | Same-document multi-object anchors | **Shipped** (lite) |

---

## Phase 20 checklist

### 20.1 Internal links

- [x] Inline `<a href="#id">` produces clickable target and source rect when geometry available
- [x] Document remaining cases where inline has no box (README / matrix honesty)
- [x] Cross-object destinations when multi-object PDF used (best-effort)
- [x] Tests: fixture-06 + `fixture-24-internal-anchors`
- [x] Path: `internal/convert/links.go`, layout locations

### 20.2 Relative / resolve flags

- [x] Implement `resolveRelativeLinks` (`--resolve-relative-links` / `--keep-relative-links`)
- [x] External http(s)/mailto still work
- [x] Tests for relative href against page base URL (`links_resolve_test.go`)

### 20.3 Header/footer edges

- [x] `[topage]` correct under `--copies` > 1 (HF after copies bake order)
- [~] HTML header/footer: **external URI** link annotations carried onto body pages; **fragment GoTo from HF** still limited
- [x] HF bold uses real face when phase 12 faces available
- [x] Path: `internal/convert/hf.go`

### 20.4 Outline / dump

- [x] `dump-outline` page numbers account for TOC pages (`DumpOutlineXMLOffset`)
- [x] Outline depth / heading collection regression coverage via golden / convert tests

### 20.5 Docs

- [x] README deferred rows for resolveRelativeLinks, copies+HF, dump-outline, thead (adjacent Tier 2)
- [ ] Matrix / older fidelity blurbs may still lag — shared Tier 2 doc-sync pending

### 20.6 Closure gates

- [x] `make lint` / `make test`
- [x] Golden multi-chapter + HF / fixture-24 regression
- [x] Parent Phase 20 core checked
- [ ] Remaining: HTML-HF fragment GoTo (see Pending)
- [x] Next: **Phase 21** arbitrary websites

---

## Pending (after #17)

> **Execution subplan:** [`subplans-tier-2/phase-20-pending.md`](subplans-tier-2/phase-20-pending.md)  
> **Shared doc honesty:** [`subplans-tier-2/00-shared-doc-honesty.md`](subplans-tier-2/00-shared-doc-honesty.md)

| Item | Notes |
|------|--------|
| HTML HF → body **fragment** (`#id`) GoTo | External URI from HTML HF shipped; same-doc GoTo from HF still limited |
| Shared matrix/fidelity refresh | Cross-cutting with phases 17–19 honesty pass |
| Full HTML HF as nested documents | Out of scope (Phase 6 model stands) |

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 6 locations map | Better navigation PDFs |
| Phase 18 page numbers | Consistent HF `[page]` / `[topage]` |

---

## Out of scope

- Full HTML HF as nested full browser documents
- PDF named actions beyond GoTo/URI
