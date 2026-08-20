# Fonts, discovery, and Unicode shaping limits

Operator and integrator notes for the bundled faces, opt-in discovery,
`@font-face`, and shaping. Architecture: [architecture.md](architecture.md).
Product claims: [fidelity.md](fidelity.md).

## Bundled faces

Every default conversion can use:

| Family | Faces | Role |
|--------|-------|------|
| **Liberation Sans** | Regular / Bold / Italic / BoldItalic | Default sans; CSS `sans-serif` / terminal fallback |
| **Liberation Serif** | Regular / Bold / Italic / BoldItalic | CSS `serif` |
| **Liberation Mono** | Regular / Bold / Italic / BoldItalic | CSS `monospace` |
| **DejaVu Sans** | Regular + Bold | Unicode fallback (`system-ui`, last-resort glyphs) |

This is **not** Liberation Sans only. Faces live in `internal/pdf/assets/`
and are parsed by `pdf.LoadDefaultFaces`.

Latin-1 text embeds as a simple TrueType subset with WinAnsi-style
single-byte codes. Runes above U+00FF take the Type0 path below.

## Resolution contract (`pdf.FontResolver`)

Layout and header/footer selection go through one resolver:

1. Exact registered / `@font-face` / `--font-path` family match per CSS token
   (internal name-table family).
2. Continue the author comma stack when a named family is absent.
3. Generics only: `serif` / `sans-serif` / `monospace` → Liberation Serif /
   Sans / Mono; `system-ui` → DejaVu. Real Liberation family names also match.
4. **Legacy display names** (`Georgia`, `Arial`, `Times`, `Courier New`, …)
   are **not** rewritten to Liberation. They win only when a face with that
   family name is supplied; otherwise the stack continues (e.g.
   `Georgia, serif` → Liberation Serif via the generic). Host Fontconfig
   aliases are never imported.
5. Terminal default after an exhausted stack: Liberation Sans
   (`FaceSet.Resolve`). No synthetic bold/italic.
6. Missing glyphs: primary face when it covers the rune → family-stack /
   Liberation weight-style → DejaVu → `Registry.FindWithGlyph`.
7. Before paint, convert/imageout **embed-preflight** the used rune set
   (`pdf.PreflightEmbed`). A failed optional face is marked unavailable and
   the object is re-laid-out onto the next CSS/bundled fallback so metrics
   stay consistent. Claiming profiles still fail closed if nothing embeddable
   remains.

`FaceSet.ResolveFamily` is generics / Liberation names / `system-ui` only.
Do not treat Liberation Serif as “actual Georgia”.

### WeasyPrint / Fontconfig differences (by design)

Same CSS family name does **not** imply the same installed face across
engines:

| Setup | Typical result for `Georgia, serif` |
|-------|-------------------------------------|
| WeasyPrint on Linux | Fontconfig may substitute Gelasio (metric-compatible) |
| Chrome with Georgia installed | Real Georgia |
| Gowkhtmltopdf, no font flags | Stack continues → `serif` → **Liberation Serif** |

That Liberation outcome is the **shipped v0.2.5 contract**, not a regression
against WeasyPrint. Gowkhtmltopdf does not import Fontconfig aliases
(`Georgia → Gelasio`, `Courier New → Cousine`, …) into **default** resolution.
To get an exact named face today, supply it with `--font-path`,
`--use-system-fonts` (when the file’s internal family name matches), or
`@font-face`.

An **opt-in** metric-compatible alias map (inspired by Fontconfig
`30-metric-aliases`, e.g. Georgia→Gelasio when Gelasio is already in the
registry) is available behind `--use-metric-font-aliases` /
`UseMetricFontAliases` (default **false**). Aliases consult the registry only;
the flag alone does not scan disk. Exact family matches still win first.

## Opt-in discovery

Discovery is **opt-in** (privacy + startup). Nothing is scanned unless the
operator asks.

| Flag | Effect |
|------|--------|
| `--font-path DIR` | Scan `DIR` and children to **depth 2** for `.ttf` / `.otf`; repeatable. Primary form is a **directory**. A bare `.ttf`/`.otf` **file** path is accepted as a convenience (loads that one face). Other file paths warn and are skipped — never treated as an empty directory. |
| `--use-system-fonts` | Also scan common OS font directories (e.g. `/usr/share/fonts`). Skips proprietary Windows/corefont trees. Independent of Fontconfig alias rules. |
| `--use-metric-font-aliases` | After an exact registry miss, try curated accepts (Georgia→Gelasio, Courier New→Cousine, …) against **registered** faces only. Default off. |

`pdf.DiscoverFonts` / `ScanFontDirs` read `.ttf` and `.otf` only. Diagnostics
report scanned paths, loaded-face count, skipped-file count, and skip reasons
(no font bytes). **CFF / `OTTO`**, **variable fonts (`fvar`)**, and parse
failures are skipped with a clear reason.

Exact family matching: a directory that supplies a face named Georgia is
selected for `font-family: Georgia`. Supplying only Gelasio does **not**
rename it to Georgia.

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

## Supported format matrix

| Format | Behavior |
|--------|----------|
| TTF (TrueType outlines) | Accepted |
| OTF with TrueType outlines (`0x00010000` / `true`) | Accepted |
| WOFF1 → SFNT (TrueType) | Accepted (`DecodeWOFF`; size/overlap caps in `woff.go`) |
| CFF / `OTTO` OpenType | Rejected / skipped with diagnostic |
| WOFF2 | Accepted (`DecodeWOFF2` via `tdewolff/font.ParseWOFF2`; then `ParseTTF`) |
| EOT | Skipped with diagnostic |
| `data:` `@font-face` src | Skipped with diagnostic |
| Variable fonts (`fvar` table) | **Rejected** with clear diagnostic; use a static face. CI Noto KR subset has no `fvar` and remains valid SFNT. |

## Type0 / CID path

When a text run contains code points above U+00FF (after punctuation
folding), the writer switches that run onto a **Type0 / CIDFontType2**
sibling resource with Identity-H encoding and Unicode CIDs. Glyphs must
exist in the selected face; Liberation Sans alone will still show `?`
for CJK. Mixed Latin + CJK splits: Latin missing from a CJK face is
drawn with bundled Liberation; CJK continues on the Type0 sibling of the
original face.

## `@font-face`

`convert/prepare.MergeFontFaces` registers document faces on **both PDF and
image** paths (including nested HTML header/footer faces on the shared merge
path).

| `src` | Behavior |
|-------|----------|
| `.eot` | **Skipped** (warning) |
| `data:` | **Skipped** (warning) |
| `https://` / `http://` TTF, OTF, WOFF1, WOFF2 | **Fetched** via `Fetch` → `load.FetchSub` — **same ACL, network policy, timeout, and body cap** as CSS/images |
| Local `url(...ttf\|otf\|woff\|woff2)` | Fetched under `--allow-local-files` / `--allow` |
| WOFF1 / WOFF2 | Decompress → `ParseTTF` (TrueType outlines only; CFF/`fvar` still rejected) |

`font-weight` / `font-style` descriptors are retained on `css.FontFace` and
applied as style overrides for `Registry.Lookup` (regular / bold / italic /
bold-italic). When omitted, the face file’s macStyle bits are kept. Multiple
rules for the same family append in source order; equal style scores keep the
earlier face. Unsupported or missing sources warn and fall through to the
author CSS stack / Liberation — they are not conversion errors when a fallback
exists.

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
