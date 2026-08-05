# Phase 20 - Headers/Footers & Links Edge Cases

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** done (core + HF fragment GoTo + doc honesty)  
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
| HTML HF links on body pages | Carry URI + fragment GoTo | **Shipped** (external URI + `#id` → body GoTo; copies-aware) |
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
- [x] HTML header/footer: **external URI** link annotations carried onto body pages; **fragment GoTo from HF** → body `AddLinkDest` (copies-aware)
- [x] HF bold uses real face when phase 12 faces available
- [x] Path: `internal/convert/hf.go`, `links.go` (`buildBodyIDIndex`)

### 20.4 Outline / dump

- [x] `dump-outline` page numbers account for TOC pages (`DumpOutlineXMLOffset`)
- [x] Outline depth / heading collection regression coverage via golden / convert tests

### 20.5 Docs

- [x] README deferred rows for resolveRelativeLinks, copies+HF, dump-outline, thead (adjacent Tier 2)
- [x] Matrix / fidelity link + HF blurbs refreshed (shared Tier 2 doc-honesty)

### 20.6 Closure gates

- [x] `make lint` / `make test`
- [x] Golden multi-chapter + HF / fixture-24 regression
- [x] Parent Phase 20 core checked
- [x] HTML-HF fragment GoTo shipped (`hf_links_test.go`)
- [x] Next: **Phase 21** arbitrary websites

---

## Pending (after #17)

> **Execution subplan:** [`subplans-tier-2/phase-20-pending.md`](subplans-tier-2/phase-20-pending.md)  
> **Shared doc honesty:** [`subplans-tier-2/00-shared-doc-honesty.md`](subplans-tier-2/00-shared-doc-honesty.md)

| Item | Notes |
|------|--------|
| HTML HF → body **fragment** (`#id`) GoTo | **[x] Shipped** — `drawHTMLHF` + `buildBodyIDIndex` + `remapPageForCopies` |
| Shared matrix/fidelity refresh | **[x]** Shared doc-honesty pass |
| Full HTML HF as nested documents | **[~] → in progress plan** [`tier-2-pending-3/nested-html-hf.md`](tier-2-pending-3/nested-html-hf.md) (pulled from v0.3.0) |

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 6 locations map | Better navigation PDFs |
| Phase 18 page numbers | Consistent HF `[page]` / `[topage]` |

---

## Out of scope

- Full browser HTML HF as nested browsing contexts
- Nested HF **document** model — **now planned in** [`tier-2-pending-3/nested-html-hf.md`](tier-2-pending-3/nested-html-hf.md) (former v0.3.0 deferral rescinded; see superseded [`subplans-tier-2/nested-hf-v0.3.0.md`](subplans-tier-2/nested-hf-v0.3.0.md))
- PDF named actions beyond GoTo/URI
- CSS running elements / named pages (still out; nested HTML HF ≠ GCPM)
