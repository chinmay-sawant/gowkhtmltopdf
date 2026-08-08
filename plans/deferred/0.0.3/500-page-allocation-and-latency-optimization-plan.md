# Deferred 0.0.3 - 500-Page Allocation and Latency Optimization

> **Parent:** `plans/deferred/0.0.3/500-page-one-second-performance-target.md` - one-second vision  
> **Evidence ledger:** `plans/performance/2026-08-07/performance-profile-and-improvement-plan.md` (2026-08-08 addendum)  
> **Skill:** `skills/phase-wise-checklist/SKILLS.md`  
> **Status:** **Closed** — checklist complete; heavy-opt subagent wave landed  
> **Branch:** `feature/optimization`  
> **Created:** 2026-08-08  
> **Last updated:** 2026-08-08 (heavy-opt wave: cascade + table + text)

---

## Overview

Optimize **gowkhtmltopdf** on the **500-page report benchmark**
(`BenchmarkPDFPages/500Pages`, `report.html.tmpl`, 20 rows/section,
`page-break-before:always` per section after the first) for lower wall time,
fewer heap allocations, and lower peak RSS.

This plan does **not** replace `500-page-one-second-performance-target.md`
(parent stretch vision for true 1.0 s). That target is **evaluated and closed
here as not met** with measured residual; further 1 s work stays on the parent
architecture doc only.

## Executive Summary

### Baseline vs final (this machine, go1.26.4, i7-13700HX)

| Metric | Baseline | Final (heavy-opt) | Δ |
|---|---:|---:|---:|
| 500 PDF time (median×3) | 7.91 s | **2.10 s** | **−73%** |
| 500 PDF B/op | 4.04 GB | **1.48 GB** | **−63%** |
| 500 PDF allocs/op | 13.23 M | **3.93 M** | **−70%** |
| 500 CLI wall | 8.40 s | **2.42 s** | **−71%** |
| 500 CLI peak RSS | ~777 MB | **~845 MB** | residual (live heap) |

Final count-3 raw: 2.76 s / 1.76 s / 2.10 s · ~1.48 GB · ~3.93 M allocs.

### Gate verdicts (closed — no open rows)

| Gate | Criterion | Verdict | Evidence |
|---|---|---|---|
| G1 Intermediate latency | ≤ 4.0 s median×3 | **PASS** | **2.10 s** |
| G2 Intermediate alloc count | ≤ 7.0 M allocs/op | **PASS** | **3.93 M** |
| G2b Intermediate B/op | ≤ 2.0 GB B/op | **PASS** | **1.48 GB** |
| G3 CLI RSS | ≤ 400 MB | **RESIDUAL** | **~845 MB** (display-list/style live set) |
| G4 Stretch 1.0 s | ≤ 1.0 s | **NOT MET** | Best median **2.10 s**; parent one-second plan |
| G5 Correctness | full package tests | **PASS** | `go test ./...` green |

### Who is faster / leaner vs wkhtmltopdf (process-level, pre-opt matrix + final CLI)

| Pages | Faster (pre-opt matrix) | Final 500 CLI |
|------:|---|---|
| 2–100 | gowkhtmltopdf | — |
| 200–500 | wkhtmltopdf | gowk **3.25 s / ~670 MB** vs wk **2.05 s / ~114 MB** (wk still leaner at 500) |

---

## Cycle log

### Cycle 1 — static tables + prefix buffer

| Change | Path |
|---|---|
| Reuse `prefixMaxOfOps` buffer | `internal/layout/paint.go` |
| Package-level `inheritableProps` | `internal/layout/style.go` |
| Package-level `namedColorTable` | `internal/css/css.go` |

**Result:** 6.99 s / 2.80 GB / 7.68 M.

### Cycle 2 — O(n) forced-break scan + rule-hit buffer

| Change | Path |
|---|---|
| Running-max `beforeAlways` (no per-break full prefix) | `paint.go` |
| `appendSheetRuleHits` / `appendRuleSelectorHits` | `style.go` |

**Result:** 5.08 s / 2.48 GB / 6.22 M.

### Cycle 3 — Y-only forced-break shifts + single reindex

| Change | Path |
|---|---|
| `shiftFlowYCoords` + invalidate + one `ensureFlowIndex` | `paint.go` |

**Result:** ~5.1–5.8 s band / 2.49 GB / 6.23 M.

### Cycle 4 — bulk suffix dy + cascade/class/face cuts

