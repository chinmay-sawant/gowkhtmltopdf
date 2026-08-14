# 00 — PDF 1.7 Version Support (Canonical Execution Ledger)

> **Parent:** `plans/0.2.2/README.md` — epic [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)
> **Issue:** [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31)
> **Status:** completed
> **Estimated effort:** 3–5 weeks across phases 1–8
> **Constraint:** pure Go, no CGO, no new direct modules. Do not add veraPDF, Brotli, or a second PDF writer.
> **Ordering principle:** version policy and serialization first, then call-site selection, then proof, then docs.
> **Workflow:** [`skills/phase-wise-checklist/SKILLS.md`](../../../skills/phase-wise-checklist/SKILLS.md)
> **Spec notes:** [SPEC-NOTES.md](SPEC-NOTES.md) (ISO 32000-1:2008)
> **Highest 1.7 compliance:** PDF/A-3a + PDF/UA-1 — [../pdf-1.7-compliance-plan/](../pdf-1.7-compliance-plan/) (not this ledger)

---

## Overview

The current writer is a **PDF 1.4 sink**. `internal/pdf.NewDocument` always
emits `%PDF-1.4`, a classic xref, an Info dictionary with Latin-1
`pdfString`, and `Producer "gowkhtmltopdf 1.4"`. Layout, fonts, images,
outlines, and links already exist. They must keep working.

Issue #31 is a **version-policy and serialization** change, not a new
engine. Callers opt into PDF 1.7. The default stays 1.4. A header-only
swap is not support.

ISO 32000-1:2008 **is** PDF 1.7. §2.1 says a conforming file need not
use every later feature. §6 says every non-deprecated 1.4 feature stays
legal. So we do **not** rebuild layout or the subsetter. We declare
1.7 and emit the document-level items the spec tells writers they
should use.

---

## Where PDF 1.7 lands

The pipeline does not change:

```text
cmd/gowkhtmltopdf | api.go | internal/app
        │
        ▼
internal/settings.PdfGlobal          ← NEW: version selection (default 1.4)
        │
        ▼
internal/convert.Request
        │
        ▼
pdf.NewDocument / NewDocumentWithPolicy   ← NEW: WriterPolicy (this plan owns the type)
        │
        ▼
internal/layout.PaintContext         ← UNCHANGED (paint ops are version-agnostic)
        │
        ▼
pdfPipeline.Assemble                 ← SMALL: pass policy; Producer string
        │
        ▼
Document.Write / finalize / writeTo  ← VERSION-AWARE header, trailer /ID, Info+XMP, UTF-16BE
```

`internal/imageout` keeps using `internal/pdf` only for fonts, shaping,
and glyph outlines. It never writes a PDF and is out of this plan.

### Package ownership

| Concern | Primary code | Change in this plan |
|---------|--------------|---------------------|
| Version policy | `internal/pdf` (new type next to `Document`) | **This plan introduces** `PDFVersion` / `WriterPolicy`. No raw `"1.7"` checks in content, fonts, or images |
| Header / xref / trailer | `internal/pdf/pdf.go` (`writePDFHeader`, `writePDFTrailer`, `const Version`) | Header from policy; trailer `/ID` on 1.7; classic xref stays |
| Catalog / Info | `internal/pdf/pdf.go` (`catalogDict`, `infoDict`, `finalize`) | Info kept (first-class in 1.7); 1.7 Metadata stream; no catalog `/Version` unless later than the header |
| Text strings | `internal/pdf/pdf.go` (`pdfString`) | 1.4 Latin-1 fold unchanged; 1.7 text strings are PDFDocEncoding **or UTF-16BE + BOM** (ISO 32000-1 §7.9.2.2). Not UTF-8 |
| Fonts / images / content | `internal/pdf/fonttype0.go`, `images.go`, `content.go` | Gates only. No new subsetter, no new paint API |
| Semantic parse | `internal/pdf/semantic.go` | Already exposes `SemanticDoc.Version`; tests must cover 1.7 |
| Settings | `internal/settings/settings.go`, `reflect.go` | `PdfGlobal` field + dotted key (`1.4` / `1.7`) |
| CLI | `internal/cli/flags.go` (`addGlobalFlags`) | `--pdf-version` (PDF mode only). 2.0 later adds `2.0` to the same flag |
| Library | `api.go` (`PDFRequest`, `PdfGlobalOptions`) | Typed setter; default omitted = 1.4 |
| Convert | `internal/convert/convert.go`, `pdf_pipeline.go` | Construct document with policy; do not hard-code 1.4 |
| Layout / HTML / CSS / load | `internal/layout`, `internal/html`, `internal/css`, `internal/load` | **No change** |
| Docs | `README.md`, `documentation/architecture.md`, `architecture/09-pdf-writer.md`, `compatibility-matrix.md` | Publish only after fixtures prove the claim |

### What already exists (do not rebuild)

