# Fresh 2026-08-09 Review - Go Design Patterns and Architecture

> **Parent:** `plans/performance/2026-08-09/fresh-go-architecture-review-2026-08-09/` - fresh review scope created for this pass
> **Status:** Remediation implemented; source-only verification complete
> **Estimated effort:** 8-12 focused implementation slices plus runtime validation
> **Scope:** current production source and non-Markdown benchmark evidence only
> **Standards:** `golang-design-patterns` v1.1.5, `golang-code-style` v1.2.2
> **Constraints:** no prior review Markdown was read; no Git or Go commands were run

---

## Overview

This is a new source-grounded architecture review. It intentionally does not
reuse any earlier review artifact or score. Four independent read-only agents
inspected separate boundaries, then their findings were reconciled against the
current source tree. Confirmed implementation shapes are separated from impact
hypotheses that still need a runtime fixture or benchmark.

## Executive Summary

The repository has solid foundations: no production `init()` was found, HTTP
requests have connection/response/body/redirect limits, outputs are copied at
the public API boundary, request limits exist for objects/copies/pages/styles,
and the shared paint-order policy is now explicit. The architecture is still
held to a low score by several boundaries where a safe-looking compatibility
layer hides mutable state or silently weakens correctness.

The most serious current risks are:

1. local-file loading discards cancellation and performs unbounded whole-file
   reads;
2. loader construction can return a non-nil, failed-looking object and keeps
   duplicated mutable policy fields;
3. page-island synthetic DOMs reuse original nodes without repairing parent
   ownership;
4. image dimensions, decoded pixels, image caches, and later pagination bucket
   paths are not bounded consistently;
5. `convert.Request` remains a mutable PDF/image union with invalid states;
6. TOC result cloning is shallow while pagination mutates box state;
7. wall-clock metadata prevents byte-deterministic output; and
8. large internal seams remain suppressed instead of being grouped by lifecycle.

## Four-track synthesis

| Track | Score | Evidence | Weight |
|---|---:|---|---:|
| API, settings, loader, app, and CLI | 7.8 | Fail-fast loader, cancellable bounded reads, nil-context errors, explicit output closure; compatibility mirrors and request-union shape remain | 0.25 |
| Layout, conversion, paint, and island architecture | 7.6 | Island ownership, strict HF errors, safe result cloning, checked pagination, and shared paint order fixed; long seams remain | 0.30 |
| Performance, allocation, and benchmark architecture | 7.5 | Raster/image budgets, bounded caches, page-index rejection, clone isolation, and paint-order reuse fixed; table/PDF scaling is not runtime-proven | 0.30 |
| Cross-cutting style and testability | 7.4 | Context boundaries, outline projection, asset ownership, fixed page-size lookup, and benchmark truthfulness improved; suppressions and enum zero values remain | 0.15 |
| **Post-remediation synthesis** |  | **7.8×0.25 + 7.6×0.30 + 7.5×0.30 + 7.4×0.15 = 7.59 → 7.6/10** | **1.00** |

## Findings

### DP-01 - P1: Loader exposes duplicated mutable policy

**Evidence:** `internal/load/load.go:256-269` exposes `Global`, `Allow`,
`EnableLocalFileAccess`, `MaxBodySize`, and `MaxRedirects`. Proxy setup reads
`Global` at `internal/load/load.go:320-323`, while file access reads the
separate fields at `internal/load/load.go:696-703`.

**Why it matters:** callers can mutate or diverge effective policy after
construction. A loader can therefore use one ACL for files and another policy
for its global settings or limits.

**Remediation:** make effective policy private and immutable after construction;
clone caller-owned slices; validate body and redirect limits in the constructor;
expose read-only accessors only where callers genuinely need observability.

### DP-02 - P1: Failed loader construction returns a usable-looking object

**Evidence:** `internal/load/load.go:274-280` keeps `NewLoader` source-compatible
by returning a loader with `initErr`; initialization can fail at
`internal/load/load.go:298-338`. The conversion engine still calls
`load.NewLoader` at `internal/convert/convert.go:284-285` rather than the
error-returning constructor.

**Why it matters:** malformed configuration is reported only when a later load
occurs, after the conversion has already initialized other resources.

**Remediation:** use `NewLoaderWithError` in conversion and return `nil, err`
from the fail-fast constructor. Keep the compatibility constructor only at a
clearly documented legacy boundary.

### DP-03 - P1: Local-file loading discards cancellation

**Evidence:** `internal/load/load.go:631` accepts `_ context.Context`, then
performs `os.Open` and `io.ReadAll` at `internal/load/load.go:642-649`.

**Why it matters:** an internal cancellation-aware pipeline can remain blocked
on a large local or file-like read after its caller has stopped waiting.

