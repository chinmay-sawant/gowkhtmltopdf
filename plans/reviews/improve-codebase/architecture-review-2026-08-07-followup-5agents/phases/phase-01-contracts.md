# Phase 1 — Parser, output, and mode contracts

> **Parent:** [`../architecture-review-2026-08-07-followup.md`](../architecture-review-2026-08-07-followup.md)
> **Status:** not started
> **Depends on:** none

## Goal

Make the engine interfaces explicit before sharing more preparation logic. Keep the
CLI parser as the adapter from argv; keep output policy out of the engine.

## Checklist

- [ ] **ARCH-01** — pass `ModePDF`/`ModeImage` into `cli.Parse`; reject flags whose
  `flagSpec.mod` excludes the selected mode. Proof: complete flag applicability matrix
  tests plus both CLI smoke paths.
- [ ] **ARCH-02** — replace direct `os.Stdout` writes in `convert.Run` with explicit
  document/outline sinks and define nil-sink behavior. Proof: buffer-backed outline,
  PDF, failed-writer, and CLI stdout/file tests.
- [ ] **ARCH-03** — split image settings from the PDF-owned `convert.Request` union
  while retaining shared preparation helpers. Proof: root `Converter` and
  `ImageConverter` tests construct only their mode-specific request invariants.
- [ ] **ARCH-04** — update command mains and compatibility adapters to build the new
  request/sink contracts without duplicating output ownership. Proof: `go test ./...`
  and CLI output compatibility checks.
- [ ] **X-01** — move CLI translation into an application adapter package or an
  equally explicit outer seam; keep engine modules independent of `internal/cli`.
  Proof: dependency inspection, CLI integration tests, `go test ./...`, and `go vet ./...`.
- [ ] **X-03** — make `Converter.AddObject` and `SetBody` deep-copy nested slices,
  maps, and inline HTML according to their documented snapshot contract. Proof:
  post-add mutation tests and `go test -race ./...`.

## Required gate

- [ ] Run `make lint` and `make test` on the final Phase 1 diff; record command,
  commit, and result here before closing any row.