| Capability | Where it already lives |
|------------|------------------------|
| HTML → boxes → display list → pagination | `internal/layout` |
| Job orchestration, TOC, HF, copies, links | `internal/convert` + `render.Pipeline` |
| TTF parse, subset, Type0/CID, ToUnicode | `internal/pdf` (`fonts.go`, `subset.go`, `fonttype0.go`) |
| JPEG DCTDecode, PNG Flate + SMask | `internal/pdf/images.go` |
| Outlines, URI / GoTo annots | `internal/pdf/pdf.go` |
| Deterministic write, counting xref | `writeTo`, `countingWriter` |
| In-tree semantic parser | `internal/pdf/semantic.go` |

---

## Relationship to #32 and #33

```text
#31 PDF 1.7          THIS PLAN: WriterPolicy{PDF14, PDF17}, reserve PDF20
        │
        ▼
#32 PDF 2.0          sibling: add PDF20 + UTF-8 strings (ISO 32000-2)
        │
        ▼
#33 PDF/A-4 + UA-2   later: claiming XMP, OutputIntent, structure tree
```

Do **not** create a second policy type in the 2.0 plan. `PDF20` exists
here as a reserved value that `NewDocumentWithPolicy` rejects until #32
implements it.

#33 may later consume the 1.7 or 2.0 path. This plan must not emit
`pdfaid` / `pdfuaid`, OutputIntent, ICC, MarkInfo, or a structure tree.

---

## Version policy seam

The exact names can move during implementation. The invariant cannot:
version and feature decisions happen **before** objects are emitted.

```go
type PDFVersion int

const (
    PDF14 PDFVersion = iota
    PDF17
    PDF20            // reserved for #32 at the time this was written; shipped as opt-in since 0.2.2
)

type WriterPolicy struct {
    Version PDFVersion
}

func NewDocument() *Document                    // PDF 1.4 default
func NewDocumentWithPolicy(p WriterPolicy) (*Document, error)
```

Policy methods (not call-site `if version ==`) own the rules:

| Question | 1.4 (default) | 1.7 | 2.0 (sibling) |
|----------|---------------|-----|----------------|
| Header | `%PDF-1.4` | `%PDF-1.7` | `%PDF-2.0` |
| Classic xref + trailer | yes | yes (no xref stream) | yes |
| Trailer `/ID` | unchanged (absent) | yes, deterministic (ISO 32000-1 §14.4 *should*) | yes |
| Info dictionary | emitted | still first-class | still emitted (deprecated in 2.0) |
| XMP `/Metadata` | no | yes, **without** A-4 / UA-2 claims | yes, same rule |
| Document text strings | Latin-1 `pdfString` | PDFDocEncoding or **UTF-16BE + BOM** | **UTF-8** (ISO 32000-2) |
| Content-stream text (`Tj`) | WinAnsi / Identity-H | unchanged | unchanged |
| Font / image objects | as today | as today; reject combinations the writer cannot prove | as today |

`const Version = "1.4"` in `pdf.go` must stop being a process-wide
header. Producer metadata should follow the **document** version
(`gowkhtmltopdf 1.4` vs `gowkhtmltopdf 1.7`).

---

## Feature matrix (emitted / accepted / validated)

Three states, as required by #31. Empty cells are out of this plan.

