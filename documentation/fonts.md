# Fonts, discovery, and Unicode shaping limits

> Phase 19 notes for operators and integrators.
> Plan amendments: [`2026-08-04-shaping-stdlib.md`](../plans/amendments/2026-08-04-shaping-stdlib.md)
> (interim), [`2026-08-05-gotext-typesetting.md`](../plans/amendments/2026-08-05-gotext-typesetting.md)
> (OT via typesetting).

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

Example (CJK / Hangul):

```sh
gowkhtmltopdf --font-path /usr/share/fonts/truetype/droid \
  --font-path testdata/fonts \
  fixture-27-cjk-fontpath.html out.pdf
# Production Hangul: fonts-noto-cjk / any Hangul-capable TTF on --font-path
#   --font-path /usr/share/fonts/opentype/noto
```

`testdata/fonts/NotoSansKR-HangulSubset.ttf` is a tiny OFL subset for CI /
fixture-27 smoke only — not a full CJK face.

## Type0 / CID path

When a text run contains code points above U+00FF (after punctuation
folding), the writer switches that run onto a **Type0 / CIDFontType2**
sibling resource with Identity-H encoding and Unicode CIDs. Glyphs must
exist in the selected face; Liberation Sans alone will still show `?`
for CJK.

## Honest shaping limits

> Amendment: [`plans/amendments/2026-08-05-gotext-typesetting.md`](../plans/amendments/2026-08-05-gotext-typesetting.md)
> (`CGO_ENABLED=0`; allowlisted module only).

- **OpenType shaping via [`go-text/typesetting`](https://github.com/go-text/typesetting)**
  when the active face has a **GSUB** table. `TextShow` / `ShapeTextFont` run
  the pure-Go HarfBuzz port, then reverse-cmap shaped glyphs to Unicode CIDs
  for Type0 Identity-H. Real CGO HarfBuzz remains out of scope.
- **Arabic / Hebrew:** OT joining + ligation (e.g. Lam-Alef) when GSUB is
  present and reverse-cmap covers the glyphs. **Fallback** (no face / no GSUB
  / unmapped glyph): RTL run reverse plus best-effort **presentation-form**
  joining in `ShapeText`. Faces without Presentation Forms **and** without
  usable GSUB reverse-cmap will still look disconnected.
- **Indic and other complex scripts:** **Partial** — OT applies when the face
  and reverse-cmap succeed; production Indic quality is **not** claimed beyond
  that (fallback keeps combining marks after the base; no in-tree matra
  reordering).
- **CJK (Han / kana / Hangul)** works when a capable TTF is on the font
  path. `writing-mode: vertical-rl|lr` rotates ideographic / Hangul / kana
  glyphs 90° (sideways) and stacks Latin upright — not full CSS vertical
  typesetting.
- **Mixed Latin + CJK:** Latin glyphs missing from a CJK face are drawn with
  embedded Liberation; CJK continues on the Type0 sibling of the original face.
- **`@font-face` (Partial):** local `url(...ttf|otf)` under loader ACL
  (`--enable-local-file-access` / `--allow`) is fetched via `FetchSub`,
  parsed, and registered for the document on **both PDF and image** paths
  (`convert.MergeFontFaces`). `.woff` / `.woff2` / `https://` / `data:` src
  are skipped with a warning. `font-weight` / `font-style` on `@font-face`
  are parsed but **ignored** at register time (alias is family name only).
  Remote webfonts and WOFF decode remain deferred.
