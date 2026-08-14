# Agent 1 — CLI/API fix log

## Scope

- `internal/cli/**`
- `api.go`
- `internal/cli/cli_test.go`
- `api_test.go`

## Changes

- Added an optional mode argument to `cli.Parse` and explicit `ParseMode`.
  Existing no-mode callers continue to parse the PDF/image flag union, while
  mode-aware callers now receive a clear applicability error for flags owned by
  the other executable.
- Added complete registry-driven rejection coverage for every non-shared long
  flag and every non-shared short flag, plus positive PDF/image/shared-mode
  cases and invalid-mode validation.
- Added deep-copy helpers for `PdfObject` nested maps, slices, POST data,
  header/footer replacement maps, ignored settings, and inline HTML. `SetBody`
  now copies the caller's byte buffer, and `Converter.AddObject` stores an
  independent snapshot.
- Added mutation and race-oriented tests proving later source mutation cannot
  alter the converter snapshot. The race-oriented test reads only the copied
  snapshot while mutating the original object.

## Validation

Run from the repository root:

```text
go test ./internal/cli .
go test -race ./internal/cli
go test ./cmd/...
```

The first two focused commands passed. `go test ./cmd/...` also passed. A
root-package race run was attempted with `go test -race . ./internal/cli`; the
CLI package passed, but the root package was temporarily blocked by an
unrelated concurrent change in `internal/convert/convert.go:516` (`res`
declared and not used), outside this agent's write scope.

## Integration note

The command entrypoints remain outside this agent's write scope and still call
the backward-compatible `cli.Parse(argv)` form. The PDF and image command
owners should switch those calls to `cli.Parse(argv, cli.ModePDF)` and
`cli.Parse(argv, cli.ModeImage)` respectively to activate strict mode gating
at the executable boundary.
