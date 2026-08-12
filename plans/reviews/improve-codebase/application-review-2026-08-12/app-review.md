# Application Review - Phase-Wise 10/10 Checklist

> **Parent:** [`../README.md`](../README.md) - architecture review index and canonical-ledger policy.
> **Related baseline:** [`../critical-go-review-2026-08-12/phase-wise-checklist.md`](../critical-go-review-2026-08-12/phase-wise-checklist.md) - CR-01 through CR-08 remediation, closed on 2026-08-12.
> **Status:** open; baseline audit complete and first implementation wave validated; P0/P1 gaps remain.
> **Created:** 2026-08-12
> **Target:** 10/10 as a controlled, server-generated report renderer. Browser/JavaScript parity remains a separate product scope, not a hidden acceptance criterion.
> **Estimated effort:** 4-8 focused engineering weeks, excluding large browser-parity work.

---

## Overview

This is the canonical execution ledger for moving the application from the
current blended **7.4/10** review score to a release-grade **10/10** within its
declared product boundary: controlled HTML to PDF/PNG/JPEG, predictable
pagination, embeddable Go APIs, explicit security policy, and measured
scalability.

The existing critical Go checklist remains the source of truth for CR-01 to
CR-08. This ledger tracks the remaining application-level gaps found by the
2026-08-12 architecture, application, performance, and release audit.

Every row is an atomic action or one validation result. A row may be checked
only after the named source, test, benchmark, or artifact proof succeeds.

## Executive Summary

| Dimension | Current | 10/10 bar | Main work |
|---|---:|---:|---|
| Architecture and seams | 7.8 | 9.5+ | Remove adapter leakage; deepen engine interfaces |
| Correctness and API contracts | 7.8 | 9.5+ | Add semantic output oracles and writer-first paths |
| Rendering fidelity | 6.8 | 9.5+ | Add visual and extracted-content regression proof |
| Performance and scalability | 7.2 | 9.5+ | Re-baseline post-remediation generic workloads |
| Security and release readiness | 7.3 | 9.5+ | Network policy, licensing, version and CI closure |
| **Blended score** | **7.4** | **10.0** | Complete all P0-P3 rows and rerate |

### Current evidence baseline

- [x] Repository audit completed with three independent review lenses and a
  weighted score of 7.4/10. Evidence: current source, tests, fixtures, docs,
  benchmarks, and clean worktree on `chore/release-prep`.
- [x] `GOCACHE=/tmp/gowk-go-cache go test ./...` passed.
- [x] `go vet ./...` passed.
- [x] `golangci-lint run ./...` passed.
- [x] `CGO_ENABLED=0 go build ./cmd/gowkhtmltopdf ./cmd/gowkhtmltoimage` passed.
- [x] `TestGoldenCorpusAllFixtures` passed for the current 58-fixture corpus.
- [x] `npm run build` passed, including generated-site copy; Vite still warns
  about a JavaScript chunk larger than 1.2 MB.
- [x] CLI smoke conversion produced a valid PDF 1.4 file.
- [x] `GOCACHE=/tmp/gowk-go-cache go test -race -count=1 ./...` passed after the
  first implementation wave. The earlier 5-second race-instrumentation budget
  failure did not reproduce in the current run; keep race correctness and
  release performance thresholds as separate gates.
- [ ] Fresh post-remediation process-level RSS and wall-time evidence is not
  yet available for the generic renderer. The stored 500-page direct-CLI
  snapshot predates CR-02 page-island changes.
- [x] The explicit 500-page benchmark was re-run with request-mode labels. One
  iteration measured generic `1.063s/op`, `226.9 MB/op`, `938,763 allocs/op`
  and certified-islands `1.071s/op`, `187.9 MB/op`, `1,225,009 allocs/op`; both
  produced 500 pages. This supersedes the earlier unlabelled ~11.6-second
  observation for the benchmark path; it is still not process RSS evidence.

## Phase 0: Freeze scope, evidence, and acceptance criteria - P0

> **Status:** baseline complete; release contract open.

### 0.1 Define the 10/10 product boundary

