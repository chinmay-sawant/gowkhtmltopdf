# Phase 61: Overflow, visibility, table

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 61
> **Status:** reopen (demote overflow-x/y, visibility, border-collapse still Partial)
> **Estimated effort:** L
> **Owner:** `internal/layout`
> **Depends on:** Phase 60
> **Unblocks:** honest table/overflow Implemented claims
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

`overflow` (single field), `table-layout`, `caption-side`, `vertical-align` can stay Implemented under their current claims. Reopen work is only:

- `overflow-x`, `overflow-y` (independent axes)
- `visibility: collapse` on table rows
- `border-collapse` conflict resolution beyond shared-grid lite

## Work order (code)

### overflow-x / overflow-y

1. Today both write one `ResolvedStyle.Overflow` (`setOverflowKeyword` in `internal/layout/style_properties.go`).
2. Split into distinct fields on `ResolvedStyle` in `internal/layout/style.go` (or an axis pair), update apply arms, and teach `overflow_clip.go` / sticky scrollport selection to honor axes.
3. Tests must show different `overflow-x` vs `overflow-y` behavior (new tests in `overflow_*_test.go` or `css_apply_test.go`).

### visibility: collapse

1. Today `hidesPaint` treats `collapse` like hidden (`internal/layout/inline.go`).
2. Implement table-row / row-group collapse in `internal/layout/layout_tables.go` (or table emit path): row contributes no height; cells not painted; geometry of later rows shifts.
3. Test: table with `tr { visibility: collapse }` reduces used table height vs `hidden`.

### border-collapse

1. Today: spacing suppressed + `emitCollapsedRowGrid` / first-visible border pick.
2. Implement adjacent border conflict resolution (width/style/color precedence) in the collapsed emitter path under `internal/layout/` table paint files.
3. Test: two adjacent cells with different border widths; winner matches CSS2.1 collapsed border rules for the cases you claim.

## Checklist

- [ ] 61.R.1 Implement overflow axes **or** keep Partial with notes that they alias one field.
- [ ] 61.R.2 Implement row collapse **or** keep Partial (paint-skip only) with notes.
- [ ] 61.R.3 Implement border conflict resolution **or** keep Partial with notes.
- [ ] 61.R.4 Flip packet + matrix + mapping recount for any Implemented promotions.
- [ ] 61.R.5 `go test ./internal/layout -run "TestOverflow|TestVisibility|TestBorderCollapse|TestTable|TestCaption"`; catalog `--check`; `make test` / `make lint`.

## Forbidden proofs

- Claiming Independent axes while both keywords write one string field
- `TestVisibilityHidden` checking only non-table paint skip for `collapse`

## Handoff

Phase 63 (`writing-mode`) is the next Partial honesty hole.
