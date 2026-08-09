# 2026-08-09 - Go Design Patterns Architecture Review

> **Parent:** `plans/performance/2026-08-09/500-page-allocation-architecture.md` - current performance and architecture ledger
> **Status:** Implementation complete; source verification complete
> **Estimated effort:** 3-5 small implementation slices plus focused validation
> **Scope:** project-wide architecture, with primary inspection of `internal/layout` and `internal/convert` plus their API, loader, app, and performance adapters
> **Standards:** `golang-design-patterns` v1.1.5, `codebase-design`, repo `ponytail` guidance
> **Constraints:** implementation allowed; no Git commands or Go commands

---

## Overview

This is a critical architecture slice review, not a claim that the renderer is
unsafe or non-functional. The audit inspected constructors, initialization,
interfaces, lifecycle and ownership, context propagation, external calls,
resource bounds, error flow, and the depth of the seams between layout,
conversion, PDF, and raster output.

Four independent read-only subagent tracks completed: layout/convert
architecture, public API and adapter boundaries, performance/allocation, and
cross-cutting architecture. Their findings were reconciled against current
numbered source reads. No subagent modified files, ran Git commands, or ran
`go test`; source and checked-in benchmark artifacts remain the evidence
boundary.

## Four-track synthesis

| Review track | Score | Material evidence | Weight |
|---|---:|---|---:|
| Layout and conversion architecture | 9.0 | Private lifecycle contexts, canonical resource delegation, bounded page/copy/style growth, scoped workspace release | 0.30 |
| Public API and adapter boundaries | 9.0 | Exported matchable errors, cloned ownership, validated proxy construction, pre-output validation | 0.25 |
| Performance and allocation architecture | 8.6 | Sparse index cap, explicit SmartWidth ceiling, cold/warm budget gate, canonical benchmark snapshots | 0.25 |
| Cross-cutting architecture | 8.8 | Shared paint-order policy covers body/header/footer; no production file exceeds 2,000 lines | 0.20 |
| **Project-wide synthesis** |  | **9.0×0.30 + 9.0×0.25 + 8.6×0.25 + 8.8×0.20 = 8.86 → 8.9/10** | **1.00** |

The synthesis is the headline rating for this review. The direct
design-patterns slice below is higher because it excludes some public-boundary
and performance-evidence debt; the lower project-wide score is intentional.

## Executive Summary

The implementation has addressed the review's actionable architecture debt:
conversion lifecycle state is grouped, request/resource boundaries validate and
own their inputs, expansion is bounded, page-island ownership is scoped, and
body/header/footer visual ordering now uses one policy. The remaining lower
scores are deliberate design decisions: PDF layout still owns font metrics, and
legacy nil-context adapters remain compatibility shims at explicit boundaries.

The completed high-leverage work is:

1. route resource fetches through `load.ResourceContext` while retaining only
   compatibility mirrors at the conversion boundary;
2. replace repeated argument plumbing with private run/object contexts;
3. expose header/footer failures as an explicit result and compatibility warning
   policy;
4. enforce object/page/copy/stylesheet and sparse-index bounds before growth;
5. preserve a small concrete paint-order seam rather than adding a speculative
   one-implementation renderer interface.

## Finding closure

All DP-01 through DP-12 rows are implemented. Source proof includes
`internal/convert/convert.go:36-48,264-318,618-645,881-930`,
`internal/convert/prepare.go:17-179`, `internal/layout/paint_order.go:1-54`,
`internal/layout/paint.go:45-115,470-560`,
`internal/convert/hf.go:423-510,641-735`,
`internal/load/load.go:274-374`, `api.go:23-31,160-190,293-322`, and
`testdata/golden/benchmarks/benchmark-results.txt:1-12,57-70`. The performance
gate validates two-run timing/page-count behavior; cold/warm PDF bytes are not
required to be identical because the command is intentionally reused. The
PDF/layout seam is intentionally concrete because layout consumes PDF font
metrics; this is a resolved architectural choice, not deferred work.

## Dependencies

- Parent ledger: `plans/performance/2026-08-09/500-page-allocation-architecture.md`.
- Review standards: `golang-design-patterns` v1.1.5, `golang-code-style` v1.2.2,
  `codebase-design`, and the local Ponytail review guidance.
- Implementation proof remains deferred: the user explicitly prohibited Git
  commands and `go test`; the checklist therefore records required future gates
  without claiming they passed.

