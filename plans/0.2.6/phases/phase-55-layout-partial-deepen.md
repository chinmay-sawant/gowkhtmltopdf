# Phase 55: Layout Partial deepen (fixture-driven)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 55
> **Status:** not started
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

- [ ] 55.1.1 `flex-flow` expands to `flex-direction` + `flex-wrap`. Proof: `go test ./internal/layout -run TestFlexFlowShorthand`.
- [ ] 55.1.2 `place-content` / `place-items` / `place-self` expand to align/justify pairs. Proof: `TestPlaceShorthands`.
- [ ] 55.1.3 `grid-template` / `grid` shorthands Partial parse into existing template fields. Proof: `TestGridTemplateShorthand`. Skip masonry values.

### 55.2 fixture-gated layout

- [ ] 55.2.1 `align-content: stretch` only if a named fixture fails. Proof: fixture name + test, or `[~] no failing fixture after 54`.
- [ ] 55.2.2 `column-rule` lite (solid stroke between columns) only if a named fixture fails. Proof: fixture or `[~]`.
- [ ] 55.2.3 `inline-grid` inline-level only if a named fixture fails. Proof: fixture or `[~]`.
- [ ] 55.2.4 Float wrap leftover from pending-02 only with a vendored wiki-like or report fixture, not live Wikipedia. Proof: fixture path.

### 55.3 file size

- [ ] 55.3.1 Do not grow `paint_flow.go` / `paint_pagination.go` past the ~2000 line cap. Extract a cohesive file in `internal/layout` if a change would. Proof: `wc -l` on touched files.

### 55.4 gates

- [ ] 55.4.1 Mapping Partial notes updated, not silently flipped to Implemented. `make lint`, `make test`, `make golden`. Record tails.

## Dependencies

Existing flex/grid/multicol engines. Pending-02 / 07 as pointers, not checklists to copy.

## Evidence

Named fixtures only. Chrome PDFs are canaries.

## Out of scope

Chrome layout tests. Full Flex/Grid. Subgrid joint intrinsic. Masonry L3.

## Handoff

Next is Phase 56.
