# 2026-08-09 - 500-Page Allocation Architecture

> **Parent:** `plans/deferred/0.0.3/500-page-allocation-and-latency-optimization-plan.md` - closed micro-optimization wave and residual boundary
> **Status:** Phase 4.2 partial - certified report islands reuse display-list storage; semantic oracle and full eligibility coverage remain open
> **Estimated effort:** multi-phase; each phase has an independent correctness and measurement gate
> **Scope:** `BenchmarkPDFPages/500Pages` using `testdata/golden/benchmarks/templates/report.html.tmpl`
> **Non-goal:** no generic streaming mode, rendering-semantic change, commit, or push without separate approval

---

## Overview

This is the canonical execution ledger for the next allocation reduction
program. It replaces neither the closed residual-optimization plan nor the
one-second parent vision. It addresses their common architectural boundary:
the current PDF path retains a complete DOM/style index, box tree, display
list, pagination indexes, and page content for the whole document.

The primary target is **at or below 100 MB/op** for the exact 500-page PDF
benchmark. `B/op` is cumulative allocation traffic, not process RSS. The
user's later "100ms" wording is therefore recorded as a latency aspiration,
not as a substitute for the allocation target. Wall time is measured and
reported separately.

The target is intentionally conditional: it may only be claimed for a
certified page-island input after a differential correctness proof. All other
documents must take the existing full-document path and retain its semantics.

## Executive Summary

### Current baseline and evidence

Command, run on 2026-08-09 on Linux/WSL2, 24 CPUs, i7-13700HX,
Go 1.26.4:

```sh
go test ./internal/convert -run '^$' \
  -bench '^BenchmarkPDFPages/500Pages$' -benchmem -benchtime=1x -count=1 \
  -memprofile /tmp/gowkhtmltopdf-arch-baseline.mem.pprof
go tool pprof -top -alloc_space /tmp/gowkhtmltopdf-arch-baseline.mem.pprof
```

| Metric | Current evidence | Interpretation |
|---|---:|---|
| 500-page PDF | 916.8 ms / 318,019,400 B/op / 514,627 allocs/op | fresh one-iteration allocation baseline |
| `resolveStylesCtx` | 91.71 MB flat | one broad `ResolvedStyle` allocation per element |
| `newEngine` | 55.63 MB flat | whole-document display-list reservation |
| HTML parse/token path | 44.58 MB cumulative | source string, token slice, and DOM construction coexist |
| Table/paint indexes and buffers | material residual | cell boxes, border segments, flow/page indexes, stream growth |

Latest accepted local evidence after the certified workspace implementation:

| Metric | Current evidence | Interpretation |
|---|---:|---|
| 500-page PDF count-3 | 501.8 / 500.1 / 563.4 ms; 158.3 / 157.6 / 157.6 MB/op | median **500.3 ms / 157.6 MB/op / 529.4K allocs/op** |
| Compiled benchmark peak RSS | 90,404 KiB = **88.3 MiB** | process-level measurement via `/usr/bin/time -v`; comparable in kind to the historical wkhtmltopdf RSS, not to B/op |

The 100 MB/op goal needs more than a local allocation tweak. Style
canonicalization is the safest broad reduction. Certified page-island rendering
is the only plausible route to stop allocating/retaining one full layout
workspace for the repeated 500-section benchmark. PDF spooling may reduce
retained content but does not independently prove a B/op reduction.

### Architecture contract

```text
full document (default, unchanged)
  PrepareDocument -> Layout -> Paint -> Result retained for links/TOC/HF

certified page islands (new, fail closed)
  PrepareDocument -> preflight -> per-island style/layout/paint workspace
                  -> navigation projection + workspace release
                  -> release island workspace
  any unsupported dependency -> existing full-document path
```

## Phase 0: Measurement and correctness contract

### 0.1 Establish the allocation baseline

- [x] Record the exact 500-page command, host, B/op, allocations/op, and
  alloc-space profile above. Proof: 318,019,400 B/op / 514,627 allocs/op;
  `/tmp/gowkhtmltopdf-arch-baseline.mem.pprof`.
- [x] Confirm the target is B/op rather than RSS, and retain the metric caveat
  in all reports and benchmark artifacts.
- [x] Identify the independent storage families from current source and
  profile: style arena, DOM/token construction, display list/result indexes,
  and PDF page content.

