# PDF 1.7 spec notes (ISO 32000-1:2008)

> **Purpose:** cited facts used by [00-canonical-pdf-17-plan.md](00-canonical-pdf-17-plan.md).
> **Primary source:** Adobe-hosted authorized copy of ISO 32000-1:2008 (technical material identical to the ISO edition; page and section numbers preserved).
> **File:** [PDF32000_2008.pdf](https://opensource.adobe.com/dc-acrobat-sdk-docs/standards/pdfstandards/pdf/PDF32000_2008.pdf)
> **Secondary identification:** [Library of Congress fdd000277](https://www.loc.gov/preservation/digital/formats/fdd/fdd000277.shtml)

This is not an implementation plan. Quotes are from the Adobe-hosted ISO 32000-1 text extracted 2026-08-14.

---

## Executive summary

ISO 32000-1 is PDF 1.7. A conforming file is **not** required to use every 1.5–1.7 feature (§2.1). Every non-deprecated 1.4 feature remains legal (§6). Object streams and xref streams are **optional** 1.5 syntax (§7.5.7, §7.5.8). Classic xref + trailer remain valid.

Honest 1.7 support in this writer is therefore **not** “rebuild the engine.” It is: declare `%PDF-1.7`, keep the existing 1.4 objects, and emit the document-level items ISO 32000-1 tells writers they **should** use (file `/ID`, preferred XMP metadata, Unicode text strings as UTF-16BE). UTF-8 text strings are **not** in 1.7; they arrive in ISO 32000-2 (PDF 2.0).

---

## 1. Conformance and version labels

| Fact | Source |
|------|--------|
| A conforming file “is not obligated to use any feature other than those explicitly required by ISO 32000-1.” | §2.1, p. 1 |
| Version self-identification is defined in §7.5.2 File Header. | §2.1 NOTE 1 |
| A conforming writer “shall comply with all requirements regarding the creation of PDF files” but ISO “does not prescribe any specific technical design.” | §2.3, p. 1 |
| All non-deprecated features from a previous version are included in later versions. ISO 32000-1 matches PDF 1.7 and can interpret files from 1.0 through 1.7. | §6, p. 10 |
| Features are marked `(PDF 1.N)` to show the version that first defined them. `(PDF 1.3)` means 1.0–1.2 did not specify it; 1.3 and later did. | §6, p. 10 |

Implication: a `%PDF-1.7` file that only uses 1.4 objects is **syntactically allowed**. Issue #31 still forbids calling that “PDF 1.7 support.” The plan therefore adds the 1.7 document-level recommendations below, not optional 1.5 compression syntax.

---

## 2. File header and catalog `/Version`

| Fact | Source |
|------|--------|
| First line shall be `%PDF–` followed by `1.N` where N is a digit 0–7. Readers shall accept `%PDF-1.0` through `%PDF-1.7`. | §7.5.2, p. 39 |
| Magic for 1.7 is `%PDF-1.7` (`25 50 44 46 2D 31 2E 37`). Catalog `/Version` may override the header. | [LoC fdd000277](https://www.loc.gov/preservation/digital/formats/fdd/fdd000277.shtml); §7.5.2 |
| Beginning with PDF 1.4, if catalog `Version` is present **and later than the header**, that value is used. If the header is later, or `Version` is absent, the header wins. | §7.5.2, p. 39; Table 28, §7.7.2, p. 73 |
| Catalog `Version` is a **name** (`/1.7`), not a number. | Table 28, §7.7.2 |
| If the file contains binary data, the header line shall be followed by a comment of at least four bytes with codes ≥ 128. | §7.5.2, p. 40 |
| Catalog may also have `Extensions` (ISO 32000 developer extensions). | Table 28; §7.12 |

Implication: emit `%PDF-1.7` plus the existing binary comment. Do **not** emit catalog `/Version` unless it would be later than the header. Do **not** emit an `Extensions` dictionary unless we implement a registered extension (we will not).

---

## 3. Classic xref and trailer

| Fact | Source |
|------|--------|
| A basic file is header + body + cross-reference table + trailer. | §7.5.1, p. 38 |
| Classic xref (`xref` / 20-byte `n`/`f` entries) remains defined. 1.5+ may alternatively use xref streams. | §7.5.4, p. 40 |
| Trailer required keys: `Size`, `Root`. Optional: `Info`, `ID`, `Encrypt`, `Prev`. | Table 15, §7.5.5, p. 43 |
| `ID` is required if `Encrypt` is present; optional otherwise (PDF 1.1). | Table 15 |
| NOTE 2: “Although this entry is optional, its absence might prevent the file from functioning in some workflows that depend on files being uniquely identified.” | Table 15 NOTE 2 |
| Object streams (`/Type /ObjStm`) are PDF 1.5, optional, for compact storage of non-stream objects. | §7.5.7, p. 45 |
| Xref streams (`/Type /XRef`) are PDF 1.5, optional. A file that uses them exclusively cannot be opened by pre-1.5 readers. | §7.5.8, p. 49 |

Implication: keep classic xref (same determinism story as the 2.0 plan). Adding object/xref streams is **not** required for a 1.7 claim and is out of scope.

---

## 4. File identifiers (`/ID`)

| Fact | Source |
|------|--------|
| File identifiers shall be defined by the optional trailer `ID` entry. “The ID entry is optional but **should** be used.” | §14.4, p. 551 |
| Value is an array of two byte strings. First = permanent id from original contents; second = id at last update. On first write both shall be equal. | §14.4 |
| Suggested uniqueness input: current time, file location, file size, Info dictionary values. Calculation “need not be reproducible; all that matters is that the identifier is likely to be unique.” | §14.4 |

Implication: 1.7 output **should** include `/ID`. This repo also needs **byte-stable goldens**, so compute the pair from injectable creation time + Info + version + a stable document fingerprint, not `time.Now()` or a pathname.

---

## 5. Metadata: Info vs XMP

| Fact | Source |
|------|--------|
| Metadata may live in a metadata stream (PDF 1.4) and/or the document information dictionary. | §14.3.1, p. 548 |
| “Document information dictionaries is the original way… Metadata streams were introduced in PDF 1.4 and is now the preferred method.” | §14.3.1 NOTE |
| Metadata stream dict: `/Type /Metadata`, `/Subtype /XML`. Contents are XMP XML. | §14.3.2; Table 315, p. 549 |
| Attach document-level XMP via catalog `Metadata`. | §14.3.2; §7.7.2 |
| If XMP’s date stamp is ≥ Info `ModDate`, XMP is authoritative; if Info `ModDate` is later, Info wins (writer unaware of XMP). | §14.3.2 |
| Info is the trailer `Info` dict. Keys: Title, Author, Subject, Keywords, Creator, Producer, CreationDate, ModDate, Trapped. Unknown values should be omitted, not empty strings. Values are text strings / dates. | §14.3.3; Table 317, p. 550 |
| XMP in PDF 1.4+ is Adobe’s RDF framework. | [LoC fdd000277](https://www.loc.gov/preservation/digital/formats/fdd/fdd000277.shtml) |

Implication: 1.7 should keep Info (still first-class in 1.7) **and** emit a non-claiming XMP stream. Do not emit `pdfaid` / `pdfuaid` (#33). PDF 2.0 later **deprecates** Info; that is the sibling plan, not this one.

---

## 6. Text strings (this is the 1.7 vs 2.0 fork)

| Fact | Source |
|------|--------|
| Text strings are for human-readable values: annotations, bookmark names, document information. | §7.9.2.2, p. 86; Table 35, p. 85 |
| Encoding shall be **PDFDocEncoding** **or** **UTF-16BE with a leading BOM**. | §7.9.2.2 |
| Unicode text strings start with bytes 254, 255 (`U+FEFF`). | §7.9.2.2 |
| PDFDocEncoding covers ISO Latin 1 (Annex D). UTF-16BE covers all Unicode. | §7.9.2.2 NOTE 2 |
| These conventions apply **outside** content streams. Content-stream strings select glyphs (clause 9), not text-string encoding. | §7.9.1, p. 84 |
| ISO 32000-2 adds UTF-8 as a third text-string encoding. ISO 32000-1 does not. | Contrast documented in [LoC / ISO 32000-2 notes](https://community.onlyoffice.com/t/update-standard-pdf-2-0-iso-32000-2-2020/7911) citing ISO 32000-2 vs 32000-1 |

Implication: 1.7 Info titles and outline titles that are not Latin-1 must be UTF-16BE + BOM. Do **not** emit UTF-8 text strings on the 1.7 path. Content-stream `Tj` (WinAnsi / Identity-H) stays as today.

---

## 7. Fonts, images, transparency (1.4 objects stay legal)

| Fact | Source |
|------|--------|
| Non-deprecated earlier features remain in 1.7. | §6 |
| Type0 / CID fonts, `ToUnicode`, `FontFile2` TrueType embedding are specified in clause 9 (9.7 composite fonts, 9.9 embedded font programs) and predate 1.7. | TOC; clause 9 |
| OpenType CFF (`FontFile3` / `OTTO`) is a later embedding path; this writer already rejects CFF. ISO 32000-1 does not require CFF for a 1.7 file. | Product constraint + §2.1 |
| JPEG (`DCTDecode`) is a normative reference (ISO/IEC 10918-1). JPEG 2000 is a separate normative reference and a 1.5+ filter — optional. | §3; §7.4 filters |
| Transparency (soft masks, ExtGState opacity) is PDF 1.4 and therefore included in 1.7. | §6; clause 11 (not re-derived here) |
| `ProcSet` in page resources is **obsolete beginning with PDF 1.4**. Writers “should continue to specify procedure sets” for old PostScript devices; readers must not depend on them. | §14.2, p. 548 |

Implication: reuse the current font/image/opacity emit paths. Do not add JPEG 2000, CFF, or ProcSet as a 1.7 gate. Gate unimplemented 1.5–1.7 families (encryption, OCG, object streams) instead of emitting them.

---

## 8. What 1.7 introduced that we will **not** implement

These are real 1.5–1.7 / ISO 32000-1 features. They are optional for a conforming file (§2.1) and out of issue #31 scope:

| Feature | First version | Spec | Why not |
|---------|---------------|------|---------|
| Object streams | 1.5 | §7.5.7 | Optional compression; breaks simple xref tests |
| Xref streams | 1.5 | §7.5.8 | Optional; excludes pre-1.5 readers |
| Incremental update | long-standing | §7.5.6 | We always rewrite |
| Linearization | Annex F | Annex F | Network hint; not our write model |
| Encryption / AES | 1.1–1.6 | §7.6 | Issue #31 out of scope |
| Optional content (layers) | 1.5 | §8.11 | Not an HTML-template need |
| JPEG 2000 | 1.5 | §7.4 | No JP2 pipeline |
| Portable collections | 1.7 | (portfolio) | Product non-goal |
| Developer `Extensions` | ISO 32000-1 | §7.12 | We implement no ADBE extension |
| Tagged PDF / structure tree | 1.3–1.7 | §14.7–14.8 | Issue #33 |
| Output intents / ICC | prepress | §14.11 | Issue #33 |

---

## 9. Mapping onto this repository

| Spec item | 1.4 writer today | 1.7 plan |
|-----------|------------------|----------|
| Header `1.N` | `const Version = "1.4"` | Policy → `%PDF-1.7` |
| Binary comment | already emitted | keep |
| Classic xref | yes | keep |
| Trailer `/ID` | absent | emit, deterministic |
| Info | Latin-1 `pdfString` | keep + UTF-16BE when needed |
| XMP `Metadata` | absent | emit, no A-4/UA-2 claims |
| Catalog `/Version` | absent | omit if header is already 1.7 |
| Content `Tj` / fonts / images | 1.4-legal | unchanged; gates only |
| UTF-8 text strings | n/a | **not** 1.7 — that is the 2.0 plan |
