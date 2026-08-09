---
name: gowkhtmltopdf-rss-phase-wise-checklist
description: Evidence-backed phase-wise workflow for reducing gowkhtmltopdf peak RSS below 50 MiB without weakening PDF, pagination, image, or CLI semantics.
---

# Gowkhtmltopdf - RSS Reduction Phase-Wise Checklist

> **Parent:** `skills/phase-wise-checklist/SKILLS.md` - repository checklist procedure
> **Status:** profiling complete; implementation not started
> **Target:** direct `gowkhtmltopdf` 500-page CLI peak RSS below **50 MiB**
> **Scope:** `BenchmarkPDFPages/500Pages` and the equivalent CLI report fixture
> **Non-goals:** no Git operations, output-semantic weakening, benchmark-only shortcut, or publication claim without current evidence

---

## Overview

This is the canonical execution ledger for the RSS reduction program. It keeps
process RSS, Go allocation traffic, retained heap, wall time, and PDF semantics
as separate measurements. A lower `B/op` value is not proof of lower peak RSS.

The default renderer must remain unchanged for unsupported documents. Any
bounded or windowed path must fail closed, preserve the existing output
contract, and prove equivalence against the full-document path before it can
be used for the target claim.

## Current profile evidence

### Host and commands

- Host: Linux/WSL2, amd64, 24 CPUs, Intel i7-13700HX.
- Toolchain: Go 1.26.4.
- Current source: `internal/convert`, `internal/layout`, `internal/pdf`.
- Profile binary: compiled from `internal/convert` with `go test -c`.
- Benchmark: `BenchmarkPDFPages/500Pages`, one iteration, exact page-count
  assertion enabled.
- Profile artifacts:
  - `/tmp/gowk-rss-profile/pdf-500.cpu.pprof`
  - `/tmp/gowk-rss-profile/pdf-500.mem.pprof`
  - `/tmp/gowk-rss-profile/convert.test`

### Baseline measurements

| Measurement | Current result | Meaning |
|---|---:|---|
| Profiled Go benchmark time | 620.575 ms | One iteration with CPU/heap profiling overhead |
| Profiled Go `B/op` | 163,018,408 | Cumulative allocation traffic, not RSS |
| Profiled Go allocations | 535,175 | Allocation count per benchmark operation |
| Profiled Go peak RSS | 94,988 KiB / 92.8 MiB | `/usr/bin/time -v` around the compiled benchmark process |
| Direct CLI time | 0.94 s | 500-page report-shaped fixture |
| Direct CLI peak RSS | 167,808 KiB / 163.9 MiB | Current process-level CLI baseline |
| RSS target gap | 113.9 MiB | Direct CLI must remove this gap to reach 50 MiB |

### CPU profile signals

- `layout.(*engine).flowOneChild`: 170 ms cumulative, 23.94% of samples.
- `compress/flate.(*compressor).deflate`: 110 ms cumulative, 15.49%.
- `layout.(*styleStore).intern`: 40 ms cumulative, 5.63%.
- `layout.emitGridVerticals`: 40 ms cumulative, 5.63%.
- `layout.resolveRawVars`, `rowspanCellCovers`, map access/assignment, and
  `appendPDFNum` remain visible lower-order costs.

### Allocation profile signals

- `styleStore.append`: 44.62 MB flat, 23.96%; it copies each non-interned
  `ResolvedStyle` into chunk storage.
- `bytes.growSlice`: 29.93 MB flat, 16.07%; repeated buffer growth remains a
  measured retained/copying cost.
- `engine.buildCell`: 20.01 MB flat, 10.74%; table cell boxes are numerous in
  the report fixture.
- `resolveStylesCtx`: 52.17 MB cumulative, 28.01%; style map plus style-store
  construction dominates the style-resolution allocation family.
- `buildFlowOpIndex`: 5.51 MB flat; `pageOf`, `pos`, counts, and page buckets
  are additional full-document indexes.
