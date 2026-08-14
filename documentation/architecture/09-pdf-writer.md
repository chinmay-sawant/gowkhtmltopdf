# PDF writer (`internal/pdf`)

## 1. Responsibility & position in the pipeline

`internal/pdf` is the **lowermost writer layer** of gowkhtmltopdf: it turns the
painted output of the layout engine into a **well-formed, viewer-openable
PDF file** (PDF 1.4 by default, or PDF 1.7 opt-in via `WriterPolicy`), entirely
with the Go standard library plus a single narrow exception for OpenType shaping
(`go-text/typesetting`, see §5 and
[`plans/0.2.0/amendments/2026-08-05-gotext-typesetting.md`](../../plans/0.2.0/amendments/2026-08-05-gotext-typesetting.md)).

It owns everything that appears after the paint step of the canonical
pipeline `load → parse → style → layout → paginate → paint → write`:

- the **version policy & serialization** (`WriterPolicy`, `PDF14` default,
  `PDF17` opt-in, reserved `PDF20`; `%PDF-1.4` or `%PDF-1.7` header; classic
  xref table; trailer with deterministic `/ID` on 1.7);
- the **document object model** (indirect objects, pages tree, catalog,
  info dictionary with Latin-1 or UTF-16BE + BOM strings, non-claiming XMP
  metadata stream on 1.7, outline tree);
- the **page content-stream language** — every PDF operator emitted for text,
  vector graphics, images, and clipping, with fonts/images registered into
  per-page `/Resources`;
- the **font pipeline**: parsing TrueType/OpenType (TrueType outlines only),
  WOFF1 unwrapping, rune-set subsetting, simple WinAnsi/Latin-1 fonts and
  **Type0 / CIDFontType2** fonts with Identity-H encoding, `/Widths` in the
  PDF 1000-unit em glyph space, and `ToUnicode` CMaps;
- **text shaping** for emission: OpenType (GSUB/GPOS) via the pure-Go
  HarfBuzz port, with a manual presentation-form/Arabic/RTL fallback;
- **image embedding** (`DCTDecode` JPEG pass-through, Flate RGB PNG
  re-encoding with alpha soft-masks) under hard size caps;
- **link annotations, named destinations (via `/Dest`), and outlines
  (bookmarks)** wired through the catalog.

The package is deliberately a *writer*: it never parses HTML/CSS and never
performs layout. It consumes a stream of paint operations (`Content` methods)
and hands back a serialized PDF. Both PDF mode (`internal/convert`) and the
raster image mode (`internal/imageout`) depend on it — image mode reuses the
font parsing, glyph metrics, and shaping pipeline while *rasterizing* instead
of writing PDF bytes.

### Output coordinate space and canvas mapping

PDF uses a bottom-left origin with positive y-coordinates extending upward (y-up
in typographical points, 72 pt/inch), whereas HTML/CSS layout operates from a
top-left origin with positive y extending downward (y-down in points). The conversion
layer translates canvas positions to PDF coordinates via
`hfGeom.pdfY(page, y) = pageH - marginTop - (y - page * contentH)`. Header and
footer bands are painted into their respective top and bottom margin strips
(`[pageH - marginTop, pageH]` for headers, `[0, marginBottom]` for footers) with
origins and baselines clamped to the margin box.

Two invariants shape everything in this package:

1. **Workable output is the bar.** The repo history records that earlier
   iterations produced PDF *files* that real viewers rejected (empty/malformed
   catalogs, wrong stream compression, broken font advances). `internal/pdf`
   is where that last-mile correctness lives: catalog/outline wiring order,
   RFC 1950 zlib streams (not raw DEFLATE), 1000-unit-em widths, correct
   `xref` offsets, Latin-1 (not UTF-8) string bytes.
2. **Determinism.** Identical input + settings must produce byte-identical
   output; creation time is injectable (a zero value pins a fixed date).
   Golden tests depend on this (§9).

## 2. Package / file map

