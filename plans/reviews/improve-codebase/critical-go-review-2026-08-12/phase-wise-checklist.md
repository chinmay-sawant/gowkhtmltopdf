# Critical Go Review — Phase-Wise Remediation Checklist

> **Parent:** [`critical-golang-architecture-review.md`](./critical-golang-architecture-review.md) — evidence and severity rationale.
> **Status:** planned from the 2026-08-12 review; no remediation is claimed complete.
> **Estimated effort:** 10–16 focused engineering days, excluding corpus-led performance experiments.

---

## Overview

This is the one canonical execution ledger for findings from the critical Go review. Rows are intentionally unchecked until their source, regression proof, and applicable gates succeed. Risks are not implementation commitments.

## Executive Summary

Fix output integrity before optimizing anything. The order is correctness and compatibility first, then API contracts, then lifecycle semantics, then performance decisions proven by representative workloads.

## Phase 1: Output integrity — P0

### 1.1 Separate incompatible stdout formats

- [ ] **CR-01** Update [`cmd/gowkhtmltopdf/main.go`](../../../../cmd/gowkhtmltopdf/main.go) / neutral app validation to reject `--dump-outline` with PDF `Output == "-"`; preserve separate writer behavior for non-CLI callers. Proof: command-level regression returns `cli.ExitError` and no mixed XML/PDF output.
- [ ] **CR-01** Add a positive regression for PDF file output plus XML stdout and verify the PDF starts `%PDF-` while XML remains a standalone document.

### 1.2 Make page islands genuinely fail closed

- [ ] **CR-02** Remove automatic HTML-content qualification from [`internal/convert/convert.go`](../../../../internal/convert/convert.go) for normal requests, or route only a private explicit benchmark request through islands. Proof: spoof marker/title input follows generic rendering.
- [ ] **CR-02** If specialized islands remain, make its virtual tree satisfy every `child.Parent == owner` invariant and avoid mutating shared source DOM. Proof: structural traversal test plus generic/specialized selector parity.
- [ ] **CR-02** Make an eligibility or page-count failure fall back to generic rendering rather than returning `errCertifiedIslandExpanded`. Proof: two-page certified-shape section completes with valid PDF output.

## Phase 2: CLI and public request contracts — P1

### 2.1 Terminal command semantics

- [ ] **CR-03** Model `--dump-default-toc-xsl` as a validated terminal action in [`internal/cli`](../../../../internal/cli), independent of page/output positional requirements. Proof: standalone binary invocation exits 0 and emits the default XSL.
- [ ] **CR-03** Preserve invalid-mode, malformed flag, and conflicting-option validation for the terminal action. Proof: table-driven negative parser/main tests.

### 2.2 Typed settings and requests

- [ ] **CR-04** Normalize image setting keys before special-case detection and implement a `Get` round trip for `background` / `web.background` in [`api.go`](../../../../api.go). Proof: alias/case/whitespace table and PNG effective-pixel test.
- [ ] **CR-05** Add root-owned, `errors.Is`-matchable typed request errors and validate at least one renderable PDF object in `RunPDF`/`ValidatePDF` before engine setup. Proof: no object, TOC-only, empty object, inline body, output, and outline sink cases perform zero output writes when invalid.
- [ ] **CR-05** Specify one public validation matrix shared by legacy `Converter`, `ImageConverter`, typed `PDFRequest`, and typed `ImageRequest`; keep documented differences explicit. Proof: matrix test table, not duplicated ad hoc adapters.

## Phase 3: Lifecycle semantics and determinism — P2

### 3.1 Prompt cancellation

- [ ] **CR-06** Thread `context.Context` through stylesheet collection, style resolution, cascade, and container passes. Check at bounded node/rule/selector intervals. Proof: generated DOM/CSS cancellation test returns `context.Canceled` within a recorded latency budget.
- [ ] **CR-06** Benchmark normal non-cancelled cascade before/after and document any overhead. Proof: `go test` benchmark with representative fixture dimensions and allocations.

### 3.2 Explicit ownership contracts

- [ ] **CR-07** Give `Registry.FindWithGlyph` deterministic tie-breaking based on stable registration order and face identity. Proof: equal-score candidates chosen identically across permutations.
- [ ] **CR-08** Decide and document `pdf.Document` ownership as single-goroutine construction, then remove/replace partial synchronization accordingly. Do not promise concurrent document assembly without a full lifecycle design. Proof: documentation boundary plus `go test -race ./...`.

## Phase 4: Measured scalability decisions — P3

### 4.1 Resource reuse and budgets

- [ ] **R-01** Build a reproducible 1/10/500-page repeated-resource corpus. Measure wall time, RSS, fetched bytes, cache hit rate, PDF bytes, and semantic/visual output. No global cache or aggregate cap is approved before this evidence.
- [ ] **R-01** If evidence warrants it, design request-scoped resource ownership/caps with explicit byte/count semantics and bounded cache eviction; add error taxonomy and negative tests.

### 4.2 Layout/raster hotspots

- [ ] **R-02** Benchmark deep transform/sticky/forced-break documents at 10/100/1,000 scales with CPU/allocation profiles and golden output comparison.
- [ ] **R-03** Replace fixture-specific selectors/text predicates only after a general geometric rule reproduces all approved visual fixtures.
- [ ] **R-04** Benchmark cold glyph raster allocation at 12/24/72 px for Latin and CJK, then decide whether scratch reuse is worth complexity.

## Phase 5: Closure gates — P4

- [ ] Run targeted new regression suites for CR-01 through CR-08 and record commands/results here.
- [ ] Run `GOCACHE=/tmp/gowk-go-cache make lint` and record the result here.
- [ ] Run `GOCACHE=/tmp/gowk-go-cache make test` and record the result here.
- [ ] Run `GOCACHE=/tmp/gowk-go-cache go test -race ./...` and record the result here.
- [ ] Run narrow golden and rendered PDF/PNG checks for changed rendering paths; record fixture names, rasterizer, and visual decision.
- [ ] Recalculate the weighted rating in the parent report from current evidence; do not close risk rows merely because source refactoring landed.

## Dependencies

`Phase 1 → Phase 2 → Phase 3 → Phase 5`. Phase 4 is evidence-gated and may run after Phase 1 in parallel only when it does not alter output semantics. Phase 5 cannot complete until all accepted CR rows are checked or explicitly deferred with a new dated ledger pointer.
