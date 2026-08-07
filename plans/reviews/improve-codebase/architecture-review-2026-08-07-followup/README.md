# Architecture deepening follow-up — gowkhtmltopdf

> **Status:** proposed follow-up; no production implementation performed
> **Generated:** 2026-08-07
> **Parent:** [`../README.md`](../README.md)
> **Method:** `improve-codebase-architecture`, `codebase-design`, and `skills/phase-wise-checklist/SKILLS.md`

This is a new residual review after the existing 2026-08-07 architecture ledger.
It preserves that ledger and the user's current worktree changes. The review uses
five parallel read-only scouts and records only friction that remains in the current
source.

## Canonical artifacts

- [Canonical ledger](./architecture-review-2026-08-07-followup.md)
- [Current architecture rating](./rating.md)
- [Visual candidate report](./architecture-review-2026-08-07-followup.html)
- [Phase-wise checklist](./phases/)

## Evidence boundary

- Current branch: `chore/ponytail-review-1`
- Current commit at scan start: `37f5ee2 fix(cli): restore pre-object pending for page-only flags`
- Worktree changes were documentation-only and were not overwritten.
- Five read-only scouts ran concurrently, covering the CLI/API, document/load pipeline,
  CSS/layout, output/fonts/raster, and whole-tree validation concerns.
- Current gates: `make lint` passed (`go vet ./...` and clean `gofmt`); `make test`
  passed (`go test ./...`).
- The checklist is intentionally not closed: every new implementation row is `[ ]`
  until its source, tests, and (where relevant) benchmark evidence exist.

## Preserved architecture

The review keeps the existing deep modules and seams: `convert.Request` as a first
CLI-independent seam, `load.NewLoader` plus `AccessController`/`FetchSub`, shared
`CollectSheets` and `MergeFontFaces`, `pagePlan` ownership/remapping, settings' shared
Get/Set descriptors, `layout.StyleOf`/`PaintBand`, per-layout image caching, and the
golden fixture corpus.
