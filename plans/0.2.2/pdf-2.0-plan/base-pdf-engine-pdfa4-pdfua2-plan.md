# gocorepdfengine — PDF Engine Plan (PDF/A-4 + PDF/UA-2)

**Status:** Draft — engine-layout scan of gopdfsuit  
**Source of truth scanned:** `gopdfsuit/internal/pdf/**`, `internal/models`, compliance tests under `test/`  
**Scope of this document:** PDF **engine only** (byte-level document construction). Not HTTP handlers, frontend, merge/redact product surface, or language bindings.  
**Style:** Architecture, objects, dictionary keys/tags, and metadata. No implementation code.

---

## 1. Purpose

Rebuild a native PDF generator in the spirit of gopdfsuit’s engine: template/content → binary PDF 2.0, with first-class **PDF/A-4** (ISO 19005-4 / PDF 2.0 archival) and **PDF/UA-2** (ISO 14289-2:2024 accessibility) modes.

The existing entry point is effectively:

| Surface | Role |
|--------|------|
| `GeneratePDF` / `GenerateTemplatePDF` | Allocate buffer, call borrow path, return `[]byte` |
| `GenerateTemplatePDFBorrowed` | Real pipeline: page manager, content, fonts, catalog, structure, trailer |
| Config flags | `pdfaCompliant`, `taggedPDF`, `arlingtonCompatible`, `embedFonts`, nested `pdfa` |

For gocorepdfengine, treat those as **engine modes**, not JSON product config.

---

## 2. Inventory of source modules (what was scanned)

### 2.1 Core generation & document shell

| Area | Location in gopdfsuit | Responsibility |
|------|----------------------|----------------|
| Generator pipeline | `internal/pdf/generator.go` | Header, object ID reservation, catalog/pages, fonts, images, XMP/ICC, structure tree, xref, trailer |
| Page / content | `internal/pdf/pagemanager.go` | Pages, content streams, annots, StructParent indexes |
| Drawing / layout | `internal/pdf/draw.go` | Tables, text, borders, marked content (BDC/EMC), artifacts |
| Helpers / types | `internal/pdf/helpers.go`, `types.go`, `utils.go`, `pdf.go` | Dimensions, text escape, shared types |
| XRef | `internal/pdf/xref/` | Offset table writing patterns |

### 2.2 PDF/A-4 compliance

| Area | Location | Responsibility |
|------|----------|----------------|
| XMP + OutputIntent path A | `internal/pdf/metadata.go` (`PDFAHandler`) | Templated XMP, ICC stream, OutputIntent, catalog extras |
| XMP + ICC path B | `internal/pdf/pdfa.go` | Alternate XMP builder, sRGB/Gray ICC profile builders, OutputIntent object |
| Liberation fonts | `internal/pdf/font/pdfa.go` | Map standard fonts → Liberation TTF, download/ensure, register for embedding |
| Font emit | `internal/pdf/font/metrics.go`, `subset.go`, `registry.go`, `ttf.go` | Type0/CIDFontType2, widths, ToUnicode, FontFile2 subset |

### 2.3 PDF/UA-2 structure / tagging

| Area | Location | Responsibility |
|------|----------|----------------|
| Structure model | `internal/pdf/structure.go` | Structure types, MCID, ParentTree, Document root, Link elements |
| Structure emit | `generator.go` (`formatStructElem*`, ParentTree, Namespace) | StructElem objects, StructTreeRoot, ParentTree Nums |
| Content tags | `draw.go` + structure Begin/End APIs | `/Tag << /MCID n >> BDC` … `EMC` |
| Links / annots | `links.go`, page annot bookkeeping | Link StructElem + `/OBJR`, page `/Tabs /S` |

### 2.4 Related engine features (secondary for v1 compliance, still part of “engine”)

- Images: `image.go` (XObject, ColorSpace rewrite to ICCBased under PDF/A)
- Encryption: `encryption/` (catalog/trailer `/Encrypt`; note PDF/A restrictions on encryption)
- Digital signature: `signature/` (AcroForm Sig fields; orthogonal to A-4/UA-2 core)
- Forms: `form/`, widget dicts with `/V`, `/AS` (AcroForm value tags — not PDF version `/V`)
- Bookmarks/outlines: `outline.go`, `bookmarks.go`
- SVG form XObjects: `svg/`

### 2.5 Compliance validation already in gopdfsuit

