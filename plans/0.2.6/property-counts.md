# Property counts (v0.2.6)

WebRef inventory size is 818 property names. That list is the classification universe, not the implement-everything target. Print-relevant `goal: implement` after Phase 48.2 is 571.

Phases 57-67 promote remaining Partial rows to Implemented (near-browser print). See `phases/phase-57-partial-to-implemented-catchup.md`.

| When | Implemented | Partial | Unsupported |
|------|------------:|--------:|------------:|
| Before this phase (first catalog on `feature/026-extended-css-support`, commit `f0cc352`) | 75 | 45 | 488 |
| After honesty sync (2026-08-28, before Partial program) | 75 | 99 | 397 |
| After Phase 57 catch-up promotions | 89 | 85 | 397 |

Sources: `catalog/coverage-summary.json` at `f0cc352` for the before row; current `catalog/coverage-summary.json` / `catalog/mapping.json` for later rows. Ignored counts are out of scope for this table (210 before Phase 48.2 reclassify, 247 now).
