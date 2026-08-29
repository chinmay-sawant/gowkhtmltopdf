# Property counts (v0.2.6)

WebRef inventory size is 818 property names.

## Current snapshot (Phase 84 Skip Print Noop 2026-08-29)

| Status | Count |
|--------|------:|
| Implemented | 356 |
| Partial | 0 |
| Unsupported | 462 |
| Ignored | 0 |

Phase 84 audited and locked all 155 skip-print-noop properties (45 time/animation/transition, 41 scroll snap/overscroll, 25 pointer/form UI, 21 anchor/timeline/motion, 19 speech/aural, 4 3D transforms) as honest Unsupported with empty code paths and clear per-bucket notes. Phase 70 SVG presentation properties (fill, stroke, stroke-width, fill-opacity, stroke-opacity) remain Implemented with honest ResolvedStyle notes.

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
