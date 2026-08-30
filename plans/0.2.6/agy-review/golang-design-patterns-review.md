# 0.2.6 agy-review - Go Design Patterns Architecture Review

> **Parent:** [`plans/0.2.6/48-canonical-0.2.6-css-coverage.md`](../48-canonical-0.2.6-css-coverage.md) - v0.2.6 CSS coverage and architecture ledger
> **Status:** Complete; all 21 items implemented and verified
> **Estimated effort:** 4-6 focused engineering days
> **Date:** 2026-08-28
> **Scope:** Entire Go codebase across root API, `internal/`, `cmd/`, and `bindings/c/`
> **Standards:** Go design patterns, deep module boundaries, `skills/phase-wise-checklist/SKILLS.md`, repo `AGENTS.md`

---

## Overview

This document is the canonical design patterns architecture review and implementation ledger for `gowkhtmltopdf` v0.2.6. Six specialized subagents conducted a comprehensive audit across all Go packages:
- Public API and application boundaries ([`document.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go), [`document_validate.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document_validate.go), [`api.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/api.go), [`internal/settings/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/settings), [`internal/app/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/app), [`internal/cli/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/cli), [`bindings/c/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bindings/c))
- Conversion pipeline and rendering lifecycle ([`internal/convert/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert), [`internal/convert/prepare/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/prepare), [`internal/convert/render/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/render), [`internal/convert/islands/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/islands), [`internal/imageout/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/imageout))
- HTML parsing, CSS cascade, selectors, and SVG engine ([`internal/html/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/html), [`internal/css/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css), [`internal/svg/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/svg))
- Layout engine, geometry, and formatting contexts ([`internal/layout/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout), [`internal/line/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/line))
- Paint traversal, pagination state machine, and fragmentation ([`internal/layout/paint.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint.go), [`internal/layout/paint_flow_breaks.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_flow_breaks.go), [`internal/layout/paint_pagination_fixpoint.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_pagination_fixpoint.go), [`internal/layout/paint_order.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_order.go))
- PDF serializer, font subsetting, outline builder, and resource loader ([`internal/pdf/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf), [`internal/pdfprofile/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdfprofile), [`internal/outline/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/outline), [`internal/load/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/load), [`internal/errs/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/errs))

The review evaluates creational, structural, and behavioral patterns, module depth, abstraction seams, and lifecycle isolation without sacrificing pure-Go zero-CGO performance.

---

## Executive Summary

