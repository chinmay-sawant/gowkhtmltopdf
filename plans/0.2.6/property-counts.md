# Property counts (v0.2.6)

WebRef inventory size is 818 property names.

## Current snapshot (Phase 79 closure 2026-08-28)

| Status | Count |
|--------|------:|
| Implemented | 202 |
| Partial | 0 |
| Unsupported | 616 |
| Ignored | 0 |

All 25 former Partial properties promoted to Implemented with full code paths and test proof across Slices 79.1 - 79.5 (linear/radial gradient rasterization, Gaussian blur convolution, inset/multi-layer box-shadow, table border conflict precedence, 2-length border-spacing, independent overflow axes, percent sizing clamps, grid line ends, word-break keep-all, generated content, break-inside avoid-page, vertical writing mode logical mappings).


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
