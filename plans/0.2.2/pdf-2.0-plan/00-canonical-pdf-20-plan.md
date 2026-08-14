# 00 — PDF 2.0 Version Support (Canonical Execution Ledger)

> **Parent:** `plans/0.2.2/README.md` — epic [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)
> **Issue:** [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32)
> **Status:** completed (2026-08-15) — all 8 phases done; PDF 2.0 opt-in shipped, default still 1.4
> **Estimated effort:** 3–5 weeks across phases 1–8
> **Constraint:** pure Go, no CGO, no new direct modules. Do not add veraPDF, Brotli, or a second PDF writer.
> **Ordering principle:** version policy and serialization first, then call-site selection, then proof, then docs.
> **Workflow:** [`skills/phase-wise-checklist/SKILLS.md`](../../../skills/phase-wise-checklist/SKILLS.md)

---

## Overview

The current writer is a **PDF 1.4 sink**. `internal/pdf.NewDocument` always
emits `%PDF-1.4`, a classic xref, an Info dictionary, and
`Producer "gowkhtmltopdf 1.4"`. Layout, fonts, images, outlines, and
links already exist. They must keep working.

Issue #32 is a **version-policy and serialization** change, not a new
engine. Callers opt into PDF 2.0. The default stays 1.4. A header-only
swap is not support.

This ledger used to describe `gocorepdfengine` (JSON templates, a second
layout engine, Liberation-from-scratch, PDF/A-4, PDF/UA-2, Zerodha
benches). That product is not this repository. Those phases are gone.

---

## Where PDF 2.0 lands

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
pdf.NewDocument / NewDocumentWithPolicy   ← NEW: WriterPolicy
        │
        ▼
internal/layout.PaintContext         ← UNCHANGED (paint ops are version-agnostic)
        │
        ▼
pdfPipeline.Assemble                 ← SMALL: pass policy; Producer string
        │
        ▼
Document.Write / finalize / writeTo  ← VERSION-AWARE header, catalog, trailer, strings
```

`internal/imageout` keeps using `internal/pdf` only for fonts, shaping,
and glyph outlines. It never writes a PDF and is out of this plan.

### Package ownership

| Concern | Primary code | Change in this plan |
|---------|--------------|---------------------|
| Version policy | `internal/pdf` (`WriterPolicy` from the 1.7 plan) | Implement `PDF20` on the existing type. No raw `"2.0"` checks in content, fonts, or images |
| Header / xref / trailer | `internal/pdf/pdf.go` (`writePDFHeader`, `writePDFTrailer`, `const Version`) | Header from policy; trailer `/ID` on 2.0; classic xref stays |
| Catalog / Info | `internal/pdf/pdf.go` (`catalogDict`, `infoDict`, `finalize`) | Optional catalog `/Version`; 2.0 Metadata stream; Info kept (deprecated, still emitted) |
| Text strings | `internal/pdf/pdf.go` (`pdfString`) | 1.4 Latin-1 fold unchanged; 2.0 UTF-8 text strings for Info / outline titles |
| Fonts / images / content | `internal/pdf/fonttype0.go`, `images.go`, `content.go` | Gates only. No new subsetter, no new paint API |
| Semantic parse | `internal/pdf/semantic.go` | Already exposes `SemanticDoc.Version`; tests must cover 2.0 |
| Settings | `internal/settings/settings.go`, `reflect.go` | `PdfGlobal` field + dotted key |
| CLI | `internal/cli/flags.go` (`addGlobalFlags`) | `--pdf-version` (PDF mode only) |
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

## Relationship to #31 and #33

```text
#31 PDF 1.7          sibling plan: introduces WriterPolicy{PDF14, PDF17}, reserves PDF20
        │
        ▼
#32 PDF 2.0          this plan: implement PDF20 + 2.0 emit rules (UTF-8 strings)
        │
        ▼
#33 PDF/A-4 + UA-2   later: claiming XMP, OutputIntent, structure tree
```

`WriterPolicy` is owned by [../pdf-1.7-plan/](../pdf-1.7-plan/). This
plan **extends** that type with working `PDF20` rules. Do **not**
create a second policy type. If the 1.7 type is not in the tree yet,
land it from the 1.7 plan first (or land the shared type here with
`PDF17` already wired, then keep this plan's work to 2.0 deltas).

#33 may consume the 2.0 path. This plan must not emit `pdfaid` /
`pdfuaid`, OutputIntent, ICC, MarkInfo, or a structure tree.

---

## Version policy seam

The exact names can move during implementation. The invariant cannot:
version and feature decisions happen **before** objects are emitted.

```go
type PDFVersion int