- [x] Record that 10/10 means excellent controlled-report conversion, not
  JavaScript execution or Chrome/WebKit parity. Proof: this ledger and
  [`documentation/fidelity.md`](../../../../documentation/fidelity.md) agree on
  the product boundary.
- [x] Add a single source-of-truth release scorecard covering architecture,
  correctness, fidelity, performance, security, and release operations. Proof:
  the Executive Summary in this ledger is the active scorecard and the parent
  index points here.
- [ ] Mark every item as **proven**, **measured**, **visually inspected**, or
  **inferred risk** in the final review. Proof: final rerating includes all four
  evidence classes explicitly.

### 0.2 Establish reproducible validation inputs

- [ ] Pin the benchmark host/toolchain metadata and create a checked-in command
  matrix for 10/50/100/500-page generic HTML, template HTML, image-heavy HTML,
  CJK/font-fallback HTML, and deep layout HTML. Paths:
  `internal/convert/*_test.go`, `testdata/golden/benchmarks/`.
- [ ] Separate ordinary generic rendering from the explicitly opted-in
  benchmark page-island renderer in every performance report. Proof: each row
  names request mode, fixture path, and eligibility state.
- [ ] Add a review artifact policy that keeps generated PDFs/PNGs out of source
  diffs unless intentionally approved. Proof: `git status --short`, artifact
  manifest, and checklist instructions agree.

## Phase 1: Product truth and release hygiene - P0

> **Status:** open. These changes prevent users from forming incorrect
> expectations before deeper engineering work lands.

### 1.1 Reconcile dependency and capability claims

- [~] Replace stale zero-dependency/stdlib-only claims in
  `documentation/library-api.md`, README, getting-started docs, and frontend
  content with the supported wording: pure Go, no-CGO, no browser process, and
  explicitly documented third-party modules. Proof: repository-wide claim scan
  has no contradictory dependency statement. The active README, API docs,
  frontend claim sources, and package comments are reconciled; historical plans,
  issue records, and fixture prose retain milestone-era wording and need a
  separately scoped archival-doc pass.
- [x] Reconcile the direct dependency allowlist in `go.mod`, Makefile comments,
  README, and the shaping amendment. Proof: `go list -m all` and the documented
  direct allowlist agree; `internal/pdf.TestDirectModuleAllowlist` passes.

### 1.2 Make version identity unambiguous

- [~] Define one version policy connecting `VERSION`, `cli.Version`,
  `LibraryVersion`, README install instructions, and release metadata. Proof:
  README and frontend now distinguish the project release number from the
  upstream compatibility identifier; a dedicated cross-surface version test is
  still required.
- [x] Update help text that still describes the implementation as a Qt WebKit
  engine. Path: `internal/cli/help.go`. Proof: help output matches the pure-Go
  report-renderer positioning and `TestPrintHelpUsesProductTruth` passes.

### 1.3 Finish contributor and generated-artifact workflows

- [x] Implement `make golden-update` or remove/replace the target with an
  explicit supported regeneration command. Path: `Makefile`, golden docs.
  Proof: the approval-gated single-fixture PDF workflow passes its refusal and
  dry-run checks without rewriting committed fixtures.
- [x] Document exactly which golden outputs are structural, semantic, visual,
  and release-only. Proof: `testdata/golden/README.md` and
  `documentation/samples.md` use the same terminology.
- [ ] Add a release checklist for font notices, generated frontend output,
  sample PDFs, benchmark snapshots, and `git diff --check`.

## Phase 2: Semantic and visual output correctness - P0

> **Status:** open. Structural PDF validity is not sufficient for a renderer.

### 2.1 Build a semantic PDF oracle

- [~] Add deterministic PDF assertions for extracted text order/content, page
  geometry, metadata, font resources, image resources, external URI links,
  internal destinations, outlines, copies, and page reordering. Paths:
  `internal/pdf/semantic_oracle_test.go` now covers header/version, page count,
  text order, geometry, title metadata, font/image resources, URI/internal
  links, and an outline; copies, page reordering, and fixture-backed coverage
  remain open.
- [x] Add negative tests proving malformed or incomplete output fails the oracle
  even when it begins with `%PDF-` and contains `%%EOF`.