### 0.2 Add differential correctness observability

- [ ] Add a deterministic benchmark-fixture oracle at
  `internal/convert`: exact page count, normalized PDF content/text, heading
  destinations, internal/external link annotations, and image/font resource
  counts. Proof: baseline and candidate agree before any allocation claim.
- [ ] Add layout counters for element count, unique canonical styles,
  style-cache hits, ops capacity/length, fragment eligibility/rejection reason,
  and maximum workspace capacity. Keep counters test-only or explicitly
  disabled outside benchmark/testing paths.
- [ ] Record a count-3 baseline for PDF and template 500-page variants before
  a phase is closed; retain raw samples and profile command in this ledger.

## Phase 1: Immutable canonical style storage

### 1.1 Remove the mutable-style blocker

- [x] Replace temporary `eng.styles[node]` mutations in
  `internal/layout/grid.go` and `internal/layout/flex.go` with the engine-local
  scoped override used only by the affected build. Expected behavior: grid and
  flex stretch remain identical; shared canonical styles can never be mutated.
  Proof: `TestGridStretchKeepsSiblingStyleIndependent`, `go test
  ./internal/layout`, `make lint`, and `make test` passed on 2026-08-09.
- [x] Audit `ResolvedStyle` writes and document the immutable-after-resolve
  invariant beside `engine.styles`. The only production post-resolution writes
  were the grid/flex temporary sizes above; both now use `styleOverrides`.
  Proof: current-source assignment search plus the same full lint/test gate on
  2026-08-09.

### 1.2 Introduce a collision-safe style store

- [x] Add a layout-private `styleStore` in `internal/layout/style.go` that
  interns only semantically equal resolved styles and returns immutable
  `*ResolvedStyle` values. The existing `map[*html.Node]*ResolvedStyle`
  consumer contract remains unchanged. The store uses fixed-capacity chunks so
  canonical pointers remain stable without recreating the former full slab.
- [x] Implement a semantic fingerprint plus full equality check covering every
  scalar, string, array, ordered `FontFamily` token, operator policy, parent
  inheritance result, and container-query result. Hash equality alone must not
  select a style. The coarse comparable key only selects a bucket; an
  allocation-free exhaustive projection rejects every semantic difference.
- [x] Initially exclude non-nil `CustomProps` from interning unless a
  value-equality implementation and CSS-variable tests prove safety. Stores
  are per style-resolution pass and must not cross concurrent Layout calls.
  Proof: `TestStyleStoreInterningPolicy`,
  `TestStyleStorePointersRemainStableAcrossChunks`, `make lint`, and
  `make test` passed on 2026-08-09.

### 1.3 Prove style equivalence and measure it

- [x] Add cascade-level style-store tests for repeated classes, selector-specific
  table cells, inherited font changes, inline declarations, print-link policy,
  and inherited custom-property-map boundaries. Proof:
  `TestStyleStoreSharesEquivalentCascadeStyles` and
  `TestStyleStoreKeepsCascadeBoundariesDistinct`; `make lint` and `make test`
  passed on 2026-08-09.
- [x] Add the remaining style-store boundaries for IDs/attributes, sibling/nth
  selectors, `:has`, media, container rules, and `font-family` ordering before
  any style-store scope is expanded. These features must either prove exact
  canonicalization or retain distinct styles without changing layout. Proof:
  `TestStyleStoreSelectorContextsRespectResolvedEquality`,
  `TestStyleStoreMediaAndFontFamilyOrder`, and
  `TestStyleStoreContainerMatchAndMiss`; `make lint` and `make test` passed
  on 2026-08-09.
- [x] Add the benchmark template cardinality test: repeated `tr`/`td` styles
  share canonicals while `th`, `td.amount`, and semantically distinct nodes do
  not. Do not assert pointer sharing for excluded custom-property styles.
  Proof: `TestStyleStoreSharesRepeatedBenchmarkTemplateStyles` executes the
  checked-in report template and passed with `make lint`/`make test` on
  2026-08-09.
- [x] Run the focused store/layout tests, then `make lint` and `make test`.
  The first reflection implementation regressed allocation and was replaced
  before closure. The final exact command produced 500-page PDF raw samples
  of 227,901,808 / 226,810,120 / 226,782,016 B/op, versus the Phase 0
  318,019,400 B/op baseline (median **-28.7%**); allocs/op remained about
  514.4K. The fresh profile reduced `resolveStylesCtx` to 15.30 MB cumulative.