const (
    PDF14 PDFVersion = iota
    PDF17            // reserved for #31; reject or treat as not-implemented until that plan lands
    PDF20
)

type WriterPolicy struct {
    Version PDFVersion
}

func NewDocument() *Document                    // PDF 1.4 default
func NewDocumentWithPolicy(p WriterPolicy) (*Document, error)
```

Policy methods (not call-site `if version ==`) own the rules:

| Question | 1.4 (default) | 2.0 |
|----------|---------------|-----|
| Header | `%PDF-1.4` | `%PDF-2.0` |
| Classic xref + trailer | yes | yes (no xref stream in this plan) |
| Trailer `/ID` | optional / unchanged | required, deterministic |
| Info dictionary | emitted | still emitted (deprecated in ISO 32000-2, not forbidden) |
| XMP `/Metadata` | no | yes, **without** A-4 / UA-2 claims |
| Document text strings | Latin-1 `pdfString` | UTF-8 (ISO 32000-2) |
| Content-stream text (`Tj`) | WinAnsi / Identity-H as today | unchanged |
| Font / image objects | as today | as today; reject combinations the writer cannot prove |

`const Version = "1.4"` in `pdf.go` must stop being a process-wide
header. Producer metadata should follow the **document** version
(`gowkhtmltopdf 1.4` vs `gowkhtmltopdf 2.0`), not a package constant
alone.

> **Shipped:** `WriterPolicy` (policy.go) carries `PDF14` / `PDF17` /
> `PDF20`; `HeaderVersion()` and `ProducerVersion()` spell the header and
> Producer tokens from the policy; `const Version = "1.4"` remains only as
> the legacy default constant. Catalog `/Version` is **not emitted** on
> any version — the file header is the sole version authority (decision
> recorded in `internal/pdf/pdf.go` `catalogDict`, matching the #31 1.7
> sibling).

---

## Feature matrix (emitted / accepted / validated)

Three states, as required by #32. Empty cells are out of this plan.

| Feature | Emitted (1.4 default) | Emitted (2.0 opt-in) | Validated by |
|---------|----------------------|----------------------|--------------|
| File header | `%PDF-1.4` | `%PDF-2.0` | `writePDFHeader` test, `ParseSemantic` |
| Binary comment | yes | yes | existing header test |
| Classic xref + `startxref` | yes | yes | `TestXrefOffsets` |
| Trailer `/Root` `/Size` `/Info` | yes | yes | existing trailer tests |
| Trailer `/ID` | no (today) | yes, two deterministic strings | new unit test |
| Catalog `/Pages` `/Outlines` | yes | yes | existing catalog tests |
| Catalog `/Version` | **not emitted** | **not emitted** — decision: the header is the sole version authority, matching the 1.7 sibling; `catalogDict` never adds `/Version` | `TestPDF20CatalogAndMetadataStream` regex pins the exact catalog shape and asserts no `/Version` |
| Info Title/Producer/dates | yes | yes (UTF-8 values when needed) | `TestInfoDict` + 2.0 case |
| XMP Metadata stream | no | yes, Dublin Core + `pdf:Producer` only, **without** A-4 / UA-2 claims | byte / parse test |
| `pdfaid` / `pdfuaid` | no | **no** | negative test |
| OutputIntent / ICC | no | **no** | negative test |
| Structure tree / BDC | no | **no** | negative test |
| Embedded subset TTF | yes | yes | existing font tests |
| Type0 / CIDFontType2 | yes | yes | existing Type0 tests |
| JPEG / PNG XObjects | yes | yes | existing image tests |
| Link annots + outlines | yes | yes | existing annot/outline tests |
| Object streams / xref streams | no | **no** | documented non-goal |
| Encryption / forms / signatures | no | **no** | documented non-goal |

"Accepted" in this product means: the writer will serialize the
combination. It does not mean the writer **reads** arbitrary PDF 2.0
files. `ParseSemantic` is an in-tree view of **our** output, not a
general PDF 2.0 reader.

---

## In scope

1. Document the current 1.4 boundary against this matrix.
2. Extend the #31 `WriterPolicy` with working `PDF20` (do not invent a second type).
3. Version-aware header, catalog, trailer `/ID`, Info, and non-claiming XMP.
4. UTF-8 document strings on the 2.0 path only.
5. Feature gates that fail before a misleading file is written.
6. Settings / CLI / library selection; default remains 1.4.
7. Convert-pipeline wiring so HTML → PDF 2.0 uses the same layout.
8. Structural tests, semantic parse, golden needles, honest docs.

## Out of scope

- Claiming PDF 2.0 by changing only `%PDF-1.4` to `%PDF-2.0`.
- Changing the default away from 1.4.
- PDF 1.7 feature work (sibling plan / #31), except sharing `WriterPolicy`.
- PDF/A-4, PDF/UA-2, tagging, ICC, OutputIntent (#33).
- Encryption, signatures, AcroForm, associated files, 3D, geospatial.
- Object streams, compressed xref, linearization, incremental update.
- A new `engine/` tree, JSON templates, or a second layout engine.
- HTML, CSS, pagination, flex/grid, or image-mode work.
- New direct modules (including veraPDF Go bindings). External
  validators may be **optional** skippable tests later; they are not a
  phase-1 dependency.

---

## Phase map

```text
1 Version policy + header
  → 2 Catalog, trailer /ID, Info + XMP
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
can emit a real 2.0 file.

