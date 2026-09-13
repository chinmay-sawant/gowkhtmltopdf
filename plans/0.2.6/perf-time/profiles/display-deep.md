# Display list and box construction deep profile, committed 0.2.6 baseline

Read-only profiling for the `plans/0.2.6/perf-time/` plan. No production code
changed. One temporary probe test was created, run, and deleted (section 11).
No git command ran.

Goal under test: halve warm 500-page render time (1,228.72 ms, phase-7 warm
matrix) while keeping B/op at or below 234.92 MB.

- date: 2026-09-11, 19:05-19:13 IST
- go: go1.26.4 linux/amd64
- cpu: 13th Gen Intel(R) Core(TM) i7-13700HX, 24 CPUs
- os: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64
- host: 7.6 GiB RAM; load average 0.70 (1 min) when the captures started;
  three sibling agents were queued for the heavy-profile lock while this
  session held it
- fixture: `testdata/golden/benchmarks/templates/report.html.tmpl`
  sha256 `e3b5387bed117844c3502915cd3def253738959feb894435720f3532746e6974`
- workload: `BenchmarkPDFPages/generic` in `internal/convert/benchmarks_test.go`,
  `-test.benchtime=1x`, cpuprofile runs at `-test.count=5` (500p) and
  `-test.count=25` (50p), workdir `internal/convert`
- binary: `go test -c -o /tmp/opencode/perf-time-display/convert.test
  ./internal/convert`, build ID `0af59f748368a8ff306507da57dae332bff66216`
- heavy-profile lock: `mkdir /tmp/opencode/perf-time-profile.lock` succeeded
  immediately at 19:05:10, no wait. Released at 19:13:25 after all heavy work.
  Two other agents queued behind it (`/tmp/opencode/perf-time-style` waiter log
  was active).

Profile overhead caveat, stated once and used everywhere: the profiled 500p
runs report 1.72-2.03 s per iteration while the same tree measures 1,228.72 ms
in the unprofiled warm matrix. The 9.69 s of 500p samples cover 5 iterations,
so samples per iteration are about 1.6x the warm matrix's per-iteration wall
time. Shares below are trustworthy; absolute milliseconds are projections
(`share x 1,228.72 ms`), not measured per-phase wall times. The 50p capture
ran while the host was noisy (wall rows 158-398 ms against a 102.59 ms warm
matrix row), so 50p is used only to check that the build and style shares hold
their shape; its pagination share is lower (12.13 against 19.71 percent)
because the pagination fixpoint is superlinear in pages.

## 1. Phase attribution

Method: `go tool pprof -top -nodecount=8000 -nodefraction=0
-show_from=<root regex>` sums the flat samples of all stacks that pass through
the phase root. Script: `raw/method/phase-attribution.sh`; raw tables
`raw/500-phase-sums.tsv`, `raw/50-phase-sums.tsv`.

| phase (root) | 500p s | 500p % | 50p s | 50p % |
|---|---:|---:|---:|---:|
| parse and prepare | 0.13 | 1.34 | 0.07 | 1.15 |
| style resolution (`resolveStylesForLayoutContext`) | 2.94 | 30.34 | 2.00 | 32.79 |
| **box build + display list (`(*engine).build`)** | **2.36** | **24.36** | **1.58** | **25.90** |
| finalize chrome | 0.00 | 0.00 | 0.01 | 0.16 |
| outline and navigation metadata | 0.14 | 1.44 | 0.12 | 1.97 |
| pagination fixpoint and seals | 1.91 | 19.71 | 0.74 | 12.13 |
| paint (page content) | 0.61 | 6.30 | 0.40 | 6.56 |
| structure tree | 0.00 | 0.00 | 0.00 | 0.00 |
| PDF finalize, serialize, compress | 0.77 | 7.95 | 0.60 | 9.84 |
| GC mark workers | 0.38 | 3.92 | 0.22 | 3.61 |
| residual (not attributed) | 0.45 | 4.64 | 0.36 | 5.90 |
| total samples | 9.69 | 100.00 | 6.10 | 100.00 |

The build share is 24.36 percent now, against 24.1 percent in the pre-improve
500p profile (`plans/0.2.6/perf-improve/profiles/warm-pdf-cpu.md`, section 4).
The absolute sample cost fell from 3.12 s to 2.36 s, so the relative share
barely moved: every other phase shrank too. The context figure of 21.3 percent
matches the pre-improve 100p row (21.7 percent), not the 500p row.

