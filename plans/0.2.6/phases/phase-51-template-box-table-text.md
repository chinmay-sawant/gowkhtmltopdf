# Phase 51: Template box, table, text

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 51
> **Status:** not started
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

- [ ] 51.1.1 `ResolvedStyle.WordSpacing` plus intern + inherit (`inheritableProps`). Apply in `applyTextGroup`. Proof: `go test ./internal/layout -run TestWordSpacingInherits`.
- [ ] 51.1.2 Consume in inline measure like `letter-spacing`. Proof: `TestWordSpacingWidensRuns`.
- [ ] 51.1.3 Matrix §2.3 row Implemented with `file:line`. Mapping flip.

### 51.2 visibility

- [ ] 51.2.1 Parse `visible` / `hidden` / `collapse`. `hidden` paints nothing, layout size unchanged. `collapse` on non-tables aliases `hidden` unless table-row collapse is implemented. Proof: `go test ./internal/layout -run TestVisibilityHidden`.
- [ ] 51.2.2 Descendants of `hidden` do not paint. Proof: nested test.
- [ ] 51.2.3 Matrix row Partial or Implemented.

### 51.3 caption-side

- [ ] 51.3.1 `caption-side: top | bottom`. Bottom paints after the table grid. Proof: `go test ./internal/layout -run TestCaptionSideBottom`.
- [ ] 51.3.2 `left` / `right` `[~]` unless a fixture needs them.
- [ ] 51.3.3 Matrix §2.5 row.

### 51.4 white-space

- [ ] 51.4.1 `pre-wrap`: preserve newlines and wrap. `pre-line`: collapse spaces, preserve newlines, wrap. Stop folding both to `pre` at `style_properties.go:1090`. Proof: `go test ./internal/layout -run TestWhiteSpacePreWrap`.
- [ ] 51.4.2 Existing `pre` / `nowrap` tests stay green.
- [ ] 51.4.3 Matrix §2.3 `white-space` row updated.

### 51.5 table-layout honesty

- [ ] 51.5.1 Matrix §2.5: `table-layout: fixed` Partial/Implemented lite, cite `layout_tables.go:45`. Proof: matrix text.
- [ ] 51.5.2 Golden or unit test that fixed layout uses the first-row / width hints the code already honors. If none exists, add one. Proof: test name.

### 51.6 gates

- [ ] 51.6.1 Mapping flips. `make lint`, `make test`, `make golden` if wrap/table paint changes. Record tails.

## Dependencies

Inline measure in `internal/layout/inline.go`. Table caption builder.

## Evidence

Tests named above. Matrix diffs.

## Out of scope

Full CSS2 table-layout algorithm. `empty-cells`. Caption left/right.

## Handoff

Next is Phase 52.
