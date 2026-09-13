# 0.2.6 benchmark comparison snapshot (temporary)

> **Purpose:** one place to see what the 0.2.6 performance recovery plan changed,
> what it did not change, and where it still stands against the supplied 0.2.4
> rows. Temporary working note, not a published reference. The committed
> narrative lives in `documentation/performance.md` (2026-09-11 capture) and
> `testdata/golden/benchmarks/benchmark-results.txt` Snapshot K.
>
> **Date:** 2026-09-11
> **Measurement:** three independent samples, medians, `-benchtime=1x -count=1`
> for in-process rows; `%M` peak RSS for CLI rows. `B/op` is cumulative
> allocation traffic, never process RSS.

## How to read the columns

- **v0.2.4 target:** the supplied recovery rows from
  `perf-review/phase-wise-checklist.md` (overview table). Different host and
  build; context only.
- **Pre-change (m):** same host, same method, measured before the changes
  (`perf-01-03.md`, `perf-04-05.md`, `img-02-04.md`,
  `impl/perf-recovery-cli-rss-500-before.txt`).
- **Pre-change (s):** the supplied current observation printed in the plan.
  Different context, shown because some rows have no same-host baseline.
- **Now:** VALID-02 medians (`valid-02-04.md`, `valid/sample1..3/`) and the
  VALID-03 CLI matrix (`testdata/golden/benchmarks/cli-compare*`).

## PDF

| Metric | v0.2.4 target | Pre-change | Now (median) | Verdict |
|---|---:|---:|---:|---|
| Internal generic 500p B/op | 237.76 MB | 557.5 MB (m) / 525.42 MB (s) | **321.1 MB** | -42% vs (m), target not met |
| Internal generic 500p time | 1,010 ms | 1,432-1,810 ms (m) / 1,545 ms (s) | **1,246 ms** | improved, target not met |
| Internal generic 2p B/op | 2.21 MB | 10.47 MB (m) / 3.32 MB (s) | **9.58 MB** | -8.5% vs (m), far above target |
| Internal generic 2p time | 3.58 ms | 10.1-10.8 ms (m) / 5.04 ms (s) | **10.06 ms** | flat vs (m), above target |
| Public library 500p B/op | 236.85 MB | 523.24 MB (s) | **322.9 MB** | -38%, target not met |
| Public library 500p time | 1,104.51 ms | 1,265.53 ms (s) | **1,268.6 ms** | unchanged, target not met |
| CLI 500p time | 1,042 ms | 1,302 ms (s) / 1.69-1.75 s (m) | **1.288 s** | -1% vs (s), target not met |
| CLI 500p RSS | 203.3 MiB | 500.8 MiB (s) / 462-503 MiB (m) | **235.3 MiB** | -53%, target not met |
| CLI 100p RSS | 59.8 MiB | 118.5 MiB (s) | **67.5 MiB** | -43%, near target |

## Image

| Metric | v0.2.4 target | Pre-change | Now (median) | Verdict |
|---|---:|---:|---:|---|
| Public image 250 tiles B/op | 20.66 MB | 46.44 MB (s) / 56.05 MB (m) | **21.50 MB** | -62% vs (m), 4.1% above target |
| Public image 500 tiles B/op | 52.00 MB | 90.05 MB (s) / 94.97 MB (m) | **27.21 MB** | -71% vs (m), beats target by 48% |
| Image output bytes 250 / 500 | n/a | 112,296 / 226,880 (m) | **94,352 / 188,268** | -16% / -17% smaller PNG |

## Current CLI matrix (VALID-03, 2026-09-11)

One warmup plus three timed runs per size, exact page count enforced,
`%M` peak RSS.