## Findings

### DP-01 - High: conversion orchestration is a shallow, stateful mega-module

**Evidence:** `internal/convert/convert.go:198-364` contains the full loading,
layout, TOC fixed-point, page reorder, outline, links, copies, headers/footers,
metadata, and final write pipeline in one function. The function itself carries
`//nolint:gocognit,contextcheck,cyclop,funlen,lll`.

**Rule mapping:** design-patterns requires simple explicit orchestration and
early error flow; codebase-design says a deep module should hide complexity
behind a small interface and preserve locality. A single function that knows
every phase is a wide interface for maintainers, even though its Go signature
has only four parameters.

**Remediation:** keep `Run` as the single public/private entry seam, but extract
four private phase functions over a small `runState`: load body/TOC objects,
assemble TOC and pages, finalize links/HF, and write the document. Each phase
should return its error and own only the state it mutates. Do not introduce an
interface unless a second concrete adapter actually varies.

**False-positive caveat:** this is intentionally a linear pipeline, and
`convert` is the application coordinator. Extraction should improve locality,
not create a speculative clean-architecture layer.

### DP-02 - High: dependency plumbing violates the small-parameter rule

**Evidence:** `internal/convert/convert.go:607` (`initTOCState`) has eight
parameters; `internal/convert/convert.go:640` (`renderObject`) has eight;
`internal/convert/convert.go:778` (`bodyLayoutOpts`) accepts seven values; and
`internal/convert/page_islands.go:108-113` accepts thirteen values. Similar
wide seams exist at `internal/convert/prepare.go:109` and
`internal/convert/hf.go:605`.

**Rule mapping:** golang-code-style recommends no more than four parameters;
golang-design-patterns recommends an options/configuration object when a
constructor or operation has grown; codebase-design treats a seam with nearly
all implementation details exposed as shallow.

**Remediation:** construct one private `renderContext` after `Run` validates
the request. Give it loader, font, registry, document, request, log, and
context. Construct one private `objectRenderContext` for per-object values.
Keep `layout.Options` as the explicit data contract at the layout seam, but
build it from the private context. Validate option invariants once.

**False-positive caveat:** grouping values is justified only if the group has a
coherent lifecycle. A generic `Options` bag or exported service interface would
make the seam worse.

### DP-03 - High: duplicate resource-context seams reduce locality

**Evidence:** `internal/load/load.go:82-112` already defines the narrow
`load.ResourceContext` with `Fetch`. `internal/convert/prepare.go:16-79` defines
another `convert.ResourceContext` with exported `Loader`, `Base`, and `Load`,
then delegates to `CollectSheets` and `MergeFontFaces`. The prepared document
retains the duplicate at `internal/convert/prepare.go:93-101`; the image and HF
paths consume it at `internal/convert/convert.go:670-680` and
`internal/convert/hf.go:315-338`.

**Rule mapping:** one seam should have one interface/contract; a shallow
pass-through adapter creates maintenance locality without leverage. The
Ponytail ladder also says to reuse an existing helper rather than reimplementing
one a few files away.

**Remediation:** make `load.Loader.ForResource` the canonical resource seam.
Change `PreparedDocument.Resources` to `load.ResourceContext`, and keep
conversion-specific stylesheet/font aggregation as explicit functions in
`convert`. Delete the duplicate wrapper and its duplicate missing-loader
behavior. If conversion needs a combined operation, make that operation a
single private preparation function rather than a second resource type.

**False-positive caveat:** the convert wrapper was likely introduced to share
PDF/image preparation. Preserve that sharing; the recommendation is to move
the shared policy down, not to duplicate it in both adapters.

### DP-04 - High: PDF types cross the layout seam, limiting real adapter depth

**Evidence:** `internal/layout/layout.go:57-74` puts `*pdf.Font`, `*pdf.FaceSet`,
and `*pdf.Registry` directly in layout options. `internal/layout/paint.go:62-71`
and `internal/layout/paint.go:432-440` expose PDF document/page/content types.
The raster adapter does share `layout.PaintOrder` at
`internal/imageout/imageout.go:329-337`, but PDF-specific fields and painting
remain inside `layout`.

**Rule mapping:** codebase-design says two adapters make a seam real, while
golang-design-patterns says dependencies should be injected and modules should
be testable through a small interface. The current seam shares policy but not a
backend-neutral display-list contract.