- `test/zerodha_compliance_test.go` — veraPDF `-f 4` and `-f ua2`
- `test/verify_pdfs.sh`, `test/verapdf_report.py`
- `test/structure_tree_check.py` — ParentTree ownership rules veraPDF misses
- Guides: `guides/PDF_VALIDATORS.md`

---

## 3. Target standards (engine must claim)

| Standard | Base format | Claim mechanism |
|----------|-------------|-----------------|
| **PDF 2.0** | Header `%PDF-2.0` + binary comment line | Always for A-4/UA-2 mode |
| **PDF/A-4** | ISO 19005-4 (2020) | XMP `pdfaid:part=4`, `pdfaid:rev=2020`; optional F/E later |
| **PDF/UA-2** | ISO 14289-2:2024 | XMP `pdfuaid:part=2`, `pdfuaid:rev=2024` + full tagging |

**Default engine profile for compliant mode:** PDF 2.0 + PDF/A-4 + PDF/UA-2 together (same as gopdfsuit Zerodha compliant path).

Optional later levels (already partially modeled in gopdfsuit `PDFAConfig.Conformance`): `1b/1a`, `2b/2a/2u`, `3b/3a/3u`, `4`, `4f`, `4e`. **v1 = part 4 only.**

---

## 4. Engine package layout (from-scratch proposal)

Layout is conceptual; implement under `gocorepdfengine` as deep modules with small public surfaces.

```
gocorepdfengine/
  plans/                          # this document lives here
  engine/
    doc/                          # Document builder, object ID allocator, buffer
    write/                        # Low-level PDF syntax: dict, stream, xref, trailer
    page/                         # Page tree, content streams, resources
    content/                      # Operators: text, path, color, images, BDC/EMC
    layout/                       # Tables, flow, margins (product of drawing, not compliance)
    font/                         # TTF load, subset, metrics, Liberation PDF/A map
    image/                        # Decode, XObject, ICC-based color for A-4
    color/                        # ICC profiles (sRGB, Gray), resource DefaultRGB/Gray
    meta/                         # XMP packet, catalog metadata refs
    pdfa/                         # OutputIntent, A-4 rules, trailer Info omission
    structure/                    # Structure tree, MCID, ParentTree, Namespace
    outline/                      # Optional bookmarks
    form/                         # Optional AcroForm (later)
    security/                     # Optional encrypt/sign (later; often OFF for pure A-4)
  compliance/
    verapdf/                      # Test harness placeholders (section 12)
  fixtures/                       # Golden PDFs / minimal templates for validation
```

### 4.1 Module seams (what each owns)

| Module | Owns | Must not own |
|--------|------|--------------|
| `doc` | Object IDs, write order, final assembly | Layout policy |
| `write` | Byte encoding of PDF objects | Semantics of compliance |
| `page` | `/Type /Page`, resources, annots list | Structure tree objects |
| `content` | Stream operators only | Catalog keys |
| `font` | Font programs + font dicts | Page layout |
| `color` + `meta` + `pdfa` | Archival claim objects | Drawing |
| `structure` | Tagged PDF claim | Color management |

---

## 5. Generation pipeline (layout of the engine runtime)

Mirror of `GenerateTemplatePDFBorrowed`, stripped to engine stages.

### Phase 0 — Mode resolution

Inputs:

- `ModePDF20` (always on for this plan)
- `ModePDFA4` (`pdfaCompliant`)
- `ModePDFUA2` (`taggedPDF` **or** implied by PDF/A compliant path)
- `ModeArlingtonFonts` (full standard-font metrics when not embedding Liberation)
- `ModeEmbedFonts` (default true under A-4)

Rules from current engine:

- `taggedPDF := TaggedPDF || PDFACompliant`
- PDF/A requires embedded fonts (Liberation substitution for Helvetica/Times/Courier families)
- PDF/UA requires structure tree + marked content even if product layout is “just tables”

### Phase 1 — Session setup

1. Buffer pool / capacity estimate (optional perf; not compliance).
2. Clone or create **per-generation font registry** (usage tracking isolation).
3. If A-4: ensure Liberation fonts available; register aliases for used standard faces.
4. Create page manager + structure manager (`Enabled` only if tagged).
5. Pre-reserve structure element capacity if tagged (HFT-scale tables).

### Phase 2 — Asset preparation

1. Decode images; dedupe identical payloads → single XObject.
2. Assign custom font object IDs **before** content generation (content refs need stable names).
3. (Optional) load custom TTF/OTF.

