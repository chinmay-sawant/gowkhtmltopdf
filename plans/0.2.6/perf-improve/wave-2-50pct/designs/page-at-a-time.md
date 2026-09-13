# Design: production page-at-a-time layout

Read-only design for PERF3-18..24. Evidence boundary: HEAD `ca761bb1`
(`perf: halve warm render time for 0.2.6`), Snapshot M, and current source.
No production code changed while writing this note.

## Plain words

Today the engine draws the whole document in memory first, then paints it,
then writes the PDF. For the 500-page report that means 66,500 drawing
records and 54,503 boxes live at once. That backing store is about 42 MB of
`[]Op` capacity, 16.6 MB of boxes, and 8.5 MB of table grid segments, and it
is why one conversion still allocates 163 MB.

Page-at-a-time means: layout one independent block, paginate it, paint it
into the shared PDF document, copy out the tiny metadata convert still needs
(headings, ids, page names), then `Workspace.Release` so the next block
reuses the same `[]Op` array. Peak live drawing state becomes one page, not
500.

This is not certified page islands. Islands are a private benchmark flag
(`convert.NewBenchmarkPDFRequest`, `internal/convert/convert.go:123-135`)
and must stay off for every published number. The production detector has
to work from CSS and box kind, not from the report fixture marker.

## Why this is the B/op half

`B/op` is cumulative malloc traffic inside one conversion. A `sync.Pool`
across conversions does not help a `-benchtime=1x` row. Reusing one
`[]Op` / box scratch inside the conversion does.

Arithmetic on the current 500-page report:

| bucket | today | page-at-a-time if one page is ~1/500 |
|---|---:|---:|
| `[]Op` capacity | ~42 MB (`layout.go:1038-1044`, `estimateOpCapacity` at `:1274-1305`) | ~0.08 MB reused |
| used ops | 28.7 MB (66,500 x 432) | ~0.06 MB reused |
| `GridSeg` | 8.5 MB | reused |
| box tree | 16.6 MB (54,503 x 304, `layout.go:1335-1393`) | reused if boxes are arena-reset or pooled in Workspace |
| PDF content `Grow` | ~21 MB for 500 buffers (`paint.go:405`) | one buffer if flate-and-release also ships |
| HTML tree | ~7 MB | still the full tree unless sections are detached |

The HTML tree stays. Style intern stays at 221,208 B. Compressed page copies
stay until phase 6. The drawing-list family is the 80 MB that has to move
for a 81.5 MB target.

`Workspace` already exists (`layout.go:312-345`) and is used only by the
island path (`page_islands.go:53-73`). Generic `convert.Run` still calls
`layout.LayoutContext` with `release=nil` (`convert.go:640-648`).

## Detector (must fail closed)

A candidate is an ordered, gap-free run of in-flow block children of `body`
where each candidate:

- has `page-break-before: always` except the first
- has `page-break-inside: avoid`
- is not floated, not `position: absolute|fixed`, not `display: none`
- is not a flex or grid item
- lays out to at most one page of content height after the first layout of
  candidate 0, within 0.1 pt
- does not need a later Y shift from a sibling (the 499 `beforeAlways`
  breaks on this fixture are exactly this shape)

Anything else, including one failing candidate, keeps today's single
`LayoutContext`. The report fixture (`testdata/golden/benchmarks/templates/report.html.tmpl:14-18,53-67`)
is the motivating case, not the only allowed case.

Do not key the detector on `benchmark-page` class names or the HTML comment
marker in `page_islands.go:16`. Those stay island-only.

## Lifecycle per candidate

1. `layout.WithWorkspace` on a view that contains only that section, with
   the document stylesheets (same cascade as today).
2. `layout.PaintContext` into the shared `pdf.Document`.
3. Copy `PageNames`, headings, and a compact navigation projection off the
   `Result`. Convert already dropped the full display list from Assemble
   except those three (`paint.go:238-243`, `convert.go:663-667`).
4. `workspace.Release(result)`.
5. Next candidate reuses `workspace.ops`.

`CloneResult` (`layout.go:210-232`) is the wrong tool here. It copies the
whole list.

## What still has to be true after the stitch

- Page count equals candidate count on the report fixture (500).
- Ordered text needles unchanged (`SKU-001-001` before `SKU-500-020`).
- `output-bytes` 1,419,234.
- Outline heading page numbers match today's `DocumentPage` view.
- Table header repeat, sticky, and split-crossing stay on the per-candidate
  `Paint` so a future non-independent document that somehow passes the
  detector still paginates locally. If a candidate paints more than one
  page, abort the fast path for the whole document and fall back. Falling
  back after painting candidate 0 requires discarding those PDF pages or
  not painting until candidate 0 is proven one page. Prefer: measure
  candidate 0 with `noEmit` or a disposable document first.

## Probe that ships or kills the phase

New test file, existing tests untouched:

- 5-section report vs one 5-section `LayoutContext`: date-normalized PDF
  bytes equal, page count equal, ordered text equal.
- 500-section generic `NewPDFRequest` (not islands): `output-bytes`
  1,419,234, `B/op` at or below 90 MB on a 1x capture.
- A document with a table that spans two pages must take the fallback and
  match today's PDF.

If the 5-section bytes differ, record the first differing operator and do
not ship.

## Interaction with later phases

Phase 5 (chrome clone) needs this detector. Phase 6 (flate-and-release)
needs per-page paint so a content buffer can die. Phase 6 overlap
(layout N+1 while compressing N) is PERFT-25, which was never attempted
because it was gated on the rejected parallel merge. Page-at-a-time unblocks
it without merging display lists.
