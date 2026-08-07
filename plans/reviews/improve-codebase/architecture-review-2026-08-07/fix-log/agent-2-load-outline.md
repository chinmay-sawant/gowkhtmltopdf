# agent-2-load-outline — remediation log (2026-08-07)

Owner: `internal/load/**`, `internal/outline/**`, and package tests.
Scope: PREP-03 and the outline-side API for OUT-01. Phase checklist files and
`fix-contract.md` were not edited.

## Per-CID status

| CID | Status | Evidence |
|---|---|---|
| PREP-03 | **done** | `data:` primary and subresource decoding now enforce `Loader.MaxBodySize`; inline HTML is capped at the same seam; empty document bases reject unresolved relative references instead of falling through to local-file access. Focused tests cover over-limit, exact-limit, empty-base, and absolute-data behavior. |
| OUT-01 (outline side) | **done** | Added the explicit `PageOf` contract plus `SortHeadingsBy`, `BuildTreeBy`, `SectionOfBy`, and `DumpOutlineXMLBy`. `DocumentPage` orders flattened headings without copying document pages into `Heading.Page`; legacy helpers remain thin local-page adapters. Convert-side migration is outside this agent's ownership. |
| PREP-02 (load seam) | **done** | Added `Loader.ForResource` and `ResourceContext.Fetch` so base URL and `LoadPage` policy travel together through the loader. Existing `FetchSub` remains available as a compatibility adapter. |

## Files changed

- `internal/load/load.go`
- `internal/load/load_test.go`
- `internal/outline/outline.go`
- `internal/outline/outline_test.go`
- `plans/reviews/improve-codebase/architecture-review-2026-08-07/fix-log/agent-2-load-outline.md`

## Validation

- `gofmt -w internal/load/load.go internal/load/load_test.go internal/outline/outline.go internal/outline/outline_test.go` — OK.
- `go test ./internal/load ./internal/outline` — OK.
- `go test -race ./internal/load ./internal/outline` — OK.
- `go vet ./internal/load ./internal/outline` — OK.
- `go test ./...` — **not green in the shared worktree**: `internal/layout/architecture_followup_test.go:63` currently reports `TestUsedImageSizeUsesOneAspectAndConstraintPolicy` returning `0.00x0.00` instead of `75x37.5`; `internal/load` and `internal/outline` pass in that run.

## Remaining markers

None. No files outside the exclusive scope were changed, and no git commands
were run.