## 2. Build phase attribution, flat and cumulative

`raw/500-layout2.tsv` and `raw/50-layout2.tsv` are the joined flat/cum
tables; source lines are from the tree at the commit profiled. Percent is of
9.69 s (500p).

| function | file:line | 500p flat | 500p cum | share | 50p cum |
|---|---|---:|---:|---:|---:|
| `(*engine).build` | `layout.go:1370` | 0.00 | 2.36 | 24.36% | 1.58 |
| `(*engine).flowChildren` | `layout_flow.go:204` | 0.05 | 2.36 | 24.36% | 1.58 |
| `(*engine).buildTable` | `layout_tables.go:10` | 0.00 | 2.25 | 23.22% | 1.52 |
| `(*engine).layoutTableGrid` | `layout_tables.go:84` | 0.01 | 1.83 | 18.89% | 1.23 |
| `(*engine).emitTableCells` | `layout_tables.go:288` | 0.00 | 1.01 | 10.42% | 0.67 |
| `(*engine).measureTableRows` | `layout_tables.go:656` | 0.00 | 0.81 | 8.36% | 0.56 |
| `(*engine).measureRowCells` | `layout_tables.go:715` | 0.01 | 0.80 | 8.26% | 0.55 |
| `(*engine).measureCellHeight` | `layout_tables.go:1214` | 0.03 | 0.79 | 8.15% | 0.54 |
| `(*engine).layoutCell` | `layout_measure.go:726` | 0.09 | 0.76 | 7.84% | 0.53 |
| `(*engine).emitCell` | `layout_tables.go:1247` | 0.05 | 0.74 | 7.64% | 0.60 |
| `(*engine).layoutInlineFloats` | `inline.go:156` | 0.01 | 1.00 | 10.32% | 0.79 |
| `(*engine).emitLine` | `inline.go:847` | 0.01 | 0.58 | 5.99% | 0.41 |
| `(*engine).collectAndPrepareInlineItems` | `inline.go:64` | 0.00 | 0.35 | 3.61% | (in 0.79) |
| `(*engine).collectInlineNode` | `inline_collect.go:86` | 0.06 | 0.35 | 3.61% | 0.34 |
| `(*engine).collectInlineText` | `inline_collect.go:102` | 0.05 | 0.29 | 2.99% | 0.29 |
| `(*engine).emitLineItems` | `inline.go:907` | 0.02 | 0.35 | 3.61% | 0.28 |
| `(*engine).splitTextByFace` | `inline_paint.go:1443` | 0.02 | 0.30 | 3.10% | 0.21 |
| `(*engine).primaryFaceRun` | `inline_paint.go:1521` | 0.05 | 0.30 | 3.10% | 0.27 |
| `(*engine).measureTableColumns` | `layout_tables.go:536` | 0.01 | 0.29 | 2.99% | (table path) |
| `(*engine).buildCell` | `layout_tables.go:1197` | 0.05 | 0.27 | 2.79% | 0.23 |
| `(*engine).measureCellMinMax` | `layout_measure.go:94` | 0.00 | 0.18 | 1.86% | 0.16 |
| `(*cellMeasure).walk` | `layout_measure.go` | 0.01 | 0.18 | 1.86% | (in 0.16) |
| `(*engine).add` | `layout.go:867` | 0.17 | 0.19 | 1.96% | 0.01 |
| `(*engine).emitBorderLine` | `layout_chrome.go:134` | 0.01 | 0.14 | 1.44% | (small) |
| `(*cellMeasure).measureElement` | `layout_measure.go:305` | 0.01 | 0.17 | 1.75% | (in 0.16) |
| `(*engine).emitInlineTextRun` | `inline_paint.go:432` | 0.01 | 0.05 | 0.52% | (small) |

Facts that fall out of the table:

- The build is a table build. `buildTable` is 2.25 s of `build`'s 2.36 s
  (95 percent). 500 tables, 10,500 rows and 52,500 cell boxes make up the
  document (probe, section 3).
- Two functions cost almost the same by different mechanisms:
  `measureCellHeight`/`layoutCell` 0.79 s is the no-emit height pass, and
  `emitCell` 0.74 s is the same content flowed again with ops on.