| Pages | gowk time | wkhtmltopdf time | Speedup | gowk RSS | wkhtmltopdf RSS | PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|
| 2 | 17 ms | 258 ms | 14.95x | 24,576 KiB | 44,716 KiB | 34,209 |
| 5 | 23 ms | 266 ms | 11.44x | 26,496 KiB | 45,068 KiB | 42,791 |
| 10 | 34 ms | 286 ms | 8.29x | 28,608 KiB | 46,024 KiB | 57,231 |
| 20 | 53 ms | 306 ms | 5.79x | 33,984 KiB | 47,772 KiB | 84,654 |
| 50 | 122 ms | 394 ms | 3.23x | 47,616 KiB | 52,240 KiB | 167,442 |
| 100 | 229 ms | 541 ms | 2.36x | 69,120 KiB | 59,380 KiB | 306,144 |
| 200 | 468 ms | 830 ms | 1.77x | 113,280 KiB | 74,492 KiB | 583,231 |
| 250 | 599 ms | 988 ms | 1.65x | 139,584 KiB | 81,884 KiB | 721,739 |
| 500 | 1.288 s | 1.753 s | 1.36x | 240,960 KiB | 123,172 KiB | 1,419,234 |

## External engines (VALID-04, elapsed medians)

Puppeteer RSS is a sampled process tree (node plus Chrome children); gowk and
WeasyPrint RSS are `%M` peak process RSS, so the RSS columns are not directly
comparable. Speedups use elapsed medians only.

| Pages | gowk | WeasyPrint | Speedup |
|---:|---:|---:|---:|
| 2 | 21 ms | 653 ms | 31.13x |
| 10 | 36 ms | 1.431 s | 39.80x |
| 50 | 124 ms | 5.482 s | 44.07x |
| 100 | 245 ms | 11.119 s | 45.29x |

| Pages | gowk | Puppeteer | Speedup |
|---:|---:|---:|---:|
| 2 | 21 ms | 1.470 s | 68.85x |
| 10 | 37 ms | 1.550 s | 42.45x |
| 50 | 120 ms | 1.815 s | 15.08x |
| 100 | 249 ms | 2.179 s | 8.74x |

## 2-page cold vs warm (method correction)

The 0.2.4 target row for 2 pages (3.58 ms / 2.21 MB) is a warm-state row. The
recovery 2-page row (10.06 ms / 9.58 MB) is a fresh-process `1x` sample, so the
one-time default-font load (about 6.8 MB of allocations and roughly 7 to 8 ms)
is charged to the single operation. The two are not like for like. The committed
`documentation/performance.md` carries the same caveat.

Same-process probe with `-benchtime=1x -count=3`: the first row is cold, the
later rows are warm and share the loaded fonts.

| Probe | Cold (row 1) | Warm (rows 2-3) |
|---|---:|---:|
| Internal generic 2p before (`2fe08fa`) | 12.17 ms / 10.47 MB | 5.00 ms / 2.66 MB; 5.91 ms / 3.48 MB |
| Internal generic 2p after (`57751d1`) | 12.34 ms / 9.58 MB | 5.16 ms / 2.59 MB; 4.40 ms / 1.77 MB |
| Public library 2p before, 10x median | n/a | 5.24 ms / 2.96 MB |
| Public library 2p after, 10x median | n/a | 5.09 ms / 1.90 MB |

Warm reading: allocation improved about 36 to 40 percent (2.96 MB -> 1.90 MB
public, 3.48 MB -> 1.77 MB internal best warm sample), matching or beating the
historical 2.21 MB row. Warm time is flat at about 4.4 to 5.2 ms and still
about 1 to 1.7 ms above the 0.2.4 3.58 ms row. The 500-page rows are unaffected
by the cold charge (about 0.6 percent of 1,246 ms).

## Historical warm vs current warm (full matrix)

Both columns are one-iteration (`-benchtime=1x -count=1`) full-matrix runs on
the same host (WSL2, i7-13700HX, 24 CPUs): historical is Snapshot I,
2026-08-19, from `testdata/golden/benchmarks/benchmark-results.txt`; current is
three invocations on 2026-09-11, medians. The 2-page historical row did not
carry today's first-run font charge; current 2-page warm comes from the
same-process `-count=3` probe (the current 1x row is 12.31 ms / 9.58 MB cold).

