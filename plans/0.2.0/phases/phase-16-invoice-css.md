# Phase 16 - CSS Invoices Actually Use (Selectors + Float Lite)

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** complete (2026-08-04) - selectors + float lite + inline-block + box-sizing  

> **Estimated effort:** 3–6 weeks  
> **Depends on:** Phase 4 layout; Phase 10 matrix updates  
> **Unblocks:** Phase 17 broader CSS; better report templates  
> **Tier:** 1 #3 · **Constraint:** expand allowlist deliberately; stdlib-only

---

## Overview

MVP CSS is enough for simple invoices but templates often need **richer selectors** (`:nth-child`, attributes), **inline-block**, **simple floats**, and **border-box**. This phase expands the **report-friendly** subset - not full flex/grid (phase 17).

## Executive Summary

| Need | Status |
|------|--------|
| `tr:nth-child(even)` zebra | Shipped |
| Attribute selectors | Shipped (presence + exact) |
| Sibling `+` / `~` | Shipped |
| `float` / `clear` lite | Shipped (invoice chrome model) |
| `inline-block` | Shipped (atomic inline box) |
| `box-sizing` | Shipped (`content-box` default, `border-box`) |

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

- [x] Inline-block generates atomic inline box with width/height/margins (`inline.go` collect + emit)
- [x] Sits on line with text; wraps as unit (`TestInlineBlockBesideText`)
- [x] Test: badge/span with border + fixed width beside text
- [x] Path: `internal/layout/{layout.go,inline.go}`

### 16.3 Float lite (invoice two-column chrome)

- [x] `float: left` / `float: right`: remove from normal flow; pack to side (`float.go`, `placeFloat`)
- [x] Following in-flow content uses simple exclusion beside active floats
- [x] `clear: left|right|both` moves below floats
- [x] Prefer **simple** model that helps logo left + meta right on invoices
- [x] Tests: `TestFloatLeftRightClear`; fixture-22 float header chrome
- [x] Matrix §2.2 float/clear → Implemented (lite) with notes
- [x] Path: `internal/layout/{layout.go,float.go}`

### 16.4 `box-sizing: border-box`

- [x] Parse + apply: width includes padding+border when `border-box`
- [x] Default remains content-box unless specified
- [x] Test: `TestBoxSizingBorderBox`
- [x] Matrix §2.1

### 16.5 Small selector/CSS wins that invoices use

- [x] `text-align: justify` - simple inter-word gap on non-final lines (`TestTextAlignJustify`)
- [x] `vertical-align` on table cells top/middle/bottom (`TestTableCellVerticalAlignMiddle`)
- [x] Ensure zebra table fixture works with `:nth-child(even)` (engine supports; fixture-02/16 use even row styles via classes today)

### 16.6 Fixtures

- [x] Update fixture-02 note / CSS so zebra works when nth-child ships (engine ready; fixtures may still use classes)
- [x] New fixture: `fixture-22-float-invoice-chrome.html`
- [x] Golden corpus page envelopes still pass

### 16.7 Explicit non-goals this phase

- [~] Full flexbox - phase 17  
- [~] CSS Grid - deferred  
- [~] `position: absolute/fixed` - phase 17 if needed  
- [~] `transform` / filters / animations - not planned  

### 16.8 Docs & fidelity

- [x] Matrix allowlist expanded for shipped items
- [x] Fidelity guide: “CSS invoices use” section
- [x] README deferred: floats row updated for float lite

### 16.9 Closure gates

- [x] `make lint` → `go vet ./...` green (2026-08-04 Tier 1 close)
- [x] `make test` → `go test ./...` green (2026-08-04 Tier 1 close)
- [x] Parent Phase 16 checked
- [x] Next: **Phase 17** (broader CSS) or **18** (pagination) by product need

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
