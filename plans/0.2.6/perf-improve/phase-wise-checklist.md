# 0.2.6 performance improvement - warm-path time and remaining allocation

> **Parent:** `plans/0.2.6/perf-review/phase-wise-checklist.md` - the recovery plan that landed the style interning, PDF lifetime, and image direct-raster fixes.
> **Status:** complete 2026-09-11. All 32 rows closed on recorded proof.
> **Estimated effort:** six implementation phases plus closure, each measured before the next.
> **Date:** 2026-09-11
> **Evidence boundary:** current source, the 2026-09-11 profiling captures under `profiles/`, the 2026-09-11 perf-review captures, and the committed 2026-08-19 Snapshot I. No Git command ran while writing this plan.
> **Execution:** complete 2026-09-11. Per-phase evidence under `results/`; final capture published as Snapshot L.

---

## overview

The recovery plan fixed the large allocation regressions: style storage is now a
constant 221,208 B from 2 to 500 pages (was 228.3 MB at 500 pages) and the
image direct-raster branch meets or beats its targets. Warm timing and the
remaining allocation are still well above the 2026-08-19 warm matrix
(Snapshot I, same host: WSL2, i7-13700HX, 24 CPUs).

| Workload | Snapshot I (2026-08-19) | Warm now (2026-09-11) | B/op now |
|---|---:|---:|---:|
| 2 pages warm | 3.58 ms / 2.21 MB | 4.40-5.16 ms | 1.77-2.59 MB |
| 10 pages | 14.86 ms / 6.08 MB | 24.30 ms | 7.46 MB |
| 50 pages | 84.17 ms / 24.98 MB | 141.10 ms | 32.37 MB |
| 100 pages | 157.69 ms / 48.73 MB | 272.86 ms | 64.51 MB |
| 250 pages | 449.51 ms / 119.36 MB | 841.12 ms | 160.34 MB |
| 500 pages | 1,009.80 ms / 237.76 MB | 1,246.05 ms standalone, 1,633.08 ms in matrix | 321.10 MB |

The first 1x row of a fresh process also carries a 6.99 MB / 709-allocation
default-font charge and about 4 ms of first-run time. Every claim below states
whether it is a cold row or a warm row; the two are never mixed.

Recovery target: the Snapshot I warm matrix. Intermediate gates per phase are
set from the measured profile shares, not from optimism.

## executive summary

Five measured sources carry the remaining gap, in payoff order:

1. Table-border sealing: 67.2 MB allocated at 500 pages (36.1 MB segment
   appends at `paint_pagination_seal.go:330`/`:339`, 31.1 MB per-Y map appends
   at `:304`/`:305`/`:311`, 128,776 growth allocations) and 5.0 percent of warm
   CPU. The 0.2.4 profile shows this family at +51.2 MB today
   (`plans/0.2.4/pdf-generation-pprof.md`).
2. `beforeAlways` and `shiftBoxesForForcedBreak` (`paint_flow_breaks.go:420`):
   0.77 s cumulative, 4.17 to 5.95 percent CPU, superlinear from 0.01 s at
   50 pages to 0.54 s at 500 pages.
3. Display list: `(*engine).add` (`layout.go:861`) is 3.79 percent CPU appending
   472-byte `Op` values; the preallocation at `layout.go:1003` charges 95.6 MB
   with 13.47 MB waste (86 percent useful, so this lever is capped at 13.5 MB).
4. Page bucketing: `bucketOpsByPage` charges 28.28 MB with up to 24.1 MB churn
   across four full-op calls per conversion, and `buildFlowOpIndex` /
   `bucketOpsByPage` are one algorithm under two names.
5. Default-font cold charge: 6.99 MB across 709 allocations, of which about
   3.15 MB is a duplicated `bytes.Clone` in `internal/pdf/assets/assets.go` and
   `internal/pdf/faces.go parseNamed`.

GC is 16.4 percent of warm CPU (34 cycles, 176 to 221 ms mark+scan per warm
500-page iteration). Landing the allocation reductions above should cut both
B/op and GC time. Style interning and the image branch are regression-pinned in
phase 1 so no later phase can silently undo them.

Profiling evidence: `profiles/warm-pdf-cpu.md`,
`profiles/warm-pdf-alloc.md`, `profiles/historical-attribution.md`,
`profiles/cli-attribution.md`, with raw profiles under `profiles/raw/`.