- The inline engine (`layoutInlineFloats` 1.00 s) is called on all three
  passes, so its 1.00 s is shared, not additional.
- `(*engine).add` is 0.17 s flat (1.75 percent). Its callers are
  `emitBorderLine` 0.13 s (68 percent), `emitInlineTextRun` 0.04 s
  (21 percent) and `emitCell` 0.02 s (11 percent). The old 490 ms `add` line
  does not exist on this tree. Appending 174,000 440-byte ops costs about
  34 ms per conversion in profile time (21 ms projected warm).
- `emitGridVerticals` 0.16 s, `emitGridTopEdges` 0.10 s,
  `emitGridBottomEdges` 0.01 s: the whole grid emission is 0.27 s
  (2.79 percent) for 118,000 ops. Emitting grid lines is cheap per op; the
  cost is that there are so many of them later.

## 3. Op inventory at 500 pages (temporary probe)

Probe: `internal/layout/zz_perf_time_op_probe_test.go`, package `layout`,
reuses the golden-test harness shape (`internal/convert/golden_test.go:596`
`collectStyleSheets`, A4 minus 10 mm margins, print media, backgrounds on),
then prints counts and bytes. Run with `ZZ_PAGES=500`. Output:
`raw/op-inventory-500.txt`, `raw/op-inventory-50.txt`. Op size is 440 bytes
(the phase-5 packing result).

| kind | count | % of ops | bytes at 440 B |
|---|---:|---:|---:|
| Line | 118,000 | 67.82 | 51.92 MB |
| Text | 53,500 | 30.75 | 23.54 MB |
| FillRect | 2,500 | 1.44 | 1.10 MB |
| StrokeRect / Image / LinkURI / Bullet | 0 | 0 | 0 |
| total | 174,000 | 100.00 | 76.56 MB |

Ids are monotonic 1 through 174,000, so the list is append-ordered and no
pagination clones exist in this fixture. Flags on the 174,000 ops:

| flag | count | reading |
|---|---:|---|
| `Width != 0` | 118,000 | exactly the Line ops, every one width 1.0 |
| `InkDescent != 0` | 53,500 | every Text op |
| `TextTransform != ""` | 53,500 | every Text op |
| `Bold` | 3,000 | 2,500 th plus 500 h1 |
| Fixed / Pinned / IsBackground / Positioned / XformSet / ZIndexSet | 0 | no repeated chrome, no fixed ops |
| Radius* / PaintOpacity / BlendMode / RotateDeg / URI / Alt / Image | 0 | no rounded, transformed, linked or replaced ops |
| StructElem | 0 | this fixture carries no PDF/UA structure pointers on ops |

Emitter inventory, from the probe's deepest-box ownership pass plus the
source of each emitter:

| emitter (source) | op kind | 500p count | per page | bytes |
|---|---|---:|---:|---:|
| `rowGridStroker.vline` (`layout_tables.go:941`) via `emitGridVerticals` (`:905`) | Line, vertical | 63,000 | 126 | 27.72 MB |
| `rowGridStroker.hline` (`:932`) via `emitGridTopEdges` (`:892`) | Line, horizontal | 52,500 | 105 | 23.10 MB |
| `rowGridStroker.hline` via `emitGridBottomEdges` (`:920`) | Line, horizontal | 2,500 | 5 | 1.10 MB |
| `emitLineItems` (`inline.go:907`) to `emitInlineText` to `emitInlineTextRun` (`inline_paint.go:432`) | Text | 53,500 | 107 | 23.54 MB |
| `emitCell` background add (`layout_tables.go:1255`) | FillRect | 2,500 | 5 | 1.10 MB |
| total | | 174,000 | 348 | 76.56 MB |

The document shape makes the counts checkable by hand. Each page is one
`<section>` with one table: 21 rows (1 thead plus 20 tbody) and 5 columns,
105 cell boxes per table, 500 tables. Per page:

- grid verticals: 21 rows x 6 column boundaries = 126
- grid horizontals: 21 rows x 5 segments plus 5 on the last row = 110
- cell backgrounds: 5 th backgrounds (td cells have no background)
- text: 105 cell texts plus h1 and p = 107

Everything else is zero. There is no repeated page chrome in this fixture,
which matters for one of the candidate ideas: header repeats and fixed
elements do not exist here, so composite chrome ops cannot help the benchmark.

Duplicated or compressible families:

1. **Table grid lines: 118,000 ops, 67.8 percent of the list, 51.92 MB.**
   These are 236 tiny 1 pt segments per page with identical color and width.
   They are geometrically distinct (6 boundaries x 21 rows; 5 segments x 21
   rows), not literal duplicates, so a "dedupe" pass finds nothing. They are
   compressible only as a batched representation: one record can carry a row's
   verticals (x list, y0, y1, color, width) and another its horizontals.
2. **Cell backgrounds: 2,500 ops, 1.4 percent.** Not a lever.
3. **Text: 53,500 ops, 30.7 percent.** One op per cell text plus two block
   texts. No two texts are identical (SKU, description, amount differ per
   row), so no content-level dedupe. Shrinking the record helps here instead.
4. **Repeated chrome: zero ops.** `repeatTableHeaders` still costs 0.13 s at
   500p because it scans for crossing tables; no table crosses a page, so no
   clones are emitted.

Per-op CPU cost, measured and projected:

| cost | measured | per op | projected warm |
|---|---:|---:|---:|
| append one Op (`layout.go:886`) | 0.170 s / 870k appends, 5 runs | 0.195 us profile | 21.5 ms/conversion (1.75% x 1,228.72) |
| emit one grid line (vertical + edge paths) | 0.27 s / 590k ops, 5 runs | 0.458 us profile | 34 ms/conversion (2.79%) |
| emit one text op (`emitInlineTextRun`) | 0.05 s / 267.5k ops | 0.187 us profile | 6 ms/conversion (0.52%) |
| paint one line (`drawLine`, `paint.go:1210`) | 0.19 s / 590k | 0.322 us profile | 24 ms/conversion (1.96%) |
| paint one text (`drawText`, `paint.go:1228`) | 0.17 s / 267.5k | 0.636 us profile | 22 ms/conversion (1.75%) |

The append itself is not the problem. One grid line costs about 0.46 us to
emit in profile time; 118,000 of them cost about 0.27 s to create and then
about 1 s of scanning and painting downstream (section 5).

## 4. Box and measurement attribution

Probe counts: 54,503 boxes, `unsafe.Sizeof(box{})` 304 bytes, 16.57 MB of box
memory. One h1, one p, one section and one table per page, 105 cell boxes per
table:

| box class | count | note |
|---|---:|---|
| cell/td | 50,000 | 500 pages x 20 rows x 5 cols |
| cell/th | 2,500 | 500 x 5 header cells |
| table/table | 500 | one per page |
| block/section, block/h1, block/p | 500 each | page wrappers and headings |
| block/html, block/body, block/#document | 1 each | document spine |

Rows are not boxes. `layoutTableGrid` stores `tableBox.rows` as
`[][]*box` slices of the cell boxes (`layout_tables.go:96`).

The build flows every table cell three times:

1. **Min/max pass.** `measureTableColumns` (`layout_tables.go:536`) calls
   `buildCell` (`:1197`) which calls `measureCellMinMax`
   (`layout_measure.go:94`), which walks the cell subtree through
   `cellMeasure.walk` and `measureElement`. Cost 0.29 s plus 0.18 s nested.
2. **Height pass with no ops.** `measureTableRows` (`:656`) calls
   `measureRowCells` (`:715`) which sets the final column width then calls
   `measureCellHeight` (`:1214`). That sets `e.noEmit` and runs the full
   recursive layout `layoutCell` (`layout_measure.go:726`) through
   `flowChildren` and `layoutInlineFloats`. Cost 0.81 s.
3. **Emit pass at the final position.** `emitTableCells` (`:288`) calls
   `emitCell` (`:1247`), which flows the same cell content again with
   emission on. Cost 1.01 s.

Passes 2 and 3 run the same content at the same width. The only difference is
`e.noEmit` and the final x/y origin. `emitCell` calls `flowChildren` for
0.62 s; `layoutCell` calls `flowChildren` for 0.61 s. That is the repeated
measurement: about 0.6 s of identical inline layout is computed twice, on top
of a 0.29 s min/max pass. Total cell-content work is about 2.11 s, 21.8
percent of the profile.

Per-box cost, projected: build cum 2.36 s is 24.36 percent of 1,228.72 ms, or
299 ms per conversion over 54,503 boxes, about 5.5 us per box, 5.7 us per
cell.