### Phase 3 — Content generation (logical → operators)

1. Walk document model (title, tables, elements, images, links).
2. For each page: write content stream into page buffer.
3. If tagged: emit structure hierarchy + marked content pairs (section 8).
4. Collect used characters for subsetting; collect used standard font names.
5. After all content (+ signature appearance if any): **generate font subsets**.

### Phase 4 — Object ID reservation (before Catalog)

Reserve IDs that Catalog must reference:

| Object | When |
|--------|------|
| Metadata (XMP stream) | Always for UA-2; always in current engine even non-A |
| StructTreeRoot | Tagged only |
| ICC sRGB profile | A-4 only |
| OutputIntent | A-4 only |
| ICC Gray profile | A-4 only (DeviceGray default) |

Also allocate: Pages kids, content streams, fonts, images, AcroForm, outlines, encrypt, as needed.

### Phase 5 — Emit document objects (order matters for xref)

Typical order in gopdfsuit:

1. Header: `%PDF-2.0` + binary comment (`%` + high bytes)
2. Catalog (obj 1)
3. Pages tree (obj 2)
4. Page objects + resource dicts
5. Content streams (`/Filter /FlateDecode` when compressed)
6. Font objects (+ descriptors, widths, FontFile2, ToUnicode, CIDToGIDMap)
7. Image XObjects (ColorSpace rewritten under A-4)
8. Extra objects (widgets, signatures, forms)
9. XMP metadata stream
10. ICC + OutputIntent (+ Gray ICC)
11. Structure tree: Namespace → StructTreeRoot → ParentTree → StructElem objects
12. Info dict **only if not PDF/A-4**
13. Encrypt dict if any
14. `xref` + `trailer` + `startxref` + `%%EOF`
15. Signature byte-range patch if signing (post-pass)

### Phase 6 — Return bytes

Borrowed buffer release API optional; compliance cares about final file bytes only.

---

## 6. PDF file skeleton — required tags & dictionaries

### 6.1 Header

- `%PDF-2.0`
- Second line: binary comment with bytes ≥ 128 (PDF conventional `%âãÏÓ` style)

### 6.2 Catalog (`/Type /Catalog`)

Always (compliant profile):

| Key | Value / notes |
|-----|----------------|
| `/Type` | `/Catalog` |
| `/Pages` | indirect ref to pages root |
| `/Lang` | e.g. `(en-US)` — PDF/UA language requirement |
| `/ViewerPreferences` | `<< /DisplayDocTitle true >>` — PDF/UA |
| `/Metadata` | indirect ref to XMP stream — **required for UA-2** (and A-4) |

When tagged (PDF/UA-2):

| Key | Value |
|-----|--------|
| `/MarkInfo` | `<< /Marked true >>` |
| `/StructTreeRoot` | indirect ref |

When PDF/A-4:

| Key | Value |
|-----|--------|
| `/OutputIntents` | array of one OutputIntent ref |

Optional product keys (engine should support hooks):

| Key | When |
|-----|------|
| `/Outlines` | bookmarks present |
| `/PageMode` | e.g. `/UseOutlines` |
| `/Names` | named destinations |
| `/AcroForm` | widgets / signatures |

### 6.3 Pages tree

| Key | Value |
|-----|--------|
| `/Type` | `/Pages` |
| `/Kids` | array of page refs |
| `/Count` | page count |

### 6.4 Page object (`/Type /Page`)

| Key | Value / notes |
|-----|----------------|
| `/Type` | `/Page` |
| `/Parent` | pages root |
| `/MediaBox` | `[0 0 width height]` |
| `/Contents` | content stream ref |
| `/Resources` | Font, XObject, ColorSpace (A-4 defaults) |
| `/Annots` | optional array |
| `/StructParents` | integer key into ParentTree — **only if page has MCIDs** |
| `/Tabs` | `/S` — **PDF/UA-2** when page has annotations (structure order) |

#### Page resources under PDF/A-4

| Resource | Form |
|----------|------|
| `/ColorSpace /DefaultRGB` | `[/ICCBased <icc-srgb> 0 R]` |
| `/ColorSpace /DefaultGray` | `[/ICCBased <icc-gray> 0 R]` |
| `/Font` | map of name → font dict |
| `/XObject` | images / form XObjects |

### 6.5 Content streams

| Key | Value |
|-----|--------|
| `/Length` | stream length |
| `/Filter` | `/FlateDecode` when compressed |