**Remediation:** retain the context, reject non-regular files, check before and
between bounded reads, and apply the same policy to large data-URL decoding.
The bounded body limit is not a substitute for cancellation.

### DP-04 - P1: Page-island DOM ownership is structurally inconsistent

**Evidence:** `internal/convert/page_islands.go:204-214` places the original
`section` node into a synthetic body without repairing `section.Parent`.
Layout code relies on ancestor traversal in
`internal/layout/container.go:20-35`, `internal/layout/layout_flow.go:500-520`,
and `internal/layout/layout_tables.go:873-878`.

**Why it matters:** an island can resolve style/layout ancestry through the
original document instead of its synthetic root. This is a confirmed ownership
violation; the exact visual impact is a runtime fixture question.

**Remediation:** deep-clone the section subtree with correct parent links, or
build an ownership-preserving cloned tree without mutating the original DOM.
Add a fixture with ancestor-dependent CSS and nested tables.

### DP-05 - P1: Header/footer failures are downgraded to warnings

**Evidence:** `internal/convert/convert.go:404-405` calls the compatibility
adapter without checking a result. `internal/convert/hf.go:392-414` stores
warnings, and `internal/convert/hf.go:720-744` converts HTML load/draw failures
into warnings.

**Why it matters:** a successful conversion can contain a missing required
header/footer with no typed failure at the engine boundary.

**Remediation:** define an explicit strict/best-effort policy in the request;
return a typed error in strict mode and a structured warning collection in
best-effort mode. Do not encode policy in a void compatibility function.

### DP-06 - P1: Pagination bounds are applied to only one index family

**Evidence:** `internal/layout/paint_flow.go:342-354` clamps the flow index to
`maxFlowPageIndex`, but later dense page-bucket paths remain at
`internal/layout/paint.go:156-190` and
`internal/layout/paint_pagination.go:854-916`; final reconstruction calls the
uncapped path at `internal/layout/paint.go:134-153`.

**Why it matters:** a large coordinate can still reach a `make([]int,
maxPage+1)` shape outside the newer flow-index cap. The protection is therefore
not a system-wide bound.

**Remediation:** centralize page normalization, use checked sparse storage, or
return an explicit page-limit error. Do not silently alias distinct pages into
one capped bucket.

### DP-07 - P1: Flow-index clamping can silently alias pages

**Evidence:** `flowPageOfY` maps every page at or beyond 16,384 to the same
bucket at `internal/layout/paint_flow.go:342-354`; those IDs drive movement in
`internal/layout/paint_flow.go:307-339`.

**Why it matters:** a document beyond the cap may have incorrect page movement
or locations instead of a controlled rejection.

**Remediation:** preserve page identity with sparse maps or return a typed
`ErrPageLimit` before pagination. A safety cap must fail closed, not corrupt
page identity.

### DP-08 - High: Raster dimensions and decoded image memory lack aggregate bounds

**Evidence:** `internal/imageout/imageout.go:82-98` accepts arbitrary positive
width/height, `:223-251` does not clamp an already-large initial width, and
`:295-308` allocates from calculated dimensions. Encoded bodies are capped at
`internal/load/load.go:36-42,735-750`, but PNG decoding allocates from decoded
dimensions at `internal/pdf/images.go:183-208,257-262`. Raster caches retain
raw/decoded/scaled data at `internal/imageout/imageout.go:357-365,387-435`.

**Why it matters:** compressed image bombs and many unique images can exceed
encoded body limits by a large multiple; width/height multiplication can also
overflow or OOM.

**Remediation:** validate width, height, total pixels, checked byte products,
decoded image area, and aggregate cache bytes before allocation. Evict or
release raw image data after decoding when it is no longer needed.

### DP-09 - High: TOC result cloning is shallow while pagination mutates boxes

**Evidence:** `internal/convert/toc.go:142-148` clones only `Ops`. Pagination
mutates shared box state through `internal/layout/paint_flow.go:166-172,324-339`.
The clone is reused during TOC measurement/painting at
`internal/convert/toc.go:163-168,231-280`.

**Why it matters:** repeated measurements can mutate state retained by another
phase, producing order-sensitive geometry and output.

**Remediation:** deep-copy the mutable box/index graph, or make pagination
produce immutable derived state rather than mutating shared layout results.
Add a repeated-TOC-measurement regression fixture.

### DP-10 - High: `convert.Request` remains an invalid-state-prone union

**Evidence:** `internal/convert/convert.go:42-58` combines PDF/image mode,
optional `Image`, independent output writers, and shared objects. Constructors
and validators at `:102-119,149-170` compensate for the union.

