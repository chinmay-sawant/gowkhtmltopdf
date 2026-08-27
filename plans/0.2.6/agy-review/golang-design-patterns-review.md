# 0.2.6 agy-review - Go Design Patterns Architecture Review

> **Parent:** [`plans/0.2.6/48-canonical-0.2.6-css-coverage.md`](../48-canonical-0.2.6-css-coverage.md) - v0.2.6 CSS coverage and architecture ledger
> **Status:** Review and ledger creation complete; implementation pending
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
- Paint traversal, pagination state machine, and fragmentation ([`internal/layout/paint.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint.go), [`internal/layout/paint_flow.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_flow.go), [`internal/layout/paint_pagination.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_pagination.go), [`internal/layout/paint_order.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_order.go))
- PDF serializer, font subsetting, outline builder, and resource loader ([`internal/pdf/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf), [`internal/pdfprofile/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdfprofile), [`internal/outline/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/outline), [`internal/load/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/load), [`internal/errs/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/errs))

The review evaluates creational, structural, and behavioral patterns, module depth, abstraction seams, and lifecycle isolation without sacrificing pure-Go zero-CGO performance.

---

## Executive Summary

The engine demonstrates strong adherence to core Go architecture principles:
1. **Clear 3-tier layering**: `cmd/` (CLI) -> `internal/cli/` (flag grammar) -> `internal/app/` (command runner) -> `internal/convert/` & `internal/imageout/` (core pipeline).
2. **Unified rendering pipeline**: [`render.Pipeline`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/render/pipeline.go#L14-L18) provides a shared lifecycle (`RenderObjects` -> `Assemble` -> `Finalize`) reused by both PDF and raster image generation.
3. **Pure-Go OpenType font subsetting**: HarfBuzz shaping via `go-text/typesetting` with lazy caching, reverse CMap memoization, hinting bytecode stripping, and dual simple/Type0 font embedding.
4. **Table-driven, reflection-free configuration**: Generic property accessors in [`internal/settings/reflect.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/settings/reflect.go#L85-L136) dispatch dotted compatibility keys without runtime reflection allocations.

Key architecture findings and refactoring opportunities include:
- **HTML Tree Builder Encapsulation**: Encapsulating free-floating parser functions and pointer-to-slice stacks into a cohesive AST builder.
- **Layout Box Context Specialization**: Extracting table and sticky metadata from the monolithic 40-field [`box`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/layout.go#L1022-L1075) struct into dedicated sub-structures.
- **Pagination State Machine Encapsulation**: Isolating destructive pagination mutations from immutable layout results.
- **Chrome Geometry Unification**: Centralizing repetitive box chrome calculations into shared helper methods.

### Architecture Scorecard

| Dimension | Score (1-10) | Evaluation |
|---|---:|---|
| **Pipeline & Lifecycle Decoupling** | 9.2 | Shared `render.Pipeline` lifecycle across PDF and image backends; prepare/render immutability. |
| **Creational & Factory Patterns** | 9.0 | Value-based options, zero-value defaults, owned content constructors, and lazy `sync.Once` caches. |
| **Structural Seams & Modularity** | 8.6 | Strong 3-tier CLI architecture; reflection-free settings dispatch; opportunity to specialize fat `box` struct. |
| **Behavioral Patterns & Matching** | 8.8 | Modular right-to-left CSS selector strategies; centralized `PaintOrder` stacking policy. |
| **Output Serialization & Adapters** | 9.4 | Single-pass streaming PDF serializer; clean OpenType and Canvas adapters; robust SSRF security facade. |
| **Overall Design Patterns Rating** | **9.0 / 10** | High-discipline pure-Go architecture with well-defined seams and zero CGO dependencies. |

---

## Phase 1: Creational & Factory Patterns

### 1.1 Public API Struct Options & Zero-Value Factory Semantics
- [ ] **DP-01 · P1 · API Options Semantics**: Maintain value-based struct configuration on [`Document`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L101-L133) and [`ImageDocument`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L135-L169), where zero values cleanly inherit engine defaults (`Copies: 0` -> 1, `PageSize: ""` -> "A4", `PDFVersion: ""` -> "1.4") without requiring pointer indirection for simple fields.
  - *Location:* [`document.go:101-169`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L101-L169)
  - *Evidence:* Optional tri-state flags (`Collate`, `Outline`, `SmartShrinking`, `Background`, `ResolveRelLinks`, `SmartWidth`) use pointer-to-bool (`*bool`), distinguishing unconfigured (`nil` -> default) from explicit disabling (`&false`).
  - *Proof:* `go test -run TestDocumentDefaults ./...`

- [ ] **DP-02 · P2 · Owned Content Factory Constructors**: Preserve explicit factory constructors [`HTML(html []byte, base ...string)`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L27-L35), [`File(path string)`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L37-L41), and [`URL(rawURL string)`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L43-L47) ensuring mutual exclusivity of input sources and immediate defensive cloning of user byte slices.
  - *Location:* [`document.go:27-47`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L27-L47)
  - *Proof:* `go test -run TestContentFactory ./...`

### 1.2 Request Construction & Lazy Initialization
- [ ] **DP-03 · P2 · Value-Complete Request Constructors**: Validate that [`BuildPDFRequest`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/app/pdf.go#L39-L51) and `imageout.NewRequest` construct complete, validated request structs without exposed mutable builder state machines.
  - *Location:* [`internal/app/pdf.go:39-51`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/app/pdf.go#L39-L51)
  - *Proof:* `go test ./internal/app/...`

- [ ] **DP-04 · P2 · Lazy Resource Parsing via `sync.Once`**: Preserve thread-safe lazy resource caching:
  - [`Font.gotextFace()`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/shape_gotext.go#L407-L419) lazily parsing HarfBuzz faces once via `f.gotOnce`.
  - [`Font.reverseCmap()`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/shape_gotext.go#L424-L447) caching `map[uint16]rune` once via `f.revOnce`.
  - [`FlatedSRGBICCProfile`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/icc.go#L9-L27) caching compressed ICC streams via `sync.OnceValue`.
  - *Location:* [`internal/pdf/shape_gotext.go:407-447`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/shape_gotext.go#L407-L447), [`internal/pdf/icc.go:9-27`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/icc.go#L9-L27)
  - *Proof:* `go test -race ./internal/pdf/...`

---

## Phase 2: Structural Patterns, Seams & Abstractions

### 2.1 Reflection-Free Table-Driven Dispatch
- [ ] **DP-05 · P2 · Generic Property Dispatch Table**: Preserve zero-allocation reflection-free settings dispatch in [`internal/settings/reflect.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/settings/reflect.go#L85-L136) using typed generic `field[T any]` closures and `sub[T, S]` structural combinators.
  - *Location:* [`internal/settings/reflect.go:85-136`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/settings/reflect.go#L85-L136)
  - *Proof:* `go test -run TestReflectParity ./internal/settings/...`

### 2.2 Layered Application Seams
- [ ] **DP-06 · P2 · 3-Tier Layer Isolation**: Ensure command mains in `cmd/gowkhtmltopdf` and `cmd/gowkhtmltoimage` import only `internal/cli` and `internal/app`, preserving clean decoupling from core rendering engines (`internal/convert` and `internal/imageout`).
  - *Location:* [`cmd/gowkhtmltopdf/main.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/cmd/gowkhtmltopdf/main.go), [`cmd/gowkhtmltoimage/main.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/cmd/gowkhtmltoimage/main.go)
  - *Proof:* `go vet ./cmd/...`

### 2.3 Box Struct Memory Footprint Specialization
- [ ] **DP-07 · P1 · Specialize Context-Specific Box Metadata**: Refactor monolithic [`box`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/layout.go#L1022-L1075) struct by moving 11 table-specific fields (`col, span, row, rowSpan, rows, headerRows...`) and 12 sticky-specific fields (`sticky, stickyID, stickyTop, stickyPort...`) into dedicated pointer structs (`tableBoxMeta`, `stickyBoxMeta`) allocated only on nodes establishing those formatting contexts.
  - *Location:* [`internal/layout/layout.go:1022-1075`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/layout.go#L1022-L1075)
  - *Cost:* Reduces memory footprint of every block, inline, and flex box across large documents.
  - *Proof:* `make test && make bench`

### 2.4 Box Chrome Geometry Helper Unification
- [ ] **DP-08 · P2 · Centralize Chrome Geometry Calculation**: Unify repeated inline calculations of horizontal chrome (`paddingLeft + paddingRight + borderLeft.Width + borderRight.Width`) and vertical chrome (`paddingTop + paddingBottom + borderTop.Width + borderBottom.Width`) into helper methods `horizChrome(sty)` and `vertChrome(sty)` on `engine`.
  - *Location:* [`internal/layout/flex.go:582, 1171`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/flex.go#L582), [`internal/layout/grid.go:53, 763`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/grid.go#L53), [`internal/layout/layout_tables.go:37, 82`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/layout_tables.go#L37), [`internal/layout/layout.go:1216, 1248`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/layout.go#L1216)
  - *Proof:* `go test ./internal/layout/...`

---

## Phase 3: Behavioral Patterns & Pipelines

### 3.1 Unified Conversion Pipeline Lifecycle
- [ ] **DP-09 · P1 · Lifecycle Template Method Preservation**: Maintain the [`render.Pipeline`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/render/pipeline.go#L14-L18) interface (`RenderObjects`, `Assemble`, `Finalize`) and `render.Run` template method coordinating execution stages with context cancellation checks.
  - *Location:* [`internal/convert/render/pipeline.go:14-58`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/render/pipeline.go#L14-L58)
  - *Proof:* `go test ./internal/convert/render/...`

### 3.2 HTML Tree Builder Encapsulation
- [ ] **DP-10 · P2 · Encapsulate HTML Tree Construction**: Group free parser functions ([`appendToken`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/html/html.go#L133), [`openElement`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/html/html.go#L162), [`closeElement`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/html/html.go#L225), [`autoCloseOpen`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/html/html.go#L249)) and pointer-to-slice stack `stack *[]*Node` into a dedicated `treeBuilder` struct adhering to standard Go parser idioms.
  - *Location:* [`internal/html/html.go:118-288`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/html/html.go#L118-L288)
  - *Proof:* `go test ./internal/html/...`

### 3.3 CSS Selector Strategy & Precomputed Specificity
- [ ] **DP-11 · P2 · Right-to-Left Matching & Strategy Dispatch**: Preserve right-to-left combinator walk ([`internal/css/has.go:256-295`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css/has.go#L256-L295)) and pseudo-class strategy dispatcher ([`matchPseudo`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css/css.go#L1594-L1614)), along with pre-parsed integer arithmetic in [`nthForm`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css/css.go#L1758-L1838) and cached specificity triples on `Selector.spec`.
  - *Location:* [`internal/css/has.go:256-295`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css/has.go#L256-L295), [`internal/css/css.go:1390-1838`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css/css.go#L1390-L1838)
  - *Proof:* `go test ./internal/css/...`

### 3.4 Centralized Paint Order Policy
- [ ] **DP-12 · P1 · Shared Paint Order Policy**: Preserve centralized [`paintOrderBefore`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_order.go#L38-L64) sorting (z-index, positioned status, chrome vs content layers) shared across PDF pages, fixed headers/footers, and raster image rendering.
  - *Location:* [`internal/layout/paint_order.go:38-64`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_order.go#L38-L64)
  - *Proof:* `go test -run TestPaintOrder ./internal/layout/...`

### 3.5 Pagination State Machine & Fragmentation Isolation
- [ ] **DP-13 · P1 · Encapsulate Pagination Fixpoint State Machine**: Encapsulate multi-pass pagination algorithms (`paginateOps`, `beforeAlways`, `afterBreaks`, `rowsIntact`, `orphansWidows`, `capTablePageBreaks`) into a formal `Paginator` struct to clarify convergence passes and decouple pagination state from raw `Result.Ops`.
  - *Location:* [`internal/layout/paint_pagination.go:457-543`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_pagination.go#L457-L543), [`internal/layout/paint_flow.go:822-1572`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_flow.go#L822-L1572)
  - *Proof:* `make test && make golden`

---

## Phase 4: Output Adapters & Protocol Serializers

### 4.1 Version-Aware PDF Object Serializer
- [ ] **DP-14 · P1 · Single-Pass Counting Writer Serializer**: Preserve version-aware PDF serialization via [`countingWriter`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/pdf.go#L43-L78) and `writeTo`, enforcing strict byte offset tracking and version-specific semantics:
  - PDF 1.4 Latin-1 encoding with `pdfDocEncodingFold`.
  - PDF 1.7 UTF-16BE encoding (`<FEFF...>`).
  - PDF 2.0 UTF-8 BOM encoding (`<EFBBBF...>`) and omission of obsolete `/ProcSet`.
  - PDF/UA-2 dual named destinations (`/D` and `/SD`) per ISO 14289-2.
  - *Location:* [`internal/pdf/pdf.go:43-78, 544-550, 666-699, 938-989, 1427-1453`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/pdf.go#L43-L78)
  - *Proof:* `go test ./internal/pdf/...`

### 4.2 OpenType HarfBuzz Adapter & Font Subsetting
- [ ] **DP-15 · P1 · Pure-Go Font Subsetting & Shaping Adapter**: Preserve OpenType HarfBuzz shaping adapter ([`internal/pdf/shape_gotext.go:109-150`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/shape_gotext.go#L109-L150)) with TrueType hinting bytecode stripping ([`stripGlyphHints` in `subset.go:207-297`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/subset.go#L207-L297)), 4-byte glyf padding, and dual simple/Type0 font embedding.
  - *Location:* [`internal/pdf/shape_gotext.go:109-150`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/shape_gotext.go#L109-L150), [`internal/pdf/subset.go:38-297`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/subset.go#L38-L297), [`internal/pdf/fonttype0.go:36-104`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/fonttype0.go#L36-L104)
  - *Proof:* `go test -run TestFontSubset ./internal/pdf/...`

### 4.3 SVG Rasterizer Adapter with Panic Recovery
- [ ] **DP-16 · P2 · Isolated Canvas Rasterizer**: Preserve [`internal/svg/raster.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/svg/raster.go#L45-L103) adapter around `tdewolff/canvas` with `defer-recover` panic isolation, intrinsic 96dpi pixel size negotiation, and dimension bounds validation.
  - *Location:* [`internal/svg/raster.go:45-103`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/svg/raster.go#L45-L103)
  - *Proof:* `go test ./internal/svg/...`

### 4.4 Outline Geometric Projection & Clamp Tree Builder
- [ ] **DP-17 · P2 · Decoupled Outline Projection**: Preserve geometric level-stack tree builder in [`internal/outline/outline.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/outline/outline.go#L21-L45,L268-L312) projecting layout coordinates via `locationReader` interface without direct layout package coupling.
  - *Location:* [`internal/outline/outline.go:21-45, 268-312`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/outline/outline.go#L21-L45)
  - *Proof:* `go test ./internal/outline/...`

---

## Phase 5: Verification & Closure Gates

- [ ] **DP-18 · P0 · Unit and Integration Suite Gate**: Run full test suite across all packages.
  - *Command:* `make test`
  - *Proof:* Exit code 0, all tests pass.

- [ ] **DP-19 · P0 · Golden Corpus Integrity Gate**: Verify all 61 golden fixtures render with exact structural fidelity, embedded fonts, and page count envelopes.
  - *Command:* `make golden`
  - *Proof:* Exit code 0, 61/61 fixtures pass.

- [ ] **DP-20 · P0 · Linter Cleanliness Gate**: Verify no lint errors across Go and frontend packages.
  - *Command:* `make lint`
  - *Proof:* Exit code 0, clean output.

- [ ] **DP-21 · P0 · Product Claims Gate**: Verify no forbidden claims in documentation or CLI help text.
  - *Command:* `make claim-scan`
  - *Proof:* Exit code 0, clean scan.

---

## Dependencies

- DP-07 (Box Struct Specialization) and DP-08 (Chrome Geometry) unblock layout memory optimization.
- DP-10 (HTML Tree Builder) unblocks HTML parser modernization.
- DP-13 (Pagination State Machine) unblocks clean pagination refactoring.
- DP-18 through DP-21 are mandatory closure gates before marking the review phase complete.