Marked content (PDF/UA) operators in stream body (not dictionary keys):

- `/TagName << /MCID n >> BDC`
- optional properties: `/Alt (...)`
- matching `EMC`
- Pagination chrome as artifact: `/Artifact << /Attached [/Top] /Type /Pagination >> BDC` … `EMC`

### 6.6 Trailer

| Key | Value / notes |
|-----|----------------|
| `/Size` | max object id + 1 |
| `/Root` | catalog |
| `/ID` | array of two byte strings (file identifiers) |
| `/Encrypt` | only if encryption enabled |
| `/Info` | **MUST be omitted under PDF/A-4** (unless PieceInfo strategy — engine currently omits entirely) |

Do **not** emit Info dictionary object under A-4; put all descriptive metadata in XMP.

### 6.7 XRef

- Classic `xref` subsections (gopdfsuit) or later PDF 2.0 cross-ref streams — v1 can keep classic xref for parity with validators already used.

---

## 7. PDF/A-4 requirements (engine checklist)

### 7.1 File-level rules (implemented / required)

| Rule | Engine behavior |
|------|-----------------|
| PDF 2.0 | Header 2.0 |
| Embedded fonts | No reliance on bare standard fonts without embedding in A-4 mode |
| XMP identification | `pdfaid:part` = 4, `pdfaid:rev` = 2020 |
| Output intent | At least one `/OutputIntent` with dest ICC profile |
| Device color | Map DeviceRGB/DeviceGray via Default* ICCBased resources; images use ICCBased |
| No trailer Info | Skip `/Info` |
| Metadata stream | Catalog `/Metadata` → `/Type /Metadata /Subtype /XML` |

### 7.2 XMP metadata packet (required fields)

Packet framing:

- `<?xpacket begin="\xEF\xBB\xBF" id="W5M0MpCehiHzreSzNTczkc9d"?>`
- `<?xpacket end="w"?>` with padding whitespace (writable packet)

Namespaces / properties used in gopdfsuit:

| Namespace | Properties |
|-----------|------------|
| `pdfaid` (`http://www.aiim.org/pdfa/ns/id/`) | `part` = 4; `rev` = 2020 (part 4); older parts may use `conformance` B/A/U |
| `pdfuaid` (`http://www.aiim.org/pdfua/ns/id/`) | `part` = 2; `rev` = 2024 |
| `pdfaExtension` schemas | Extension schema registration for `pdfuaid` properties (part, rev) for A-aware processors |
| `xmp` | `CreateDate`, `ModifyDate`, `MetadataDate`, `CreatorTool` |
| `dc` | `format` = `application/pdf`; optional `title`, `creator`, `description`, `subject` |
| `pdf` | `Producer` |
| `xmpMM` | `DocumentID`, `InstanceID` (uuid:…) |

Metadata stream object tags:

```
/Type /Metadata
/Subtype /XML
/Length …
```

### 7.3 OutputIntent object

Dictionary keys:

| Key | Value |
|-----|--------|
| `/Type` | `/OutputIntent` |
| `/S` | `/GTS_PDFA1` (still used for A-4 in this codebase) |
| `/OutputConditionIdentifier` | e.g. `sRGB IEC61966-2.1` |
| `/RegistryName` | e.g. `http://www.color.org` |
| `/Info` | human-readable sRGB label |
| `/DestOutputProfile` | ref to ICC stream |

### 7.4 ICC profile stream objects

sRGB profile stream keys:

| Key | Value |
|-----|--------|
| `/N` | `3` |
| `/Alternate` | `/DeviceRGB` |
| `/Filter` | `/FlateDecode` |
| `/Length` | … |

Gray profile:

| Key | Value |
|-----|--------|
| `/N` | `1` |
| `/Alternate` | `/DeviceGray` |
| `/Filter` | `/FlateDecode` |

Profile content: valid ICC v2.1 monitor profiles (sRGB TRC + XYZ tags; gray kTRC). Pre-compress once at process init for performance.

### 7.5 Fonts under PDF/A-4

Liberation mapping (standard name → file face):

| Standard | Liberation |
|----------|------------|
| Helvetica / Bold / Oblique / BoldOblique | LiberationSans-* |
| Times-Roman / Bold / Italic / BoldItalic | LiberationSerif-* |
| Courier / Bold / Oblique / BoldOblique | LiberationMono-* |