| Feature | Emitted (1.4 default) | Emitted (1.7 opt-in) | Validated by |
|---------|----------------------|----------------------|--------------|
| File header | `%PDF-1.4` | `%PDF-1.7` | `writePDFHeader` test, `ParseSemantic` |
| Binary comment | yes | yes | existing header test |
| Classic xref + `startxref` | yes | yes | `TestXrefOffsets` |
| Trailer `/Root` `/Size` `/Info` | yes | yes | existing trailer tests |
| Trailer `/ID` | no (today) | yes, two deterministic strings | new unit test |
| Catalog `/Pages` `/Outlines` | yes | yes | existing catalog tests |
| Catalog `/Version` | no | omit (header already 1.7) | documented |
| Catalog `/Extensions` | no | **no** | negative test |
| Info Title/Producer/dates | yes, Latin-1 | yes; UTF-16BE + BOM when needed | `TestInfoDict` + 1.7 Unicode title |
| XMP Metadata stream | no | yes, Dublin Core + `pdf:Producer` only | byte / parse test |
| `pdfaid` / `pdfuaid` | no | **no** | negative test |
| OutputIntent / ICC | no | **no** | negative test |
| Structure tree / BDC | no | **no** | negative test |
| Embedded subset TTF | yes | yes | existing font tests |
| Type0 / CIDFontType2 | yes | yes | existing Type0 tests |
| JPEG / PNG XObjects | yes | yes | existing image tests |
| ExtGState opacity / SMask | yes | yes | existing image/opacity tests |
| Link annots + outlines | yes | yes | existing annot/outline tests |
| Object streams / xref streams | no | **no** | documented non-goal |
| Encryption / forms / signatures | no | **no** | documented non-goal |
| UTF-8 text strings | no | **no** (that is #32) | negative test |

"Accepted" in this product means: the writer will serialize the
combination. It does not mean the writer **reads** arbitrary PDF 1.7
files. `ParseSemantic` is an in-tree view of **our** output.

---

## In scope

1. Document the current 1.4 boundary against this matrix and [SPEC-NOTES.md](SPEC-NOTES.md).
2. Introduce the central version policy with `PDF14` and `PDF17` (reserve `PDF20`).
3. Version-aware header, trailer `/ID`, Info, UTF-16BE text strings, and non-claiming XMP.
4. Feature gates that fail before a misleading file is written.
5. Settings / CLI / library selection; default remains 1.4.
6. Convert-pipeline wiring so HTML → PDF 1.7 uses the same layout.
7. Structural tests, semantic parse, golden needles, honest docs.

## Out of scope

- Claiming PDF 1.7 by changing only `%PDF-1.4` to `%PDF-1.7`.
- Changing the default away from 1.4.
- PDF 2.0 serialization or UTF-8 text strings (sibling plan / #32).
- PDF/A-4, PDF/UA-2, tagging, ICC, OutputIntent (#33).
- Encryption, signatures, AcroForm, associated files, portfolios, 3D.
- Object streams, compressed xref, linearization, incremental update.
- JPEG 2000, CFF/OTTO, optional content groups, developer Extensions.
- A new `engine/` tree, JSON templates, or a second layout engine.
- HTML, CSS, pagination, flex/grid, or image-mode work.
- New direct modules. External validators may be optional skippable tests; they are not a phase-1 dependency.

---

## Phase map

```text
1 Version policy + header
  → 2 Catalog, trailer /ID, Info + UTF-16BE + XMP
      → 3 Font / image / content gates
          → 4 Settings, CLI, library
              → 5 Convert pipeline wiring
                  → 6 Validation + goldens
                      → 7 Docs
                          → 8 Closure
```

Phases 2 and 3 may overlap after phase 1 if they do not fight over
`finalize`. Phase 4 can start the settings key once the policy type
exists, but convert (phase 5) must not ship a user path until 2 and 3
can emit a real 1.7 file.

| Phase | File | Goal |
|------:|------|------|
| 1 | [phase-01-version-policy-and-header.md](phase-01-version-policy-and-header.md) | Policy type; header; default 1.4 |
| 2 | [phase-02-catalog-trailer-metadata.md](phase-02-catalog-trailer-metadata.md) | `/ID`, Info, UTF-16BE, XMP |
| 3 | [phase-03-fonts-images-content-gates.md](phase-03-fonts-images-content-gates.md) | Gates on existing emit paths |
| 4 | [phase-04-settings-cli-library.md](phase-04-settings-cli-library.md) | User-visible selection |
| 5 | [phase-05-convert-pipeline.md](phase-05-convert-pipeline.md) | HTML job uses the policy |
| 6 | [phase-06-validation-and-goldens.md](phase-06-validation-and-goldens.md) | Proof |
| 7 | [phase-07-docs-and-honesty.md](phase-07-docs-and-honesty.md) | Honest claims |
| 8 | [phase-08-closure.md](phase-08-closure.md) | Lint, test, default-1.4 proof |

---

## Success criteria (issue #31)

- [x] Current PDF 1.4 behavior remains covered by baseline fixtures and tests (`internal/convert/golden_test.go:737-754`, `internal/pdf/pdf_test.go:37-77`)
- [x] Feature matrix above is filled with emitted / accepted / validated states (`00-canonical-pdf-17-plan.md`)
- [x] PDF 1.7 output has an explicit version selection path and structural validation (`--pdf-version 1.7`, `WithPDFVersion("1.7")`, `internal/pdf/semantic_oracle_test.go:213-292`)
- [x] Unsupported combinations fail before misleading output (`internal/pdf/policy.go:56-78`, `internal/convert/convert_test.go:853-905`)
- [x] Tests cover header, catalog, xref, fonts, images, links, and metadata in scope (`internal/pdf/policy_test.go`, `internal/convert/golden_test.go:683-878`)
- [x] Default PDF 1.4 output remains unchanged (`internal/convert/convert_test.go:733-748`)
- [x] Docs distinguish PDF 1.7 **version** support from PDF 2.0 and from PDF/A-4 / PDF/UA-2 (`documentation/compatibility-matrix.md:257-261`, `documentation/deferred.md:74-76`, `documentation/architecture/09-pdf-writer.md:15-35, 490-502`)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Existing `internal/pdf` 1.4 writer | Object to version, not replace |
| ISO 32000-1 notes in SPEC-NOTES.md | Feature baseline |
| v0.2.1 convert / settings / library contracts | Selection path |

Provides `WriterPolicy` to the #32 plan.

Does **not** depend on layout changes, image mode, #32 implementation, or #33.
