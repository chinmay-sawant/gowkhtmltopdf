# Font Phase 4 — Format, Fixture, and Asset Policy

> **Status:** complete — validated 2026-08-19
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phase 3
> **Unblocks:** Phase 5

## Overview

Define which fonts can safely enter the existing pure-Go parser and PDF
writer. A visually attractive font is not automatically a usable compliance
font. This phase prevents a new asset from bypassing parser, subsetting,
licensing, or embedding constraints.

**Constraint:** no new direct Go modules without an explicit allowlist
amendment (`AGENTS.md`, `TestDirectModuleAllowlist`). WOFF2/Brotli therefore
remains rejected here even though pending
[`09-remote-webfonts.md`](../../0.2.0/phases/pending-phase-items/09-remote-webfonts.md)
may pursue it later.

## 4.1 Supported-format matrix

- [x] Lock and document the behavior for:
  - TTF with TrueType outlines;
  - OTF with TrueType outlines;
  - WOFF1 converted to a supported TrueType representation (already shipped;
    keep caps in `woff.go`);
  - CFF/`OTTO` OpenType;
  - WOFF2 (skip + diagnostic; decode is a separate epic);
  - EOT;
  - `data:` sources;
  - variable TrueType fonts.
- [x] Decide whether variable fonts are supported as a stable default
  instance, require a static face, or are rejected with a clear message
  (`fvar` is unused today; CI Noto KR subset is treated as normal SFNT).
- [x] Add parser tests for every accepted and rejected category.
- [x] Confirm malformed table directories, missing cmap/metrics, unsupported
  outlines, and truncated streams fail safely.
- [x] Confirm every accepted test face can be reparsed after subsetting.
- [x] Add discovery-path diagnostics for silent `ScanFontDirs` / `ParseTTF`
  skips (count + reason), without dumping font bytes.

## 4.2 Test-font catalog

- [x] Create a small legally redistributable test-font manifest under
  `testdata/fonts` or a focused font fixture directory.
- [x] Include regular, bold, italic, bold-italic, Unicode, composite-glyph,
  and duplicate-family cases.
- [x] Do not add Microsoft Georgia or another proprietary face without a
  written redistribution decision and license record.
- [x] If a Georgia-compatible open font is evaluated, record its license,
  static/variable status, name-table families, and visual rationale before
  bundling it.
- [x] Keep test fonts minimal and purpose-built so CI does not acquire a
  large font corpus.

## 4.3 Name and style metadata

- [x] Test family names, PostScript names, typographic names, and aliases with
  spaces and punctuation.
- [x] Test fonts whose internal family name differs from the file name.
- [x] Test missing style faces and nearest-face selection.
- [x] Ensure PDF name-token sanitization remains valid and unique.
- [x] Ensure same display names with different bytes do not share subsets or
  cache entries.

## 4.4 Closure gates

- [x] Supported-format matrix is documented in `documentation/fonts.md`.
- [x] Test-font licenses and provenance are recorded.
- [x] No candidate asset is promoted to bundled production faces without the
  Phase 5 compliance gates.

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
