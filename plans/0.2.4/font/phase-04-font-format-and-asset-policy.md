# Font Phase 4 — Format, Fixture, and Asset Policy

> **Status:** planned
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phase 3
> **Unblocks:** Phase 5

## Overview

Define which fonts can safely enter the existing pure-Go parser and PDF
writer. A visually attractive font is not automatically a usable compliance
font. This phase prevents a new asset from bypassing parser, subsetting,
licensing, or embedding constraints.

## 4.1 Supported-format matrix

- [ ] Lock and document the behavior for:
  - TTF with TrueType outlines;
  - OTF with TrueType outlines;
  - WOFF1 converted to a supported TrueType representation;
  - CFF/`OTTO` OpenType;
  - WOFF2;
  - EOT;
  - `data:` sources;
  - variable TrueType fonts.
- [ ] Decide whether variable fonts are supported as a stable default
  instance, require a static face, or are rejected with a clear message.
- [ ] Add parser tests for every accepted and rejected category.
- [ ] Confirm malformed table directories, missing cmap/metrics, unsupported
  outlines, and truncated streams fail safely.
- [ ] Confirm every accepted test face can be reparsed after subsetting.

## 4.2 Test-font catalog

- [ ] Create a small legally redistributable test-font manifest under
  `testdata/fonts` or a focused font fixture directory.
- [ ] Include regular, bold, italic, bold-italic, Unicode, composite-glyph,
  and duplicate-family cases.
- [ ] Do not add Microsoft Georgia or another proprietary face without a
  written redistribution decision and license record.
- [ ] If a Georgia-compatible open font is evaluated, record its license,
  static/variable status, name-table families, and visual rationale before
  bundling it.
- [ ] Keep test fonts minimal and purpose-built so CI does not acquire a
  large font corpus.

## 4.3 Name and style metadata

- [ ] Test family names, PostScript names, typographic names, and aliases with
  spaces and punctuation.
- [ ] Test fonts whose internal family name differs from the file name.
- [ ] Test missing style faces and nearest-face selection.
- [ ] Ensure PDF name-token sanitization remains valid and unique.
- [ ] Ensure same display names with different bytes do not share subsets or
  cache entries.

## 4.4 Closure gates

- [ ] Supported-format matrix is documented in `documentation/fonts.md`.
- [ ] Test-font licenses and provenance are recorded.
- [ ] No candidate asset is promoted to bundled production faces without the
  Phase 5 compliance gates.
