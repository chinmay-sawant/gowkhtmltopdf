# Style resolution deep profile, 0.2.6 perf-time baseline

Task: attribute CPU and allocation inside `convert.Run` for the committed
post-perf-improve baseline, count style-resolution work at 2, 50, and 500
pages with a temporary probe, and rank architectural options for cutting
style time without growing B/op.

Read-only result: the source tree is byte-identical to the pre-profile state
(sha256 proof in section 4.3). The temporary probe was deleted after the
capture. No Git command ran.

## 1. Capture header

- date: 2026-09-11, 19:14:32-19:15:03 IST
- go: go1.26.4 linux/amd64
- cpu: 13th Gen Intel(R) Core(TM) i7-13700HX, 24 CPUs
- os: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64
- host state: four perf-time agents shared the host. Load average 1.09 at
  19:07. The heavy-profile lock
  (`/tmp/opencode/perf-time-profile.lock`) was held by another agent when
  this session started. Waiter started 19:05:21, acquired after 541 s at
  19:14:22 (recorded in `raw/style/lock-wait.log`). The lock directory was
  observed removed by another agent at about 19:16, before this session's
  `rmdir`; no other profiler was running during the capture interval (pgrep
  checked at 19:14:17).
- binary: `go test -c -o /tmp/opencode/perf-time-style/convert.test
  ./internal/convert`, built from the committed tree at 19:05, build ID
  `0af59f748368a8ff306507da57dae332bff66216`.
- workdir for every benchmark: `internal/convert`
- fixture: `testdata/golden/benchmarks/templates/report.html.tmpl`
- benchmark anchors: `-test.run '^$' -test.bench
  '^BenchmarkPDFPages$/^generic$/^NPages$' -test.benchtime=1x`, per-element
  anchors as required by the earlier capture note. `-test.count=5` for 500p,
  `-test.count=25` for 50p, `-count=2` for the allocation profile.
- commands: `raw/style/capture.sh`; attribution scripts
  `raw/style/style-attribution.sh` and `raw/style/style-alloc.sh`.
- quiet re-capture: 19:43:00-19:43:36, started with load average 0.50 and no
  other heavy process (`pgrep -af 'convert.test.*test.bench'` empty apart
  from the checking shell). Outputs under `raw/style/quiet/`. The 500p
  `-count=8` process drifted (rows 1.40-2.39 s) and is kept as run B; a fresh
  `-count=5` process immediately after ran 1.16-1.26 s and is run C, the
  anchor-paced capture. Both are analyzed beside the first under-load run
  (run A, section 3.4).

Honesty note on wall time: the under-load capture (run A) ran 1.32-1.50 s
warm. The quiet re-capture produced both a drifting process (run B,
1.40-2.39 s) and an anchor-paced process (run C, 1.16-1.26 s) from the same
binary. The committed baseline this report anchors to, 1,228.72 ms /
234.92 MB (perf-improve phase 7), matches run C within 1 to 3 percent.
Absolute estimates below use the committed row plus run C's per-phase
measurements; runs A and B are kept as share cross-checks.

## 2. Warm baseline in this session

500p, no profiler, `-test.count=5` (row 1 cold, rows 2-5 warm):

```
1442.9  1459.7  1502.8  1406.7  1499.4  ms/op
B/op: 235,732,208  234,930,560  235,054,184  235,185,880  235,061,520
allocs/op: 1,225,494 ... 1,225,285
```

50p, no profiler, `-test.count=25` (warm median about 116 ms; B/op
24.82-24.90 MB flat).

500p CPU-profile run rows: 1428.2, 1382.7, 1429.0, 1344.5, 1315.5 ms.
500p allocation run (`-count=2`): 1447.1 and 1440.3 ms, B/op 235.3 MB and
235.2 MB. The committed 234.92 MB row is reproduced within 0.2 percent.

Quiet re-capture, same binary, no other heavy process:

```
run B 500p count=8   : 1778.8 1400.8 1865.3 2119.7 1540.8 1849.5 2393.6 1791.5 ms
run B 500p mem count=2: 1615.8 1300.4 ms, B/op 235.6 / 236.0 MB
run C 500p count=5   : 1241.6 1247.6 1264.2 1161.6 1262.2 ms, B/op 234.8-235.4 MB
run C gctrace count=3: 1312.8 1232.4 1231.2 ms
```

Run C reproduces the committed row. The same binary produced 1.16 s and
2.39 s rows in different processes, so the spread is process and heap state,
as the perf-improve profile already noted, not measurement drift within a
single run. B/op is flat across all runs at 234.8 to 236.0 MB.

