# Deferred 0.0.3 - 500-Page Allocation and Latency Optimization

> **Parent:** `plans/deferred/0.0.3/500-page-one-second-performance-target.md` - one-second vision  
> **Evidence ledger:** `plans/performance/2026-08-07/performance-profile-and-improvement-plan.md` (2026-08-08 addendum)  
> **Skill:** `skills/phase-wise-checklist/SKILLS.md`  
> **Status:** measurement complete; implementation not started  
> **Branch:** `feature/optimization`  
> **Estimated effort:** multi-wave (correctness-first); intermediate gates before 1 s stretch  
> **Created:** 2026-08-08  

---

## Overview

Optimize **gowkhtmltopdf** on the **500-page report benchmark**
(`BenchmarkPDFPages/500Pages`, `report.html.tmpl`, 20 rows/section,
`page-break-before:always` per section after the first) for:

1. **Lower wall time** (milliseconds / seconds).
2. **Fewer heap allocations** (`allocs/op`) and less heap traffic (`B/op`).
3. **Lower peak process RSS** under CLI conversion of the same HTML.

This plan **does not replace** `500-page-one-second-performance-target.md`.
That file remains the stretch vision. This file is the **execution checklist**
driven by the 2026-08-08 profile and the wkhtmltopdf process-level comparison.

## Executive Summary

### 2026-08-08 baseline (this machine)

| Metric | gowkhtmltopdf | wkhtmltopdf 0.12.6.1 (same HTML) |
|---|---:|---:|
| 500-page in-process bench | **7.91 s · 4.04 GB B/op · 13.2 M allocs** | n/a (native) |
| 500-page CLI wall + peak RSS | **8.40 s · ~777 MB RSS** | **2.05 s · ~114 MB RSS** |

### Who is faster / leaner (full matrix, process-level)

| Page range | Faster | Lower peak RSS |
|---|---|---|
| **2–100** | **gowkhtmltopdf** | gowk on 2–10; **wk** from ~20 up |
| **200–500** | **wkhtmltopdf** | **wkhtmltopdf** |

Full table is in the performance-profile addendum (2026-08-08).

### Profile root causes (own findings)

1. **`beforeAlways` + `prefixMaxOfOps`:** rebuilds a full op-prefix max array
   after **every** forced break → ~17% flat CPU and ~677 MB alloc_space at 500
   pages. Correct, O(breaks × ops), not scalable.
2. **Style cascade object churn:** `inheritableProps` alone is ~**40%** of
   alloc_objects; `resolveStylesCtx` owns ~half of alloc_space cumulatively.
3. **Pagination shifts:** `shiftOpsBucket` / `shiftIndexedBox` / `shiftIndexedOp`
   remain material after forced-break placement (~13% flat combined).
4. **GC:** `runtime.scanObject` ~20% flat is **follow-on cost** of (1)+(2).
5. **2026-08-07 “final” 5.61 M allocs / 1.91 GB / 6.93 s** is **not** what we
   measure on 2026-08-08 tip; re-baseline before claiming progress.

### Optimization north star (gates for this checklist)

| Gate | Metric | Pass when |
|---|---|---|
| G1 Intermediate latency | `BenchmarkPDFPages/500Pages` median `-count=3 -benchtime=1x` | ≤ **4.0 s** |
| G2 Intermediate allocs | same | ≤ **7.0 M allocs/op** and ≤ **2.0 GB B/op** |
| G3 CLI RSS | `/usr/bin/time` peak RSS on `report-500.html` | ≤ **400 MB** |
| G4 Stretch (parent doc) | median | ≤ **1.0 s** (only after G1–G3 + correctness) |
| G5 Correctness | always | `make test`, page count 500, golden fixtures, no silent layout weaken |

---

## Phase 0: Measurement lock-in

### 0.1 Reproducible 500-page commands

- [x] Record in-process profile command and raw result (7.91 s / 4.04 GB / 13.2 M).
- [x] Record process-level matrix vs wkhtmltopdf (2…500) and “who wins” table.
- [x] Append evidence to `plans/performance/2026-08-07/performance-profile-and-improvement-plan.md`.
- [ ] Re-run PDF 500 with `-count=3` and record median/spread before first code wave.
- [ ] Capture one `memprofile` + `cpuprofile` pair after each closed phase.

**Proof:** commands and tables in the performance-profile addendum.

---

## Phase 1: Forced-break / prefix cost (highest leverage)

### 1.1 Incremental or single-pass prefix