Embedded CID font chain tags (custom/Liberation subset path):

| Object | Key tags |
|--------|----------|
| Type0 font | `/Type /Font`, `/Subtype /Type0`, `/BaseFont`, `/Encoding /Identity-H`, `/DescendantFonts`, `/ToUnicode` |
| CIDFont | `/Subtype /CIDFontType2`, `/CIDSystemInfo` (`/Registry`, `/Ordering`, `/Supplement`), `/FontDescriptor`, `/DW`, `/W`, `/CIDToGIDMap` |
| FontDescriptor | `/FontName`, `/Flags`, `/FontBBox`, `/ItalicAngle`, `/Ascent`, `/Descent`, `/CapHeight`, `/StemV`, `/XHeight`, `/FontFile2` |
| Font program stream | `/FontFile2` stream (subset TTF), typically FlateDecode |

**Arlington / non-A path (reference only):** Type1 standard fonts with `/FirstChar`, `/LastChar`, `/Widths`, full `/FontDescriptor` metrics — not sufficient alone for A-4 embedding.

### 7.6 Images under PDF/A-4

Image XObject keys:

| Key | Value |
|-----|--------|
| `/Type` | `/XObject` |
| `/Subtype` | `/Image` |
| `/Width`, `/Height` | … |
| `/ColorSpace` | under A-4: `[/ICCBased <id> 0 R]` not bare `/DeviceRGB` |
| `/BitsPerComponent` | … |
| `/Filter` | `/DCTDecode` (JPEG) or `/FlateDecode` (PNG raw) |
| `/Length` | … |

Figures in structure tree should use `/Figure` with `/Alt` when meaningful.

### 7.7 Explicit non-goals for A-4 v1 (document and defer)

- PDF/A-4e (engineering 3D) / 4f (file attachments) special rules
- Encryption in A-4 mode (usually disallowed / restricted — keep encrypt path for non-A only until specified)
- External streams, JS, non-embedded fonts

---

## 8. PDF/UA-2 requirements (engine checklist)

### 8.1 Catalog / page (see §6)

Reiterate critical UA-2 bits:

- `/Lang`
- `/MarkInfo << /Marked true >>`
- `/StructTreeRoot`
- `/ViewerPreferences << /DisplayDocTitle true >>`
- XMP with `pdfuaid`
- Page `/StructParents` when MCIDs exist
- Page `/Tabs /S` when annotations exist

### 8.2 Namespace object (PDF 2.0 structure namespaces)

| Key | Value |
|-----|--------|
| `/Type` | `/Namespace` |
| `/NS` | `(http://iso.org/pdf2/ssn)` |

Referenced from StructTreeRoot `/Namespaces` array and from Document StructElem `/NS`.

### 8.3 StructTreeRoot

| Key | Value |
|-----|--------|
| `/Type` | `/StructTreeRoot` |
| `/K` | Document struct element ref |
| `/ParentTree` | number tree ref |
| `/Namespaces` | `[ <namespace> 0 R ]` |

### 8.4 Structure element (`/Type /StructElem`)

Common keys:

| Key | Meaning |
|-----|---------|
| `/S` | structure type name (see §8.5) |
| `/P` | parent struct elem (or StructTreeRoot for Document) |
| `/K` | kids: MCIDs (integers), child StructElem refs, and/or OBJR dicts |
| `/Pg` | page where content lives (required for leaf content elements) |
| `/T` | title (e.g. bookmark section title) |
| `/Alt` | alternate text (figures, etc.) |
| `/NS` | namespace ref (Document) |
| `/Lang` | optional per-element language (supported in model) |

**Link element special kid (OBJR):**

```
/K [ << /Type /OBJR /Obj <annot> 0 R /Pg <page> 0 R >> ]
```

### 8.5 Standard structure types used in gopdfsuit

| `/S` value | Role |
|------------|------|
| `/Document` | Required top-level (under StructTreeRoot) |
| `/Part` | Grouping |
| `/Sect` | Section; also bookmark targets |
| `/Div` | Generic block |
| `/H1` … `/H6` | Headings |
| `/P` | Paragraph |
| `/L`, `/LI`, `/Lbl`, `/LBody` | Lists |
| `/Table`, `/TR`, `/TH`, `/TD` | Tables (dominant Zerodha path) |
| `/Figure`, `/Caption` | Images |
| `/Form` | Form region |
| `/Link` | Link annotation wrapper (UA-2) |
| `/Reference` | Reference (declared; sparingly used) |

