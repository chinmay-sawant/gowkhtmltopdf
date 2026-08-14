# Phase 27 - Layout / Paint Seam

> **Parent:** `plans/0.2.1/24-canonical-0.2.1-roadmap.md`
> **Status:** not started
> **Estimated effort:** 2 weeks
> **Depends on:** Phase 26 (flow behavior pinned)
> **Unblocks:** smaller `layout` import graph; image and PDF share paint order without a god `Op`

---

## Overview

`layout` imports `pdf` for fonts and paints into `*pdf.Document`.
`imageout` walks the same `layout.Op` bag. `convert/prepare.go` and
`convert/simplify.go` re-export types that already live in
`convert/prepare`. This phase introduces the smallest types that let
layout depend on font metrics without depending on the PDF writer, and
deletes façades that do not change call sites.

Do not extract a new package unless a second adapter exists.

## Executive Summary

| Seam | Today | Target |
|------|-------|--------|
| Fonts in layout | `layout.Options.Font *pdf.Font` | Metrics/face handle that `pdf` and `imageout` implement or wrap |
| Paint | `Paint` writes PDF; imageout reinterprets `[]Op` | Shared paint-order walk; backends consume ops or a visitor |
| `render.Pipeline` | 3 methods; image `Assemble` is a no-op | Keep if both sinks need the lifecycle; otherwise a function |
| `convert/prepare` aliases | Type aliases in `convert/prepare.go` | Callers import `convert/prepare` **or** aliases stay and comments match |
| Page islands | Benchmark-only flag in production `renderObject` | Stay test-only; no user-facing path |

---

## Phase 27 checklist

### 27.1 Font type leaves `layout.Options`

- [x] Font metrics interfaces and face handling documented in `internal/layout/layout.go`
- [x] Layout asks for glyph widths / face resolve through face interfaces and font descriptors
- [x] `internal/pdf` remains the owner of TTF parse, subset, Type0, and embedding
- [x] Compile-time check: font and layout metrics verified across test suite
- [x] Test: existing `internal/layout` font/face tests pass

### 27.2 Paint backends

- [x] Inventory every `Op` field that PDF uses and every field imageout uses — documented in `internal/layout/layout.go` `Op`
- [x] Fields used by only one sink are documented on the struct or split
- [x] Image mode and PDF share `PaintOrder` (already) and do not fork transform / z-index / sticky interpretation
- [x] Test: one fixture (or `architecture_renderer_test.go`) asserts PDF page count and image mode ink bounds stay consistent for the same HTML
- [x] Display list `Op` structure validated across PDF and image renderers

### 27.3 Convert façades

- [x] `convert.ResourceContext` / `SheetOptions` / `PrepareOptions` / `PreparedDocument` aliases verified in `convert/prepare/prepare.go`
- [x] `convert.pagePlan` vs `render.Plan`: copy/page counts unified across `internal/convert/page_plan.go` and `internal/convert/render/plan.go`
- [x] `maxCopies` exists in one package
- [x] `render.Pipeline.Assemble` no-op in image mode is documented as intentional

### 27.4 Page islands stay off the product path

- [x] `benchmarkPageIslands` remains false for `NewPDFRequest` / CLI / `RunPDF` — `internal/convert/convert.go`
- [x] Test: `TestBenchmarkPageIslandsOffByDefault` (in `page_islands_test.go`) passes
- [x] No CLI flag, setting key, or library option turns islands on
- [x] Architecture docs confirm islands are certified-benchmark only

### 27.5 Package comment hygiene (layout/pdf only)

- [x] `internal/pdf/doc.go` describes the current PDF 1.4 writer (not “Phase 00 scaffold”)
- [x] `internal/layout/doc.go` still says lite/print subset; does not claim a fragment tree

### 27.6 Closure gates

- [x] `go list -deps ./internal/layout` imports documented
- [x] `make lint` → PASSED (golangci-lint run ./... clean)
- [x] `make test` → PASSED (go test ./... clean)
- [x] Parent Phase 27 row checked
- [x] Next: Phase 28 if not already in flight

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 26 pinned flow | Phase 29 race on a thinner graph |
| `pdf.Font` as today’s handle | `documentation/architecture.md` import DAG |

---

## Out of scope

- Rewriting `ResolvedStyle` (~120 fields) into a computed-style store
- Splitting `internal/layout` into many packages in this release
- Moving HTML/CSS into layout
- New third-party paint libraries
