# Phase 1 — Parser, output, and mode contracts

> **Parent:** [`../architecture-review-2026-08-07-followup.md`](../architecture-review-2026-08-07-followup.md)
> **Status:** complete
> **Depends on:** none

## Goal

Make engine interfaces explicit before sharing preparation logic. CLI parsing is
mode-aware, output ownership is explicit, and compatibility adapters remain
thin and contract-preserving.

## Checklist

- [x] **ARCH-01** — `cli.Parse` accepts an explicit mode and rejects every
  inapplicable flag. `ParseMode` and registry-driven matrix tests cover PDF,
  image, shared, short, long, and invalid-mode cases; both command mains pass
  their mode and CLI rejection/success smoke paths passed.
- [x] **ARCH-02** — `convert.Request` now has explicit document and outline
  sinks with deterministic nil behavior. `Run` has no output fallback; buffer,
  missing-sink, dedicated-outline-sink, and failed-writer tests pass.
- [x] **ARCH-03** — the fix-contract `convert.Request` shape is retained for
  compatibility, while `NewPDFRequest`/`NewImageRequest` and
  `ValidatePDF`/`ValidateImage` make the mode-specific invariants explicit.
  Root PDF and image converters use those constructors and their cross-mode
  validation tests pass.
- [x] **ARCH-04** — `internal/app` owns command-to-request translation and
  output opening; PDF and image command mains use `RunPDF`/`RunImage`. Existing
  compatibility adapters remain available and `go test ./...` plus CLI output
  smoke checks pass.
- [x] **X-01** — application adapters are the outer CLI seam; `internal/convert`
  and `internal/imageout` retain only narrow compatibility adapters for old
  callers. Dependency inspection, `go vet ./...`, package tests, and both
  executable paths pass.
- [x] **X-03** — `SetBody` and `Converter.AddObject` deep-copy inline HTML,
  maps, slices, POST data, and nested settings. Mutation and race-oriented
  tests pass under `go test -race ./...`.

## Required gate

- [x] Final Phase 1 gate: `make lint`, `make test`, and `go test -race ./...`
  passed on 2026-08-07. No commit was created, per request; evidence is in the
  five-agent fix logs and the working-tree validation run.
