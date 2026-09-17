# Fonts, discovery, and Unicode shaping limits

Operator and integrator notes for the bundled faces, opt-in discovery,
`@font-face`, and shaping. Architecture: [architecture.md](architecture.md).
Product claims: [fidelity.md](fidelity.md).

## Bundled faces

Every default conversion can use:

| Family | Faces | Role |
|--------|-------|------|
| **Liberation Sans** | Regular / Bold / Italic / BoldItalic | Default sans; `sans-serif`, Arial, Helvetica, … |
| **Liberation Serif** | Regular / Bold / Italic / BoldItalic | `serif`, Times, Georgia, … |
| **Liberation Mono** | Regular / Bold / Italic / BoldItalic | `monospace`, Courier, Consolas, … |
| **DejaVu Sans** | Regular + Bold | Unicode fallback (`system-ui`, last-resort glyphs) |

This is **not** Liberation Sans only. Faces live in `internal/pdf/assets/`
and are parsed by `pdf.LoadDefaultFaces`.

Latin-1 text embeds as a simple TrueType subset with WinAnsi-style
single-byte codes. Runes above U+00FF take the Type0 path below.

## CSS generic / common-name mapping

`pdf.FaceSet.ResolveFamily` (used by layout after the opt-in registry):

| CSS token | Bundled face |
|-----------|----------------|
| `serif`, `georgia`, `times`, `times new roman`, `liberation serif` | Liberation Serif |
| `monospace`, `courier`, `courier new`, `consolas`, `monaco`, `liberation mono` | Liberation Mono |
| `sans-serif`, `arial`, `helvetica`, `tahoma`, `verdana`, `calibri`, `liberation sans` | Liberation Sans |
| `system-ui` | DejaVu Sans |

Named families that are not in this table are **not** rewritten to Liberation.
They resolve as named against the opt-in registry first; if nothing matches,
layout falls through the author’s comma stack and then to Liberation Sans.

Missing glyphs walk: CSS family (registry, then bundled) → Liberation
weight/style → DejaVu Regular/Bold → any opt-in registry face that covers the
codepoint (`FindWithGlyph`, prefers DejaVu/Noto names).

## Opt-in discovery

Discovery is **opt-in** (privacy + startup). Nothing is scanned unless the
operator asks.

| Flag | Effect |
|------|--------|
| `--font-path DIR` | Scan `DIR` and children to **depth 2** for `.ttf` / `.otf`; repeatable |
| `--use-system-fonts` | Also scan common OS font directories (e.g. `/usr/share/fonts`). Skips proprietary Windows/corefont trees |

`pdf.ScanFontDirs` reads `.ttf` and `.otf` only. **CFF / `OTTO` OpenType is
rejected** (TrueType outlines only). A file that fails `ParseTTF` is skipped.

Example (CJK / Hangul):

```sh
gowkhtmltopdf --font-path /usr/share/fonts/truetype/droid \
  --font-path testdata/fonts \
  fixture-27-cjk-fontpath.html out.pdf
# Production Hangul: any Hangul-capable TTF on --font-path
#   --font-path /usr/share/fonts/opentype/noto
```

`testdata/fonts/NotoSansKR-HangulSubset.ttf` is a **tiny CI subset** for
fixture-27 smoke — not a full CJK face. Full Noto CJK is not shipped.

## Type0 / CID path

When a text run contains code points above U+00FF (after punctuation
folding), the writer switches that run onto a **Type0 / CIDFontType2**
sibling resource with Identity-H encoding and Unicode CIDs. Glyphs must
exist in the selected face; Liberation Sans alone will still show `?`
for CJK. Mixed Latin + CJK splits: Latin missing from a CJK face is
drawn with bundled Liberation; CJK continues on the Type0 sibling of the
original face.

## `@font-face`

`prepare.ResourceContext.MergeFontFaces` registers document faces on **both PDF and
image** paths.

