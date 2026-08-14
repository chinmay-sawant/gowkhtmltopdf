# Architecture deepening follow-up — gowkhtmltopdf

> **Status:** complete; five-agent implementation and validation closed
> **Generated:** 2026-08-07
> **Parent:** [`../README.md`](../README.md)
> **Method:** `improve-codebase-architecture`, `codebase-design`, and `skills/phase-wise-checklist/SKILLS.md`

This is the residual review and implementation record after the existing
2026-08-07 architecture ledger. It preserves that ledger and records the
five-agent implementation, integration, and validation evidence.

## Canonical artifacts

- [Canonical ledger](architecture-review-2026-08-07-followup.md)
- [Current architecture rating](rating.md)
- [Visual candidate report](architecture-review-2026-08-07-followup.html)
- [Phase-wise checklist](phases)

## Evidence boundary

- Five focused implementation agents ran concurrently across the CLI/API,
  document/load/outline, CSS/layout, convert/application, and PDF/image/font
  scopes; the orchestrator integrated the cross-scope seams.
- Final gates: `make lint`, `make test`, `go test -race ./...`, and the focused
  CSS/layout gate all passed.
- Every phase row is closed with current source, test, and benchmark evidence;
  no checklist row remains deferred or unstarted.

## Preserved architecture

The review keeps the existing deep modules and seams: `convert.Request` as a first
CLI-independent seam, `load.NewLoader` plus `AccessController`/`FetchSub`, shared
`CollectSheets` and `MergeFontFaces`, `pagePlan` ownership/remapping, settings' shared
Get/Set descriptors, `layout.StyleOf`/`PaintBand`, per-layout image caching, and the
golden fixture corpus.