## 3. CPU attribution inside convert.Run

### 3.1 Phase shares

Method: `go tool pprof -top -nodefraction=0 -show_from=<root>` sums flat
samples on all stacks through a root. Style root is
`internal/layout.resolveStylesForLayoutContext$`. Total sample time is the
profile denominator; `convert.Run` is the benchmark's timed function.

| root | 500p samples | share of total | 50p samples | share of total |
|---|---:|---:|---:|---:|
| total profile | 7.440 s | 100.0% | 3.150 s | 100.0% |
| `convert.Run` | 6.920 s | 93.0% | 2.820 s | 89.5% |
| **style resolution** | **2.350 s** | **31.6%** | **1.070 s** | **34.0%** |
| `engine.build` (box construction) | 1.840 s | 24.7% | 0.830 s | 26.3% |
| `PaintContext` (paint + pagination) | 1.650 s | 22.2% | 0.460 s | 14.6% |
| `pdfPipeline.Finalize` | 0.620 s | 8.3% | 0.300 s | 9.5% |
| GC mark workers | 0.340 s | 4.6% | 0.190 s | 6.0% |

Style share of `convert.Run` alone: 34.0% at 500p, 37.9% at 50p. The share
is size-stable. The anchor-paced quiet capture (run C) puts 500p at 33.8% of
total samples and 35.4% of `convert.Run`; section 3.4 reconciles five
captures. The earlier perf-improve profile measured 23.3% at 500p
(`plans/0.2.6/perf-improve/profiles/warm-pdf-cpu.md`); the rise is not style
growth, it is the pagination, forced-break, seal and page-bucketing cuts
landing in phases 3-6 while style was untouched.

Style phase internals at 500p (`raw/style/500-peek-style-phase.txt`):

```
resolveStylesCtx                 cum 2.35s  31.59%   (flat 0)
  resolveStylesCtx.func1 (walk)  cum 2.34s  31.45%   (flat 0.03s 0.40%)
    resolveElementStyle          cum 1.79s  24.06%   (flat 0)
      cascadeRaw                 cum 0.50s   6.72%
      applyRawToUsed             cum 1.29s  17.34%
    styleStore.append            cum 0.29s   3.90%
    mapassign_fast64ptr          flat 0.23s  3.10%  (out[node] = sty)
  countStyleNodesContext         cum 0.01s   0.13%
```

### 3.2 Function table, 500p (7.44 s samples)

Flat is self time, cum includes callees. "Share" is share of the full
profile. Hot lines are from `raw/style/500-lists/`.