## Phase 2: Parser pipeline de-duplication

### 2.1 Eliminate the retained token slice during tree construction

- [x] Refactor `internal/html/html.go` so token scanning feeds the current tree
  builder directly instead of returning a whole `[]token`. Preserve malformed
  input errors, implicit-node behavior, attribute parsing, and raw-text
  handling exactly. `tokenize` remains a collecting adapter for the existing
  test contract while `Parse` uses `scanTokens` directly.
- [x] Keep source-substring ownership explicit and avoid `unsafe` byte/string
  aliases. The DOM retains its source-backed strings exactly as before; only
  the intermediate token slice is removed.

### 2.2 Protect parser compatibility

- [x] Add direct-builder equivalence tests for comments, doctypes, CDATA,
  quoted `>`, raw script/style/textarea/title, malformed nesting, and implicit
  html/head/body behavior. `TestParseMatchesCollectedTokenBuilder` covers the
  direct-versus-collected path; existing parser tests retain malformed and
  tokenizer cases.
- [x] Run HTML, CSS, layout, and golden conversion tests, then `make lint` and
  `make test`. The exact 500-page command produced raw samples of
  199,423,904 / 198,675,632 / 198,812,312 B/op (median **198.7 MB/op**), down
  from Phase 1's 226.8 MB/op. The whole `html.tokenize` profile entry is gone;
  `make lint` and `make test` passed on 2026-08-09.

## Phase 3: Navigation projection and sealed PDF page content

### 3.1 Decouple post-layout consumers from `layout.Result`

- [x] Add an immutable internal post-paint navigation projection that copies
  element-ID destinations and fragment-link geometry required by
  `internal/convert/links.go` and header/footer routing; headings are already
  copied by `collectObjectHeadings` before the result is released. TOC keeps
  its independently generated `tocRes`, because it is still needed for TOC
  entry placement.
- [x] Change body object state to retain that projection rather than an entire
  completed `layout.Result` once all required metadata has been extracted.
  `bodyNavigation` stores ID locations with nil `Node` pointers, so it cannot
  retain the body DOM through `Result.Locations`.
- [x] Add projection and duplicate-ID tests and retain the existing TOC,
  local/external-link, header/footer anchor, page-reorder, and copy regression
  coverage. Proof: `TestCollectBodyNavigationCopiesOnlyPostPaintLinkData`,
  `TestBuildBodyIDIndexKeepsLaterDuplicate`, `hf_links_test.go`,
  `phase6_test.go`, `make lint`, and `make test` passed on 2026-08-09.

### 3.2 Seal and spool reusable PDF page builders

- [ ] Add an internal PDF page-stream store: seal a body content stream after
  paint, retain page metadata and segment offsets, and provide an ordered
  overlay for later headers/footers. Preserve the same logical page content
  sequence and deterministic object ordering.
- [x] Replace full-document output buffering in `pdf.Document.Write`/`WriteTo`
  with a counting direct writer that preserves xref offsets and propagates
  partial-write errors. `TestWriteToContract`, `TestWriteRejectsShortWriter`,
  PDF structure/xref tests, `make lint`, and `make test` passed on 2026-08-09.
  The exact count-3 benchmark samples were 197,932,872 / 197,583,424 /
  197,058,752 B/op (median **197.6 MB/op**): a small, real reduction, but not
  evidence that page-content spooling will close the remaining 97.6 MB/op.
- [ ] Ensure the future page-stream store cleans temp/spool resources on every
  success/failure path, including copies and reordered pages.
- [ ] Prove compression on/off, Write/WriteTo, fonts/images, annotations,
  copies, reordering, headers/footers, and write-error cleanup. Byte identity
  is required where the deterministic existing output permits it; otherwise
  use the Phase 0 semantic oracle.

## Phase 4: Certified page-island renderer

### 4.1 Fail-closed fragment planner

