# Deferred 0.0.3: 500-Page PDF in One Second

**Status:** Deferred investigation  
**Created:** 2026-08-07  
**Current branch:** `feature/optimization`

**Original benchmark snapshot:** `2a0f18b`
**Current optimization commit:** `aa8d446`

## Objective

Investigate whether the current Go PDF pipeline can generate the 500-page PDF
benchmark within approximately one second on the reference benchmark machine.

This is a separate high-performance architecture target, not a continuation of
the completed incremental optimization checklist.

## Current position

The latest locked count-3 benchmark median for the PDF path is:

| Workload | Current median | Target | Further reduction required |
|---|---:|---:|---:|
| PDF, 500 pages | **1.628 s** | 1.00 s | **38.6%** |
| Template PDF, 500 pages | 1.693 s* | 1.00 s | **40.9%*** |

The same one-iteration comparison in the benchmark snapshot reduced the
original PDF baseline from 14.135 s to 1.903 s, approximately **86.5% faster**;
the current count-3 median is 1.628 s. The one-second target still requires a
materially different level of optimization.

\* The current template value is the recorded one-iteration matrix result; a
separate count-3 template gate has not been recorded in this wave.

## Why incremental fixes are insufficient

The final PDF profile still contains several substantial costs:

- style application (`resolveElementStyle` / `resolveStylesCtx`)
- forced-break placement (`beforeAlways`)
- map operations, text measurement, and zlib compression
- allocation traffic from the style arena, HTML tokenization, display-list
  growth, crossing-rectangle splitting, and table-cell construction

The current profile measured approximately 391 MiB peak RSS for the benchmark
process. These costs remain serial and retained-state heavy; removing one
hotspot completely would not be enough to reach one second or the aspirational
sub-100-MB process target.

## Candidate architecture directions

1. Parallelize independent page painting and PDF stream compression after global
   pagination has settled.
2. Reuse resolved styles, layout structures, fonts, glyphs, and repeated table
   geometry more aggressively.
3. Reduce or redesign global pagination fixpoint passes so page placement does
   not repeatedly move large suffixes of the display list.
4. Add a specialized repeated/template-page fast path where the input contract
   permits it.
5. Measure compression, serialization, and output buffering separately; assess
   whether optional or parallel PDF compression is acceptable.
6. Re-evaluate the target against a clearly defined output contract, because a
   one-second target with full CSS layout, pagination, PDF compression, and
   byte-equivalent output is substantially harder than a one-second fast path.

## Proposed execution sequence

1. Reproduce the 500-page benchmark on the reference machine and split wall time
   into parsing, cascade/style resolution, layout, pagination, rasterization,
   PDF serialization, compression, and output writing.
2. Establish an intermediate target of 4–5 seconds and confirm correctness,
   page counts, output validity, and allocation behavior.
3. Prototype parallel page painting/compression behind a benchmark-only or
   opt-in seam, then measure scaling and memory overhead.
4. Prototype repeated-page/template reuse and compare output and locations.
5. Re-profile after each architectural change before committing to the next
   target, with 2, 10, 50, 100, 250, and 500-page coverage.
6. Treat one second as achieved only when three repeated runs meet the target
   without weakening layout, PDF, image, or correctness requirements.

## Acceptance criteria

- 500-page PDF median at or below **1.00 s** on the declared reference setup.
- Three-run spread is reported and no run exceeds the agreed stability bound.
- Template PDF meets the same target or has a separately justified target.
- Page count, PDF validity, visual output, links, locations, and existing tests
  remain correct.
- Allocation and peak-memory limits are reported alongside wall time.
- The implementation does not silently change compression, output, or layout
  semantics to claim the target.

## wkhtmltopdf reference

The earlier process-level reference on the same fixture recorded wkhtmltopdf
0.12.6.1 at **2.05 s / approximately 114 MB peak RSS / approximately 2.0 MB
PDF** for 500 pages. The current Go benchmark records **1.903 s / 678.6 MB
B/op / 1.103 M allocs** in-process, with a profiled process RSS of about
**391 MiB**. `B/op` is cumulative allocation traffic, not RSS, so these are
not interchangeable memory figures. The full page matrix and caveat are in
[`testdata/golden/benchmarks/README.md`](../../../../testdata/golden/benchmarks/README.md).