| function | flat | flat% | cum | cum% | hot lines |
|---|---:|---:|---:|---:|---|
| `resolveStylesCtx` | 0 | 0 | 2.35 s | 31.59% | `style.go:622` append 300 ms |
| `resolveStylesCtx.func1` (walk) | 0.03 s | 0.40% | 2.34 s | 31.45% | `style.go:639` text check 20 ms; `mapassign_fast64ptr` 230 ms |
| `resolveElementStyle` | 0 | 0 | 1.79 s | 24.06% | `style.go:800-801` |
| `applyRawToUsed` | 0.02 s | 0.27% | 1.29 s | 17.34% | `style.go:764` initialStyle 70 ms |
| `applyRestProps` | 0.03 s | 0.40% | 0.79 s | 10.62% | `:1185` raw lookup 100 ms, `:1190` shorthand apply 300 ms, `:1194` keys slice 50 ms, `:1204` sort 20 ms, `:1207` longhand apply 250 ms |
| `inheritProps` | 0.07 s | 0.94% | 0.32 s | 4.30% | `style_cascade.go:332` `raw[name]` 250 ms, `:327` loop 40 ms, `:341` copy 30 ms |
| `applyStyleProp` | 0.10 s | 1.34% | 0.53 s | 7.12% | `:1317` group dispatch 520 ms, `:1310` vendor normalize 10 ms |
| `applyBoxGroup` | 0.01 s | 0.13% | 0.20 s | 2.69% | box longhands |
| `applyBorderGroup` | 0.01 s | 0.13% | 0.20 s | 2.69% | border longhands |
| `resolveRawVars` | 0.01 s | 0.13% | 0.07 s | 0.94% | `containsVarFunc` 20 ms |
| `mergeCustomProps` | 0.01 s | 0.13% | 0.02 s | 0.27% | `style_cascade.go:69` |
| `applyFontProps` | inlined | not visible at 500p | 0.02 s at 50p | 0.63% at 50p | probe says 5.2 ms at 500p |
| `initialStyle` | 0.06 s | 0.81% | 0.06 s | 0.81% | `style.go:393`, `:472` |
| `cascadeRaw` | 0.02 s | 0.27% | 0.50 s | 6.72% | `:504` applyCascadeDeclaration 210 ms, `:540` wins to out 50 ms |
| `applyCascadeDeclaration` | 0 | 0 | 0.20 s | 2.69% | `:629` `prop+"-"+side` |
| `applyCascadeWin` | 0.02 s | 0.27% | 0.09 s | 1.21% | specificity compares |
| `supportedDeclaration` | 0 | 0 | 0.03 s | 0.40% | |
| `matchedRules` | 0 | 0 | 0.18 s | 2.42% | `style_cascade.go:390-402` |
| `appendRuleSelectorHits` | 0.03 s | 0.40% | 0.15 s | 2.02% | `:418` selectorMatches 140 ms |
| `selectorMatches` | 0 | 0 | 0.12 s | 1.61% | |
| `css.Match` | 0.01 s | 0.13% | 0.12 s | 1.61% | `match.go:202-203` |
| `css.leftmostMatch` | 0.07 s | 0.94% | 0.11 s | 1.48% | `match.go:259` matchPart 70 ms |
| `css.matchPart` | 0.01 s | 0.13% | 0.04 s | 0.54% | `strings.EqualFold` 20 ms, `hasClasses` 10 ms |
| `styleStore.append` | 0 | 0 | 0.29 s | 3.90% | |
| `styleInternFingerprint` | 0.05 s | 0.67% | 0.19 s | 2.55% | 230 field hashes, no single line above 10 ms |
| `styleInternHashString` | 0.09 s | 1.21% | 0.09 s | 1.21% | `style_intern_gen.go:17-25` |
| `styleInternEqual` | 0.06 s | 0.81% | 0.09 s | 1.21% | `runtime.memequal` 20 ms, `maps.Equal` 10 ms |
| `mapaccess2_faststr` (all callers) | 0.10 s | 1.34% | 0.46 s | 6.18% | 0.37 s of it under style: 0.23 s inheritProps, 0.14 s applyRestProps |

The anchor-paced run C reproduces the ordering and each share within 1 to 2
points at a 6.48 s total: `applyRawToUsed` 1.32 s, `applyRestProps` 0.83 s,
`applyStyleProp` 0.52 s, `cascadeRaw` 0.38 s, `styleStore.append` 0.34 s,
`matchedRules` 0.06 s. The 50p profile has the same shape (3.3).

Selector parsing is not in this phase. Stylesheets and selectors are parsed
during prepare; `css.Match` is already operating on parsed `Selector` structs.
There is no per-element selector compile step.

### 3.3 Function table, 50p (3.15 s samples)

Same shape, so only the differences worth noting:

| function | cum | cum% |
|---|---:|---:|
| style resolution | 1.07 s | 34.0% |
| `applyRawToUsed` | 0.69 s | 21.9% |
| `applyRestProps` | 0.40 s | 12.7% |
| `applyStyleProp` (under applyRestProps) | 0.29 s | 9.2% |
| `inheritProps` | 0.21 s | 6.7% |
| `styleStore.append` | 0.17 s | 5.4% |
| `cascadeRaw` | 0.16 s | 5.1% |
| `matchedRules` | 0.06 s | 1.9% |

### 3.4 Cross-capture reconciliation

Five independent captures of the 500p style phase: three from this session
(run A under load, runs B and C quiet) and the two sibling reports.

| capture | 500p rows | total samples | style samples | ms/iter | share of total | style / convert.Run |
|---|---:|---:|---:|---:|---:|---:|
| run A, under load, count=5 | 1.32-1.43 s | 7.44 s | 2.35 s | 470 | 31.6% | 34.0% |
| run B, quiet, count=8, drifted | 1.40-2.39 s | 15.78 s | 4.51 s | 564 | 28.6% | 30.7% |
| run C, quiet, count=5, anchor-paced | 1.16-1.26 s | 6.48 s | 2.19 s | 438 | 33.8% | 35.4% |
| `time-cpu.md` (sibling) | n/a | n/a | 2.38 s | 476 | 29.2% | n/a |
| `pagination-paint-deep.md` (sibling) | n/a | n/a | 2.50 s | 500 | 31.3% | n/a |

