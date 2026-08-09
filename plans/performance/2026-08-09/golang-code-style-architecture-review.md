# 2026-08-09 - Go Code Style Architecture Review

> **Parent:** `plans/performance/2026-08-09/500-page-allocation-architecture.md` - current performance and architecture ledger
> **Status:** Implementation complete; source verification complete
> **Estimated effort:** 2-4 small cleanup slices plus focused validation
> **Scope:** project-wide architecture, with primary inspection of `internal/layout` and `internal/convert` plus their API, loader, app, and performance adapters
> **Standards:** `golang-code-style` v1.2.2, `golang-design-patterns` v1.1.5, `codebase-design`, repo `ponytail` guidance
> **Constraints:** implementation allowed; no Git commands or Go commands

---

## Overview

This report reviews code clarity and architecture together. It treats linter
suppression as a claim requiring a narrow reason, not as proof that a large
function is healthy. The review checks line breaking, parameter shape, variable
and slice declarations, control flow, function responsibility, file
organization, and whether the code exposes a deep or shallow module seam.

Four independent read-only subagent tracks completed: layout/convert
architecture, public API and adapter boundaries, performance/allocation, and
cross-cutting architecture. Their findings were reconciled against current
numbered source reads. No subagent modified files, ran Git commands, or ran
`go test`; source and checked-in benchmark artifacts remain the evidence
boundary.

## Four-track synthesis

| Review track | Style/architecture score | Material evidence | Weight |
|---|---:|---|---:|
| Layout and conversion architecture | 8.6 | Private contexts, lifecycle seams, scoped release, and focused responsibility boundaries | 0.35 |
| Public API and adapter boundaries | 8.8 | Exported matchable errors, cloned ownership, validated construction, signal-aware adapters | 0.20 |
| Performance and allocation code shape | 8.5 | Bounded indexes, explicit SmartWidth ceiling, cold/warm budget gate and honest evidence | 0.20 |
| Cross-cutting locality and policy | 8.7 | Shared paint-order traversal, parser state grouping, no production file exceeds 2,000 lines | 0.25 |
| **Project-wide synthesis** |  | **8.6×0.35 + 8.8×0.20 + 8.5×0.20 + 8.7×0.25 = 8.645 → 8.6/10** | **1.00** |

This remains stricter than a cosmetic style score: API ownership, performance
evidence quality, and policy locality affect how safely the code can be
changed. The 2,000-line threshold is satisfied, and the remaining deductions
are for intentional compatibility shims, retained fixed-point suppressions,
and runtime validation unavailable under the no-Go-command constraint.

## Executive Summary

The implementation has addressed the actionable style debt: dependency-heavy
helpers now use private parser/render contexts, standard/local imports are
grouped, private accumulators have intentional capacity, comments and
benchmark run counts are truthful, ownership is explicit at boundaries, and
paint-order control flow has one named home. The inspected production tree
contains no file over the repository's 2,000-line base (the largest are
`internal/css/css.go` at 1,745 lines, `internal/layout/grid.go` at 1,664,
`paint_flow.go` at 1,591, and `paint_pagination.go` at 1,459).

The completed critical style actions are:

- reduce parameter-heavy seams without introducing public abstraction;
- split orchestration by lifecycle phase, not arbitrary line count;
- make import, accumulator, ownership, and benchmark conventions consistent;
- replace broad suppression debt where the owning seam was changed; and
- retain only documented exceptions where the pipeline is genuinely clearer
  whole.

## Finding closure

All CS-01 through CS-11 rows are implemented. Source proof includes the
private `parseState` in `internal/cli/cli.go:85-210`, render contexts in
`internal/convert/convert.go:197-226,881-930`, import groups in
`internal/layout/style_cascade.go:3-9` and `layout_images.go:3-10`, cached
settings/page-size tables in `internal/settings/reflect.go:131-185` and
`pagesize.go:10-43`, shared paint traversal in
`internal/layout/paint.go:470-560`, and explicit cold/warm benchmark documentation
in `internal/convert/perf_test.go:72-132`.

