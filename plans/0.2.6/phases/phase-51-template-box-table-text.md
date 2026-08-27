# Phase 51: Template box, table, text

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 51
> **Status:** in progress (word-spacing, visibility, caption-side, pre-wrap, table-layout test landed; full gates open)
> **Estimated effort:** 4-6 days
> **Owner:** `internal/layout`
> **Depends on:** Phase 50 for lengths if `word-spacing` uses `marginLen`
> **Unblocks:** Phase 52

---

## Overview

These properties show up in invoices, contracts, and tables. They have no apply arm today, or they lie.

- `word-spacing`: matrix §2.3 Not implemented. Absent from `applyTextGroup`.
- `visibility`: no field. `hidden` should skip paint and keep the box.
- `caption-side`: captions always above (`layout_tables.go:49-57`).
- `white-space: pre-wrap` / `pre-line` become `pre` (`style_properties.go:1086-1092`).
- `table-layout: fixed` is already consumed lite (`layout_tables.go:45`). Matrix is wrong.

## Goals

Land the missing text/box/table properties reports already write. Fix the matrix for `table-layout`.

## Checklist

### 51.1 word-spacing

- [x] 51.1.1 `ResolvedStyle.WordSpacing` intern + inherit. Proof: `TestWordSpacingInherits`.
- [x] 51.1.2 Consume in inline measure. Proof: `TestWordSpacingWidensRuns`.
- [x] 51.1.3 Matrix §2.3 Implemented. Mapping `--write` flipped `word-spacing` to partial/implemented family.

### 51.2 visibility

- [x] 51.2.1 `hidden`/`collapse` skip paint, keep size. Proof: `TestVisibilityHidden`.
- [x] 51.2.2 Nested hidden content not painted. Proof: same test.
- [x] 51.2.3 Matrix visibility Partial.

### 51.3 caption-side

- [x] 51.3.1 Bottom caption below the grid. Proof: `TestCaptionSideBottom`.
- [~] 51.3.2 `left` / `right` not implemented.
- [x] 51.3.3 Matrix caption-side Partial.

### 51.4 white-space

- [x] 51.4.1 `pre-wrap` / `pre-line` no longer fold to `pre`. Proof: `TestWhiteSpacePreWrap`.
- [x] 51.4.2 Existing `TestWhiteSpacePre` green (`go test ./internal/layout`).
- [x] 51.4.3 Matrix white-space row updated.

### 51.5 table-layout honesty

- [x] 51.5.1 Matrix table-layout Partial lite. Proof: matrix row.
- [x] 51.5.2 `TestTableLayoutFixed` / `TestTableLayoutFixedIgnoresContentMax`.

### 51.6 gates

- [x] 51.6.1 Mapping `--check`. `make lint`/`test`/`golden` green 2026-08-27.

## Dependencies

Inline measure in `internal/layout/inline.go`. Table caption builder.

## Evidence

Tests named above. Matrix diffs.

## Out of scope

Full CSS2 table-layout algorithm. `empty-cells`. Caption left/right.

## Handoff

Next is Phase 52.