| File | Lines | Responsibility |
|------|------:|----------------|
| `policy.go` | 134 | `WriterPolicy`, `PDFVersion` (`PDF14`, `PDF17`, reserved `PDF20`), policy validation, header/producer version resolution, feature gates |
| `pdf.go` | 1198 | Document object model, page objects, `finalize` (catalog/pages/info/outlines/XMP metadata), serialization with counting xref offsets, RFC 1950 flate pool, `ReorderPages`/`DuplicatePage`, outline tree finalization, `pdfString`/`utf16BEString`/`winAnsiFold` encoding, trailer `/ID` |
| `semantic.go` | 275 | In-tree semantic PDF parser for structural testing and validation of emitted output |
| `content.go` | 672 | Content-stream builder: every PDF operator (`q/Q`, `rg/RG`, `m/l/c/re/f/S/W n`, `cm`, `BT/ET/Td/Tm/TL/Tc/T*`, `Tj`, `Tr`), font rune recording for subsetting, mixed Latin/CJK run splitting, image-resource registration |
| `fonts.go` | 777 | TrueType/OpenType table-directory parsing (`head`, `maxp`, `hhea`, `hmtx`, `OS/2`, `post`, `name`, `cmap` formats 0/4/6/12), the `Font` struct (metrics, advances, composite-glyph traversal, fingerprint), PDF-em metric conversion |
| `subset.go` | 758 | The subsetter: collect used glyphs (incl. composite children), strip hinting bytecode, remap composite references, rebuild cmap format 4 with delta segments, assemble a minimal SFNT with correct `checksumAdjustment` and 4-byte-aligned `loca` |
| `images.go` | 441 | JPEG pass-through embedding (`jpegScan` for SOF dimensions), PNG decode → RGB Flate XObject with alpha `SMask`, grayscale Rec.601 fold at embed time, size caps (`maxEmbedded*`) |
| `shape.go` | 377 | No-face shaping fallback: combining-mark strip, Arabic presentation-form joining (incl. Lam-Alef), RTL run reversal; `ShapeNeeded` heuristic |
| `shape_gotext.go` | 462 | OpenType shaping via `go-text/typesetting` (`shaping.HarfbuzzShaper`, `Segmenter` pools), reverse cmap glyph→Unicode mapping, CJK `halt`/`palt` font features, CSS `font-feature-settings` parser |
| `glyph.go` | 409 | `glyf` outline decoding to contour points (simple + composite, incl. F2DOT14 scales), `FlattenContour` quadratic-to-polyline conversion |
| `registry.go` | 326 | Opt-in font registry: family-name index, CSS-lookup (`Lookup`), last-resort glyph fallback (`FindWithGlyph`), system font-dir scanning (`ScanFontDirs`, depth-limited) |
| `fonttype0.go` | 225 | `ensureFont` dispatch (simple vs Type0), `FontFile2` + `FontDescriptor` embedding, Type0/CIDFontType2 + Identity-H + `CIDToGIDMap` emission, `/W` width runs |
| `faces.go` | 212 | `FaceSet` (Liberation Sans/Serif/Mono + DejaVu fallback), lazy `LoadDefaultFaces` via `sync.Once`, CSS family/weight/italic resolution |
| `fontpdf.go` | 185 | PDF name tokens, 1000-em width conversion (`widthsInEm`, `subsetWidths`), `ToUnicode` CMap emission, rune-set cache keys |
| `numbers.go` | 133 | Shared numeric constants (PDF metrics, sfnt/glyf/cmap geometry, JPEG markers, shaping feature helpers, Arabic tiers) |
| `policy_test.go` | 170 | Tests for `WriterPolicy`, version validation, reserved `PDF20` rejection, feature gates |
| `pdf_test.go` | 980 | Header/xref/trailer `/ID`/determinism, content operators, links, outlines, info dict, UTF-16BE strings, XMP metadata stream, reorder/duplicate validation, rich-document structure, short-writer contract |
| `semantic_test.go` | 120 | Tests for semantic parser against PDF 1.4 and 1.7 emitted files |
| `struct_test.go` | 283 | `TestRichDocStructure`, `TestWriteToContract`, `TestWriteRejectsShortWriter`, `TestSubsetGlyfFourByteAligned` |
| `font_test.go` | 415 | Parse defaults, cmap formats, subsetting/checksum, font cache identity, mixed Latin/CJK, `TestDirectModuleAllowlist` |
| `fonttype0_test.go` | 245 | Type0 CJK embedding, mixed Latin fallback, `ToUnicode` coverage |
| `shape_test.go` | 276 | RTL/Arabic/lam-alef, OT-vs-fallback behavior, feature parsing, module allowlist |
| `image_test.go` | 475 | JPEG scan, PNG/JPEG embedding, resource-name collision, grayscale folds |
| `woff_test.go` | 235 | WOFF round-trip, OTTO rejection, overlap rejection, WOFF2 gap |
| `subset_align_test.go` | 73 | 4-byte glyf alignment of subsets (CJK viewer correctness) |
| `faces_test.go` | 159 | Default faces load, resolution aliases, Unicode fallback coverage |
| `registry_lookup_test.go` | 58 | Family matching order (exact before generic) |
| `bench_test.go` | 70 | `BenchmarkWrite50Pages`, `BenchmarkShapeRun` |
| `doc.go` | 3 | Package doc (stub retained from phase scaffold) |
| `assets/assets.go` | 95 | `//go:embed` bundled Liberation Sans/Serif/Mono (regular/bold/italic/bold-italic) + DejaVuSans Unicode fallback; every accessor returns an isolated `bytes.Clone` copy |

Total ≈ 9,500 lines (≈ 13% of the ~73k Go LOC, and the largest single
writer component).

## 3. Key types, functions & entry points

### Document model (`pdf.go`, `policy.go`)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `PDFVersion` | `policy.go:19` | Version enum: `PDF14` (default), `PDF17` (opt-in), reserved `PDF20` |
| `WriterPolicy` | `policy.go:45` | Serialization policy: `Version`, validation, feature gates, header/producer version strings |
| `Document` | `pdf.go:114` | Document under construction: `policy WriterPolicy`, `objects []*object`, `pages []*Page`, `info map[string]string`, `outlineRoot *Outline`, `fontCache` (subset key → font dict ref), document-wide rune sets, `catalogRef`/`infoRef`/`metadataRef` (set at finalize), `finalized` flag |
| `NewDocument` | `pdf.go:142` | Empty document with default `PDF14` policy; compression on by default |
| `NewDocumentWithPolicy` | `pdf.go:154` | Empty document configured with explicit `WriterPolicy`; validates policy and rejects unsupported versions |
| `(d) Policy` | `pdf.go:166` | Returns the document's active `WriterPolicy` |
| `(d) SetCompression / SetGrayscale / SetCreationTime / SetInfo` | `pdf.go:170-182` | Hooks used by the convert pipeline before painting |
| `(d) AddPage(w, h)` | `pdf.go:245` | Allocates the page object **and its content-stream object** up front; page refs are stable from here on |
| `(d) PageRef / PageAt` | `pdf.go:267/277` | Read page refs/objects after all `AddPage` calls (used to wire outline destinations and annotations) |
| `(d) ReorderPages(order)` | `pdf.go:290` | Permutation of page order for TOC-first assembly and copies/collate; the pages tree is a flat single-level `/Kids` list, so reordering `d.pages` is sufficient |
| `(d) DuplicatePage(i)` | `pdf.go:330` | Clones a page for copies: same size, a **new** `/Contents` object with the same stream bytes, independent annotation copies, copied (not aliased) resource maps |
| `(d) Write / WriteTo` | `pdf.go:388/395` | Serialize without staging a second full copy in memory; xref offsets come from the `countingWriter` |
| `Page` | `pdf.go:223` | `Content() *Content`, `AddLinkURI(rect, uri)`, `AddLinkDest(rect, page, x, y)` |
| `Outline` | `pdf.go:374` | Bookmark node: `Title`, `PageRef` (set by caller after layout), `X, Y`, `Children`; ref assigned at finalize |
| `(d) SetOutline` | `pdf.go:384` | Installs the outline tree; must happen before `Write` |
| `SortOutlines` | `pdf.go:1186` | Deterministic outline ordering by (page, y-down, x) — used by layout/outline |
| `(d) finalize` | `pdf.go:530` | One-time assembly: catalog/info/pages/metadata refs, rune union, pages tree, **outlines before catalog**, per-page resources/annots |
| `(d) writeTo` | `pdf.go:490` | Header (`writePDFHeader`), object loop, xref + trailer (`writePDFTrailer`) |

