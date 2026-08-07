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

The 2026-08-07 architecture review and remediation are complete. Future architecture
work should use a new dated ledger under this directory and preserve the separation
from the ponytail leanness review.