Inline item bookkeeping is where the repetition lands:
`collectAndPrepareInlineItems` 0.35 s, `emitLine` 0.58 s,
`collectInlineNode` 0.35 s, `collectInlineText` 0.29 s, `splitTextByFace`
0.30 s, `primaryFaceRun` 0.30 s, `inlineFontMetrics` 0.34 s cum. These run on
all three passes.

## 5. What op count costs after build

Functions that walk the whole op list, from the same profile:

| function | file:line | cum s | share | scales with |
|---|---|---:|---:|---|
| `paginateOps` (dispatch) | `paint_pagination_fixpoint.go:22` | 1.22 | 12.59% | op list |
| `paginationFixpoint` | `paint_pagination_fixpoint.go` | 0.32 | 3.30% | ops and pages |
| `settleBeforeAlways` | `paint_flow_breaks.go` | 0.32 | 3.30% | breaks x boxes, not ops |
| `snapCrossingTextOps` | `paint_pagination_fixpoint.go:135` | 0.17 | 1.75% | text ops |
| `normalizeTableRowGaps` | `paint_flow_tables.go:43` | 0.14 | 1.44% | row ops |
| `repeatTableHeaders` | `paint_flow_tables.go:361` | 0.13 | 1.34% | ops (scan only here) |
| `ensureFlowIndex` | `paint_flow_index.go:417` | 0.10 | 1.03% | boxes |
| `capTablePageBreaks` | `paint_pagination_seal.go:33` | 0.24 | 2.48% | line ops |
| `buildPageIndex` | `paint_flow_index.go:355` | 0.32 | 3.30% | all ops, 0.23 flat |
| `stripOrphanRowChrome` | `paint_flow_tables.go` | 0.13 | 1.34% | ops |
| `paintPages` | `paint.go:357` | 0.61 | 6.30% | ops |
| `(*pagePainter).paintOp` | `paint.go:462` | 0.41 | 4.23% | ops |
| `drawLine` | `paint.go:1210` | 0.19 | 1.96% | line ops |
| `drawText` | `paint.go:1228` | 0.17 | 1.75% | text ops |
| `sortPaintIndices` | `paint.go` | 0.05 | 0.52% | ops per page |
| `rowInkBand` | `paint_flow_tables.go:166` | 0.06 | 0.60% | row ops |
| `rowYBounds` | `paint_flow_tables.go` | 0.09 | 0.93% | cell ops |
| `boxInkExtent` | `paint_flow_breaks.go:311` | 0.08 | 0.83% | cell ops |
| `collectBorderSegmentOps` | `paint_pagination_seal.go:319` | 0.05 | 0.52% | line ops |
| `collectBodyNavigation` | `internal/convert/links.go:97` | 0.09 | 0.93% | all ops |

The per-op append cost is 1.75 percent. The per-op scan cost spread over the
rest of the pipeline is several times larger. Removing 68 percent of the ops
does not remove 68 percent of these functions (they still walk the remaining
entries) but it does remove the line-specific share, and the generic share
falls with the entry count.

## 6. Ranked architectural options

Expectations are profile shares projected onto the 1,228.72 ms warm row.
Every option must be re-measured with the phase-7 harness
(`scripts/bench-performance-recovery.sh --mode=internal-pdf-warm
--benchtime=1x --count=1`) and must keep `output-bytes` at 1,419,234 and the
golden corpus green unless the plan explicitly accepts an output change.
B/op must stay at or below 234.92 MB.

### Option 1: batched grid emission (composite grid record per row)

Replace the 236 individual `OpLine` segments per page with about 43 records:
one vertical-run per row (x list + y0/y1 + color + width), one horizontal-run
per row, one for the closing bottom edge. Entry count falls from 174,000 to
about 77,500 (-55 percent). The painter loops inside the record and can emit
the same PDF path operators in the same order, so output bytes can stay
identical.

- Expected recovery: 0.89 s of profile samples, 9.2 percent, about 113 ms
  warm. Breakdown: 0.59 s from generic per-entry scans (`buildPageIndex`,
  `snapCrossingTextOps`, `stripOrphanRowChrome`, `repeatTableHeaders`,
  `normalizeTableRowGaps`, `collectBodyNavigation`, paint dispatch) dropping
  with entry count; 0.22 s from grid emission itself; 0.08 s from `add`.
  `drawLine` (24 ms warm) stays: the same segments are still drawn.
