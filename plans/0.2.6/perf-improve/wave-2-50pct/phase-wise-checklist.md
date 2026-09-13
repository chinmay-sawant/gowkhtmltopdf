# 0.2.6 performance improve wave 2 - cut remaining ns/op and B/op in half

> **Parent:** `plans/0.2.6/perf-time/phase-wise-checklist.md` (Snapshot M) and `plans/0.2.6/perf-improve/phase-wise-checklist.md` (Snapshot L). This file does not reopen those rows.
> **Status:** in progress 2026-09-12. Implementation is in the tree. PDF 50% time and B/op not met (624 ms / 121 MB vs 367 ms / 81.5 MB). Image 500-tile B/op met (10.6 MB vs 13.2 MB target). Evidence: `results/wave2-capture.md`. `make golden` 65/65. Lint not run this capture.
> **Estimated effort:** eight measured phases. Architecture (phases 4 and 5) is required for the 50/50 target. Sequential leftovers alone are not enough.
> **Date:** 2026-09-11
> **Evidence boundary:** current source on `chore/review-026`, Snapshot M in `testdata/golden/benchmarks/benchmark-results.txt`, the 2026-09-11 perf-time profiles and captures, and four read-only explorations of benches, layout/pdf hotspots, recent landed work, and CLI vs library. No git command ran while writing this plan.
> **Tree:** `VERSION` still reads 0.2.5. Snapshot M is the published current capture.

---

## overview

The last three waves already took the cheap sequential cuts. Recovery interned
styles and fixed the image raster. Wave 1 packed the display list, sealed
table borders without growth, and merged the page index. The time wave
memoized style, batched grid lines, turned span checks into a binary search,
and compressed page streams on a retained pool. Warm 500-page generic PDF is
now 733.48 ms and 163.02 MB on the published Snapshot M median, or 576.33 ms
at the same 163 MB in a faster same-source window.

This run asks for half of what is left: about 367 ms and 81.5 MB on the
published median, for the library and the CLI.

That is not another scan-skip. After the time wave the leftover cost is one
432-byte display list for 500 pages, a 54,503-box tree, a three-pass table
for 52,500 cells, and pagination that still sees the whole list. Halving
allocation traffic means not keeping 500 pages of drawing records at once.
Halving time means not rebuilding the same table chrome 500 times. Those are
the two architectural phases. Everything else is a few percent.

CLI 500-page time is already the same conversion as the library (0.70 s vs
698.79 ms). There is no CLI-only 50%. Image 500-tile `B/op` is already 62%
one RGBA canvas (1024x4040x4 = 16.5 MB of 26.4 MB), so a 13.2 MB image
target needs tiled raster or it is a documented miss.

## executive summary

Five facts from current code, not from the pre-time-wave profiles:

1. `Op` is still a 432-byte value in one `[]Op` (`internal/layout/layout.go:408-505`).
   The 500-page report keeps 66,500 of them, capacity about 97,533 slots
   (`estimateOpCapacity` at `layout.go:1274-1305`), about 42 MB reserved.
2. Each table cell still flows three times (`layout_tables.go:1216-1303`).
   PERFT-12, which tried to splice pass 2 into pass 3, stays rejected.
3. Production `convert.Run` still calls `layout.LayoutContext` with no
   `Workspace` (`convert.go:640-648`). `Workspace.Release` exists
   (`layout.go:312-345`) and is used only by certified page islands
   (`page_islands.go:53-73`). Published benches use `NewPDFRequest`, not
   islands (`internal/convert/benchmarks_test.go:349-350`).
4. Parallel section layout is identity-safe and gained 5.4% against a 15%
   floor. It stays unwired (`internal/layout/parallel.go`,
   `plans/0.2.6/perf-time/results/phase-6/concurrency.md`).
5. Pipeline overlap (PERFT-25) was never attempted. It was gated on the
   parallel merge. Page-at-a-time unblocks it without merging lists.

Payoff order:

1. Fresh profiles of *this* tree. The 1,228 ms phase shares are stale.
2. Cheap assemble and measure guards, plus a glyph-advance cache.
3. Compact `Op` side payload. Tens of MB, not 81 MB.
4. Production page-at-a-time with `Workspace` reuse. This is the `B/op` half.
   Design: `designs/page-at-a-time.md`.
5. Repeated-section chrome clone with unique-text reflow. This is the time
   half. Design: `designs/section-chrome-clone.md`.
6. Flate-and-release per page, then overlap layout N+1 with compress N.
7. Image tiled raster, only after the PDF path lands.
8. Closure capture as Snapshot N.