Run C internals at 500p: `engine.build` 1.64 s (25.3%),
`PaintContext` 1.42 s (21.9%), `Finalize` 0.61 s (9.4%), GC mark 0.13 s
(2.0%), `applyRawToUsed` 1.32 s (20.4%), `applyRestProps` 0.83 s (12.8%),
`applyStyleProp` 0.52 s (8.0%), `cascadeRaw` 0.38 s (5.9%),
`styleStore.append` 0.34 s (5.3%), `styleInternFingerprint` 0.25 s (3.9%),
`inheritProps` 0.22 s (3.4%), `resolveRawVars` 0.13 s (2.0%),
`matchedRules` 0.06 s (0.9%).

Style per iteration is 438 ms (run C) to 564 ms (run B), 470 ms run A,
476 ms time-cpu.md, 500 ms pagination-paint-deep.md. The spread tracks total
row time: memory stalls and GC assists inflate both the style work and the
denominator. The share therefore moves while the element-bound work does
not. Ranked by closeness to the committed 1,228.72 ms row, run C is the best
estimate: **438 ms per warm 500p conversion, 33.8% of total samples, 35.4%
of `convert.Run`, about 35.6% of the committed row**. The probe's direct
phase timing (section 4.2) lands on the same budget: 468.5 ms at a 1,447 ms
wall, 32.4%.

Component-level agreement with the siblings is close. `time-cpu.md` reports
`applyRestProps` 860 ms cum and `inheritProps` 290 ms at the `raw[name]`
lookup (`style_cascade.go:332`), against run A's 790 ms and 250 ms.
`pagination-paint-deep.md` reports `resolveStylesCtx` 2.50 s at 500p and
1.07 s at 50p (34.97%), against run A's 2.35 s and 1.07 s (34.0%). The
three reports agree on the shape and size of the style phase; the share
differences are denominator effects, not conflicting measurements.

## 4. Counts (temporary probe)

Probe design: a temporary `internal/layout/zz_style_probe.go` with a global
counter struct, calls added to `resolveStylesCtx`, `styleStore.append`,
`appendRuleSelectorHits`, `cascadeRaw`, `applyRawToUsed`, `applyStyleProp`,
and `applyCascadeDeclaration`, plus a temporary test
`internal/convert/zz_style_probe_test.go` that runs the committed benchmark
template through `convert.Run` at 2, 50, and 500 pages after one warm
conversion. `time.Now` pairs measured the phase split. Both probe files were
deleted after the run; the probe source is archived under
`raw/style/probe/` for reproducibility.

### 4.1 Work counters

| metric | 2 pages | 50 pages | 500 pages |
|---|---:|---:|---:|
| elements styled | 271 | 6,607 | 66,007 |
| text nodes | 289 | 6,913 | 69,013 |
| `styleStore.append` calls | 271 | 6,607 | 66,007 |
| intern hits | 256 | 6,592 | 65,992 |
| intern misses (records created) | 15 | 15 | 15 |
| intern hit rate | 94.5% | 99.77% | 99.977% |
| cascade raw entries (total) | 2,576 | 64,064 | 640,514 |
| raw entries per element | 9.51 | 9.70 | 9.70 |
| properties applied per element | 9.51 | 9.70 | 9.70 |
| selector checks per element | 12.38 | 12.17 | 12.17 |
| selector matches per element | 1.15 | 1.17 | 1.17 |
| declarations considered per element | 2.94 | 3.00 | 3.00 |

The decisive number: **the entire 500-page document resolves to 15 distinct
`ResolvedStyle` records.** The pinned style storage, 221,208 B, is exactly
one `styleStoreChunkSize = 64` chunk: 64 x 3,432 B (`unsafe.Sizeof`, DWARF
verified) = 219,648 B plus slice and map overhead (`internal/layout/style.go:705-721`).
Resolution still runs 66,007 full cascades to produce those 15 records. The
deduplication happens at insertion, after the work is paid.

### 4.2 Time split at 500p

First probe run, steady state (one warm conversion before the timed one),
wall 1,447.4 ms. Counts were identical in a second run; timings there were
inflated by host load (wall 1,986.8 ms), so the first run is reported.

