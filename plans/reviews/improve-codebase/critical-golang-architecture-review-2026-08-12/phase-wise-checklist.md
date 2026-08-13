# Critical Go Architecture Review — Completed Phase-Wise Checklist

> **Parent:** [`critical-golang-architecture-review.md`](../../../../reports/critical-golang-architecture-review.md)
> **Status:** Complete — every action row is implemented, measured, or closed with an explicit compatibility decision.
> **Date:** 2026-08-12
> **Rule:** This checklist contains no pending or deferred action rows. Each checked row has proof below it.

## Baseline and rerated outcome

- [x] **BASE-01** Cross-check the report against the current tree.
  - **Evidence:** 23 Go packages; 47,334 non-test LOC; 32,140 test LOC; production `nolint:exhaustruct` count 399 before the lint-policy update.
- [x] **BASE-02** Recalculate the review score from current evidence.
  - **Evidence:** 8.6/10 = `0.2×9.2 + 0.2×9.0 + 0.2×8.4 + 0.15×8.4 + 0.2×8.5 + 0.05×8.3`.

## Phase 1 — Public API correctness

- [x] **API-01** Add an explicit typed-builder-to-Converter path.
  - **Implementation:** `Converter.WithGlobal` clones the supplied `GlobalSettings`.
  - **Proof:** `api_test.go` verifies Letter geometry, title propagation, snapshot independence, and nil receiver behavior.
- [x] **API-02** Cover the root `PdfGlobalOptions` surface.
  - **Proof:** root tests cover page size, margins, title, copies, outline, shrinking, background, compression, relative links, dotted settings, snapshots, and Converter integration.
- [x] **API-03** Enforce the nil-builder policy.
  - **Decision:** existing fluent methods panic on a nil builder as a programmer error; `WithSetting` returns errors for invalid/unknown values.
  - **Proof:** `TestPdfGlobalOptionsNilReceiverPanics` and `TestPdfGlobalOptionsValidationPaths`.
- [x] **API-04** Route validation failures to `OnError`.
  - **Implementation:** PDF and image converter preflight errors invoke the configured callback.
  - **Proof:** `TestConverterValidationErrorsReachOnError` covers both modes.
- [x] **API-05** Validate typed and dotted page-size input through the canonical parser.
  - **Implementation:** `WithPageSize`, `WithSetting`, and `GlobalSettings.Set` use `ParsePageSize`; `PageSize` is the sole named geometry field.
  - **Proof:** invalid page-size panic/error tests and settings parity tests.
- [x] **API-06** Establish copies and margin validation.
  - **Decision:** copies below one are rejected at builder, public request, and engine boundaries; negative header/footer margins remain valid layout inputs.
  - **Proof:** invalid copies and negative-margin tests.
- [x] **API-07** Protect settings descriptor parity.
  - **Implementation:** the descriptor parity test checks every global descriptor has setter/getter behavior; root dotted-key tests cover the public round trip.
  - **Proof:** `internal/settings/reflect_parity_test.go`, `api_test.go`.

## Phase 2 — CLI and boundary hygiene

- [x] **BOUND-01** Define and test documentation-flag boundaries.
  - **Decision:** `--help=x` remains an invalid boolean; `--no-help` and `-xyz` are rejected as unknown options.
  - **Proof:** `TestDocFlagBoundaryBehavior`, `TestShortFlagClusterIsRejected`.
- [x] **BOUND-02** Bound validation ownership.
  - **Decision:** direct engines validate once; public/app paths use two deliberate gates (side-effect-free preflight plus engine defense-in-depth). The image CLI no longer has a third production adapter.
  - **Proof:** preflight ordering/output non-creation tests in `internal/app`, `internal/convert`, and `internal/imageout`.
- [x] **BOUND-03** Unify public and engine sentinels where matching is promised.
  - **Implementation:** public PDF output/outline errors alias `convert.ErrMissingOutput`/`ErrMissingOutlineOutput`; copies aliases `convert.ErrInvalidCopies`.
  - **Proof:** `errors.Is` assertions in root/app/convert tests.
- [x] **BOUND-04** Wire canonical app and engine sentinels.
  - **Implementation:** app nil-command, convert nil-request/command/context, load nil-loader/context, prepare nil-loader/context, imageout nil-request/command/context, layout context, paint context, and render context use `internal/errs` aliases.
  - **Proof:** nil-boundary tests and targeted race suite.
- [x] **BOUND-05** Audit context guards and context-containing state.
  - **Decision:** boundary guards remain at ownership transitions; `containedctx` structures and cancellation wrappers retain lifecycle comments because they carry active cancellation state.
  - **Proof:** build/vet/race gates and existing cancellation tests.

## Phase 3 — Performance measurements and fixes

- [x] **PERF-01** Benchmark border-heavy image overlay lookup.
  - **Proof:** `BenchmarkRoundedBorderLineOverlay` reports 32/128/512 cards: 8.3/13.9/57.6 µs, 0 B/op, 0 allocs/op.
