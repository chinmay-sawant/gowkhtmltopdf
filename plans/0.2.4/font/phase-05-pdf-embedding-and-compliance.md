# Font Phase 5 — PDF Embedding and Conformance Profiles

> **Status:** planned
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phases 3–4
> **Unblocks:** Phase 6

## Overview

Prove that every selected external or bundled face travels through the
existing writer correctly. This phase does not add a second font writer. It
extends the existing `subsetFont` → `FontFile2` → simple/Type0 emission path
and its compliance tests.

## 5.1 Writer coverage

- [ ] Render a supplied custom face with Latin-1-only text through the simple
  TrueType path.
- [ ] Render the same face with non-Latin text through Type0/CIDFontType2,
  Identity-H, CIDToGIDMap, and ToUnicode.
- [ ] Verify composite glyph closure and advance widths survive subsetting.
- [ ] Verify regular, bold, italic, and bold-italic face resources are distinct
  and selected correctly.
- [ ] Verify two files sharing a display name retain distinct subset cache
  identities.
- [ ] Verify font descriptor flags, `/FontName`, `/BaseFont`, `/Widths`, and
  `/ToUnicode` remain valid for custom faces.
- [ ] Verify missing glyphs use the documented fallback without losing text
  extraction semantics.
- [ ] Add an embed-preflight step over the actual used rune set before final
  paint/output commit.
- [ ] If preflight fails for an optional face, select the next valid fallback
  and re-layout the affected object so changed metrics are reflected in line
  breaks and pagination.
- [ ] Prove that a failed fallback does not leave partially painted content or
  a stale font resource in the final document.

## 5.2 Version and conformance matrix

For each accepted test face, generate and inspect:

| Output | Required evidence |
|--------|--------------------|
| PDF 1.4 unclaimed | valid header, embedded font, `ToUnicode` |
| PDF 1.7 unclaimed | valid version path and embedded font |
| PDF 1.7 + PDF/A-3a + PDF/UA-1 | XMP claim, `FontFile2`, `ToUnicode`, tagged output, no unembedded standard font |
| PDF 2.0 unclaimed | valid version path and embedded font |
| PDF 2.0 + PDF/A-4 + PDF/UA-2 | XMP claim, `FontFile2`, `ToUnicode`, namespaced/tagged output, no unembedded standard font |

- [ ] Extend focused PDF tests for external/custom faces under both version
  policies.
- [ ] Extend compliance tests to exercise a custom face, not only bundled
  Liberation/DejaVu.
- [ ] Verify an unparseable optional face falls back without failing when a
  valid CSS/bundled face exists.
- [ ] Verify an unembeddable selected face falls back and re-layouts before a
  claiming PDF is written.
- [ ] Verify conversion fails only when no fallback can satisfy the text and
  profile; no claiming PDF is written in that terminal case.
- [ ] Verify a skipped optional face falls through only when the CSS fallback
  contract permits it, with an operator warning.
- [ ] Run the repository's veraPDF wrapper for A-3/UA-1 and A-4/UA-2 outputs
  when the validator is available; record a documented skip otherwise.

## 5.3 No-regression gates

- [ ] Existing bundled font tests remain green.
- [ ] Existing 1.7 and 2.0 font-cache invariants remain green.
- [ ] Existing compliance fixtures remain byte/structure-valid where their
  contract requires it.
- [ ] No compliance profile is weakened to accommodate a font.

## 5.4 Closure gates

- [ ] Every selected face in the matrix has `FontFile2` and `ToUnicode`.
- [ ] Both claiming profiles pass the available compliance validator.
- [ ] The plan records exact commands, versions, output paths, and results.