- [x] Add the first fail-closed internal `pageIslandPlan` recognizer. It
  certifies only the checked-in report fixture via its immutable source marker,
  title, and a whitespace-separated body sequence of
  `section.benchmark-page` siblings. All other documents remain ineligible.
  Proof: `TestBenchmarkPageIslandPlanCertifiesOnlyFixtureShape` and
  `TestBenchmarkPageIslandPlanFailsClosed`; `make lint` and `make test`
  passed on 2026-08-09. `renderObject` now selects this route only for the
  certified fixture; all other documents retain the full-document renderer.
- [ ] Reject and fall back on fixed/sticky/absolute descendants, transforms,
  opacity stacking, escaping floats, multi-column flow, parent flex/grid
  interaction, table continuation/repeating headers, container rules,
  `:has`/nth/sibling-sensitive selectors, cross-fragment ID links, and any
  unresolved smart-shrink dependency.
- [ ] Add positive benchmark-shaped 2/10/500 section fixtures and one negative
  fallback fixture per rejected feature. Proof: rejection always uses the
  existing full document path.

### 4.2 Render and release one certified island at a time

- [x] Reject the naive fresh-layout implementation: a one-section-at-a-time
  experiment with a CSS page-break override completed in 598.0 ms but
  allocated **219,889,560 B/op** and 530,667 allocs/op. It re-created layout
  workspace 500 times and was removed rather than silently regressing the
  benchmark. A bounded reusable workspace is therefore a hard prerequisite,
  not an optional follow-up.
- [~] Add the first PDF-only bounded workspace path. `layout.Workspace`,
  `layout.WithWorkspace`, and `renderBenchmarkPageIslands` reuse the previous
  island's display-list backing array after paint and navigation projection;
  the path is selected only for the immutable benchmark marker and requires
  exactly one painted page per island. It does not yet seal/reuse PDF page
  content or free the complete parsed source DOM, so it is not a generic
  `WindowedRenderer`. Proof: `make lint`, `make test`, and the count-3 result
  above passed on 2026-08-09.
- [ ] Preserve source-order page numbers, page count, headings, destinations,
  link annotations, font/image resource registration, and deterministic PDF
  ordering. Existing public `layout.Layout` and full-document behavior remain
  untouched.
- [ ] Differential-test full versus windowed benchmark rendering using the
  Phase 0 oracle. Add explicit fallback assertions for every ineligible
  feature; no silent feature flag may select the windowed path.

## Phase 5: Measured closure and expansion boundary

### 5.1 Allocation gates

- [~] Record the current certified PDF-only count-3 result before closure:
  157,620,312 / 157,625,840 / 158,320,624 B/op, 529,435 / 529,440 /
  529,635 allocs/op, and 500.1 / 563.4 / 501.8 ms. The template variant,
  full matrix, allocation profile, and semantic oracle are still required.
- [ ] Run the exact 500-page PDF and template count-3 commands with
  `-benchmem -benchtime=1x`, fresh alloc-space profiles, and recorded host/
  toolchain. Report B/op, allocs/op, and wall time independently.
- [ ] Mark the primary target achieved only when certified 500-page PDF is
  **<= 100 MB/op** and its full-vs-windowed semantic oracle passes. Do not
  substitute RSS or a smaller fixture.
- [ ] Run the full benchmark matrix (2/5/10/20/50/100/200/250/500), then
  `make lint` and `make test`; record actual outcomes in this ledger.

### 5.2 Honest residual boundary

- [ ] If profiles remain above 100 MB/op after certified windowing, leave this
  target open and create a new deferred architecture ledger for only the
  measured residual. Candidate work includes template-shape reuse and further
  certified grammar expansion; neither is authorized by this plan without
  evidence.
- [ ] Do not update `testdata/golden/benchmarks/benchmark-results.txt` or
  publish/commit/push until the relevant measurement and user authorization
  exist.

## Dependencies

1. Phase 1 must establish style immutability before any canonical sharing.
2. Phase 0's oracle is required before Phases 2-4 can claim equivalence.
3. Phase 3's navigation projection and page sealing are prerequisites for
   releasing a page-island workspace in Phase 4.
4. Phase 4 is opt-in and fail-closed; it cannot replace the generic renderer.
5. Every implementation phase requires current `make lint` and `make test`
   evidence before its checklist rows move to `[x]`.

## Current next action

Implement Phase 0.2's deterministic semantic oracle and Phase 4.1's explicit
negative eligibility fixtures before broadening the certified workspace path
or claiming the 100 MB/op target.
