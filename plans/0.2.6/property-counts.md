# Property counts (v0.2.6)

WebRef inventory size is 818 property names.

## Current snapshot (honesty revert 2026-08-28)

| Status | Count |
|--------|------:|
| Implemented | 166 |
| Partial | 9 |
| Unsupported | 643 |
| Ignored | 0 |

Four explore audits of phases 58-78 found mass **catalog flips** (checklist `[x]` + `mapping.json` Implemented) without matching engine work, especially phases **69-77** (all 247 former Ignored names). Those rows were reverted to **unsupported** (`goal: implement`) except `filter` (opacity-only **partial**). Over-promoted Partial-program names demoted back to **partial**: `box-shadow`, `background`, `background-image`, `overflow-x`, `overflow-y`, `visibility`, `border-collapse`, `writing-mode`.

Phase 68 inventory/reopen policy stays. Phases 69-77 are **not started** again. Phase 67/78 closures are **reopen**.

## Partial program history (before honesty revert)

| When | Implemented | Partial | Unsupported | Ignored |
|------|------------:|--------:|------------:|--------:|
| First catalog (`f0cc352`) | 75 | 45 | 488 | 210 |
| After honesty sync / Phase 57 | 89 | 85 | 397 | 247 |
| Fake Partial close (`fbab822` tags) | 174 | 0 | 397 | 247 |
| Fake Ignored close (pre-revert HEAD) | 421 | 0 | 397 | 0 |
| **After audit revert** | **166** | **9** | **643** | **0** |

Sources: `catalog/coverage-summary.json`, `catalog/mapping.json`, `ignored-inventory.json`.
