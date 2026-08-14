## Summary

Fixes table colspan nesting (document order + no measure-pass leaks), paints `tr` row backgrounds onto transparent cells, and composites `rgba()` fills against white so alpha bands look translucent instead of solid dark blue.

---

## Motivation / context

User reports after prior fidelity PR:

- **fixture-10**: colspan header / nested table borders and heights looked wrong; nested “1×” lines appeared out of order.
- **fixture-14**: KPI rows (`.good`/`.warn`/`.bad` on `<tr>`) had no green/yellow/red cell fills; `.alpha` `rgba(15,58,95,0.15)` painted solid dark blue.

---

## Changes

### Nested tables / colspan (`layout.go`)

- Restore caller `noEmit` in `buildCell` so nested tables do not emit during outer measure.
- `flowChildren`: lay out blocks and inlines in **document order** (text before nested table in the same cell).
- `border-collapse: collapse` → zero `border-spacing` for tighter shared edges.

### Row backgrounds

- `cellBG`: if the cell has no background, use the parent `table-row` background (browser-like show-through for `tr.good` etc.).

### rgba fills (`paint.go`)

- Pre-composite `Alpha ∈ (0,1)` fills against white instead of relying on fragile PDF `/opacity` ExtGState.

### Tests / samples

- `TestRowBackgroundShowsThroughCells`, `TestRGBABackgroundCompositesLight`, nested order assertions
- Golden page envelopes for fixtures 02/16 relaxed after denser collapse spacing
- `make samples` regenerated

---

## Test plan

- [x] `make test`
- [x] `make lint`
- [x] `make samples`
- [x] Manual content-stream checks: fixture-14 green/yellow/pink fills; alpha band ~0.86 light blue-gray; fixture-10 Gateway above nested X200 lines

---

## Related issues

- Relates to #2 (rendering quality epic)
- Relates to #5 (CSS/table fidelity)

---

## PR metadata checklist (author)

- [x] Self-assigned
- [x] Labels: bug, enhancement
- [x] Related issues filled
