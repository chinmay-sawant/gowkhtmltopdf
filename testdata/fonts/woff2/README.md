# WOFF2 test font

`LiberationSans-Regular-latin.woff2` (30 KB) is a Latin subset of the bundled
`internal/pdf/assets/LiberationSans-Regular.ttf` (Liberation OFL, same terms
as `../implemented-audit/LICENSE.txt`), Brotli-compressed in a WOFF2 container.
Family: **Liberation Sans**; PostScript name: **LiberationSans**. Covers
U+0020-007E, U+00A0-00FF, and common punctuation (dashes, curly quotes,
ellipsis). The uncompressed subset is 50 KB.

Regenerate (fontTools 4.61.1 + brotli):

```sh
pyftsubset internal/pdf/assets/LiberationSans-Regular.ttf \
  --unicodes='U+0020-007E,U+00A0-00FF,U+2013-2014,U+2018-2019,U+201C-201D,U+2026' \
  --output-file=/tmp/LiberationSans-Regular-latin.ttf
python3 - <<'PY'
from fontTools.ttLib import TTFont
font = TTFont("/tmp/LiberationSans-Regular-latin.ttf")
font.flavor = "woff2"
font.save("testdata/fonts/woff2/LiberationSans-Regular-latin.woff2")
PY
```

SHA-256 of the committed file:
`14088dcd0a16ada031485426b245ee59360ad0683389c04b2477f3e6766451d9`.

## roboto-mono-v13-latin-500.woff2

`roboto-mono-v13-latin-500.woff2` (12 KB) is the exact face the w3schools.com
audit reported as failing to decode (row w3schools-6, evidence dir
`plans/0.2.7/real-sites/w3schools/evidence/2026-09-16-audit2/`). It is the
Google Fonts Roboto Mono v13 latin 500 subset, served at
`https://www.w3schools.com/lib/fonts/roboto-mono-v13-latin-500.woff2`,
licensed Apache-2.0. Fetched 2026-09-16.

SHA-256 of the committed file:
`34e45e19c86321affecb63210e78cc2b706041dc27ba7074050767805433b5ff`.

The face is spec conforming: its transformed glyf table sets the explicit
bbox bit for all 65 composite glyphs. The decode failure comes from an
upstream `tdewolff/parse` bitmap reader off-by-one; see
`TestDecodeWOFF2W3SchoolsRobotoMono` in `internal/pdf/woff_test.go` and
`fix-report-woff2.md` in the evidence dir above.