Hierarchy pattern for tables (critical for validators stricter than veraPDF):

```
Document
  └── Table
        └── TR
              └── TD | TH   ← each owns its MCID; ParentTree[page][mcid] must point to TD/TH, not TR
```

Title text often as `/H1` with `/T` title property.

### 8.6 Marked content in streams

Emit pattern:

```
/<S> << /MCID <n> >> BDC
… painting operators …
EMC
```

Optional:

```
/<S> << /MCID <n> /Alt (…) >> BDC
```

MCID allocation:

- Per-page counter starting at 0
- ParentTree[pageIndex][i] = struct element that **owns** MCID i (must be the leaf StructElem with that MCID — typically TD/TH)
- Page dict `/StructParents` = pageIndex key into ParentTree number tree

### 8.7 ParentTree (number tree)

Object form:

```
<< /Nums [
  <pageKey> [ <structElemRef> … ]   % index = MCID
  …
  <annotStructParentKey> <linkStructElemRef>
] >>
```

PDF/UA-2 annotation mapping:

- Each annot gets unique `StructParent` integer
- ParentTree maps that key → Link StructElem
- Link StructElem `/K` contains `/OBJR` to annot

### 8.8 Artifacts vs real content

Non-semantic chrome (headers as decoration, pure borders if not in table structure) should be marked `/Artifact` so AT does not read them as content. gopdfsuit uses pagination artifacts for some header chrome while table cells remain real structure.

### 8.9 Bookmarks / destinations (UA-2 note)

When outlines exist, structure destinations (`/SD`) and section structure elements improve UA-2 navigation compliance. Engine should plan `/Sect` creation for bookmark targets (`CreateBookmarkSect` pattern).

---

## 9. Document Info vs XMP (summary)

| Item | Non-A PDF 2.0 | PDF/A-4 |
|------|---------------|---------|
| Trailer `/Info` | Allowed (`/CreationDate`, `/ModDate`, optional `/Title`) | **Omit** |
| XMP dates | Recommended | Required for claim packet |
| Document title | Info and/or XMP | XMP `dc:title` + ViewerPreferences DisplayDocTitle |
| Producer/Creator | Often Info | XMP `pdf:Producer`, `xmp:CreatorTool` |

PDF date string form when needed: `D:YYYYMMDDHHmmSSOHH'mm'`  
XMP date form: `YYYY-MM-DDTHH:mm:ssZ` (or offset).

---

## 10. Object graph (mental model)

```
Catalog
├── Pages → Page* → Contents, Resources, StructParents?, Tabs?, Annots?
├── Metadata (XMP)
├── OutputIntents → OutputIntent → DestOutputProfile (ICC sRGB)
├── MarkInfo
├── StructTreeRoot
│     ├── Namespaces → Namespace (pdf2 ssn)
│     ├── ParentTree (Nums)
│     └── K → Document StructElem
│           └── … Table/TR/TD/H1/Figure/Link …
└── (optional) Outlines, Names, AcroForm

Resources.ColorSpace.DefaultRGB → ICC sRGB
Resources.ColorSpace.DefaultGray → ICC Gray
Resources.Font.* → Type0 → CIDFont → FontDescriptor → FontFile2
Resources.XObject.* → Image (ICCBased ColorSpace under A-4)
```

---

## 11. Implementation plan (phased, no code)

### Phase A — Core PDF 2.0 writer

**Deliverable:** Minimal untagged PDF: Catalog, Pages, one Page, content stream, one font, xref, trailer `/ID`.

Acceptance:

- Opens in viewers
- Header is 2.0
- Valid xref offsets

### Phase B — Layout primitives (untagged)

**Deliverable:** Text, tables, borders, multi-page, images (DeviceRGB ok).

Acceptance:

- Visual parity fixtures vs simple gopdfsuit templates (manual)

### Phase C — Font embedding pipeline

**Deliverable:** TTF load, subset, Type0/CIDFontType2 emit, ToUnicode, CIDToGIDMap, Liberation manager.

Acceptance:

- No missing glyphs for used set
- Widths consistent with subset (hyphen width sanity, etc.)

### Phase D — PDF/A-4 objects

**Deliverable:** XMP (pdfaid), OutputIntent, ICC sRGB + Gray, DefaultRGB/Gray, image ColorSpace rewrite, no trailer Info, Liberation-only fonts in A-4 mode.