- [ ] Run the semantic oracle against representative fixtures 01, 03, 16, 21,
  23, 27, 32, 43, 55, and 56 before changing layout or PDF internals.

### 2.2 Build a raster visual regression harness

- [ ] Add a deterministic Ghostscript raster command using `/usr/bin/gs` and a
  fixture manifest with page selection, DPI, crop regions, and approved image
  baselines. Do not compare only page counts or file bytes.
- [ ] Add visual assertions for invoice geometry, table borders, repeated
  headers, floats, flex/grid, typography/letter spacing, CJK, images, links,
  headers/footers, and multi-page dossier layout.
- [ ] Record expected visual tolerances and classify failures as layout,
  pagination, paint, typography, image, or PDF serialization defects.
- [ ] Add browser/reference comparison for a small controlled HTML subset. The
  comparison must validate the declared report subset, not claim browser parity
  for arbitrary sites.

### 2.3 Close known visual-validation gaps

- [ ] Re-render and inspect fixtures 21, 23, 28, 43, and 55 at 100% scale;
  capture affected-page screenshots and record the result in the fixture
  manifest. Existing page-count success is not closure proof.
- [ ] Add a focused typography crop comparison for fixture 55's masthead and
  any remaining Arial/Helvetica fallback mismatch.
- [ ] Add semantic-tag and graceful-degradation assertions for fixture 56;
  ensure unsupported modern CSS is ignored without damaging supported fallback
  layout.

## Phase 3: Deepen architecture and remove adapter leakage - P1

> **Status:** open. Preserve output behavior while improving locality and
> reusability.

### 3.1 Keep CLI concerns outside the engine

- [ ] Remove direct `internal/cli` imports from core conversion/image packages;
  translate CLI commands into engine requests only in `internal/app` and command
  adapters. Proof: dependency graph shows CLI -> app -> convert, never convert
  -> CLI.
- [ ] Add package-boundary tests that construct requests without argv parsing or
  process-global stdout/stderr.
- [ ] Preserve existing CLI behavior with black-box tests for page/cover/toc,
  output files, stdout, outline sinks, terminal actions, and mode-specific flags.

### 3.2 Separate lifecycle from PDF-specific page planning

- [ ] Split generic stage ordering/cancellation from PDF page-copy, outline,
  link, and page-reorder logic. Paths: `internal/convert/render/`,
  `internal/convert/pdf_pipeline.go`, `internal/convert/render/plan.go`.
- [ ] Keep the lifecycle interface small and deep: callers know stage ordering
  and cancellation, while PDF assembly details remain behind the adapter seam.
- [ ] Add tests proving lifecycle behavior remains mode-neutral and PDF-specific
  assembly remains deterministic.

### 3.3 Seal request and configuration ownership

- [ ] Replace the broad internal PDF/image request union with sealed mode-owned
  execution inputs or a narrower adapter contract. Keep public typed request
  APIs source-compatible where practical.
- [ ] Create immutable per-run configuration snapshots so a caller cannot mutate
  settings during conversion. Proof: mutation tests and race tests cover maps,
  slices, inline HTML, callbacks, and output writers.
- [ ] Document that one converter/document is single-owner while independent
  conversions may run concurrently.

## Phase 4: Security and operational safety - P1

> **Status:** open. Existing defaults are good; production policy needs a
> stronger explicit seam.

### 4.1 Add an explicit network-policy seam

- [x] Introduce a loader policy interface/configuration for allowed schemes,
  host allowlists, private/link-local IP blocking, redirect policy, and proxy
  behavior. Preserve an explicit compatibility mode for existing trusted URL
  users. Proof: public `GlobalSettings.SetNetworkPolicy`, compatible/restricted
  constructors, loader enforcement, and focused tests are implemented.
- [~] Add tests for DNS names resolving to loopback, RFC1918, link-local,
  metadata-service ranges, cross-host redirects, credentials, and second-hop
  image/CSS URLs. Literal loopback, explicit private-host exceptions, and
  cross-host redirect rejection are covered; injectable DNS and second-hop
  coverage remain open.
- [ ] Update `documentation/THREAT-MODEL.md`, `integration-security.md`, and
  CLI docs with the exact default and compatibility behavior.