## non-negotiable rules

- Do not change page counts, ordered text, fonts, links, outlines, structure
  tags, PDF profile output, image dimensions, transparency, or PNG semantics.
- Do not regress the recovered numbers: style storage stays at 221,208 B,
  250/500 tile B/op stays at or below 21.5/27.2 MB, 500-page B/op must fall
  from 321.10 MB and never rise.
- Warm and cold rows stay separate in every report. Never average `B/op`.
- One agent owns `internal/layout` paint paths at a time; phases 3 to 6 run
  sequentially.
- A phase closes only when its stated measurement lands and `make lint` plus
  `make test` exit 0.

## phase 1: freeze the warm and cold measurement contract

### 1.1 Capture the warm matrix and baseline

- [x] **PERF2-01** Add an explicit warm-matrix capture path that runs the full ascending matrix (2, 5, 10, 20, 50, 100, 200, 250, 500) in one process and labels the first row cold and the rest warm. Reuse `scripts/bench-performance-recovery.sh`; keep certified islands out. Proof: dry-run output names only existing commands; one capture lands under `results/`.
- [x] **PERF2-02** Record the pre-change warm baseline from the 2026-09-11 captures and the profiles in this ledger, with host header and exact commands. Proof: the capture file names and numbers match this ledger.

### 1.2 Pin the recovered fixes

- [x] **PERF2-03** Pin style storage: 221,208 B constant at 2 and 500 pages, and `styleStore.append` allocation at or under the profiled 223 KB. Proof: `go test ./internal/layout -run 'Test.*Style.*(Share|Reuse|Immutable)|TestStyleIntern' -count=1` plus a 500-page allocation profile line recorded beside this row.
- [x] **PERF2-04** Pin image bytes and quality: 250 tiles 21.5 MB, 500 tiles 27.2 MB, dimensions 1024x2056 and 1024x4040, IMG-03 quality assertions green. Proof: `scripts/bench-performance-recovery.sh --mode=public-image --sizes=250,500` and `go test ./internal/imageout -count=1` exit 0.
- [x] **PERF2-05** Pin output contracts: `make golden` 65/65, benchmark validators for page count plus ordered text, and `compliance/verify_pdfs.sh` for PDF/A-4 and PDF/UA-2. Proof: exit codes recorded at every phase closure.

### 1.3 Method rules

- [x] **PERF2-06** State cold or warm on every row: cold is the first size in a fresh process, warm is a later row in the same process or explicit `-count` iteration 2 and above. Proof: capture headers carry the label and this ledger uses the same wording.

## phase 2: default-font cold charge

### 2.1 Single copy and lazy parse

- [x] **PERF2-07** Remove the double `bytes.Clone` on the default faces path: the `internal/pdf/assets/assets.go` accessors copy once and `internal/pdf/faces.go parseNamed` copies again. Hand off a single owned copy. Proof: `go test ./internal/pdf -count=1` exits 0 and the cold 2-page B/op falls from 9,578,008 to about 6.4 MB; the 3,145,728-byte duplicate disappears.
- [x] **PERF2-08** Evaluate the eager 14-face parse in `internal/pdf/faces.go:42-135`. A fresh CLI process pays 6 to 11 ms in `LoadDefaultFaces`, and an empty document still costs 14.0 ms process wall (2-page CLI: 21.7 ms = 0.43 ms exec floor + 3.35 ms runtime init + 10.3 ms first-conversion fixed work + 7.7 ms for two pages). Make the parse lazy only if the golden corpus and the font fallback chain stay identical; keep eager parse if any fixture changes glyph selection. Target: cold fixed work falls from 10.3 ms and the empty-document floor falls below 14 ms. Proof: before/after cold rows plus `make golden`.
- [x] **PERF2-09** Measure the cold 2-page row before and after with `-benchtime=1x -count=2` so the one-time part is isolated. Proof: raw ns/op and B/op rows recorded; warm 2-page rows do not regress.
- [x] **PERF2-10** Phase gate: `make lint` and `make test` exit 0. Proof: exit codes recorded beside this row.

## phase 3: table-border sealing allocation

### 3.1 Capacity and reuse

