# Test fonts

`NotoSansKR-HangulSubset.ttf` — OFL TrueType subset of Noto Sans KR (variable
TTF) covering Latin + Hangul (+ many CJK glyphs) used by fixture-27. Family
name: **Noto Sans KR**. See `OFL.txt`.

Some Simplified Chinese glyphs in fixture-27 (e.g. 汉, 圳) are **not** in this
KR face; samples also pass `--font-path` to Droid Sans Fallback so CSS
font-family fallback can supply them. Full Noto CJK is not vendored (size).
