# Performance Executive Summary

**Date:** 2026-08-07  
**Branch:** `feature/performance-improvements`  
**Implementation commit:** `c103141`  
**Final rating:** **10.0/10**

## Executive summary

The current version is substantially better than the earlier baseline. The
comparison uses the landed one-iteration benchmark snapshot against the final
count-3 medians on the same local benchmark environment.

| Workload | Earlier | Current median | Improvement |
|---|---:|---:|---:|
| PDF, 500 pages | 14.14 s | 6.93 s | **50.96% faster** |
| Template PDF, 500 pages | 13.67 s | 5.82 s | **57.44% faster** |
| Web images, 500 | 970.72 ms | 196.26 ms | **79.78% faster** |
| Inline images, 500 | 788.43 ms | 229.25 ms | **70.92% faster** |

## Allocation improvements

- PDF memory: **49.72% less**
- Template PDF memory: **49.66% less**
- Image memory: **54.91% less**
- PDF allocations: approximately **60.8% fewer**
- Image allocations: approximately **99.7% fewer**

## Main improvements

- Replaced repeated pagination-wide scans with page-indexed operation and box
  updates.
- Optimized forced page breaks and table-border candidate searches.
- Reduced text, style, PDF formatting, and compression allocations.
- Added direct NRGBA image paths, image caching, and precomputed glyph edges.
- Preserved page counts, PDF output, image pixels, and existing test behavior.

The original top hotspot, `stripOrphanRowChrome`, consumed **29.73% flat CPU**
and is no longer among the dominant profile nodes. The remaining `shiftFlowY`
cost is indexed movement work rather than repeated full-list/tree scanning.

## Validation

The final `go test ./...`, `make lint`, focused profiling, count-3 benchmark
matrix, and `git diff --check` all passed. The changes were committed as
`c103141` and pushed to `feature/performance-improvements`.

Detailed phase-wise evidence is available in
[`performance-profile-and-improvement-plan.md`](performance-profile-and-improvement-plan.md).