## Dependencies

- Parent ledger: `plans/performance/2026-08-09/500-page-allocation-architecture.md`.
- Review standards: `golang-code-style` v1.2.2, `golang-design-patterns` v1.1.5,
  `codebase-design`, and the local Ponytail review guidance.
- Implementation proof remains deferred: the user explicitly prohibited Git
  commands and `go test`; the checklist therefore records required future gates
  without claiming they passed.

## Findings

### CS-01 - High: parameter-heavy functions expose implementation details

**Evidence:** `internal/convert/convert.go:607` has eight parameters;
`internal/convert/convert.go:640` has eight; `internal/convert/convert.go:778`
has seven; `internal/convert/page_islands.go:108-113` has thirteen;
`internal/convert/prepare.go:109` has eight; and
`internal/convert/hf.go:605` has eight.

**Rule mapping:** golang-code-style recommends four or fewer parameters and
suggests an options struct when the signature is growing. The design-patterns
skill also requires context first and favors explicit construction. These
signatures force every caller to know font, registry, document, request,
logging, geometry, and policy details simultaneously.

**Remediation:** create a private `renderContext` or `objectRenderContext` with
one constructor at the orchestration seam. Keep fields grouped by lifecycle;
do not export it. Use a small explicit argument list for the actual operation,
for example `renderObject(ctx, object)` where the private object owns the
per-run dependencies.

**False-positive caveat:** a context struct is not automatically clearer. Use
one only for values that travel together and share a lifetime; keep pure
geometry helpers as small functions with direct arguments.

### CS-02 - High: large functions are being exempted instead of narrowed

**Evidence:** `internal/convert/convert.go:198` suppresses
`gocognit`, `contextcheck`, `cyclop`, `funlen`, and `lll`; `renderObject` at
`internal/convert/convert.go:640` suppresses `cyclop`, `funlen`, `gocognit`,
and `lll`; `CollectSheets` at `internal/convert/convert.go:905` suppresses
`gocognit`, `cyclop`, `funlen`, and `lll`; and
`internal/convert/hf.go:605` suppresses `gocognit`, `cyclop`, `funlen`, and
`lll`.

**Rule mapping:** “clear is better than clever”; functions should be short and
focused, and long signatures are a signal to reduce parameters. A suppression
can be right for a fixed-point or hot path, but four separate complexity
exemptions on an application phase indicate missing locality.

**Remediation:** first extract phase boundaries with named results and early
returns. Then retain a suppression only on the irreducible fixed-point loop or
hot scanner, with a comment naming the invariant and the reason extraction
would make it less clear. Remove `lll` by reducing the seam rather than merely
wrapping lines.

**False-positive caveat:** `internal/layout/paint_pagination.go:445-503` is a
legitimate fixed-point algorithm with a bounded ten-iteration guard. Splitting
every helper there could damage the algorithm's readability; review the
orchestration files first.

### CS-03 - Medium: import grouping is inconsistent with standard Go layout

**Evidence:** `internal/layout/style_cascade.go:3-7` places local `gowkhtmltopdf`
imports before the standard `strings` import without a blank group.
`internal/layout/layout_images.go:3-8` similarly mixes `encoding/binary` with
local HTML and standard string/strconv imports in one group.

**Rule mapping:** the style skill emphasizes standard Go clarity and linter
enforcement (`gofmt`, `goimports`, `gofumpt`). Import grouping is a small issue,
but inconsistent grouping increases review noise and makes dependency direction
less visible.

**Remediation:** use standard-library imports first, a blank line, then local
modules. Apply only to affected files; do not reformat unrelated files.

**False-positive caveat:** `gofmt` alone does not always create goimports-style
groups. This is a low-risk consistency issue, not a compiler or runtime defect.

### CS-04 - Medium: slice declarations are safe but inconsistent with the stated style

