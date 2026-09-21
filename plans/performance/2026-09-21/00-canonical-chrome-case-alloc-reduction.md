# Performance - Chrome 40-case allocation reductions

> **Parent:** `plans/0.2.7/chrome-flex-interactions/00-canonical-chrome-flex-interaction-plan.md` (case corpus). Allocation profile captured 2026-09-21 in the parent worktree; printed in session, not committed.
> **Status:** Complete. Phases 0 to 2 shipped; Phase 3 deferred by measurement (about 1% waste vs a 5% gate); Phase 4 gates all green on the final tree.
> **Estimated effort:** 2 to 3 sessions

---

## Overview

The 40 Chrome interaction cases under `test/chrome/cases/` had no allocation
profile. `BenchmarkChromeCasePDFs` now renders every manifest case through
`convert.Run` so the corpus can be profiled with pprof. The first capture
(2026-09-21) attributed 694.96 MB of allocations across 400 renders (40 cases
x 10) to two sites plus one wasted pass:

| Site | Size | Share | Cause |
|------|------|-------|-------|
| `layout.(*styleStore).append` | 248.92 MB | 35.8% | `ResolvedStyle` is 4056 B; chunks are 64 records (253.5 KiB); ~152 records per case; the tail chunk is mostly empty |
| fresh deflate states via `pdf.flateBytes` | 229.20 MB | 33.0% | `sync.Pool` is emptied by GC (probe: one `runtime.GC()` forces a fresh 817 KiB state); single-page docs use this weak path |
| discarded `IndependentBlocksForOptions` style pass | 106.12 MB | 15.3% | `convert.go:601` resolves styles to probe independent blocks; the layout then resolves again; | 
| everything else | 110.7 MB | 15.9% | long tail: `bytes.growSlice`, regexp, HTML parse, layout build |

This plan ships the reductions in risk order and gates every step on the
golden corpus. No engine behavior change is allowed; the metric is B/op and
allocs/op on `BenchmarkChromeCasePDFs`.

## Executive Summary

Targets:

1. Phase 1 retires the GC-cleared serial flate pool: expected to remove
   close to the full 229 MB (33%).
2. Phase 2 resolves the document cascade once and shares it across the
   independent-blocks probe, layout, smart-shrink, and independent-block
   assembly: expected to remove the 106 MB probe pass (15%).
3. Phase 3 is decision-gated on the post-Phase-2 profile. The style chunk
   tail can waste up to 39 records x 4056 B per case; if that stays at or
   above 5% of corpus allocations, the chunk sizing changes under a
   measured no-regression rule.
4. Phase 4 runs the release gates (`make test`, `make golden`, `make lint`,
   `make claim-scan`) once and syncs the knowledge base.

## Phase 0: Harness and baseline

### 0.1 Harness in this worktree

- [x] `test/chrome/profile_bench_test.go` defines `BenchmarkChromeCasePDFs`
      over all 40 manifest cases, using the shared `chromeCasePDFRequest`
      helper from `test/chrome/pdf_output_test.go`.
      Proof: `go test ./test/chrome -count=1` exits 0 (0.137s, 2026-09-21).
- [x] `golangci-lint run ./test/chrome/...` exits 0 (2026-09-21).

### 0.2 Baseline capture (this worktree, engine at HEAD b229151)

- [x] Warm corpus totals from
      `-test.benchtime=1x -test.count=10 -test.benchmem`:
      **64.3 MB B/op, 70,091 allocs per 40-case pass**
      (`/tmp/opencode/perf-alloc/baseline-run.txt`). Parent-worktree
      reference was 65.2 MB / 70,329; the 1.4% gap is run-to-run sample
      variance, not a source difference.

### 0.3 Baseline attribution

- [x] Memprofile capture
      (`/tmp/opencode/perf-alloc/baseline.pprof`, rate 65536):
      total **713.50 MB**; top sites: `styleStore.append` 248.86 MB flat
      (34.9%), `flate.NewWriter` 201.24 MB flat / 245.15 MB cum (34.4%),
      `bytes.growSlice` 26.07 MB, `bufio.NewWriterSize` 25.97 MB (inside
      flate state creation).
- [x] Flate pool probe (overlay test, no tree change): first call
      817,592 B, next 9 calls 2,016 B, after one `runtime.GC()` 3,432 B
      (state survives one cycle in the victim cache), after two GCs
      817,592 B again. A GC can also strand the state on another P, which
      is why the corpus capture shows roughly one recreation per
      conversion. The pool never survives two GC cycles.

