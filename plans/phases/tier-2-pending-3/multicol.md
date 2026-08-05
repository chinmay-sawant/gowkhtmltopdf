# Tier 2 Pending-3 — CSS Multi-column layout lite

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Status:** not started  
> **Estimated effort:** 2–4 weeks  
> **Constraint:** stdlib-only  
> **Spec:** [CSS Multi-column Layout L1](https://www.w3.org/TR/css-multicol-1/) · [CSS Fragmentation L3](https://www.w3.org/TR/css-break-3/)

---

## Overview

Implement a **report-friendly multicol subset**: `column-count` / `column-width`
/ `columns` shorthand, `column-gap`, `column-span: all|none`, and
`column-fill: balance|auto`. Column boxes are fragmentainers; they must not
straddle page boundaries — pagination starts a new multicol line on the next
page.

---

## Executive Summary

| Property | Target values |
|----------|---------------|
| `column-count` | `auto` \| integer ≥ 1 |
| `column-width` | `auto` \| `<length>` |
| `columns` | shorthand |
| `column-gap` | `normal` (1em) \| `<length>` (distinct from flex/grid gap where needed) |
| `column-span` | `none` \| `all` |
| `column-fill` | `balance` \| `auto` |
| Deferred | `column-rule*`, L2 partial spans, continuous overflow columns |

---

## Phase 1: Design

### 1.1 Evidence

- [ ] Confirm no multicol layout today (`rg column-count internal/`)
- [ ] Note: flex/grid already use `column-gap` as gap — avoid conflating property application contexts
- [ ] Depends on orphans/widows parse ideally first (Class B breaks in columns)

### 1.2 Algorithm notes (from research)

- [ ] Used column count/width from container used width + gap (spec §3.3)
- [ ] Each column box = independent BFC + fragmentainer
- [ ] Column box never splits across pages; new multicol line on next page
- [ ] `column-span: all` → spanner; preceding columns balance
- [ ] Path sketch: `internal/layout/multicol.go` + dispatch in `build` after grid

---

## Phase 2: Parse & style

### 2.1 Properties

- [ ] Parse `column-count`, `column-width`, `columns`, `column-gap` (multicol context), `column-span`, `column-fill`
- [ ] Store on `ResolvedStyle`; `display`/flow enters multicol mode when count/width ≠ auto appropriately
- [ ] Path: `style.go` `applyRestProps`
- [ ] Proof: parse unit tests

### 2.2 Break aliases

- [ ] Honor `break-before/after/inside: column | avoid-column` if cheap; else document subset
- [ ] Interact with existing `page-break-*`

---

## Phase 3: Layout & pagination

### 3.1 Layout

- [ ] `buildMulticol` creates column tracks; flows in-flow content across columns
- [ ] Balance vs auto fill for definite heights
- [ ] Spanner `column-span: all` mid-flow
- [ ] Nested floats/abspos inside columns: best-effort BFC; document limits
- [ ] Path: `multicol.go`, `layout.go` dispatch

### 3.2 Pagination

- [ ] Column boxes do not cross page boundaries
- [ ] Orphans/widows (when parsed) apply per column fragmentainer
- [ ] Path: integrate with `paginateOps` / fragmenter
- [ ] Proof: multi-page 2-column article fixture

---

## Phase 4: Fixtures & tests

### 4.1 Required

- [ ] Unit: 2-column equal widths + gap
- [ ] Unit: `column-span: all` heading mid-flow
- [ ] Convert/golden: `fixture-39-multicol-article.html` (≥2 pages)
- [ ] Envelope in `fixturePageBounds`
- [ ] Document in golden README

### 4.2 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Flip phase-17 `column-count` `[~]` → `[x]` (lite)
- [ ] Matrix Multicol → Partial with property list
- [ ] Next: static-transforms

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Fragmentation + orphans/widows | Column breaks |
| Block/float lite | Content inside columns |

---

## Out of scope

- Exact Chrome balancing with floats/abspos
- `column-rule*`
- Overflow-x scrolling multicol
- CSS Multicol L2 integer spans