### 4.2 Add workload isolation guidance and limits

- [ ] Document a recommended isolated worker/container profile for untrusted
  HTML, including egress restrictions, filesystem restrictions, credentials,
  timeout, body, image, page, and concurrency limits.
- [ ] Add request-level aggregate budgets for total fetched bytes, resource
  count, DOM size, CSS rule count, rendered pages, and output size where the
  corpus proves safe limits.
- [ ] Add bounded-concurrency integration guidance rather than promising that a
  single in-process converter is a complete SSRF or DoS boundary.

## Phase 5: Performance, memory, and scalability - P1

> **Status:** open. Optimize only after the current post-remediation baseline
> and semantic oracle are in place.

### 5.1 Re-establish the complete performance baseline

- [ ] Re-run 2/5/10/20/50/100/200/250/500-page matrices after CR-02 using at
  least three iterations per row. Record median, min/max, wall time, Go B/op,
  allocations/op, PDF bytes, exact page count, and request mode.
- [ ] Run the same workloads as direct CLI processes under `/usr/bin/time` and
  record peak RSS separately from Go allocation traffic.
- [ ] Include generic full-document fixtures for tables, flex/grid, images,
  CJK/fallback, forced breaks, headers/footers, and deep nested layout.
- [ ] Store raw results in `testdata/golden/benchmarks/` with host/toolchain,
  command, cache state, and historical/current labels.

### 5.2 Explain and repair the current benchmark-path regression

- [x] Profile the current explicit 500-page benchmark from `internal/convert`
  and identify why recursive island cloning increased the measured run to about
  11.6 seconds. Use CPU and allocation profiles before changing code. The
  current profile points to GC/layout allocation pressure and approximately
  16.5 MB of recursive-cloning allocations per operation.
- [x] Restore a bounded reusable workspace or revise the benchmark fixture path
  without weakening layout, pagination, link, font, image, or PDF semantics.
  The benchmark now explicitly separates generic and certified-islands request
  modes and retains a 500-page assertion for both.
- [ ] Add a differential oracle between generic and certified paths covering page
  count, extracted text, links, headings, fonts/images, PDF structure, and
  rasterized output.
- [ ] Do not claim the optimized result until the fresh count-3 result and
  direct-CLI RSS result are both recorded.

### 5.3 Reduce output and layout memory without semantic weakening

- [ ] Profile raw versus compressed page-stream retention in `internal/pdf` and
  evaluate sealed/spooled page content with cleanup on success, error, copies,
  reordering, and header/footer assembly.
- [ ] Add a writer-first public conversion path that avoids accumulating and
  copying the complete PDF when the caller already supplied an `io.Writer`.
- [ ] Measure and cap smart-width/layout pass counts for image conversion;
  record when re-layout occurs and whether it is caused by real overflow.
- [ ] Add allocation/RSS evidence for repeated resources, large images, CJK
  glyphs, font fallback, and multi-object documents before adding global caches.

### 5.4 Add stage-level observability

- [ ] Expose optional internal/test instrumentation for load, parse, CSS,
  cascade, layout, pagination, paint, assembly, compression, and write
  durations.
- [ ] Record resource fetch count/bytes, page count, output bytes, layout pass
  count, and cancellation stage in benchmark artifacts.
- [ ] Keep production callbacks lightweight and preserve the existing public
  human-readable phase/progress API.

### 5.5 Measure bounded concurrency

- [ ] Add a benchmark for independent concurrent conversions at worker counts
  1, 2, 4, and 8 with peak RSS, throughput, latency distribution, and output
  validity.
- [ ] Add a recommended worker/backpressure policy for HTTP service embedding;
  do not make `pdf.Document` concurrently mutable.

## Phase 6: Frontend and documentation delivery - P2

> **Status:** open. The documentation site is part of the application surface.

### 6.1 Reduce frontend delivery cost

- [ ] Split dossier/showcase/heavy fixture assets with dynamic imports or
  route-level chunks. Proof: production build no longer emits an unjustified
  >1.2 MB primary JavaScript chunk.
- [ ] Add a production preview smoke test for landing, documentation, dossier,
  showcase, dark mode, keyboard navigation, and the `/gowkhtmltopdf/` base path.
