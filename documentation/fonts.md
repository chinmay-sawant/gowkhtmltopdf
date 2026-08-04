# Fonts, discovery, and Unicode shaping limits

> Phase 19 notes for operators and integrators.

## Bundled faces

By default every PDF embeds the Liberation Sans family (Regular / Bold /
Italic / BoldItalic). Latin-1 text uses a simple TrueType subset with
WinAnsi-style single-byte codes.

## Opt-in discovery

| Flag | Effect |
|------|--------|
| `--font-path DIR` | Scan `DIR` (and shallow children) for `.ttf` faces; repeatable |
| `--use-system-fonts` | Also scan common OS font directories (e.g. `/usr/share/fonts`) |

Discovery is **opt-in** (privacy + startup). CSS `font-family` lists are
matched against discovered family names (name table) before falling back
to Liberation.

Example (CJK):

```sh
gowkhtmltopdf --font-path /usr/share/fonts/truetype/droid \
  fixture-27-cjk-fontpath.html out.pdf
```

## Type0 / CID path

When a text run contains code points above U+00FF (after punctuation
folding), the writer switches that run onto a **Type0 / CIDFontType2**
sibling resource with Identity-H encoding and Unicode CIDs. Glyphs must
exist in the selected face; Liberation Sans alone will still show `?`
for CJK.

## Honest shaping limits

- **No HarfBuzz / no OpenType shaping.** Glyphs are placed left-to-right
  with advance widths only.
- **CJK (Han / kana / hangul)** works when a capable TTF is on the font
  path: characters render, but vertical writing modes, ruby, and complex
  line-breaking are not claimed.
- **Arabic, Indic, and other complex scripts** are **not claimed**: no
  reordering, ligation, or mark positioning.
- **`@font-face` network downloads** are out of scope; local `src` may
  land later under the same ACL rules as other file loads.
