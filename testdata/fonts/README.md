# Test fonts

Redistributable fixtures only. **Do not** add Microsoft Georgia or other
proprietary faces without a written redistribution decision and license
record.

| File | License | Notes |
|------|---------|-------|
| `NotoSansKR-HangulSubset.ttf` | OFL (`OFL.txt`) | Tiny CI subset of Noto Sans KR for fixture-27. Family: **Noto Sans KR**. Static SFNT (no `fvar`). Covers Latin + Hangul (+ many CJK glyphs). Not a full CJK face. |
| `OFL.txt` | — | SIL Open Font License text for the Noto subset. |

## Catalog intent (phases 3–4)

Tests that need regular / bold / italic / bold-italic, Unicode, composite
glyphs, or duplicate-family cases **reuse** Liberation faces from
`internal/pdf/assets/` (bundled, already licensed) or synthesize temp copies
with patched name tables (e.g. Georgia / Gelasio exact-match regressions).
Those patched copies are created under `t.TempDir()` — they are not vendored
here.

If a Georgia-compatible open font is ever evaluated for bundling, record
license, static/variable status, name-table families, and visual rationale
**before** adding it to this directory.

## Sample usage

Some Simplified Chinese glyphs in fixture-27 (e.g. 汉, 圳) are **not** in the
KR subset; samples may also pass `--font-path` to Droid Sans Fallback so CSS
font-family fallback can supply them. Full Noto CJK is not vendored (size).