- [ ] Keep generated `docs/` and `dist/` outputs produced only by
  `npm run build`; do not hand-edit them.

### 6.2 Synchronize product documentation

- [ ] Make README, fidelity guide, compatibility matrix, API docs, security
  docs, frontend copy, and roadmap agree on supported/partial/deferred
  features.
- [ ] Add a documentation claim scan to CI for forbidden stale phrases such as
  `stdlib-only`, `zero third-party`, `full browser parity`, and unqualified
  `deterministic bytes`.

## Phase 7: Release closure and rerating - P0

> **Status:** blocked until Phases 1-5 produce current proof; this is a closure
> phase, not a substitute for implementation.

### 7.1 Separate and repair validation gates

- [x] Make the normal correctness gate pass: `make test` (passed in the
  current worktree).
- [x] Make the style/static gate pass: `make lint` and `go vet ./...` (both
  passed in the current worktree).
- [x] Make the race gate pass without a false performance-budget failure:
  `go test -race -count=1 ./...`.
- [x] Make the build gate pass with `CGO_ENABLED=0` for both binaries.
- [x] Make the semantic golden gate pass for the full fixture corpus.
- [ ] Make the visual regression gate pass for the approved fixture manifest.
- [~] Make the frontend gate pass: `npm run build` plus production preview
  smoke. `npm --prefix frontend run build` passed and regenerated `docs/`; the
  production preview/browser smoke remains open and the build still warns about
  a 1.2 MB JavaScript chunk.
- [ ] Make the release benchmark gate pass with current generic wall time, RSS,
  allocation, output-size, and page-count thresholds.

### 7.2 Recalculate the score

- [ ] Re-run the four review lenses with current source and artifacts.
- [ ] Recalculate the weighted score using the same dimensions and arithmetic
  at the top of this file.
- [ ] Mark 10/10 only when every P0/P1 row is `[x]`, every remaining `[~]` row
  has an explicit product-scope reason, all closure gates pass, and no stale
  release claim remains.
- [ ] If a row is postponed, move it to a dated deferred ledger and change this
  row to `[~]` with a pointer; do not leave duplicate active work here.

## Dependencies

```text
Phase 0 scope/evidence
  ├──> Phase 1 product truth/release hygiene
  ├──> Phase 2 semantic + visual correctness
  ├──> Phase 3 architecture seams
  └──> Phase 4 security policy

Phase 2 + Phase 3 ──> Phase 5 performance and memory
Phase 1 + Phase 4 + Phase 5 ──> Phase 6 delivery polish
All phases ──> Phase 7 release gates and rerating
```

## Validation command set

```sh
GOCACHE=/tmp/gowk-go-cache go test ./...
go vet ./...
make lint
CGO_ENABLED=0 go build ./cmd/gowkhtmltopdf ./cmd/gowkhtmltoimage
GOCACHE=/tmp/gowk-go-cache go test ./internal/convert -run '^TestGoldenCorpusAllFixtures$' -count=1
GOCACHE=/tmp/gowk-go-cache go test -race -count=1 ./...
npm --prefix frontend run build
```

Performance commands must additionally record exact fixture, request mode,
cache state, iteration count, wall time, RSS, B/op, allocations/op, PDF bytes,
and page count. A passing command alone is not benchmark proof.

## 10/10 definition of done

- No contradictory dependency, version, capability, or licensing claims.
- Public CLI, library, image, and PDF contracts are tested at their seams.
- Representative PDFs pass structural, semantic, and raster visual checks.
- Generic and specialized rendering paths are explicitly separated and
  differentially proven where both exist.
- Current performance is measured after the latest code, with wall time and RSS
  reported separately from Go allocation traffic.
- Output buffering, smart-width passes, concurrency, and resource budgets have
  evidence-backed behavior.
- Untrusted URL conversion has an explicit network policy and isolation guidance.
- Full tests, vet, lint, race, static builds, golden tests, visual checks, and
  frontend production checks pass.
- The final weighted score is at least 9.5 in every dimension and rounds to
  **10.0/10** under the declared controlled-report product scope.