- `paintPages`: 24.98 MB cumulative, 13.41%; page content growth and paint
  ordering retain per-page output state.
- `pdf.Content.textShowSimple`: 4.01 MB flat; text stream construction is a
  smaller but real output-side allocation family.

The `inuse_space` heap sample at benchmark end was only 8.94 MB because the
runtime had already reclaimed much of the cumulative allocation traffic. It is
not a peak-RSS explanation. Peak RSS must be measured independently with
`/usr/bin/time -v` and, in Phase 1, with a sampled runtime/process metric.

## Phase 0: Measurement and semantic contract

### 0.1 Reproducible baseline

- [x] Compile the benchmark binary from `internal/convert`; use `-test.*`
  flags, not unprefixed test-binary flags. Proof: profile binary executed and
  rendered the 500-page benchmark.
- [x] Capture CPU, alloc-space, in-use heap, `B/op`, allocations/op, wall time,
  and peak RSS on the declared host. Proof: current artifacts and table above.
- [x] Capture a direct CLI RSS baseline separately from the in-process
  benchmark. Proof: `/tmp/gowk-rss-profile/gowkhtmltopdf-500.pdf` run measured
  167,808 KiB peak RSS.
- [ ] Repeat the direct CLI baseline three times and record median, minimum,
  maximum, fixture bytes, page count, and output bytes in the ledger.

### 0.2 Hard correctness oracle

- [ ] Define a normalized PDF oracle for page count, page dimensions, text,
  resource counts, annotations, links, destinations, headers/footers, and
  deterministic object ordering where the current writer guarantees it.
- [ ] Add the oracle for full-document versus candidate RSS-reduced rendering;
  the candidate must be rejected on any mismatch.
- [ ] Add negative fixtures for tables, forced breaks, fixed/sticky content,
  links, fonts, images, transforms, floats, flex/grid, and unsupported CSS.

## Phase 1: Peak-RSS observability

- [ ] Add a test-only or opt-in RSS sampler that records peak `VmRSS` at a
  fixed interval and emits start, high-water, post-render, and post-GC values.
  Keep `/usr/bin/time -v` as the external gate.
- [ ] Add phase markers for parse, style resolution, layout, pagination, paint,
  PDF finalization, and output writing. Markers must be disabled by default.
- [ ] Record live object counts and capacities for DOM nodes, resolved styles,
  `Result.Ops`, page buckets, boxes, PDF pages, content buffers, fonts, and
  image resources at each marker.
- [ ] Verify that the sampler itself changes neither output bytes nor the
  benchmark's selected rendering path.

## Phase 2: Input and document-retention budget

- [ ] Measure source HTML, DOM, attribute strings, text nodes, token/build
  buffers, and cloned input bytes separately. Do not claim a reduction from
  aggregate GC or `B/op` alone.
- [ ] Remove only duplicate retained copies whose ownership and lifetime are
  proven. Preserve source-backed string behavior and malformed-input handling.
- [ ] Release or shorten lifetimes after each consumer has completed: source
  buffer, temporary token/build state, parsed document indexes, and completed
  navigation projections.
- [ ] Add a retention test proving links, headings, anchors, headers/footers,
  and errors do not depend on released DOM or layout objects.

## Phase 3: Style and layout allocation reduction

### 3.1 Style storage

- [ ] Measure style cardinality, intern hit rate, custom-property exclusions,
  and bytes per canonical style on the benchmark fixture.
- [ ] Reduce `styleStore.append` traffic only with semantic equality coverage
  for inheritance, selectors, media/container rules, font-family ordering,
  custom properties, grid, and flex overrides.
- [ ] Prove that style canonicalization does not mutate shared styles and does
  not retain a larger global store than the current per-document lifetime.

### 3.2 Table and box storage

- [ ] Measure `buildCell` box fields and identify fields that are required after
  paint, after pagination, and after PDF finalization.