- [x] **PERF-02** Benchmark repeated-resource PDF conversion with distinct metrics.
  - **Proof:** `BenchmarkRepeatedResourcePDF`: 500 pages ≈418.8 ms/op, 2.72 MB PDF/op, 61,673,472-byte process HWM, 417.7 MB Go allocation traffic/op, 1,459,761 allocs/op.
- [x] **PERF-03** Measure header/footer placeholders and page-count parity.
  - **Proof:** `BenchmarkHeaderFooterPlaceholders` records 2/10/50 actual pages and 7.35/1.24/2.83 ms one-iteration timings; page-count test passes.
- [x] **PERF-04** Benchmark font lookup/name loading and supersample pooling.
  - **Proof:** cached names ≈4.9 µs/136 B/7 allocs; glyph lookup ≈3.5 µs/64 B/2 allocs; small/large pool ≈0.33/9.07 ms.
- [x] **PERF-05** Replace rounded-border backward scanning.
  - **Implementation:** immediate predecessor validation is O(1).
  - **Proof:** fixture/image tests and PERF-01.
- [x] **PERF-06** Deduplicate repeated PNG XObjects per document.
  - **Implementation:** SHA-256 plus grayscale-mode key reuses the indirect image resource while preserving unique draw names.
  - **Proof:** `TestRepeatedPNGImageReusesXObject`.
- [x] **PERF-07** Remove the unconditional 16 MiB supersample allocation.
  - **Implementation:** zero-value pool with exact required capacity on a miss.
  - **Proof:** small/large pool benchmark and image tests.
- [x] **PERF-08** Cache font names and remove the registry face-slice copy.
  - **Implementation:** `Font.LoadNames` uses `sync.Once`; `FindWithGlyph` scans stable faces under `RLock`.
  - **Proof:** registry benchmarks and race suite.
- [x] **PERF-09** Measure pagination/header/footer/shaping/font-stack tradeoffs and close the design decision.
  - **Decision:** retain semantically coupled pagination, placeholder relayout, shaping, and font-stack state after representative timing/page-count evidence; no unsafe simplification was justified.
  - **Proof:** general 500-page benchmark, header/footer benchmark, and full tests.

## Phase 4 — Architecture and maintainability

- [x] **ARCH-01** Remove production engine-to-CLI imports.
  - **Implementation:** `internal/app.RunPDF` and `RunImage` own command translation/output lifecycle; legacy same-package callers are `_test.go` compatibility adapters.
  - **Proof:** `rg` import audit and build/test gates.
- [x] **ARCH-02** Canonicalize nil-context errors.
  - **Implementation:** package-local aliases point to `internal/errs.ErrNilContext` and related canonical sentinels.
  - **Proof:** errors-is boundary tests and targeted race suite.
- [x] **ARCH-03** Configure `exhaustruct` under `enable-all`.
  - **Implementation:** `.golangci.yml` sets `exhaustruct.check-exported: true`; intentional literal suppressions remain documented.
  - **Proof:** `make lint` passes.
- [x] **ARCH-04** Audit read-only lookup maps and immutable representations.
  - **Decision:** package-private descriptor/flag maps are initialized once and never exposed or mutated; the already immutable page-size table remains an array. Replacing hot lookup maps would add complexity without improving the current safety contract.
  - **Proof:** source audit, descriptor parity test, and lint gate.
- [x] **ARCH-05** Audit complexity suppressions.
  - **Decision:** retain suppressions for bounded state machines and explicit zero-value literals; the new image/PDF code has focused comments and passes the full lint policy.
  - **Proof:** `make lint`, build, vet, and tests.

## Phase 5 — Closure and rerating

- [x] **CLOSE-01** Run focused tests after each implementation slice.
  - **Proof:** API, app, CLI, settings, convert, imageout, outline, PDF, and benchmark-specific tests all pass.
- [x] **CLOSE-02** Run repository test and lint gates.
  - **Proof:** `GOCACHE=/tmp/gowk-go-cache make test` and `GOCACHE=/tmp/gowk-go-cache make lint` pass.
- [x] **CLOSE-03** Run build, vet, full tests, and race validation.
  - **Proof:** `go build ./...`, `go vet ./...`, `go test ./...`, and `go test -race -count=1 ./internal/pdf ./internal/imageout ./internal/convert ./internal/layout` pass.
- [x] **CLOSE-04** Record distinct benchmark metrics.
  - **Proof:** timings, Go allocation traffic, allocation counts, PDF bytes/op, fetch counts/bytes, and process HWM/RSS are recorded in the parent report and PERF rows above.
- [x] **CLOSE-05** Publish the current review score.
  - **Proof:** parent report matrix is rerated to 8.6/10 with visible arithmetic.

## Final state

- [x] No checklist row is pending.
- [x] No checklist row is deferred.
- [x] No partial or open status markers remain in this checklist.
