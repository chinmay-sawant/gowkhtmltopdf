# Phase 6 — Validation and Goldens

> **Parent:** `plans/0.2.2/pdf-1.7-plan/00-canonical-pdf-17-plan.md`
> **Status:** not started
> **Estimated effort:** 3–5 days
> **Depends on:** Phases 1–5
> **Unblocks:** phases 7–8

---

## Overview

#31 requires structural fixtures, golden outputs, and an independent
parser or validator. This repo already has:

- Package tests in `internal/pdf/*_test.go`
- `ParseSemantic` (`semantic.go`) — in-tree view of **our** files
- Convert goldens under `testdata/golden/` (1.4 envelopes)

Use those. Do not add veraPDF as a required gate (A-4 / UA-2 is #33).
An optional external parser is welcome if it is skippable when missing.

Do not replace the 1.4 golden corpus. Add 1.7 needles beside it.

---

## Executive Summary

| Gate | What it proves |
|------|----------------|
| Unit (writer) | Header, xref, `/ID`, Metadata, Info, UTF-16BE title, fonts, images on 1.7 |
| Semantic parse | `ParseSemantic(data).Version == "1.7"` and page text/annots still extract |
| Convert golden needles | One small HTML fixture emits `%PDF-1.7`, `/ID`, `/Metadata`, and not `pdfaid` or UTF-8 BOM-less titles |
| 1.4 regression | Existing goldens and `TestDeterministicOutput` unchanged |
| Optional parser | If present, opens the 1.7 fixture; if absent, test skips |

---

## Phase 6 checklist

### 6.1 Writer unit tests

- [ ] 1.7 header + binary comment
- [ ] 1.7 xref offsets
- [ ] 1.7 trailer `/ID` present (two equal strings on first write); 1.4 trailer unchanged
- [ ] 1.7 catalog `/Metadata` present; bytes contain no `pdfaid` / `pdfuaid`
- [ ] 1.7 Unicode Info title is UTF-16BE (`FE FF` prefix), not UTF-8
- [ ] 1.7 determinism (two writes, equal bytes)
- [ ] 1.7 rich document: text + image + link + outline
- [ ] Short-writer contract still fails closed on 1.7
- [ ] Default `NewDocument` tests still assert `%PDF-1.4`

### 6.2 Semantic parser

- [ ] `ParseSemantic` accepts a 1.7 file this package emits — `semantic.go` / `semantic_*_test.go`
- [ ] `SemanticDoc.Version` is `"1.7"`
- [ ] Page text, images, and annots still populate
- [ ] A 1.4 file still reports `"1.4"`

### 6.3 Convert / golden needles

- [ ] Add one small committed HTML fixture (or reuse a tiny existing one) converted with version 1.7
- [ ] Needle assertions: `%PDF-1.7`, trailer `/ID`, `/Metadata`, Producer contains `1.7`, absence of `pdfaid`
- [ ] Same fixture without the setting still matches the 1.4 envelope
- [ ] Do not pixel-diff PDFs as the default gate
- [ ] TOC + HF job covered by a structural test (page count + header version), not a new visual corpus

### 6.4 Negative tests

- [ ] Unsupported version string never produces a file that claims 1.7
- [ ] `2.0` still errors (until #32)
- [ ] 1.7 + a not-implemented combination errors before `Write`
- [ ] Image mode has no version claim

### 6.5 Optional independent parser

- [ ] Document one optional command (e.g. `qpdf --check` or `mutool info`) in the phase notes or a test helper
- [ ] Test skips when the binary is missing; does not fail CI
- [ ] Do **not** add veraPDF flavour `4` / `ua2` as a #31 gate

---

## Explicitly out of scope

- Refreshing every `testdata/golden` PDF
- `make samples` regeneration as a required gate
- Compliance profiles
- PDF 2.0 goldens (#32)

---

## Done when

A 1.7 HTML conversion is proven by unit + semantic + needle tests, and
the 1.4 corpus is still green.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 1–5 | Phase 7 documentation claims |
