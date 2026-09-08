---
name: solid-go-review
description: Review a Go codebase for SOLID principles, idiomatic Go design, and reusable package seams. Return evidence-backed findings and a reproducible score out of 10. Read-only, no Git commands.
---

# SOLID and Go review

Use this skill when the user asks whether a Go codebase follows SOLID principles, idiomatic Go design, or reusable code and package boundaries.

The goal is useful reuse. A finding must explain what can be reused, what is coupled, what contract is unclear, or what change would become safer. Do not add abstractions only to make a SOLID letter look satisfied.

## Constraints

- Review source and tests without editing them. Return the report in the response unless the user explicitly requests a report file.
- Do not run Git commands. This includes read-only commands such as `git status` and `git diff`.
- Read `knowledge-base/wiki/index.md` first when it exists. Treat it as orientation, then verify claims against current source and tests.
- Keep package ownership disjoint during parallel work. A worker may inspect a neighboring package only to understand one named seam.
- Workers do not run lint or mutate the tree. One orchestrator owns any repository-wide validation.
- Use current paths discovered from the tree. Do not assume a file name from an older report or plan.
- Every source claim gets a `file:line` citation. Mark unverified ideas as hypotheses and name the check that would confirm them.
- Apply the repository's phase-wise checklist format to findings and apply `/unslop` to the report. Use plain prose and no em dashes.

## Review workflow

1. Discover the repository before assigning work.

   Read `knowledge-base/wiki/index.md`, package documentation, `go.mod`, the test commands, and the package graph. Use `go list ./...`, `rg --files`, and focused source reads when available. Record the actual packages, large files, exported entry points, interfaces, constructors, registries, pools, and external I/O boundaries.

2. Build a package ownership map.

   Assign each package to one worker. Split a packet when it contains a large stateful package, more than one major pipeline stage, or enough files that a worker cannot cite the relevant code. Keep `internal/convert`, `internal/layout`, `internal/pdf`, and `internal/imageout` separate in this repository because each owns a substantial stage or output path. Group only small, cohesive packages.

   Use one bounded wave with no more workers than packets. Four workers are not a requirement. Keep enough context and memory for the orchestrator to inspect seams afterward. If delegation is unavailable, inspect the same packets sequentially and say so. Never claim that a parallel wave ran when it did not.

3. Run bounded discovery packets.

   Give every worker the same short contract:

   ```text
   Review only these packages: <owned packages>.
   Do not edit files, run Git commands, or run lint. Do not inspect another worker's package except for the named seam: <seam or none>.
   Return: package and file responsibilities, confirmed findings, hypotheses, good choices, and an area score.
   Cite every source claim as path:line. For each finding include rule ID, severity, impact, fix direction, and proof needed.
   Treat an absent interface as neutral unless a concrete dependency prevents reuse or testing.
   ```

4. Inspect seams after the packets return.

   The orchestrator checks the calls and types that cross packet boundaries. Pay special attention to data ownership, mutation, error identity, context flow, resource limits, concrete construction, and whether a proposed interface belongs to the consuming package. Reject findings that disappear when the adjacent package is read.

5. Validate load-bearing findings.

   Use the smallest check that proves the claim. Read source for structure claims, run a targeted test for behavior claims, run a benchmark for performance claims, and use a grep or package listing for absence claims. For this repository, use the Makefile gates rather than bare `go test ./...`: `make test`, `make test-race`, `make lint`, `make claim-scan`, and `make golden` as the scope requires. Run repository-wide checks once, by the orchestrator, and record skipped or failed checks.

6. Synthesize only after validation.

   Report confirmed strengths, confirmed findings, hypotheses, coverage gaps, per-area scores, the weighted overall score, and the highest-value fixes. Do not convert a theoretical preference into a violation.

## SOLID review

Review each principle as a design question, not as a demand for interfaces.

- **SRP:** Can a type, function, or package change for several unrelated reasons? Look for mixed policy, orchestration, I/O, parsing, layout, and output responsibilities. A large type is evidence to inspect, not proof of a violation. Name the separate reasons to change.
- **OCP:** Can a new supported case be added through data, a registry, a narrow policy, or a new implementation? A switch is acceptable for a closed domain or a central parser. Flag it only when unrelated behavior must edit the same core path.
- **LSP:** Apply this only where an interface or substitutable implementation exists. Check documented preconditions, postconditions, error behavior, cancellation, and resource ownership. Report `N/A` when no substitution contract exists. Do not penalize a package for having no hierarchy.
- **ISP:** Prefer small interfaces defined by consumers. Flag interfaces that force unrelated methods, or concrete structs passed across packages when a narrow behavior contract would improve reuse or testing. Do not create an interface for a single implementation without a real seam.
- **DIP:** Keep high-level policy independent from low-level loading, time, filesystem, network, rendering, and output details. Construction may remain concrete at a boundary. Flag concrete construction inside reusable policy when it prevents testing, replacement, or controlled resource use.

For every SOLID finding, include the affected symbol, the reason it changes, the dependency or contract that causes the problem, and the smallest seam that would improve reuse.

## Idiomatic Go and reusable design

Check these dimensions in addition to SOLID:

- **Package cohesion:** Packages have one clear purpose, useful package docs, stable exported names, and no accidental reverse dependency. Reuse should happen through a small package contract, not by reaching into internals.
- **Composition:** Prefer concrete types, embedding, function values, and small consumer interfaces over inheritance-shaped hierarchies. A shared helper is justified when it removes duplicated policy, not merely duplicated syntax. Keep intentional duplication when two similar paths have different reasons to change.
- **API and zero values:** Check whether the zero value is useful or clearly rejected, constructors validate required invariants, options are used only when configuration is genuinely optional, and exported structs do not expose unsafe mutable state without a reason. Parameter count is a review signal, never a rule.
- **Errors and context:** Preserve sentinel identity with `%w`, add useful operation context, define errors at the boundary that owns them, and pass `context.Context` through cancellable work without storing it in long-lived structs.
- **Ownership and concurrency:** State who owns slices, maps, buffers, files, writers, and pooled objects. Check documented single-goroutine contracts, synchronization, cancellation, and race-test coverage. Do not recommend pooling without measured allocation or lifetime benefit.
- **Boundaries and resource use:** Validate external input at the boundary, keep policy near the boundary, bound body sizes, pages, dimensions, recursion, and other untrusted work, and avoid hidden network or filesystem access in reusable functions.
- **Reuse and test seams:** A reusable component has a narrow contract, explicit inputs and outputs, controlled side effects, and a test seam that does not require the whole pipeline. Check whether tests exercise the contract rather than private implementation details.
- **Dependencies:** Count direct modules from `go.mod`, inspect why each exists, and distinguish direct dependencies from transitive ones. Do not reward a low count if the code replaces a needed library with harder-to-maintain code.
- **Language and tooling:** Check naming, doc comments, `io.Reader` and `io.Writer` use, generics only where they clarify a real repeated type operation, fuzz tests for parsers, benchmarks for hot paths, and static analysis findings. Treat `//nolint` as a prompt to inspect the design, not as automatic failure.

For each dimension, report one good choice, one confirmed risk if present, and the evidence. Skip irrelevant dimensions and say why.

Use this reuse ladder when recommending an abstraction, in order:

1. Reuse an existing function or type when its contract already fits.
2. Move repeated policy behind data or a small pure helper.
3. Define a consumer-owned interface when a caller needs substitution or a test seam.
4. Extract a package only when at least two real callers share a stable concept and ownership is clear.

Stop at the first level that solves the problem. Do not create an interface, generic helper, registry, or package only to remove a few similar lines.

## Findings ledger

Return findings in this shape. Keep one atomic row per issue or fix.

```markdown
## Findings

### Phase 1: Correctness and coupling

- [ ] `GO-DIP-01` `internal/foo/bar.go:42` - DIP - reusable service constructs a concrete network client inside policy code; expected behavior: the boundary supplies the client or a consumer-owned narrow interface; proof: targeted test with a fake client and the package test.

### Phase 2: API and reuse

- [ ] `SOLID-ISP-01` `internal/foo/api.go:18` - ISP - callers must implement unrelated methods; expected behavior: consumers depend on the smallest behavior they use; proof: compile and package tests after the seam is split.

### Phase 3: Cleanup

- [~] `GO-POOL-01` `internal/foo/pool.go:90` - deferred because allocation impact is unmeasured; owner boundary: package owner; next gate: benchmark allocation count before changing pooling.
```

Use `[ ]` for an open finding, `[x]` only when the matching implementation and validation are already present in the current source, and `[~]` only for an intentionally deferred item with a reason, owner boundary, and next gate. A review must not mark a proposed fix `[x]` merely because the fix direction is clear.

Each finding must contain:

- a stable rule ID such as `SOLID-SRP`, `SOLID-DIP`, `GO-API`, or `GO-OWNERSHIP`;
- severity: critical, high, medium, or low;
- source evidence and, when relevant, test, benchmark, lint, race, or grep proof;
- the user-visible or maintenance impact;
- a fix direction, not an unsolicited code change.

Order findings by correctness and coupling, then API and reuse, then performance and cleanup. Keep hypotheses separate from confirmed findings.

## Scoring

Score each applicable category from 0 to 10 using the same scale:

- `0-2`: repeated or contract-breaking problems;
- `3-4`: material problems with limited safe reuse;
- `5-6`: mixed design with useful seams and clear debt;
- `7-8`: strong design with isolated issues;
- `9-10`: clear contracts, low coupling, and current validation with no material finding in scope.

Use these weights for the overall codebase score:

| Category | Weight |
| --- | ---: |
| SRP | 10% |
| OCP | 10% |
| LSP | 10% |
| ISP | 10% |
| DIP | 10% |
| Package cohesion and dependency direction | 15% |
| API ergonomics and reusable seams | 10% |
| Errors, context, ownership, and resource bounds | 10% |
| Tests, benchmarks, and validation | 10% |
| Dependency simplicity and language fit | 5% |

If a SOLID category is `N/A`, omit its weight and renormalize the remaining weights. Show the arithmetic. Do not apply an undefined coupling adjustment. Cross-package coupling belongs in DIP, package cohesion, and API seams.

Give each packet an area score using the applicable categories it reviewed. The overall score comes from the weighted table, not from averaging worker opinions. State confidence as high, medium, or low based on package coverage and validation gates.

## Final response

Keep the final response short enough to scan. Include:

1. scope and whether workers or sequential inspection were used;
2. one SOLID table with `PASS`, `MIXED`, `FAIL`, or `N/A` and citations;
3. the strongest reusable design choices with citations;
4. confirmed findings ordered by coupling payoff;
5. the findings ledger;
6. packet scores, weighted arithmetic, overall score, and confidence;
7. three to five fixes naming the target package and seam;
8. validation commands, exit results, and skipped gates.

Apply `/unslop` before sending. Explain technical terms in plain words, avoid generic praise, and never claim that a review is complete without stating its coverage and evidence limits.