| Pages | Hist. time | Current warm time | Hist. B/op | Current B/op | Bytes vs hist. |
|---:|---:|---:|---:|---:|---:|
| 2 warm | 3.58 ms | 4.40-5.16 ms | 2,209,400 | 1,765,328-2,588,872 | equal or better |
| 5 | 7.29 ms | 15.06 ms | 3,611,912 | 4,388,808 | +21.5% |
| 10 | 14.86 ms | 24.30 ms | 6,078,448 | 7,457,064 | +22.7% |
| 20 | 28.64 ms | 58.91 ms | 10,731,688 | 13,587,136 | +26.6% |
| 50 | 84.17 ms | 141.10 ms | 24,977,520 | 32,371,520 | +29.6% |
| 100 | 157.69 ms | 272.86 ms | 48,728,040 | 64,506,208 | +32.4% |
| 200 | 384.45 ms | 621.68 ms | 96,316,552 | 129,079,864 | +34.0% |
| 250 | 449.51 ms | 841.12 ms | 119,357,760 | 160,339,632 | +34.3% |
| 500 | 1,009.80 ms | 1,246.05 ms standalone; 1,633.08 ms in-matrix | 237,755,328 | 321,104,720 | +35.0% |

Reading: allocation is still 20 to 35 percent above the 2026-08-19 matrix at
5 to 500 pages, while it is equal or better at 2 pages warm. The interning
change moved the tree from about +80 to +90 percent above historical bytes
(parent commit) to this +20 to +35 percent band. Time medians on this shared
desktop drift 20 to 30 percent between capture windows; `B/op` is the stable
metric. The 500-page row is context sensitive: 1.246 s when the process runs
only 2 and 500 pages, 1.633 s after the full 2 to 250 page matrix.

## Perf-improve final (phase 7, 2026-09-11, published as Snapshot L)

The perf-improve plan executed after this file's recovery capture. Columns:
Snapshot I is the committed 2026-08-19 historical matrix; Recovery is this
file's earlier capture; Final is the phase-7 capture published as Snapshot L.

### Warm internal matrix, B/op

| Pages | Snapshot I | Recovery | Final | Recovery -> Final | Final vs Snapshot I |
|---:|---:|---:|---:|---:|---:|
| 5 | 3.61 MB | 4.39 MB | **2.89 MB** | -34.1% | -19.9% |
| 10 | 6.08 MB | 7.46 MB | **5.23 MB** | -29.9% | -14.0% |
| 20 | 10.73 MB | 13.59 MB | **10.75 MB** | -20.9% | +0.2% |
| 50 | 24.98 MB | 32.37 MB | **24.82 MB** | -23.3% | -0.6% |
| 100 | 48.73 MB | 64.51 MB | **48.37 MB** | -25.0% | -0.7% |
| 200 | 96.32 MB | 129.08 MB | **95.42 MB** | -26.1% | -0.9% |
| 250 | 119.36 MB | 160.34 MB | **118.43 MB** | -26.1% | -0.8% |
| 500 | 237.76 MB | 321.10 MB | **234.92 MB** | -26.8% | -1.2% |

### Warm internal matrix, time

| Pages | Snapshot I | Recovery | Final | Recovery -> Final | Final vs Snapshot I |
|---:|---:|---:|---:|---:|---:|
| 5 | 7.29 ms | 15.06 ms | **10.98 ms** | -27.1% | +50.6% |
| 10 | 14.86 ms | 24.30 ms | **20.05 ms** | -17.5% | +34.9% |
| 20 | 28.64 ms | 58.91 ms | **42.82 ms** | -27.3% | +49.5% |
| 50 | 84.17 ms | 141.10 ms | **102.59 ms** | -27.3% | +21.9% |
| 100 | 157.69 ms | 272.86 ms | **225.10 ms** | -17.5% | +42.7% |
| 200 | 384.45 ms | 621.68 ms | **487.72 ms** | -21.5% | +26.9% |
| 250 | 449.51 ms | 841.12 ms | **606.98 ms** | -27.8% | +35.0% |
| 500 | 1,009.80 ms | 1,633.08 ms matrix / 1,246.05 ms standalone | **1,228.72 ms** | -24.8% matrix / -1.4% standalone | +21.7% |

Method note: the Recovery time column came from an ad-hoc matrix captured while
profiling agents loaded the host. The plan's dedicated phase-1 warm capture
(same mode as Final) recorded 500p at 1,489.42 ms, so the attributable warm
500p time gain is 17.5 percent. B/op is the stable comparison across windows.

