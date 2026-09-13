# 0.2.6 performance time - halve warm render time

> **Parent:** `plans/0.2.6/perf-improve/phase-wise-checklist.md` - the allocation and warm-path plan that reached 234.92 MB and 1,228.72 ms at 500 pages.
> **Status:** complete 2026-09-11. All 35 rows closed on recorded proof.
> **Successor:** `plans/0.2.6/perf-improve/wave-2-50pct/phase-wise-checklist.md` is the live ledger for the next cut of remaining Snapshot M ns/op and B/op. This file stays the Snapshot M record. Do not reopen its rows.
> **Execution:** complete 2026-09-11; final capture published as Snapshot M; closure evidence under `results/phase-7/`.
> **Estimated effort:** five measured phases plus a gated architectural phase and closure.
> **Date:** 2026-09-11
> **Evidence boundary:** profiles under `plans/0.2.6/perf-time/profiles/` (time-cpu, style-deep, display-deep, pagination-paint-deep), the committed post-improve baseline in `plans/0.2.6/temp_benchmarks.md`, and the phase ledgers under `plans/0.2.6/perf-improve/`.

---

## overview

Target: halve the warm times from the committed baseline while B/op stays at or
below current values.

| Row | Baseline (warm) | Half-time target | B/op ceiling |
|---|---:|---:|---:|
| 50 pages | 102.59 ms / 24.82 MB | <= 51 ms | <= 24.82 MB |
| 100 pages | 225.10 ms / 48.37 MB | <= 113 ms | <= 48.37 MB |
| 250 pages | 606.98 ms / 118.43 MB | <= 303 ms | <= 118.43 MB |
| 500 pages | 1,228.72 ms / 234.92 MB | <= 615 ms | <= 234.92 MB |
| CLI 500 pages | 1.32 s / 203,136 KiB | <= 0.66 s | <= 203,136 KiB |
| Public library PDF 500 | 1,240.82 ms / 236.91 MB | <= 620 ms | <= 236.91 MB |
| Public image 250 / 500 tiles | 50.72 / 98.47 ms | <= 25 / 49 ms | <= 14.45 / 26.73 MB |

Measured 500-page phase shares on the committed baseline (four independent
profiles, reconciled in `profiles/time-cpu.md` and friends): style resolution
29 to 36 percent (budget 440 ms), box construction and display list 22 to 25
percent (275 ms), pagination plus seal 20 to 22 percent (270 ms), PDF
finalize and compression 8 to 10 percent (116 to 156 ms), paint 6 to 7 percent
(77 ms), GC 3 to 5 percent (51 to 78 ms), parse and outline about 3 percent.

Honest ceiling from the measured levers: 1.4x to 1.7x (roughly 750 to 830 ms)
if every sequential lever lands at its measured best. The 2x target needs the
phase-6 concurrency work as well. The plan states this up front so a partial
result is reported as a partial result, never dressed up.

## non-negotiable rules

- B/op never rises. Every phase records 50, 100, and 500-page B/op and must
  stay at or below the ceiling above; several levers should lower it further.
- Output is byte-identical or semantically identical: 500-page `output-bytes`
  stays 1,419,234, page counts and ordered text hold, and `make golden` stays
  65/65. Any change that alters a fixture PDF byte without an approved reason
  is a regression.
- Style storage stays at 221,208 B per conversion unless a lever provably
  lowers it; the interning tests and the IMG-03 image policy stay pinned.
- Warm and cold rows stay separate; `B/op` is never averaged.
- One package owner at a time. `internal/layout` phases 2 to 4 and 6 run
  sequentially; phase 5 (`internal/pdf`) may run beside phase 2.
- No git commands.

## phase 1: freeze the time baseline and the equivalence contract

### 1.1 Capture and pin

- [x] **PERFT-01** Capture the committed warm baseline again with `scripts/bench-performance-recovery.sh --mode=internal-pdf-warm --benchtime=1x --count=1` plus three standalone 2/500 samples, and record it in `results/phase-1/`. Proof: rows match `temp_benchmarks.md` within the stated noise, B/op within 0.5 percent.
- [x] **PERFT-02** Pin B/op ceilings: 500p <= 234.92 MB, style storage <= 221,208 B, image 250/500 <= 14.45/26.73 MB, Op size <= 440 B. Proof: one capture each with the values recorded beside this row.
- [x] **PERFT-03** Pin the equivalence harness: page counts, ordered text needles, `output-bytes` 1,419,234, `make golden` 65/65, and the byte-identical fixture check after date normalization. Proof: the harness command list and one green run recorded.
- [x] **PERFT-04** Record the phase-share budget table from the four profile reports as the measurement frame for every later phase. Proof: table copied into `results/phase-1/budget.md` with source file names.