**Remediation:** do not add a broad renderer interface. First isolate the
backend-neutral display-list/result types and ordering/pagination policy. Move
PDF drawing into a small PDF adapter package only after the font-metrics
dependency is measured and a second consumer needs the same contract. Keep the
current `PaintOrder` seam as the minimal first step.

**False-positive caveat:** the layout engine legitimately uses PDF font metrics
for text measurement. A wholesale split now could increase complexity; this is
a medium-term seam decision, not a license to add abstraction immediately.

### DP-05 - High: header/footer failures are intentionally hidden from `Run`

**Evidence:** `internal/convert/hf.go:601-605` declares
`drawHeadersFooters` without an error return. Loading and drawing failures are
converted to warning logs at `internal/convert/hf.go:672-692`. The caller at
`internal/convert/convert.go:354-355` cannot distinguish a successful final
document from one missing header/footer content.

**Rule mapping:** expected operational errors should be returned, not silently
absorbed; error flow should remain explicit and testable.

**Remediation:** return a structured error or a typed warning collection from
the HF phase. Make the policy explicit in `Request` or settings: strict mode
returns the first failure, compatibility mode records warnings and returns a
successful result. Preserve existing compatibility behavior only through an
explicit adapter choice.

**False-positive caveat:** the source comment says best-effort HF behavior is
intentional because body content is already painted. That is a valid product
policy, but it should be represented as data rather than encoded as a void
function.

### DP-06 - Medium: nil contexts are normalized repeatedly to unbounded contexts

**Evidence:** `internal/convert/convert.go:203-205`,
`internal/convert/prepare.go:44-51` and `109-116`,
`internal/layout/layout.go:571-595`, and `internal/layout/paint.go:62-71` turn
nil contexts into `context.Background()`.

**Rule mapping:** context should be explicit and propagated; external work
should have a timeout. Repeated nil acceptance makes an absent deadline look
like a supported runtime mode.

**Remediation:** normalize nil once at the outer compatibility adapter, or
reject nil in internal context-aware functions. Keep legacy `Layout`, `Paint`,
and `RunPDF` wrappers if needed, but make the `*Context` forms require a real
context. Document that HTTP timeout remains a loader safety net, not a complete
conversion deadline.

**False-positive caveat:** `internal/load` still applies request-level timeouts
and `http.NewRequestWithContext` at `internal/load/load.go:633-703`; this is not
a claim that network calls are currently unbounded.

### DP-07 - Medium: document/page/style growth is only partly bounded

**Evidence:** `internal/convert/convert.go:400-417` appends one owner per
logical page; `internal/convert/convert.go:546-579` duplicates every page for
every requested copy with no local budget; `internal/convert/convert.go:918-965`
accumulates every stylesheet and only emits a soft warning at 25,000 rules.
`internal/layout/layout.go:788-807` caps the initial op capacity at 1<<20,
but later slices and PDF page duplication are not covered by that cap.

**Rule mapping:** limit pools, queues, buffers, and user-controlled expansion;
fail early at configuration boundaries.

**Remediation:** add validated budgets to `Request.Validate` for object count,
copies, logical pages, stylesheet rules, and final page count. Make the limits
observable in errors and configurable only where the CLI already exposes a
policy. Validate before `materializeCopies`, not after PDF pages have expanded.

**False-positive caveat:** the loader already caps body size and redirects at
`internal/load/load.go:36-41` and `659-675`; do not duplicate those limits in
`convert`.

### DP-08 - Medium: borrowed workspace ownership is a manual lifecycle contract

**Evidence:** `internal/layout/layout.go:108-138` says `Workspace` is not
concurrent and requires callers to call `Release` after all consumers finish.
`internal/convert/page_islands.go:121-150` manually follows that protocol.
`Release` clears the result's slices and box tree, so an early release is a
use-after-reuse hazard and a missed release retains the whole result.

**Rule mapping:** resource ownership and lifecycle should be explicit and
hard to misuse; design for testability and bounded memory.

**Remediation:** keep the workspace private to the island renderer and expose a
single helper that paints, projects navigation, and releases in one scope. If
the public `WithWorkspace` contract remains, add a result-state guard or make
the borrowed result type impossible to retain after release.

**False-positive caveat:** this is an internal package and the current comments
are unusually clear. It is a risk boundary, not a confirmed leak or race.

