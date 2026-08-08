# Deferred 0.0.3 - 500-Page Allocation and Latency Optimization

> **Parent:** `plans/deferred/0.0.3/500-page-one-second-performance-target.md` - one-second vision  
> **Evidence ledger:** `plans/performance/2026-08-07/performance-profile-and-improvement-plan.md` (2026-08-08 addendum)  
> **Skill:** `skills/phase-wise-checklist/SKILLS.md`  
> **Status:** Cycle 1 implemented; further cycles in progress  
> **Branch:** `feature/optimization`  
> **Estimated effort:** multi-wave (correctness-first); intermediate gates before 1 s stretch  
> **Created:** 2026-08-08  
> **Last updated:** 2026-08-08 (Cycle 1 close)

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

### 2026-08-08 baseline (pre-optimization)

| Metric | gowkhtmltopdf | wkhtmltopdf 0.12.6.1 (same HTML) |
|---|---:|---:|
| 500-page in-process bench | **7.91 s · 4.04 GB B/op · 13.2 M allocs** | n/a (native) |
| 500-page CLI wall + peak RSS | **8.40 s · ~777 MB RSS** | **2.05 s · ~114 MB RSS** |

### Cycle 1 result (post-fix, median of 3 where noted)

| Metric | Baseline | After Cycle 1 | Δ |
|---|---:|---:|---:|
| 500 PDF time | 7.91 s | **6.99 s** (median of 3) | **−11.6%** |
| 500 PDF B/op | 4.04 GB | **2.80 GB** | **−30.7%** |
| 500 PDF allocs/op | 13.23 M | **7.68 M** | **−42.0%** |

Count-3 raw after Cycle 1: 7.40 s / 6.96 s / 6.99 s · ~2.80 GB · ~7.68 M allocs.

### Who is faster / leaner (full matrix, process-level, pre-opt)

| Page range | Faster | Lower peak RSS |
|---|---|---|
| **2–100** | **gowkhtmltopdf** | gowk on 2–10; **wk** from ~20 up |
| **200–500** | **wkhtmltopdf** | **wkhtmltopdf** |

### Profile root causes (baseline findings)

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

| Gate | Metric | Pass when | Cycle 1 status |
|---|---|---|---|
| G1 Intermediate latency | `BenchmarkPDFPages/500Pages` median `-count=3 -benchtime=1x` | ≤ **4.0 s** | [ ] 6.99 s |
| G2 Intermediate allocs | same | ≤ **7.0 M allocs/op** and ≤ **2.0 GB B/op** | [~] 7.68 M / 2.80 GB |
| G3 CLI RSS | `/usr/bin/time` peak RSS on `report-500.html` | ≤ **400 MB** | [ ] not remeasured |
| G4 Stretch (parent doc) | median | ≤ **1.0 s** | [ ] |
| G5 Correctness | always | `make test`, page count 500, golden fixtures | [x] layout/css/convert green after Cycle 1 |

---

## Cycle log

### Cycle 1 — buffer reuse + static tables (2026-08-08)

**Shipped code**

| Change | Path | Effect |
|---|---|---|
| Reuse `prefixMaxOfOps` buffer across `beforeAlways` walk | `internal/layout/paint.go` | Removes ~500 large `[]float64` allocs per 500-page convert |
| Package-level `inheritableProps` table | `internal/layout/style.go` | Removes per-node slice + closure factory (~40% of baseline alloc_objects) |
| Package-level `namedColorTable` | `internal/css/css.go` | Removes per-parse color map (~281 MB baseline alloc_space) |

**Validation**

```sh
go test ./internal/layout ./internal/css ./internal/convert -count=1 -timeout 180s  # ok
go test ./internal/convert -run '^$' -bench '^BenchmarkPDFPages/500Pages$' \
  -benchmem -benchtime=1x -count=3  # median ~6.99s / 2.80GB / 7.68M
```

**Outcome:** large allocation win; modest latency win. G2 nearly met on alloc
count; B/op and wall time still open.

---

## Phase 0: Measurement lock-in

### 0.1 Reproducible 500-page commands

- [x] Record in-process profile command and raw result (7.91 s / 4.04 GB / 13.2 M).
- [x] Record process-level matrix vs wkhtmltopdf (2…500) and “who wins” table.
- [x] Append evidence to `plans/performance/2026-08-07/performance-profile-and-improvement-plan.md`.
- [x] Re-run PDF 500 with `-count=3` after Cycle 1 (median 6.99 s).
- [x] Capture baseline `memprofile` + `cpuprofile` (pre-Cycle 1 under `/tmp/gowk-profile/`).
- [ ] Capture profile pair after each subsequent cycle.

**Proof:** commands and tables in the performance-profile addendum + Cycle log.

---

## Phase 1: Forced-break / prefix cost (highest leverage)

### 1.1 Incremental or single-pass prefix

- [x] Replace full `prefixMaxOfOps` **allocation** after every break with a
      **reused buffer** full recompute (`paint.go`).
- [x] Keep mutation-safe semantics (full recompute into buf; no partial max).
- [x] Prove: layout page-break tests + convert fixture-08 + convert package tests.
- [ ] **Cycle 2+:** stop O(breaks × ops) CPU — incremental max after shift, or
      single-pass forced-break placement without rescanning all ops each break.

**Expected (1.1 buffer reuse):** large drop in prefix alloc_space — **landed**.  
**Expected (1.1 CPU):** still open until incremental/single-pass.

### 1.2 Avoid O(breaks) full-suffix shifts when possible

- [ ] For dense `page-break-before:always` sequences, evaluate placing each
      section onto `pageIndex * contentH` arithmetic vs repeated `shiftFlowY`.
- [ ] Gate behind correctness tests for avoid/inside/after policies still in use.

**Expected:** lower `shiftOpsBucket` / `shiftIndexed*` time on the 500-page report.

---

## Phase 2: Style / cascade allocation collapse

### 2.1 `inheritableProps` object factory

- [x] Package-level static inherit table (no per-call `[]inheritCopy` / closures).
- [x] Prove: `go test ./internal/layout` green.
- [ ] Further: reduce remaining cascade temps if profile still shows style walk
      as top alloc after Cycle 1.

### 2.2 Named colors / rule hit lists

- [x] Cache `namedColors` map at package level (`css.go`).
- [ ] Audit `ruleSelectorHits` / `sheetRuleHits` for per-node rebuilds; cache at
      stylesheet or context scope if still hot post-Cycle 1.
- [x] Prove: `go test ./internal/css` green.

### 2.3 `resolveStylesCtx` walk

- [ ] Reduce per-element temporary maps/slices in cascade (`cascadeRaw`,
      `applyRestProps`) without changing cascade winners.
- [ ] Prove: golden corpus page envelopes and HF/link tests still pass.

**Expected:** cut total `allocs/op` toward G2 — **partially landed** (7.68 M).

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