- [x] **PERF2-11** Replace append-growth in `collectBorderSegmentOps` (`internal/layout/paint_pagination_seal.go:319`) and `collectTableBorderSegments` (`:294`) with exact pre-counting or a reusable scratch buffer scoped to one Paint. Target: the 67.2 MB seal family at 500 pages falls by about 40 MB.
- [x] **PERF2-12** Preallocate the per-Y map rows (`:304`, `:305`, `:311`) from the measured row count so the 128,776 growth allocations collapse. Target: another 15 to 25 MB at 500 pages.
- [x] **PERF2-13** Keep the Op append path allocation-neutral in `clusterVerticals`; reuse scratch where the profile shows the same growth pattern.
- [x] **PERF2-14** Keep golden fixtures 60, 31, and 29 plus the table pagination suites green; page-break placement must stay byte-stable. Proof: `go test ./internal/layout -run 'Test.*(Table|Seal|Pagination)' -count=1`, then `make golden` exit 0.
- [x] **PERF2-15** Phase measure: warm 500-page B/op at or below 266 MB from 321.10 MB, and the `capTablePageBreaks` CPU share down from 5.0 percent. Proof: script capture plus a fresh CPU profile.
- [x] **PERF2-16** Phase gate: `make lint` and `make test` exit 0.

## phase 4: forced-break shift CPU

### 4.1 Quantify, bound, preserve placement

- [x] **PERF2-17** Add test-only counters for `beforeAlways` iterations, `processBeforeAlwaysTarget` calls, and boxes scanned by `shiftBoxesForForcedBreak` (`internal/layout/paint_flow_breaks.go:420`); record the 500-page counts. Proof: counters printed by a test probe behind a test hook.
- [x] **PERF2-18** Make the shift scan sublinear: index boxes by Y or start each scan at the first moved box instead of scanning all boxes per break. Target: recover at least 4 percent of warm 500-page time (the measured flat 0.54 s). Final placements must be identical.
- [x] **PERF2-19** Keep golden fixtures 62, 31, and 60 and the pagination placement suites byte-stable. Proof: `make golden` plus the pagination suites exit 0.
- [x] **PERF2-20** Phase measure: warm 500-page time improves by at least 4 percent against the phase-1 baseline and `paginateOps` CPU share falls from 12.75 percent. Proof: CPU profile plus raw rows.
- [x] **PERF2-21** Phase gate: `make lint` and `make test` exit 0.

## phase 5: display-list append and capacity

### 5.1 Measure, then stay within the cap

- [x] **PERF2-22** Record `(*engine).add` (`internal/layout/layout.go:861`) flat CPU and the 472-byte `Op` size; confirm the callers named by the profile (`emitGridVerticals`, `emitInlineTextRun`, `rowGridStroker.hline`). Proof: fresh profile `-list` output beside this row.
- [x] **PERF2-23** Tune `estimateOpCapacity` (`layout.go:1236`) only under a hard rule: total 500-page B/op must fall, and a single growth reallocation is acceptable only when the total is lower than today. The measured ceiling for this lever is 13.47 MB; do not spend a phase on it if the measurement does not move.
- [x] **PERF2-24** If the append path is the better lever, reduce `Op` pointer-field weight or hoist repeated work in the three named callers without removing a field and without changing output. Proof: golden plus semantic checks; B/op must not rise.
- [x] **PERF2-25** Phase gate: `make lint` and `make test` exit 0.

## phase 6: page bucketing and duplicate scans

### 6.1 One algorithm, one buffer

- [x] **PERF2-26** Merge the duplicate implementations `buildFlowOpIndex` (`internal/layout/paint_flow_index.go:244`) and `bucketOpsByPage` (`paint.go:317`) behind one function, keeping the PDF-04 and PDF-05 lifetime rules. Proof: pagination and table-continuation suites exit 0.
- [x] **PERF2-27** Reuse the page index across the four full-op calls per conversion when Y is unchanged (`buildPagesAfterSplits`, `pageIndexedOps`, `ensureFlowIndex`), targeting the 28.28 MB with up to 24.1 MB churn. Proof: allocation profile line for the site plus `make golden`.
- [x] **PERF2-28** Remove duplicate all-op scans where the document does not need them (`validatePaintPageIndices` called three times, `clearStructureElements`, `fixedOpIndices`), keeping a cheap guard for the documents that do. Proof: ownership tests plus `make golden`.
- [x] **PERF2-29** Phase gate: `make lint` and `make test` exit 0.

