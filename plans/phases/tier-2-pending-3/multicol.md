# Tier 2 Pending-3 — CSS Multi-column layout lite

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Status:** done (2026-08-05)  
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

- [x] Confirm no multicol layout today (`rg column-count internal/` — pre-ship; now in `multicol.go` / `style.go`)
- [x] Note: flex/grid already use `column-gap` as gap — avoid conflating property application contexts (`ColumnGapNormal`; multicol → 1em, flex/grid → 0)
- [x] Depends on orphans/widows parse ideally first (Class B breaks in columns) — orphans/widows already on branch; apply per block inside columns

### 1.2 Algorithm notes (from research)

- [x] Used column count/width from container used width + gap (spec §3.3) — `usedColumnCountWidth`
- [x] Each column box = independent BFC + fragmentainer (children built at column width)
- [x] Column box never splits across pages; new multicol line on next page — `flowMulticolSegment`
- [x] `column-span: all` → spanner; preceding columns balance — segment split in `buildMulticol`
- [x] Path sketch: `internal/layout/multicol.go` + dispatch in `build` after grid

---

## Phase 2: Parse & style

### 2.1 Properties

- [x] Parse `column-count`, `column-width`, `columns`, `column-gap` (multicol context), `column-span`, `column-fill`
- [x] Store on `ResolvedStyle`; `display`/flow enters multicol mode when count/width ≠ auto appropriately (`isMulticol`)
- [x] Path: `style.go` `applyRestProps`
- [x] Proof: parse unit tests — `TestMulticolParseProps`

### 2.2 Break aliases

- [x] Honor `break-before/after/inside: column | avoid-column` if cheap; else document subset — aliased to page `always`/`avoid` (lite)
- [x] Interact with existing `page-break-*`

---

## Phase 3: Layout & pagination

### 3.1 Layout

- [x] `buildMulticol` creates column tracks; flows in-flow content across columns
- [x] Balance vs auto fill for definite heights
- [x] Spanner `column-span: all` mid-flow
- [x] Nested floats/abspos inside columns: best-effort BFC; document limits (matrix §2.9)
- [x] Path: `multicol.go`, `layout.go` dispatch

### 3.2 Pagination

- [x] Column boxes do not cross page boundaries
- [x] Orphans/widows (when parsed) apply per column fragmentainer (block Rule 3 still runs in `paginateOps`)
- [x] Path: layout-time fragmentation via `opts.Height` (page content height from convert)
- [x] Proof: multi-page 2-column article fixture — `fixture-39-multicol-article.html`

---

## Phase 4: Fixtures & tests

### 4.1 Required

- [x] Unit: 2-column equal widths + gap — `TestMulticolTwoColumnEqualWidths`
- [x] Unit: `column-span: all` heading mid-flow — `TestMulticolColumnSpanAll`
- [x] Convert/golden: `fixture-39-multicol-article.html` (≥2 pages)
- [x] Envelope in `fixturePageBounds`
- [x] Document in golden README

### 4.2 Gates

- [x] `make lint` → pass (`go vet ./...`, 2026-08-05)
- [x] `make test` → pass via `go test ./internal/layout ./internal/convert -count=1` (layout 0.076s, convert 0.862s)
- [x] Flip phase-17 `column-count` `[~]` → `[x]` (lite)
- [x] Matrix Multicol → Partial with property list (§2.9)
- [x] Next: static-transforms

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
