# Font Phase 6 — Corpus Rendering and Engine Comparison

> **Status:** planned
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phase 5
> **Unblocks:** Phase 7

## Overview

Validate visible behavior on the real fixture corpus. Visual acceptance is
required for this work because font metrics change wrapping, line height,
pagination, and perceived weight even when the PDF structure is valid.

## 6.1 Gowkhtmltopdf scenarios

- [ ] Render fixture-55 with no font flags and record the deterministic
  bundled fallback.
- [ ] Render fixture-55 with an explicit supported family directory and record
  the selected embedded family.
- [ ] Render fixture-55 through the public `Document.WritePDF` path with the
  same font options.
- [ ] Render fixture-55 through image mode and confirm the same resolver and
  metrics are used.
- [ ] Render a wrapping-sensitive fixture with a deliberately invalid optional
  face and confirm it completes with the bundled Liberation fallback.
- [ ] Render a face that fails subset/embed preflight and confirm fallback
  re-layout, page count, and text geometry are internally consistent.
- [ ] Render fixture-15 and the typography-focused fixtures to detect the
  broader font issues already observed.
- [ ] Render the CSS family/style matrix with page counts and text extraction.
- [ ] Rasterize affected pages and inspect crops, especially fixture-55 page 3
  and wrapping-sensitive headings/body rows.

## 6.2 Controlled external comparison

- [ ] Compare Chrome and WeasyPrint only when all engines receive the same
  explicit font file through `@font-face` or an equivalent controlled input.
- [ ] Keep a separate environment-resolution comparison documenting that
  Chrome on Windows may use actual Georgia while Linux WeasyPrint may use a
  Fontconfig substitute.
- [ ] Record engine versions, operating system, installed-font context, CSS,
  selected PDF font names, page counts, and text geometry.
- [ ] Treat external tools as differential evidence, not compliance oracles
  and not mandatory runtime dependencies.
- [ ] Do not use a browser pixel diff as the only acceptance criterion; combine
  screenshots with extracted text, font resources, and structural checks.

## 6.3 Golden policy

- [ ] Do not overwrite committed fixture PDFs until the selected fallback or
  supplied-font behavior is approved.
- [ ] If output changes intentionally, record the reason, selected font,
  expected pagination changes, and visual review artifact.
- [ ] Keep a dedicated font-resolution fixture for future regressions instead
  of relying only on large showcase PDFs.

## 6.4 Closure gates

- [ ] Fixture-55 page 3 has an accepted font family, weight, width, and wrap.
- [ ] Fixture-15 and the selected corpus have no unexplained font changes.
- [ ] Default output is stable across two runs on the same host.
- [ ] Explicit-font output is stable across two runs with the same font bytes.
- [ ] Comparison artifacts are stored in a temporary or documented evidence
  location until the product decision is made.