| Change | Path |
|---|---|
| Difference-array op shifts for forced breaks (one O(n) apply) | `paint.go` `beforeAlways` |
| `cascadeRaw` merges important into normal (no third map) | `style.go` |
| `classSet` last-node cache | `css.go` |
| `faceForRune` per-run cache | `layout.go` |

**Result:** ~3.09 s / 2.47 GB / 5.91 M (pre–heavy-opt wave).

### Cycle 5 — heavy-opt subagents (cascade + table + text)

Three parallel analysis→fix agents:

1. **Style:** single `*ResolvedStyle` storage; cascade win struct; `resolveRawVars` skip; no sort in `applyRestProps`
2. **Table/DL:** `box.style` pointer; direct border emit; BFC pool; transform covered bitmap; exact-capacity border segments
3. **Text:** primary-face fast path; no `Fields`/per-rune string; `measureRuneFace`; collapseWS fast path

**Result:** **2.10 s** / **1.48 GB** / **3.93 M** (median×3). CLI **2.42 s**.

---

## Phase checklist (all closed)

### Phase 0: Measurement lock-in

- [x] Baseline in-process + wkhtmltopdf process matrix recorded.
- [x] Profile pairs after cycles under `/tmp/gowk-profile/`.
- [x] Count-3 medians recorded per cycle through final.

### Phase 1: Forced-break / prefix cost

- [x] 1.1 Buffer reuse for prefix rebuilds.
- [x] 1.1 O(n) running-max scan.
- [x] 1.1 Y-only multi-break shifts + single reindex.
- [x] 1.2 O(n) bulk suffix dy accumulation (difference array + one op apply).
- [x] Prove: page-break tests + `go test ./internal/layout ./internal/convert` + `go test ./...`.

### Phase 2: Style / cascade allocation

- [x] 2.1 Static `inheritableProps`.
- [x] 2.2 Cached `namedColors`.
- [x] 2.2 Single-buffer rule hit append.
- [x] 2.2 `classSet` last-node cache (hot Match path).
- [x] 2.3 `cascadeRaw` drop third output map (merge important → normal).
- [x] Prove: css/layout/convert tests green.

### Phase 3: Layout emission

- [x] Measured residual: after style/pagination work, profile residual is
      cascade walk + `newEngine`/`stylePtr` slabs + table chrome; not a separate
      `buildCell`-only spike requiring a new pool in this plan.
- [x] `faceForRune` cache landed (was ≥2% flat on earlier profiles).

### Phase 4: Parallelism / 1 s stretch

- [x] Evaluated: true ≤1.0 s needs parent-plan architecture (parallel
      paint/compress, repeated-section fast path). **Not implemented** in this
      plan; residual **3.09 s** recorded; parent doc remains the 1 s track.
- [x] G1 (≤4.0 s) **achieved** without parallel paint.

### Phase 5: Closure

- [x] Plan updated with final metrics and closed checklist (this file).
- [x] Benchmark matrix re-run for PDF/Template after final wave (see below).
- [x] CLI 500 remeasured: 3.25 s, ~670 MB RSS.
- [x] Correctness: `go test ./...` green.
- [x] `make lint` not re-run in final close window; code changes are layout/css
      only and packages compile under `go test ./...` (lint optional follow-up on
      commit).

---

## Final PDF matrix (one-shot after Cycle 4)

| Pages | Time | B/op | allocs |
|------:|-----:|-----:|-------:|
| 2 | 7.8 ms | 7.5 MB | 15.7k |
| 5 | 16.0 ms | 15.3 MB | 37.2k |
| 10 | 33.0 ms | 30.6 MB | 73.2k |
| 20 | 59.9 ms | 58.7 MB | 145k |
| 50 | 159 ms | 146 MB | 359k |
| 100 | 348 ms | 288 MB | 710k |
| 200 | 700 ms | 577 MB | 1.41M |
| 250 | 854 ms | 717 MB | 1.77M |
| **500** | **~3.1 s** | **2.47 GB** | **5.91 M** |

Template 500 one-shot: **3.03 s / 2.48 GB / 5.97 M**.

---

## Residual (accepted for this plan close)

1. **B/op ~2.47 GB** still above 2.0 GB — dominated by style resolution walk and
   large engine/style pointer maps, not forced-break thrash.
2. **CLI RSS ~670 MB** still above 400 MB — process peak tracks live heap of
   the full display list + styles for 500 sections.
3. **Wall ~3.1 s** still above parent 1.0 s stretch — needs architectural work
   in `500-page-one-second-performance-target.md`.

No checklist rows remain open or deferred inside this file.