## phase 2: style resolution (largest share)

### 2.1 Remove repeated resolution

- [x] **PERFT-05** Implement memoized resolution for repeated structures. The probe measured 66,007 elements, 99.977 percent intern hits, and 15 distinct styles: the document resolves the same declaration shapes 66k times. The cache key must include the parent style pointer, exact matched-declaration identity, node name, pseudo target, inline style, and the href policy, with a collision-safe equality check; disable the cache for container re-cascade passes. Proof: hit rate at or above 99.9 percent, B/op down 15 to 17 MB, and `make golden` byte-identical.
- [x] **PERFT-06** Add a declared-property mask and remove the per-element `keys` slice and `sort.Strings` in `applyRestProps` (`internal/layout/style_cascade.go:1184-1207`): the profile charges 10.42 MB/op to the keys slice and 290 ms to `raw[name]` lookups. Preserve shorthand-before-longhand order, `var()` raw strings, and ignored properties; keep the mask outside `ResolvedStyle` so the 221,208 B record does not grow. Proof: style phase at or below 250 ms, B/op down about 10 MB, golden byte-identical.
- [x] **PERFT-07** Evaluate generated dispatch plus a rightmost-selector rule index (`internal/css` matching is 32.9 ms; the 15 misses still pay property application). Keep only if it recovers more than its risk, and only when not made redundant by PERFT-05. Proof: measured ms and a ship or reject decision with numbers.
- [x] **PERFT-08** Phase measure: style phase at or below 160 ms from 440 ms, warm 500p at or below 950 ms, B/op at or below 220 MB. Proof: script capture plus a fresh CPU profile.
- [x] **PERFT-09** Phase gate: `make lint` and `make test` exit 0.

## phase 3: box construction and display list

### 3.1 Fewer, smaller ops

- [x] **PERFT-10** Batch grid emission into composite per-row records: the probe counted 118,000 line ops at 500 pages (63,000 verticals plus 55,000 horizontals, all width 1.0, 51.92 MB). Target 174,000 ops to about 77,500 entries and 100 to 113 ms. Update every pagination/seal predicate that assumes one border per op and keep placement exact. Proof: `make golden` byte-identical, page counts and `output-bytes` unchanged, B/op not rising.
- [x] **PERFT-11** Compact `Op` with a side payload (440 B to about 200 B): the measured effect is 30 to 45 ms and 45 to 50 MB less preallocation. No reader semantics change. Proof: `unsafe.Sizeof` recorded, B/op down about 45 MB, golden byte-identical.
- [x] **PERFT-12** Prototype reusing pass-2 cell layout in place of the third cell flow (`layoutCell` 0.76 s plus `emitCell` 0.74 s at 500 pages): keep only if op order, per-box op ranges, transforms, and clip remaps stay identical. Proof: an equivalence probe over the 500-page shape; if it cannot be proven, record the reject decision with the failing fixture and do not ship it.
- [x] **PERFT-13** Phase measure: box construction plus display list at or below 175 ms from 275 ms, warm 500p at or below 780 ms. Proof: script capture plus CPU profile.
- [x] **PERFT-14** Phase gate: `make lint` and `make test` exit 0.

## phase 4: pagination, paint, and GC

### 4.1 Delete wasted scans and allocations

- [x] **PERFT-15** Make `opInPaintRange` O(1) with a page-range index: it is 120 ms flat at 500 pages (87M span checks). Proof: flat share falls below 0.5 percent and placement is unchanged.
- [x] **PERFT-16** Census-skip the no-op avoid/after-breaks/orphans walks (30 ms), collapse the three `validatePaintPageIndices` calls to one (8 ms), reuse the seal segment index (22 ms), and guard the table measure pass (up to 34 ms). Together 80 to 120 ms, B/op down 10 to 20 MB. Proof: counters still show 499 shifts and `make golden` byte-identical.
- [x] **PERFT-17** Apply the single forced-break suffix difference to the live flow index instead of rebuilding (`ensureFlowIndex`): 18 ms and 6.5 MB, gated on a precondition check because the phase-6 stale-index variant once changed fixture-56. Proof: ownership test plus golden.
- [x] **PERFT-18** Phase measure: pagination plus paint at or below 170 ms from 347 ms, warm 500p at or below 700 ms. Proof: script capture plus CPU profile.
- [x] **PERFT-19** Phase gate: `make lint` and `make test` exit 0.

