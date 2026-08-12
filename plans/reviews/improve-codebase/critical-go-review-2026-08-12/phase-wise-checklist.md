# Critical Go Review — Phase-Wise Remediation Checklist

> **Parent:** [`critical-golang-architecture-review.md`](./critical-golang-architecture-review.md) — evidence and severity rationale.
> **Status:** remediation implemented and validated on 2026-08-12; all rows below are closed with current source, test, benchmark, or documentation evidence.
> **Estimated effort:** 10–16 focused engineering days, excluding corpus-led performance experiments.

---

## Overview

This is the one canonical execution ledger for findings from the critical Go review. Each row records its implementation and current proof. Evidence-gated risks are closed with an explicit measured decision.

## Executive Summary

Fix output integrity before optimizing anything. The order is correctness and compatibility first, then API contracts, then lifecycle semantics, then performance decisions proven by representative workloads.

## Phase 1: Output integrity — P0

### 1.1 Separate incompatible stdout formats

- [x] **CR-01** Updated `internal/app/pdf.go` validation to reject `--dump-outline` with CLI PDF stdout before opening a sink; explicit library writers remain supported. Proof: command-level regression passed.
- [x] **CR-01** Added positive file-PDF plus XML-stdout regression; PDF begins `%PDF-` and XML is independently emitted. Proof: `internal/app/pdf_test.go` and `cmd/gowkhtmltopdf/main_test.go` passed.

### 1.2 Make page islands genuinely fail closed

- [x] **CR-02** Normal requests no longer infer page islands from HTML; only `NewBenchmarkPDFRequest` opts into the internal benchmark path. Proof: spoof/fallback tests passed.
- [x] **CR-02** Island trees recursively clone attributes and descendants with owned `Parent` pointers; source DOM remains unchanged. Proof: structural traversal passed under race testing.
- [x] **CR-02** Unsupported eligibility and expanded sections use generic pagination rather than returning `errCertifiedIslandExpanded`. Proof: fallback and two-section PDF tests passed.

## Phase 2: CLI and public request contracts — P1

### 2.1 Terminal command semantics

- [x] **CR-03** Modeled `--dump-default-toc-xsl` as a validated terminal action independent of page/output positionals. Proof: standalone command exits 0 and emits the default XSL.
- [x] **CR-03** Preserved invalid-mode, malformed-boolean, positional, and `--dump-outline` conflicts. Proof: table-driven parser and command tests passed.

### 2.2 Typed settings and requests

- [x] **CR-04** Normalized image keys before special-case detection and implemented `background`/`web.background` Get round trips, including case and whitespace. Proof: API alias and effective PNG pixel tests passed.
- [x] **CR-05** Added root-owned matchable request errors and renderable-object preflight before engine/output setup. Proof: nil/no-object/TOC-only/empty/missing-sink tests perform zero writes.
- [x] **CR-05** Unified renderable-object validation across legacy and typed PDF/image entry points through `convert.ValidateRenderableObjects`; matrix tests passed.

## Phase 3: Lifecycle semantics and determinism — P2

### 3.1 Prompt cancellation

- [x] **CR-06** Threaded context through stylesheet collection, style resolution, cascade, and container measurement with bounded polling. Proof: generated DOM/CSS cancellation test returns `context.Canceled`.
- [x] **CR-06** Captured non-cancelled style benchmark: `BenchmarkStyleResolutionContext` one iteration = 145.63 ms, 176,088 B/op, 25 allocs/op on the current host.

### 3.2 Explicit ownership contracts

- [x] **CR-07** Added stable face registration order, deduplication, and fingerprint/name tie-breaking outside the registry lock. Proof: equal-score permutation test passed.
- [x] **CR-08** Removed partial `Document` synchronization and documented single-goroutine ownership in `pdf.go` and `documentation/architecture/09-pdf-writer.md`; race gate is recorded below.

## Phase 4: Measured scalability decisions — P3

### 4.1 Resource reuse and budgets

- [x] **R-01** Built and ran a local 1/10/500-page repeated-resource corpus with eight identical image references per page. Evidence: fetches = 1/10/500, fetched bytes = 142/1,420/71,000, PDF bytes = 44,239/388,773/19,105,376, and valid PDF output; full rows are Snapshot E. The rows report wall time and Go allocation traffic; they do not infer process RSS from `B/op`.
- [x] **R-01** Evidence did not warrant a cross-object cache or aggregate cap in this change: per-layout cache hit behavior is proven, repeated bytes are tiny, and 500-page cost is conversion allocation traffic. The separate direct-CLI matrix supplies process-RSS evidence for its own workload. Decision is recorded in the benchmark README.

### 4.2 Layout/raster hotspots

- [x] **R-02** Added and ran 10/100/1,000 transform/sticky/forced-break benchmarks: 9.19/1.93/24.06 ms and 7.21/1.10/9.85 MB B/op; CPU profile inspected with `go tool pprof -top`; repeated 10/100-item PDFs are byte-stable.
- [x] **R-03** Replaced fixture-ID/class/text predicates with generic frame, pagination-shift, and rounded-rail geometry rules; fixture CSS now expresses the neutral frame. Full layout fixture/regression tests pass.
- [x] **R-04** Added cold Latin and bundled Unicode-fallback CJK raster benchmarks at 12/24/72px and reused active-edge scratch per glyph. Existing image tests show no raster drift; full rows are Snapshot E.

## Phase 5: Closure gates — P4

- [x] Targeted CR-01–CR-08 suites passed: `GOCACHE=/tmp/gowk-go-cache go test ./cmd/gowkhtmltopdf ./internal/cli ./internal/app ./internal/convert ./internal/convert/islands ./internal/layout ./internal/pdf ./internal/imageout`.
- [x] `GOCACHE=/tmp/gowk-go-cache make lint` passed after remediation.
- [x] `GOCACHE=/tmp/gowk-go-cache make test` passed after remediation.
- [x] `GOCACHE=/tmp/gowk-go-cache go test -race -count=1 ./...` passed with no race reports.
- [x] Narrow rendering checks passed for fixture-56 architecture diagram and fixture-23 repeated-header regression; generated deep-chrome PDFs were byte-stable and benchmark page-count checks remained green.
- [x] Recalculated review evidence: confirmed output-integrity/API risks are remediated, lifecycle contracts are explicit, and measured scalability rows are baselined. Parent score is updated to **8.8 / 10** with arithmetic recorded in the parent report.

## Dependencies

`Phase 1 → Phase 2 → Phase 3 → Phase 5`. Phase 4 is evidence-gated and may run after Phase 1 in parallel only when it does not alter output semantics. Phase 5 cannot complete until all accepted CR rows are checked or explicitly deferred with a new dated ledger pointer.