### Content builder (`content.go`)

`Document`, `Page`, and `Content` have single-goroutine ownership during
assembly and finalization. Callers must not mutate or serialize one document
concurrently; parallel conversions should use separate documents. The PDF
writer does not expose a partially synchronized concurrency contract.

| Symbol | Location | Purpose |
|--------|----------|---------|
| `Content` | `content.go:41` | Stream buffer + font/image resource maps + font-state stack; owns `curFont`/`curSize` and a `fontStack` saved across `q/Q` so a skipped redundant `Tf` is never emitted against a stale font |
| `NewContent` | `content.go:88` | Empty builder |
| Graphics state | `content.go:150-210` | `Save/Restore` (`q/Q`), `SetFillColor`/`SetStrokeColor` (`rg/RG`, grayscale fold here when `doc.grayscale`), `SetLineWidth` (`w`), `SetOpacity` (`/opacity gs` + ExtGState) |
| Path ops | `content.go:212-260` | `MoveTo/LineTo/CurveTo/Rect/Fill/Stroke/Clip` (`m/l/c/re/f/S/W n`) |
| `Transform` | `content.go:263` | 6-element `cm` CTM (used for images, page islands, and vector transforms) |
| Text ops | `content.go:269-333` | `SetFont` (`Tf`, dedupes identical state), `BeginText/EndText`, `TextAt` (`Td`), `TextMatrix` (`Tm`), `TextLeading` (`TL`), `SetCharSpacing` (`Tc`), `TextNextLine` (`T*`), `TextRenderMode` (`Tr`, mode 2 = fake bold) |
| `UseEmbeddedFont` | `content.go:302` | Registers a parsed TTF under a resource name; runes drawn under it are subset into the PDF |
| `TextShow` | `content.go:414` | The text emitter: ASCII fast path; otherwise shape, decide Type0, split mixed runs; records runes for the subsetter |
| `(c) fonts()` | `content.go:635` | Lazy font-object allocation — **one subset per font per document** via `ensureFont`; a subset failure propagates (text naming a missing `/Resources` entry is invisible) |
| `uniqueImageName` | `images.go:438` | Suffix-collision-free page-local image names for body + header/footer bands |

### Fonts (`fonts.go`, `faces.go`, `registry.go`, `woff.go`)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `Font` | `fonts.go:41` | Parsed TrueType view: raw bytes, `fingerprint` (sha256 of file), `PostScriptName`, unitsPerEm/ascender/descender/capHeight/bbox/macStyle, table map, `cmap`, per-glyph advances + LSB; immutable derived caches (`sync.Once` go-text face, reverse cmap) |
| `ParseTTF` | `fonts.go:80` | Validates sfnt magic (`0x00010000`/`true`), rejects `OTTO` (CFF), parses tables in dependency order with fail-fast errors |
| `(f) GlyphID / Advance / AdvanceInPoints / GlyphAdvancePoints` | `fonts.go:530-684` | Rune → glyph id, advances in font units and points — the shared metrics surface for layout and both output sinks |
| `(f) LoadNames / FamilyNames` | `fonts.go:550/541` | Name-table decode (NameIDs 1/4/6/16, UTF-16BE); fills `PostScriptName` from ID 4/6 when empty |
| `(f) GlyphContours / FlattenContour` | `glyph.go:15/291` | Outline decoding used by the image rasterizer (`ttfraster`), not the PDF writer |
| `ParseFontBytes` | `woff.go:41` | Dispatch: TTF/OTF unchanged, WOFF1 → `DecodeWOFF`, WOFF2 → explicit error (no Brotli in the allowlist) |
| `DecodeWOFF` | `woff.go:243` | stdlib-zlib WOFF1 → SFNT with hard caps: table count ≤ 1024, per-table ≤ 16 MiB, reconstructed ≤ 32 MiB, overlap rejection |
| `FaceSet` / `LoadDefaultFaces` / `DefaultFont` | `faces.go:14/42/205` | Bundled Liberation + DejaVu faces, lazily parsed once (process-wide cache via `sync.Once`) |
| `Registry` / `Lookup` / `FindWithGlyph` | `registry.go:16/71/120` | Opt-in family index for `--font-path`/`--use-system-fonts`; last-resort glyph fallback |
| `ScanFontDirs` / `DefaultSystemFontDirs` | `registry.go:270/241` | Depth-limited directory scan (`.ttf`/`.otf`), **opt-in only** — nothing is scanned by default |

