# Fonts, discovery, and Unicode shaping limits

> Phase 19 notes for operators and integrators.

## Bundled faces

By default every PDF embeds the Liberation Sans family (Regular / Bold /
Italic / BoldItalic). Latin-1 text uses a simple TrueType subset with
WinAnsi-style single-byte codes.

## Opt-in discovery

| Flag | Effect |
|------|--------|
| `--font-path DIR` | Scan `DIR` (and shallow children) for `.ttf` / TrueType-flavored `.otf`; repeatable |
| `--use-system-fonts` | Also scan common OS font directories (e.g. `/usr/share/fonts`) |

Discovery is **opt-in** (privacy + startup). CSS `font-family` lists are
matched against discovered family names (name table) before falling back
to Liberation. **CFF / `OTTO` OpenType is rejected** (TrueType outlines only).

Example (CJK / Hangul when installed):

```sh
gowkhtmltopdf --font-path /usr/share/fonts/truetype/droid \
  fixture-27-cjk-fontpath.html out.pdf
# Hangul needs a Hangul-capable face, e.g. fonts-noto-cjk:
#   --font-path /usr/share/fonts/opentype/noto
```

## Type0 / CID path

When a text run contains code points above U+00FF (after punctuation
folding), the writer switches that run onto a **Type0 / CIDFontType2**
sibling resource with Identity-H encoding and Unicode CIDs. Glyphs must
exist in the selected face; Liberation Sans alone will still show `?`
for CJK.

## Honest shaping limits

- **No HarfBuzz / no OpenType shaping** (stdlib-only product constraint).
  Glyphs are placed with advance widths only. Adding HarfBuzz would require
  a plan amendment and a non-stdlib dependency.
- **Arabic / Hebrew:** best-effort **RTL run reverse** at emit time. **Joining,
  ligation, and mark positioning are NOT implemented.**
- **Indic and other complex scripts** are **not claimed**.
- **CJK (Han / kana / hangul)** works when a capable TTF is on the font
  path: characters render, but vertical writing modes are a lite stacked
  glyph path (`writing-mode: vertical-rl|lr`), not full CSS vertical
  typesetting. Hangul requires a Hangul-capable face.
- **Mixed Latin + CJK:** Latin glyphs missing from a CJK face are drawn with
  embedded Liberation; CJK continues on the Type0 sibling of the original face.
- **`@font-face`:** local `url(...ttf|otf)` under ACL; `.woff`/network skipped.