## Phase 1: Retained serial flate state (`internal/pdf`)

### 1.1 Implementation

- [x] Replaced the `sync.Pool` serial path in `internal/pdf/pdf.go` with a
      GC-surviving `atomic.Pointer[flateState]` (`flateSerial`, pdf.go:1770
      to 1805). The retention cap is kept and renamed
      `maxRetainedFlateBufferSize` (16 MiB). Parallel page workers keep
      their own states in `internal/pdf/flate_parallel.go`.
- [x] Comment records the measured reason: the pool never survives two GC
      cycles (probe: 817,592 B reallocated after the second GC) and the
      corpus capture paid a fresh state per conversion (245 MB cum).

### 1.2 Regression test

- [x] `internal/pdf/flate_serial_test.go`:
      `TestFlateStateRetainedAcrossGC` warms the state, runs two
      `runtime.GC()` cycles, then requires the next call to allocate at
      most 256 KiB (fresh state is about 817 KiB).
      Proof: `go test ./internal/pdf -run 'TestFlateState' -count=1 -v`
      exits 0; both flate tests pass.

### 1.3 Phase gate

- [x] `go test ./internal/pdf/... -count=1` exits 0 (2.494s).
- [x] Fresh chrome capture (`/tmp/opencode/perf-alloc/p1-run.txt` and
      `p1.pprof`): total alloc_space **713.50 MB -> 485.98 MB (-31.9%)**;
      warm corpus **64.3 MB -> 43.8 MB B/op**; allocs 70,091 -> 69,340.
      `flate.NewWriter` flat 201.24 MB -> 18.35 MB, cum 245.15 MB ->
      23.00 MB.
      Note: the row expected under 5 MB; the 23 MB residual is one-time
      and out of this phase's scope: the 3-page fixture
      `wpt-break-nested-float-print` starts the 8 retained parallel workers
      (about 6.4 MB once), and `image/png` encoders create their own
      writers per rasterized SVG. `bufio.NewWriterSize` 25.71 MB is the
      `pdf.(*Document).writeTo` output buffer, not flate.
- [x] CPU sanity: mean of per-case warm ns/op 1.43 ms -> 1.38 ms, within
      host noise, no regression observed.
- [x] `go test -p 1 -parallel 2 ./internal/convert/ -run TestGoldenCorpus
      -count=1` exits 0 (4.986s).

## Phase 2: Resolve the cascade once and share it (`internal/layout`, `internal/convert`)

### 2.1 Layout API

- [x] `internal/layout/layout_context_styles.go` (new, 89 lines) holds
      `ContextWithStyles` and the shared `layoutContextWithStyles`
      body. It reuses a non-nil caller map only when
      `!css.HasContainerRules(opts.Sheets)` (guard at line 73); otherwise
      the full container remount runs. `layoutContext` (layout.go:1154) is
      now a 6-line adapter, so layout.go shrank 2411 -> 2362 lines and
      `scripts/file-size-allowlist.txt` records the new count.
- [x] Zoom independence verified: `opts.Zoom` only feeds `engine.scale`
      (layout.go:1174 after the move) and `resolveStylesWithContext` never
      reads Zoom. The wrapper doc comment records it.

### 2.2 Convert wiring

- [x] `internal/convert/convert.go:600-628`: one `layout.ResolveStyles`
      call feeds `layout.IndependentBlocks` and the `layoutBody` closure
      (`ContextWithStyles`). A resolve error sets the map to nil and
      falls through to `layoutBody`, which reports the same error.
- [x] `internal/convert/page_blocks.go:12-30`: `renderIndependentBlocks`
      takes the shared map and no longer resolves; the dead `root` param
      was dropped.
- [x] `IndependentBlocksForOptions` deleted
      (`internal/layout/independent_blocks.go`, 148 -> 132 lines): its only
      caller was convert.go:601 and no test called it. `IndependentBlocks`
      stays exported and tested.

### 2.3 Phase gate

- [x] `gofmt -l`, `go build ./...`, and `go test -p 2 -parallel 2
      ./internal/layout/... ./internal/convert/... -count=1` all exit 0.
- [x] `go test ./test/chrome -count=1` exits 0 (0.080s; both chrome tests
      ran, not skipped).