**Evidence:** `internal/convert/convert.go:571` declares `var order []int`,
`internal/convert/convert.go:918` declares `var sheets []*css.Stylesheet`, and
`internal/layout/paint.go:122` declares `var fixedIdx []int` before append.

**Rule mapping:** the requested code-style skill says slices and maps should be
initialized explicitly, especially when nil serialization would surprise an
API consumer.

**Remediation:** where the slice is an internal append-only accumulator, use a
named empty slice with a useful capacity if the bound is known, such as
`order := make([]int, 0, expected)`. Do not invent a speculative capacity.
Preserve nil as a deliberate “no output” signal when callers depend on it.

**False-positive caveat:** nil slices are valid Go and append-safe. These are
consistency findings, not panic findings; the rule is most important for
exported JSON/data contracts, not private page assembly.

### CS-05 - Medium: control flow and error policy are split across nested closures

**Evidence:** `internal/convert/hf.go:617-701` loops pages and defines a nested
`draw` closure which performs content checks, lazy loading, cache mutation,
error-to-warning conversion, and drawing. `internal/convert/convert.go:217-225`
also closes over progress/log/request policy. The source uses `//nolint:nestif`
at `internal/convert/hf.go:661`.

**Rule mapping:** handle errors first, keep the happy path flat, and scope
variables to the smallest useful block. Closures are helpful for tiny local
policies, but this one is a second state machine inside a per-page loop.

**Remediation:** extract `drawHeaderFooter(ctx, pageState, value, isHeader)`
with a result that distinguishes skipped, drawn, and failed. Keep the page loop
responsible only for choosing owner/page and aggregating the result.

**False-positive caveat:** the closure does avoid duplicating header/footer
calls. Extract the behavior only if the returned failure policy is explicit;
otherwise this is a cosmetic move.

### CS-06 - Medium: file organization follows feature history more than type locality

**Evidence:** `internal/layout/layout.go` is 1,257 lines and contains public
options/results, the private engine, caching, context, construction, and block
building. `internal/convert/convert.go` is 1,146 lines and contains request
contracts, the Run coordinator, page planning, rendering, stylesheet loading,
font-face loading, and link URI resolution. The repository's size threshold is
not crossed, but multiple independent responsibilities are co-located.

**Rule mapping:** one primary type per file when it has significant methods;
group related declarations; create deep modules with small interfaces and
strong locality.

**Remediation:** split by lifecycle seam: `convert/request.go`,
`convert/run.go`, `convert/page_plan.go`, and `convert/resources.go`; in layout,
keep the public display-list contract together and move private engine helpers
only when their dependency cluster is coherent. Avoid splitting merely to make
line counts smaller.

**False-positive caveat:** Go package locality is more important than file
count. The requested 2,000-line base is a useful guardrail, not a requirement
to fragment files under it.

### CS-07 - Low: comments explain many exceptions, but not all define an upgrade path

**Evidence:** `internal/layout/layout.go:224` suppresses `containedctx` with a
good explanation; `internal/layout/style_cascade.go:125`, `491`, and `543`
explain immutable static tables; and `internal/layout/paint.go:221` explains
why a page pass remains one function. In contrast, the broad conversion
suppression at `internal/convert/convert.go:198` describes a linear pipeline
but does not identify the next extraction boundary.

**Rule mapping:** when ignoring a style rule, add a comment. The comment should
make the exception legible and prevent permanent debt.

**Remediation:** update only broad suppressions during their owning refactor to
name the invariant, the reason the function remains whole, and the next
remediation gate. Do not add comments that merely restate the linter name.

**False-positive caveat:** comments are not a substitute for a refactor. This
finding is intentionally low severity because several existing exceptions are
well documented.

### CS-08 - Positive evidence: local control flow and declarations are often clear

**Evidence:** `internal/layout/layout.go:584-629` validates nil roots and
contexts before constructing the engine; `internal/convert/convert.go:106-122`
validates request sinks before work; `internal/load/load.go:570-577` defers file
close immediately; and `internal/layout/paint.go:260-289` checks cancellation
inside page and operation loops.

