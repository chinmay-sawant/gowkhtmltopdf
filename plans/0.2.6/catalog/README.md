# Catalog snapshot for v0.2.6 CSS coverage

Counts from `coverage-summary.json`, generated 2026-08-28 by walking `webref-css.json` plus the apply handlers in `internal/layout/style_properties.go` and `internal/layout/style_cascade.go`. Phase 48.2 reclassified print-noop UI and SVG fill/stroke.

| Kind | Total | implemented | partial | unsupported | ignored |
|------|------:|------------:|--------:|------------:|--------:|
| Properties | 818 | 89 | 85 | 397 | 247 |
| At-rules | 55 | 0 | 11 | 34 | 10 |
| Selectors | 158 | 14 | 10 | 113 | 21 |
| Functions | 162 | 5 | 24 | 109 | 24 |
| Units | 30 | 7 | 8 | 15 | 0 |

`ignored` is permanent for this engine: animation, transition, 3D transforms, filter blur, scroll snap, anchor positioning, speech, vendor prefixes, print-noop UI (`cursor`, `caret-color`, `resize`, `user-select`, `pointer-events`, `touch-action`, `appearance`), and SVG presentation `fill`/`stroke` (plus `fill-*` and `stroke-*`). A PDF has no mouse pointer, caret, or drag handle, so those names stay out of the work list.

`unsupported` is the work list. `goal: implement` is 571 properties after Phase 48.2 (was 608 on the first pass; 37 print-noop and SVG rows moved to `ignored`).

Engine apply handlers: 147 named properties after phases 48-56. Custom properties (`--*`) are a separate map.

Check the mapping against apply arms: `python3 scripts/css-catalog-map.py --check`. After a reclassify, rewrite mapping counts with `--write`.

How to add a property: see `plans/0.2.6/AGENTS.md` and the pipeline notes in the canonical ledger. Parser already keeps unknown names. Layout must grow a `ResolvedStyle` field, an apply arm, and a consumer.

Sources and checksums: `SOURCE.md`. Full rows: `mapping.json`.
