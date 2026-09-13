# Warm PDF path CPU attribution after perf-improve (0.2.6 perf-time)

Task: profile the committed post-perf-improve baseline, produce a phase
attribution with source lines, measure GC headroom, delta against the
pre-improve profile, and rank candidates for halving warm render time while
keeping `B/op` at or below current values.

Read-only: no production or test code changed. No git command ran. Scratch
under `/tmp/opencode/perf-time/`; this file and its raw text/`.pprof` siblings
live under `plans/0.2.6/perf-time/profiles/`.

## 1. Capture header

- date: 2026-09-11, 19:15:43 to 19:16:10 IST
- go: go1.26.4 linux/amd64
- cpu: 13th Gen Intel(R) Core(TM) i7-13700HX, 24 CPUs
- memory at start: 7.6 GiB total, 4.0 GiB free, no swap pressure
- os: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64
- binary: `go test -c -o /tmp/opencode/perf-time/convert.test ./internal/convert`,
  sha256 `b40b8cdd8810ea211f3f51e98b6d496916eee346db64bd5a59d7d3f6aa3a2d46`
- workdir for every benchmark: `internal/convert` so the report template
  resolves
- fixture: `testdata/golden/benchmarks/templates/report.html.tmpl`
- mode: `-test.run '^$' -test.benchtime=1x -test.benchmem`, one process per
  size row, iteration 1 cold and the rest warm. Without `-test.run '^$'` the
  package's CLI comparison tests run once per count and poison the profile.
- bench filters: 500p `'^BenchmarkPDFPages/generic/500Pages$'`; 250p, 100p and
  50p use per-element anchors (`'^BenchmarkPDFPages$/^generic$/^50Pages$'`)
  because the unanchored 50Pages pattern also selects 250Pages.
- lock note: the main thread owned heavy-run scheduling for this pass. A
  sibling profiler (`/tmp/opencode/perf-time-style/convert.test`) was running
  at 19:14; `pgrep -af 'convert.test.*test.bench'` was empty at 19:15:25 and
  the four CPU profiles plus the gctrace run started immediately. No lock
  wait is charged to these results. Another agent wrote unrelated raw files
  (`*-flat2.tsv`, `peeks-*`, `style/`) into the same `raw/` directory while
  this run was in flight; this report cites only its own files.
- host drift caveat: these rows ran 1.15x to 2.1x slower than the published
  phase-7 warm matrix (table below). `B/op` matches the published numbers to
  within 0.3 percent, so the workload is identical and phase shares are the
  comparable quantity. Absolute milliseconds in this report are shares
  applied to the published 1,228.72 ms row unless stated otherwise.

Commands (all exit 0):

```sh
/tmp/opencode/perf-time/convert.test -test.run '^$' \
  -test.bench '<filter>' -test.benchtime=1x -test.count=<N> -test.benchmem \
  -test.cpuprofile=/tmp/opencode/perf-time/<size>-cpu.pprof
```

`GODEBUG=gctrace=1 /tmp/opencode/perf-time/convert.test -test.run '^$' \
 -test.bench '^BenchmarkPDFPages/generic/500Pages$' -test.benchtime=1x \
 -test.count=5 -test.benchmem`

## 2. Raw benchmark rows

`ns/op` converted to ms. Row 1 is cold, rows 2..N warm.

| size | count | rows (ms) | warm median | warm min | B/op rows (MB) |
|---|---:|---|---:|---:|---|
| 500p | 5 | 1411.7 / 1437.1 / 1636.9 / 1433.4 / 1735.2 | 1560.7 | 1433.4 | 235.60 / 234.92 / 234.92 / 235.05 / 235.19 |
| 250p | 5 | 688.7 / 646.6 / 660.7 / 656.1 / 642.7 | 651.5 | 642.7 | 118.54 / 118.17 / 118.17 / 118.29 / 118.16 |
| 100p | 10 | 267.4 / 263.1 / 345.8 / 271.9 / 288.2 / 270.2 / 264.6 / 256.2 / 249.9 / 408.8 | 270.2 | 249.9 | 48.68 down to 48.29 |
| 50p | 20 | 191.8 / 206.4 / 201.1 / 220.6 / 222.8 / 236.6 / 193.0 / 206.3 / 208.1 / 224.7 / 222.6 / 220.6 / 207.1 / 216.1 / 211.4 / 256.0 / 233.7 / 235.5 / 209.5 / 191.4 | 216.1 | 191.4 | 24.82 to 25.17 |

