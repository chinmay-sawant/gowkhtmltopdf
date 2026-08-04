# Tier 2 Subplan - Nested HTML headers/footers as documents

> **Parent:** [`plans/phases/phase-20-hf-links-edges.md`](../phase-20-hf-links-edges.md)  
> **Status:** `[~]` **deferred to product version 0.3.0**  
> **Estimated effort:** 2–6 weeks (when scheduled)  
> **Constraint:** stdlib-only; no full browser HF  
> **Target release:** **v0.3.0** (current tree is 0.1.x / post-MVP Tier 2)

---

## Overview

Phase 6/20 model loads HTML HF as a **separate mini-layout** stamped into the
margin band (text + URI/GoTo links). “Full nested HTML HF” means running HF
through a **document-class pipeline** (shared convert/layout subset) with
explicit compositing into page chrome — still not a nested browser.

This work is **explicitly deferred** to **version 0.3.0** so Tier 2 closure and
near-term subplans (sticky, flex/grid, typesetting, image `@font-face`) are not
blocked.

## Executive Summary

| Today (≤0.2.x target) | v0.3.0 target |
|-----------------------|---------------|
| `loadHTMLHF` + margin stamp | Nested document layout pass for HF HTML |
| Links: URI + fragment GoTo | Same + richer HF CSS/images as body subset |
| Out of scope: full browser HF | Still out of scope |

---

## Phase 0: Deferral record (now)

- [x] Decision: defer nested HF documents to **v0.3.0**
- [x] Pointer from Phase 20 Out of scope / Pending
- [ ] When cutting 0.3.0: move this file’s Status to `not started` and schedule

---

## Phase 1: Design (v0.3.0)

### 1.1 Model

- [ ] Define HF as child `objectState` / layout result with viewport = margin band
      width × HF height
- [ ] Reuse load → CSS → layout → paint subset; clip to header/footer rect
- [ ] Page geometry: HF height still reserved from content box (existing
      `hfHeightFor`)
- [ ] Path sketch: `internal/convert/hf.go`, possibly `hf_doc.go`

### 1.2 Links & assets

- [ ] Keep fragment GoTo / URI behavior; re-test copies remapping
- [ ] Images/fonts in HF under same ACL as body
- [ ] Nested tables/flex in HF — inherit whatever layout Tier 2+ supports

### 1.3 Non-goals (even in 0.3.0)

- [~] Running elements / CSS Paged Media named pages
- [~] HF that paginates independently across many pages of its own content
      beyond height clamp
- [~] Browser-engine HF

---

## Phase 2: Implementation checklist (v0.3.0)

- [ ] Spike: one HTML HF with flex + image matches body-quality paint in band
- [ ] Tests: extend `hf_links_test.go` + visual/golden envelope
- [ ] Docs: matrix HF section; README deferred row
- [ ] `make lint` / `make test`

---

## Phase 3: Closure (v0.3.0)

- [ ] Nested HF document model documented and tested
- [ ] VERSION / changelog note for 0.3.0
- [ ] Phase 20 out-of-scope bullet rewritten to “shipped in 0.3.0” or narrowed

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 20 HF GoTo / URI | Link semantics to preserve |
| Flex/grid/sticky as of 0.3.0 | HF layout richness |
| Product versioning 0.3.0 | Schedule gate |

---

## Out of scope (permanent unless amended)

- Full browser HTML HF as nested browsing contexts
- PDF named actions beyond GoTo/URI
