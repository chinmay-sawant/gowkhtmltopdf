# PDF 1.7 compliance spec notes (PDF/A-3 + PDF/UA-1)

> **Purpose:** cited facts for [00-canonical-pdf-17-compliance-plan.md](00-canonical-pdf-17-compliance-plan.md).
> **This is not an implementation plan.**

The 1.7 **version** plan ([../pdf-1.7-plan/](../pdf-1.7-plan/)) only makes a legal ISO 32000-1 file. This note is the **highest conformance pair that still sits on PDF 1.7**. PDF/A-4 and PDF/UA-2 require PDF 2.0 and stay on [#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33).

---

## Executive summary

| Role | 1.7 highest | 2.0 highest (#33) |
|------|-------------|-------------------|
| Base file format | PDF 1.7 / ISO 32000-1:2008 | PDF 2.0 / ISO 32000-2 |
| Archival | **PDF/A-3** (ISO 19005-3:2012), level **A** | PDF/A-4 (ISO 19005-4) |
| Accessibility | **PDF/UA-1** (ISO 14289-1:2014) | PDF/UA-2 (ISO 14289-2:2024) |
| Default dual claim | PDF 1.7 + PDF/A-3a + PDF/UA-1 | PDF 2.0 + PDF/A-4 + PDF/UA-2 |
| veraPDF flavours | `-f 3a` and `-f ua1` | `-f 4` and `-f ua2` |

PDF/A-3 is the last PDF/A part defined on PDF 1.7. It is PDF/A-2 plus associated files. Level **A** is the highest of the A/B/U triple (tagged + Unicode + visual). PDF/UA-1 is the accessibility profile for ISO 32000-1; it may also claim PDF/A-2 or PDF/A-3, not PDF/A-1.

A header of `%PDF-1.7` plus non-claiming XMP is **not** PDF/A-3 or PDF/UA-1.

---

## 1. Why not PDF/A-2, A-1, or A-4?

| Profile | Base PDF | Why it is or is not the 1.7 ceiling |
|---------|----------|-------------------------------------|
| PDF/A-1 (ISO 19005-1) | 1.4 | Older; no transparency; PDF Association does **not** recommend pairing it with PDF/UA-1 |
| PDF/A-2 (ISO 19005-2) | 1.7 | Valid 1.7 archival. A-3 is the same rules plus embedded files |
| **PDF/A-3 (ISO 19005-3)** | **1.7** | **Highest archival part on 1.7** |
| PDF/A-4 (ISO 19005-4) | 2.0 | Wrong base file. That is #33 |

Inside PDF/A-3 the levels are:

| Level | Meaning | XMP `pdfaid:conformance` |
|-------|---------|--------------------------|
| B | Visual reproducibility (fonts embedded, no encryption, OutputIntent, …) | `B` |
| U | B + Unicode mapping (`ToUnicode`) for all text | `U` |
| **A** | U + tagged structure (logical structure tree) | **`A`** |

**Highest 1.7 archival claim: PDF/A-3a.**

Sources: [PDF Association — ISO 19005-3](https://pdfa.org/resource/iso-19005-3-pdf-a-3/), [LoC PDF/A family fdd000318](https://www.loc.gov/preservation/digital/formats/fdd/fdd000318.shtml), [PDFlib PDF/A overview](https://www.pdflib.com/pdf-knowledge-base/pdfa/the-pdfa-standards/).

---

## 2. Why PDF/UA-1, not UA-2?

| Profile | Base PDF | Notes |
|---------|----------|-------|
| **PDF/UA-1** (ISO 14289-1:2014) | **PDF 1.7 / ISO 32000-1** | Tagged PDF per ISO 32000-1 §14.8. Widely deployed. |
| PDF/UA-2 (ISO 14289-2:2024) | PDF 2.0 / ISO 32000-2 | New structure namespaces, MathML, PDF 2.0 tags. That is #33. |

PDF/UA-1 documents **may also** conform to PDF/A-2 or PDF/A-3. Dual claim needs the PDF Association XMP extension schema for `pdfuaid` inside a PDF/A file.

Sources: [PDF Association — ISO 14289-1](https://pdfa.org/resource/iso-14289-pdfua/), [LoC fdd000350](https://www.loc.gov/preservation/digital/formats/fdd/fdd000350.shtml).

---

## 3. PDF/A-3 objects this writer must add

ISO 19005-3 is paywalled. The following are the stable, publicly documented requirements that veraPDF flavour `3a` enforces and that every PDF/A writer implements:

| Requirement | Typical emit |
|-------------|--------------|
| File is PDF 1.7 | `%PDF-1.7` (already from the 1.7 version plan) |
| XMP identification | `pdfaid:part` = `3`, `pdfaid:conformance` = `A` (or `B`/`U` for lesser modes) |
| Catalog `/Metadata` | `/Type /Metadata /Subtype /XML` (1.7 plan already emits **non-claiming** XMP; this plan must add the pdfaid properties) |
| OutputIntent | Catalog `/OutputIntents` → `/S /GTS_PDFA1` + ICC DestOutputProfile (sRGB is the usual office profile) |
| Fonts | Every used font fully embedded (we already subset-embed Liberation / Type0) |
| Unicode (levels U and A) | `ToUnicode` on every text font |
| File identifier | Trailer `/ID` (already on 1.7) |
| No encryption, JS, launch actions, external content | Already rejected by `WriterPolicy` |
| Device color | Images/content use ICCBased or DefaultRGB/Gray tied to the OutputIntent — not bare DeviceRGB under A-3 |
| Transparency | Allowed in A-2/A-3 (unlike A-1). Soft-masks we already emit are acceptable if color is managed |
| Associated files | Allowed in A-3. We do **not** need them for HTML→PDF; claiming A-3 without attachments is valid |

Info dictionary may remain, but must not contradict XMP (ISO 32000-1 §14.3.2 date-stamp rule already noted in the version SPEC-NOTES).

---

## 4. PDF/UA-1 objects this writer must add

Public requirements (LoC fdd000350 + ISO 32000-1 tagged PDF + PDF Association):

| Requirement | Typical emit |
|-------------|--------------|
| XMP identification | `pdfuaid:part` = `1` (+ extension schema when dual-claimed with PDF/A) |
| Catalog language | `/Lang` (e.g. `(en-US)`), from HTML `lang` or a setting |
| Marked | `/MarkInfo << /Marked true >>` |
| Title shown | `/ViewerPreferences << /DisplayDocTitle true >>` and a non-empty document title (`dc:title` / `--title`) |
| Structure tree | Catalog `/StructTreeRoot`; standard types from ISO 32000-1 §14.8.4 (`Document`, `H1`–`H6`, `P`, `Table`/`TR`/`TH`/`TD`, `L`/`LI`, `Figure`, `Link`, …) or a `/RoleMap` |
| Real content tagged | `/Tag << /MCID n >> BDC` … `EMC` in content streams |
| Artifacts not tagged | Header/footer/page numbers: `/Artifact << /Type /Pagination >> BDC` … `EMC` |
| Reading order | Structure order = logical reading order |
| Figures | `/Figure` + `/Alt` (from HTML `alt`) |
| Links | Link StructElem + `/OBJR` to the annotation; page `/Tabs /S` |
| ParentTree | Number tree, index = MCID; page `/StructParents` |
| Fonts embedded | Same as PDF/A |
| No dynamic XFA | Already out of scope |

PDF/UA-1 does **not** use the PDF 2.0 `/Namespace` (`http://iso.org/pdf2/ssn`). That is UA-2.

---

## 5. Dual claim (the default compliant 1.7 profile)

One file can be both PDF/A-3a and PDF/UA-1:

- Header `%PDF-1.7`
- XMP contains `pdfaid:part=3`, `pdfaid:conformance=A`, **and** `pdfuaid:part=1` with the UA extension schema
- OutputIntent + ICC (A-3)
- Full structure tree that also satisfies UA-1 (A-3a only requires *some* tags; UA-1 is stricter about roles, Alt, reading order)
- veraPDF **both** `-f 3a` and `-f ua1` PASS

Lesser modes (optional later, not the headline):

| Mode | When |
|------|------|
| PDF/A-3b only | Visual archive, no Unicode/tagging claim |
| PDF/A-3u only | Searchable archive, no accessibility claim |
| PDF/UA-1 only | Accessible, not an archival profile |

This plan’s **done** state is the dual headline, not the lesser modes.

---

## 6. Mapping onto this repository

| Concern | Today | This plan |
|---------|-------|-----------|
| `%PDF-1.7`, `/ID`, Info, non-claiming XMP | Shipped in #31 | Keep; **add claims** to XMP |
| `WriterPolicy.ConformanceProfile` | Any non-empty value → `ErrConformanceProfilesUnsupported` (#33) | Accept `PDF/A-3a`, `PDF/UA-1`, and dual; keep rejecting A-4 / UA-2 |
| Fonts / Type0 / ToUnicode | Already embedded | Force embed under A-3 / UA-1; fail if a face cannot be embedded |
| Images | DeviceRGB + SMask | Rewrite to ICCBased / DefaultRGB under A-3 |
| OutputIntent / ICC | none | New objects in `internal/pdf` |
| Structure tree / BDC | none | New; layout must mark ops |
| Layout | Display list, no tags | Map HTML semantics → structure types |
| veraPDF | not in repo | Optional skippable tests, flavours `3a` and `ua1` |

`internal/imageout` is out of this plan.

---

## 7. Sources

- ISO 19005-3:2012 (PDF/A-3) — [ISO](https://www.iso.org/standard/57229.html), [PDF Association](https://pdfa.org/resource/iso-19005-3-pdf-a-3/)
- ISO 14289-1:2014 (PDF/UA-1) — [ISO](https://www.iso.org/standard/64599.html), [PDF Association](https://pdfa.org/resource/iso-14289-pdfua/), no-cost bundle via [pdfa-inc.org](https://www.pdfa-inc.org/product/pdf-ua-bundle/)
- ISO 32000-1:2008 tagged PDF §14.7–14.8 — [Adobe-hosted copy](https://opensource.adobe.com/dc-acrobat-sdk-docs/standards/pdfstandards/pdf/PDF32000_2008.pdf)
- [LoC PDF/A family](https://www.loc.gov/preservation/digital/formats/fdd/fdd000318.shtml), [LoC PDF/UA-1 fdd000350](https://www.loc.gov/preservation/digital/formats/fdd/fdd000350.shtml)
- veraPDF flavours: `3a` / `3b` / `3u` / `ua1` — [verapdf.org](https://verapdf.org/)
