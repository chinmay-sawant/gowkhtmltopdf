# Property counts (v0.2.6)

WebRef inventory size is 818 property names.

## Current snapshot (Advanced Print Waves A-D 2026-08-29)

| Status | Count |
|--------|------:|
| Implemented | 378 |
| Partial | 0 |
| Unsupported | 440 |
| Ignored | 0 |

Phase 80 and Advanced Waves A-D implemented 176 print properties across all waves with full code paths, consumers, and unit tests (PDF bookmarks & outline levels, line clamping, text truncation & ellipsis, universal reset, box decoration break, image orientation & resolution, print color adjust, color scheme, container sizing, variable fonts, OpenType overrides, blend modes, text emphasis, and text decoration skip).

## Unsupported triage (phases 80-84, not started)

The 616 Unsupported names are owned by [unsupported-triage.md](unsupported-triage.md) and phase checklists:

| Phase | Tier | Count |
|------:|------|------:|
| 80 | implement for print | 232 |
| 81 | niche / draft | 94 |
| 82 | alias when base done | 48 |
| 83 | hard defer | 87 |
| 84 | skip print noop | 155 |
| | **sum** | **616** |

Agents must update `catalog/mapping.json` and `catalog/coverage-summary.json` together after any status change. No git commands unless the user explicitly asks.


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
