# Reviews / Improve codebase architecture

Reserved for **architecture deepening** reviews, separate from ponytail leanness.
The skill pack is [`skills/improve-codebase/`](../../../../skills/improve-codebase/):
`/improve-codebase` (all three lenses → one ledger), plus
`/improve-codebase-architecture`, `/improve-codebase-extension`, and
`/improve-codebase-practices`.

## Relation to ponytail

| Folder | Focus | Skill |
|---|---|---|
| [`../ponytail/`](../ponytail) | Over-engineering / dead code / YAGNI — what to **delete** | `skills/ponytail-audit`, `skills/ponytail-review` |
| `improve-codebase/` (this folder) | Module depth, extension honesty, Go practices — what to **deepen** | `skills/improve-codebase` (`/improve-codebase`), `codebase-design` |

Run architecture reviews **after** (or in parallel with) ponytail Phase 0–1 deletes so deepening does not solidify stub surfaces.

## Status

- [x] Improve-codebase wave closed (2026-08-13): [codebase-2026-08-13/phase-wise-checklist.md](codebase-2026-08-13/phase-wise-checklist.md) — 5 explore agents, 18 active IDs; implementation complete.
- [x] Post-0.2.4 improve-codebase wave implemented (2026-08-21): [`../../../0.2.4/improve-codebase/codebase-2026-08-20/phase-wise-checklist.md`](../../../0.2.4/improve-codebase/codebase-2026-08-20/phase-wise-checklist.md) — **8.8 / 10** after implementation (pre-fix **7.8 / 10**). Uncommitted on `chore/0.2.4-improve-codebase`.
- [x] Architecture review complete (2026-08-07): [architecture-review-2026-08-07/](architecture-review-2026-08-07) — 7 explore agents, 49 findings, 46-row phase-wise checklist.
- [x] Ponytail baseline: [`../ponytail/ponytail-2026-08-06/ponytail-ultra-2026-08-06.md`](../ponytail/ponytail-2026-08-06/ponytail-ultra-2026-08-06.md) — **5.7 / 10** leanness.
- [x] Critical Go contract remediation complete (2026-08-12): [critical-go-review-2026-08-12/](critical-go-review-2026-08-12) — CR-01 through CR-08 closed with current source/test evidence.
- [~] Earlier same-day application ledger: [application-review-2026-08-12/app-review.md](application-review-2026-08-12/app-review.md) — historical **7.4 / 10** scorecard. Remaining work and the current rating live in the 3-lens health review below.
- [x] Codebase health review closed (2026-08-12): [codebase-review-2026-08-12/phase-wise-checklist.md](codebase-review-2026-08-12/phase-wise-checklist.md) — implementation wave complete; recalculated score **8.8 / 10**.

The 2026-08-07 and 2026-08-13 architecture waves are closed. New post-0.2.4
architecture work lives under
[`plans/0.2.4/improve-codebase/`](../../../0.2.4/improve-codebase/). Keep the
separation from the ponytail leanness review.