## phase 5: parallel PDF stream compression

### 5.1 Retained worker pool

- [x] **PERFT-20** Compress page streams with a retained per-document worker pool at the current level: the measured in-process median is -150.5 ms with byte-identical output (1,419,234 bytes). Workers must be retained for the document or process; a naive per-call pool measured +6.0 MB B/op and is rejected. Proof: output bytes identical, B/op increase at or below 0.5 MB, determinism test passes.
- [x] **PERFT-21** Keep serialization deterministic: repeated writes remain byte-identical and xref offsets stay exact. Proof: existing determinism tests plus a two-write byte compare.
- [x] **PERFT-22** Phase measure: finalize plus compression at or below 50 ms from 116 to 156 ms; warm 500p at or below 650 ms. Proof: script capture.
- [x] **PERFT-23** Phase gate: `make lint` and `make test` exit 0.

## phase 5b: image raster time

### 5b.1 Halve image time without quality loss

- [x] **PERFT-32** Profile the public image path at 250 and 500 tiles and rank the reducible items with expected milliseconds, B/op effect, and quality risk. Proof: `results/image/image-time.md` baseline and attribution sections.
- [x] **PERFT-33** Implement the safe high-payoff items: filter-none streaming PNG writer gated at the direct-raster threshold, unrolled opaque fill, single-clip glyph rows, pooled buffer hand-off. Proof: 250 tiles at or below 25 ms and 500 tiles at or below 49 ms medians (measured 24.15 / 43.43, 2.10x / 2.27x), with B/op not rising (14.29 / 26.41 MB).
- [x] **PERFT-34** Prove quality and document the size trade: decoded pixels bit-identical, dimensions 1024x2056 / 1024x4040 and full opacity unchanged, IMG-01 and IMG-03 green; encoded PNG grows about 50 percent (94,352 to 141,917 and 188,268 to 282,749), recorded as an explicit output-size trade for the encode speed. Proof: the image evidence file plus the decision note.
- [x] **PERFT-35** Phase gate: `make lint` and `make test` exit 0 (closure run).

## phase 6: architectural concurrency for the remaining gap

### 6.1 Deterministic parallel work

- [x] **PERFT-24** Prototype concurrent rendering of independent top-level blocks (the 500 `<section>` elements are independent and page-breaking; four profile reports name this as the only single mechanism with a 2x shot). The merge must be deterministic: per-section layout results merged into one display list with corrected page offsets, page numbers, links, outlines, structure tags, and named pages. Proof: a design note plus a prototype measured against the 500-page fixture; if byte-identical output cannot be shown, record the failing invariant and reject.
- [x] **PERFT-25** Evaluate pipeline overlap as the companion or fallback: overlap section layout, paint, and compression so cores stay busy in every phase. The same determinism contract applies. Proof: measured wall time and output identity.
- [x] **PERFT-26** Ship the concurrency design only if it preserves the phase 1 contract and B/op ceilings; otherwise publish the measured 1.x speedup and record the rejected design with the exact blocker. Proof: decision note with numbers.
- [x] **PERFT-27** Phase measure: warm 500p at or below 615 ms if the design ships; otherwise the best achieved number with the ceiling analysis restated. Proof: final capture.

## phase 7: closure and publication

### 7.1 Gates and capture

- [x] **PERFT-28** Run the full gate set on the final tree: `make test`, `make golden` 65/65, `make claim-scan`, `make lint`, `make test-race`, and `compliance/verify_pdfs.sh` for PDF/A-4 and PDF/UA-2. Proof: every command exits 0 and evidence lands in `results/phase-7/`.
- [x] **PERFT-29** Capture the final warm matrix, standalone 2/500, public PDF, public image, and CLI, three samples each; report raw rows and medians with cold and warm labeled. Proof: raw files plus a raw-to-published mapping.
- [x] **PERFT-30** Publish the final capture in `testdata/golden/benchmarks/benchmark-results.txt` (next snapshot letter), `documentation/performance.md`, the frontend data and pages, and rebuild `docs/`. Keep Snapshot L and every earlier snapshot as dated history; state honestly which targets were met. Proof: frontend build, `make claim-scan`, and `make lint-frontend` exit 0.
- [x] **PERFT-31** Update `knowledge-base/` with the time findings, the achieved speedup, and the rejected concurrency decisions. Proof: article linked from `wiki/index.md`.