## baseline and 50% targets

Published Snapshot M medians. `B/op` is the raw value of the median-time
sample, never averaged. Row 1 of a warm matrix is cold. CLI RSS is
`/usr/bin/time %M`, not `B/op`.

| Row | Snapshot M | 50% target |
|---|---:|---:|
| Internal generic PDF, warm 500 pages | 733.48 ms / 163,021,712 B | <= 366.74 ms / 81,510,856 B |
| Internal generic PDF, warm 250 pages | 320.62 ms / 82,247,280 B | <= 160.31 ms / 41,123,640 B |
| Internal generic PDF, warm 100 pages | 123.90 ms / 33,972,640 B | <= 61.95 ms / 16,986,320 B |
| Internal generic PDF, warm 50 pages | 56.36 ms / 17,675,408 B | <= 28.18 ms / 8,837,704 B |
| Internal standalone 500 pages | 695.42 ms / 167,865,712 B | <= 347.71 ms / 83,932,856 B |
| Public `Document.WritePDF` 500 pages | 698.79 ms / 169,650,976 B | <= 349.40 ms / 84,825,488 B |
| CLI 500 pages (`cli-rss`) | 0.70 s / 147,264 KiB | <= 0.35 s / 73,632 KiB |
| Public image 250 tiles | 25.72 ms / 14,296,096 B | <= 12.86 ms / 7,148,048 B |
| Public image 500 tiles | 44.27 ms / 26,414,016 B | <= 22.14 ms / 13,207,008 B |

Same-source faster window, not the published median: warm 500 pages
576.33 ms / 163,032,880 B. Half of that is 288.17 ms. Report it when a
capture lands in that band. Do not replace Snapshot M with it.

CLI 2-page 0.01 s is `/usr/bin/time %e` at 10 ms. It is not a 50% gate.
Library 2-page cold is 5.50 to 6.11 ms and is mostly first-parse. Do not
mix it with warm 500-page rows.

Capture command, unchanged from Snapshot M:

```sh
./scripts/bench-performance-recovery.sh --mode=internal-pdf-warm --benchtime=1x --count=1 --out=plans/0.2.6/perf-improve/wave-2-50pct/results/<phase>
```

## honest ceiling

Sequential leftovers (assemble guards, glyph cache, compact `Op`, skip empty
table walks) are about 10 to 25% time and about 20 to 30 MB. Amdahl on the
576 ms row caps serial-only work near 360 to 411 ms, which is not 288 ms
and is only barely the 367 ms published-median target if every leftover
lands at its best.

The 81.5 MB line is not reachable while pagination still retains 66,500 ops
plus 54,503 boxes. Compact `Op` at 432 to 200 bytes saves about 20 MB, not
81 MB.

If phases 4 and 5 fail their equivalence probes, this wave publishes the
sequential package as a partial result. It does not dress 20% up as 50%.

## non-negotiable rules

- Do not modify existing tests, existing golden fixtures, or existing
  `Benchmark*` functions. New test files and new benches are allowed.
- Do not change page counts, ordered text, fonts, links, outlines, structure
  tags, PDF profile output, image dimensions, decoded pixels, or PNG
  opacity. 500-page `output-bytes` stays 1,419,234.
- Do not turn on `benchmarkPageIslands` for any published number
  (`internal/convert/convert.go:65-69, 123-135`).
- Do not wire `ParallelLayout` (`internal/layout/parallel.go`). Rejected at
  5.4% paired gain.
- Do not revive PERFT-07, PERFT-12, PERFT-17, live page-index across
  invalidation, `estimateOpCapacity` tightening, naive per-call flate pools,
  or parallel style resolution without new measured evidence that the old
  blocker is gone.
- Style storage stays at 221,208 B per conversion unless a lever provably
  lowers it. Image 250/500 `B/op` must not rise.
- Warm and cold rows stay labeled. `B/op` is never averaged.
- One package owner at a time. Phases 3 to 5 all touch `internal/layout`
  and run sequentially. Phase 6 may run beside phase 7 once phase 4 has
  landed.
- No git commands.
- A phase closes only when its stated measurement lands and `make lint`
  plus `make test` exit 0.

## phase 1: freeze this tree and re-profile

Old CPU shares (style 29 to 36%, build 22 to 25%) describe the 1,228.72 ms
tree. They are not a budget for this wave.

### 1.1 Capture and pin

