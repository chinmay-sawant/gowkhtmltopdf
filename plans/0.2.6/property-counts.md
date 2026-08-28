# Property counts (v0.2.6)

WebRef inventory size is 818 property names. That list is the classification universe, not the implement-everything target. Print-relevant `goal: implement` after Phase 48.2 is 571.

Phases 57-67 promote remaining Partial rows to Implemented (near-browser print). See `phases/phase-57-partial-to-implemented-catchup.md`.

| When | Implemented | Partial | Unsupported |
|------|------------:|--------:|------------:|
| Before this phase (first catalog on `feature/026-extended-css-support`, commit `f0cc352`) | 75 | 45 | 488 |
| After honesty sync (2026-08-28, before Partial program) | 75 | 99 | 397 |
| After Phase 57 catch-up promotions | 89 | 85 | 397 |
| After Phase 58 paint finishes | 102 | 72 | 397 |
| After Phase 59 logical box | 127 | 47 | 397 |
| After Phase 60 text, lists, generated content | 137 | 37 | 397 |
| After Phase 61 overflow, visibility, table | 145 | 29 | 397 |
| After Phase 62 breaks and page | 154 | 20 | 397 |
| After Phase 63 writing-mode vertical | 155 | 19 | 397 |
| After Phase 64 flex near-print | 160 | 14 | 397 |
| After Phase 65 grid near-print | 170 | 4 | 397 |
| After Phase 66 position / transform / stacking | 174 | 0 | 397 |

Sources: `catalog/coverage-summary.json` at `f0cc352` for the before row; current `catalog/coverage-summary.json` / `catalog/mapping.json` for later rows. Ignored counts are out of scope for this table (210 before Phase 48.2 reclassify, 247 now).