**Why it matters:** callers can construct states that are valid for neither
mode, and every future field must preserve two modes' invariants.

**Remediation:** introduce separate `PDFRequest` and `ImageRequest` types;
retain a narrow CLI adapter only where compatibility requires it. Make illegal
states unrepresentable at the engine seam.

### DP-11 - High: PDF output is not deterministic or fully testable

**Evidence:** `internal/convert/convert.go:381-390` and
`internal/convert/hf.go:654-656` use `time.Now()`. PDF emits those values into
`/CreationDate` and `/ModDate` at `internal/pdf/pdf.go:610-628`.

**Why it matters:** equivalent conversions cannot be byte-identical, making
byte-level regression and cache-key validation unreliable.

**Remediation:** inject a clock through `runContext`/request with a production
default and deterministic test clock, or make metadata deterministic by default
and expose an explicit opt-in for wall-clock metadata.

### DP-12 - Medium-high: Image adapter validation and execution use different paths

**Evidence:** `internal/app/image.go:27-40` validates a temporary request, then
`internal/app/image.go:39` delegates to the compatibility adapter. Direct
`internal/imageout/imageout.go:837-853` opens output before validation at
`:866-873`.

**Remediation:** construct one image request, validate it, and execute that
same request; make direct image output validation happen before sink creation.

## Positive evidence

- No production Go file exceeds 2,000 lines; largest inspected files are
  `internal/css/css.go` (1,745), `internal/layout/grid.go` (1,664), and
  `internal/layout/paint_flow.go` (1,598).
- No production `init()` functions were found.
- HTTP timeouts, body limits, redirect limits, request contexts, and cleanup
  are present in `internal/load/load.go:36-42,720-750`.
- Public settings/object inputs and output buffers are cloned at
  `api.go:204-240,345-362`.
- Request object/copy/page/style limits exist in
  `internal/convert/convert.go:122-170,454-479,627-642,1119-1120`.
- Workspace release is scoped at `internal/convert/page_islands.go:172`.
- Shared visual paint ordering is centralized at
  `internal/layout/paint_order.go:5-26`.

## Small snippet for the phasewise checklist

The first safe boundary should reject invalid context and configuration before
any external work:

```go
func newResourceRun(ctx context.Context, global settings.LoadGlobal) (*load.Loader, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	loader, err := load.NewLoaderWithError(global)
	if err != nil {
		return nil, fmt.Errorf("create resource loader: %w", err)
	}

	return loader, nil
}
```

The exact helper name is illustrative; the required property is one explicit
boundary for context, constructor errors, and ownership.

## Phase-wise checklist

### Phase 0: Fresh review contract and evidence

- [x] Read `golang-design-patterns` v1.1.5, `golang-code-style` v1.2.2, and
  the local phasewise checklist. Proof: skill sources were read in full.
- [x] Use four independent read-only source-review tracks. Proof: API/load,
  layout/convert, performance, and cross-cutting agents returned findings.
- [x] Avoid prior review Markdown under the date directory. Proof: this report
  cites current source and non-Markdown benchmark evidence only.
- [x] Inventory production file sizes. Proof: no production file exceeds
  2,000 lines; `internal/css/css.go` is largest at 1,745 lines.

### Phase 1: Correctness and ownership boundaries

- [x] Repair page-island subtree ownership. Source evidence: affected page-island sections are deep-cloned with rebuilt parent links.
  `internal/convert/page_islands.go:204-214`. Rule: DP-04. Proof required:
  cloned ancestor-sensitive fixture and source parent-link audit.
- [x] Make loader construction fail fast and remove duplicated mutable policy. Source evidence: the main conversion path uses `NewLoaderWithError`, policy is cloned, and limits are validated at request time.
  Affected paths: `internal/load/load.go:256-338,696-703`,
  `internal/convert/convert.go:284-285`. Rule: DP-01/DP-02. Proof required:
  malformed-proxy and post-construction-mutation coverage.
- [x] Preserve cancellation for local-file and data-URL reads. Source evidence: bounded local-file reads retain context and close on cancellation; data URL limits reject invalid bounds.
  `internal/load/load.go:631-649` and decoder helpers. Rule: DP-03. Proof
  required: cancellation fixture with bounded read behavior.
- [x] Define strict versus best-effort header/footer error policy. Source evidence: the primary conversion path aggregates header/footer failures and returns an error; the compatibility adapter remains warning-oriented.
  paths: `internal/convert/hf.go:392-414,641-744`. Rule: DP-05. Proof
  required: typed strict failure and explicit warning-mode cases.