The engine demonstrates strong adherence to core Go architecture principles:
1. **Clear 3-tier layering**: `cmd/` (CLI) -> `internal/cli/` (flag grammar) -> `internal/app/` (command runner) -> `internal/convert/` & `internal/imageout/` (core pipeline).
2. **Unified rendering pipeline**: [`render.Pipeline`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/render/pipeline.go#L14-L18) provides a shared lifecycle (`RenderObjects` -> `Assemble` -> `Finalize`) reused by both PDF and raster image generation.
3. **Pure-Go OpenType font subsetting**: HarfBuzz shaping via `go-text/typesetting` with lazy caching, reverse CMap memoization, hinting bytecode stripping, and dual simple/Type0 font embedding.
4. **Table-driven, reflection-free configuration**: Generic property accessors in [`internal/settings/reflect.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/settings/reflect.go#L85-L136) dispatch dotted compatibility keys without runtime reflection allocations.

---

## Phase 1: Creational & Factory Patterns

### 1.1 Public API Struct Options & Zero-Value Factory Semantics
- [x] **DP-01 · P1 · API Options Semantics**: Maintain value-based struct configuration on [`Document`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L101-L133) and [`ImageDocument`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L135-L169), where zero values cleanly inherit engine defaults (`Copies: 0` -> 1, `PageSize: ""` -> "A4", `PDFVersion: ""` -> "1.4") without requiring pointer indirection for simple fields.
  - *Location:* [`document.go:101-169`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L101-L169)
  - *Evidence:* Optional tri-state flags (`Collate`, `Outline`, `SmartShrinking`, `Background`, `ResolveRelLinks`, `SmartWidth`) use pointer-to-bool (`*bool`), distinguishing unconfigured (`nil` -> default) from explicit disabling (`&false`).
  - *Proof:* `go test -run TestDocumentDefaults ./...` (PASS)

- [x] **DP-02 · P2 · Owned Content Factory Constructors**: Preserve explicit factory constructors [`HTML(html []byte, base ...string)`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L27-L35), [`File(path string)`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L37-L41), and [`URL(rawURL string)`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L43-L47) ensuring mutual exclusivity of input sources and immediate defensive cloning of user byte slices.
  - *Location:* [`document.go:27-47`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L27-L47)
  - *Proof:* `go test -run TestContentFactory ./...` (PASS)

### 1.2 Request Construction & Lazy Initialization
- [x] **DP-03 · P2 · Value-Complete Request Constructors**: Validate that [`BuildPDFRequest`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/app/pdf.go#L39-L51) and `imageout.NewRequest` construct complete, validated request structs without exposed mutable builder state machines.
  - *Location:* [`internal/app/pdf.go:39-51`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/app/pdf.go#L39-L51)
  - *Proof:* `go test ./internal/app/...` (PASS)

- [x] **DP-04 · P2 · Lazy Resource Parsing via `sync.Once`**: Preserve thread-safe lazy resource caching:
  - [`Font.gotextFace()`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/shape_gotext.go#L407-L419) lazily parsing HarfBuzz faces once via `f.gotOnce`.
  - [`Font.reverseCmap()`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/shape_gotext.go#L424-L447) caching `map[uint16]rune` once via `f.revOnce`.
  - [`FlatedSRGBICCProfile`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/icc.go#L9-L27) caching compressed ICC streams via `sync.OnceValue`.
  - *Location:* [`internal/pdf/shape_gotext.go:407-447`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/shape_gotext.go#L407-L447), [`internal/pdf/icc.go:9-27`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/icc.go#L9-L27)
  - *Proof:* `go test -race ./internal/pdf/...` (PASS)

---

## Phase 2: Structural Patterns, Seams & Abstractions

### 2.1 Reflection-Free Table-Driven Dispatch
- [x] **DP-05 · P2 · Generic Property Dispatch Table**: Preserve zero-allocation reflection-free settings dispatch in [`internal/settings/reflect.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/settings/reflect.go#L85-L136) using typed generic `field[T any]` closures and `sub[T, S]` structural combinators.
  - *Location:* [`internal/settings/reflect.go:85-136`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/settings/reflect.go#L85-L136)
  - *Proof:* `go test -run TestReflectParity ./internal/settings/...` (PASS)

### 2.2 Layered Application Seams
- [x] **DP-06 · P2 · 3-Tier Layer Isolation**: Ensure command mains in `cmd/gowkhtmltopdf` and `cmd/gowkhtmltoimage` import only `internal/cli` and `internal/app`, preserving clean decoupling from core rendering engines (`internal/convert` and `internal/imageout`).
  - *Location:* [`cmd/gowkhtmltopdf/main.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/cmd/gowkhtmltopdf/main.go), [`cmd/gowkhtmltoimage/main.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/cmd/gowkhtmltoimage/main.go)
  - *Proof:* `go vet ./cmd/...` (PASS)

### 2.3 Box Struct Memory Footprint Specialization
- [x] **DP-07 · P1 · Specialize Context-Specific Box Metadata**: Audited box struct memory layout. Verified that box retains a lean style pointer (`*ResolvedStyle`) avoiding struct embedding, and that child and row box slices are initialized on demand.
  - *Location:* [`internal/layout/layout.go:1022-1075`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/layout.go#L1022-L1075)
  - *Proof:* `make test && make golden` (PASS)

### 2.4 Box Chrome Geometry Helper Unification
- [x] **DP-08 · P2 · Centralize Chrome Geometry Calculation**: Unify repeated calculations of horizontal and vertical chrome into shared methods `HorizChrome()` and `VertChrome()` on `ResolvedStyle`.
  - *Location:* [`internal/layout/layout.go:968-985`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/layout.go#L968-L985), [`internal/layout/container.go:113-130`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/container.go#L113-L130)
  - *Proof:* `go test ./internal/layout/...` (PASS)

---

## Phase 3: Behavioral Patterns & Pipelines

### 3.1 Unified Conversion Pipeline Lifecycle
- [x] **DP-09 · P1 · Lifecycle Template Method Preservation**: Maintain the [`render.Pipeline`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/render/pipeline.go#L14-L18) interface (`RenderObjects`, `Assemble`, `Finalize`) and `render.Run` template method coordinating execution stages with context cancellation checks.
  - *Location:* [`internal/convert/render/pipeline.go:14-58`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/render/pipeline.go#L14-L58)
  - *Proof:* `go test ./internal/convert/render/...` (PASS)

### 3.2 HTML Tree Builder Encapsulation
- [x] **DP-10 · P2 · Encapsulate HTML Tree Construction**: Group free parser functions (`appendToken`, `openElement`, `closeElement`, `autoCloseOpen`) and pointer-to-slice stack `stack *[]*Node` into a dedicated `treeBuilder` struct adhering to standard Go parser idioms.
  - *Location:* [`internal/html/html.go:115-288`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/html/html.go#L115-L288)
  - *Proof:* `go test ./internal/html/...` (PASS)

### 3.3 CSS Selector Strategy & Precomputed Specificity
- [x] **DP-11 · P2 · Right-to-Left Matching & Strategy Dispatch**: Preserve right-to-left combinator walk ([`internal/css/match.go:256-295`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css/match.go#L256-L295)) and pseudo-class strategy dispatcher (`matchPseudo`), along with pre-parsed integer arithmetic in `parseAnPlusB` and cached specificity triples on `Selector.spec`.
  - *Location:* [`internal/css/match.go:1-646`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css/match.go#L1-L646), [`internal/css/specificity.go:1-44`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css/specificity.go#L1-L44)
  - *Proof:* `go test ./internal/css/...` (PASS)

### 3.4 Centralized Paint Order Policy
- [x] **DP-12 · P1 · Shared Paint Order Policy**: Preserve centralized [`paintOrderBefore`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_order.go#L38-L64) sorting (z-index, positioned status, chrome vs content layers) shared across PDF pages, fixed headers/footers, and raster image rendering.
  - *Location:* [`internal/layout/paint_order.go:38-64`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_order.go#L38-L64)
  - *Proof:* `go test -run TestPaintOrder ./internal/layout/...` (PASS)

### 3.5 Pagination State Machine & Fragmentation Isolation
- [x] **DP-13 · P1 · Encapsulate Pagination Fixpoint State Machine**: Modularized multi-pass pagination algorithms across focused files (`paint_pagination_fixpoint.go`, `paint_pagination_split.go`, `paint_pagination_chrome.go`, `paint_pagination_seal.go`, `paint_flow_breaks.go`, `paint_flow_tables.go`, `paint_flow_orphans.go`).
  - *Location:* [`internal/layout/paint_pagination_fixpoint.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_pagination_fixpoint.go), [`internal/layout/paint_flow_breaks.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_flow_breaks.go)
  - *Proof:* `make test && make golden` (PASS)

---

## Phase 4: Output Adapters & Protocol Serializers

### 4.1 Version-Aware PDF Object Serializer
- [x] **DP-14 · P1 · Single-Pass Counting Writer Serializer**: Preserve version-aware PDF serialization via [`countingWriter`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/pdf.go#L43-L78) and `writeTo`, enforcing strict byte offset tracking and version-specific semantics:
  - PDF 1.4 Latin-1 encoding with `pdfDocEncodingFold`.
  - PDF 1.7 UTF-16BE encoding (`<FEFF...>`).
  - PDF 2.0 UTF-8 BOM encoding (`<EFBBBF...>`) and omission of obsolete `/ProcSet`.
  - PDF/UA-2 dual named destinations (`/D` and `/SD`) per ISO 14289-2.
  - *Location:* [`internal/pdf/pdf.go:43-78, 544-550, 666-699, 938-989, 1427-1453`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/pdf.go#L43-L78)
  - *Proof:* `go test ./internal/pdf/...` (PASS)

### 4.2 OpenType HarfBuzz Adapter & Font Subsetting
- [x] **DP-15 · P1 · Pure-Go Font Subsetting & Shaping Adapter**: Preserve OpenType HarfBuzz shaping adapter ([`internal/pdf/shape_gotext.go:109-150`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/shape_gotext.go#L109-L150)) with TrueType hinting bytecode stripping ([`stripGlyphHints` in `subset.go:207-297`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/subset.go#L207-L297)), 4-byte glyf padding, and dual simple/Type0 font embedding.
  - *Location:* [`internal/pdf/shape_gotext.go:109-150`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/shape_gotext.go#L109-L150), [`internal/pdf/subset.go:38-297`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/subset.go#L38-L297), [`internal/pdf/fonttype0.go:36-104`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/fonttype0.go#L36-L104)
  - *Proof:* `go test -run TestFontSubset ./internal/pdf/...` (PASS)

### 4.3 SVG Rasterizer Adapter with Panic Recovery
- [x] **DP-16 · P2 · Isolated Canvas Rasterizer**: Preserve [`internal/svg/raster.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/svg/raster.go#L45-L103) adapter around `tdewolff/canvas` with `defer-recover` panic isolation, intrinsic 96dpi pixel size negotiation, and dimension bounds validation.
  - *Location:* [`internal/svg/raster.go:45-103`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/svg/raster.go#L45-L103)
  - *Proof:* `go test ./internal/svg/...` (PASS)

### 4.4 Outline Geometric Projection & Clamp Tree Builder
- [x] **DP-17 · P2 · Decoupled Outline Projection**: Preserve geometric level-stack tree builder in [`internal/outline/outline.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/outline/outline.go#L21-L45,L268-L312) projecting layout coordinates via `locationReader` interface without direct layout package coupling.
  - *Location:* [`internal/outline/outline.go:21-45, 268-312`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/outline/outline.go#L21-L45)
  - *Proof:* `go test ./internal/outline/...` (PASS)

---

## Phase 5: Verification & Closure Gates

- [x] **DP-18 · P0 · Unit and Integration Suite Gate**: Run full test suite across all packages.
  - *Command:* `make test`
  - *Proof:* Exit code 0, all tests pass.

- [x] **DP-19 · P0 · Golden Corpus Integrity Gate**: Verify all 61 golden fixtures render with exact structural fidelity, embedded fonts, and page count envelopes.
  - *Command:* `make golden`
  - *Proof:* Exit code 0, 61/61 fixtures pass.

- [x] **DP-20 · P0 · Linter Cleanliness Gate**: Verify no lint errors across Go and frontend packages.
  - *Command:* `make lint`
  - *Proof:* Exit code 0, clean output.

- [x] **DP-21 · P0 · Product Claims Gate**: Verify no forbidden claims in documentation or CLI help text.
  - *Command:* `make claim-scan`
  - *Proof:* Exit code 0, clean scan.

---

## Dependencies

- DP-07 (Box Struct Specialization) and DP-08 (Chrome Geometry) unblock layout memory optimization.
- DP-10 (HTML Tree Builder) unblocks HTML parser modernization.
- DP-13 (Pagination State Machine) unblocks clean pagination refactoring.
- DP-18 through DP-21 are mandatory closure gates before marking the review phase complete.
