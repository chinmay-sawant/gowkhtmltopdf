# Catalog snapshot for v0.2.6 CSS coverage

Counts from `coverage-summary.json`, generated 2026-08-27 by walking `webref-css.json` plus the apply handlers in `internal/layout/style_properties.go` and `internal/layout/style_cascade.go`.

| Kind | Total | implemented | partial | unsupported | ignored |
|------|------:|------------:|--------:|------------:|--------:|
| Properties | 818 | 75 | 45 | 488 | 210 |
| At-rules | 55 | 0 | 4 | 41 | 10 |
| Selectors | 158 | 8 | 5 | 124 | 21 |
| Functions | 162 | 3 | 7 | 128 | 24 |
| Units | 30 | 7 | 6 | 17 | 0 |

`ignored` is permanent for this engine: animation, transition, 3D transforms, filter blur, scroll snap, anchor positioning, speech, vendor prefixes.

`unsupported` is the work list. The first pass marked 608 properties `goal: implement`. That is generous. It still includes print-noop UI (`cursor`, `resize`, `user-select`) and SVG fill/stroke. Phase 48 reclassifies those to `ignored` or a later bucket before anyone implements them.

Engine apply handlers: about 120 named properties. Custom properties (`--*`) are a separate map.

How to add a property: see `plans/0.2.6/AGENTS.md` and the pipeline notes in the canonical ledger. Parser already keeps unknown names. Layout must grow a `ResolvedStyle` field, an apply arm, and a consumer.

Sources and checksums: `SOURCE.md`. Full rows: `mapping.json`.
