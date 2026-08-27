# Phase 53: Generated content, lists, counters

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 53
> **Status:** in progress (counters, quotes, list-style-position inside landed; list-style-image `[~]`)
> **Estimated effort:** 5-8 days
> **Owner:** `internal/layout/pseudo_content.go`, list layout in `layout_flow.go`
> **Depends on:** Phase 49 not required
> **Unblocks:** numbered headings, legal-style lists, quote marks from CSS

---

## Overview

`::before` / `::after` already paint quoted strings and `attr()` (`pseudo_content.go`). `counter()`, `counters()`, `url()`, `open-quote` are skipped (`pseudo_content.go:195-201`).

`list-style` shorthand takes type tokens only (`style_properties.go:1157-1163`). Markers are always outside (`layout_flow.go:507`).

Reports use counters for clause numbers. Lists use `inside` in compact invoices.

## Goals

- `counter-reset` / `counter-increment` / `content: counter()`
- `quotes` + open/close-quote subset
- `list-style-position: inside`

## Checklist

### 53.1 counters

- [x] 53.1.1 `counter-reset` / `counter-increment` stored and walked. Proof: `TestCounterResetIncrement`, `TestCounterResetIncrementLayout`.
- [x] 53.1.2 `counter()` / `counters()` in generated content. Proof: `TestCounterInBefore`.
- [~] 53.1.3 Nested ol covered by unit test 1 / 1.1 / 1.2, no new golden this session.
- [x] 53.1.4 Matrix counter row. Mapping `--write`.

### 53.2 quotes

- [x] 53.2.1 Quotes + open/close-quote with nesting. Proof: `TestQuotes`.
- [~] 53.2.2 `no-open-quote` / `no-close-quote` not this session.

### 53.3 list-style-position

- [x] 53.3.1 inside marker. Proof: `TestListStylePositionInside`.
- [x] 53.3.2 list-style shorthand reads position. Proof: `TestListStylePositionParse`.
- [x] 53.3.3 list-style-image via resolveImage. Proof: `TestListStyleImage`.

### 53.4 gates

- [x] 53.4.1 Mapping + matrix. Gates green 2026-08-27.

## Dependencies

`pseudo_content.go` parseContentValue. List marker emission in `layout_flow.go`.

## Evidence

Counter golden. List-inside unit test.

## Out of scope

Full CSS Generated Content. Bookmark properties. `::marker` styling beyond type/position. GCPM running elements.

## Handoff

Next is Phase 54.