| Phase | File | Goal | Status |
|------:|------|------|--------|
| 1 | [phase-01-version-policy-and-header.md](phase-01-version-policy-and-header.md) | Policy type; header; default 1.4 | completed (2026-08-15) |
| 2 | [phase-02-catalog-trailer-metadata.md](phase-02-catalog-trailer-metadata.md) | Catalog, `/ID`, Info + XMP | completed (2026-08-15) |
| 3 | [phase-03-fonts-images-content-gates.md](phase-03-fonts-images-content-gates.md) | Gates on existing emit paths | completed (2026-08-15) |
| 4 | [phase-04-settings-cli-library.md](phase-04-settings-cli-library.md) | User-visible selection | completed (2026-08-15) |
| 5 | [phase-05-convert-pipeline.md](phase-05-convert-pipeline.md) | HTML job uses the policy | completed (2026-08-15) |
| 6 | [phase-06-validation-and-goldens.md](phase-06-validation-and-goldens.md) | Proof | completed (2026-08-15) |
| 7 | [phase-07-docs-and-honesty.md](phase-07-docs-and-honesty.md) | Honest claims | completed (2026-08-15) |
| 8 | [phase-08-closure.md](phase-08-closure.md) | Lint, test, default-1.4 proof | completed (2026-08-15) |

---

## Success criteria (issue #32)

- [x] Feature matrix above is filled with emitted / accepted / validated states
  (matrix reconciled to shipped behavior on 2026-08-15: catalog `/Version`
  row changed from "optional" to "not emitted" with the decision recorded
  above; every row's "Validated by" names a real green test)
- [x] Callers can select PDF 2.0 without changing the 1.4 default
  (`--pdf-version 2.0` (PDF mode), `PdfGlobal.PdfVersion`,
  `PdfGlobalOptions.WithPDFVersion("2.0")`, dotted `Set("pdfversion",
  "2.0")`; default/empty stays 1.4 — `TestConvertPDFVersion`,
  `TestPDFVersionAPI`, `TestPDFVersionFlag`,
  `TestConvertPDF20GoldenNeedles` step 2, `TestDefaultNewDocumentAsserts14`)
- [x] Generated 2.0 files pass structural parse and version fixtures
  (`ParseSemantic(data).Version == "2.0"` with page text/images/annots —
  `TestPDF20RichDocument`, `TestSemanticPDF20OracleConvertedFixtures`;
  golden needles `TestConvertPDF20GoldenNeedles` +
  `TestConvertPDF20MultiPageTOCHF`; qpdf/mutool optional check
  `TestOptionalPDFValidation`)
- [x] Unsupported combinations fail before misleading output
  (`policy.Validate()` rejects encryption/forms/signatures/object streams
  on 2.0; A-4/UA-2 profiles return
  `pdf.ErrConformanceProfilesUnsupported` (deferred #33); garbage
  versions fail before any bytes with `settings.ErrInvalidPDFVersion` —
  `TestPDFVersionNegativeValidation`; short writers fail closed
  `TestPDF20ShortWriterContract`)
- [x] PDF 1.4 (and 1.7, if present) regression fixtures stay green
  (`go test ./...` exit 0 on 2026-08-15: full golden corpus, 1.7 golden
  needles, `TestDeterministicOutput`, and every 1.4/1.7 suite pass)
- [x] Docs distinguish PDF 2.0 **version** support from PDF/A-4 and PDF/UA-2
  (phase 7 closed: README, compatibility matrix, deferred, cli, library
  API, architecture docs all state "2.0 = opt-in version, not a
  conformance claim"; `make claim-scan` clean)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Existing `internal/pdf` 1.4 writer | Object to version, not replace |
| #31 `WriterPolicy` ([../pdf-1.7-plan/](../pdf-1.7-plan/)) | Single policy seam |
| v0.2.1 convert / settings / library contracts | Selection path |

Does **not** depend on layout changes, image mode, or #33.