Acceptance:

- veraPDF flavour `4` PASS on fixture set (section 12)

### Phase E — PDF/UA-2 tagging

**Deliverable:** StructureManager, BDC/EMC, Document/Table/TR/TD/H1, ParentTree ownership rules, Namespace, MarkInfo, Lang, ViewerPreferences, Link OBJR, Tabs /S.

Acceptance:

- veraPDF flavour `ua2` PASS
- Custom structure-tree checker: ParentTree[MCID] → TD/TH; TR `/Pg` consistent with children on multi-page tables

### Phase F — Performance & pooling (optional parity)

Buffer pools, structure arenas for large tables, parallel content compression — only after D+E green.

### Phase G — Optional product features

Signatures, encryption (non-A), AcroForm `/V`/`/AS` widgets, XFDF, merge — separate plans.

---

## 12. Testing plan — veraPDF placeholders

> **Placeholder section:** flesh out once `engine` can emit files. Paths assume future `gocorepdfengine` layout.

### 12.1 Tooling

| Tool | Role | Command shape |
|------|------|----------------|
| **veraPDF** | Primary ISO gate PDF/A-4 + PDF/UA-2 | `verapdf -f 4 file.pdf` · `verapdf -f ua2 file.pdf` |
| **structure_tree_check** (port or reuse) | ParentTree leaf ownership | `python3 tools/structure_tree_check.py file.pdf` |
| **avalpdf** (optional) | Heuristic accessibility | warnings only unless strict |

Install notes (from gopdfsuit experience):

- Java 11+ for veraPDF CLI
- Prefer project-local `verapdf/` binary
- Env override: `VERAPDF_BIN`

### 12.2 Fixture matrix (to implement)

| Fixture ID | Description | Expect A-4 | Expect UA-2 |
|------------|-------------|------------|-------------|
| `minimal-text` | Single page, one paragraph, Liberation Sans | PASS | PASS |
| `table-simple` | 3×3 table, TH + TD | PASS | PASS |
| `table-multipage` | Table spanning pages (ParentTree + `/Pg` stress) | PASS | PASS |
| `heading-title` | H1 + body | PASS | PASS |
| `figure-alt` | Image with `/Figure` + `/Alt` | PASS | PASS |
| `link-annot` | URI link + Link StructElem + `/Tabs /S` | PASS | PASS |
| `zerodha-retail-like` | Dense single-page statement style | PASS | PASS |
| `zerodha-hft-like` | Large table MCID volume | PASS | PASS |

### 12.3 Test harness placeholders

```
compliance/verapdf/
  README.md                 # install + how to run
  run_verapdf.sh            # PLACEHOLDER: wrap VERAPDF_BIN, flavours 4 + ua2
  report.py                 # PLACEHOLDER: parse veraPDF XML/JSON, exit non-zero on fail
  fixtures/                 # generated or committed PDFs under test
  expected/                 # optional: store last good veraPDF reports

# Go test placeholders (names only):
# TestCompliance_PDFA4_Minimal
# TestCompliance_PDFUA2_Minimal
# TestCompliance_PDFA4_Table
# TestCompliance_PDFUA2_Table
# TestCompliance_Matrix_AllFixtures
# TestCompliance_StructureTree_ParentTreeOwnership
```

### 12.4 Suggested Go test skeleton (behavior only)

For each fixture PDF:

1. Generate with engine (`ModePDFA4` + `ModePDFUA2`).
2. Write to temp path.
3. Skip if `VERAPDF_BIN` missing (document in CI that install is required for merge).
4. Run veraPDF flavour `4`; require exit success / report status PASSED.
5. Run veraPDF flavour `ua2`; require PASSED.
6. Run structure-tree checker; require zero hard errors.

### 12.5 CI gate (later)

| Job | Command (placeholder) |
|-----|------------------------|
| unit | `go test ./engine/...` |
| compliance | `go test ./compliance/... -count=1` |
| verify script | `./compliance/verapdf/run_verapdf.sh fixtures/*.pdf` |

### 12.6 Failure triage guide (from gopdfsuit lessons)