Published phase-7 warm matrix for reference: 500p 1,228.72 ms /
234.92 MB; 250p 606.98 ms / 118.43 MB; 100p 225.10 ms / 48.37 MB;
50p 102.59 ms / 24.82 MB. The 50p row is the most load-sensitive; this
window was slower than the published one, so its absolute time is not a
regression signal.

`output-bytes` is 1,419,234 at 500p in every iteration of every run.
`-test.bench` asserts the page count inside each run.

## 3. Phase attribution

Method: for each phase root regex, `go tool pprof -top -nodecount=4000
-nodefraction=0 -show_from=<regex>` sums the flat samples of every stack that
passes through that root (`/tmp/opencode/perf-time/phase-attribution.sh`).
Rows are disjoint for this workload. The residual row is `total_profile`
minus the named rows and holds benchmark scaffolding outside
`convert.Run` plus runtime work on stacks that pass through no root.
The "on baseline" column converts share to milliseconds of the published
1,228.72 ms 500p row and is a planning number, not a measurement.

### 500p (5 iterations, 8.14 s of samples)

| phase | seconds | share | on baseline (ms) | top contributors (flat or cum, source line) |
|---|---:|---:|---:|---|
| parse and prepare | 0.110 | 1.35% | 17 | `prepare.(*sheetCollector).visit` 20 ms (`internal/convert/prepare/styles.go:90`); `html.scanStartTag` (`internal/html/html.go:642`) |
| style resolution | 2.380 | 29.24% | 359 | `applyRestProps` 860 ms cum (`style_cascade.go:1184`, `:1194`, `:1204`, `:1207`); `inheritProps` 380 ms cum, 290 ms of it at the `raw[name]` lookup (`style_cascade.go:332`); `applyStyleProp` 500 ms cum (`style_cascade.go:1316-1317`); `cascadeRaw` 580 ms cum (`style_cascade.go:497-504`); `styleInternHashString` 70 ms flat (`style_intern_gen.go:20-21`); `styleInternEqual` 100 ms flat (`style_intern_gen.go:137+`) |
| box construction + display list | 1.820 | 22.36% | 275 | `layoutTableGrid` 1.40 s cum (`layout_tables.go:89` measure pass, `:107` emit pass); `measureTableRows` 590 ms cum (`layout_tables.go:682`); `emitTableCells` 810 ms cum; `emitCell` 580 ms cum (`layout_tables.go:1280`); `layoutInlineFloats` 830 ms cum (`inline.go:160`, `:229`); `emitLine` 550 ms cum (`inline.go:885`, `:894`); `primaryFaceRun` 400 ms cum (`inline_paint.go:1543`); `pdf.(*Font).GlyphID` 420 ms cum (`internal/pdf/fonts.go:630`); `splitTextByFace` 300 ms cum (`inline_paint.go:1448`) |
| pagination fixpoint and seal | 1.790 | 21.99% | 270 | `paginateOps` 1.08 s cum; `buildPageIndex` 230 ms flat / 260 ms cum with three full op passes (`paint_flow_index.go:365`, `:388`, `:400`); `beforeAlways` 360 ms cum (inside it `collectBeforeAlwaysBoxes.func1` 80 ms flat `paint_flow_breaks.go:914`, `(*breakScanState).advance` 70 ms flat `:402`, `boxInsideTable` 70 ms flat `:119`); `snapCrossingTextOps` 170 ms cum (`paint_pagination_fixpoint.go:144`); `normalizeTableRowGaps` 150 ms cum; `stripOrphanRowChrome` 150 ms; `rowInkBand` 80 ms flat (`paint_flow_tables.go:166`); `opInPaintRange` 90 ms flat (`paint_pagination_fixpoint.go:190`); `capTablePageBreaks` 220 ms cum; `validatePaintPageIndices` 60 ms cum over three calls (`paint.go:316`) |
| paint (page content) | 0.510 | 6.27% | 77 | `drawPageOp` 400 ms cum (`paint.go:545` drawLine 270 ms, `:547` drawText 120 ms); `strconv.genericFtoa` 200 ms cum; `pdf.(*Content).SetStrokeColor` 110 ms cum (`internal/pdf/content.go:267`); `paintOp` 420 ms cum (`paint.go:478`) |
| outline and navigation | 0.100 | 1.23% | 15 | `collectBodyNavigation` 30 ms flat (`internal/convert/links.go:97`); `outline.Lookup` (`internal/outline/outline.go:203-204`) |
| structure tree | 0.000 | 0% | 0 | no samples for this non-UA fixture (`buildStructureTree` is called at `paint.go:224` and returns early) |
| PDF finalize, serialize, compress | 0.660 | 8.11% | 100 | `flateBytes` 620 ms cum (`internal/pdf/pdf.go:1641`); `compress/flate.(*compressor).deflate` 540 ms cum (`deflate.go:471-473`, `:500`); `findMatch` 320 ms cum (`deflate.go:259-260`, `:273`, `:277`) |
| GC mark workers | 0.340 | 4.18% | 51 | `runtime.gcBgMarkWorker` subtree, all inside `gcDrain`/`scanObject` |
| residual / other | 0.430 | 5.28% | 65 | benchmark scaffolding outside `convert.Run`, runtime work on unrooted stacks |

