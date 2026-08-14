# gowkhtmltopdf — PDF/A-4 + PDF/UA-2 Compliance Plan (Canonical Ledger)

> **Issue:** [#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33)
> **Parent epic:** [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)
> **Depends on:** [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32) / [pdf-2.0-plan/](pdf-2.0-plan/) (PDF 2.0 version path)
> **Reuses:** [pdf-1.7-compliance-plan/](pdf-1.7-compliance-plan/) (A-3a + UA-1 machinery: XMP, ICC, OutputIntent, structure tree)
> **Status:** draft — not started
> **Constraint:** pure Go, no CGO, no new direct modules. veraPDF (`verapdf/`) is already in-tree as an external validator, not a Go dependency.
> **Workflow:** [`skills/phase-wise-checklist/SKILLS.md`](../../skills/phase-wise-checklist/SKILLS.md)

---

## 1. Purpose

The highest standards pair that sits on PDF 2.0 (ISO 32000-2):

| | Archival | Accessibility |
|--|----------|----------------|
| 1.7 (done) | PDF/A-3a (ISO 19005-3, level A) | PDF/UA-1 (ISO 14289-1:2014) |
| **2.0 (this plan)** | **PDF/A-4** (ISO 19005-4:2020) | **PDF/UA-2** (ISO 14289-2:2024) |

Default compliant 2.0 mode = **both claims on one file**. Gate: veraPDF `-f 4` **and** `-f ua2`.

This is **not a new engine**. The 1.7 compliance plan already built the machinery:
XMP builder (`internal/pdf/metadata.go`), ICC profiles (`icc.go`), OutputIntent
(`outputintent.go`), structure tree with MCIDs and ParentTree (`structure.go`),
embedded Liberation/DejaVu subset fonts (`subset.go`, `fonttype0.go`), and a
layout→structure bridge (`internal/layout`). This plan extends that machinery
with the 2.0 deltas: `pdfaid:part=4`, `pdfuaid:part=2`, `/Namespace` objects,
trailer Info omission, and the A-4 color rules — then proves them with veraPDF.

**Default output stays unclaimed PDF 1.4.** Compliance is opt-in and implies PDF 2.0.

---

## 2. What already exists (do not rebuild)

| Capability | Where it lives | Delta for this plan |
|------------|----------------|---------------------|
| `WriterPolicy` + `ConformanceProfile` | `internal/pdf/policy.go` | Accept A-4 / UA-2 only on `PDF20`; new errors otherwise |
| Profile validation matrix | `policy.go` `Validate()`, `CanonicalProfile()` | A-4/UA-2 strings stop returning `ErrConformanceProfilesUnsupported` |
| XMP metadata stream | `internal/pdf/metadata.go` (`buildXMPMetadata`) | `pdfaid:part=4, rev=2020`; `pdfuaid:part=2, rev=2024`; `pdfaExtension` schema for `pdfuaid` |
| ICC profiles | `internal/pdf/icc.go` (`sRGBICCProfile`) | Add Gray ICC (`/N 1`); reuse sRGB |
| OutputIntent | `internal/pdf/outputintent.go` | Same `/S /GTS_PDFA1` form; A-4 allows multiple intents (one is enough for v1) |
| Structure tree | `internal/pdf/structure.go` (`StructTreeRoot`, `StructElem`, `AllocMCID`, `buildParentTree`) | Add `/Namespace` objects, StructTreeRoot `/Namespaces`, Document `/NS` |
| Marked content | `internal/layout` (BDC/EMC, `/Alt`, artifacts) | UA-2 tags reuse UA-1 tags; namespaced role map only if needed |
| Embedded fonts | `internal/pdf` (`fonts.go`, `subset.go`, `fonttype0.go`, `assets/`) | Liberation/DejaVu already embedded; no new font work |
| Images | `internal/pdf/images.go` | Rewrite `DeviceRGB` → `[/ICCBased <srgb>]` under A-4 |
| Header / version | `internal/pdf/pdf.go` + `policy.go` `HeaderVersion()` | `%PDF-2.0` comes from #32; this plan only adds claims |
| Trailer / Info | `internal/pdf/pdf.go` | Omit `/Info` **only** under A-4 (1.4/1.7 keep it) |
| veraPDF harness | `verapdf/` in repo | Add `-f 4` and `-f ua2` runs; skip if binary missing |
| Semantic parser | `internal/pdf/semantic.go` | Must see `pdfaid`/`pdfuaid`, OutputIntent, Namespace in 2.0 output |

---

## 3. Target standards

| Standard | Base format | Claim mechanism |
|----------|-------------|-----------------|
| **PDF 2.0** | Header `%PDF-2.0` + binary comment line | Required base for A-4/UA-2 (from #32) |
| **PDF/A-4** | ISO 19005-4:2020 | XMP `pdfaid:part=4`, `pdfaid:rev=2020` |
| **PDF/UA-2** | ISO 14289-2:2024 | XMP `pdfuaid:part=2`, `pdfuaid:rev=2024` + full tagging + `/Namespace` |

Optional later levels (documented, not shipped): A-4e (engineering 3D), A-4f (file attachments), multiple OutputIntents. **v1 = part 4 + UA-2 only.**

---

## 4. Dependencies and relationship to sibling plans

```text
#31 PDF 1.7            → 1.7 compliance plan: A-3a + UA-1 (completed, reuses machinery)
#32 PDF 2.0            → this plan's base: PDF20 emit, UTF-8 strings, non-claiming XMP
        │
        ▼
#33 PDF/A-4 + UA-2     this plan: claims, OutputIntent, Namespace, tagging on 2.0
```

- #32 must land first (or at least the `PDF20` header + XMP path). A-4/UA-2 on a 1.4 or 1.7 file is an error, not a best-effort claim.
- The 1.7 compliance plan's structure tree, ParentTree, BDC/EMC, and layout bridge are the starting point. UA-2 adds `/Namespace` and namespaced structure; the tree shape (Document → Table → TR → TD/TH) is identical.
- Do **not** create a second writer or an `engine/` tree. Extend `internal/pdf` and `internal/layout`.

---

## 5. Profile policy (WriterPolicy extension)

Mirror the 1.7 pattern in `internal/pdf/policy.go`:

```go
const (
    ProfilePDFA4        = "PDF/A-4"        // ISO 19005-4
    ProfilePDFUA2       = "PDF/UA-2"       // ISO 14289-2:2024
    ProfilePDFA4PDFUA2  = "PDF/A-4+PDF/UA-2"
)
```

Rules:

| Policy | 1.4 default | 1.7 | 2.0 (#32) |
|--------|-------------|-----|------------|
| `ConformanceProfile` empty | valid, unclaimed | valid, unclaimed | valid, unclaimed |
| A-3a / UA-1 / dual-1.7 | error (requires 1.7) | valid (existing) | error (`ErrConformanceRequiresPDF17`-style) |
| A-4 / UA-2 / dual-2.0 | error | error | **valid (new)** |
| Unknown profile | error | error | error |

- `Validate()`: A-4/UA-2 requires `Version == PDF20`; new error e.g. `ErrConformanceRequiresPDF20` for a 2.0 profile on a 1.x base. Remove A-4/UA-2 strings from `isUnsupportedPDF20Profile` gate → they become accepted on PDF20, still rejected on PDF17.
- `CanonicalProfile()` accepts the aliases (`a4`, `ua2`, `a4+ua2`, etc.).
- Add `IsPDFA4()`, `IsPDFUA2()` helpers mirroring `IsPDFA3()` / `IsPDFUA1()`.
- `ProfileDualA4UA2` = default when a caller asks for compliant 2.0 without a specific profile.

---

## 6. PDF file skeleton — required tags and dictionaries

### 6.1 Catalog (`/Type /Catalog`)

Compliant 2.0 profile:

| Key | Value / notes |
|-----|----------------|
| `/Type` | `/Catalog` |
| `/Pages` | indirect ref to pages root |
| `/Lang` | `(en-US)` — PDF/UA requirement |
| `/ViewerPreferences` | `<< /DisplayDocTitle true >>` — PDF/UA |
| `/Metadata` | indirect ref to XMP stream — required for A-4 and UA-2 |
| `/MarkInfo` | `<< /Marked true >>` when tagged |
| `/StructTreeRoot` | indirect ref when tagged |
| `/OutputIntents` | array of one OutputIntent ref under A-4 |

Optional existing keys stay: `/Outlines`, `/Names`, `/PageMode`.

### 6.2 Pages tree / page object

- `/Type /Pages`, `/Kids`, `/Count` — unchanged.
- Page: `/MediaBox`, `/Contents`, `/Resources`, `/Annots` — unchanged.
- Page `/StructParents` when MCIDs exist (already emitted by `structure.go`).
- Page `/Tabs /S` when annotations exist (UA-2 navigation order).
- Page resources under A-4:

| Resource | Form |
|----------|------|
| `/ColorSpace /DefaultRGB` | `[/ICCBased <icc-srgb> 0 R]` |
| `/ColorSpace /DefaultGray` | `[/ICCBased <icc-gray> 0 R]` |
| `/Font` | embedded Type0 chain (already true) |
| `/XObject` | images with ICCBased color space under A-4 |

### 6.3 Content streams

Marked content operators (already emitted for UA-1, reused for UA-2):

- `/<S> << /MCID n >> BDC` … `EMC`
- optional `/Alt (…)` in BDC properties
- pagination chrome: `/Artifact << /Attached [/Top] /Type /Pagination >> BDC` … `EMC`

### 6.4 Trailer

| Key | Value / notes |
|-----|----------------|
| `/Size`, `/Root`, `/ID` | unchanged (classic xref stays; no xref streams in this plan) |
| `/Info` | **omit under A-4** (ISO 19005-4 forbids relying on Info; all descriptive metadata in XMP). 1.4/1.7 paths keep emitting it |

### 6.5 XRef

Classic `xref` subsections — unchanged (xref streams are an explicit non-goal everywhere).

---

## 7. PDF/A-4 requirements (engine checklist)

### 7.1 File-level rules

| Rule | Behavior |
|------|----------|
| PDF 2.0 header | from #32 |
| Embedded fonts | already embedded (Liberation/DejaVu subset); never bare standard fonts under A-4 |
| XMP identification | `pdfaid:part` = 4, `pdfaid:rev` = 2020 |
| Output intent | at least one `/OutputIntent` with dest ICC profile |
| Device color | DeviceRGB/DeviceGray mapped via Default* ICCBased resources; images use ICCBased |
| No trailer Info | skip `/Info` and the Info object under A-4 |
| Metadata stream | Catalog `/Metadata` → `/Type /Metadata /Subtype /XML` |

### 7.2 XMP metadata packet

Framing (unchanged from 1.7 plan):

- `<?xpacket begin="\xEF\xBB\xBF" id="W5M0MpCehiHzreSzNTczkc9d"?>`
- `<?xpacket end="w"?>` with padding whitespace

Namespaces / properties:

| Namespace | Properties |
|-----------|------------|
| `pdfaid` (`http://www.aiim.org/pdfa/ns/id/`) | `part` = **4**; `rev` = **2020** (part 4 uses part/rev, not conformance levels) |
| `pdfuaid` (`http://www.aiim.org/pdfua/ns/id/`) | `part` = **2**; `rev` = **2024** |
| `pdfaExtension` schemas | extension schema registration for `pdfuaid` `part` + `rev` |
| `xmp` | `CreateDate`, `ModifyDate`, `MetadataDate`, `CreatorTool` |
| `dc` | `format` = `application/pdf`; optional `title`, `creator`, `description`, `subject` |
| `pdf` | `Producer` = `gowkhtmltopdf 2.0` (policy `ProducerVersion()`) |
| `xmpMM` | `DocumentID`, `InstanceID` (`uuid:…`) |

Deltas from `buildXMPMetadata()`: branch on policy — `pdfaid` part/rev for A-3a (1.7) vs A-4 (2.0); add `pdfuaid` for UA-1 vs UA-2; `pdfaExtension` schema only when a UA claim exists.

### 7.3 OutputIntent object

| Key | Value |
|-----|--------|
| `/Type` | `/OutputIntent` |
| `/S` | `/GTS_PDFA1` (current codebase convention; valid for A-4) |
| `/OutputConditionIdentifier` | e.g. `sRGB IEC61966-2.1` |
| `/RegistryName` | e.g. `http://www.color.org` |
| `/Info` | human-readable sRGB label |
| `/DestOutputProfile` | ref to ICC stream |

### 7.4 ICC profile stream objects

sRGB (exists in `icc.go`):

| Key | Value |
|-----|--------|
| `/N` | `3` |
| `/Alternate` | `/DeviceRGB` |
| `/Filter` | `/FlateDecode` |

Gray (new; needed because A-4 forbids bare DeviceGray):

| Key | Value |
|-----|--------|
| `/N` | `1` |
| `/Alternate` | `/DeviceGray` |
| `/Filter` | `/FlateDecode` |

Valid ICC v2.1 profile bytes; pre-compress once at package init (perf).

### 7.5 Fonts under A-4

- Liberation mapping already in `assets/` (Sans/Serif/Mono families) — no change.
- Embedded CID chain tags (existing): Type0 `/Encoding /Identity-H` + `/ToUnicode`; CIDFontType2 with `/CIDSystemInfo`, `/FontDescriptor`, `/DW`, `/W`, `/CIDToGIDMap`; `/FontFile2` subset stream.
- Non-embedded standard-font path must be rejected in A-4 mode (gate, not rebuild).

### 7.6 Images under A-4

| Key | Value |
|-----|--------|
| `/ColorSpace` | under A-4: `[/ICCBased <id> 0 R]`, never bare `/DeviceRGB` |
| `/Filter` | `/DCTDecode` (JPEG) or `/FlateDecode` (PNG raw) — unchanged |

Figures in the structure tree use `/Figure` with `/Alt` (already enforced for UA-1; reuse the `ErrPDFUAMissingAlt` gate).

### 7.7 Non-goals for A-4 v1

- A-4e / A-4f
- Encryption in A-4 mode (encryption stays unsupported everywhere: `ErrEncryptionUnsupported`)
- External streams, JS, non-embedded fonts

---

## 8. PDF/UA-2 requirements (engine checklist)

### 8.1 Catalog / page

- `/Lang` (configurable, default `en-US`)
- `/MarkInfo << /Marked true >>`
- `/StructTreeRoot`
- `/ViewerPreferences << /DisplayDocTitle true >>`
- XMP with `pdfuaid` claim
- Page `/StructParents` when MCIDs exist
- Page `/Tabs /S` when annotations exist

All of these exist from UA-1 except the `pdfuaid` values and the Namespace objects.

### 8.2 Namespace object (PDF 2.0 structure namespaces — the UA-2 delta)

| Key | Value |
|-----|--------|
| `/Type` | `/Namespace` |
| `/NS` | `(http://iso.org/pdf2/ssn)` |

Referenced from StructTreeRoot `/Namespaces` array and from the Document StructElem `/NS`. UA-1 explicitly did **not** use this (see `pdf-1.7-compliance-plan/SPEC-NOTES.md`); UA-2 requires it.

### 8.3 StructTreeRoot

| Key | Value |
|-----|--------|
| `/Type` | `/StructTreeRoot` |
| `/K` | Document struct element ref |
| `/ParentTree` | number tree ref |
| `/Namespaces` | `[ <namespace> 0 R ]` — new for UA-2 |

### 8.4 Structure element (`/Type /StructElem`)

Common keys (already serialized by `serializeStructElem`):

| Key | Meaning |
|-----|---------|
| `/S` | structure type name |
| `/P` | parent struct elem (or StructTreeRoot for Document) |
| `/K` | kids: MCIDs, child StructElem refs, OBJR dicts |
| `/Pg` | page where content lives (leaf content elements) |
| `/T` | title |
| `/Alt` | alternate text |
| `/NS` | namespace ref on Document — new for UA-2 |
| `/Lang` | optional per-element language |

Link element kid: `/K [ << /Type /OBJR /Obj <annot> 0 R /Pg <page> 0 R >> ]` (exists).

### 8.5 Structure types

Reuse the UA-1 set from `internal/pdf/structure.go` — the PDF 2.0 tag set is a superset:

| `/S` value | Role |
|------------|------|
| `/Document` | required top-level |
| `/H1` … `/H6`, `/P` | headings, paragraphs |
| `/Table`, `/TR`, `/TH`, `/TD` | tables (dominant HTML path) |
| `/L`, `/LI`, `/Lbl`, `/LBody` | lists |
| `/Figure` | images |
| `/Link` | link annotation wrapper |
| `/Part`, `/Sect`, `/Div` | grouping (Sect for TOC/bookmark targets) |

Defer until a real consumer exists: `/Formula` (MathML), `/Caption`, `/Form`, `/Reference`, `/Aside`.

Table hierarchy (critical for validators stricter than veraPDF):

```
Document
  └── Table
        └── TR
              └── TD | TH   ← each owns its MCID; ParentTree[page][mcid] → TD/TH, never TR
```

### 8.6 Marked content in streams

Existing emit pattern reused:

```
/<S> << /MCID n >> BDC … painting operators … EMC
/<S> << /MCID n /Alt (…) >> BDC … EMC
```

- Per-page MCID counter from 0 (`Page.AllocMCID`).
- ParentTree[pageIndex][i] = leaf StructElem owning MCID i (typically TD/TH) — `buildParentTree` already enforces this.
- Page `/StructParents` = pageIndex key into ParentTree number tree.

### 8.7 ParentTree (number tree)

`<< /Nums [ pageKey [ elemRefs… ] … annotStructParentKey linkStructElemRef ] >>` — exists; UA-2 adds no changes beyond what the 1.7 plan built.

### 8.8 Artifacts vs real content

Non-semantic chrome (page numbers, running headers) stays `/Artifact` so AT skips it — layout already does this.

### 8.9 Bookmark / section structure

When outlines exist, `/Sect` structure elements for bookmark targets improve UA-2 navigation; wire `internal/outline` targets into the structure tree (defer until fixtures demand it).

---

## 9. Document Info vs XMP (summary)

| Item | Non-A PDF 2.0 | PDF/A-4 |
|------|---------------|---------|
| Trailer `/Info` | allowed (deprecated in 32000-2, still emitted) | **omit** |
| XMP dates | recommended | required |
| Document title | Info and/or XMP | XMP `dc:title` + ViewerPreferences DisplayDocTitle |
| Producer | Info and/or XMP | XMP `pdf:Producer`, `xmp:CreatorTool` |

Date forms: PDF `D:YYYYMMDDHHmmSSOHH'mm'`; XMP `YYYY-MM-DDTHH:mm:ssZ`.

---

## 10. Object graph (mental model)

```
Catalog
├── Pages → Page* → Contents, Resources (DefaultRGB/Gray ICCBased, Font, XObject), StructParents?, Tabs?, Annots?
├── Metadata (XMP)
├── OutputIntents → OutputIntent → DestOutputProfile (ICC sRGB)
├── MarkInfo
├── StructTreeRoot
│     ├── Namespaces → Namespace (http://iso.org/pdf2/ssn)
│     ├── ParentTree (Nums)
│     └── K → Document StructElem (/NS)
│           └── … Table/TR/TD/H1/Figure/Link …
└── (optional) Outlines, Names
```

---

## 11. Phase map

Mirrors the 1.7 compliance plan's shape. Phases 2 and 3 may overlap after phase 1.

```text
1 Profile policy + standards matrix
  → 2 PDF/A-4 archival objects (XMP part 4, OutputIntent, ICC, no Info)
      → 3 Color + fonts under A-4 (ICCBased images, embed gates)
          → 4 PDF/UA-2 structure (Namespace, dual XMP, UA-2 tags)
              → 5 Layout → tagging bridge (UA-2 deltas)
                  → 6 Settings, CLI, library (opt-in on 2.0)
                      → 7 veraPDF + goldens (-f 4 and -f ua2)
                          → 8 Docs + closure
```

| Phase | Goal | Gate |
|------:|------|------|
| 1 | Profile policy: accept A-4 / UA-2 / dual on PDF20; still error on 1.4/1.7 | Unit: `policy_test.go` matrix updated |
| 2 | `pdfaid:part=4/rev=2020`, OutputIntent, sRGB+Gray ICC, omit trailer Info | Structure tests on 2.0 writer output; no veraPDF yet |
| 3 | ICCBased images, A-4 font/embed gates | Existing font/image tests + A-4 cases |
| 4 | `/Namespace`, StructTreeRoot `/Namespaces`, Document `/NS`, `pdfuaid:part=2/rev=2024` + `pdfaExtension` | Writer-level tagged 2.0 fixture |
| 5 | HTML → UA-2 structure types on 2.0 (mostly reuse of the 1.7 bridge) | Convert fixture: headings/tables/figures/links |
| 6 | Opt-in profile selection on the 2.0 base | CLI/library tests; default still unclaimed 1.4 |
| 7 | Dual fixtures + veraPDF `-f 4` and `-f ua2` (skip if binary missing) | External validator green |
| 8 | Honest docs: version ≠ conformance; A-4/UA-2 rows in matrix | `make lint` + `make test` + claim-scan |

---

## 12. Testing plan — veraPDF gates

`verapdf/` is already in the repo (v1.25+ handles flavour `4` and `ua2`).

### 12.1 Commands

| Tool | Role | Command |
|------|------|---------|
| veraPDF | ISO gate for A-4 + UA-2 | `verapdf -f 4 file.pdf` · `verapdf -f ua2 file.pdf` |
| `internal/pdf/semantic.go` | in-tree structural claims check | unit tests assert `pdfaid`/`pdfuaid`, Namespace, OutputIntent |

Skip logic: if the veraPDF binary is missing, tests skip (existing `compliance_test.go` pattern); CI documents install as required for the merge gate.

### 12.2 Fixture matrix

| Fixture ID | Description | `-f 4` | `-f ua2` |
|------------|-------------|--------|----------|
| `minimal-text` | Single page, one paragraph, embedded Liberation | PASS | PASS |
| `table-simple` | 3×3 table, TH + TD | PASS | PASS |
| `table-multipage` | Table spanning pages (ParentTree + `/Pg` stress) | PASS | PASS |
| `heading-title` | H1 + body | PASS | PASS |
| `figure-alt` | Image with `/Figure` + `/Alt` | PASS | PASS |
| `link-annot` | URI link + Link StructElem + `/Tabs /S` | PASS | PASS |

### 12.3 Go test names

```
TestCompliance_PDFA4_Minimal
TestCompliance_PDFUA2_Minimal
TestCompliance_PDFA4_Table
TestCompliance_PDFUA2_Table
TestCompliance_Matrix_AllFixtures   // -f 4 + -f ua2
TestCompliance_StructureTree_ParentTreeOwnership
```

### 12.4 Failure triage (lessons from the 1.7 plan + gopdfsuit)

| Symptom | Likely cause |
|---------|----------------|
| A-4 fails on metadata | Missing `pdfaid:part/rev`, bad xpacket framing, trailer Info present |
| A-4 fails on fonts | Unembedded standard font; subset width mismatch |
| A-4 fails on color | Image still DeviceRGB; missing OutputIntent/ICC; bare DeviceGray |
| UA-2 fails on structure | Missing StructTreeRoot / MarkInfo / Lang / Namespace |
| UA-2 fails on ParentTree | MCID parent is TR instead of TD; wrong Nums order |
| PAC/Adobe fail while veraPDF passes | ParentTree ownership / multi-page `/Pg` / missing `/Tabs /S` |

---

## 13. Config surface (settings / CLI / library)

| Surface | Key / setter | Effect |
|---------|--------------|--------|
| `internal/settings.PdfGlobal` | version (`2.0`) + conformance profile | Opt-in; default 1.4 unclaimed |
| `internal/cli/flags.go` | `--pdf-version 2.0 --pdf-profile a4-ua2` | PDF mode only |
| `api.go` `PDFRequest` | typed profile setter | `ConformanceProfile` on the 2.0 request |
| `internal/pdf.WriterPolicy` | `Version: PDF20` + `ConformanceProfile: ProfilePDFA4PDFUA2` | The only place claim objects get emitted |

- `ProfileDualA4UA2` implies tagging (match 1.7 behavior: `tagged := Tagged || PDFA`).
- A-4 implies embedded fonts (already always embedded; gate rather than feature).
- 2.0 without a profile: XMP emitted (from #32) but **no** `pdfaid`/`pdfuaid`, no OutputIntent, no Namespace.

---

## 14. Out of scope

- PDF/A-4e / 4f; multiple OutputIntents; A-4 on a 1.x base
- Encryption, signatures, AcroForm, object streams, xref streams, linearization (still error out via existing policy checks)
- MathML / `/Formula`; Matterhorn Protocol manual PAC checklist (optional pre-release)
- A second writer package, an `engine/` tree, JSON templates, Zerodha benchmarks
- Changing the 1.4 default or the 1.7 compliance behavior

---

## 15. Recommended next actions

1. Land #32 (or at least `PDF20` header + non-claiming XMP) as the base.
2. Phase 1: extend `WriterPolicy` so `PDF20 + A-4/UA-2` validates; update `policy_test.go` matrix (A-4/UA-2 on PDF17 still errors).
3. Phase 2: branch `buildXMPMetadata` on policy (part 4 / rev 2020), wire OutputIntent + Gray ICC, omit trailer Info under A-4.
4. Phase 4: add `/Namespace` emit + `pdfuaid` XMP; verify structure serialization on a tagged 2.0 fixture.
5. Phase 7: run `verapdf -f 4` and `-f ua2` on `minimal-text` first, then grow the fixture matrix.

---

## 16. Appendix — quick tag cheat sheet

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
