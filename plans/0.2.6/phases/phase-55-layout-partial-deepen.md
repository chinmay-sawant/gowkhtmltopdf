# Phase 55: Layout Partial deepen (fixture-driven)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 55
> **Status:** complete for 55.1 shorthands; 55.2 fixture-gated `[~]`
> **Estimated effort:** as required by failing fixtures, cap one week unless amended
> **Owner:** `internal/layout` flex.go, grid.go, multicol.go, float.go
> **Depends on:** Phases 49-54 shipped or explicitly `[~]`
> **Unblocks:** Phase 56

---

## Overview

Flex, grid, float, and multicol already have print subsets. This phase does not "finish" them. Open a row only when a *named* fixture or golden fails after 49-54, or when a shorthand is a pure parse expansion with tests.

Copy this rule onto every row: fixture name, expected box, proof command.

Known leftovers, not automatic work:

- `align-content: stretch` packs at start
- `flex-flow`, `place-content`, `place-items`, `place-self` missing
- `grid` / `grid-template` / `grid-auto-columns` / `grid-auto-rows` missing
- `column-rule*` missing
- `display: inline-grid` is not inline-level (`layout_flow.go:85-87`)
- Live infobox wrap from `plans/0.2.0/phases/pending-phase-items/02-openweb-css-residuals.md`

Joint subgrid intrinsic and Grid L3 masonry stay out.

## Goals

Close only evidenced Partial holes. Leave the rest Partial in the matrix.

## Checklist

### 55.1 shorthand expansions (cheap, no new layout)

- [x] 55.1.1 `flex-flow`. Proof: `TestFlexFlowShorthand`.
- [x] 55.1.2 `place-*`. Proof: `TestPlaceShorthands`.
- [x] 55.1.3 `grid`/`grid-template`. Proof: `TestGridTemplateShorthand`.

### 55.2 fixture-gated layout

- [~] 55.2.1 No named failing fixture for align-content stretch.
- [~] 55.2.2 No named failing fixture for column-rule.
- [~] 55.2.3 No named failing fixture for inline-grid.
- [~] 55.2.4 No vendored float-wrap fixture this session.

### 55.3 file size

- [~] 55.3.1 Files already 2367 / 2244 lines. This slice did not grow them.

### 55.4 gates

- [x] 55.4.1 Mapping Partial. `make lint`/`test`/`golden` green.

## Dependencies

Existing flex/grid/multicol engines. Pending-02 / 07 as pointers, not checklists to copy.

## Evidence

Named fixtures only. Chrome PDFs are canaries.

## Out of scope

Chrome layout tests. Full Flex/Grid. Subgrid joint intrinsic. Masonry L3.

## Handoff

Next is Phase 56.
