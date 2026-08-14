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

- [ ] `layout.Options` no longer names `*pdf.Font`, `*pdf.FaceSet`, or `*pdf.Registry` — `internal/layout/layout.go`
- [ ] Layout asks for glyph widths / face resolve through a small interface or a layout-owned handle
- [ ] `internal/pdf` remains the owner of TTF parse, subset, Type0, and embedding
- [ ] Compile-time check: `var _ layout.Face = (*pdf.Font)(nil)` or equivalent
- [ ] Test: existing `internal/layout` font/face tests still pass without importing `pdf` from production layout files (tests may import `pdf` to build a handle)

### 27.2 Paint backends

- [ ] Inventory every `Op` field that PDF uses and every field imageout uses — `internal/layout/layout.go` `Op`
- [ ] Fields used by only one sink are documented on the struct or split
- [ ] Image mode and PDF share `PaintOrder` (already) and do not fork transform / z-index / sticky interpretation
- [ ] Test: one fixture (or `architecture_renderer_test.go`) asserts PDF page count and image mode ink bounds stay consistent for the same HTML
- [ ] `[~]` Full visitor / tagged union for `Op` — only if 27.2 inventory shows the struct is blocking a test

### 27.3 Convert façades

- [ ] `convert.ResourceContext` / `SheetOptions` / `PrepareOptions` / `PreparedDocument` aliases: either delete and update call sites, or keep and stop calling them a “focused module” in `documentation/architecture.md`
- [ ] `convert.pagePlan` vs `render.Plan`: one type owns copy/page counts; the other is a thin wrapper or is deleted — `internal/convert/page_plan.go`, `internal/convert/render/plan.go`
- [ ] `maxCopies` exists in one package
- [ ] `render.Pipeline.Assemble` no-op in image mode is documented as intentional **or** image mode stops implementing `Pipeline`

### 27.4 Page islands stay off the product path

- [ ] `benchmarkPageIslands` remains false for `NewPDFRequest` / CLI / `RunPDF` — `internal/convert/convert.go`
- [ ] Test: `TestBenchmarkPageIslandsOffByDefault` (or the existing check in `page_islands_test.go`) stays
- [ ] No CLI flag, setting key, or library option turns islands on
- [ ] Architecture docs already say islands are certified-benchmark only; keep that sentence accurate

### 27.5 Package comment hygiene (layout/pdf only)

- [ ] `internal/pdf/doc.go` describes the current PDF 1.4 writer (not “Phase 00 scaffold”)
- [ ] `internal/layout/doc.go` still says lite/print subset; do not claim a fragment tree unless Phase 25 built one

### 27.6 Closure gates

- [ ] `go list -deps ./internal/layout` does not include `gowkhtmltopdf/internal/pdf` **or** the remaining import is listed here with a reason
- [ ] `make lint` →
- [ ] `make test` →
- [ ] Parent Phase 27 row checked
- [ ] Next: Phase 28 if not already in flight

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