| bucket | ms | share of wall |
|---|---:|---:|
| selector matching (`matchedRules`) | 32.9 | 2.3% |
| raw declaration collection (rest of `cascadeRaw`) | 66.2 | 4.6% |
| property application (`applyRawToUsed`) | 302.9 | 20.9% |
| - inherit | 68.7 | 4.7% |
| - custom props | 11.0 | 0.8% |
| - var() | 16.0 | 1.1% |
| - font | 5.2 | 0.4% |
| - rest (`applyRestProps`) | 174.8 | 12.1% |
| - initialStyle copy, famHash, unitless line-height | 27.2 | 1.9% |
| intern fingerprint | 40.8 | 2.8% |
| intern equality check | 25.7 | 1.8% |
| **style total** | **468.5** | **32.4%** |

The probe adds about 10 `time.Now` pairs and several counter increments per
element, so the wall and the style total are inflated by an unquantified
few percent. The CPU profile, which has no per-element instrumentation,
agrees at 34.0% of `convert.Run`. The two independent methods bound the
style share at 32 to 34 percent.

At 50 pages the same split gives 44.9 ms of 121.3 ms (37%) in the first
run; at 2 pages, 2.4 ms of 6.7 ms. Per-element cost is flat once past the
fixed conversion setup: 500p style is 10.4x the 50p style time for 10.0x
the elements (7.1 vs 6.8 microseconds per element).

### 4.3 Probe deletion proof

```
$ find internal -name '*style_probe*'
(no output)
$ ls internal/layout/zz_style_probe.go internal/convert/zz_style_probe_test.go
ls: cannot access 'internal/layout/zz_style_probe.go': No such file or directory
ls: cannot access 'internal/convert/zz_style_probe_test.go': No such file or directory
$ sha256sum internal/layout/style.go internal/layout/style_cascade.go
aaf2fb5aa505c9dd9f5c44d33bf3c015b3e2dee3dc083db79418da0a0036074e  internal/layout/style.go
3df117522c42cbc8a39fe30fb59f87cbd8c077c7af440fa9a72210711bfee5e2  internal/layout/style_cascade.go
$ diff -q internal/layout/style.go /tmp/opencode/perf-time-style/orig/style.go
$ diff -q internal/layout/style_cascade.go /tmp/opencode/perf-time-style/orig/style_cascade.go
(no output: restored files are byte-identical to the pre-probe backups)
$ go test ./internal/layout -run 'TestStyleIntern' -count=1
ok  github.com/chinmay-sawant/gowkhtmltopdf/internal/layout  0.066s
```

## 5. Allocation attribution, 500p, alloc_space, rate 65536

Profile `500-mem.pprof`, `-test.count=2` (two consecutive conversions), total
465.30 MB, which is 232.7 MB per conversion. Only the style subtree is shown
plus the phases needed for the ceiling. Full views:
`raw/style/alloc/alloc-top-flat-full.txt` (function list) and
`raw/style/alloc/alloc-style-phase-flat.txt` (style subtree only).

| site | bytes, 2 ops | bytes/op | share of profile | line |
|---|---:|---:|---:|---|
| `applyRestProps` keys slice | 20.84 MB | 10.42 MB | 4.48% | `style_cascade.go:1194` `keys := make([]string, 0, len(raw))` |
| `applyCascadeDeclaration` box-shorthand property strings | 12.88 MB | 6.44 MB | 2.77% | `style_cascade.go:629` `prop+"-"+side` |
| `resolveStylesCtx` node-to-style map | 8.17 MB | 4.08 MB | 1.76% | `style.go:608` `make(map[*html.Node]*ResolvedStyle, nodeCount)` |
| `styleStore.append` chunks and intern map | 0.44 MB | 0.22 MB | 0.09% | `style.go:732-744`, equals the 221,208 B pin |
| **style subtree total** | **42.33 MB** | **21.17 MB** | **9.10%** | |

What is not there: per-element raw maps. `cascadeWins` and `cascadeProps` are
allocated once per `styleContext` (`cascadeWinHint = 8`,
`style_cascade.go:477-488` and `:527-538`) and reused with `clear()`. Their
cost is CPU (`mapclear` 20 ms, `mapSet`/`mapIter` about 60 ms at 500p), not
bytes. Per-element temporaries that do allocate are exactly the three style
sites above plus the 6.44 MB/op of concatenated shorthand names. The
fingerprint and equality passes allocate nothing; they are pure CPU.

The quiet re-capture (run B mem, two conversions, total 470.99 MB) puts the
style subtree at 43.80 MB (9.30%) with the same four sites and the same
per-op order: `applyRestProps` 19.90 MB, `applyCascadeDeclaration` 13.44 MB,
`resolveStylesCtx` 10.02 MB, `styleStore.append` 0.44 MB. Style's 21.2 to
21.9 MB/op is about 9% of the 234.9 MB/op at 500p.