- [ ] **PERF3-01** Recapture the warm matrix, three standalone 2/500 samples,
      public PDF, public image, and CLI `cli-rss` with
      `scripts/bench-performance-recovery.sh` into
      `plans/0.2.6/perf-improve/wave-2-50pct/results/phase-1/`. Proof: rows
      match Snapshot M within host noise, `B/op` within 0.5%, `output-bytes`
      1,419,234, page counts exact.
- [ ] **PERF3-02** Pin ceilings from that capture, not from memory: warm 500p
      `B/op` <= the captured value, style storage 221,208 B, image 250/500
      `B/op` at or below 14,296,096 / 26,414,016, `Op` size recorded with
      `unsafe.Sizeof(Op{})` in a *new* test file. Proof: numbers written
      beside this row and in `results/phase-1/pins.md`.
- [ ] **PERF3-03** Pin the equivalence harness without editing existing
      tests: new files assert 500-page `output-bytes` 1,419,234 on generic
      `NewPDFRequest`, golden 65/65 still invoked as `make golden`, and
      ordered text needles via the existing unexported helpers copied or
      re-derived in the new file. Proof: the new tests fail if bytes drift
      and pass on this tree.
- [ ] **PERF3-04** Capture fresh 500-page CPU and alloc profiles of *this*
      tree (`-test.run '^$' -test.bench '^BenchmarkPDFPages$/^generic$/^500Pages$'
      -test.benchtime=1x`). Write phase shares into
      `profiles/wave2-cpu.md` and `profiles/wave2-alloc.md`. Proof: shares
      sum, host header present, and every later phase cites these files
      rather than the 1,228 ms reports.

### 1.2 Method

- [ ] **PERF3-05** State cold or warm on every row in this ledger and in
      every capture header. Proof: phase-1 capture headers carry the label.
- [ ] **PERF3-06** Phase gate: `make lint` and `make test` exit 0. Proof:
      exit codes recorded. No production code is required in this phase
      except new test files.

## phase 2: cheap leftover scans (not the 50%)

These are real and small. Ship them so later architecture is not blamed
for leaving 5 ms on the table. None of them is allowed to claim the 50%
target.

### 2.1 Convert guards

- [ ] **PERF3-07** Skip `drawHeadersFootersResult`'s per-page loop
      (`internal/convert/hf.go:737-779`) when both header and footer have
      no content and no `HTMLURL`. `headerHasContent` already exists
      (`hf.go:165`) but runs inside the loop (`hf.go:797`). Target: no
      `SectionOfBy` and no `idIndex` build on the report fixture. Proof:
      new `TestAssembleSkipsEmptyChrome` plus `make golden`.
- [ ] **PERF3-08** Skip `collectBodyNavigation` (`internal/convert/links.go`)
      when layout reports no ids and no fragment links. The report fixture
      has neither. Proof: new counter test, navigation still collected when
      a fixture has `id` or `#` hrefs, `make golden`.
- [ ] **PERF3-09** Stop walking all ops in `measuredWidth`
      (`internal/convert/convert_helpers.go:253-268`) when layout already
      knows max rect/image X. The report does not re-layout (overflow 0.00
      pt, `convert.go:26-31, 694-702`) but still scans. Proof: new
      `TestSmartShrinkNoRelayoutWhenWithinTenthPoint` counts `layoutFn == 1`
      on the report shape; `make golden`.

### 2.2 Glyph advances

- [ ] **PERF3-10** Cache `AdvanceInPoints` per (face, size, rune) on the
      engine for the conversion (`internal/layout/inline_paint.go:1521-1543`).
      Header cells repeat "Line" / "SKU" 500 times; body text is unique so
      do not memo whole rows. Proof: new hit-counter test, ordered text
      unchanged, warm 500p `B/op` does not rise.
- [ ] **PERF3-11** Phase measure: warm 500p time down, `B/op` not up.
      Record the actual ms. Expected band is a few percent, not 50%.
      Proof: script capture vs phase-1.
- [ ] **PERF3-12** Phase gate: `make lint` and `make test` exit 0.

## phase 3: compact Op side payload

PERFT-11 removed `FontFeatures` and landed 432 B. The 200 B side-payload
option was not shipped because readers live in `internal/imageout` and
`internal/convert`.

### 3.1 Move rare fields off the hot record

- [ ] **PERF3-13** Record current `unsafe.Sizeof(Op{})` and field use on
      the 500-page report (how many ops have non-zero Radius, Xform, Image,
      URI, BlendMode, StructElem). Proof: probe output under
      `results/phase-3/`.