### DP-09 - Positive evidence: initialization and basic cleanup are sound

**Evidence:** no production `func init()` was found under `internal/layout` or
`internal/convert`. File and HTTP response cleanup is immediate at
`internal/load/load.go:570-577` and `633-652`. `Request.Validate` rejects
missing output at `internal/convert/convert.go:106-122`, and `Run` wraps output
write errors at `internal/convert/convert.go:359-361`.

**Rule mapping:** this satisfies the no-implicit-init, `defer Close`, and
explicit-error-contract expectations.

**Caveat:** this positive finding does not offset the separate HF error policy,
copy growth, or duplicate resource seam findings.

### DP-10 - High: public error and mutable-policy contracts are not deep enough

**Evidence:** `api.go:23-28` declares `errNoPageObjectsAdded`, `errEmptyHTML`,
and `errNoInputPageAdded` as unexported even though the public comments promise
that callers can use `errors.Is`. `internal/load/load.go:244-269` exposes and
duplicates mutable loader policy, including an aliased `Allow` map. Malformed
proxy parsing at `internal/load/load.go:286-289` is ignored rather than
returned from construction.

**Remediation:** export stable sentinel or typed errors where matching is part
of the API contract; clone caller-owned policy maps; and make loader
construction validate proxy configuration and return an error. Keep the
compatibility surface small and avoid adding an options framework without a
second concrete implementation.

### DP-11 - High: the shared paint-order policy is incomplete across adapters

**Evidence:** `internal/layout/paint_order.go:5` and the raster call at
`internal/imageout/imageout.go:329-337` establish a shared ordering policy, but
body/header-footer paths still contain direct operation loops at
`internal/layout/paint.go:492` and `internal/convert/hf.go:452`. This leaves
two real adapters with a locality gap: a paint-order change can be correct for
raster and body PDF output while remaining wrong for header/footer output.

**Remediation:** route every paint band through one `PaintBandContext` and the
shared `layout.PaintOrder` policy, while preserving links/metadata as a
separate explicit projection. Add differential fixtures for body, header, and
footer ordering before moving any PDF-specific types.

### DP-12 - High: performance boundaries are measured but not yet architectural

**Evidence:** `internal/layout/paint_flow.go:223-307` grows page-index slices
from the computed page number, so adversarial sparse page coordinates can cause
large allocations. `internal/imageout/imageout.go:213-268` can perform up to
eight SmartWidth layouts, while the benchmark artifact contains separately
labeled count-1 and count-3 snapshots in
`testdata/golden/benchmarks/benchmark-results.txt:3-6,18-21` rather than one
canonical comparison. `internal/convert/perf_test.go:72-132` also documents
three runs while executing two and checks only a page-count/time budget.

**Remediation:** add bounded sparse-index construction, isolate SmartWidth
cost in a benchmark, and normalize benchmark command/count/toolchain metadata
before using the result as an architecture gate. Treat `B/op` as allocation
traffic, not resident memory.

## Small snippet for the phase checklist

Current code already has the right shape for the first safe seam:

```go
func PaintOrder(ops []Op) []int {
	idx := make([]int, len(ops))
	for i := range ops {
		idx[i] = i
	}
	sortPaintIndices(ops, idx)
	return idx
}
```

The next change should preserve this minimal policy seam and deepen the caller
around it, rather than adding a renderer interface with one implementation.

## Phase-wise checklist

### Phase 0: Review contract and evidence

- [x] Record scope, standards, constraints, and direct-inspection method in
  this file. Proof: current source reads; no Go code changed.
- [x] Confirm no production `init()` in the target packages. Proof: focused
  `rg` search; no matches.
- [x] Record existing positive bounds and cleanup in `internal/load`. Proof:
  `internal/load/load.go:36-41`, `570-577`, `633-675`.
- [x] Reconcile four independent read-only review tracks against current
  source and checked-in benchmark artifacts. Proof: Four-track synthesis and
  DP-01 through DP-12; no code or test commands were run.

### Phase 1: Deepen conversion seams

- [x] Replace `convert.ResourceContext` with the canonical load resource seam.
  Affected paths: `internal/convert/prepare.go:16-79`,
  `internal/load/load.go:82-112`. Rule: DP-03. Proof: fetches delegate to
  `load.ResourceContext`; compatibility mirrors no longer own fetch policy.