GC in this profile is 0.34 s at 500p (4.6%) and 0.19 s at 50p (6.0%),
mark workers only; the anchor-paced run C has GC at 0.13 s (2.0%).

## 6. Intern guarantees that must survive

`internal/layout/style_intern_gen_test.go` is generated-code contract, not
decoration. Any option below must keep all five green:

1. `TestStyleInternGeneratedFieldsComplete`: the generated field list must
   match `ResolvedStyle` exactly. Adding a field (for example a
   property-presence mask) requires `go generate ./...` and a regenerated
   `style_intern_gen.go` (`internal/layout/style.go:2`).
2. `TestStyleInternGeneratedEqualityMatchesReflect`: `styleInternEqual` must
   agree with `reflect.DeepEqual` on random styles, including NaN fields
   (NaN must compare unequal) and every single-field mutation.
3. `TestStyleInternGeneratedEqualityComparesContents`: slices and maps
   compare by content, and equal styles share a fingerprint.
4. `TestStyleInternGeneratedFingerprintNormalizesZero`: -0.0 and +0.0 must
   hash identically because they compare equal.
5. `styleStore.append` must return a shared, immutable pointer for equal
   declarations. `internal/layout/style_share_test.go` pins pointer identity
   (`TestRepeatedReportCellsStyleReuseImmutable`), cross-resolution
   immutability (`TestStoredStyleValuesImmutableAcrossResolutions`),
   independent custom-property maps, page-name clone isolation, and
   container re-cascade sharing.

Constraints that shape the options:

- Style storage must stay at 221,208 B. That is one chunk of 64 records at
  3,432 B each. A new `ResolvedStyle` field grows every record and breaks
  the pin, so a presence mask must live beside the record (a local, a
  parameter, or `styleStore.candidate`), not inside the stored struct.
- File size: `style_cascade.go` is 1,384 lines, `style.go` 836. New cache
  code belongs in its own file.
- One owner per `internal/layout` at a time; golden corpus gates every
  layout change.

## 7. Architectural options, ranked

Anchors for the estimates: committed warm 500p is 1,228.72 ms. The
anchor-paced profile (run C) puts style at 438 ms per conversion (2.19 s
over 5 iterations), 33.8% of total samples and 35.4% of `convert.Run`; the
probe's direct timing is 468.5 ms at a 1,447 ms wall. Across the five
captures style is 438 to 564 ms; the plan budget is **440 ms**, about 36% of
the committed row. "Recover" means milliseconds removed from that budget
under the stated option. All estimates are arithmetic on measured
subcomponent shares, not measured implementations.

### Option 1: memoize resolution for repeated structures (highest payoff)

Mechanism. Cache the resolved style for (parent style pointer, exact ordered
matched-declaration identities, node name, pseudo target, inline style text,
href under the link-underline policy). On hit, skip `cascadeRaw` declaration
application, all of `applyRawToUsed`, and the fingerprint/equality scan;
return the pointer already stored by `styleStore.append`. The exact hit
list makes :nth-child, :first-child, sibling combinators, :has and
container gates part of the key, because rules that did not match are simply
absent. Collision safety follows the existing intern pattern: store the key
components with the entry and compare them, do not trust the hash. Elements
without a parent (`html`, which sets `ctx.remBase`) are excluded.

Measured targets: run C `cascadeRaw` 0.38 s (of which `matchedRules`
0.06 s), `applyRawToUsed` 1.32 s, `styleStore.append` 0.34 s. Together
those are 1.98 s of the 2.19 s style phase, 90% of it, or 30.5% of the
6.48 s total. Residual after a perfect cache: `matchedRules` 0.06 s, the
`out`-map write 0.13 s, the walk, plus key hashing and lookup, call it 0.26
to 0.31 s of profile, 49 to 59 ms at the 1,228.72 ms anchor, plus 20 to 30
ms of cache overhead. Net expectation: recover **300 to 360 ms**, leaving
style at roughly 80 to 140 ms and the 500p row at about 870 to 930 ms
(1.32x to 1.41x). At 500p the cache sees 66,007 lookups against 15 distinct
keys.

B/op: down. The keys slice (10.42 MB/op) and the shorthand-name strings
(6.44 MB/op) are only produced on misses, so expect 500p B/op to fall by
roughly 15 to 17 MB. Style storage stays 221,208 B.