- B/op: neutral if the ops array prealloc is unchanged (the composite payload
  is smaller than the 118,000 free slots it replaces only if the prealloc is
  also retuned). With a retuned prealloc, B/op falls.
- Contract risk: high but local. Pagination and seal predicates
  (`opOwnsBoxSide`, `isVerticalChromeForBox`, `collectBorderSegmentOps`,
  `capTablePageBreaks`, `stripOrphanRowChrome`, `stretchPaginatedChrome`,
  `splitCrossingRects`) all assume one border per op. Text order, page counts
  and link annots are untouched because text and URI ops do not change.
  StructElem is nil on this fixture, but tagged documents would need the
  composite to carry per-segment structure elements or stay out of tagging.
- Falsifiable probe: implement the vertical-run composite only. Require
  `output-bytes` 1,419,234 at 500p, `make golden` exit 0, entry count from the
  probe at or below 130,000, and `add` flat plus `emitGridVerticals` cum to
  fall by at least 0.15 s in profile time. Reject if page counts change on
  fixtures 10, 48, 56 or 60 (table-heavy).

### Option 2: stop re-flowing cell content a third time

Passes 2 and 3 at the same width are identical except for `noEmit` and the
origin. Reuse the pass-2 line layout in pass 3, or emit into a per-cell
scratch buffer during pass 2 and splice it in on commit. This targets the
0.62 s of `flowChildren` under `emitCell` and part of its inline work.

- Expected recovery: 0.4 to 0.6 s, 4 to 6 percent, about 50 to 75 ms warm.
  Lower bound is the inline item collection and line packing for pass 3
  (0.33 s of `layoutInlineFloats`) plus the per-cell flow setup.
- B/op: must use pooled scratch, not retained per-cell buffers. A passive
  cache holding 52,500 cell layouts would add tens of MB and is disallowed.
- Contract risk: very high. Op order must stay exactly cell-content ops then
  that row's grid ops; `opStart`/`opEnd` per box must stay correct for
  pagination remaps (`remapBoxOpRanges`), transform stamping
  (`stampBoxTransforms`) and `emitCell`'s own clip pass over its index range
  (`layout_tables.go:1298-1305`). Any splice bug shows up as shifted ops or
  lost clips, not as obvious text errors.
- Falsifiable probe: a test-only counter on `layoutInlineFloats` keyed by
  node, which today counts 3 per cell. After the change, require 2 per cell,
  identical `output-bytes`, and `make golden` exit 0. Reject if any fixture
  with rowspan, nested tables or transforms changes.

### Option 3: compact Op record with side payload

`Op` is 440 bytes. The common shapes use a fraction of it: lines carry
`ID/Kind/X/Y/W/H/Width/RGB`; text adds `Text/Font/Size/TextTransform/
InkDescent`. `Radius` (10 floats), `Xform`, `Image`, `Alt`, `URI`,
`BlendMode`, `StructElem` are rare or zero here. Moving rare payloads to side
tables shrinks the record (a 200-byte target is plausible) and the 202,533
slot prealloc from 89.1 MB to about 40 MB.

- Expected recovery: 0.25 to 0.35 s, 2.5 to 3.5 percent, about 30 to 45 ms
  warm: less append copy work (0.17 s flat today), less zeroing
  (`memclrNoHeapPointers` 0.36 s, 0.22 s of it chunked zeroing of large
  makes), and fewer pointers for GC scan (`gcDrain`/`scanObjectsSmall`).
- B/op: falls by about 45 to 50 MB at 500p (prealloc shrink). This is the
  option that most directly protects the B/op ceiling while other options
  add payload arrays.
- Contract risk: low for output (representation only), high for effort.
  Every reader indexes `Op`. Field moves are mechanical but broad; a missed
  side-table lookup silently reads a zero `Radius` or `Xform`.
- Falsifiable probe: `unsafe.Sizeof(Op{})` at or below 256 and B/op at 500p
  below 234.92 MB, with `output-bytes` 1,419,234 and `make test` exit 0.

### Option 4: advance and line-metric memoization

`primaryFaceRun` (0.30 s), `inlineFontMetrics` (0.34 s cum) and
`splitTextByFace` (0.30 s) re-derive advances from the font tables every pass.
The 2,500 header cells repeat identical text ("Line", "SKU", ...) 500 times,
and every rune advance is a pure function of (face, size, rune). Cache the
advance per rune and the face run per (text, style); unitless key hits are
measurable.

