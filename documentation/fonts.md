# Fonts, discovery, and Unicode shaping limits

> Phase 19 notes for operators and integrators.
> Plan amendment: [`plans/amendments/2026-08-04-shaping-stdlib.md`](../plans/amendments/2026-08-04-shaping-stdlib.md).

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

- **No HarfBuzz / no OpenType GSUB/GPOS** (stdlib-only + `CGO_ENABLED=0`).
  Real HarfBuzz remains out of scope; see the plan amendment above.
- **Arabic / Hebrew:** RTL run reverse **plus** best-effort **Arabic
  presentation-form joining** (initial/medial/final/isolated) and Lam-Alef
  ligatures in `ShapeText`. This is **not** OpenType shaping — faces without
  Presentation Forms glyphs will still look disconnected.
- **Indic and other complex scripts** are **not claimed** (combining marks
  kept after base; no matra reordering).
- **CJK (Han / kana / Hangul)** works when a capable TTF is on the font
  path. `writing-mode: vertical-rl|lr` rotates ideographic / Hangul / kana
  glyphs 90° (sideways) and stacks Latin upright — not full CSS vertical
  typesetting.
- **Mixed Latin + CJK:** Latin glyphs missing from a CJK face are drawn with
  embedded Liberation; CJK continues on the Type0 sibling of the original face.
- **`@font-face` (PDF Partial):** local `url(...ttf|otf)` under loader ACL
  (`--enable-local-file-access` / `--allow`) is fetched via `FetchSub`,
  parsed, and registered for the document. **Image mode does not call
  `mergeFontFaces` (N/A).** `.woff` / `.woff2` / `https://` / `data:` src are
  skipped with a warning. `font-weight` / `font-style` on `@font-face` are
  parsed but **ignored** at register time (alias is family name only).
  Remote webfonts and WOFF decode remain deferred.
