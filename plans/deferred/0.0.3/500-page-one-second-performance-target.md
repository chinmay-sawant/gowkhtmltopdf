# Deferred 0.0.3: 500-Page PDF in One Second

**Status:** Deferred investigation  
**Created:** 2026-08-07  
**Current branch:** `feature/performance-improvements`  
**Baseline implementation commit:** `c103141`

## Objective

Investigate whether the current Go PDF pipeline can generate the 500-page PDF
benchmark within approximately one second on the reference benchmark machine.

This is a separate high-performance architecture target, not a continuation of
the completed incremental optimization checklist.

## Current position

The final count-3 benchmark medians are:

| Workload | Current median | Target | Further reduction required |
|---|---:|---:|---:|
| PDF, 500 pages | 6.93 s | 1.00 s | **85.6%** |
| Template PDF, 500 pages | 5.82 s | 1.00 s | **82.8%** |

The completed optimization work reduced the original PDF baseline from 14.14 s
to 6.93 s, approximately **51% faster**. The current implementation is better,
but the one-second target requires a materially different level of optimization.

## Why incremental fixes are insufficient

The final PDF profile still contains several substantial costs:

- `shiftFlowY`: **15.88% flat CPU**
- Indexed operation movement: **10.06% flat CPU**
- Indexed box movement: **9.30% flat CPU**
- Garbage collection via `runtime.scanObject`: **16.26% flat CPU**
- Pagination overall: approximately **45% cumulative CPU**

These percentages overlap by call path. Removing one hotspot completely would
not be enough to reach one second. The target requires reducing or restructuring
multiple serial stages.

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