| Symptom | Likely cause |
|---------|----------------|
| veraPDF A-4 fails on metadata | Missing `pdfaid:part/rev`, bad XMP packet, Info present |
| A-4 fails on fonts | Standard font not embedded; subset width mismatch |
| A-4 fails on color | Image still DeviceRGB; missing OutputIntent/ICC |
| UA-2 fails on structure | Missing StructTreeRoot / MarkInfo / Lang |
| UA-2 fails on ParentTree | MCID parent is TR instead of TD; wrong Nums order |
| PAC/Adobe fail while veraPDF passes | Same ParentTree ownership / multi-page `/Pg` issues |
| Large table OOM/slow | Structure arena / batch TD emit not implemented yet |

---

## 13. Config surface for the new engine (conceptual)

| Flag | Effect |
|------|--------|
| `PDFVersion` | Fixed 2.0 for this plan |
| `PDFA` | Enable A-4 object set + font embedding policy |
| `PDFA.Conformance` | Default `4`; reserve `4f`/`4e` |
| `Tagged` / `PDFUA` | Structure tree + marked content |
| `Lang` | Catalog `/Lang` default `en-US` |
| `Title`, `Author`, `Subject`, `Keywords`, `Creator` | XMP fields |
| `EmbedFonts` | Forced true when PDFA |
| `ArlingtonCompatible` | Full metrics for non-embedded standard fonts (non-A) |

When `PDFA` is true, engine implies `Tagged` true (match gopdfsuit generator logic).

---

## 14. Traceability: gopdfsuit → gocorepdfengine

| gopdfsuit concept | New module |
|-------------------|------------|
| `GenerateTemplatePDFBorrowed` | `engine/doc` pipeline orchestrator |
| `PageManager` | `engine/page` |
| `StructureManager` | `engine/structure` |
| `PDFAHandler` + `pdfa.go` | `engine/pdfa` + `engine/meta` + `engine/color` |
| `font.PDFAFontManager` | `engine/font` (Liberation) |
| `draw.go` marked content calls | `engine/content` + `engine/layout` |
| veraPDF tests | `compliance/verapdf` |

---

## 15. Out of scope for this plan document

- HTTP API, Gin handlers, auth
- Frontend editor
- Python/CGO bindings
- Merge/split/redact product features
- Performance benchmark targets (separate plan after green compliance)
- Full Matterhorn Protocol manual PAC checklist (optional pre-release only)

---

## 16. Recommended next actions

1. Scaffold `gocorepdfengine/engine` packages with empty interfaces matching §4.
2. Implement Phase A (minimal PDF 2.0 writer) with unit tests on object offsets.
3. Wire `compliance/verapdf/run_verapdf.sh` placeholder even before fixtures exist (skip logic).
4. Implement Phase C → D until first `minimal-text` PDF passes veraPDF `-f 4`.
5. Implement Phase E until same PDF passes `-f ua2`.
6. Expand fixture matrix; add structure_tree_check before large-table work.

---

## 17. Appendix — quick tag cheat sheet

### Catalog essentials

`/Type /Catalog` · `/Pages` · `/Lang` · `/Metadata` · `/MarkInfo << /Marked true >>` · `/StructTreeRoot` · `/ViewerPreferences << /DisplayDocTitle true >>` · `/OutputIntents [ … ]`

### Structure essentials

`/Type /StructTreeRoot` · `/ParentTree` · `/Namespaces` · `/K`  
`/Type /StructElem` · `/S` · `/P` · `/K` · `/Pg` · `/Alt` · `/T` · `/NS`  
`/Type /Namespace` · `/NS (http://iso.org/pdf2/ssn)`  
OBJR: `/Type /OBJR` · `/Obj` · `/Pg`  
ParentTree: `/Nums [ key arrayOrRef … ]`

### PDF/A essentials

Metadata: `/Type /Metadata` · `/Subtype /XML`  
OutputIntent: `/Type /OutputIntent` · `/S /GTS_PDFA1` · `/DestOutputProfile`  
ICC stream: `/N` · `/Alternate` · `/Filter /FlateDecode`  
Page: `/DefaultRGB [/ICCBased …]` · `/DefaultGray [/ICCBased …]`  
Trailer: **no** `/Info`

### XMP claim essentials

`pdfaid:part` 4 · `pdfaid:rev` 2020 · `pdfuaid:part` 2 · `pdfuaid:rev` 2024

### Stream tagging essentials

`/TD << /MCID n >> BDC` … `EMC`  
`/Artifact << /Type /Pagination … >> BDC` … `EMC`

### Page UA extras

`/StructParents n` · `/Tabs /S`

---

*End of plan. veraPDF harness details intentionally left as placeholders in §12 until the engine emits first compliant fixtures.*