### Cold, library, image, and CLI rows

| Metric | Recovery | Final | Delta | 0.2.4 / target |
|---|---:|---:|---:|---:|
| Internal 2p fresh B/op | 9.58 MB | **2.67 MB** | -72.2% | 2.21 MB |
| Internal 2p fresh time | 10.06 ms (1x) | 6.69 ms warm-matrix row 1, 6.11 ms standalone | about -40% | 3.58 ms |
| Public library 2p B/op / time | 9.60 MB / 10.88 ms | **2.68 MB / 7.14 ms** | -72.1% / -34.4% | 2.21 MB / 3.58 ms |
| Public library 500p B/op / time | 322.89 MB / 1,268.56 ms | **236.91 MB / 1,240.82 ms** | -26.6% / -2.2% | 236.85 MB / 1,104.51 ms |
| Public image 250 tiles B/op | 21.50 MB | **14.45 MB** | -32.8% | 20.66 MB, now -30.1% below |
| Public image 500 tiles B/op | 27.21 MB | **26.73 MB** | -1.8% | 52.00 MB, now -48.6% below |
| Image encoded bytes 250 / 500 | 94,352 / 188,268 | 94,352 / 188,268 | unchanged | n/a |
| CLI empty-document floor | 18.54 ms | **9.56 ms** | -48.4% | target `< 14 ms` met |
| CLI 2p time / RSS | 17 ms / 24,576 KiB | about 10 ms / **17,472 KiB** | -41% / -28.9% | n/a |
| CLI 100p time / RSS | 229 ms / 69,120 KiB | 240 ms / **54,912 KiB** | +4.8% / -20.6% | 59.8 MiB target, now below |
| CLI 500p time / RSS | 1.288 s / 240,960 KiB | 1.32 s / **203,136 KiB** | +2.5% / -15.7% | 1.042 s / 208,128 KiB, RSS parity |
| 500p PDF bytes | 1,419,234 | 1,419,234 | unchanged | n/a |

### Structural changes

| Item | Recovery | Final |
|---|---:|---:|
| `Op` size | 472 B | **440 B** |
| Seal family allocation at 500p | 71.1 MB | **14.95 MB** |
| `paginateOps` CPU share | 12.75% | **8.04%** |
| `capTablePageBreaks` CPU share | 5.02% | **2.00%** |
| Forced-break box visits at 500p | 27,196,997 | **0** (batched, placements bit-identical) |
| Style storage per conversion | 221,208 B | **221,208 B** (pinned) |
| 500p output bytes | 1,419,234 | **1,419,234** (unchanged) |

### Final verdict

- Warm 500p B/op 234.92 MB meets the 240 MB acceptance and edges the 0.2.4 /
  Snapshot I row of 237.76 MB.
- Warm 500p time 1,228.72 ms improves 17.5 percent against the phase-1 warm
  capture but stays 21.7 percent above Snapshot I; the 1.10 s acceptance is
  not met.
- Style storage (221,208 B) and 500p output bytes (1,419,234) are unchanged,
  and all 65 golden fixtures plus PDF/A-4 and PDF/UA-2 still pass.

## Perf-time final (phase 7, 2026-09-11, published as Snapshot M)

The perf-time plan ran after the perf-improve capture. It halves the warm
times: warm 500p 1,228.72 -> 576.33 ms in the phase-6 same-source capture
(2.13x), B/op 234.92 -> 163.02 MB (-30.6%), and the image rows are 2.10x /
2.27x faster. Snapshot M records the phase-7 closure window, which ran hot;
both numbers are shown.

### Warm internal matrix

Baseline is the perf-improve final capture. Snapshot M is the closure window
(median of three independent processes). The phase-6 anchor is the same-source
capture that crossed the 2x target.

