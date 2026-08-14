# Snapshot Architecture Findings - Phase-Wise Checklist

> **Parent:** [`../README.md`](../README.md) - architecture-deepening review index.
> **Related historical ledger:** [`phase-wise-checklist.md`](phase-wise-checklist.md) - completed earlier critical-Go remediation; this file does not reopen or rewrite those closed rows.
> **Status:** this implementation wave is closed for ARC-01, ARC-02, and ARC-05. ARC-03 and ARC-04 remain explicitly documented as follow-up architecture work.
> **Review base:** `64f4ca7059295a49ea4630d68c603c03049d2bee` (`chore(lint): fix remaining lint findings across pdf, layout, load, and convert`).
> **Excluded work:** every non-base worktree change present during review, including ongoing `internal/layout` source/test work and generated output artifacts.
> **Created:** 2026-08-12
> **Method:** three independent source-only review lenses: entry contracts/configuration, conversion lifecycle/image output, and layout/PDF/SVG output state.

## Overview

This is the canonical follow-up ledger for maintainability and extension seams
found against the committed snapshot. It makes no performance claim and does
not rate or modify the excluded worktree changes.

The implementation wave centralized effective load-policy resolution, made the
image boundary explicitly single-input, and made the private load resource
context authoritative during preparation. Those changes preserve the existing
cancellation-aware render pipeline, shared display-list paint ordering, outline
projection, and SVG adapter.

## Executive Summary

| ID | Priority | Finding | Current disposition |
|---|---|---|---|
| ARC-01 | P0 | Image mode discarded the global network policy. | Implemented with one effective-load resolver and policy regression coverage. |
| ARC-02 | P1 | Image jobs and request types had contradictory ownership models. | Implemented boundary enforcement: exactly one input; replacement semantics documented and tested. |
| ARC-03 | P1 | Image stylesheet selection was disconnected from the responsive viewport. | Follow-up roadmap; no implementation claimed in this wave. |
| ARC-04 | P1 | Layout results and PDF documents had mutable post-handoff state. | Follow-up roadmap; no implementation claimed in this wave. |
| ARC-05 | P1 | Resource context was duplicated through load, prepare, and convert facades. | Implemented safety slice: private load seam is authoritative; compatibility snapshots remain for migration. |

## Evidence Baseline

- [x] Review scope was frozen at `64f4ca7`; original source evidence refers to
  that snapshot, not the live dirty tree.
- [x] Three independently scoped reviews completed. Findings are source-proven
  defects or architectural friction, not performance hypotheses.
- [x] Existing completed ledgers remain historical records and are not marked
  incomplete by this file.
- [x] The implementation wave used no git commands and preserved unrelated
  worktree changes.
- [x] Repository gates passed after the changes: `make lint`, `make test`,
  `go vet ./...`, and `go test -race -count=1 ./...`.

## Phase 0: Preserve review scope and invariants - P0

### 0.1 Controlled-report product boundary

- [x] JavaScript/browser parity and performance optimisation remain outside
  this architecture ledger; they require separate acceptance criteria.
- [x] The existing `render.Pipeline` stage order and cancellation semantics
  remain unchanged (`internal/convert/render/pipeline.go`).
- [x] Shared display-list ordering remains unchanged for PDF and raster output
  (`internal/layout/paint_order.go`).

### 0.2 Implementation contract matrix

- [x] The following matrix records the contract surface used by the focused
  implementation tests:

  | Boundary | PDF | Image | Evidence |
  |---|---|---|---|
  | Library and CLI settings | Shared global load settings | Shared global plus image mode settings | `api_test.go`, `internal/cli/cli_test.go`, `internal/imageout/policy_test.go` |
  | Primary and subresource loads | Existing loader policy | Same effective policy snapshot now reaches image loader | `internal/load/load_test.go`, `internal/imageout/policy_test.go` |
  | Valid input and error input | Existing request validation | Exactly one renderable object; multiple objects fail before output opens | `internal/imageout/request_test.go`, `internal/app/image_test.go` |
  | Cancellation and output lifecycle | Existing pipeline contract | Preflight remains before output ownership; run uses existing pipeline | `internal/app/image_test.go`, repository test gate |
- [x] Implemented slices do not change PDF bytes, raster paint ordering, page
  count, links, images, fonts, or outlines. Visual comparison remains a
  prerequisite for the future output-state work in ARC-03 and ARC-04.

## Phase 1: Unify effective network policy - P0

### 1.1 ARC-01 - Mode-neutral policy resolution

- [x] Added `load.ResolveEffectiveLoadGlobal` as the single resolver for shared
  and mode-owned load settings (`internal/load/load.go:148-190`). It gives an
  explicit shared network policy precedence, permits mode policy fallback when
  no shared policy exists, merges proxy and ACL settings deliberately, and
  clones slices for an owned per-run snapshot.
- [x] Replaced image-only field copying with the shared resolver
  (`internal/imageout/imageout.go:1580-1586`). Image loader construction now
  receives the same complete network policy shape as PDF loader construction.
- [x] Kept public and settings structs source-compatible while centralizing the
  effective-policy translation. A future policy-field addition has one resolver
  seam; full public/settings type unification is intentionally a later design
  decision, not a claim of this wave.

### 1.2 ARC-01 - Regression coverage

