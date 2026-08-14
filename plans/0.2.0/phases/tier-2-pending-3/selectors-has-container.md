# Tier 2 Pending-3 — `:has()` and container queries lite

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Status:** done (Phase 1 `:has` + Phase 2 `@container` size lite + Phase 3 docs)  
> **Estimated effort:** 1.5–3 weeks (`:has` first; `@container` second)  
> **Constraint:** stdlib-only  
> **Spec:** [Selectors 4 — `:has()`](https://www.w3.org/TR/selectors-4/#relational) · [CSS Conditional 5 — `@container`](https://www.w3.org/TR/css-conditional-5/) · [CSS Containment 3](https://www.w3.org/TR/css-contain-3/)

---

## Overview

High-ROI selectors for report HTML: implement **`:has()`** in the selector
engine, then a minimal **`@container` size query** subset with `inline-size`
containment. Style queries, scroll-state queries, and `cq*` units stay deferred.

---

## Executive Summary

| Feature | Subset |
|---------|--------|
| `:has()` | Descendant / `>` / `+` / `~`; type/class/id/attr/` :not()`; no nested `:has()` |
| `container-type` | `inline-size` (preferred) \| `size` |
| `@container` | named + unnamed; `inline-size`/`width` comparisons; `and`/`or`/`not` |
| Deferred | `style()` queries, scroll-state, `cqw`/`cqh`, nested `:has(:has())` |

---

## Phase 1: `:has()` (do first)

### 1.1 Evidence

- [x] Cite `css.go:parseCompound` — pre-impl unknown pseudos dropped (including `:has`); now parses `:has` / `:not`
- [x] Cite `matchPseudo` — pre-impl only first/last/nth-child; now also `:has` / `:not`
- [x] Proof: pre-impl `tr:has(td.warning)` parsed as bare `tr` (pseudos empty); closed by unit tests in `has_test.go`

### 1.2 Parse & match

- [x] Parse `:has( <relative-selector-list> )`
- [x] Match relative to subject element against descendants/siblings per combinator
- [x] Specificity = most specific argument in the list (Selectors 4)
- [x] Reject nested `:has()` and pseudo-elements inside `:has()` (invalid)
- [x] Path: `internal/css/css.go` + `internal/css/has.go` (+ `has_test.go`)
- [x] Proof: `tr:has(td.warning)`, `section:has(> h2)`, `div:has(+ p)` unit cases (`TestHasParseAndMatch`)

### 1.3 Layout fixture

- [x] Convert/layout test: `article:has(.footnote) { border-left: … }` toggles style (`TestHasSelectorArticleBorder`)
- [x] Table row highlight: `tr:has(td.neg) td { color: red }` (`TestHasSelectorTableRowHighlight`)

---

## Phase 2: `@container` size lite (after widths exist)

### 2.1 Containment prerequisites

- [x] Parse `container-type`, `container-name`, `container` shorthand (`internal/css/container.go` + `applyRestProps` in `style.go`)
- [x] `inline-size` containment: principal box inline intrinsic as-if-empty (floats / inline-block); still layout+style containment lite
- [x] Without containment, **reject** size queries (avoid layout cycles) — name-only ancestors never enter the size-container map (`TestContainerQueryRequiresContainment`)

### 2.2 Query evaluation

- [x] Parse `@container` rules into stylesheet (`css.Parse` + `Rule.Container`)
- [x] After container used inline size known: match queries → apply conditional rules → layout with final styles (two-pass in `Layout` via `measureSizeContainers` + `resolveStylesWithContainers`)
- [x] Nearest eligible named/unnamed ancestor wins (`TestContainerQueryNearestNamedWins`)
- [x] Path: `internal/css/container.go` + `internal/layout/container.go` + `style.go` / `layout.go`
- [x] Proof: `container: card / inline-size` + `@container card (inline-size > 20em)` switches layout (`TestContainerQueryNamedInlineSize`, `TestContainerQueryLayoutSwitch`, `fixture-42-container-inline-size.html`)

### 2.3 Non-goals (permanent product boundaries)

- [x] Style queries `style(--x: y)` — out of scope by design
- [x] Scroll-state queries — out of scope by design (PDF has no scroll)
- [x] Container query length units (`cqw`, …) — out of scope by design
- [x] Chrome restyle invalidation performance model — out of scope by design

---

## Phase 3: Docs & closure

### 3.1 Honesty

- [x] Matrix Selectors: `:has` Partial; `@container` Partial (size only) — 2026-08-05
- [x] Flip phase-17 container queries / `:has()` → `[x]` shipped lite (non-goals above)
- [x] Fixtures: `fixture-41-has-selector.html`; `fixture-42-container-inline-size.html` (golden bounds registered)

### 3.2 Gates

- [x] `make lint` → `go vet ./...` OK (2026-08-05 Phase 2)
- [x] Scoped tests → `go test ./internal/css ./internal/layout ./internal/convert -count=1` OK (2026-08-05 Phase 2)
- [x] Record outcomes (above)
- [x] Phase 3 docs honesty closed; wave-2 sibling tracks complete

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| CSS Match / cascade | `:has` |
| Definite widths / flex/grid | `@container` size |
| Phase 21 report HTML | Card/table patterns |

---

## Out of scope

- Full Selectors 4
- Full Containment / Conditional Level 5
- Using `@container` against page size (that's `@media`)