| Pages | Baseline time | Snapshot M time | Diff | Baseline B/op | Snapshot M B/op | Diff |
|---:|---:|---:|---:|---:|---:|---:|
| 2 cold | 6.69 ms | 5.50 ms | -17.8% | 2.67 MB | 4.03 MB | +51% |
| 5 | 10.98 ms | 11.15 ms | +1.5% | 2.89 MB | 7.94 MB | +174% |
| 10 | 20.05 ms | 11.88 ms | -40.7% | 5.23 MB | 5.43 MB | +3.8% |
| 20 | 42.82 ms | 22.36 ms | -47.8% | 10.75 MB | 7.86 MB | -26.9% |
| 50 | 102.59 ms | 56.36 ms | -45.1% | 24.82 MB | 17.68 MB | -28.8% |
| 100 | 225.10 ms | 123.90 ms | -45.0% | 48.37 MB | 33.97 MB | -29.8% |
| 200 | 487.72 ms | 243.13 ms | -50.1% | 95.42 MB | 66.51 MB | -30.3% |
| 250 | 606.98 ms | 320.62 ms | -47.2% | 118.43 MB | 82.25 MB | -30.6% |
| 500 | 1,228.72 ms | 733.48 ms | -40.3% | 234.92 MB | 163.02 MB | -30.6% |
| 500 anchor, phase 6 | 1,228.72 ms | **576.33 ms** | **-53.1%** | 234.92 MB | **163.03 MB** | -30.6% |

Small sizes carry new one-time per-process structures (the parallel flate pool
start and the new caches), so 2 to 5 pages read higher than the baseline while
every row from 20 pages up is 27 to 31 percent below it; the 500-page
acceptance is met.

### Image

| Row | Baseline | Phase 5b | Closure window | Verdict |
|---|---:|---:|---:|---|
| 250 tiles time | 50.72 ms | **24.15 ms** | 25.72 ms | 2.10x, 0.72 ms over in the hot window |
| 500 tiles time | 98.47 ms | **43.43 ms** | 44.27 ms | 2.27x, met |
| B/op 250 / 500 | 14.45 / 26.73 MB | 14.29 / 26.41 MB | same | slightly down |
| PNG bytes 250 / 500 | 94,352 / 188,268 | 141,917 / 282,749 | same | +50.4% / +50.2% intended trade |

Decoded pixels are bit-identical, dimensions 1024x2056 / 1024x4040 and full
opacity are unchanged, and IMG-01 plus IMG-03 stay green. The PNG size growth
buys the encode speed with filter-none at deflate level 2.

### CLI and public library

| Row | Baseline | Now | Diff |
|---|---:|---:|---:|
| CLI 500p | 1.32 s / 203,136 KiB | 0.70 s / 147,264 KiB | time -47%, RSS -27.5% |
| Standalone internal 500p | 1,297.98 ms / 235.50 MB | 695.42 ms / 167.87 MB | time -46% in window; phase-6 anchor 576 ms |
| Public PDF 500p | 1,240.82 ms / 236.91 MB | 698.79 ms / 169.65 MB | time -44% in window; B/op -28.4% |

### Structural changes

| Item | Before | After |
|---|---:|---:|
| Style resolution | 444 ms | **30 ms** (memoized repeated structures) |
| Display-list ops at 500p | 174,000 | **66,500** (OpGridRun batching) |
| Paint-range checks | 87,000,000 span checks | **597,168 binary steps** |
| Finalize plus compression | 100.5 ms | **22.1 ms** (retained parallel flate) |
| Op size | 440 B | 432 B |
| `validatePaintPageIndices` calls | 3 | 1 |
| `afterBreaks` walk | 1 call | 0 (style census) |
| Parallel layout prototype | none | identity-safe, 5.4% paired, rejected at a 15% floor |

### Gates

`make test`, `make golden` 65/65 fresh, `make claim-scan`, `make lint`,
`make test-race`, compliance (PDF/A-4 109 rules / 14,386 checks PASS, PDF/UA-2
1,727 rules / 33,392 checks PASS, structure tree PASS), the public validators,
and the image quality suite (138 PASS) all exit 0.

### Honest caveats

- The Snapshot M window swung 626 to 1,036 ms on byte-identical source; the
  2.13x anchor is phase 6's 576.33 ms capture.
- CLI and public PDF time targets (0.66 s and 620 ms) were not reproduced in
  the hot window; their B/op and RSS targets hold.
- Two phase split metrics were missed while total times beat their targets,
  the concurrency design was rejected on measurement, and the encoded PNG is
  50 percent larger by design.
