# Property counts (v0.2.6)

WebRef inventory size is 818 property names.

## Current snapshot (Implemented honesty pass 2026-08-28)

| Status | Count |
|--------|------:|
| Implemented | 177 |
| Partial | 25 |
| Unsupported | 616 |
| Ignored | 0 |

Four explore subagents audited all former Implemented rows (apply arm + consumer + matrix agreement). **14** over-claims demoted to Partial (list in `implemented-honesty-pass.md`). No DEMOTE_UNSUPPORTED.


## Partial program history (before honesty revert)

| When | Implemented | Partial | Unsupported | Ignored |
|------|------------:|--------:|------------:|--------:|
| First catalog (`f0cc352`) | 75 | 45 | 488 | 210 |
| After honesty sync / Phase 57 | 89 | 85 | 397 | 247 |
| Fake Partial close (`fbab822` tags) | 174 | 0 | 397 | 247 |
| Fake Ignored close (pre-revert HEAD) | 421 | 0 | 397 | 0 |
| **After audit revert** | **166** | **9** | **643** | **0** |
| **After Implemented honesty pass** | **152** | **23** | **643** | **0** |

Sources: `catalog/coverage-summary.json`, `catalog/mapping.json`, `ignored-inventory.json`.