- [x] Added precedence and fallback tests for shared versus mode network
  policy (`internal/load/load_test.go:34-117`).
- [x] Added ownership tests proving effective policy slices do not alias caller
  state (`internal/load/load_test.go:95-117`).
- [x] Added image-mode policy propagation coverage through the image loader
  seam (`internal/imageout/policy_test.go:12-59`). Existing loader tests cover
  private-address, redirect, allow-host, and subresource policy behavior.
- [x] Focused load/image tests and the full repository gates passed after the
  resolver was integrated.

## Phase 2: Deepen image request and responsive-preparation seams - P1

### 2.1 ARC-02 - Make an image job exactly one source

- [x] `imageout.Request.Validate` now rejects more than one object with the
  typed `imageout.ErrMultipleInputs` error before rendering
  (`internal/imageout/request.go:13-16, 53-75`).
- [x] `app.RunImage` maps that validation failure to
  `app.ErrMultipleImageObjects` before opening the output path
  (`internal/app/image.go:14-50`).
- [x] `ImageConverter.AddObject` documentation now states the implemented
  replacement semantics: the most recently added page renders
  (`api.go:735-749`). The behavior is pinned by
  `TestImageConverterAddObjectReplacesInput` in `api_test.go`.
- [x] Focused request, application preflight, typed API, and repository tests
  passed.

### 2.2 Compatibility boundary retained deliberately

- [x] The existing `convert.ImageRequest` to `imageout.Request` adapter remains
  as a compatibility seam. This wave enforces its one-source invariant without
  widening the change into a mode-union removal.
- [x] The remaining union removal and neutral validation-module design are
  recorded in the roadmap below, with no false completion claim in this ledger.

## Phase 3: Responsive stylesheet eligibility - P1 roadmap

ARC-03 remains future work. The original evidence is retained here so it is not
lost: image preparation uses a fixed `768x576` viewport while rendering uses
`Image.Width`, smart-width can grow through repeated layouts, and linked media
eligibility is selected in `internal/convert/prepare/styles.go`. The next wave
must derive preparation from the configured image layout, bound any re-evaluation
cache, and add a visible/semantic fixture proving 1400px versus 1024px behavior.

## Phase 4: Layout output and PDF finalization - P1 roadmap

ARC-04 remains future work. The original evidence is retained here: pagination
and chrome repair mutate layout operations and box ranges in
`internal/layout/paint_pagination.go`, while PDF mutators remain available after
some finalization paths in `internal/pdf/pdf.go`. The next wave must define a
private per-paint pagination plan, a documented PDF finalize state machine, and
semantic plus raster regressions for repeated writes and post-finalize mutation.

## Phase 5: Collapse duplicate resource ownership - P1

### 5.1 ARC-05 - Keep one authoritative resource context

- [x] `prepare.ResourceContext` now tracks readiness and fetches through its
  private `load.ResourceContext`; its compatibility `Loader`, `Base`, and
  `Load` snapshots are documented as deprecated and are not consulted by
  `Fetch`, `CollectSheets`, or font preparation (`internal/convert/prepare/prepare.go:26-103`).
- [x] Added a mutation regression proving that changing compatibility snapshots
  cannot redirect resolution or alter the canonical request header/policy
  (`internal/convert/prepare/resource_context_test.go:13-58`).
- [x] Focused prepare/convert tests and the repository gates passed.

### 5.2 Migration boundary

- [x] The duplicate exported snapshots and forwarding aliases remain only for
  source compatibility. Their removal, after package-local call sites migrate,
  is recorded as a follow-up design task rather than marked complete here.

## Phase 6: Validation and rerating

- [x] Focused tests ran for every implemented ARC slice.
- [x] `make lint` passed with `golangci-lint v1.64.8`.
- [x] `make test` passed for all packages.
- [x] `go vet ./...` passed.
- [x] `go test -race -count=1 ./...` passed.
- [x] No renderer or PDF finalization code changed in this wave, so no new
  semantic/raster artifact was regenerated. Those checks are explicitly part
  of the ARC-03/ARC-04 implementation acceptance criteria.
- [x] Findings were rerated by disposition: ARC-01, ARC-02, and the ARC-05
  safety slice are implemented; ARC-03, ARC-04, full type unification, and
  compatibility-facade removal remain roadmap work.
- [x] Performance remains a separate dimension and was not inferred from this
  architecture work.

## Follow-up roadmap

1. Complete ARC-03 responsive stylesheet eligibility with a bounded media-query
   re-evaluation strategy and visible/semantic fixture coverage.
2. Complete ARC-04 immutable layout handoff and PDF finalization state machine,
   including repeated-write and post-finalize mutation tests.
3. Remove the `convert.ImageRequest` compatibility union once callers migrate,
   then centralize shared object validation in a neutral package.
4. Remove deprecated preparation resource snapshots and forwarding aliases after
   package-local call sites use `load.ResourceContext` directly.
5. Add end-to-end CLI and public-API image network-policy tests for primary,
   CSS, font, image, and redirect requests before expanding policy fields.

## Dependencies

```text
Completed: ARC-01 effective policy
    -> ARC-02 single-input boundary
    -> ARC-05 authoritative preparation resource seam

Roadmap: ARC-03 responsive stylesheet eligibility
    -> ARC-04 immutable layout/PDF finalization
    -> visual/semantic acceptance and final architecture score
```