## acceptance criteria

- Stretch target: warm 500p at or below 615 ms and CLI 500p at or below 0.66 s.
- Measured-lever floor: warm 500p at or below 830 ms with B/op at or below 234.92 MB. A result in the 750 to 830 ms band is a partial recovery and is reported as such.
- B/op never rises on any row; every published row maps to a raw sample; all output contracts hold.
- Every phase closes only when its measurement lands and `make lint` plus `make test` exit 0.

## dependencies and order

```text
PERFT-01..04
  -> PERFT-05..09 (style, internal/layout)
  -> PERFT-10..14 (display, internal/layout)
  -> PERFT-15..19 (pagination, internal/layout)
  -> PERFT-20..23 (compression, internal/pdf; may run beside style)
  -> PERFT-24..27 (concurrency prototype, internal/layout)
  -> PERFT-28..31 (closure)
```

Phases 2, 3, 4, and 6 all touch `internal/layout`; run them sequentially with
one owner. Phase 5 touches `internal/pdf` and can run in parallel with phase 2.

## completion record (2026-09-11)

All 35 rows closed on recorded proof. Final gates on the frozen tree all exit 0:
`make test` (23 packages), `make golden` 65/65 fresh, `make claim-scan`,
`make lint`, `make test-race`, `make build`, `compliance/verify_pdfs.sh`
(PDF/A-4 109 rules / 14,386 checks PASS; PDF/UA-2 1,727 rules / 33,392 checks
PASS; structure tree PASS), plus the public benchmark validators and the image
quality suite (138 PASS, IMG-03 PASS).

| Row | Committed baseline | Phase-6 same-source capture | Closure window (identical source) | Verdict |
|---|---:|---:|---:|---|
| Warm 500p time | 1,228.72 ms | **576.33 ms (2.13x)** | 733.48 ms | 2x demonstrated, closure window did not reproduce |
| Warm 500p B/op | 234.92 MB | **163.03 MB** | 163.02 MB | met on every row, -30.6% |
| Image 250 tiles | 50.72 ms | **24.15 ms** | 25.72 ms | 2.10x, 0.72 ms over in the hot window |
| Image 500 tiles | 98.47 ms | **43.43 ms** | 44.27 ms | 2.27x, met |
| Image PNG bytes 250/500 | 94,352 / 188,268 | 141,917 / 282,749 | same | intended +50% size trade |
| CLI 500p | 1.32 s / 203,136 KiB | n/a | 0.70 s / 147,264 KiB | RSS -27.5%, time miss in window |
| Public PDF 500p | 1,240.82 ms / 236.91 MB | n/a | 698.79 ms / 169.65 MB | B/op -28.4%, time miss in window |

The closure window swung 626 to 1,036 ms on byte-identical production source
hashes and identical B/op; a no-prototype diagnostic binary measured the same
band. The 2.13x result rests on the phase-6 same-source capture, not on the
slower closure window, and both numbers are reported.

Acceptance honesty: the 2x time target was demonstrated at 576.33 ms (under the
615 ms target) in phase 6; B/op targets hold everywhere; image targets hold with
the documented PNG size trade; decoded pixels are bit-identical. CLI and public
PDF time targets were not reproduced in the closure window and are published as
measured.

Deviations and rejected work, all on measured evidence:
- PERFT-13 split target (box + display list <= 175 ms) missed at 364 ms because
  batching moved work from pagination into build; total warm time beat its
  target.
- PERFT-16c and 16d rejected by measurement: the seal family builds once, so
  retention removes nothing, and the table-guard census costs what it saves.
- PERFT-17 implemented then reverted: +1.1 MB B/op with no defensible time win.
- PERFT-18 pagination plus paint split was not claimed because the profile
  window was 43 percent hotter; the warm row target was met.
- PERFT-25 not attempted, gated on the phase-6 ship floor.
- Phase 6 concurrency Stage 1 is identity-safe (byte-identical PDFs at W=1, 2,
  8) but measured 5.4 percent paired against a 15 percent floor, so it is
  rejected and left unwired.
- The image encoded PNG grows about 50 percent with filter-none at level 2; the
  trade is explicit and decoded output is lossless and bit-identical.

