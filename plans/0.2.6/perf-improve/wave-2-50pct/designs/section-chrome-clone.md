# Design: repeated-section chrome clone

Read-only design for PERF3-25..29. Evidence boundary: HEAD `ca761bb1`,
Snapshot M, `internal/layout/layout_tables.go`, and
`testdata/golden/benchmarks/templates/report.html.tmpl`. No production
code changed while writing this note.

## Plain words

The 500-page report is 500 invoices that share one table shape. Every page
has the same header cells ("Line", "SKU", "Description", "Quantity",
"Amount"), the same collapsed border grid, and the same heading style.
Only the body cell strings change (SKU, description, quantity, amount).

Today the engine still flows every cell three times
(`buildCell` / `layoutCell` / `emitCell` in `layout_tables.go:1216-1303`)
for all 52,500 cells. Page-at-a-time (phase 4) reuses the backing array.
It does not skip that work. This phase skips the repeated chrome.

Chrome here means geometry that does not depend on the unique cell text:
column widths if they are table-layout or nowrap-and-identical-font,
header row boxes, collapsed border `OpGridRun` segments translated in Y,
and fill rects for header backgrounds.

## Why PERFT-12 stays rejected and this is different

PERFT-12 tried to reuse pass 2 of the *same* cell as pass 3. That has to
reproduce `Op.ID`, `opStart`/`opEnd`, clip, and transform stamps on every
cell. No equivalence probe completed. This phase does not splice pass 2
into pass 3 for arbitrary cells.

It clones a *finished* section display list and rewrites the unique text
ops, then re-measures only those text ops whose width could change a
column. On this fixture every `td` is `white-space: nowrap` at 9pt sans
(`report.html.tmpl:7-46`), so a longer SKU can widen a column. Column
widths must be taken from a full measure of that section, or from a
running max across sections. The cheap clone is illegal if any unique
string overflows the cloned column.

## Detector on top of page-at-a-time

After phase 4's independent-block detector:

1. Hash the section DOM with text nodes replaced by a hole token, plus
   the computed style pointer identity of each element.
2. Sections that share the hash are a clone group.
3. Layout section 0 fully (three cell passes, unchanged).
4. For section k>0, clone ops/boxes from section 0, add
   `k * pageHeight` to Y, then replace text on the unique cells.

Abort the clone group and fully layout that section if:

- column min-content of unique text exceeds the cloned column
- row height would change (wrapped text, images, rowspan)
- the section has transforms, floats, or absolutely positioned boxes
- `border-collapse` is not `collapse`, or grid segments are not a pure Y
  translate of section 0

The 500-page report is one clone group if column widths are stable. They
are not automatically stable: SKU and amount strings grow. A running max
column width from a first measure pass over unique strings, followed by
one geometry layout, then 499 text patches, is the honest algorithm.
Two-pass over 500 sections is still far cheaper than 500 full table
layouts if pass 1 only measures the unique strings.

## IDs, ranges, and output

Cloned ops need new `Op.ID` values in document order. `opStart`/`opEnd`
on cloned boxes must match the cloned op index ranges. Painters must see
the same kind order as a full layout: cell fills, cell text, then that
row's `OpGridRun` (`grid_run.go:8-11`).

`output-bytes` 1,419,234 is the kill switch. Date-normalized PDF of clone
vs full layout on 5 sections must be byte-identical before any 500-page
claim.

## Probe that ships or kills the phase

New tests only:

- Counter: full `emitCell` runs once per clone group, not once per
  section, on the report shape.
- Ordered text needles still list every SKU in order.
- 5-section clone PDF equals 5-section full layout after date
  normalization.
- A section whose unique string overflows the cloned column takes the
  full layout path and still matches today's PDF.

If the 5-section bytes differ, record the first operator delta and do
not ship. Time target for the phase is not "clone is faster". It is
warm 500-page time moving toward the 366.74 ms line when combined with
phase 4.

## What this does not do

- It does not wire `internal/layout/parallel.go`. That prototype merged
  500 full layouts and gained 5.4% (`plans/0.2.6/perf-time/results/phase-6/concurrency.md`).
- It does not reuse pass-2 inline layout of a cell as pass 3.
- It does not turn on page islands.