Correctness risks: high, concentrated in key completeness. Inheritance and
custom properties require the parent pointer in the key. `var()` values are
part of the raw declarations and are covered by the hit identity. Page
names are stamped later on cloned records, which the share tests already
pin. Container queries are resolved into the hit list, but the list must
also capture the gate outcome for elements whose ancestor container sizes
change between passes; simplest safe rule is to disable the cache on any
pass with a non-nil `ctx.containers`. Inline `style` attributes and the
`--print-link-underline` anchor policy must be key inputs or excluded.

Falsifiable probe: temporary hit/miss and key-build counters at 500p.
Require hits >= 65,900 and misses <= 100 on the report template, style
phase cum <= 0.8 s (from 2.19 to 2.35 s), total samples <= 5.5 s on the
anchor-paced process, byte-identical
output for all 65 fixtures after date normalization, `TestStyleIntern*` and
the `style_share_test.go` pointer-identity suite green, and a new
differential test that resolves the same tree with and without the cache
for documents containing :nth-child, :first-child, +, ~, :has, inline
styles, anchors with href, custom properties, and @container rules.

### Option 2: declared-property mask, ID-keyed map, no per-element keys slice

Mechanism. Three related changes that remove per-element lookups and
temporary slices: (a) while building the cascade winners, OR a bitmask of
declared properties (a fixed table maps property name to bit); `inheritProps`
tests bits instead of `raw[name]` for 70 entries per element; (b) iterate
the winners directly in the longhand pass instead of allocating
`keys := make([]string, 0, len(raw))` and sorting it; shorthands already run
first in a fixed order; (c) optionally back the winners with parallel
property-ID and value arrays so `raw[prop]` lookups in the shorthand pass
become array reads. Keep any mask out of `ResolvedStyle` so the 221,208 B
pin and the generated intern code stay untouched.

Measured targets: `inheritProps` 0.22 s in run C and 0.32 s in run A
(`raw[name]` 0.25 s of it), `applyRestProps` map lookups plus keys slice
plus sort (about 0.21 s of its 0.79 to 0.83 s), and `mapaccess2_faststr`
0.46 to 0.47 s overall. Expect **90 to 120 ms** at the anchor, about 8 to 10
percent of total. B/op: down by 10.42 MB/op for the keys slice, and 6.44
MB/op more if (c) replaces the string concatenation; 500p B/op moves toward
218 to 222 MB.

Correctness risks: medium. The per-element sort in `applyRestProps` was
added for deterministic order (`style_cascade.go:1193`); if two remaining
longhands can write the same field, map order would become observable.
A static audit plus a differential test that applies the same winners in
map order and sorted order over a declaration corpus is the gate. var()
values still need their raw strings, and `applyIgnoredGroup` fallthrough
must be preserved.

Falsifiable probe: counter for `inheritProps` lookups (currently about 70
per element) must fall to near zero; `sort.Strings` and the
`style_cascade.go:1194` makeslice must disappear from the allocation
profile; `inheritProps` cum <= 0.1 s; golden byte-identical; B/op at or
below 234.92 MB.

### Option 3: generated dispatch table plus a rightmost-selector rule index

Mechanism. (a) Replace the 11-entry `styleGroups` scan in `applyStyleProp`
(`style_cascade.go:1316-1320`, 0.04 to 0.10 s flat, 640,514 calls) with one
generated switch or a property-ID table so a miss does not walk up to 11
function values. (b) Index author rules by rightmost tag/class so
`matchedRules` stops testing all 12 rules for all 66,007 elements; merge
bucket hits back in stylesheet order. Parsed selectors already exist, so
this is index construction, not a compile step.

Measured targets: `applyStyleProp` flat 0.04 s in run C and 0.10 s in run A
(0.52 to 0.53 s cum) plus part of the failed-group attempts, and
`matchedRules` 0.06 to 0.18 s. Note the group bodies themselves
(`applyBoxGroup` 0.20 s, `applyBorderGroup` 0.20 s) do the real application
and are not removed by dispatch. Expect **50 to 90 ms** at the anchor,
about 4 to 7 percent of total. B/op is neutral.

Correctness risks: medium. Vendor prefix normalization
(`normalizeVendorPrefix`), webkit value remaps, and `applyIgnoredGroup`
fallthrough must all survive; the rule index must preserve declaration
order because the cascade compares source order. Both changes are
mechanical and well covered by the existing property tests.

Falsifiable probe: `applyStyleProp` cum <= 0.25 s (from 0.53 s),
`matchedRules` cum <= 0.06 s (from 0.18 s), golden byte-identical, and a
declaration corpus test comparing applied fields between the old and new
dispatch paths.