| `src` | Behavior |
|-------|----------|
| `https://` / `http://` TTF, OTF, WOFF1, WOFF2 | **Fetched** via `Fetch` → `load.FetchSub` - **same ACL, network policy, timeout, and body cap** as CSS/images |
| Local `url(...ttf\|otf\|woff\|woff2)` | Fetched under `--allow-local-files` / `--allow` |
| `data:` | Decoded and registered when the embedded format is WOFF2/WOFF1/TTF/OTF; skipped with a metadata-only warning otherwise |
| WOFF1 | Decompress → `ParseTTF` (TrueType outlines only) |
| WOFF2 | Brotli decompress plus `glyf`/`loca` reconstruction via `github.com/tdewolff/font`, then `ParseTTF` (TrueType outlines only); the decoder caps memory at 30 MiB |
| WOFF2 (known upstream gap) | Some bitmap-heavy faces fail to decode: tdewolff/parse refuses the last bit of a bitmap whose length is an exact multiple of 8, so the trailing composite glyph loses its bbox and tdewolff/font rejects the glyph. The Roboto Mono fixture (224 glyphs) is pinned as a graceful skip: warning, no panic, no partially decoded face (`internal/pdf/woff_test.go:270`) |
| `.eot`, SVG fonts | **Skipped** (warning). `.eot` has an explicit policy check; SVG font payloads do not parse as TTF/OTF/WOFF |

`font-weight` / `font-style` / `unicode-range` on `@font-face` are parsed
(`css.go:567`, `fontface_descriptors.go:39`) and now select faces instead of
being dropped. The descriptors travel as a `pdf.FaceSpec` (`face_spec.go:19`)
through `prepare.mergeFontFace` (`styles.go:451`) into
`Registry.AddFamilyAliasSpec` (`registry.go:115`). `Registry.LookupRune`
(`registry.go:200`) resolves one code point at a time: faces whose declared
range excludes it are not candidates, and a face that actually maps it beats
one that only declares it. That per-code-point path is the mechanism a family
split into `unicode-range` partitions (the Google Fonts `latin` and
`latin-ext` pattern) needs to paint each rune with its declared face. Absent
descriptors keep the old behavior: the face's file-declared weight/style and
full coverage.

## Honest shaping limits

OpenType shaping uses [`go-text/typesetting`](https://github.com/go-text/typesetting)
when the active face has a **GSUB** table. `TextShow` / `ShapeTextFont` run
that shaper, then reverse-cmap shaped glyphs to Unicode CIDs for Type0
Identity-H. **There is no CGO HarfBuzz.**

- **Arabic / Hebrew:** OT joining + ligation (e.g. Lam-Alef) when GSUB is
  present and reverse-cmap covers the glyphs. **Fallback** (no face / no GSUB
  / unmapped glyph): RTL run reverse plus best-effort **presentation-form**
  joining in `ShapeText`. Faces without Presentation Forms **and** without
  usable GSUB reverse-cmap will still look disconnected.
- **Indic and other complex scripts:** **Partial** — OT applies when the face
  and reverse-cmap succeed; production Indic quality is **not** claimed
  (fallback keeps combining marks after the base; no in-tree matra
  reordering).
- **CJK (Han / kana / Hangul)** works when a capable TTF is on the font
  path. `writing-mode: vertical-rl|lr` is **parsed** but lays out
  **horizontal**. There is no rotated CJK / vertical typesetting path.
- **IPA / uncommon Unicode:** when the CSS `font-family` face and Liberation
  lack a glyph, layout falls back to DejaVu (bundled) and then to any
  covering face on the opt-in registry. Use `--use-system-fonts` or
  `--font-path` for extra coverage — see [cli.md](cli.md#url-mode--chrome-strip---simplify-dom).
- **OpenType `halt` / `palt`:** requested via typesetting `FontFeatures` for
  CJK / East-Asian punctuation runs in `ShapeTextFont`, and via
  `ParseFontFeatureSettings` / `ShapeTextFontWithFeatures` when CSS
  `font-feature-settings` is supplied.

## Image mode

Image output uses the **same** faces, metrics, and `pdf.ShapeRun` shaping as
PDF. Glyphs are filled from TTF outlines with coverage AA on a 2×
supersampled canvas. The 5×7 bitmap font is not the primary path.
See [architecture.md](architecture.md#image-mode).
