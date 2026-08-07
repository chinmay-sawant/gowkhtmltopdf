## Summary

This PR deepens the conversion architecture around explicit mode/output
contracts, shared document preparation, resource ownership, layout state,
PDF/raster painting, font shaping, and cancellation. It also adds regression
coverage and closes the architecture follow-up checklist with a current rating
of 9.0/10.

## Motivation / context

- Plans: `plans/reviews/improve-codebase/architecture-review-2026-08-07-followup/`
- Contract: `plans/reviews/improve-codebase/architecture-review-2026-08-07/fix-contract.md`
- The Markdown phase/checklist/fix-log files are review and planning artifacts,
  not production behavior. Please ignore those files when reviewing the code
  changes. They will later be used to enhance the Goslop application with
  additional rules and to tighten existing rules.
- Related issue: no issue number was supplied for this review-driven change.

## Changes

### Contracts and preparation

- Enforce PDF/image flag applicability at the CLI parser boundary.
- Add explicit PDF/image request constructors and output/outline sinks while
  retaining compatibility adapters.
- Move command translation into `internal/app`.
- Share document preparation and resource context across PDF and image paths.
- Deep-copy public converter snapshots and inline HTML inputs.
- Make outline ordering explicitly document-page based.

### Rendering and validation

- Include container font size in convergence state.
- Preserve display-list identity across pagination rewrites and centralize image
  used-size policy.
- Add collision-safe PDF resources, cloned-page ownership, stable font identity,
  shared shaped runs, and raster/PDF paint-order parity.
- Carry cancellation through layout, PDF/header-footer paint, and rasterization.
- Add cross-mode, security, output-sink, resource, font, image, and benchmark
  coverage.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Adds focused benchmarks; no intentional hot-path regression identified. |
| **Memory** | Public converter snapshots are safely owned; cloned PDF resource maps are independent. |
| **Behavior / correctness** | Mode-invalid flags reject early; resource limits, outline ordering, image sizing, paint parity, and cancellation are more deterministic. |
| **API / CLI** | Existing compatibility APIs remain; new explicit constructors/context entrypoints are available. |
| **Dependencies** | No new dependency. |
| **Binary size / build time** | No material change expected; build and vet pass. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | Existing `Parse`, `Layout`, `Render`, and compatibility adapters remain available. New callers may use the explicit mode/context seams. |

## Test plan

- [x] `make test`
- [x] `make lint` / `go vet ./...`
- [x] `go test -race ./...`
- [x] `go test ./internal/css ./internal/layout -count=1`
- [x] Focused layout, PDF, image, load, outline, CLI, and API regressions
- [x] Release/debug CLI smoke timing with fixture and cold/warm conditions

### Commands

```sh
make lint
make test
go test -race ./...
go test ./internal/css ./internal/layout -count=1
go test ./internal/layout -run '^$' -bench 'Benchmark(DisplayListIdentity10kOps100Pages|UsedImageSize)' -benchmem -count=1
go test ./internal/pdf -run '^$' -bench 'Benchmark(ShapeRun|Write50Pages)' -benchmem -count=1
```

## Screenshots / sample output

No UI changes. CLI mode smoke checks reject invalid flags with explicit errors;
PDF and image output smoke paths succeed.

## Related issues

- No issue number supplied; this PR is the implementation of the architecture
  follow-up review.

## PR metadata checklist (author)

- [x] Self-assigned (`@me`)
- [x] Labels applied
- [x] Related-issue status stated without inventing a ticket ID
- [x] Filled body committed under `plans/PR/pr-architecture-enhancements.md`

## Follow-ups (out of scope)

- Use the Markdown review artifacts later as input to Goslop rule additions and
  tightening existing rules; that work is intentionally outside this PR.
- Review and merge the PR through the normal `master` branch workflow.

## Reviewer checklist

- [ ] Behavior matches the summary and test plan
- [ ] No unrelated code changes are included
- [ ] Public API and CLI changes are documented
- [ ] Resource, font, image, and cancellation regressions are covered
- [ ] PR has assignee and labels
- [ ] No secrets or generated artifacts are committed