- [ ] **PERF3-14** Move rare payloads to side tables so `Op` is at or
      below 256 B. Keep accessors so paint, imageout, and convert do not
      read a zero Radius by mistake. Do not tighten `estimateOpCapacity`
      globally. Proof: new `TestOpSizePacked`, `make golden` byte-identical
      after date normalization, 500-page `B/op` down at least 15 MB.
- [ ] **PERF3-15** Keep `OpGridRun` as the batched line path
      (`internal/layout/grid_run.go`). Do not explode runs back into
      `OpLine`. Proof: golden fixtures 60, 31, 29 plus the table suites.
- [ ] **PERF3-16** Phase measure: 500p `B/op` down, time not worse by more
      than host noise. Proof: script capture plus sizeof.
- [ ] **PERF3-17** Phase gate: `make lint` and `make test` exit 0.

## phase 4: production page-at-a-time (the B/op half)

Design: `designs/page-at-a-time.md`. This is architecture. It is also the
only lever that can cut `B/op` by ~50% without shrinking every `Op`.

### 4.1 Detector and reuse

- [ ] **PERF3-18** Implement an independent-block detector that fails
      closed: in-flow body children, `page-break-before` except the first,
      `page-break-inside: avoid`, not floated or absolutely positioned, each
      candidate at most one page. Do not key on the island fixture marker
      (`page_islands.go:16`) or `benchmark-page` class names. Proof: new
      tests for match on the report shape, no-match on a spanning table,
      no-match on a flex body.
- [ ] **PERF3-19** Wire `layout.WithWorkspace` into generic `convert.Run`
      for detected candidates: layout, paint, copy headings/nav/page names,
      `Workspace.Release`, next candidate. Production stays on
      `NewPDFRequest`. Proof: `benchmarkPageIslands` remains false in
      public and CLI paths; `grep` in the proof file.
- [ ] **PERF3-20** Equivalence: 5-section date-normalized PDF equals one
      5-section `LayoutContext`. 500-page `output-bytes` 1,419,234, page
      count 500, ordered text unchanged. If bytes differ, record the first
      operator delta and do not ship. Proof: new `TestPageAtATimeBytes`
      plus `make golden`.
- [ ] **PERF3-21** Fallback: a document that fails the detector, or a
      candidate that paints more than one page, uses today's single
      `LayoutContext` and matches today's PDF. Proof: spanning-table
      fixture in a new test file.
- [ ] **PERF3-22** Phase measure: warm 500p `B/op` at or below 90 MB as
      the phase gate (halfway to 81.5 MB is acceptable here if phase 3
      already moved some). Time may be flat. Proof: script capture plus
      alloc profile showing `[]Op` capacity reused.
- [ ] **PERF3-23** Phase gate: `make lint` and `make test` exit 0.
- [ ] **PERF3-24** If PERF3-20 fails, mark PERF3-18..23 `[~]` with the
      failing invariant, keep the sequential path, and skip phase 5's
      clone (it needs this detector). Do not force a 50% `B/op` claim.

## phase 5: repeated-section chrome clone (the time half)

Design: `designs/section-chrome-clone.md`. Depends on phase 4's detector.
PERFT-12 stays rejected: this clones a finished section and rewrites
unique text, it does not splice pass 2 of a cell into pass 3.

### 5.1 Clone chrome, reflow unique strings

- [ ] **PERF3-25** Hash section structure with text nodes as holes plus
      style-pointer identity. Sections that share the hash are a clone
      group. Proof: new test that the 500-page report is one group and a
      rowspan/nested-table fixture is not.
- [ ] **PERF3-26** Layout section 0 fully. For later sections, clone ops
      and boxes, translate Y, replace unique cell text, and full-layout
      any section whose unique string overflows the cloned column or
      would change row height. New `Op.ID` values in document order.
      Proof: 5-section clone PDF byte-identical to 5-section full layout
      after date normalization; overflow section takes the full path.
- [ ] **PERF3-27** Counter: `emitCell` runs once per clone group on the
      report shape, not once per section. Ordered SKU needles still
      complete. Proof: new test file plus `make golden`.
- [ ] **PERF3-28** Phase measure: warm 500p time moving toward 366.74 ms
      when combined with phase 4. `B/op` must not rise. Proof: script
      capture vs phase-4 capture.
- [ ] **PERF3-29** Phase gate: `make lint` and `make test` exit 0. If
      PERF3-26 fails, mark `[~]` with the operator delta and leave
      page-at-a-time in place.

