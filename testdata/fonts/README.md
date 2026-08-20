# Test fonts

Redistributable fixtures only. **Do not** add Microsoft Georgia or other
proprietary faces without a written redistribution decision and license
record.

| File | License | Notes |
|------|---------|-------|
| `NotoSansKR-HangulSubset.ttf` | OFL (`OFL.txt`) | Tiny CI subset of Noto Sans KR for fixture-27. |
| `LiberationSans-Regular.woff2` | OFL | Liberation Sans as WOFF2 for DecodeWOFF2 / fixture-57. |
| `AbrilFatface-Regular.ttf`, `Acme-Regular.ttf`, … | OFL | Curated static faces for fixture-57 `@font-face` gallery (display, script, mono, serif, sans). |
| `Arimo-Regular.ttf`, `Tinos-Regular.ttf`, `Gelasio-Regular.ttf`, `Cousine-Regular.ttf` | OFL | Metric-compatible open faces used by fixture-57 exact + alias demos. |
| `OFL.txt` | — | SIL Open Font License text. |

The same curated set is also installed under
`~/.local/share/fonts/gowk-fixture57/` on developer machines so
`--use-system-fonts` can discover them.

## Fixture-57

`fixture-57-font-resolution-showcase.html` is a **10-page** showcase of:

- exact `@font-face` (TTF + WOFF2) from this directory
- CSS generics → Liberation
- metric-alias stacks (`Georgia`, `Courier New`, `Arial`, `Times New Roman`)

Pair with `--font-path testdata/fonts`, and optionally
`--use-system-fonts --use-metric-font-aliases` (or
`make samples` / `make samples-metric-aliases`).

## Sample usage (CJK)

Some Simplified Chinese glyphs in fixture-27 are **not** in the KR subset;
samples may also pass `--font-path` to Droid Sans Fallback. Full Noto CJK is
not vendored (size).