Context rows from the same profile: `convert.Run` 7.63 s (93.73%),
`layoutContext` 4.40 s (54.05%), `resolveStylesForLayoutContext` 2.38 s
(29.24%), `PaintContext` 2.30 s (28.26%), `(*engine).build` 1.82 s
(22.36%), `pdfPipeline.Finalize` 0.66 s (8.11%).

### 50p (20 iterations, 4.67 s of samples)

| phase | seconds | share | top contributors (flat or cum, source line) |
|---|---:|---:|---|
| parse and prepare | 0.120 | 2.57% | same roots as 500p |
| style resolution | 1.530 | 32.76% | `applyRestProps` 600 ms cum (12.85%); `applyStyleProp` 420 ms cum (8.99%); `styleInternEqual` 150 ms flat (3.43%, `style_intern_gen.go` field comparisons); `(*styleStore).append` 280 ms cum (6.00%); `inheritProps` 170 ms cum; `cascadeRaw` 260 ms cum |
| box construction + display list | 1.190 | 25.48% | `flowChildren` 1.19 s cum; `layoutTableGrid` 890 ms cum; `layoutInlineRun` 680 ms cum; `emitTableCells` 560 ms cum; `emitCell` 480 ms cum; `emitLine` 450 ms cum; `primaryFaceRun` 330 ms cum (7.07%); `pdf.(*Font).AdvanceInPoints` 140 ms flat (`fonts.go:785`); `splitTextByFace` 250 ms cum; `collectInlineNode` 200 ms cum (`inline_collect.go:87`) |
| pagination fixpoint and seal | 0.610 | 13.06% | `paginateOps` 370 ms cum; `normalizeTableRowGaps` 80 ms cum; `capTablePageBreaks` 100 ms cum; `rowYBounds` 40 ms flat (`paint_flow_tables.go:739`); `validatePaintPageIndices` 40 ms flat (`paint.go:316`); `flowPageOfY` 30 ms (`paint_flow_index.go:529`) |
| paint (page content) | 0.300 | 6.42% | `paintOp` 260 ms cum; `strconv.AppendUint`; `drawText` (`paint.go:1241`) |
| outline and navigation | 0.090 | 1.93% | `collectBodyNavigation` 70 ms flat, 1.50% of the profile, one loop over all ops (`internal/convert/links.go:97`); `outline.Lookup` |
| structure tree | 0.000 | 0% | early return |
| PDF finalize, serialize, compress | 0.410 | 8.78% | `flateBytes` 370 ms cum; `deflate` 320 ms cum; `findMatch` 190 ms cum |
| GC mark workers | 0.160 | 3.43% | `runtime.gcBgMarkWorker` subtree |
| residual / other | 0.260 | 5.57% | same as 500p |

Context: `convert.Run` 4.33 s (92.72%), `layoutContext` 2.76 s (59.10%),
`resolveStylesForLayoutContext` 1.53 s (32.76%), `PaintContext` 0.90 s
(19.27%), `(*engine).build` 1.19 s (25.48%), `Finalize` 0.41 s (8.78%).

### 250p and 100p shares (evidence for scaling, no line views)