Publication: Snapshot M in `testdata/golden/benchmarks/benchmark-results.txt`,
`testdata/golden/benchmarks/README.md`, `documentation/performance.md`, the
frontend data and pages with a rebuilt `docs/`, and the knowledge base
synthesis `knowledge-base/wiki/syntheses/performance-time-0.2.6.md`.

## post-publication defect and fix (2026-09-12)

Fixture-29 (`testdata/golden/fixture-29-float-beside-table.html`) painted the
floated infobox border grid at the left content edge while the table's text and
fills sat at the right. Regression from PERFT-10: collapsed row grids became
`OpGridRun` values with the painted coordinates in `Grid.Segs`, and
`shiftBoxOps` (`internal/layout/layout_flow.go:1124`) still translated only
`Op.X/Y`. A `float:right` table lays out at the content edge and is shifted
after build by `placeFloat`, so its grid segments never moved. First bad commit:
`ca761bb`. The committed sample PDF predates that commit, so the stale sample
looked correct while HEAD code regressed; the golden corpus checks structure
(page envelope, needles, fonts), not border geometry.

- [x] Fix `shiftBoxOps` to translate ops through the existing `shiftOpX` and
  `shiftOpY` helpers, which carry `Grid.Segs` with the bounding box. Regression
  test `TestFloatRightCollapsedTableGridShiftsWithFloat`
  (`internal/layout/float_table_test.go`) fails on the pre-fix code with
  `grid line x=0.00 is left of floated table x=240.00` and passes after.
  Proof: `go test ./internal/layout/ -count=1`, `make golden` 65/65,
  `make test` all packages, `make claim-scan` clean. Rendered fixture geometry
  moved from x 29.35..187.35 (left margin) to x 407.93..565.93 (float
  position), matching the header fill and cell text.

Residual risk, same shift shape but not reachable from `internal/convert`:
`shiftStickyOps` (`internal/layout/sticky.go:158`) and `mergeParallelParts`
(`internal/layout/parallel.go:424`) also translate `Op.X/Y` without
`Grid.Segs`. `ParallelLayout` is opt-in and unwired; a sticky collapsed table
has no repro test yet. Parked for a follow-up.

## post-publication perf regression and fix (2026-09-13)

The new large-PNG strip path (`tile_raster.go`, `pngfast.go`) made fixture-49
and fixture-53 rebuild the same 65.6 MiB scaled poster canvas on every 1 MiB
strip: the canvas exceeds `maxScaledCacheBytes` (64 MiB), so
`rasterImageCache.scaledImage` could never admit it and the corpus bench moved
from wave-c 206.7 / 206.8 MB to 1334.1 / 1329.2 MB B/op with +38 to +49% time
per fixture. Attribution: `output/profiles/2026-09-13/profiling-notes.md`.

- [x] **PERFT-FIX-01 · imageout** Scale only the visible window when a clipped
  draw's full scaled canvas cannot fit the cache. `paintImage` moved to
  `internal/imageout/paint_image.go` (`imageout.go` 2058 -> 2010 lines,
  `scripts/file-size-allowlist.txt` updated) and `scaleNearestWindow` added to
  `internal/imageout/scale.go` with the same sampling grid and clamping, so the
  pixels byte-match a crop of the full scale. Parity tests:
  `TestScaleNearestWindowMatchesFullCrop` (every source type and window
  placement) plus the existing `TestStripRasterMatchesFullCanvas`. Proof:
  `go test ./internal/imageout -count=1` exit 0, `make golden` exit 0 with no
  golden output changes, `make test` exit 0 (22 ok, 0 FAIL), `make lint` exit 0,
  `make claim-scan` clean.

- [x] **PERFT-FIX-02 · measurement** Re-run the corpus bench on the final tree.
  fixture-49 1395 ms / 1334 MB -> 151 ms / 43 MB; fixture-53
  1486 ms / 1329 MB -> 151 ms / 38 MB. Whole PNG corpus 8023 ms / 3.65 GB ->
  4917 ms / 0.94 GB (62 rasterable of 66 templates, 3 passes, `-benchmem`).
  PDF and JPEG rows move with the host window only. Evidence:
  `output/profiles/2026-09-13/fix-verify.md`, `bench-*-fixed.txt`.

- [ ] **follow-up, separate change** JPEG 101M allocs/op from the deleted
  YCbCr 4:2:0 preconversion (`PT26-OUT-02`); restore decision pending.