### Shaping (`shape.go`, `shape_gotext.go`)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `ShapeRun` | `shape_gotext.go:51` | Returns `ShapedRun{Text, Runes, Advances}` — one shape pass feeding both PDF and raster emitters, so advances can never drift from the shaped text |
| `ShapeTextFont(WithFeatures)` | `shape_gotext.go:74/80` | OT path when the face has GSUB (or features requested); else fallback; CJK `halt`/`palt` enabled implicitly |
| `ParseFontFeatureSettings` | `shape_gotext.go:196` | CSS `font-feature-settings` → `shaping.FontFeature` (4-letter tags only) |
| `ShapeText` | `shape.go:18` | No-face fallback: orphan-mark strip → Arabic presentation forms → RTL run reverse |
| `ShapeNeeded` | `shape.go:366` | Script heuristic (RTL ranges, combining marks) |

### Font embedding (`fonttype0.go`, `fontpdf.go`, `subset.go`)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `(d) ensureFont` | `fonttype0.go:34` | Subsets `f` for `used` once per document (cache key = mode + fingerprint + base name + rune key), then dispatches to simple or Type0 emission |
| `(d) embedFontFile` | `fonttype0.go:89` | Shared `FontFile2` (Flate) + `FontDescriptor` used by both font modes |
| `(d) emitSimple` | `fonttype0.go:118` | Simple TrueType: `/FirstChar /LastChar /Widths []` (char-code indexed, 0 for missing), `/Encoding /WinAnsiEncoding`, `/ToUnicode` |
| `(d) emitType0` | `fonttype0.go:140` | Type0 + CIDFontType2: Identity-H, `CIDToGIDMap` (kept **uncompressed** — some viewers mishandle Flate maps), `/W` width runs, `/ToUnicode` |
| `subsetFont` | `subset.go:38` | Rune→glyph collection (walking composite children), hint stripping, outline cloning + component remap, cmap4 rebuild, SFNT assembly with checksum adjustment |
| `(s) build` | `subset.go:372` | Table assembly: head (indexToLocFormat=1), hhea/maxp (patched counts), hmtx, cmap4, **long loca always**, glyf 4-byte padded, OS/2, post |
| `widthsInEm` / `subsetWidths` | `fontpdf.go:18/35` | The single home of font-units → PDF 1000-em conversion for both simple `/Widths` and Type0 `/W` |
| `(d) ensureToUnicode` | `fontpdf.go:60` | `Adobe-Identity-UCS` CMap; 1-byte codespace for simple, 2-byte for Identity-H; bfchar chunks of 100 |
| `pdfString` | `pdf.go:809` | Literal-string encoding: code points ≤ U+00FF become single bytes (matches subset cmap + widths); above folds via `winAnsiFold` or `?` — raw UTF-8 bytes caused mojibake |

### Images (`images.go`)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `(c) AddJPEGImage` | `images.go:154` | `jpegScan` (single marker walk for dims + component count), DCTDecode pass-through XObject, grayscale re-encode when enabled, `cm` + `Do` |
| `(c) AddPNGImage` | `images.go:279` | png.Decode → Flate RGB XObject, optional alpha `SMask`, grayscale fold |
| `validateEmbeddedImage` | `images.go:47` | Caps: encoded ≤ 32 MiB, dims ≤ 16 384, pixels ≤ 16 Mpix, decoded working set ≤ 128 MiB |

## 4. Data & control flow

### 4.1 External flow (who calls what)

```
internal/convert (pipeline)          internal/layout              internal/pdf
─────────────────────────            ──────────────────          ─────────────────
convert.go:338  doc := pdf.NewDocument()
                ...
convert.go:549  layout.PaintContext(ctx, run.doc, lres, …)
        │  ── pages / content ops ───────────────────────────►   AddPage, Content().*
        │                                                          (per-page UseEmbeddedFont,
        │                                                           SetFont, TextShow, Rect…,
        │                                                           SetFillColor…, image adds)
pdfPipeline.Assemble:
  assembleTOC      → doc.ReorderPages(tocFirstOrder)
  assembleOutline  → emitOutline → doc.SetOutline(root)
  assembleLinks    → doc.PageAt(i).AddLinkURI/AddLinkDest
  assembleDocument → doc.SetInfo(Title/Producer),
                     doc.SetCompression, SetGrayscale,
                     doc.SetCreationTime(run.req.now())
  assembleCopies   → materializeCopies → doc.DuplicatePage → doc.ReorderPages
  assembleHeadersFooters → draws H/F bands into doc via layout
pdfPipeline.Finalize → doc.Write(run.req.Output)  ──► serialized bytes
```

Notes on that seam:

- `internal/imageout` uses only the **fonts + shaping + glyph-outline**
  surface of `internal/pdf` (`pdf.DefaultFont`, `pdf.ShapeRun`,
  `pdf.ParseFontBytes`, `pdf.FlattenContour`, `pdf.Registry`,
  `pdf.DefaultSystemFontDirs`) and never builds a PDF.
- `internal/layout` uses `pdf.Font`/`pdf.FaceSet`/`pdf.Registry` for face
  selection/metrics and paints through `Page.Content()`; it must register
  every font it draws with `UseEmbeddedFont` under a resource name, then
  address it by that name in `SetFont` (see `paint.go` `resName` helpers).
- Determinism is injected at the pipeline: `Request.now()` (`convert.go:68`)
  returns the typed request's `Now` func or `time.Now()`; `pdf.Document`
  itself defaults to a fixed 2000-01-01 date when `SetCreationTime` is never
  called (`pdf.go:629` `infoDict`).

### 4.2 Inside `finalize` (single assembly pass, `pdf.go:502`)

1. **Preflight**: error if already finalized or zero pages (`errPDFNoPages`).
2. **Allocate** `catalogRef`, `infoRef`, `pagesRef` first — page dicts
   reference `/Parent` (`pagesRef`), so the ref must exist.
3. **`unionFontRunes`**: consolidates the per-page rune sets recorded during
   painting into a document-wide sorted rune union per font resource, and
   precomputes each font's subset cache key + Type0 decision. One subset per
   font across all pages instead of near-identical per-page subsets.
