# Phase 61: Overflow, visibility, table

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 61
> **Status:** complete (honest: independent overflow axes, table row collapse, border-collapse conflict resolution Implemented)
> **Estimated effort:** L
> **Owner:** `internal/layout`
> **Depends on:** Phase 60
> **Unblocks:** honest table/overflow Implemented claims
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

`overflow`, `overflow-x`, `overflow-y`, `visibility`, `border-collapse`, `table-layout`, `caption-side`, `vertical-align` are Implemented with real layout and paint consumers.

## Work order (code)

### overflow-x / overflow-y

1. Split into distinct fields `OverflowX` and `OverflowY` on `ResolvedStyle` in `internal/layout/style.go`, updated `setOverflowKeyword` in `internal/layout/style_properties.go`.
2. Tests: `TestOverflowAxesIndependent` in `css_apply_test.go`.

### visibility: collapse

1. Implemented table row/row-group collapse in `internal/layout/layout_tables.go`: row contributes no height, cells not painted, geometry of subsequent rows shifts up.
2. Tests: `TestTableRowVisibilityCollapseReducesHeight` in `table_collapse_grid_test.go`.

### border-collapse

1. Implemented adjacent border conflict resolution (`resolveBorderConflict` with `borderStyleRank`) in `internal/layout/layout_tables.go`.
2. Tests: `TestCollapsedTableBorderConflictWiderWins` in `table_collapse_grid_test.go`.

## Checklist

- [x] 61.R.1 Implement overflow axes (`OverflowX`/`OverflowY` on `ResolvedStyle`). Proof: `internal/layout/style.go:139`, `style_properties.go:99`, `TestOverflowAxesIndependent`.
- [x] 61.R.2 Implement row collapse for table rows. Proof: `internal/layout/layout_tables.go:321`, `TestTableRowVisibilityCollapseReducesHeight`.
- [x] 61.R.3 Implement border conflict resolution. Proof: `internal/layout/layout_tables.go:896`, `TestCollapsedTableBorderConflictWiderWins`.
- [x] 61.R.4 Flip packet + matrix + mapping recount for Implemented promotions. Proof: `mapping.json` and `compatibility-matrix.md` §2.1, §2.3, §2.5.
- [x] 61.R.5 `go test ./internal/layout -run "TestOverflow|TestVisibility|TestBorderCollapse|TestTable|TestCaption"`; catalog `--check`; `make test` / `make lint`. Proof: all exit 0.

## Forbidden proofs

- Claiming Independent axes while both keywords write one string field
- `TestVisibilityHidden` checking only non-table paint skip for `collapse`

## Handoff

Phase 63 (`writing-mode`) is the next Partial honesty hole.