- [x] Introduce a private run/object context to reduce parameter plumbing in
  `convert.go`, `page_islands.go`, and `hf.go`. Rule: DP-01/DP-02. Proof
  `runContext` and `objectRenderContext` are private; callers use grouped
  lifecycle state and no new exported abstraction was added.
- [x] Decide the minimal PDF/raster seam after measuring font-metric coupling.
  Affected paths: `internal/layout/layout.go:57-74`, `paint.go:62-71`.
  Rule: DP-04. Proof: concrete PDF font-metric coupling is retained by
  decision; body/raster/header/footer ordering is shared through `PaintOrder`.

### Phase 2: Make failure and lifecycle policy explicit

- [x] Return or collect header/footer failures instead of hiding them behind a
  void phase. Rule: DP-05. Proof: `hfDrawResult` collects failures and the
  compatibility adapter emits explicit warnings.
- [x] Normalize or reject nil contexts at one boundary. Rule: DP-06. Proof
  `beginPaintContext`, signal-aware CLI contexts, and context checks in load,
  layout, and paint establish one owned compatibility boundary.
- [x] Encapsulate `Workspace.Release` in the island renderer. Rule: DP-08.
  Proof: island rendering uses scoped `defer workspace.Release(res)` after
  navigation extraction.

### Phase 3: Bound expansion and close

- [x] Define and validate object, copy, page, stylesheet-rule, and final-output
  budgets before expensive allocation. Rule: DP-07. Proof: request validation,
  page-plan preflight, copy preflight, and stylesheet rule limits are present.
- [x] Bound sparse pagination-index growth and isolate SmartWidth's repeated
  layout cost. Affected paths: `internal/layout/paint_flow.go:223-307`,
  `internal/imageout/imageout.go:213-268`. Rule: DP-12. Proof: page index
  allocation is capped at 16,384 and SmartWidth is capped at eight passes.
- [x] Canonicalize benchmark snapshots by command, count, toolchain, and
  workload before using them as a performance gate. Affected path:
  `testdata/golden/benchmarks/benchmark-results.txt`. Rule: DP-12. Proof:
  Snapshots A/B/C carry commands, counts, workloads, toolchains, and units.
- [x] Route body and header/footer paint bands through the shared paint-order
  policy. Affected paths: `internal/layout/paint_order.go`,
  `internal/layout/paint.go:492`, `internal/convert/hf.go:452`. Rule: DP-11.
  Proof: `PaintBandContext` uses `PaintOrder`; link annotations remain a
  separate source-order projection and focused policy coverage was added.
- [x] Export documented matchable errors, clone loader policy maps, and reject
  malformed proxy configuration during construction. Affected paths: `api.go`,
  `internal/load/load.go:244-289`. Rule: DP-10. Proof: exported sentinels,
  deep settings copies, cloned loader policy, validated proxy construction,
  and pre-output request validation are implemented.
- [x] Complete the relevant source-level validation gates under the command
  constraint. Proof: symbol/reference, ownership, import, line-count, and
  changed-path inspections completed; Git and Go commands were prohibited.
- [x] Re-score this slice from current source evidence; the new arithmetic is
  recorded below and does not inherit the prior architecture headline score.

## Rating

**Headline project-wide rating: 8.9/10.** The four-track arithmetic appears
above. The direct design-patterns slice is also 8.9/10:

| Area | Score | Weight | Weighted |
|---|---:|---:|---:|
| Constructors and API shape | 9.0 | 0.15 | 1.350 |
| Globals and initialization | 9.4 | 0.10 | 0.940 |
| Lifecycle and resource ownership | 9.0 | 0.15 | 1.350 |
| Context and timeout propagation | 8.8 | 0.15 | 1.320 |
| Bounded resources | 9.0 | 0.15 | 1.350 |
| Error flow and failure policy | 8.8 | 0.15 | 1.320 |
| Seams, depth, and testability | 8.7 | 0.15 | 1.305 |
| **Total** |  | **1.00** | **8.935 → 8.9/10** |

The remaining deductions are for intentional PDF/font coupling, compatibility
nil-context shims, and the absence of runtime validation imposed by the
no-Go-command constraint; no implementation row remains open.

## Validation boundary

Go source and review-plan files were modified. No Git commands and no Go
commands were run. Source-level inspection was the only permitted validation
boundary for this implementation turn.