- Raw captures live under `plans/0.2.6/perf-time/results/` (local by policy).

## What improved

- Style interning removed the largest allocator (228 MB of 557 MB):
  500-page generic B/op down 42%.
- The image direct raster branch removed the 2x canvas plus the downscale
  canvas: 250 tiles -62%, 500 tiles -71%.
- CLI peak RSS halved at 500 pages and dropped 43% at 100 pages.
- Page counts, ordered text, fonts, links, PDF/A-4, and PDF/UA-2 all still pass
  (`make golden` 65/65, compliance PASS).

## What got worse or changed

- Nothing measured with the same method regressed. The 2-page row looks worse
  only against the supplied snapshot; the same-method pre-change baseline was
  10.47 MB / 10.1-10.8 ms and now is 9.58 MB / 10.06 ms.
- Against v0.2.4, the PDF family is still behind: 500p B/op +35% internal,
  time +23%; public library PDF +36% B/op and flat time; CLI 500 time +24%,
  RSS +16%; 100p RSS +13% above target.
- Large images now paint at final resolution, so text antialiasing coverage and
  1 px border placement differ from the supersampled path (flat fills are
  byte-identical). The PNG output is 16-17% smaller. This was the documented
  IMG-03 decision.
- Remaining PDF allocation is identified, not fixed: one-shot display-list
  preallocation (95.6 MB at `internal/layout/layout.go:1003`) plus pagination
  scratch (113.7 MB). No checklist row owned those.

## Public library PDF spot check (2/10/50 pages)

Same host, same method: parent commit `2fe08fa` in a throwaway worktree versus
the committed recovery tree `57751d1`. Three independent samples each, medians.
Rows ran in one process in the order 2, 10, 50, 250, so the 2-page row carries
the one-time font load and the later rows do not.

| Row | Metric | Before (`2fe08fa`) | After (`57751d1`) | Delta |
|---|---:|---:|---:|---:|
| 2 pages | B/op | 10,485,016 | 9,596,184 | -8.5% |
| 2 pages | time | 13.21 ms | 13.23 ms | flat |
| 2 pages | allocs/op | 7,009 | 7,024 | flat |
| 10 pages | B/op | 11,952,864 | 7,500,944 | -37.2% |
| 10 pages | time | 27.44 ms | 25.98 ms | -5.3% |
| 10 pages | allocs/op | 26,767 | 26,762 | flat |
| 50 pages | B/op | 55,486,744 | 32,548,872 | -41.3% |
| 50 pages | time | 152.60 ms | 124.56 ms | -18.4% |
| 50 pages | allocs/op | 129,024 | 128,919 | flat |
| 250 pages | B/op | 275,847,264 | 161,189,216 | -41.6% |
| 250 pages | time | 824.5 ms | 878.2 ms | flat within sample spread |

Allocation improved at every size. Time improved clearly at 50 pages and moved
inside the noise band at 2, 10, and 250 pages. The 2-page byte gain is small
because the one-time font load dominates that row.

## Fresh head-to-head vs tag v0.2.4 (capture 2026-09-11T19:33Z to 19:38Z UTC)

The earlier 0.2.4 columns in this file are supplied or historical rows from a
different window. This capture rebuilds v0.2.4 itself: tag
`v0.2.4` = `80cbb47` extracted with `git archive` to
`/tmp/opencode/gowk-v0.2.4`, then the canonical harness runs on both trees
with rounds alternating so host drift hits both sides.

- Command:
  `scripts/bench-performance-recovery.sh --mode=<mode> --sizes=2,5,10,20,50,100,200,250,500 --benchtime=1x --count=1`
- Three independent process rounds per mode per tree; every cell is the median
- CLI rows pool the nine timed runs per size (warmup excluded)
- Same fixture hash in both trees (`report.html.tmpl`, sha256 `e3b5387b...`)
- Current tree: HEAD `ca761bb` plus the uncommitted work, which at this
  capture includes the in-progress wave 2
  (`plans/0.2.6/perf-improve/wave-2-50pct/`), so its rows sit below Snapshot M
  (733 ms / 163.02 MB at 500 pages) on the same tree
