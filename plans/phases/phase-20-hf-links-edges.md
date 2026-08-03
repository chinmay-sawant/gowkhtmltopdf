# Phase 20 — Headers/Footers & Links Edge Cases

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** not started  
> **Estimated effort:** 2–4 weeks  
> **Depends on:** Phase 6 HF/TOC/links MVP  
> **Unblocks:** Tier 2 #9 polish  
> **Tier:** 2 #9 · **Constraint:** stdlib-only

---

## Overview

Close known **edge gaps** listed in README deferred table for headers/footers, anchors, and relative links—without redesigning the Phase 6 model.

## Executive Summary

| Gap (README) | Target |
|--------------|--------|
| Inline `#anchor` source rects | Best-effort boxes for inlines |
| `resolveRelativeLinks` | Implement flag behavior |
| HTML HF links on body pages | Carry link annotations if feasible |
| `[topage]` with copies | Correct when copies > 1 |
| `dump-outline` TOC offset | Include TOC pages if applicable |
| Cross-object URL map | Same-document multi-object anchors |

---

## Phase 20 checklist

### 20.1 Internal links

- [ ] Inline `<a href="#id">` produces clickable target and source rect when geometry available
- [ ] Document remaining cases where inline has no box
- [ ] Cross-object destinations when multi-object PDF used
- [ ] Tests: fixture-06 + multi-section fixture
- [ ] Path: `internal/convert/links.go`, layout locations

### 20.2 Relative / resolve flags

- [ ] Implement `resolveRelativeLinks` (or document permanent ignore with matrix row)
- [ ] External http(s)/mailto still work
- [ ] Tests for relative href against page base URL

### 20.3 Header/footer edges

- [ ] `[topage]` correct under `--copies` > 1 (bake order fix)
- [ ] HTML header/footer: if links present, either paint annotations on body pages or document “text only”
- [ ] HF bold uses real face if phase 12 done
- [ ] Path: `internal/convert/hf.go`

### 20.4 Outline / dump

- [ ] `dump-outline` page numbers account for TOC pages when TOC object present
- [ ] Outline depth / heading collection regression tests

### 20.5 Docs

- [ ] Matrix / CLI flag rows for resolveRelativeLinks, copies+HF
- [ ] README deferred rows cleared or narrowed

### 20.6 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Golden multi-chapter + HF regression
- [ ] Parent Phase 20 checked
- [ ] Next: **Phase 21**

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 6 locations map | Better navigation PDFs |
| Phase 18 page numbers | Consistent HF |

---

## Out of scope

- Full HTML HF as nested full browser documents
- PDF named actions beyond GoTo/URI
