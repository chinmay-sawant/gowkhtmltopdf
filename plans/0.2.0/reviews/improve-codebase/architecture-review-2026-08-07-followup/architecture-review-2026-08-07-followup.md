# Gowkhtmltopdf — residual architecture deepening review

> **Parent:** [`../README.md`](../README.md) — architecture deepening reviews
> **Status:** complete; implementation and validation closed
> **Estimated effort:** five focused agent slices plus orchestrator integration
> **Plan shape:** [`skills/phase-wise-checklist/SKILLS.md`](../../../../../skills/phase-wise-checklist/SKILLS.md)

This ledger records residual architectural opportunities after the existing
`architecture-review-2026-08-07` remediation. It is about module depth, seams,
locality, leverage, and testability; it is not a second ponytail deletion audit.

## Executive summary

The current architecture is strong enough to support substantial product work, but
The implementation closed the following previously leaking invariants:

1. the CLI declares a `Mode` on each flag but does not enforce it while parsing;
2. PDF output and outline bytes bypass `convert.Request.Output`, while image mode has
   a different nil-output contract;
3. PDF and image conversion duplicate document preparation and repeatedly carry the
   same `(loader, base, LoadPage)` resource context;
4. layout pagination, output resource naming, raster traversal, and font identity
   still expose backend or storage details across real seams.
5. CLI translation, cancellation, and public snapshot ownership are not fully
   localized at the cross-cutting seams.

The phase order was followed: mode/output contracts first, then shared preparation,
layout state, output resources, and final cross-mode validation.

## Current architecture rating

**9.0 / 10** — see [rating.md](rating.md) for the weighted calculation.

This is an architecture rating, not the separate 9.5/10 ponytail leanness rating in
the prior review. The score reflects the completed seams and current executable
evidence; compatibility wrappers and background-context adapters account for the
remaining deductions.

## Findings and recommendation map

| ID | Residual opportunity | Severity | Strength | Phase |
|---|---|---:|---|---:|
| ARCH-01 | Enforce CLI mode in the parser | High | Strong | 1 |
| ARCH-02 | Make document and outline output explicit sinks | High | Strong | 1 |
| ARCH-03 | Split the PDF-owned request union from image mode | Medium | Worth exploring | 1 |
| PREP-01 | One shared document-preparation module | High | Strong | 2 |
| PREP-02 | Bind one resource context to base and load policy | High | Strong | 2 |
| OUT-01 | Replace `Heading.Page` view overloading with explicit ordering | Medium | Worth exploring | 2 |
| CSS-01 | Include container font size in convergence equality | High | Strong | 3 |
| LAYOUT-01 | Preserve display-list identity across op rewriting | High | Strong | 3 |
| IMG-01 | One used-image-size policy for measure and build | Medium | Strong | 3 |
| PDF-01 | Centralize page resource naming and clone ownership | High | Strong | 4 |
| PAINT-01 | Share traversal policy across PDF, header/footer, and raster adapters | High | Strong | 4 |
| FONT-01 | Give font identity and shaping one owning module | High conditional | Worth exploring | 4 |
| X-01 | Move CLI adapters out of deep engine modules | High | Strong | 1 |
| X-02 | Carry cancellation through layout and raster | High | Strong | 5 |
| X-03 | Make public object snapshots deep copies | Medium | Strong | 1 |

The phase files record completed implementation rows and proof gates.

## Dependency-ordered phases

| Phase | Goal | Depends on | Checklist |
|---|---|---|---|
| 1 | Close parser, output, and mode request contracts | — | [phase 1](phases/phase-01-contracts.md) |
| 2 | Prepare documents and resources once; make outline ordering explicit | 1 | [phase 2](phases/phase-02-document-prep-and-outline.md) |
| 3 | Stabilize CSS/layout state and image geometry | 2 | [phase 3](phases/phase-03-css-layout-state.md) |
| 4 | Deepen output resources, traversal, and typography seams | 3 | [phase 4](phases/phase-04-output-paint-fonts.md) |
| 5 | Cross-mode regression, benchmark, and closure gates | 1–4 | [phase 5](phases/phase-05-validation-closure.md) |

## Status rules

- A phase row is closed only when implemented and validated with current evidence.
- No implementation row may be closed before the matching `make lint` and
  `make test` pass. Performance rows also require the exact release/debug command,
  fixture, OS, cold/warm state, concurrency, and metric.

## Closure evidence already available

- `make lint`: pass on the current source.
- `make test`: pass on the current source, including the repository's package tests
  and golden-corpus checks that run under `go test ./...`.
- The existing review's shared helpers and current test suite remain the baseline;
  they are not repeated as new findings merely because they are large modules.