- Raw captures and the full comparison, including the 2/5/10 page cold and
  warm probes and the standalone 500 page rows:
  `plans/0.2.6/perf-time/results/vs-0.2.4/` (local, gitignored)

### Internal generic PDF, warm matrix

| Pages | v0.2.4 | current | Δ time | v0.2.4 B/op | current B/op | Δ B/op |
|---:|---:|---:|---:|---:|---:|---:|
| 2 | 9.80 ms | 5.95 ms | -39.3% | 9.20 MB | 4.11 MB | -55.3% |
| 5 | 8.19 ms | 7.94 ms | -3.1% | 3.61 MB | 6.89 MB | +90.8% |
| 10 | 14.90 ms | 12.77 ms | -14.3% | 5.26 MB | 4.72 MB | -10.1% |
| 20 | 30.54 ms | 23.16 ms | -24.2% | 10.73 MB | 6.20 MB | -42.2% |
| 50 | 82.64 ms | 57.22 ms | -30.8% | 24.97 MB | 12.07 MB | -51.7% |
| 100 | 166.2 ms | 113.1 ms | -32.0% | 48.73 MB | 24.12 MB | -50.5% |
| 200 | 393.7 ms | 248.8 ms | -36.8% | 96.47 MB | 46.62 MB | -51.7% |
| 250 | 493.9 ms | 292.3 ms | -40.8% | 119.35 MB | 58.82 MB | -50.7% |
| 500 | 1,142 ms | 560.8 ms | -50.9% | 237.85 MB | 116.06 MB | -51.2% |

### Public library PDF

| Pages | v0.2.4 | current | Δ time | v0.2.4 B/op | current B/op | Δ B/op |
|---:|---:|---:|---:|---:|---:|---:|
| 2 | 10.31 ms | 6.10 ms | -40.8% | 9.21 MB | 4.13 MB | -55.2% |
| 5 | 8.08 ms | 8.90 ms | +10.2% | 2.81 MB | 6.92 MB | +146.2% |
| 10 | 17.03 ms | 12.70 ms | -25.5% | 6.12 MB | 4.76 MB | -22.2% |
| 20 | 34.31 ms | 21.93 ms | -36.1% | 10.81 MB | 6.28 MB | -41.9% |
| 50 | 83.35 ms | 57.08 ms | -31.5% | 25.14 MB | 13.02 MB | -48.2% |
| 100 | 183.2 ms | 109.7 ms | -40.1% | 49.12 MB | 24.41 MB | -50.3% |
| 200 | 377.3 ms | 219.6 ms | -41.8% | 97.03 MB | 47.20 MB | -51.4% |
| 250 | 486.0 ms | 278.9 ms | -42.6% | 120.31 MB | 59.64 MB | -50.4% |
| 500 | 1,055 ms | 563.5 ms | -46.6% | 239.39 MB | 117.71 MB | -50.8% |

### Public library image

| Tiles | v0.2.4 | current | Δ time | v0.2.4 B/op | current B/op | Δ B/op |
|---:|---:|---:|---:|---:|---:|---:|
| 2 | 20.86 ms | 14.60 ms | -30.0% | 18.73 MB | 11.91 MB | -36.4% |
| 5 | 14.99 ms | 11.01 ms | -26.6% | 12.01 MB | 3.45 MB | -71.3% |
| 10 | 15.11 ms | 13.69 ms | -9.4% | 12.69 MB | 3.68 MB | -71.0% |
| 20 | 14.97 ms | 14.06 ms | -6.1% | 12.82 MB | 3.82 MB | -70.2% |
| 50 | 23.39 ms | 19.82 ms | -15.2% | 13.22 MB | 4.20 MB | -68.2% |
| 100 | 38.13 ms | 36.98 ms | -3.0% | 20.70 MB | 20.00 MB | -3.4% |
| 200 | 66.37 ms | 62.29 ms | -6.2% | 37.87 MB | 37.09 MB | -2.1% |
| 250 | 83.42 ms | 18.01 ms | -78.4% | 47.72 MB | 6.38 MB | -86.6% |
| 500 | 167.9 ms | 35.97 ms | -78.6% | 91.92 MB | 10.08 MB | -89.0% |

