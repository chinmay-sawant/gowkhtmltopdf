# 0.2.6 agy-review - Go Code Style and Idioms Review

> **Parent:** [`plans/0.2.6/48-canonical-0.2.6-css-coverage.md`](../48-canonical-0.2.6-css-coverage.md) - v0.2.6 CSS coverage and architecture ledger
> **Status:** Review and ledger creation complete; implementation pending
> **Estimated effort:** 3-5 focused engineering days
> **Date:** 2026-08-28
> **Scope:** Entire Go codebase across root API, `internal/`, `cmd/`, and `bindings/c/`
> **Standards:** Go code style, effective Go idioms, `skills/phase-wise-checklist/SKILLS.md`, repo `AGENTS.md`

---

## Overview

This document is the canonical Go code style, idioms, and robustness review and implementation ledger for `gowkhtmltopdf` v0.2.6. Six specialized subagents conducted a comprehensive audit across all Go packages:
- Public API and application boundaries ([`document.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go), [`document_validate.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document_validate.go), [`api.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/api.go), [`internal/settings/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/settings), [`internal/app/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/app), [`internal/cli/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/cli), [`bindings/c/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bindings/c))
- Conversion pipeline and rendering lifecycle ([`internal/convert/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert), [`internal/convert/prepare/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/prepare), [`internal/convert/render/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/render), [`internal/convert/islands/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/islands), [`internal/imageout/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/imageout))
- HTML parsing, CSS cascade, selectors, and SVG engine ([`internal/html/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/html), [`internal/css/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css), [`internal/svg/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/svg))
- Layout engine, geometry, and formatting contexts ([`internal/layout/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout), [`internal/line/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/line))
- Paint traversal, pagination state machine, and fragmentation ([`internal/layout/paint.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint.go), [`internal/layout/paint_flow.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_flow.go), [`internal/layout/paint_pagination.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_pagination.go), [`internal/layout/paint_order.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_order.go))
- PDF serializer, font subsetting, outline builder, and resource loader ([`internal/pdf/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf), [`internal/pdfprofile/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdfprofile), [`internal/outline/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/outline), [`internal/load/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/load), [`internal/errs/`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/errs))

The review checks error handling, context propagation, data ownership, zero-allocation idioms, file length soft limits, nil safety, and linter hygiene.

---

## Executive Summary

The Go code exhibits strong idioms and high mechanical quality:
1. **Defensive intake cloning**: Slices, maps, and byte buffers are copied upon entry at public constructors ([`HTML`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L28-L29), [`NewDocument`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L179-L188)) and return points ([`Document.PDF`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L248-L250), [`ImageDocument.Image`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document.go#L309-L311)).
2. **Context discipline**: `ctx context.Context` is passed as the first parameter across all public and internal entry points, with immediate `ErrNilContext` validation.
3. **Zero-allocation parsing paths**: High-frequency ASCII scanning routines ([`hasClassToken`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css/css.go#L1874-L1901), [`containsWord`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css/css.go#L1565-L1592), [`endTagName`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/html/html.go#L529-L572)) scan byte-by-byte with zero heap allocations.
4. **Buffer efficiency**: PDF stream operators use [`AvailableBuffer()`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/pdf/content.go#L88-L122) directly, avoiding format string allocations.

Key code style findings and actionable improvements include:
- **Nil Safety in Style Lookups**: Replacing direct `e.styles[node]` map accesses with `e.stylePtr(node)` across layout submodules.
- **TOC HTML Escaping**: Escaping dynamic heading titles and anchors during TOC HTML generation.
- **Outline XML Dump Typo**: Correcting literal `\node` strings in `DumpOutlineXML`.
- **Deferred Output File Closure**: Unifying `defer closeOut()` handling in `internal/app/` using `errors.Join`.
- **File Length Soft Limit Splits**: Partitioning 4 large files approaching or exceeding 2,000 lines (`internal/css/css.go`, `internal/layout/grid.go`, `internal/layout/paint_flow.go`, `internal/layout/paint_pagination.go`).

### Code Style Scorecard

| Dimension | Score (1-10) | Evaluation |
|---|---:|---|
| **Nil Safety & Correctness** | 8.8 | Strong bounds checks; style map lookups should systematically route through `stylePtr`. |
| **Error Handling & Sentinels** | 9.0 | Systematic `%w` wrapping; typed sentinels; opportunities to unify sentinel prefixes. |
| **Context & Concurrency** | 9.5 | Exemplary context propagation, interruptible I/O watcher, single-goroutine discipline. |
| **Allocation & Buffer Idioms** | 9.2 | High use of `AvailableBuffer()`, zero-alloc ASCII fast paths, and `sync.Pool`. |
| **File Organization & Seams** | 8.2 | Four legacy files require modular splits to satisfy the ~2,000-line soft limit. |
| **Overall Code Style Rating** | **8.9 / 10** | High-quality, idiomatic Go codebase with concrete, actionable refinement opportunities. |

---

## Phase 1: Correctness & Nil Safety

### 1.1 Layout Style Map Lookup Safety
- [ ] **STYLE-01 · P0 · Safe Style Pointer Resolution**: Replace direct map lookups `e.styles[node].Field` with `e.stylePtr(node)` or `eng.stylePtr(node)` across layout formatting context modules to guarantee that active style overrides are respected and unstyled or synthetic nodes safely return the zero-value fallback without panics.
  - *Location:* [`internal/layout/grid.go:196, 284, 674, 847, 1601`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/grid.go#L196), [`internal/layout/layout_flow.go:76, 326, 333, 359`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/layout_flow.go#L76), [`internal/layout/layout_tables.go:320, 349`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/layout_tables.go#L320), [`internal/layout/multicol.go:104, 127, 164`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/multicol.go#L104), [`internal/layout/flex.go:1344`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/flex.go#L1344)
  - *Proof:* `go test ./internal/layout/...`

- [ ] **STYLE-02 · P0 · Sticky Positioning Nil Guard**: Add explicit nil checks on `boxNode.style` in [`internal/layout/sticky.go:21, 69`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/sticky.go#L21) before dereferencing `boxNode.style.Position` or `boxNode.style.Overflow`.
  - *Location:* [`internal/layout/sticky.go:21, 69`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/sticky.go#L21)
  - *Proof:* `go test -run TestSticky ./internal/layout/...`

### 1.2 Output Generation Escaping & String Correctness
- [ ] **STYLE-03 · P0 · Correct Outline XML String Literal**: Fix typo in [`internal/outline/outline.go:380-391`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/outline/outline.go#L380-L391) where `\"/>\node"` and `</item>\node"` contain literal `"node"` characters after `\n`, replacing with clean `\n`.
  - *Location:* [`internal/outline/outline.go:380-391`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/outline/outline.go#L380-L391)
  - *Proof:* `go test -run TestDumpOutlineXML ./internal/outline/...`

- [ ] **STYLE-04 · P0 · HTML Escape Table of Contents Headings**: In [`internal/convert/toc.go:114-135`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/toc.go#L114-L135), escape dynamic heading titles `hVal.Title` and anchors `hVal.Anchor` using `html.EscapeString` when generating TOC HTML markup, preventing malformed DOM trees when headings contain `<`, `>`, or `&`.
  - *Location:* [`internal/convert/toc.go:114-135`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/toc.go#L114-L135)
  - *Proof:* `go test -run TestTOCSpecialCharacters ./internal/convert/...`

- [ ] **STYLE-05 · P1 · CLI Header/Footer Replace Routing**: In [`internal/cli/flags.go:543-561`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/cli/flags.go#L543-L561), update `replaceHF` to inspect both `obj.HeaderSet` and `obj.FooterSet` so `--replace` arguments properly target object footers when only footer flags are configured.
  - *Location:* [`internal/cli/flags.go:543-561`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/cli/flags.go#L543-L561)
  - *Proof:* `go test ./internal/cli/...`

---

## Phase 2: Error Handling, Sentinels & API Contracts

### 2.1 Unified Error Sentinels & Multi-Error Wrapping
- [ ] **STYLE-06 · P2 · Standardize Sentinel Error Prefixes**: Standardize root-re-exported error sentinels in [`internal/errs/errs.go`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/errs/errs.go#L7-L20) to use the unified module prefix `"gowkhtmltopdf: ..."` matching root API sentinels.
  - *Location:* [`api.go:54-68`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/api.go#L54-L68), [`internal/errs/errs.go:7-20`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/errs/errs.go#L7-L20)
  - *Proof:* `go test ./...`

- [ ] **STYLE-07 · P2 · Direct Use of Canonical Sentinels**: Eliminate unexported error aliases in [`internal/load/load.go:61-75`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/load/load.go#L61-L75) (`errNilLoader`, `errNilContext`) and reference `errs.ErrNilLoader` and `errs.ErrNilContext` directly at call sites.
  - *Location:* [`internal/load/load.go:61-75`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/load/load.go#L61-L75)
  - *Proof:* `go test ./internal/load/...`

- [ ] **STYLE-08 · P2 · Clean Up Redundant Error Wrapping**: Remove redundant `fmt.Errorf("%w", err)` in [`internal/convert/prepare/styles.go:272`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/prepare/styles.go#L272), returning either `nil, err` directly or adding descriptive context (`"fetch imported stylesheet %q: %w"`).
  - *Location:* [`internal/convert/prepare/styles.go:272`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/prepare/styles.go#L272)
  - *Proof:* `go test ./internal/convert/prepare/...`

### 2.2 Output Stream Lifecycle & Preflight Validation
- [ ] **STYLE-09 · P1 · Consistent Deferred Output Closing**: Unify output file handle closing in [`internal/app/pdf.go:90-100`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/app/pdf.go#L90-L100) and [`internal/app/image.go:52-65`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/app/image.go#L52-L65) using `defer` and `errors.Join(err, closeOut())` to ensure file descriptors are never leaked if a panic or error occurs during conversion.
  - *Location:* [`internal/app/pdf.go:90-100`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/app/pdf.go#L90-L100), [`internal/app/image.go:52-65`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/app/image.go#L52-L65)
  - *Proof:* `go test ./internal/app/...`

- [ ] **STYLE-10 · P2 · Image Document Preflight Validation Parity**: Enhance [`ImageDocument.Validate`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document_validate.go#L83-L109) to validate `Quality` range (0-100) and non-negative `Crop` dimensions before pipeline dispatch, matching the rigor of `Document.Validate`.
  - *Location:* [`document_validate.go:83-109`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/document_validate.go#L83-L109)
  - *Proof:* `go test -run TestImageDocumentValidate ./...`

---

## Phase 3: Concurrency, Context & Resource Management

### 3.1 Context Cancellation & Interruptible I/O
- [ ] **STYLE-11 · P1 · Asynchronous Context Watcher in File I/O**: Preserve the context watcher pattern in [`internal/load/load.go:1094-1130`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/load/load.go#L1094-L1130) which closes file descriptors upon context cancellation while guaranteeing goroutine cleanup via `close(stop); <-watcherDone`.
  - *Location:* [`internal/load/load.go:1094-1130`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/load/load.go#L1094-L1130)
  - *Proof:* `go test -run TestLoadFileCancellation ./internal/load/...`

- [ ] **STYLE-12 · P2 · Precompute SSRF Network CIDRs**: Precompute `net.IPNet` instances for CGNAT (`100.64.0.0/10`) and cloud metadata (`169.254.169.0/24`) at package level in [`internal/load/load.go:738-765`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/load/load.go#L738-L765) rather than re-parsing CIDR strings on every IP validation check.
  - *Location:* [`internal/load/load.go:738-765`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/load/load.go#L738-L765)
  - *Proof:* `go test ./internal/load/...`

### 3.2 Buffer Pool Retention Bounds
- [ ] **STYLE-13 · P2 · Cap Supersample Pool Buffer Retention**: In [`internal/imageout/imageout.go:319-354`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/imageout/imageout.go#L319-L354), place an upper capacity bound (e.g. 32 MiB) on buffers returned to `supersamplePixPool` to avoid pinning multi-hundred-megabyte allocations in memory after rendering large rasters.
  - *Location:* [`internal/imageout/imageout.go:319-354`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/imageout/imageout.go#L319-L354)
  - *Proof:* `go test ./internal/imageout/...`

---

## Phase 4: High-Performance Idioms & Allocations

### 4.1 Pointer vs Value Efficiency in Style Checks
- [ ] **STYLE-14 · P2 · Avoid By-Value Copies of ResolvedStyle**: In [`internal/layout/inline_collect.go:91-109`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/inline_collect.go#L91-L109), replace `e.styleVal(container)` with `e.stylePtr(container)` during DOM ancestor traversal to avoid copying the ~1.3 KB `ResolvedStyle` struct on every inline node inspection.
  - *Location:* [`internal/layout/inline_collect.go:91-109`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/inline_collect.go#L91-L109)
  - *Proof:* `go test ./internal/layout/...`

### 4.2 Pool Allocation Capacity Hints
- [ ] **STYLE-15 · P3 · Provide Initial Capacity Hint in Inline Item Pool**: In [`internal/layout/inline.go:158`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/inline.go#L158), allocate `make([]inlineItem, 0, 32)` when `inlineItemPool` is empty, avoiding geometric slice resizes during initial inline collection.
  - *Location:* [`internal/layout/inline.go:158`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/inline.go#L158)
  - *Proof:* `go test ./internal/layout/...`

### 4.3 CGo Bridge Allocation Efficiency
- [ ] **STYLE-16 · P3 · Eliminate Redundant Byte Copy in C-ABI Image Creation**: In [`bindings/c/options_image.go:42`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bindings/c/options_image.go#L42), construct `Content{HTML: html}` directly with owned Go bytes from `C.GoBytes` instead of calling `gowkhtmltopdf.HTML(html)` which performs a redundant second copy.
  - *Location:* [`bindings/c/options_image.go:42`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bindings/c/options_image.go#L42)
  - *Proof:* `go test ./bindings/c/...`

---

## Phase 5: Code Organization, File Length & Linter Hygiene

### 5.1 Modular File Splits (<2,000 Lines per AGENTS.md)
- [ ] **STYLE-17 · P1 · Split `internal/css/css.go` (1,980 lines)**: Partition along natural seams into:
  - `css.go`: Top-level AST, rule types, top-level parser.
  - `match.go`: Element matching, combinator evaluation, pseudo-classes.
  - `selector_parser.go`: Selector tokenization, compound parser, attribute operators.
  - `specificity.go`: Specificity calculation.
  - *Location:* [`internal/css/css.go:1-1980`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/css/css.go#L1-L1980)
  - *Proof:* `go build ./internal/css/... && go test ./internal/css/...`

- [ ] **STYLE-18 · P1 · Split `internal/layout/grid.go` (1,891 lines)**: Partition into:
  - `grid.go`: Core `buildGrid`, `layoutStandardGrid`, `emitGridBoxes`.
  - `grid_parse.go`: Track tokenization, repeat expansion, template area parser.
  - `grid_placement.go`: Item placement, occupation matrix, auto-flow slot finders.
  - `grid_tracks.go`: Track sizing resolution and intrinsic measurements.
  - `grid_masonry.go`: Masonry track list, item packing, box shifting.
  - *Location:* [`internal/layout/grid.go:1-1891`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/grid.go#L1-L1891)
  - *Proof:* `go build ./internal/layout/... && go test -run TestGrid ./internal/layout/...`

- [ ] **STYLE-19 · P1 · Split `internal/layout/paint_flow.go` (2,368 lines)**: Partition into:
  - `paint_flow_index.go`: Flow op and box index caching, coordinate shifting.
  - `paint_flow_breaks.go`: Forced break rules, heading keep-with-next, aside callout lifting.
  - `paint_flow_tables.go`: Table row intactness, thead cloning onto continuation pages.
  - `paint_flow_orphans.go`: CSS Fragmentation orphans and widows, line baseline counting.
  - *Location:* [`internal/layout/paint_flow.go:1-2368`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_flow.go#L1-L2368)
  - *Proof:* `go build ./internal/layout/... && go test ./internal/layout/...`

- [ ] **STYLE-20 · P1 · Split `internal/layout/paint_pagination.go` (2,245 lines)**: Partition into:
  - `paint_pagination_fixpoint.go`: Pagination coordination and boundary snapping.
  - `paint_pagination_split.go`: Page-boundary rect & border splitting, fragment stroke masking.
  - `paint_pagination_chrome.go`: Box chrome stretching, vertical rail realignment.
  - `paint_pagination_seal.go`: Table continuation border capping, orphan row chrome stripping, section sealing.
  - *Location:* [`internal/layout/paint_pagination.go:1-2245`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_pagination.go#L1-L2245)
  - *Proof:* `go build ./internal/layout/... && go test ./internal/layout/...`

### 5.2 Linter Cleanliness & Complexity Refactoring
- [ ] **STYLE-21 · P2 · Decompose `associateUnmappedOps`**: Refactor [`internal/layout/tagging.go:60-150`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/tagging.go#L60-L150) into specialized sub-functions for link, image, and text ops, eliminating 7 `nolint` suppressions (`cyclop, gocyclo, funlen, gocognit, nestif, varnamelen, wsl`).
  - *Location:* [`internal/layout/tagging.go:60-150`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/tagging.go#L60-L150)
  - *Proof:* `make lint`

- [ ] **STYLE-22 · P2 · Decompose `beforeAlways`**: Extract the scan state and break event accumulator in [`internal/layout/paint_flow.go:822-947`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_flow.go#L822-L947) into a helper struct `breakScanState` to eliminate `//nolint:gocognit,cyclop,funlen`.
  - *Location:* [`internal/layout/paint_flow.go:822-947`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_flow.go#L822-L947)
  - *Proof:* `make lint`

- [ ] **STYLE-23 · P3 · Clean Up Dead Variables and Unused Parameters**:
  - Remove unused `opPage` parameter from `splitCrossingRects` in [`internal/layout/paint_pagination.go:799`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_pagination.go#L799) and [`paint.go:132`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint.go#L132).
  - Remove dead assignment `node := loc.Node` and `_ = node` in [`internal/convert/links.go:53, 65`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/links.go#L53).
  - Clean up unused blank parameters in `collectObjectHeadings` in [`internal/convert/outline.go:137`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/outline.go#L137).
  - Remove duplicate doc comment block on `(*hfDrawResult).Err()` in [`internal/convert/hf.go:386-389`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/hf.go#L386-L389).
  - *Location:* [`internal/layout/paint_pagination.go:799`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/layout/paint_pagination.go#L799), [`internal/convert/links.go:53`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/links.go#L53), [`internal/convert/outline.go:137`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/outline.go#L137), [`internal/convert/hf.go:386`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/convert/hf.go#L386)
  - *Proof:* `go test ./... && make lint`

- [ ] **STYLE-24 · P3 · Implement String and Prefix on Line Severity**: In [`internal/line/line.go:13-41`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/line/line.go#L13-L41), add standard `String()` and `Prefix()` methods on `Severity` and add a nil guard on `writer` in `Emit`.
  - *Location:* [`internal/line/line.go:13-41`](file:///home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/internal/line/line.go#L13-L41)
  - *Proof:* `go test ./internal/line/...`

---

## Phase 6: Verification & Closure Gates

- [ ] **STYLE-25 · P0 · Unit and Integration Suite Gate**: Run full test suite across all packages.
  - *Command:* `make test`
  - *Proof:* Exit code 0, all tests pass.

- [ ] **STYLE-26 · P0 · Golden Corpus Integrity Gate**: Verify all 61 golden fixtures render with exact structural fidelity, embedded fonts, and page count envelopes.
  - *Command:* `make golden`
  - *Proof:* Exit code 0, 61/61 fixtures pass.

- [ ] **STYLE-27 · P0 · Linter Cleanliness Gate**: Verify no lint errors across Go and frontend packages.
  - *Command:* `make lint`
  - *Proof:* Exit code 0, clean output.

- [ ] **STYLE-28 · P0 · Product Claims Gate**: Verify no forbidden claims in documentation or CLI help text.
  - *Command:* `make claim-scan`
  - *Proof:* Exit code 0, clean scan.

---

## Dependencies

- STYLE-01 through STYLE-05 (Phase 1 Correctness & Nil Safety) are top-priority invariants that must precede refactorings.
- STYLE-17 through STYLE-20 (File Splits) unblock complexity reductions in STYLE-21 and STYLE-22.
- STYLE-25 through STYLE-28 are mandatory closure gates before marking the review phase complete.
