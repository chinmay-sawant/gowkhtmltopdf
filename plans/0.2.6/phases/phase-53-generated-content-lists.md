# Phase 53: Generated content, lists, counters

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 53
> **Status:** not started
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

- [ ] 53.1.1 Parse `counter-reset` and `counter-increment` onto the style or a document counter map. Proof: `go test ./internal/layout -run TestCounterResetIncrement`.
- [ ] 53.1.2 `content: counter(name)` and `counters(name, sep)` in `parseContentValue`. Proof: `TestCounterInBefore`.
- [ ] 53.1.3 Nested `ol` fixture showing 1 / 1.1 / 1.2. Add golden + `fixturePageBounds` if paint changes. Proof: `make golden` includes the new fixture.
- [ ] 53.1.4 Matrix generated-content row. Mapping flip.

### 53.2 quotes

- [ ] 53.2.1 `quotes` property inherited. `content: open-quote` / `close-quote` uses the pair. Nesting depth increments. Proof: `go test ./internal/layout -run TestQuotes`.
- [ ] 53.2.2 `no-open-quote` / `no-close-quote` may `[~]`.

### 53.3 list-style-position

- [ ] 53.3.1 `list-style-position: inside` puts the marker in the first line box. `outside` keeps current hanging marker. Proof: `go test ./internal/layout -run TestListStylePositionInside`.
- [ ] 53.3.2 `list-style` shorthand still parses type and also position when present. Proof: shorthand test.
- [ ] 53.3.3 `list-style-image` `[~]` unless Phase 52 image paint makes it cheap. Pointer if deferred.

### 53.4 gates

- [ ] 53.4.1 Mapping + matrix. `make lint`, `make test`, `make golden`. Record tails.

## Dependencies

`pseudo_content.go` parseContentValue. List marker emission in `layout_flow.go`.

## Evidence

Counter golden. List-inside unit test.

## Out of scope

Full CSS Generated Content. Bookmark properties. `::marker` styling beyond type/position. GCPM running elements.

## Handoff

Next is Phase 54.