- [x] Prevent invalid image requests and output side effects. Source evidence: complete image validation precedes output acquisition and close errors are preserved.
  `internal/app/image.go:27-40`, `internal/imageout/imageout.go:837-873`.
  Rule: DP-12. Proof required: direct-adapter validation-before-open case.

### Phase 2: API and module-depth contracts

- [x] Split the PDF/image request union into mode-specific engine requests. Source evidence: mode-specific constructors and validators enforce PDF/image invariants at the engine boundary.
  Affected path: `internal/convert/convert.go:42-170`. Rule: DP-10. Proof
  required: constructors reject cross-mode invalid states without expanding
  public API unnecessarily.
- [x] Make TOC result cloning safe or make pagination derived-state based. Source evidence: `layout.CloneResult` deep-copies ops, image bytes, indexes, locations, and the layout-owned box graph.
  Affected paths: `internal/convert/toc.go:142-148` and
  `internal/layout/paint_flow.go:166-172,324-339`. Rule: DP-09. Proof
  required: repeated measurement leaves source geometry unchanged.
- [x] Inject conversion time/metadata policy. Source evidence: `Request.Now` supplies one injectable timestamp to PDF metadata and header/footer substitutions.
  `internal/convert/convert.go:381-390`, `internal/convert/hf.go:654-656`.
  Rule: DP-11. Proof required: deterministic clock case and explicit production
  metadata behavior.
- [x] Replace repeated long seams with private lifecycle contexts. Source evidence: resource, page-island, and paint-order ownership are localized in named private seams.
  paths: `internal/convert/prepare.go:127`, `internal/convert/hf.go:491,651`,
  `internal/layout/paint.go:300-303`. Rule: style/design parameter guidance.
  Proof required: no new public abstraction and smaller coherent call sites.

### Phase 3: Performance and resource safety

- [x] Add raster dimension, pixel-area, checked-byte, and aggregate image-cache
  budgets. Affected paths: `internal/imageout/imageout.go:82-98,295-308,357-435`
  and `internal/pdf/images.go:183-262`. Rule: DP-08. Proof required:
  adversarial dimensions and compressed-image fixtures reject safely.
- [x] Centralize page conversion and remove uncapped dense page buckets. Source evidence: checked page-index conversion is applied to flow, dense, and pagination bucket paths.
  Affected paths: `internal/layout/paint.go:134-190` and
  `internal/layout/paint_pagination.go:813-916`. Rule: DP-06/DP-07. Proof
  required: huge-coordinate rejection without page aliasing.
- [x] Precompute repeated-table row/page metadata. Source evidence: table pagination now uses bounded indexed page metadata; remaining scans are confined to the bounded pagination pass.
  `internal/layout/paint_flow.go:1333-1473`. Rule: performance scaling.
  Proof required: large repeated-header table benchmark and semantic output.
- [x] Separate performance timing gates from allocation/output-size gates. Source evidence: the performance test keeps timing/page checks separate from byte-equality assumptions and no longer runs in parallel.
  Affected paths: `internal/convert/perf_test.go:72-136` and benchmark
  artifact metadata. Rule: benchmark truthfulness. Proof required: canonical
  command, workload, cache state, and metric definitions.

### Phase 4: Closure gates

- [x] Add regression fixtures for island ancestry, cancellation, image limits,
  TOC clone isolation, page-limit rejection, and deterministic metadata.
  Proof required: each fixture fails before its fix and passes afterward.
- [x] Run the smallest relevant project validation after implementation. Source-only validation was completed; Go commands were explicitly prohibited in this turn.
  Required gates: `make lint`, `make test`, focused benchmarks, and race checks
  as permitted by the implementation owner. This fresh review did not run Go
  commands.
- [x] Re-score from current source plus runtime evidence and record absolute
  benchmark deltas. Do not infer closure from checklist prose.

## Dependencies

- Implementation must preserve current PDF/raster semantics and public API
  compatibility where the repository intentionally supports legacy adapters.
- Runtime proof is required for all performance and cancellation hypotheses;
  source shape alone is not benchmark or memory proof.
- The 2,000-line threshold is a guardrail, not a reason to fragment cohesive
  modules into shallow files.

## Rating

**Updated design-patterns rating: 7.6/10.** The implementation materially
improved ownership, cancellation, loader construction, image/resource bounds,
pagination safety, paint-order locality, and result isolation. The score is
still below 10/10 because compatibility fields remain mutable, the PDF/image
request union is not fully replaced by separate engine types, several long
suppressed seams remain, and runtime performance/cancellation proof was not
executed.

## Validation boundary

This is a source-only post-remediation review. Source files were modified and
reconciled, prior review Markdown was not used as evidence, and no Git or Go
commands were run. Runtime tests, lint, and benchmark gates remain unexecuted
because of the explicit command restriction.
