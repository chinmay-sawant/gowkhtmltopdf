# Phase 7 — Optional Product Features

**Status:** ⏸️ NOT STARTED — deprioritized; Phase 1–5 complete  
**Depends on:** Phase 1–5 for anything that must remain A-4/UA-2; features may have non-compliant modes  
**Base plan refs:** §2.4, §11 Phase G, §15 out of scope list

---

## Goal

Add product-facing engine capabilities **without breaking** the default compliant profile. Each feature is its own sub-checklist; implement only as needed.

---

## 7.1 Bookmarks / outlines

- [ ] Outline tree objects (`/Outlines`, outline items with `/Title`, `/Parent`, `/Next`, `/Prev`, `/First`, `/Last`, `/Count`, `/Dest` or actions)
- [ ] Catalog `/Outlines` + optional `/PageMode /UseOutlines`
- [ ] UA-2: `/Sect` structure targets for destinations when tagged
- [ ] Fixture + open-in-viewer check

## 7.2 Links (non-structure bits already partial in phase 5)

- [ ] Link annotations: `/Subtype /Link`, `/Rect`, `/Border`, `/A << /S /URI … >>`
- [ ] Ensure phase-5 Link StructElem + ParentTree + `/Tabs /S` still pass
- [ ] veraPDF ua2 on `link-annot` fixture

## 7.3 AcroForm / widgets

- [ ] Catalog `/AcroForm` with `/Fields`
- [ ] Widget annots `/Subtype /Widget`
- [ ] Field value tags: `/V`, appearance state `/AS` (form values — not PDF version)
- [ ] Default appearance `/DA` with embedded font under A-4
- [ ] Decision: A-4 + forms scope (may restrict); document mode matrix
- [ ] Optional XFDF fill path (separate sub-plan)

## 7.4 Digital signatures

- [ ] Signature field + `/V` value (sig dictionary)
- [ ] Byte-range placeholder + post-pass CMS embed
- [ ] Visible appearance optional
- [ ] ECDSA / RSA support as required
- [ ] Compliance: confirm whether signed files still pass veraPDF for target profile
- [ ] Non-compliant fast path allowed for benches

## 7.5 Encryption

- [ ] Encrypt dict (`/Filter`, `/V`, `/R`, `/O`, `/U`, `/P`, `/CF`, …) — product encrypt `/V` is **encryption version**, not form value
- [ ] Stream/string encryption by object number
- [ ] Trailer `/Encrypt`
- [ ] **Default off under PDF/A-4** until explicitly allowed and validated
- [ ] Separate non-A fixtures only for first implementation

## 7.6 SVG / form XObjects

- [ ] Parse SVG paths → PDF path operators
- [ ] `/Subtype /Form` XObject with `/BBox`, `/Resources`, stream
- [ ] Structure: treat as `/Figure` with `/Alt` when tagged

## 7.7 Merge / split / redact

- [ ] **Separate plans** — not part of core generate path
- [ ] Placeholder only: do not implement inside phase 1–5 modules

---

## Compliance rules for phase 7

- [ ] Default `ModePDFA4` + `ModePDFUA2` path must remain green on phase-5 matrix after each feature that can appear in compliant docs
- [ ] Features incompatible with A-4 stay behind explicit flags
- [ ] Add fixture rows to veraPDF matrix when feature ships in compliant mode

---

## Acceptance criteria (per feature)

- [ ] Feature works in at least one dedicated fixture
- [ ] Documented interaction with A-4 / UA-2
- [ ] No regression on `minimal-text` dual veraPDF gates

---

## Done when

Chosen product features ship behind flags with documented compliance matrix; core dual-mode path still passes phase-5 gates.
