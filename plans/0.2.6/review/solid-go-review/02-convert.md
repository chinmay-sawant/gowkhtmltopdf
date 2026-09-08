# Review - Conversion pipeline

> **Parent:** `skills/solid-go-review/SKILLS.md` - review workflow
> **Status:** Complete. All findings were implemented or closed by current-source proof.
> **Estimated effort:** Not estimated

---

## Overview

This packet covered `internal/convert`, `internal/convert/prepare`, `internal/convert/render`, and `internal/convert/islands`.

## Executive Summary

The conversion package has a useful lifecycle interface, early request validation, explicit stylesheet bounds, and a clear preparation result. The main reuse problems are duplicated copy and resource validation, coarse cancellation, and a benchmark-only island path that does not share all generic body policies.

## Files and responsibilities

- `internal/convert/convert.go:49-175` - request contract, validation, and object rendering.
- `internal/convert/convert.go:467-629` - per-object loading, layout, policies, and painting.
- `internal/convert/pdf_pipeline.go:10-51` - PDF adapter over the lifecycle.
- `internal/convert/page_plan.go:29-215` - page ownership, copy limits, and materialization.
- `internal/convert/prepare/prepare.go:146-198` - resource, DOM, stylesheet, and font preparation.
- `internal/convert/prepare/styles.go:33-55` - stylesheet and font compatibility helpers.
- `internal/convert/render/pipeline.go:11-57` - lifecycle ordering and cancellation checks.
- `internal/convert/render/plan.go:28-150` - page index mapping and copy order.
- `internal/convert/islands/plan.go:16-153` - certified benchmark-island recognition and cloning.

## SOLID and Go verdicts

| Area | Verdict | Evidence |
| --- | --- | --- |
| SRP | MIXED | `render.Pipeline` owns lifecycle order (`internal/convert/render/pipeline.go:11-18`), but `renderObject` coordinates loading, margins, optimization, link policy, and painting (`internal/convert/convert.go:467-629`). |
| OCP | MIXED | The lifecycle accepts PDF and image adapters, but benchmark islands return before generic body policies (`internal/convert/convert.go:564-615`). |
| LSP | PASS | The pipeline implementations satisfy the same three-stage contract in current tests (`internal/convert/render/pipeline.go:14-18`, `internal/convert/render/pipeline_test.go:35-77`). |
| ISP | MIXED | `render.Pipeline` is narrow, but preparation retains parallel compatibility helpers beside the resource context (`internal/convert/prepare/prepare.go:26-35`, `internal/convert/prepare/styles.go:33-55`). |
| DIP | MIXED | Lifecycle order is injected through `Pipeline`, while page and resource policy are repeated across request, plan, and materialization seams (`internal/convert/page_plan.go:43-49`, `internal/convert/render/plan.go:38-46`). |

## Confirmed findings

### Phase 1: Correctness and policy parity

- [x] `PB-003` `internal/convert/convert.go:155-160` - copy validation is owned by `render.ValidateCopies`; request, page-plan, and non-collated paths share the same bounds and wrapped errors. Proof: `TestCopySeamsShareValidation` and `TestNonCollateOrder` pass.
- [x] `PB-002` `internal/convert/convert.go:564-571` - certified islands use the shared smart-shrink and body-policy path. Relative and protocol-relative HTML links now become resolved URI annotations in both generic and certified output; fragments remain local links and unsupported schemes are ignored. Proof: `TestGenericVsCertifiedIslandsDifferentiallyEqual`, `TestCertifiedIslandsApplyGenericLinkPolicies`, `TestRelativeHTMLAnchorsResolveForGenericAndCertifiedRendering`, and URI classification tests pass.

### Phase 2: Cancellation and resource seams

- [x] `PB-001` `internal/convert/render/pipeline.go:37-55` - cancellation checks now cover assembly boundaries and copy materialization before later work or output. Proof: `TestMaterializeCopiesStopsBeforeWorkWhenCanceled` and `TestLayoutBodyStopsBeforeLayoutWhenCanceled` pass.
- [x] `PB-004` `internal/convert/prepare/styles.go:33-55` - stylesheet and font compatibility helpers now delegate through one resource-context owner. Proof: preparation parity tests pass.
- [x] `PB-005` `internal/convert/hf.go:1` - the file-wide lint suppression was removed; only narrow, justified function suppressions remain. Proof: `make lint` passes.

## Hypotheses

- [x] `B-H1` `internal/convert/outline.go:32-83` - validated non-finding: navigation is projected into copied locations and links without retaining unsafe layout ownership. This row records a disproved hypothesis, not a new architecture claim. Proof: `TestBodyNavigationProjectionIsIndependentOfLayoutResult` passes.
- [x] `B-H2` `internal/convert/convert.go:467-629` - validated design decision: the suspected testing barrier was not reproduced because layout, policy, and smart-shrink seams are directly replaceable in focused tests. No extra abstraction was added. Proof: `TestLayoutBodyKeepsSmartShrinkAtAReplaceableSeam` passes.

Rows marked as validated non-findings or validated design decisions close a review question through source and test evidence. They do not claim that production architecture changed.

## Area score

**7.0/10.** The package boundaries are purposeful and tested. Copy validation, duplicate preparation APIs, policy parity, and cancellation reduce safe reuse.

## Validation

Focused tests and full `make test` passed. `make test-race`, `make lint`, `make claim-scan`, and `make golden` passed.

## Dependencies

Unify validation and cancellation before splitting orchestration functions. Benchmark-island changes require benchmark and golden evidence.