| phase | 250p | 100p |
|---|---:|---:|
| parse and prepare | 2.2% | 2.2% |
| style resolution | 33.1% | 32.9% |
| box construction | 23.5% | 24.6% |
| pagination and seal | 14.0% | 13.1% |
| paint | 7.3% | 6.7% |
| outline | 1.4% | 1.6% |
| PDF finalize | 8.4% | 10.2% |
| GC mark workers | 5.9% | 3.5% |
| residual | 4.2% | 5.1% |
| total samples | 3.57 s | 3.13 s |

Style resolution is the largest phase at every size. Pagination's share
grows with page count (13.1% at 100p, 22.0% at 500p).

## 4. GC

### gctrace, 500p, count=5

37 cycles total, 8 forced (startup plus one at each benchmark iteration
boundary), 29 non-forced. Peak heap 212 MB. After a forced boundary GC the
live heap drops to 6 MB. Per benchmark iteration (non-forced cycles only):

| benchmark row | wall ms | cycles | GC CPU ms | bg mark | idle mark | assist | stw+sweep | concurrent mark clock ms |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 (1470.2 ms) | 1470 | 5 | 128.3 | 63.0 | 38.0 | 0.8 | 26.5 | 13.7 |
| 2 (1397.6) | 1398 | 4 | 143.9 | 54.9 | 53.6 | 1.1 | 34.3 | 11.9 |
| 3 (1548.2) | 1548 | 5 | 141.1 | 62.5 | 60.9 | 0.3 | 17.4 | 13.8 |
| 4 (2006.7) | 2007 | 5 | 268.3 | 92.1 | 136.1 | 0.7 | 39.5 | 20.5 |
| 5 (1636.1) | 1636 | 5 | 181.7 | 72.6 | 66.6 | 0.2 | 42.2 | 16.6 |

Mean over the five rows 172.6 ms of GC CPU per conversion; warm rows 2 to 5
mean 183.8 ms. Most of it is background and idle mark on other Ps; mark
assist is 0.2 to 1.1 ms per conversion. Stop-the-world clock per cycle is
0.1 to 0.6 ms; summed over a conversion it is under 3 ms.

Representative per-cycle lines (all 37 are in `500-gctrace.txt`). Format is
`stw_sweep + concurrent_mark + stw_mark ms clock, mark_worker +
assist/background/idle + sweep ms cpu, before->during->after MB`:

```
gc 13 @0.562s 0%: 0.11+5.0+0.017 ms clock, 2.8+0.40/26/24+0.42 ms cpu, 123->124->113 MB
gc 19 @2.030s 0%: 0.12+5.1+0.16 ms clock, 2.9+0.93/25/36+3.9 ms cpu, 114->114->113 MB
gc 25 @4.470s 0%: 0.17+3.8+0.026 ms clock, 4.1+0/15/9.5+0.62 ms cpu, 209->211->28 MB
gc 31 @6.218s 0%: 0.24+6.4+0.20 ms clock, 5.8+0.062/35/66+4.8 ms cpu, 203->203->153 MB
gc 20 @2.925s (forced, iteration boundary): 0.11+0.28+0.005 ms clock, 2.6+0/0.86/0.12+0.12 ms cpu, 212->212->6 MB
```

The mark work per cycle scales with the live heap it scans: 69 ms of CPU for
the gc19 cycle (114 MB live), 112 ms for the gc31 cycle (153 MB live after
the sweep). The 268 ms row-4 figure is the sum of five cycles, not one.

### GC share in the CPU profile

- `runtime.gcBgMarkWorker` subtree: 0.340 s of 8.14 s at 500p = **4.18%**;
  0.160 s of 4.67 s at 50p = 3.43%.
- Allocator primitives inside phase stacks (`runtime.mallocgc` 0.330 s =
  4.05%, `runtime.memclrNoHeapPointers` 0.250 s = 3.07%) add 7.1% of samples
  at 500p. These are already inside the phase rows above, not extra.
- The pre-improve profile put GC mark workers at 10.05% and stopped at
  1.30 s on a 12.94 s profile. GC CPU per conversion fell roughly 25 percent
  since then (warm-row mean background+idle mark 149.8 ms here versus the 176
  to 221 ms recorded per warm iteration before).

### Amdahl headroom from GC (estimate)

The task asks what halving live heap and allocation traffic could remove of
the 1,228.72 ms row. Estimate, not a measurement:

- Halving both roughly halves mark work per cycle while the cycle count per
  conversion stays similar (cycles scale with allocation-to-live ratio, which
  is unchanged). That removes about half of `bg mark + idle mark`, near
  75 ms of parallel CPU per conversion, plus assist (under 1 ms).
- That CPU mostly runs on other Ps, so its direct wall contribution is much
  smaller than 75 ms. The serial part of GC is the STW windows (under 3 ms
  per conversion) and memory-bandwidth interference on the main goroutine,
  which this profile cannot separate.
- Planner's number: halving heap and allocation traffic plausibly returns
  20 to 50 ms of the 1,228.72 ms row, 1.6 to 4.0 percent. Full GC
  elimination cannot return more than about 5 percent. GC alone cannot
  deliver 2x.

## 5. Delta versus the pre-improve profile

Pre-improve source: `plans/0.2.6/perf-improve/profiles/warm-pdf-cpu.md`
(500p, 12.94 s of samples, host window slower than this one). Absolute
sample seconds are host-confounded; shares and `B/op` are the comparable
numbers. The row below pairs each phase share.

| phase | pre-improve share | now (500p) | direction |
|---|---:|---:|---|
| parse and prepare | 1.2% | 1.35% | flat |
| style resolution | 23.3% | 29.24% | share up; now the largest phase |
| box construction + display list | 24.1% | 22.36% | slightly down |
| pagination fixpoint and seal | 23.7% | 21.99% | slightly down |
| paint | 4.6% | 6.27% | share up |
| outline and navigation | 1.6% | 1.23% | flat |
| PDF finalize, serialize, compress | 6.3% | 8.11% | share up |
| GC mark workers | 10.05% | 4.18% | down, the largest relative cut |
| residual | 5.2% | 5.28% | flat |

What phases 3 to 6 removed, with their own measured numbers:

- Phase 3 (seal allocation): seal family 71.10 MB to 14.95 MB per 500p op
  (-79%); `capTablePageBreaks` share 5.02% to 2.00% in the same window.
  This profile still measures `capTablePageBreaks` at 0.22 s (2.70%).
- Phase 4 (forced-break batching): `shiftBoxesForForcedBreak` has no samples
  here (500 ms flat before); `paginateOps` was 8.04% right after phase 4.
  This profile puts `paginateOps` at 1.08 s, 13.27%, back at the
  pre-improve 12.75%. That is a divergence worth a dedicated probe; it is
  either host state, the phase-6 three-pass `buildPageIndex` on every
  rebuild, or more rebuilds. A CPU profile carries no call counts.
- Phase 5 (`Op` packing): `Op` 472 to 440 bytes; `(*engine).add` flat
  490 ms (3.79%) before, 60 ms (0.74%) now.
- Phase 6 (bucketing): `bucketOpsByPage` (310 ms cum, 28.28 MB allocated)
  is gone; `buildPageIndex` is 260 ms cum, 18.67 MB allocated in the phase
  evidence. `fixedOpIndices` allocation 1.40 MB to 0.
- Net: 500p `B/op` 321.31 MB to 235.36 MB (-26.7% self-reported; this run
  measures 234.92 MB, matching the published 234.92 MB). GC mark share fell
  10.05% to 4.18%. Warm matrix time 1,489.42 ms to 1,228.72 ms (-17.5%).

What stayed or grew in share: style resolution and the two map-driven
cascade helpers inside it (`applyRestProps` 5.18% cum before, 10.57% now;
`inheritProps` 3.56% before, 4.67% now); PDF compression (`deflate` 5.26%
before, 6.63% now, absolute 680 ms to 540 ms); paint (`drawPageOp` 4.91%
now). No project function exceeds 2.83% flat in this profile.

## 6. Ranked candidates for a 2x warm-time cut

The 2x budget is 614 ms of the published 1,228.72 ms row. Expectations are
`share x 1228.72 ms x an assumed reduction`, and every "max" assumes the
whole phase disappears, which no plan can do. All candidates must keep
500p `B/op` at or below 234.92 MB.

### 1. Style resolution: per-element cascade map traffic and sort (359 ms ceiling)

- Measured: phase 29.24% at 500p, 32.76% at 50p. `applyRestProps` 860 ms cum
  (10.57%) at 500p, `inheritProps` 380 ms cum (4.67%), `applyStyleProp`
  500 ms cum (6.14%), `cascadeRaw` 580 ms cum (7.13%), `styleStore.append`
  260 ms cum, `styleInternEqual` 100 ms flat (150 ms at 50p, 3.43%).
