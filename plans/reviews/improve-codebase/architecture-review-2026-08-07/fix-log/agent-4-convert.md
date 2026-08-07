# Agent 4 — convert/app follow-up

Scope honored: `internal/convert/**`, `internal/app/**`, convert-owned tests,
and this fix log. No phase checklist or `fix-contract.md` was edited. No git
commands were run.

## Findings

| CID | Status | Evidence |
|---|---|---|
| ARCH-02 | done | `convert.Request` validates an explicit PDF `Output` sink and a separate `OutlineOutput` sink. `convert.Run` no longer falls back to `os.Stdout`; only the compatibility/CLI adapters select stdout. `internal/convert/seams_test.go` covers both missing-sink errors. |
| ARCH-04 | done in owned scope | `internal/app/pdf.go` centralizes parsed-command to `convert.Request` translation and output ownership. `convert.RunPDFContext` and `RunPDF` remain thin compatibility adapters and preserve the fix-contract request shape. Existing command mains can migrate to `app.RunPDF` in their owning scope. |
| PREP-01 | done | `PrepareDocument` owns load, HTML parse, stylesheet collection, simplify stylesheet insertion, and font-face merging. PDF `renderObject` now consumes this path. `TestPrepareDocumentBindsSharedResourceContext` proves the seam. |
| PREP-02 | done | `ResourceContext` binds loader, resolved base URL, and `settings.LoadPage`; CSS, fonts, and images use its methods. This preserves `CollectSheets` and `MergeFontFaces` while eliminating repeated policy tuples in the PDF path. |
| X-01 | partial in owned scope | The outward `internal/app` adapter is implemented and tested. Existing `cmd/*` call sites are outside this agent's exclusive scope and should switch to `app.RunPDF` / the matching image adapter. `convert.RunPDFContext` remains intentionally for compatibility. |

## Imageout integration notes

The imageout owner should replace its load/parse/gather/merge block with:

```go
prep, err := convert.PrepareDocument(ctx, loader, obj.Page, obj.Load, registry,
    convert.PrepareOptions{
        ViewportW:  768,
        ViewportH:  576,
        MediaType:  media,
        SimplifyDOM: enabled,
        SimplifyProfile: profile,
    }, log)
```

Then use `prep.Resource` for skip handling, `prep.Root`, `prep.Sheets`,
`prep.Registry`, and `prep.Resources.Fetch` for image bytes. Do not call
`CollectSheets` or `MergeFontFaces` again after consuming `PreparedDocument`.

## Files changed

- `internal/convert/convert.go`
- `internal/convert/prepare.go`
- `internal/convert/seams_test.go`
- `internal/app/pdf.go`
- `internal/app/pdf_test.go`
- `plans/reviews/improve-codebase/architecture-review-2026-08-07/fix-log/agent-4-convert.md`

## Validation

- `gofmt -w internal/convert/prepare.go internal/convert/convert.go internal/convert/seams_test.go internal/app/pdf.go internal/app/pdf_test.go` — passed
- `go test ./internal/convert ./internal/app` — passed
- `go test ./...` — passed
- `go vet ./...` — passed

## Remaining integration marker

```go
// FIX-REVIEW: X-01 cmd/gowkhtmltopdf/main.go and cmd/gowkhtmltoimage/main.go
// should call the outward app adapters; those files are owned by another agent.
```