### Options considered and not ranked

- Fingerprint/equality cost (0.29 s combined): subsumed by Option 1, which
  makes these run only for the 15 misses. Cache the fingerprint with the
  record only if Option 1 is rejected.
- `initialStyle()` 3,432-byte copy per element (0.06 s profile, 60 ms):
  small; a parent-copy-then-reset design overlaps Option 2 but adds field
  semantics risk for little gain.
- The `out` node-to-style map (0.23 s CPU, 4.08 MB/op, 135k entries): a
  dense node index would need a node-to-index map anyway; leave it.
- Parallel subtree resolution: could cut style wall time on 24 cores, but
  per-worker `styleStore` chunks multiply style storage past the 221,208 B
  pin (one chunk per worker), a shared store reintroduces contention, and
  determinism of interned pointer identity becomes hard to test. High risk,
  not a first move.

## 8. Honest ceiling for a 2x warm cut

Warm 500p is 1,228.72 ms. Style is 438 ms per conversion at anchor pace
(run C) and 468 ms by probe timing; across captures the budget is 400 to
470 ms. The arithmetic:

- Delete all style work (impossible; matching, the walk and map writes
  remain): 1,228.72 - 438 = about **791 ms, 1.55x**. Across the 400 to
  470 ms budget range that is 1.48x to 1.62x. This is the theoretical
  ceiling from style alone.
- Land Option 1: recover 300 to 360 ms, row about **870 to 930 ms,
  1.32x to 1.41x**. Options 2 and 3 are not additive on top: once cache
  hits skip `applyRestProps` and `applyStyleProp`, those options only run
  for the 15 misses and add almost nothing. Without Option 1, Options 2 and
  3 combine to about 150 to 210 ms. Either way, the realistic style ceiling
  is about 360 ms.
- 2x means 614 ms, a 615 ms cut. Style cannot supply more than about 438 ms
  even in the limit, and about 360 ms realistically. The remaining 255 to
  315 ms must come from the other phases at run C's shares: `engine.build`
  (~311 ms at 25.3%), `PaintContext` (~269 ms at 21.9%), `Finalize`
  (~116 ms at 9.4%), GC (~25 to 57 ms at 2.0 to 4.6%), and parse/outline
  residual.

Stated plainly: style is the single largest block and the plan should land
Option 1 first, but style alone caps out at about 1.55x in the theoretical
limit and about 1.40x realistically. Reaching 2x requires the display-list
and paint workstreams to roughly halve their remaining time in the same
release. Every option in section 7 reduces or holds B/op, so the 234.92 MB
and 221,208 B pins are not a trade-off against the time work.

## 9. Evidence index

All under `plans/0.2.6/perf-time/profiles/raw/style/` unless noted.

- this report: `plans/0.2.6/perf-time/profiles/style-deep.md`
- profiles: `500-cpu.pprof`, `50-cpu.pprof`, `500-mem.pprof`
- raw runs: `500-run.txt`, `50-run.txt`, `500-cpu-run.txt`, `50-cpu-run.txt`,
  `500-mem-run.txt`, `500-gctrace.txt`, `probe-run.txt`
- pprof text: `500-cpu-top-flat-full.txt`, `500-cpu-top-cum-full.txt`,
  `50-cpu-top-flat-full.txt`, `50-cpu-top-cum-full.txt`,
  `500-peek-style-phase.txt`, `500-peek-style-functions.txt`,
  `500-peek-intern.txt`, `500-peek-selector.txt`, `50-peek-*.txt`,
  `500-lists/`, `50-lists/`, `alloc/`, `500-cpu-phase-sums.tsv`,
  `50-cpu-phase-sums.tsv`
- quiet re-capture: `quiet/` holds `capture-quiet.sh`, the run logs,
  `500-cpu.pprof`, `50-cpu.pprof`, `500-cpu2.pprof`, `500-mem.pprof`, and
  the attribution text under `quiet/run-b/` (500p count=8, 50p, alloc) and
  `quiet/run-c/` (anchor-paced 500p count=5)
- probe: `probe/zz_style_probe.go`, `probe/zz_style_probe_test.go`
  (deleted from the tree; archived here)
- tooling: `capture.sh`, `style-attribution.sh`, `style-alloc.sh`,
  `lock-wait.log`
- sibling cross-checks: `../time-cpu.md`, `../pagination-paint-deep.md`