- Lines: `style_cascade.go:1184-1190` (a `raw[prop]` lookup per shorthand
  prop), `:1194-1204` (per-element `[]string` plus `sort.Strings`),
  `:1206-1207`; `:332` (`raw[name]` per inheritable prop, 290 ms cum here);
  `:1316-1317` (11-entry linear group dispatch per property);
  `style_intern_gen.go:20-21` (byte-loop hash).
- Expected: 100 to 160 ms from replacing the per-element sort with a fixed
  longhand order and reading the raw map once; up to 240 ms if interning and
  dispatch are also touched. Max (delete the phase) 359 ms = 1.41x alone.
- Risk: cascade order and determinism are correctness contracts; the golden
  corpus covers them. Deterministic order must stay byte-stable.
- Probe: temporary counter for `raw` lookups, `keys` allocations and sort
  calls per conversion. After the change require `applyRestProps` cum below
  0.50 s at 500p with the style share under 18%, `make golden` exit 0, and
  500p `B/op` not above 234.92 MB.

### 2. Box construction: table measure pass and text advance recomputation (275 ms ceiling)

- Measured: phase 22.36% at 500p, 25.48% at 50p. `layoutTableGrid` 1.40 s cum
  (17.20%) splits into `measureTableRows` 590 ms (7.25%) and
  `emitTableCells` 810 ms (9.95%). Cell content is flowed in both passes:
  `emitCell` calls `flowChildren` at `layout_tables.go:1280`.
  `layoutInlineFloats` 830 ms cum (10.20%), `emitLine` 550 ms cum (6.76%),
  `primaryFaceRun` 400 ms cum (4.91%, rune-by-rune
  `AdvanceInPoints` at `inline_paint.go:1543`), `GlyphID` 420 ms cum
  (5.16%, cmap map lookup at `internal/pdf/fonts.go:630`),
  `splitTextByFace` 300 ms cum, `inlineFontMetrics` 330 ms cum,
  `measuredWidth` 60 ms flat scanning all ops (`internal/convert/convert_helpers.go:256`).
- Expected: 80 to 140 ms from caching per-(face, size, rune) advances and
  reusing the measure pass's row results in emit. Max: the measure pass
  alone is 118 ms per conversion (590 ms / 5); deleting the whole phase is
  275 ms = 1.29x alone.
- Risk: measuring at one width and emitting at another changes wrapping;
  a naive cache keyed without the scale or face changes glyph advances.
  Caches raise `B/op`, so the probe must re-check the 234.92 MB bound.
- Probe: count `measureCellHeight`, `emitCell` and `GlyphID` calls per
  conversion with a test hook, then after the change require `GlyphID` cum
  below 0.15 s and `measureTableRows` cum below 0.35 s at 500p, with
  `output-bytes` still 1,419,234 and `make golden` exit 0.

### 3. Pagination: three-pass index rebuilds and duplicate full-op scans (270 ms ceiling)

- Measured: phase 21.99% at 500p. `paginateOps` 1.08 s cum (13.27%);
  `buildPageIndex` 230 ms flat / 260 ms cum with three full passes over
  174,000 ops (`paint_flow_index.go:365` 80 ms, `:388` 40 ms, `:400` 90 ms);
  `validatePaintPageIndices` is called three times (`paint.go:165`, `:184`,
  `:207`), 60 ms cum; `beforeAlways` 360 ms cum; `snapCrossingTextOps`
  170 ms; `normalizeTableRowGaps` 150 ms; `stripOrphanRowChrome` 150 ms;
  `capTablePageBreaks` 220 ms; `opInPaintRange` 90 ms flat
  (`paint_pagination_fixpoint.go:190`).
- Expected: 80 to 130 ms by computing op-to-page once and reusing it for
  validation, buckets and locations. Max (delete the phase) 270 ms =
  1.28x alone.
- Risk: phase 6 already rejected a live-index epoch reuse because it changed
  fixture-56 output; any reuse must prove the same invalidation semantics.
  The three validation scans are guards, so removing them needs a cheaper
  invariant, not just deletion.
- Probe: temporary counters for `buildPageIndex` and
  `validatePaintPageIndices` calls per conversion. Require at most 2 and at
  most 1 respectively, `make golden` exit 0, and byte-identical fixture-56
  output.

