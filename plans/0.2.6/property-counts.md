# Property counts (v0.2.6)

WebRef inventory size is 818 property names.

## Current snapshot

| Status | Count |
|--------|------:|
| Implemented | 174 |
| Partial | 0 |
| Unsupported | 397 |
| Ignored | 247 |

Phases **57-67** closed the Partial set (near-browser print). Phases **68-78** reopen **all 247 Ignored** names for **browser-level print** (phase-wise checklist). See `phases/phase-68-ignored-inventory-policy.md` and `ignored-inventory.json`.

After Phase 68 reclassification (planned): Ignored moves onto the work list as Unsupported, so expected counts become **174 / 0 / 644 / 0** until later phases promote names to Implemented.

## Partial program history (Implemented / Partial / Unsupported)

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

Ignored was 210 at first catalog, then 247 after Phase 48.2 print-noop / SVG reclassify. Sources: `catalog/coverage-summary.json` / `catalog/mapping.json`.