- [x] Fresh chrome capture (`/tmp/opencode/perf-alloc/p2-run.txt`,
      `p2.pprof`): total alloc_space 485.98 -> **327.88 MB**; warm corpus
      43.8 -> **28.0 MB B/op** (baseline 64.3 MB, **-56.4%**); allocs
      69,340 -> 63,716 (baseline 70,091, -9.1%); `styleStore.append`
      249.11 -> 100.65 MB.
- [x] CPU sanity: mean per-case warm ns/op 1.38 -> 1.20 ms, improved by
      fewer GCs; no regression.
- [x] `go test -p 1 -parallel 2 ./internal/convert/ -run TestGoldenCorpus
      -count=1` exits 0 (4.488s); `scripts/check-file-size.sh` clean.

## Phase 3: Style store tail waste (`internal/layout`) - decision gated

### 3.1 Measurement

- [x] Post-Phase-2 profile: `styleStore.append` 100.65 MB / 400 renders =
      about 252 KiB per case = about 62 records at 4056 B. One 64-slot
      chunk per case covers it, so at most 2 slots are wasted: about
      8 KiB per case, about 3.2 MB per 400-render pass = **about 1.0% of
      the 327.88 MB corpus total**. The 5% gate is not met.

### 3.2 Fix

- [~] Deferred: waste is about 1% of corpus B/op, below the 5% gate. No
      chunk-sizing change was made. Reopen only if record size or the
      number of resolution passes grows again; the measurement lives in
      3.1.

### 3.3 Phase gate

- [~] Not applicable; no code change was made (3.2 deferred).

## Phase 4: Closure

- [x] `make test` exits 0 on the final tree (23 packages `ok`, no `FAIL`;
      `/tmp/opencode/perf-alloc/make-test2.log`).
- [x] `make golden` exit 0 with a fresh `-count=1` run (4.764s;
      `/tmp/opencode/perf-alloc/final-golden.log`).
- [x] `make lint` exit 0. The first run reported two findings: `revive`
      stutter on the new API name and `paralleltest` on the GC test.
      Fixed by renaming `LayoutContextWithStyles` to `ContextWithStyles`
      and adding a documented `//nolint:paralleltest` because the MemStats
      delta must not run beside sibling parallel tests.
- [x] `make claim-scan` exits 0 (`claim-scan: clean`).
- [x] Final chrome capture on the frozen tree
      (`/tmp/opencode/perf-alloc/final-run.txt`, `final.pprof`): total
      alloc_space **326.98 MB** (baseline 713.50 MB, -54.2%); warm corpus
      **28.0 MB B/op, 63,699 allocs** (baseline 64.3 MB / 70,091; -56.4% /
      -9.1%); top site `styleStore.append` 100.52 MB. Phase deltas:
      P1 713.50 -> 485.98 MB, P2 485.98 -> 327.88 MB, final 326.98 MB.
      ns/op is host-noise sensitive: 1.47 ms in this capture, 1.20 ms in
      the Phase 2 capture, B/op is the stable metric.
- [x] `plans/README.md` indexes this ledger;
      `knowledge-base/wiki/log.md` and
      `knowledge-base/wiki/syntheses/performance-chrome-alloc-2026-09-21.md`
      record what shipped and the numbers.
      `documentation/architecture/07-layout.md` updated for the deleted
      wrapper and the 132-line `independent_blocks.go`.

## Dependencies

- Phase 1 and Phase 2 touch different packages but are sequenced so each
  phase gets a clean allocation attribution.
- Phase 3 depends on Phase 2 (same package, same capture).
- Phase 4 depends on all phases.
- Ownership: one writer per package at a time. `internal/pdf` writer and
  `internal/layout`/`internal/convert` writer never run in parallel on this
  tree. No agent runs `make lint` or `make test` until Phase 4.

## Guardrails

- Golden corpus is the behavior contract: `make golden` after every phase
  that changes `internal/layout`, `internal/convert`, or `internal/pdf`.
- No `git add` / `git commit` / `git push`; the only git command used is the
  worktree creation. Work ran in
  `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf-perf` on
  `perf/chrome-case-alloc-reductions`, then was copied file-for-file into
  `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf` on the existing
  `feature/027-next-72-with-chrome-test` working tree at the user's request
  (2026-09-21, no branch change). The worktree remains a backup snapshot.
- Benchmark numbers are dev-loop measurements on a shared host; comparisons
  use the same command and count within one session.
- Do not grow `internal/layout/layout.go` or `style_properties.go`; extract
  when needed.
