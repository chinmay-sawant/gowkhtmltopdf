# Implemented honesty pass (2026-08-28)

Four explore subagents audited every `engine_status: implemented` property (166 at start) for apply arm + consumer + matrix agreement (`HONESTY-GATES.md`).

## Result counts

| Status | Before | After |
|--------|-------:|------:|
| Implemented | 166 | 152 |
| Partial | 9 | 23 |
| Unsupported | 643 | 643 |
| Ignored | 0 | 0 |

## Demoted to Partial (14)

| Property | Reason |
|----------|--------|
| `accent-color` | progress/meter only; matrix Partial |
| `border-spacing` | single float; no H/V pair |
| `break-inside` | only `avoid` honored in pagination |
| `content` | pseudo path only; no ResolvedStyle apply arm |
| `grid-column-end` | bare line folds to start |
| `grid-row-end` | bare line folds to start |
| `opacity` | matrix Partial |
| `max-width` | percent not in block clamp |
| `max-inline-size` | same as max-width |
| `min-height` | percent not in block height constraints |
| `min-block-size` | same as min-height |
| `overflow-wrap` | matrix Partial |
| `word-wrap` | alias; same |
| `word-break` | keep-all treated as normal |

## Kept Implemented

152 properties kept (apply + consumer matched claimed print subset). Residual notes (not demoted): `list-style-image` inheritance gap when set only on `ul`.

## Method

Chunks in `/tmp/impl_chunk_{0..3}.json`. No DEMOTE_UNSUPPORTED in this pass.
