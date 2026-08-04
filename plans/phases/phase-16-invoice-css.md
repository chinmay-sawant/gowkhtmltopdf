# Phase 16 - CSS Invoices Actually Use (Selectors + Float Lite)

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** partial (2026-08-04) - selector expansion shipped; float/inline-block/box-sizing remain  

> **Estimated effort:** 3–6 weeks  
> **Depends on:** Phase 4 layout; Phase 10 matrix updates  
> **Unblocks:** Phase 17 broader CSS; better report templates  
> **Tier:** 1 #3 · **Constraint:** expand allowlist deliberately; stdlib-only

---

## Overview

MVP CSS is enough for simple invoices but templates often need **richer selectors** (`:nth-child`, attributes), **inline-block**, **simple floats**, and **border-box**. This phase expands the **report-friendly** subset - not full flex/grid (phase 17).

## Executive Summary

| Need | Today |
|------|-------|
| `tr:nth-child(even)` zebra | Pseudo dropped; matches all rows |
| Attribute selectors | Dropped at tokenize |
| Sibling `+` / `~` | Parsed as descendant |
| `float` | Parsed, not positioned |
| `inline-block` | Degrades to block (except img hack) |
| `box-sizing` | Not implemented |

---

## Phase 16 checklist

### 16.1 Selector expansion

- [x] Implement attribute selectors subset: `[attr]`, `[attr="value"]` (exact) - `css.go` AttrSelector
- [~] `[attr~=]`, `[attr|=]`, `[attr^=]`, `[attr$=]`, `[attr*=]` - only if fixtures need
- [x] `:first-child`, `:last-child`
- [x] `:nth-child(n)` / `odd` / `even` / `an+b` simple subset (`TestNthChildZebraSheet`, `TestMatch`)
- [x] Fix sibling combinators: `A + B`, `A ~ B` match correctly (not descendant) - css_test sibling cases
- [x] Tests in `internal/css/css_test.go` for each
- [x] Path: `internal/css/css.go` tokenize + `Match`
- [x] Matrix §4 updated row-by-row

### 16.2 `display: inline-block` real layout

- [ ] Inline-block generates atomic inline box with width/height/margins (parsed; full layout model incomplete)
- [ ] Sits on line with text; wraps as unit
- [ ] Test: badge/span with border + fixed width beside text
- [ ] Path: `internal/layout/{layout.go,inline.go}`

### 16.3 Float lite (invoice two-column chrome)

- [ ] `float: left` / `float: right`: remove from normal flow; pack to side
- [ ] Following in-flow content wraps (simple exclusion) **or** document reduced model if only “float then clear” supported first
- [ ] `clear: left|right|both` moves below floats
- [ ] Prefer **simple** model that helps logo left + meta right on invoices
- [ ] Tests: logo float left + address block; clear after
- [ ] Matrix §2.2 float/clear → Partial or Implemented with notes
- [ ] Path: `internal/layout/layout.go` (still documents floats out of scope)

### 16.4 `box-sizing: border-box`

- [ ] Parse + apply: width includes padding+border
- [ ] Default remains content-box unless specified
- [ ] Test: fixed width box with padding matches expected border box
- [ ] Matrix §2.1

### 16.5 Small selector/CSS wins that invoices use

- [ ] `text-align: justify` - either implement simple or leave Partial documented
- [ ] `vertical-align` on table cells baseline/middle/top subset if cheap
- [x] Ensure zebra table fixture works with `:nth-child(even)` (engine supports; fixture-02/16 use even row styles via classes today)

### 16.6 Fixtures

- [x] Update fixture-02 note / CSS so zebra works when nth-child ships (engine ready; fixtures may still use classes)
- [ ] New or extended fixture: float header chrome invoice
- [x] Golden corpus page envelopes still pass

### 16.7 Explicit non-goals this phase

- [~] Full flexbox - phase 17  
- [~] CSS Grid - deferred  
- [~] `position: absolute/fixed` - phase 17 if needed  
- [~] `transform` / filters / animations - not planned  

### 16.8 Docs & fidelity

- [x] Matrix allowlist expanded only for shipped items (selectors)
- [ ] Fidelity guide: “CSS invoices use” section checked
- [ ] README deferred: floats row updated when float lite ships

### 16.9 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Parent Phase 16 checked
- [ ] Next: **Phase 17** (broader CSS) or **18** (pagination) by product need

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| CSS cascade engine | Phase 17 flex |
| Layout BFC | Float exclusion |

---

## Out of scope

- Grid, multi-column, sticky, container queries
- Full CSS2 float edge-case corpus (years of browser bugs)