## phase 6: flate-and-release and pipeline overlap

PERFT-20 flated all pages in parallel, then materialized them. PERF3-30
now flates a window of at most 8, finalizes that window in index order,
then starts the next window (`internal/pdf/flate_parallel.go:90-170`).
PERFT-25 (overlap) was not attempted.

### 6.1 Writer

- [x] **PERF3-30** Flate page N, `finalizePage` N, `releaseBuffer` N,
      then page N+1. Keep input-index order so output bytes stay
      1,419,234. Proof: `go test ./internal/pdf -count=1` exit 0 (7.079s);
      existing determinism tests still pass (not edited);
      `TestFlateReleaseTwoWritesByteIdentical` two-write byte compare plus
      PDF-06 empty `Content.Bytes`; evidence
      `results/phase-6/flate-release.md`. Convert 500-page `output-bytes`
      1,419,234 not re-measured here (`internal/pdf` owner only).
- [ ] **PERF3-31** CLI 500p RSS at or below 110,000 KiB as the phase
      gate, stretch 73,632 KiB if page-at-a-time also dropped live ops.
      Proof: `cli-rss` capture.
- [ ] **PERF3-32** Overlap layout of candidate N+1 with compress of page
      N only after phase 4 ships. Deterministic join: paint order is
      document order. Proof: measured wall time vs serial page-at-a-time
      on the same process pair; output bytes identical. If gain is under
      10%, record reject and leave serial page-at-a-time.
- [ ] **PERF3-33** Phase measure: warm 500p time and `B/op` vs phase 5.
      Proof: script capture.
- [ ] **PERF3-34** Phase gate: `make lint` and `make test` exit 0.

## phase 7: image leftover

After `pngfast.go`, encode is not the wall. 500-tile `B/op` is 26.41 MB,
of which the canvas is 16.5 MB. A 13.21 MB target is impossible while the
full RGBA image lives.

### 7.1 Raster

- [ ] **PERF3-35** Profile public image 250/500 on this tree. Rank
      layout vs raster vs encode with ms and bytes. Proof:
      `results/phase-7/image-alloc.md`.
- [ ] **PERF3-36** Tile or stream the raster so the full 1024x4040 RGBA
      buffer is not required, only if decoded pixels stay bit-identical
      at 1024x2056 / 1024x4040 and full opacity. Proof: IMG-01 / IMG-03
      still invoked as existing tests (do not edit them); new pixel-compare
      helper if needed; encoded PNG size may stay at the filter-none
      trade (141,917 / 282,749).
- [ ] **PERF3-37** Phase measure: image 250/500 time toward 12.86 / 22.14
      ms. `B/op` toward 7.15 / 13.21 MB only if PERF3-36 ships; otherwise
      record the canvas floor (16.5 MB at 500 tiles) as the honest `B/op`
      stop. Proof: three `public-image` samples.
- [ ] **PERF3-38** Phase gate: `make lint` and `make test` exit 0. If
      tiled raster changes pixels, revert and mark `[~]`.

## phase 8: closure and publication

### 8.1 Gates and capture

- [ ] **PERF3-39** Full gate set on the final tree: `make test`,
      `make golden` 65/65, `make claim-scan`, `make lint`,
      `make test-race`, `make build`, and `compliance/verify_pdfs.sh` for
      PDF/A-4 and PDF/UA-2. Proof: every command exits 0, logs under
      `results/phase-8/`.
- [ ] **PERF3-40** Final capture: warm matrix, standalone 2/500, public
      PDF, public image, CLI `cli-rss`, three samples each. Report raw
      rows and medians with cold and warm labeled. Proof: raw files plus
      a raw-to-published mapping in `results/phase-8/final-capture.md`.
- [ ] **PERF3-41** Publish only when the measured result is honest.
      Stretch is 50% of Snapshot M on the table above. If architecture
      missed, publish the sequential (and any shipped architecture) deltas
      as Snapshot N with the miss named. Keep Snapshots I, K, L, M as
      dated history. Proof: `testdata/golden/benchmarks/benchmark-results.txt`,
      `documentation/performance.md`, frontend data, rebuilt `docs/`,
      `make claim-scan` and `make lint-frontend` exit 0.
- [ ] **PERF3-42** Update `knowledge-base/` with the wave-2 findings, the
      shipped vs rejected architecture, and the new snapshot letter.
      Proof: article linked from `wiki/index.md`.
- [ ] **PERF3-43** Confirm existing tests were not modified. Proof: a
      path list of new files only in the closure note.

