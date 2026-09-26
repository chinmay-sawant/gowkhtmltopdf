# Test fonts

`../fixture-27-fonts/DroidSansFallbackFull.ttf` is a subset of Droid Sans
Fallback for fixture 27.
It was made from Debian's `fonts-droid-fallback` copy of
`DroidSansFallbackFull.ttf` with:

```sh
pyftsubset /usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf \
  --text-file=testdata/golden/fixture-27-cjk-fontpath.html \
  --output-file=testdata/fixture-27-fonts/DroidSansFallbackFull.ttf \
  --layout-features='*' --name-IDs='*' --name-languages='*'
```

Droid Sans Fallback is Copyright 2006-2010 Google Corp. and licensed under
Apache 2.0. See `../fixture-27-fonts/APACHE-2.0.txt`.

`NotoSansKR-HangulSubset.ttf` is an OFL TrueType subset of Noto Sans KR
covering Latin, Hangul, and many CJK glyphs used by fixture 27. Its family
name is **Noto Sans KR**. See `OFL.txt`.

Some Simplified Chinese glyphs in fixture 27, including 汉 and 圳, are absent
from the Noto KR subset. The Droid subset supplies them. The fixture 27 tests
scan both directories; other fixtures scan only `testdata/fonts`. This keeps
their font fallback stable without relying on system font packages.

`implemented-audit/` contains Liberation OFL faces for fixtures 60-62. Pin
them with `--font-path testdata/fonts/implemented-audit`. This includes Sans
Regular/Bold/Italic, Serif Regular/Bold/Italic/BoldItalic, and Mono Regular.