**Rule mapping:** this meets early-return, immediate cleanup, context-first,
and explicit error-flow expectations.

**Caveat:** positive local style does not eliminate the larger seam and
parameter problems in the findings above.

### CS-09 - High: benchmark documentation and executable gates do not align

**Evidence:** `internal/convert/perf_test.go:72-132` says three cold/warm runs
in its comments but executes two iterations and checks only elapsed time/page
count. `testdata/golden/benchmarks/benchmark-results.txt:3-6,18-21` contains
separate count-1 and count-3 snapshots with different headline numbers, so a
reader cannot infer one canonical current baseline from the file alone.

**Rule mapping:** good style includes truthful comments, reproducible evidence,
and names that make state and units clear. Stale run-count prose is a
maintenance defect, and ambiguous benchmark snapshots create review noise.

**Remediation:** make comments match the loop, record command/count/toolchain
metadata beside each result, and publish one explicitly selected baseline with
absolute and percentage deltas. Keep allocation bytes distinct from RSS.

### CS-10 - High: ownership and validation style is inconsistent at boundaries

**Evidence:** `api.go:162-186` clones object settings, but `api.go:293-296`
aliases global settings. `internal/load/load.go:261-269` aliases `Allow`, while
`internal/load/load.go:286-289` ignores proxy parse errors. `internal/app/pdf.go:48-58`
acquires an output sink before request validation. These are not formatting
issues; they make ownership and failure behavior harder to read at the call
site.

**Rule mapping:** keep ownership obvious, validate before side effects, and
return errors at the boundary where the invalid input is known.

**Remediation:** clone caller-owned maps, make global settings immutable or
copy-on-use, validate proxy configuration in the constructor, and build/validate
the request before opening output. Add focused contract coverage with explicit
names for aliasing and validation order.

### CS-11 - Medium-high: the shared paint-order rule is not visually local

**Evidence:** `internal/layout/paint_order.go:5` centralizes the policy and
`internal/imageout/imageout.go:329-337` consumes it, but direct loops remain at
`internal/layout/paint.go:492` and `internal/convert/hf.go:452`. The same visual
rule is therefore split between a named helper and adapter-specific loops.

**Rule mapping:** related policy should have one obvious home; duplicated
control flow should be removed when two real adapters depend on the same rule.

**Remediation:** use one small `PaintBandContext` operation for body,
header, and footer bands. Keep link projection separate and name any intentional
exception at its call site. Add body/header/footer differential fixtures.

## Small snippet for the phase checklist

The preferred style for a narrowed operation is already visible in the layout
boundary:

```go
if root == nil {
	return nil, errors.New("layout: nil root")
}
if err := ctx.Err(); err != nil {
	return nil, fmt.Errorf("layout: context: %w", err)
}
```

The cleanup target is to apply this guard-clause clarity to the larger
conversion phase functions without hiding dependencies in a public abstraction.

## Phase-wise checklist

### Phase 0: Style baseline and evidence

- [x] Audit production file sizes in `internal/layout` and `internal/convert`.
  Proof: largest inspected file is `internal/layout/grid.go` at 1,664 lines;
  no file exceeds the 2,000-line base.
- [x] Search for long production signatures and broad `nolint` suppressions.
  Proof: exact rows recorded in CS-01 and CS-02.
- [x] Inspect imports, guards, closures, slice declarations, and lifecycle
  comments. Proof: exact rows recorded in CS-03 through CS-08.
- [x] Reconcile four independent read-only review tracks against current
  source and checked-in benchmark artifacts. Proof: Four-track synthesis and
  CS-01 through CS-11; no code or test commands were run.

### Phase 1: Narrow signatures and phase responsibilities

- [x] Introduce private run/object context values for the repeated dependency
  groups. Affected paths: `internal/convert/convert.go`, `page_islands.go`,
  `hf.go`. Rule: CS-01. Proof: call sites remain explicit and no public API
  expands; `runContext` and `objectRenderContext` are private.