- [ ] Reuse or compact table-cell and border storage only behind a lifetime
  proof; preserve colspan, rowspan, collapsed borders, row breaks, and paint
  ordering.
- [ ] Bound `bytes.growSlice` and display-list growth with measured capacities;
  do not preallocate the full 500-page output without proving RSS improvement.

### 3.3 Index lifecycle

- [ ] Measure `pageOf`, `pos`, page buckets, flattened boxes, fixed-op indexes,
  and location maps separately.
- [ ] Reuse or release indexes after the last mutation/consumer; retain them
  when pagination or links still require them.
- [ ] Prove exact page counts and no phantom paint after every index-lifetime
  change.

## Phase 4: Bounded page-island or streaming architecture

- [ ] Define a fail-closed eligibility contract for independent page islands;
  unsupported cross-page flow, links, fixed/sticky elements, tables, floats,
  transforms, flex/grid interactions, and dynamic selectors must fall back.
- [ ] Build one island at a time with bounded style/layout/display-list storage;
  release each island only after navigation projection and PDF page content are
  sealed.
- [ ] Spool sealed PDF page streams or equivalent page content without changing
  compression, resource numbering, xref offsets, annotations, or ordering.
- [ ] Prove the candidate uses bounded peak storage at 2, 10, 100, 250, and
  500 pages; a 500-page-only special case is not sufficient.
- [ ] Differential-test the bounded path against the full-document path using
  the Phase 0 oracle and negative fallback fixtures.

## Phase 5: PDF and output-side retention

- [ ] Profile PDF page content, zlib writers, font subsets, glyph maps, image
  resources, xref/object tables, and final output buffers as separate families.
- [ ] Reuse or stream completed page content only when partial-write errors,
  retries, copies, page reordering, and output ownership remain correct.
- [ ] Keep compression settings and PDF validity fixed while measuring memory;
  disabling compression is diagnostic only, not an optimization result.
- [ ] Compare file-backed output, stdout output, and `io.Discard` separately;
  report them as distinct workloads.

## Phase 6: Closure gates

- [ ] Direct CLI 500-page peak RSS is **<= 50 MiB** across three runs, with no
  run exceeding the agreed stability bound.
- [ ] Compiled benchmark process peak RSS is reported separately and does not
  substitute for the direct CLI gate.
- [ ] 500-page PDF and template benchmarks report median wall time, `B/op`,
  allocations/op, peak RSS, PDF bytes, and page count.
- [ ] Full matrix passes at 2/5/10/20/50/100/200/250/500 pages and image tiles.
- [ ] Full-vs-candidate semantic oracle passes; existing tests, visual goldens,
  link/location tests, PDF structure tests, and image pixel tests pass.
- [ ] Run the repository's required lint and test gates for any implementation
  phase. Documentation-only checklist edits do not close implementation rows.
- [ ] Update this ledger with exact commands, host, cache state, raw samples,
  median arithmetic, remaining risks, and the next unchecked phase.

## Forbidden shortcuts

- Do not equate `B/op` with RSS.
- Do not claim the 50 MiB target from the in-process benchmark alone.
- Do not remove CSS/layout features, links, fonts, images, compression, or PDF
  structure merely to reduce memory.
- Do not use a benchmark-only input marker without a fail-closed eligibility
  check and a full-document semantic oracle.
- Do not mark a phase complete from a profile or plan statement without current
  source, test, and measurement evidence.

## Required report format

| Engine/path | Pages | Median wall | Peak RSS | B/op | Allocs/op | PDF bytes | Oracle/tests |
|---|---:|---:|---:|---:|---:|---:|---|
| Full-document CLI | 500 | pending | pending | n/a | n/a | pending | pending |
| Candidate CLI | 500 | pending | pending | n/a | n/a | pending | pending |
| In-process benchmark | 500 | pending | pending | pending | pending | pending | pending |

Every future result must state whether the measurement is a direct CLI process,
compiled test binary, or in-process benchmark. Historical snapshots remain
historical and cannot close current rows.
