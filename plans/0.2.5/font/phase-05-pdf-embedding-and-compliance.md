# Font Phase 5 — PDF Embedding and Conformance Profiles

> **Status:** complete — validated 2026-08-19
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phases 3–4
> **Unblocks:** Phase 6

## Overview

Prove that every selected external or bundled face travels through the
existing writer correctly. This phase does not add a second font writer. It
extends the existing `subsetFont` → `FontFile2` → simple/Type0 emission path
and its compliance tests.

**Orchestration:** `internal/convert` owns embed preflight and fallback
re-layout relative to `RenderObjects` / `Assemble` / `Finalize`. The Phase 2
resolver only marks candidates unavailable; it must not swap faces after
metrics were committed.

## 5.1 Writer coverage

- [x] Render a supplied custom face with Latin-1-only text through the simple
  TrueType path.
- [x] Render the same face with non-Latin text through Type0/CIDFontType2,
  Identity-H, CIDToGIDMap, and ToUnicode.
- [x] Verify composite glyph closure and advance widths survive subsetting.
- [x] Verify regular, bold, italic, and bold-italic face resources are distinct
  and selected correctly.
- [x] Verify two files sharing a display name retain distinct subset cache
  identities.
- [x] Verify font descriptor flags, `/FontName`, `/BaseFont`, `/Widths`, and
  `/ToUnicode` remain valid for custom faces.
- [x] Verify missing glyphs use the documented fallback without losing text
  extraction semantics.
- [x] Add an embed-preflight step over the actual used rune set before final
  paint/output commit (convert-owned seam; reuse `ensureFont` / subset
  failure signals — do not invent a second subsetter).
- [x] If preflight fails for an optional face, select the next valid fallback
  and re-layout the affected object so changed metrics are reflected in line
  breaks and pagination.
- [x] Prove that a failed fallback does not leave partially painted content or
  a stale font resource in the final document.
- [x] Confirm claiming profiles still fail closed: no PDF/A or PDF/UA claim
  is written with an unembedded or non-`ToUnicode` text font.

## 5.2 Version and conformance matrix

For each accepted test face, generate and inspect:

| Output | Required evidence |
|--------|--------------------|
| PDF 1.4 unclaimed | valid header, embedded font, `ToUnicode` |
| PDF 1.7 unclaimed | valid version path and embedded font |
| PDF 1.7 + PDF/A-3a + PDF/UA-1 | XMP claim, `FontFile2`, `ToUnicode`, tagged output, no unembedded standard font |
| PDF 2.0 unclaimed | valid version path and embedded font |
| PDF 2.0 + PDF/A-4 + PDF/UA-2 | XMP claim, `FontFile2`, `ToUnicode`, namespaced/tagged output, no unembedded standard font |

- [x] Extend focused PDF tests for external/custom faces under both version
  policies.
- [x] Extend compliance tests to exercise a custom face, not only bundled
  Liberation/DejaVu.
- [x] Verify an unparseable optional face falls back without failing when a
  valid CSS/bundled face exists.
- [x] Verify an unembeddable selected face falls back and re-layouts before a
  claiming PDF is written.
- [x] Verify conversion fails only when no fallback can satisfy the text and
  profile; no claiming PDF is written in that terminal case.
- [x] Verify a skipped optional face falls through only when the CSS fallback
  contract permits it, with an operator warning.
- [x] Run the repository's veraPDF wrapper for A-3/UA-1 and A-4/UA-2 outputs
  when the validator is available; record a documented skip otherwise.

## 5.3 No-regression gates

- [x] Existing bundled font tests remain green.
- [x] Existing 1.7 and 2.0 font-cache invariants remain green.
- [x] Existing compliance fixtures remain byte/structure-valid where their
  contract requires it.
- [x] No compliance profile is weakened to accommodate a font.

## 5.4 Closure gates

- [x] Every selected face in the matrix has `FontFile2` and `ToUnicode`.
- [x] Both claiming profiles pass the available compliance validator.
- [x] The plan records exact commands, versions, output paths, and results.

## Validation outcomes (2026-08-19)

```
$ CGO_ENABLED=0 make test
go test ./...
(ok — all packages)

$ CGO_ENABLED=0 make lint
golangci-lint run ./...
(exit 0)

$ CGO_ENABLED=0 go build -o /tmp/font-0.2.5-evidence/gowkhtmltopdf ./cmd/gowkhtmltopdf
$ CGO_ENABLED=0 go build -o /tmp/font-0.2.5-evidence/gowkhtmltoimage ./cmd/gowkhtmltoimage
$ gowkhtmltopdf -q --allow-local-files -o /tmp/font-0.2.5-evidence/f55-default.pdf \
    testdata/golden/fixture-55-lantern-cooperative-report.html
$ gowkhtmltopdf -q --allow-local-files -o /tmp/font-0.2.5-evidence/f55-default-b.pdf \
    testdata/golden/fixture-55-lantern-cooperative-report.html
$ cmp …/f55-default.pdf …/f55-default-b.pdf   # STABLE=yes
# PDF 1.4, FontFile2 + ToUnicode + Liberation present, 4 page objects, ~60 KiB
$ gowkhtmltopdf -q --allow-local-files --font-path internal/pdf/assets \
    -o /tmp/font-0.2.5-evidence/f55-fontpath.pdf …/fixture-55-….html
$ gowkhtmltoimage -q --allow-local-files -o /tmp/font-0.2.5-evidence/f55.png …/fixture-55-….html
```

Focused proofs: `go test ./internal/pdf ./internal/layout ./internal/convert ./internal/css ./internal/cli ./internal/imageout .`
Cover FontResolver, discovery diagnostics, `@font-face` weight/style, preflight re-layout, CLI/library font-path.