- [x] Split `Run` and `renderObject` at lifecycle boundaries, preserving early
  returns and error wrapping. Rule: CS-02/CS-05. Proof: each phase has one
  responsibility; object rendering, preparation, and final assembly now
  have named private owners.
- [x] Decide file moves only after the phase seams are stable. Rule: CS-06.
  Proof: the 2,000-line guard is satisfied and responsibility clusters were
  kept cohesive rather than fragmented into shallow files.

### Phase 2: Apply small consistency cleanup

- [x] Normalize import groups in `style_cascade.go` and `layout_images.go`.
  Rule: CS-03. Proof: standard-library and local imports are separated in both
  affected files.
- [x] Normalize private accumulator initialization where capacity or nil
  semantics are known. Rule: CS-04. Proof: page-order and fixed-op accumulators
  use explicit capacity while empty-result contracts remain unchanged.
- [x] Replace broad suppression comments with narrow invariant/upgrade-path
  comments during the owning refactor. Rule: CS-07. Proof: changed seams use
  lifecycle/invariant comments; remaining fixed-point suppressions name their
  reason and are not moved to new public abstractions.
- [x] Correct benchmark run-count prose and establish one canonical baseline
  record. Affected paths: `internal/convert/perf_test.go:72-132`,
  `testdata/golden/benchmarks/benchmark-results.txt`. Rule: CS-09. Proof:
  comments, loop count, command, toolchain, and units agree in Snapshots A/B/C.
- [x] Make ownership and validation order explicit at public/load/app seams.
  Affected paths: `api.go:162-296`, `internal/load/load.go:244-289`,
  `internal/app/pdf.go:48-58`. Rule: CS-10. Proof: settings/policy cloning,
  proxy validation, signal contexts, and pre-output request validation are
  source-visible.
- [x] Consolidate body/header/footer paint-order control flow under the shared
  policy. Affected paths: `internal/layout/paint_order.go`,
  `internal/layout/paint.go:492`, `internal/convert/hf.go:452`. Rule: CS-11.
  Proof: `PaintBandContext` uses `PaintOrder`; link projection stays separate
  and focused policy coverage was added.

### Phase 3: Closure

- [x] Complete source-level closure inspection under the command constraint.
  Proof: imports, signatures, ownership, benchmark metadata, changed paths,
  and line counts were inspected; no Git or Go commands were run.
- [x] Re-read the changed signatures and compare both PDF and raster call paths.
  Proof: shared `PaintOrder` remains the visual ordering policy and link
  annotations remain an explicit metadata projection.
- [x] Re-score this slice with visible arithmetic; keep the earlier score
  using current source evidence; the updated arithmetic is recorded below.

## Rating

**Headline project-wide rating: 8.6/10.** The four-track arithmetic appears
above. The direct code-style slice is also 8.6/10:

| Area | Score | Weight | Weighted |
|---|---:|---:|---:|
| Guard clauses and error-flow clarity | 9.2 | 0.15 | 1.380 |
| Function size and parameter shape | 8.4 | 0.20 | 1.680 |
| Control-flow complexity | 8.4 | 0.15 | 1.260 |
| Declarations and data-shape consistency | 8.9 | 0.10 | 0.890 |
| File/type organization | 8.5 | 0.15 | 1.275 |
| Comments and suppression discipline | 8.1 | 0.10 | 0.810 |
| Architecture locality and testability | 8.7 | 0.15 | 1.305 |
| **Total** |  | **1.00** | **8.600 → 8.6/10** |

This is intentionally critical: the score is held below 10 by compatibility
adapters, a few documented fixed-point suppressions, and unavailable runtime
gates—not by any remaining unchecked implementation row.

## Validation boundary

Go source and review-plan files were modified. No Git commands and no Go
commands were run. Source-level inspection was the only permitted validation
boundary for this implementation turn.
