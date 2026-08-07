# Phase 4 — Output resources, paint traversal, and fonts

> **Parent:** [`../architecture-review-2026-08-07-followup.md`](../architecture-review-2026-08-07-followup.md)
> **Status:** complete
> **Depends on:** Phase 3

## Goal

Use the real PDF/raster adapters to deepen resource ownership and rendering
policy while preserving pagination, annotations, raster alpha, and font
compatibility.

## Checklist

- [x] **PDF-01** — page-local image names allocate collision-safe suffixes and
  cloned pages deep-copy resource maps while immutable font data remains shared.
  Body/header/footer image and duplicate-page resource tests pass.
- [x] **PAINT-01** — PDF body and HTML header/footer use the shared layout
  dispatch/traversal policy; raster uses the same z/layer ordering and semantic
  style table while retaining destination-specific alpha, pagination, and
  annotations. Transformed/stacked/transparent coverage and existing goldens
  pass.
- [x] **FONT-01** — face-byte fingerprints provide stable identity independent
  of display name, and `ShapeRun` supplies paired shaped runes/advances to PDF
  and raster. Same-family, Arabic, CJK, Type0/ToUnicode, and shape tests pass;
  `BenchmarkShapeRun` captured `1313 ns/op`, `528 B/op`, `2 allocs/op`.

## Required gate

- [x] Final Phase 4 gate: `make lint`, `make test`, `go test -race ./...`,
  `go test ./internal/pdf ./internal/imageout ./internal/svg`, and the PDF
  benchmarks passed. PDF `BenchmarkWrite50Pages` captured `24,287,371 ns/op`,
  `1.43 MB/s`, `46,766,645 B/op`, and `23,914 allocs/op`.
- [x] Release/debug CLI smoke benchmark recorded separately with fixture
  `testdata/golden/fixture-16-invoice-with-css.html`, Go 1.26.4, Linux WSL2,
  `GOMAXPROCS=1`, 20 copies, and separate cold/warm runs: debug `0.03s/0.04s`,
  stripped release `0.03s/0.04s`. These are directional timings only.