- [ ] Replace full `prefixMaxOfOps` rebuild-after-every-break in
      `internal/layout/paint.go` (`beforeAlways` / `shiftForcedBreak`) with either:
  - incremental max maintenance when only a suffix moves, or
  - one-pass forced-break placement that does not rescan all ops per break.
- [ ] Keep mutation-safe semantics for interleaved content (no empty-page drift).
- [ ] Prove: `TestPageBreakBeforeStacked`, golden fixture-08, benchmark
      page counts 2…500 exact; no hang in `shiftFlowY`.

**Expected:** large drop in `prefixMaxOfOps` flat CPU and ~677 MB alloc_space.

### 1.2 Avoid O(breaks) full-suffix shifts when possible

- [ ] For dense `page-break-before:always` sequences, evaluate placing each
      section onto `pageIndex * contentH` arithmetic vs repeated `shiftFlowY`.
- [ ] Gate behind correctness tests for avoid/inside/after policies still in use.

**Expected:** lower `shiftOpsBucket` / `shiftIndexed*` time on the 500-page report.

---

## Phase 2: Style / cascade allocation collapse

### 2.1 `inheritableProps` object factory

- [ ] Profile-driven change in `internal/layout/style.go` (and helpers) so
      inheritance does not allocate ~5.6 M objects per 500-page convert.
- [ ] Prefer shared immutable tables, pooling, or bitsets over per-node maps.
- [ ] Prove: style regression tests, wiki/print fixtures, `make test`.

### 2.2 Named colors / rule hit lists

- [ ] Audit `css.namedColors` and `ruleSelectorHits` / `sheetRuleHits` for
      per-node rebuilds; cache at stylesheet or context scope.
- [ ] Prove: CSS unit tests + full suite; heap profile shows reduced
      alloc_objects flat share.

### 2.3 `resolveStylesCtx` walk

- [ ] Reduce per-element temporary maps/slices in cascade (`cascadeRaw`,
      `applyRestProps`) without changing cascade winners.
- [ ] Prove: golden corpus page envelopes and HF/link tests still pass.

**Expected:** cut total `allocs/op` toward G2; GC flat share falls as a side effect.

---

## Phase 3: Layout emission and table display-list

### 3.1 Cell / border op emission

- [ ] Measure residual cost of `buildCell`, `borderLineOps`,
      `collectTableBorderSegments` after Phase 1–2.
- [ ] Only then pool op slices or share chrome geometry for repeated rows.

### 3.2 Font face selection

- [ ] Cache `faceForRune` results per face set + rune where safe.
- [ ] Prove: CJK/IPA/fallback tests unchanged.

---

## Phase 4: Parallelism and 1 s stretch (only after G1–G3)

### 4.1 Parallel paint / compress

- [ ] Prototype parallel page content generation **after** pagination settles
      (parent deferred plan item 1).
- [ ] Measure wall time and RSS; reject if RSS exceeds G3 without clear win.

### 4.2 Repeated-section fast path

- [ ] Optional specialized path when input is N identical forced sections
      (parent deferred plan item 4); must not change general HTML semantics.

### 4.3 One-second gate

- [ ] Meet parent doc acceptance only with three-run median ≤ 1.00 s and
      correctness intact; document if unreachable without contract changes.

---

## Phase 5: Closure

- [ ] Update performance-profile addendum with post-change profiles (do not
      erase 2026-08-08 baseline rows; append).
- [ ] Refresh `testdata/golden/benchmarks/benchmark-results.txt` PDF/Template
      500 (and full matrix if practical) with exact commands.
- [ ] Re-run process-level compare vs wkhtmltopdf on 500 pages; record new
      “who wins” row.
- [ ] `make lint` and `make test` both green (required for any implementation
      phase closure per phase-wise-checklist skill).

---

## Dependencies

| Depends on | Why |
|---|---|
| Pagination correctness tests | Phase 1 can re-break page counts / empty pages |
| Cascade tests / golden corpus | Phase 2 can change computed style |
| Parent one-second plan | Stretch target and architecture options |
| 2026-08-08 profile addendum | Evidence and priority order |

## Risks

- Arithmetic page placement for forced breaks may disagree with avoid/orphans
  policies if applied too broadly.
- Style pooling bugs are silent (wrong cascade winner) — require strong tests.
- Parallel PDF write may increase peak RSS while cutting wall time.
- Comparing only to wkhtmltopdf fidelity is unfair; always keep correctness
  gates on gowkhtmltopdf’s own contract.

## Non-goals (this plan)

- Pixel-identical WebKit output.
- Changing the published image-tile matrix in the same waves (optional later).
- Editing or closing rows in the 2026-08-07 phase checklist as if they still
  describe tip HEAD.
