# Reviews / Improve codebase architecture

Reserved for **architecture deepening** reviews (`improve-codebase-architecture` skill), separate from ponytail leanness.

## Relation to ponytail

| Folder | Focus | Skill |
|---|---|---|
| [`../ponytail/`](../ponytail/) | Over-engineering / dead code / YAGNI — what to **delete** | `skills/ponytail-audit`, `skills/ponytail-review` |
| `improve-codebase/` (this folder) | Module depth, seams, locality — what to **deepen** | `improve-codebase-architecture`, `codebase-design` |

Run architecture reviews **after** (or in parallel with) ponytail Phase 0–1 deletes so deepening does not solidify stub surfaces.

## Status

- [x] Architecture review complete (2026-08-07): [architecture-review-2026-08-07/](./architecture-review-2026-08-07/) — 7 explore agents, 49 findings, 46-row phase-wise checklist.
- [x] Ponytail baseline: [`../ponytail/ponytail-ultra-2026-08-06.md`](../ponytail/ponytail-ultra-2026-08-06.md) — **5.7 / 10** leanness.
- [x] Critical Go contract remediation complete (2026-08-12): [critical-go-review-2026-08-12/](./critical-go-review-2026-08-12/) — CR-01 through CR-08 closed with current source/test evidence.
- [~] Earlier same-day application ledger: [application-review-2026-08-12/app-review.md](./application-review-2026-08-12/app-review.md) — historical **7.4 / 10** scorecard. Remaining work and the current rating live in the 3-lens health review below.
- [x] Codebase health review closed (2026-08-12): [codebase-review-2026-08-12/phase-wise-checklist.md](./codebase-review-2026-08-12/phase-wise-checklist.md) — implementation wave complete; recalculated score **8.8 / 10**.

The 2026-08-07 architecture review and remediation are complete. Future architecture
work should use a new dated ledger under this directory and preserve the separation
from the ponytail leanness review.
