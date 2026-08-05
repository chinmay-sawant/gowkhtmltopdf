# Tier 2 Pending-3 — `:has()` and container queries lite

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Status:** not started  
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

- [ ] Cite `css.go:parseCompound` — unknown pseudos dropped (including `:has`)
- [ ] Cite `matchPseudo` — only first/last/nth-child
- [ ] Proof: selector unit tests currently reject `:has`

### 1.2 Parse & match

- [ ] Parse `:has( <relative-selector-list> )`
- [ ] Match relative to subject element against descendants/siblings per combinator
- [ ] Specificity = most specific argument in the list (Selectors 4)
- [ ] Reject nested `:has()` and pseudo-elements inside `:has()` (invalid)
- [ ] Path: `internal/css/css.go` (+ tests)
- [ ] Proof: `tr:has(td.warning)`, `section:has(> h2)`, `div:has(+ p)` unit cases

### 1.3 Layout fixture

- [ ] Convert/layout test: `article:has(.footnote) { border-left: … }` toggles style
- [ ] Table row highlight: `tr:has(td.neg) td { color: red }`

---

## Phase 2: `@container` size lite (after widths exist)

### 2.1 Containment prerequisites

- [ ] Parse `container-type`, `container-name`, `container` shorthand
- [ ] `inline-size` containment: principal box inline intrinsic as-if-empty; still layout+style containment lite
- [ ] Without containment, **reject** size queries (avoid layout cycles) or document degrade

### 2.2 Query evaluation

- [ ] Parse `@container` rules into stylesheet
- [ ] After container used inline size known: match queries → apply conditional rules → relayout descendants (two-pass)
- [ ] Nearest eligible named/unnamed ancestor wins
- [ ] Path: `css.go` + second style pass hook near `resolveStyles` / post-measure
- [ ] Proof: `container: card / inline-size` + `@container card (inline-size > 20em)` switches layout

### 2.3 Non-goals

- [~] Style queries `style(--x: y)`
- [~] Scroll-state queries
- [~] Container query length units (`cqw`, …)
- [~] Chrome restyle invalidation performance model

---

## Phase 3: Docs & closure

### 3.1 Honesty

- [ ] Matrix Selectors: `:has` Partial; `@container` Partial (size only)
- [ ] Flip phase-17 container queries / `:has()` `[~]` → split `[x]` / remaining `[~]`
- [ ] Fixtures: `fixture-41-has-selector.html`; optional `fixture-42-container-inline-size.html`

### 3.2 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Record outcomes
- [ ] Next: flex-grid-remaining

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