- Expected recovery: up to 0.3 s, 3 percent, about 37 ms warm. The repeated
  header text alone is 4.8 percent of cells, worth about 12 ms; rune advance
  caching is the larger part. Row memoization keyed by text is worthless here
  because every body row is unique, which the probe proves.
- B/op: caches must be bounded (per conversion or LRU). An unbounded
  per-rune map is stable in size (few thousand runes) and cheap.
- Contract risk: low. Advances and font metrics do not affect pagination
  unless a cached number differs from the uncached one; the probe is a
  character-for-character text-needle comparison.
- Falsifiable probe: count cache hits with a temporary counter; require
  `primaryFaceRun` cum to fall below 0.15 s and `splitTextByFace` below
  0.15 s at 500p with identical ordered text needles.

### Option 5: per-page op arenas

Give each page its own op slice (or chunk) during build instead of one flat
slice, so pagination can stop re-scanning global ops and paint reads are
page-local. Entry count does not change.

- Expected recovery: below 1 percent, about 10 ms. Appends are already
  amortized, and the phase-3/4 work removed most append growth. The benefit is
  locality for `buildPageIndex` and `paintPages`, which the profile cannot
  separate from other cache effects.
- B/op: neutral if arenas are pooled via `Workspace.Release`; slightly higher
  if each page slice grows independently.
- Contract risk: medium. `opStart`/`opEnd` indices are global and pagination
  splits and shifts use global indexes. Page assignment is not known at build
  time for content that spans pages, so arenas need a two-level index.
- Falsifiable probe: measure `buildPageIndex` flat and the 500p warm row
  before and after; reject if the delta is inside run-to-run noise.

### Option 6: composite chrome ops and lazy construction (not applicable here)

`repeatTableHeaders` costs 0.13 s at 500p even though no table crosses a page,
so no header clones exist; `Fixed`/`Pinned` op counts are zero. Emitting one
chrome op per repeated element and referencing it per page would help
documents with crossing tables, not this benchmark. Lazy op construction for
off-page content has nothing to skip: every op in the list paints. Keep both
ideas for the corpus, do not credit them in the 2x estimate.

- Falsifiable probe: count `Pinned` and `Fixed` ops and header clones with the
  probe on the target document; if the count is zero, the option is worth
  zero for that document.

### Option ordering

1. Grid batching: about 113 ms, -55 percent entries, high local risk.
2. Third-pass elimination: about 50 to 75 ms, very high risk.
3. Op compaction: about 30 to 45 ms, low output risk, large B/op win.
4. Rune advance memoization: about 37 ms, low risk, small B/op cost.
5. Per-page arenas: under 10 ms, no output risk, mostly a scaffold for other
   work.
6. Chrome/lazy: zero on this fixture by measurement.

## 7. The ceiling for a 2x cut

Box construction plus display list is 24.36 percent of the warm 500p profile,
about 299 ms of 1,228.72 ms. If a perfect redesign deleted all of it and
nothing else changed, the warm row would be about 930 ms, a 1.32x speedup.
That is the hard ceiling for anything done inside `internal/layout`'s build
phase alone.

A 2x cut needs to remove 614 ms. The options above sum to about 113 + 63 + 38
+ 37 + 10 = 261 ms at their midpoints (about 1.27x). The remaining roughly
350 ms has to come from phases this probe did not chase:

- style resolution: 373 ms warm share. `applyRawToUsed` 1.69 s cum,
  `applyRestProps` 1.07 s, `applyStyleProp` 0.64 s, `inheritProps` 0.33 s,
  style interning about 0.5 s in profile samples. Halving it is worth about
  186 ms.
- pagination fixpoint and seals: 242 ms warm share. Grid batching already
  claims the op-scan part; the `settleBeforeAlways`/`shiftBoxesForForcedBreak`
  part (0.32 s here, previously measured as breaks x boxes) is separate work.
- PDF finalize and compression: 98 ms warm share, mostly `flate` on 1.42 MB
  of output. Halving it is worth about 49 ms and would change output bytes.
- paint: 77 ms warm share; `drawLine`/`drawText` are irreducible while the
  PDF must contain the same operators.

Two honest routes to 2x:

1. **Serial compounding.** Grid batching plus pass elimination plus
   compaction plus a serious style cut (roughly half) plus a pagination cut
   (roughly half) plus deflate tuning add to about 550-650 ms. Every phase
   must contribute; no single lever is close.
2. **Parallel page sections.** The benchmark is 500 independent `<section>`
   elements, each with `page-break-before: always`. Concurrent layout of
   sections is the only mechanism with a shot at 2x on its own, because it
   can compress style, build and paint together. It requires per-section
   workspaces (`Workspace` already exists), globally unique op IDs, and a
   pagination merge; this probe did not prototype it, so treat it as a
   hypothesis with a falsifiable probe (two sections on two goroutines,
   identical `output-bytes` and page order).

## 8. Measurements versus projections

Measured in this session:

- phase tables 500p and 50p (`raw/500-phase-sums.tsv`, `raw/50-phase-sums.tsv`);
- flat and cumulative function tables (`raw/500-layout2.tsv`,
  `raw/50-layout2.tsv`, `raw/500-cpu-top-flat-full.txt`, list and peek views
  under `raw/lists/` and `raw/peeks-*`);
- op counts: 174,000 ops, 118,000 Lines (63,000 vertical, 55,000 horizontal,
  all width 1.0), 53,500 Texts, 2,500 FillRects, 76.56 MB at 440 B
  (`raw/op-inventory-500.txt`, `raw/op-inventory-50.txt`);
- exactly 10x scaling from 50p to 500p in ops (17,400 to 174,000) and boxes
  (5,453 to 54,503);
- `output-bytes` 1,419,234 in all 500p profile rows (`raw/500-run.txt`);
- `add` flat 0.17 s with callers 68 percent `emitBorderLine`, 21 percent
  `emitInlineTextRun`, 11 percent `emitCell`;
- the three-pass shape: `layoutCell` 0.76 s, `emitCell` 0.74 s,
  `measureTableColumns`/`buildCell` 0.29/0.27 s, all measured.

Projections, labeled as such:

- all option milliseconds (share applied to 1,228.72 ms);
- the claim that entry-count-driven savings transfer proportionally to
  `buildPageIndex`, `snapCrossingTextOps` and friends;
- B/op outcomes of compaction and arena pooling;
- the parallel-section route, which has no prototype.

Known limits:

- CPU profiles cannot be sliced per iteration, so the 500p table includes the
  cold first iteration (about 19 percent of row time before this tree; the
  first profiled row was 1.73 s against 1.72-2.03 s later, so cold is not
  separable here).
- The profiled runs are 1.6x wall per iteration. Profiler overhead and host
  sharing are not separable in these numbers.
- Calls-per-function counts are read from the probe and source, not from the
  CPU profile; Go CPU profiles carry no call counts.

## 9. Raw evidence index

Under `plans/0.2.6/perf-time/profiles/raw/`:

- `500-run.txt`, `50-run.txt`: benchmark rows with `output-bytes` and B/op
- `500-phase-sums.tsv`, `50-phase-sums.tsv`: phase attribution tables
- `500-build-sums.tsv`, `50-build-sums.tsv`: build sub-phase table
- `500-layout2.tsv`, `50-layout2.tsv`: joined flat/cum function tables
- `500-cpu-top-flat(-full).txt`, `500-cpu-top-cum(-full).txt`, same for 50
- `op-inventory-500.txt`, `op-inventory-50.txt`: probe output
- `lists/`: `pprof -list` source-line views for the cited functions
- `peeks-*.txt`: caller/callee views for `add`, `flowChildren`,
  `layoutInlineFloats`, `paginateOps`, `paintPages`, `drawPageOp`,
  `buildPageIndex`, `memclrNoHeapPointers`
- `method/phase-attribution.sh`, `method/list-views.sh`: the exact scripts

Scratch copies of the `.pprof` binaries and the test binary stay in
`/tmp/opencode/perf-time-display/` (not committed): `500-cpu.pprof`,
`50-cpu.pprof`, `convert.test`.

## 10. Probe cleanup

The temporary probe `internal/layout/zz_perf_time_op_probe_test.go` was
deleted. Verified: `ls internal/layout/ | grep -c zz_perf_time` prints 0. No
production file was modified. The heavy-profile lock was released with
`rmdir` at 2026-09-11T19:13:25+05:30.