## new tests (do not edit existing files)

Create these as new `*_test.go` files. Names are suggestions.

| Test | Package | Pins |
|---|---|---|
| `TestOpSizePacked` | `internal/layout` | `unsafe.Sizeof(Op{})` at or below the phase-3 cap |
| `TestReportFixtureOutputBytesPin` | `internal/convert` | generic 500-page `len(pdf)==1419234` |
| `TestAssembleSkipsEmptyChrome` | `internal/convert` | empty HF does not enter the per-page draw |
| `TestSmartShrinkNoRelayoutWhenWithinTenthPoint` | `internal/convert` | `layoutFn` once on the report shape |
| `TestPageAtATimeBytes` | `internal/convert` or `internal/layout` | 5-section fast path equals full layout |
| `TestIndependentBlockDetectorFallback` | `internal/convert` | spanning table uses `LayoutContext` |
| `TestSectionChromeCloneEquivalence` | `internal/layout` | clone PDF equals full layout; overflow falls back |
| `TestGlyphAdvanceCacheHits` | `internal/layout` | hit counter on repeated header text |
| `TestLibraryPDFAllocCeiling` | root, new file | public 500p `B/op` ceiling from the live pin |
| `TestCLIProcessRSSReportFixture` | `internal/app` or cmd integration | opt-in; page count 500, bytes 1,419,234, RSS ceiling |

Do not add these by editing `document_bench_test.go`,
`internal/convert/benchmarks_test.go`, or any golden fixture.

## rejected levers (do not revive without new evidence)

| Lever | Why dead |
|---|---|
| PERFT-07 generated dispatch / rule index | After memo, `applyStyleProp` is ~20 calls per 500 pages |
| PERFT-12 reuse pass-2 cell as pass 3 | Equivalence not proved; `Op.ID` / ranges / clip |
| PERFT-17 live flow-index suffix | +1.1 MB `B/op`, ~4 ms, fixture-56 history |
| Concurrent `ParallelLayout` | 5.4% vs 15% floor, merge recopies the list |
| `estimateOpCapacity` tighter multiplier | corpus max 15.16 ops/node; one growth was ~120 MB |
| Live page-index across invalidation | changed fixture-56 |
| Naive per-call flate pool | +6.0 MB `B/op` |
| Parallel style resolution | would break the 221,208 B pin |
| Certified page islands for published numbers | private flag, not a user path |
| Process-reuse daemon for CLI | out of scope for one-shot `gowkhtmltopdf` |
| Disabling outline or smart-shrink in benches only | would change production defaults or skip a cheap scan |

## acceptance criteria

- Stretch: 50% of Snapshot M on internal warm 50/100/250/500, public PDF
  500, CLI 500 time and RSS, and image 250/500 *time*. Image `B/op` 50%
  is gated on PERF3-36.
- Floor if architecture misses: publish whatever sequential plus shipped
  architecture actually moved, with the 50% miss named. A result in the
  400 to 550 ms band at 100 to 140 MB is a partial wave, not a failure to
  record.
- Existing tests unmodified. New tests green. `make lint` and `make test`
  exit 0 at every phase close.
- Every published row maps to a raw capture. Cold and warm stay labeled.

## dependencies and order

```text
PERF3-01..06          # measure this tree
  -> PERF3-07..12     # convert guards + glyph cache
  -> PERF3-13..17     # compact Op (internal/layout + imageout readers)
  -> PERF3-18..24     # page-at-a-time (internal/layout then convert)
  -> PERF3-25..29     # chrome clone (internal/layout), needs detector
  -> PERF3-30..34     # pdf flate-and-release; overlap needs phase 4
  -> PERF3-35..38     # imageout, after shared layout lands
  -> PERF3-39..43     # closure
```

Phases 3, 4, and 5 all touch `internal/layout`. One owner, sequential.
Phase 2's convert guards may run beside phase 3 if a second owner is
free and the captures do not overlap on the same tree.

## files already optimized (do not reopen as new work)

Style intern and memo: `style.go`, `style_intern_gen.go`, `style_memo.go`,
declared-property mask in `style_cascade.go`. Seal pre-count:
`paint_pagination_seal.go`. Shift batching: `paint_flow_breaks.go`. Merged
page index: `paint_flow_index.go`. `OpGridRun`: `grid_run.go`. Lazy faces:
`internal/pdf/faces.go`. Retained parallel flate: `flate_parallel.go`.
Fast PNG: `internal/imageout/pngfast.go`. Unwired parallel layout:
`parallel.go`.