4. **Pages tree**: flat single-level `/Kids [pageRefs…]` + `/Count`.
5. **Outlines before Catalog** (explicit ordering constraint, commented in
   code): `finalizeOutlines` assigns each node its ref and serializes
   `/First /Last /Prev /Next /Parent /Title /Dest`; only after `refStr` is
   set can `catalogDict` safely write `/Outlines` — a malformed empty value
   made viewers show nothing/fail to open.
6. **Catalog** (`/Type`, `/Pages`, optional `/Outlines`, `/PageMode /UseOutlines`, optional `/Metadata` on 1.7) and **Info** (Title/Subject/Author/Keywords when set + forced `Creator`, `Producer` per policy e.g. `"gowkhtmltopdf 1.4"` or `"gowkhtmltopdf 1.7"`, `CreationDate`, `ModDate`). On 1.7, non-PDFDocEncoding Info and outline strings use UTF-16BE + BOM (`FE FF`), and a non-claiming XMP Metadata stream object is attached to `/Metadata`.
7. **Per page**: flate the content stream (`/Filter /FlateDecode` when
   compression on), attach stream, build `/Resources` from `content.fonts()`
   (may allocate + subset font objects) + image XObjects + optional
   ExtGState, emit `/Annots` for link annotations.

### 4.3 `Write` / `writeTo` (streaming, `pdf.go:490`)

`writeTo` never assembles a second full byte slice in memory:

1. `finalize()`.
2. Header: `%PDF-1.4` or `%PDF-1.7` (via `policy.HeaderVersion()`) + binary comment `%\xe2\xe3\xcf\xd3`.
3. Object loop through a `countingWriter` (records exact byte offsets;
   turns silent short writes into `io.ErrShortWrite`). Objects allocated but
   never materialized (`dict == ""`) are **skipped** and their xref entries
   left unrecorded so they cannot point at the next object.
4. xref section: entry `0` is the free-list head (`0000000000 65535 f`), each
   object entry `%010d 00000 n`; trailer `/Size /Root /Info`, plus deterministic `/ID [ <a> <b> ]` on PDF 1.7; `startxref`;
   `%%EOF`.

### 4.4 Text emission path (`TextShow`, `content.go:414`)

```
TextShow(str)
   │  pure ASCII? ─────────────► textShowSimple (folds/escapes + records runes,
   │                              Latin-1 literal string + Tj)
   ▼
ShapeTextFont(str, fnt)   (OT via go-text when GSUB; else ShapeText fallback)
   ▼
textNeedsType0?  ───────── no ─► textShowSimple
   ▼ yes
emitTextRuns(splitType0Runs(str, fnt))
   │   codes > U+00FF (or missing on CJK face) → Type0 run
   │   Latin fallback on a CJK face → Liberation face ("FL")
   ├─ textShowType0: SetFont(base+"_u"), hex <CID> text (CID == Unicode
   │    code point under Identity-H), record runes under the _u resource
   └─ textShowSimple on base (or "FL")
```

The `_u` resource-name suffix marks the Type0 sibling of a base face; both
share the same underlying font file, and `useEmbeddedFont` keeps the Type0
subset tied to the original Unicode face (not `FL`).

### 4.5 Font subsetting flow (`ensureFont` → `subsetFont`)

```
ensureFont(fnt, name, used)
  key := v{0|1}|{fingerprint:%x}|{baseName}|{runesKey}      // cache identity
  fontCache[key] hit? ──► return existing ref
  scope := subsetSimple | subsetUnicode
  subsetFont(fnt, used, scope)
     collectUsedGlyphs (runes → glyf ids, incl. composite children + .notdef)
     sort glyphs (deterministic)
     collectGlyphData (advances, lsbs, raw outlines)
     cloneOutlines: strip hints + remap composite component ids
     rebuild cmap4 (coalesce consecutive code/glyph runs into delta segments)
     build: head/hhea/maxp patched, hmtx, cmap, long loca (4-byte aligned),
            glyf, OS/2, post; table checksums + head checksumAdjustment
  embedFontFile(sub) → FontFile2 (Flate) + FontDescriptor (1000-em metrics)
  emitSimple | emitType0
  fontCache[key] = ref
```

## 5. Cross-package dependencies

### What `internal/pdf` imports

- **Stdlib only** for the write path: `bytes`, `compress/zlib` (RFC 1950),
  `encoding/binary`, `crypto/sha256`, `image`/`image/color`/`image/jpeg`/
  `image/png`, `os`, `path/filepath`, `sort`, `strconv`, `strings`, `sync`,
  `time`, `errors`, `fmt`, `io`, `math`, `slices`, `unicode`, `unicode/utf8`,
  `unicode/utf16`.
- **One external direct dependency**: `github.com/go-text/typesetting`
  (`di`, `font`, `font/opentype`, `shaping`) — the repo's fixed, allowlisted
  text-shaping exception (see `shape_gotext.go` import nolint and
  `go.mod:8`). `TestDirectModuleAllowlist` (`shape_test.go:187`) enforces
  that only `go-text/typesetting` and `tdewolff/canvas` are direct requires
  anywhere in the module and documents the CGO-HarfBuzz rejection.
- **`internal/pdf/assets`** (embedded font bytes) — the only internal import.

### What depends on `internal/pdf`

Non-test importers:

| Importer | Use |
|----------|-----|
| `internal/convert` (+ `prepare/`, `render/plan.go`, `hf.go`, `toc.go`, `links.go`, `outline.go`, `page_islands.go`, `page_plan.go`) | `pdf.NewDocument`, `DefaultFont`, `Registry`, layout paint into `*pdf.Document`, TOC/outline/links/copies/headers-footers assembly, `doc.Write` |
| `internal/layout` (`layout.go`, `inline_paint.go`, `paint.go`) | `pdf.Font`/`FaceSet`/`Registry` face selection and metrics; painting into `Page.Content()` |
| `internal/imageout` (`imageout.go`, `ttfraster.go`) | `pdf.Font` parsing, `pdf.ShapeRun`/`ShapeTextFont`, `pdf.FlattenContour` for glyph rasterization, `pdf.Registry`/`DefaultSystemFontDirs` |