## phase 7: closure and publication

### 7.1 Validate on the final tree

- [x] **PERF2-30** Run the full gate set: `make test`, `make golden`, `make claim-scan`, `make lint`, `make test-race`, and `compliance/verify_pdfs.sh` for PDF/A-4 and PDF/UA-2. Proof: every command exits 0 and the output lands under `results/`.
- [x] **PERF2-31** Capture the final warm matrix and B/op table; publish only when the phase targets hold, keeping the 2026-09-11 and 2026-08-19 rows as dated history. Proof: every published row maps to a raw capture.
- [x] **PERF2-32** Update `knowledge-base/` with the warm-path findings and the final deltas. Proof: article linked from `wiki/index.md`.

## acceptance criteria

- Warm 500-page B/op at or below 240 MB and warm 500-page time at or below 1.10 s, with style storage at 221,208 B and image rows unchanged. Snapshot I (237.8 MB / 1.010 s) stays the aspiration, not a gate that can be met by weakening checks.
- No change to page counts, ordered text, fonts, links, outlines, structure tags, profile validity, image dimensions, transparency, or PNG semantics.
- Every phase closure records `make lint` and `make test` exit 0.

## dependencies and order

```text
PERF2-01..06
  -> PERF2-07..10
  -> PERF2-11..16
  -> PERF2-17..21
  -> PERF2-22..25
  -> PERF2-26..29
  -> PERF2-30..32
```

Phases 3 to 6 all touch `internal/layout` paint paths. Run them sequentially
with one owner per package at a time. Phase 2 touches `internal/pdf` only and
could run beside phase 3 if a second owner is free, but the allocation captures
must not overlap on the same tree.

## completion record (2026-09-11)

All 32 rows closed on recorded proof. Final gates on the frozen tree: `make test`,
`make golden` (65/65), `make claim-scan`, `make lint`, `make test-race`, and
`compliance/verify_pdfs.sh` (PDF/A-4 109 rules / 796 checks PASS, PDF/UA-2 1727
rules / 1817 checks PASS) all exit 0. The public benchmark validators (page
count, ordered text, dimensions, tile count, transparency, negative cases) pass.

| Metric | Final | Snapshot I / 0.2.4 | Verdict |
|---|---:|---:|---|
| Warm 500p B/op, warm matrix | 234,923,560 B (234.92 MB) | 237.76 MB | acceptance `<= 240 MB` met, 1.2 percent below the 0.2.4 row |
| Warm 500p B/op, standalone median | 235,496,640 B (235.50 MB) | 237.76 MB | met |
| Public library PDF 500p B/op | 236,911,328 B (236.91 MB) | 236.85 MB | parity |
| Warm 500p time, warm matrix | 1,228.72 ms | 1,009.80 ms | acceptance `<= 1.10 s` not met, 17.5 percent better than the pre-improve 1,489.42 ms matrix |
| Warm 500p time, standalone median | 1,297.98 ms | 1,010 ms | not met |
| Public image 250 / 500 tiles B/op | 14.45 / 26.73 MB | 20.66 / 52.00 MB | met or beaten |
| CLI 500p time / RSS | 1.32 s / 203,136 KiB | 1,042 ms / 208,128 KiB | time not met, RSS parity |

Preserved throughout: style storage at 221,208 B per conversion, the interning
tests, the image direct-raster policy, 500-page `output-bytes` at 1,419,234, all
65 golden fixtures, and PDF/A-4 plus PDF/UA-2 validity. Every structural change
was proven by equivalence: 174,000 ops deep-equal, 54,503 box Ys bit-identical,
and every fixture PDF byte-identical after date normalization.

Remaining time gap, honestly stated: warm 500-page time is still about 22 to 29
percent above Snapshot I. The remaining measured hotspots are the border seal
residual, the remaining pagination index work, GC at roughly 10 to 16 percent,
and style resolution at 28.2 percent of the 500-page profile, which no phase in
this plan targeted.

Publication: Snapshot L in `testdata/golden/benchmarks/benchmark-results.txt`,
`documentation/performance.md`, the frontend data and pages, and the knowledge
base synthesis `knowledge-base/wiki/syntheses/performance-improve-0.2.6.md`.