### CLI process (`cli-rss`)

| Pages | v0.2.4 time | current time | Δ time | v0.2.4 RSS | current RSS | Δ RSS |
|---:|---:|---:|---:|---:|---:|---:|
| 2 | 10 ms | 10 ms | 0.0% | 23.4 MiB | 19.1 MiB | -18.4% |
| 5 | 20 ms | 10 ms | -50.0% | 24.4 MiB | 21.6 MiB | -11.5% |
| 10 | 30 ms | 20 ms | -33.3% | 27.0 MiB | 24.4 MiB | -9.7% |
| 20 | 40 ms | 30 ms | -25.0% | 29.6 MiB | 25.7 MiB | -13.3% |
| 50 | 100 ms | 70 ms | -30.0% | 42.0 MiB | 29.4 MiB | -29.9% |
| 100 | 200 ms | 130 ms | -35.0% | 60.8 MiB | 35.1 MiB | -42.3% |
| 200 | 440 ms | 240 ms | -45.5% | 96.0 MiB | 44.8 MiB | -53.3% |
| 250 | 540 ms | 300 ms | -44.4% | 115.1 MiB | 50.1 MiB | -56.5% |
| 500 | 1,280 ms | 610 ms | -52.3% | 204.8 MiB | 78.4 MiB | -61.7% |

### Reading

- From 10 pages up the current tree leads both PDF paths on time and
  `B/op`. At 500 pages: internal -50.9% time / -51.2% B/op, public library
  -46.6% / -50.8%.
- Cold starts are much cheaper: 2 pages internal -39.3% time / -55.3% B/op,
  public -40.8% / -55.2%.
- Warm steady state at 2 pages is the one remaining gap: internal
  3.70 ms / 2.21 MB (v0.2.4) against 5.53 ms / 3.76 MB (current); public
  4.48 ms / 2.22 MB against 5.07 ms / 3.77 MB.
- The 5-page `B/op` row of the warm matrix is a process-context artifact: the
  one-time per-process structures are charged there. A fresh-process warm 5p
  conversion allocates 4.01 MB against v0.2.4's 3.61 MB (+11%), and 5p time
  is still faster in the current tree (6.61 ms against 8.44 ms).
- Image tiles are the largest win: 250 and 500 tiles are 4.6x and 4.7x
  faster with 87 to 89% less allocation traffic. The encoded PNG stays about
  50% larger by design; decoded pixels are validated bit-identical by the
  benchmark.
- CLI 500 pages: -52.3% wall time and -61.7% peak RSS. RSS is lower at every
  size. Wall time resolution is `/usr/bin/time %e` at 10 ms, so rows at 2 to
  20 pages quantize.
- 500-page PDF output is 1,420,537 bytes against v0.2.4's 1,390,014 (+2.2%).
  Page counts and ordered text needles validate in both trees.
- The historical Snapshot I 0.2.4 row (1,009.80 ms at 500 pages) reads faster
  than this session's fresh v0.2.4 build (1,142 ms), which is the host-window
  drift the interleaved method removes from the comparison.

## Evidence paths

- Plan and row ledger: `plans/0.2.6/perf-review/phase-wise-checklist.md`
- Profiles and attribution: `plans/0.2.6/perf-review/results/2026-09-11/perf-04-05.md`
- PDF implementation evidence: `.../pdf-02-05.md`, `.../pdf-06.md`
- Image evidence: `.../img-01.md`, `.../img-02-04.md`, `.../artifacts/`
- Final capture: `.../valid-02-04.md`, `.../valid/sample1..3/`
- Committed publication: `documentation/performance.md`,
  `testdata/golden/benchmarks/benchmark-results.txt` Snapshot K,
  `testdata/golden/benchmarks/{cli,weasyprint,puppeteer}-compare.md`
- Fresh v0.2.4 head-to-head (2026-09-11T19:33Z): raw captures and
  `comparison.md` under `plans/0.2.6/perf-time/results/vs-0.2.4/` (local,
  gitignored); this file carries the summary tables above
- Wave 2 ledger for the current tree's extra rows:
  `plans/0.2.6/perf-improve/wave-2-50pct/phase-wise-checklist.md`