### Import-direction rule

`internal/pdf` is a **leaf of the dependency graph** (below an
implementation detail: it imports only stdlib, the one allowlisted shaping
module, and its own assets). `layout` and `convert` sit *above* it; nothing
inside `internal/pdf` knows HTML, CSS, settings, or CLI. Any change to the
paint surface (`Content` API) ripples into `layout/paint.go` and, for fonts
and shaping, into `imageout`. The unit-test files inside the package may
also be imported by other packages' tests (e.g. layout golden tests construct
`pdf.Document`s directly) — a sign the package is the shared writer/test
foundation.

## 6. Design decisions & trade-offs

| Decision | Rationale / trade-off |
|----------|------------------------|
| **PDF 1.4 default, opt-in PDF 1.7 via `WriterPolicy`** | Minimal, deterministic, widely readable; xref/trailer written in one pass. Classic xref maintained on both versions (no object/xref streams). Trailer `/ID`, Info + UTF-16BE strings, and non-claiming XMP emitted on 1.7. PDF 2.0 (#32) and PDF/A / PDF/UA (#33) are explicit separate tracks. |
| **One-time `finalize` with strict object-ordering constraints** | Catalog/outline wiring must be ordered (outlines → catalog); refs are allocated before dicts that reference them. The alternative (post-hoc patch refs) is rejected — it historically produced malformed catalogs. |
| **`countingWriter` for xref offsets** | Streams output without a second in-memory copy; turns silent short writes into errors so a truncated stream never gets a "valid" xref. |
| **RFC 1950 zlib for all `/FlateDecode`** | PDF spec requires zlib wrapper, not raw DEFLATE; raw streams made pages render empty. Compressors are pooled per page (`flatePool`). |
| **TrueType outlines only; CFF/`OTTO` rejected** | Subsetting re-quires glyf/loca surgery; CFF hinting is out of scope. Clear, early error instead of a broken embed. |
| **WOFF1 in-tree, WOFF2 rejected** | WOFF1 needs only stdlib zlib; WOFF2 needs Brotli (not allowlisted). Hard decompress-bomb caps + overlap rejection treat fonts as untrusted input (THREAT-MODEL §fonts). |
| **Subset at finalize, one subset per font per document** | Document-wide rune union (instead of per-page subsets) minimizes embedded font size and makes subset cache keys stable/deterministic. |
| **Simple Latin-1 fonts + Type0/CIDFontType2 for anything above U+00FF** | Latin-1 path: small, golden-test-friendly, WinAnsi single-byte codes, `/Widths` indexed by char code. Higher Unicode: Identity-H CIDs equal to Unicode code points — no CID renumbering, but requires `CIDToGIDMap` + `ToUnicode`. |
| **`winAnsiFold` + `?` fallback instead of raw UTF-8 bytes** | Literal strings are byte-oriented; emitting UTF-8 made viewers show mojibake and missing glyphs. Folding common punctuation (en/em dash → `-`, curly quotes, bullets → middle dot) keeps the subset cmap and `/Widths` consistent with the emitted bytes. |
| **Hinting stripped from subsets; long `loca` always** | Subsets omit `fpgm`/`prep`/`cvt`, so leftover instructions garbled CJK composites (broken 東京都 in PDFium); unaligned glyf offsets had the same effect. 4-byte alignment + format-1 loca fixed viewer rendering. |
| **Mixed Latin/CJK run splitting with Liberation fallback** | CJK faces often lack Latin glyphs; ASCII that maps to `.notdef` would become tofu. Runs are split (`splitType0Runs`) and missing Latin falls back to the embedded Liberation face (`FL`), all within one `Tj` sequence. |
| **OT shaping via pure-Go HarfBuzz port, only when GSUB present** | Real CGO HarfBuzz is explicitly out of scope (`CGO_ENABLED=0`); reverse-cmap restores Unicode CIDs; manual presentation-form Arabic/RTL fallback covers no-GSUB faces. Indic remains Partial. |
| **JPEG pass-through vs PNG re-encode** | JPEG bytes are embedded as-is (`/DCTDecode`) — no decode/re-encode loss in the normal path; PNG is decoded and re-encoded as Flate RGB (with `SMask` when alpha), bounded by strict caps. Grayscale folds Rec.601 luma at embed/paint time. |
| **Flat, single-level pages tree + page permutation** | TOC-first ordering and copies/collate become a simple `ReorderPages` permutation; pages own their content/annots so duplication doesn't need deep tree surgery. |
| **Bundled Liberation + opt-in system fonts** | Product rule: dependable Latin output with zero system-font dependence; `--font-path`/`--use-system-fonts` are operator-controlled and skip proprietary trees (privacy + licensing). |
| **Deterministic by default, time injectable** | Zero `creationTime` pins 2000-01-01; typed requests inject `Now` for byte-stable CI/golden output. |

## 7. Notable patterns & invariants

- **Typed object references** (`objRef`, `pdf.go:77`): "the `N 0 R`
  spelling is a formatting concern, not a data type" — refs can't be
  malformed; `parseRef` validates string refs from the exported surface.
- **Ordered `dict` builder** (`pdf.go:99`): PDF dict syntax/escaping lives in
  one place instead of ~20 `fmt.Sprintf` sites.
- **Resource-name discipline**: fonts and images are addressed by *name*
  (e.g. `F1`, `I0`, `FL`, `base_u`); the page `/Resources` dict is built at
  finalize from what content actually used. A font whose subset fails
  propagates an error because text naming a missing resource renders
  invisible (`content.go:635`).
- **Document-wide rune union + deterministic keys**: sorted runes, sorted
  glyph ids, sorted string keys everywhere (`sortedStringKeys`,
  `sortedGlyphs`, `runesKey`) — determinism is structural, not incidental.
- **Fingerprint cache keys**: two faces may share a `PostScriptName` but
  differ in bytes; subset cache keys include the sha256 fingerprint so the
  cache never merges them (`fontCacheSeparatesLoadedFacesWithSameDisplayName`).
- **Entity-local caches on `Font`** via `sync.Once` (`gotFace`, `rev`):
  derived data lives on and dies with the font that derives from it (no
  package-level `sync.Map`).
- **Fail-fast, specific errors** for fonts (`errFontCFFNotSupported`,
  `errFontTruncatedHmtx`, …) and WOFF (`errWOFFOverlap`, …) — bad input
  never degenerates into a corrupt-ish output.
- **Extension points are intentionally tiny**: the writer exposes a
  paint API + settings hooks (compression/grayscale/time/info), not a plugin
  framework. New PDF features mean edits inside this package plus an update
  to the compatibility matrix.
- **`q/Q` state pairing with font-state tracking**: `Save` records the
  active font so `Restore` never lets a later `SetFont` dedupe against a
  stale state.

## 8. Security considerations

Security posture for fonts/images is specified in
[`documentation/THREAT-MODEL.md`](../THREAT-MODEL.md) (§ Fonts, § Images);
`internal/pdf` is where the technical enforcement lives:

- **Fonts are untrusted parse input.** `@font-face` `url(...)` bytes, WOFF1
  payloads, and `--font-path` files all pass through size-bounded parsers:
  - WOFF1: table-count ≤ 1024, per-table ≤ 16 MiB, reconstructed SFNT ≤
    32 MiB, **overlapping compressed tables are rejected**, decompressed
    length must exactly equal the declared `origLength`
    (`woff.go`, caps at `woff.go:15-20`).
  - WOFF2 is rejected outright — no Brotli in the module allowlist
    (`errWOFF2Unsupported`).
  - TrueType tables are bounds-checked on every read
    (`errFontTooShort/TruncatedDirectory/TruncatedHmtx/…`); malformed fonts
    fail the conversion rather than corrupting the PDF.
  - Remote `https://` `@font-face` is **not fetched** (product policy,
    THREAT-MODEL); only local/`file:`/`data:` under the ACL.
- **System-font discovery is opt-in only** (`--use-system-fonts`,
  `DefaultSystemFontDirs`): nothing is scanned at startup (privacy + startup
  cost + avoids surprise opens of proprietary trees).
- **Image embedding caps** (`images.go:47`): encoded ≤ 32 MiB, dimension ≤
  16 384, decoded working set ≤ 128 MiB — bounded decompression for PNG and
  bounded JPEG header scans (no full decode in the normal pass-through path).
- **Deterministic metadata**: no timestamps or absolute paths leak into the
  PDF unless set; `Producer` is fixed; `CreationDate` defaults to a fixed
  date, and in the pipeline comes from `Request.now()`.
- **Filenames/paths never enter the PDF**; outline `/Dest` targets are
  validated as page refs within object bounds (`outlineDest`) so a bogus
  `PageRef` fails the document instead of emitting a corrupt `/Dest`.

## 9. Testing & verification

The package is validated at three levels:

1. **Unit tests inside the package** (≈ 2,900 lines of tests):
   - Serialization correctness: `TestWriteHeaderAndTrailer`, `TestXrefOffsets`
     (every `n` xref entry must point at the start of `"N 0 obj"`),
     `TestEmptyDocFails`, `TestWriteRejectsShortWriter` (silent short writes
     surface as errors, xref integrity), `TestDeterministicOutput`
     (byte-identical PDFs across two builds).
   - Structure: `TestRichDocStructure` (2-page doc exercising every feature:
     graphics, text, links, outlines, images), `TestWriteToContract`,
     `TestSubsetGlyfFourByteAligned`.
   - Fonts/subsetting: `TestParseDefaultFont`, `TestParseCmapFormats`,
     `TestSubsetFont`, `TestSubsetChecksum`, `TestCompositeSubset`,
     `TestFontCacheSharedAcrossPages`, `TestFontCacheSeparatesLoadedFacesWithSameDisplayName`,
     `TestType0CJKEmbedding`, `TestType0MixedLatinFallback`,
     `TestSubsetWidthsArePDFUnits`, `TestPDFStringLatin1NotUTF8`.
   - Images: `TestJPEGScan`, `TestAddJPEGImage`, `TestAddPNGImage`,
     `TestAddPNGNoAlpha`, `TestImageResourceNamesDoNotCollideAcrossBands`,
     `TestAddInvalidImage`, `TestGrayscalePNGFold/JPEGFold/PNGAlphaKept`.
   - WOFF: `TestDecodeWOFFRoundTripParseTTF`, `TestDecodeWOFFRejectsOTTO`,
     `TestDecodeWOFFRejectsOverlap`, `TestDecodeWOFF2Gap`,
     `TestParseFontBytesTTFUnchanged`.
   - Shaping: `TestShapeTextRTLReverse`, `TestArabicJoiningBehProducesConnectedForms`,
     `TestArabicLamAlefLigature`, `TestShapeTextFontArabicOTJoining`,
     `TestShapeTextFontArabicOTLamAlef`, `TestShapeTextFontFallsBackWithoutFace`,
     `TestShapeTextFontLatinUnchanged`, `TestIndicCombiningNotDroppedMidWord`,
     `TestCJKPunctFontFeatures`, `TestParseFontFeatureSettings`.
   - Policy: `TestDirectModuleAllowlist` shells out to `go list -m` and
     fails if any third-party direct require appears beyond
     `go-text/typesetting` and `tdewolff/canvas`.
   - Pages/annotations: `TestLinkAnnotations`, `TestOutlines`,
     `TestOutlineBadPageRefFails`, `TestReorderPagesKidsOrder`,
     `TestDuplicatePageOwnsResourceMaps`, `TestReorderPagesValidation`,
     `TestInfoDict`.
2. **Cross-package golden tests**: `internal/convert/golden_test.go` (and the
   many `internal/layout/*_test.go` files) build full documents through the
   real pipeline and compare committed sample PDFs under `testdata/` — the
   end-to-end "openable in a viewer" gate. Test names in fixture tests
   reference workable-PDF regressions (the feature history of this package
   is largely a series of viewer-correctness fixes).
3. **Benchmarks**: `BenchmarkWrite50Pages` (write throughput), `BenchmarkShapeRun`
   (shaping hot path).

`Makefile` remains the operator entry point (`make test`, `make samples`),
and `make samples` regenerates committed `output/` PDFs/PNGs from fixtures as
a manual viewer check.

## 10. Known limitations, deferred items & open questions

Cross-reference [`documentation/deferred.md`](../deferred.md),
[`documentation/fidelity.md`](../fidelity.md),
[`documentation/compatibility-matrix.md`](../compatibility-matrix.md),
[`documentation/fonts.md`](../fonts.md).

- **PDF 1.4 default, PDF 1.7 opt-in**: no PDF 1.5+ object streams, no compression of xref
  tables, no incremental update/append, no linearization. Every conversion
  is a full regenerate. PDF 2.0 (ISO 32000-2 / UTF-8 strings) is tracked in #32;
  PDF/A-4 and PDF/UA-2 (claiming XMP, OutputIntents, structure tree) are tracked in #33.
- **CFF/PostScript-outline OpenType is rejected** (`errFontCFFNotSupported`);
  only TrueType outlines embed or subset. OTF-flavored fonts can only be
  used accidentally-fail today — a deliberate scope cut, documented in
  `fonts.md`.
- **WOFF2 not supported** (Brotli not allowlisted); WOFF1 only. Remote
  `https://` `@font-face` is never fetched.
- **Simple fonts are WinAnsi/Latin-1 single-byte** with punctuation folding;
  code points that fold or hit `?` lose fidelity. Full
  PDFDocEncoding/TrueType-encoding tables are not implemented.
- **Complex scripts are Partial**: Arabic/Hebrew work via OT (GSUB) or
  presentation-form fallback; **Indic shaping is Partial** without a GSUB
  face; no CGO HarfBuzz ever.
- **Grayscale is a paint-time Rec.601 fold**, not a colorspace-level conversion
  — matches the documented promise (`SetGrayscale`), but no ICC/CMYK/DeviceN
  support exists for color workflows.
- **Image support covers JPEG + PNG**: no GIF/TIFF/BMP/WebP/SVG-as-PDF-vector
  (SVG is rasterized upstream via `tdewolff/canvas` in `internal/svg` before
  reaching the writer as a PNG-equivalent image).
- **Transparency**: alpha soft masks exist; PDF 1.4 transparency *groups*
  (blend modes) are not implemented — only per-image `SMask`.
- **No real text extraction fidelity beyond `ToUnicode`**: extraction works
  for the covered runes; folded punctuation extracts as its ASCII stand-in.
- **Deterministic default date is a fixed constant (2000-01-01)**: consumers
  who want wall-clock dates must set `CreationTime`/inject `Now`, otherwise
  two conversions of the same input are byte-identical but "stale" dated.
- **Open questions:** (a) whether post-MVP Type0 work should switch from
  Identity-H + CIDToGIDMap to a compacted CID space (smaller fonts, more
  code); (b) whether an opt-in CFF subsetter is worth the complexity vs.
  the clean rejection today; (c) whether `/Widths`-space compression
  (`/W` arrays with run-length `c [w1 … wn]` groups) is needed for large CJK
  documents (currently each CID emits a `cid [w]` pair, which grows linearly).
- **Deferred roadmap context:** `plans/0.2.0/10-canonical-post-mvp-roadmap.md`
  phases 10–23 outline where Unicode/CID fonts and richer CSS /
  positioning land; this package is where those land first.

## 11. Related documents

- [`../architecture.md`](../architecture.md) — package map and pipeline
  overview (this document expands its "PDF writer notes" section).
- [`../library-api.md`](../library-api.md) — how the public API configures
  compression/grayscale/copies/outlines that surface here.
- [`../fonts.md`](../fonts.md) — font discovery, Type0/CID path, honest
  shaping limits.
- [`../THREAT-MODEL.md`](../THREAT-MODEL.md) — font/image trust boundaries
  enforced in this package.
- [`../fidelity.md`](../fidelity.md) + [`../compatibility-matrix.md`](../compatibility-matrix.md)
  — what output features are claimed.
- [`../deferred.md`](../deferred.md) — deferred capability list.
- [`../../plans/0.1.0/00-canonical-pure-go-rewrite.md`](../../plans/0.1.0/00-canonical-pure-go-rewrite.md)
  and [`../../plans/0.2.0/amendments/2026-08-05-gotext-typesetting.md`](../../plans/0.2.0/amendments/2026-08-05-gotext-typesetting.md)
  — plan/amendment ledgers behind this package.
- Architecture deep-dives in this directory: `01-entrypoints-cli.md`,
  `02-library-api.md`, `03-settings.md`, `04-load.md`, `05-html-parser.md`,
  `06-css.md`, `07-layout.md`, `08-convert-pipeline.md`,
  `10-imageout-svg.md` — the writer is the sink for `07` and `08`, and the
  shared font/shaping foundation for `10`.
