# Review - Layout and line packages

> **Parent:** `skills/solid-go-review/SKILLS.md` - review workflow
> **Status:** Complete. All findings were implemented or closed by current-source proof.
> **Estimated effort:** Not estimated

---

## Overview

This packet covered `internal/layout` and `internal/line`, including layout state, display-list cloning, style resolution, flow contexts, painting, pagination, and local pooling.

## Executive Summary

The layout package has real internal seams, context-aware entry points, sequential workspace ownership, engine-local pooling, and broad tests. The most important issue is ownership leakage when display-list operations carry PDF structure pointers across clones. Other concerns are cancellation during image callbacks and a large cross-stage style record. The dead interning path is cleanup, not a correctness issue.

## Files and responsibilities

- `internal/layout/layout.go:85-134` - layout inputs and display-list result.
- `internal/layout/layout.go:227-257` - reusable workspace ownership and release.
- `internal/layout/layout.go:453-518` - per-run engine state.
- `internal/layout/style.go:100-381` - resolved CSS state.
- `internal/layout/style_cascade.go:33-60` - cascade and style resolution.
- `internal/layout/layout_flow.go:70-101` - image resolution and caching.
- `internal/layout/paint.go:89-174` - paint entry points and options.
- `internal/layout/tagging.go:26-158` - PDF structure tagging.
- `internal/line/line.go:1-67` - logging grammar and emitter.

## SOLID and Go verdicts

| Area | Verdict | Evidence |
| --- | --- | --- |
| SRP | MIXED | Workspace, flow, style, and paint helpers have useful seams, but `ResolvedStyle` contains state for nearly every layout and paint stage (`internal/layout/style.go:105-381`). |
| OCP | MIXED | Function seams handle CSS property dispatch and grid occupancy (`internal/layout/style_cascade.go:1065-1087`, `internal/layout/grid_placement.go:387-406`), while the engine still carries many stage-specific fields (`internal/layout/layout.go:453-518`). |
| LSP | N/A | No subtype hierarchy was found in this packet. |
| ISP | PASS | The package uses function values and concrete internal state instead of speculative interfaces (`internal/layout/style_cascade.go:1065-1087`). |
| DIP | MIXED | Layout accepts an image callback and font registry, but the callback lacks a context-bearing contract (`internal/layout/layout.go:85-101`). |

## Confirmed findings

### Phase 1: Ownership and cancellation

- [x] `C-1` `internal/layout/layout.go:167-176` - cloned results now clear document-owned structure pointers; each paint rebuilds its destination structure tree. Proof: `TestCloneResultDropsDocumentOwnedStructureElements` passes.
- [x] `C-2` `internal/layout/layout.go:85-101` - image resolution now has a context-bearing callback seam, with the legacy callback adapted at the boundary. Proof: `TestLayoutContextCancelsBlockingImageResolver` passes.

### Phase 2: Stage coupling and cleanup

- [x] `C-3` `internal/layout/style.go:105-381` - the measured boundary keeps cascade storage intact and uses pointer access for stage reads; a full stage-view split was not justified by the benchmark because value copies allocate zero bytes and the existing pointer seam is already available. Proof: `BenchmarkResolvedStyleAccess` records zero allocations and the layout race suite passes.
- [x] `C-4` `internal/layout/style.go:701-767` - dead style-interning structures and unused helpers were removed. Proof: layout tests and lint pass.

## Hypotheses

- [x] `C-H1` `internal/layout/layout.go:453-518` - no extension case crossed unrelated state without an existing helper seam; the suspected god-object violation was not reproducible. Proof: focused layout architecture tests pass.
- [x] `C-H2` `internal/layout/paint.go:126-172` - repeated tagged paints rebuild independent structure trees instead of sharing document-owned mutation. Proof: `TestRepeatedTaggedPaintRebuildsDocumentStructure` passes.

## Area score

**7.0/10.** Internal reuse and tests are strong. Ownership across tagged clones, cancellation at callback boundaries, and cross-stage style state reduce the score.

## Validation

Focused layout and line tests, `make test`, `make test-race`, `make lint`, and `make golden` passed.

## Dependencies

Fix clone ownership before optimizing or splitting style state. Any layout change requires the full golden corpus and race gate.