### Secondary candidates

| candidate | measured share | ceiling | realistic | risk | probe |
|---|---:|---:|---:|---|---|
| PDF compression (`flateBytes` 620 ms cum, `deflate` 540 ms, `findMatch` 320 ms) | 8.11% | 100 ms | 40 to 90 ms by compressing page streams in parallel or moving to a faster deflate | output bytes and determinism; worker buffers raise `B/op` | `bytes.Equal` the 500p output and compare `B/op` before and after |
| Paint content formatting (`drawPageOp` 400 ms cum, `strconv.genericFtoa` 200 ms cum, `SetStrokeColor` 110 ms) | 6.27% | 77 ms | 30 to 50 ms via a fixed-precision number writer and content buffer reuse | coordinate formatting changes output bytes if precision changes | paint phase share below 4% with byte-identical output |
| Cross-cutting map cost (`matchH2` 5.28% flat, `mapaccess2_faststr` 6.76% cum, `mapaccess2_fast32` 4.30% cum, `aeshashbody` 1.97%) spread over candidates 1 to 3 | 15 to 19% across phases | hard to isolate | 60 to 120 ms by replacing rune-to-glyph maps with a direct table for ASCII and property-name string maps with small integer IDs | keys with few entries do not repay a new table; `B/op` rises | after one map swap, require the matching `mapaccess*` row to leave the top 60 and no `B/op` increase |

### What cannot deliver 2x

- GC: mark workers are 4.18% at 500p and 3.43% at 50p. Halving heap and
  allocation traffic returns an estimated 20 to 50 ms (1.6 to 4.0 percent).
  The remaining GC CPU runs in parallel; the serial STW part is under 3 ms
  per conversion.
- PDF compression: even making compression free saves 100 ms = 1.09x.
- Paint: 77 ms = 1.07x.
- Parse and outline together: 32 ms = 1.03x.
- Any single function: the largest project flat function is `buildPageIndex`
  at 2.83%; deleting it saves 35 ms = 1.03x. The largest profile flat row is
  a runtime map internal at 5.28% = 1.06x.
- The top 12 project flat functions total about 1.05 s of 8.14 s (12.9%);
  deleting all twelve would save about 158 ms on the published row = 1.15x.
- Style interning alone (`styleInternEqual` 100 ms flat, `styleInternHashString`
  70 ms, `styleInternFingerprint` 120 ms cum) is 1.5 to 3.4% = at most 1.04x.

### Honest combined reading

- Halving all three leaders: 452 ms saved, 776 ms left, 1.58x.
- Halving the three leaders and cutting the five small phases (PDF, paint,
  GC, parse, outline, 260 ms combined) by 75 percent: 647 ms saved, 582 ms
  left, 2.11x. That is the only arithmetic path to 2x, and it assumes
  near-perfect execution on compression, paint formatting and GC at once.
- A plan built on the measured levers should target 1.4x to 1.6x, with 1.8x
  as a stretch if the table double pass and compression both land. The
  profile is flat, so no single change carries the target.

## 7. Evidence files

Owned by this run, under `plans/0.2.6/perf-time/profiles/raw/`:

- `500-cpu.pprof`, `250-cpu.pprof`, `100-cpu.pprof`, `50-cpu.pprof`: raw
  profiles.
- `500-cpu-run.txt`, `250-cpu-run.txt`, `100-cpu-run.txt`, `50-cpu-run.txt`:
  benchmark stdout including `B/op`.
- `500-gctrace.txt`, `500-gctrace-summary.txt`: gctrace run and parsed table.
- `N-cpu-phase-sums.tsv`: phase sums for N in 500, 250, 100, 50.
- `N-cpu-top-flat.txt`, `N-cpu-top-cum.txt`: top 80 flat and cumulative.
- `N-cpu-phase-<phase>-lines.txt`: per-phase line attribution for 500 and 50.
- `N-cpu-list-<function>.txt`: `-list` views for the top flat functions and
  the leaf hotspots named above.

Scratch (not committed): `/tmp/opencode/perf-time/` holds the test binary,
the phase-attribution and extraction scripts, and the benchmark runs.

Pre-improve comparison: `plans/0.2.6/perf-improve/profiles/warm-pdf-cpu.md`
and `plans/0.2.6/perf-improve/results/phase-{3,4,5,6}/`.
