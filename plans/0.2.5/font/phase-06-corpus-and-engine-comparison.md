# Font Phase 6 — Corpus Rendering and Engine Comparison

> **Status:** complete — validated 2026-08-19
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phase 5
> **Unblocks:** Phase 7

## Overview

Validate visible behavior on the real fixture corpus. Visual acceptance is
required for this work because font metrics change wrapping, line height,
pagination, and perceived weight even when the PDF structure is valid.

Existing anchors: `testdata/golden` fixture-55 / fixture-15 / fixture-27
(CJK `--font-path`), `internal/layout/fixture55_font_test.go` (letter-spacing
today, not family identity), convert/imageout `@font-face` tests. Prefer a
dedicated font-resolution fixture over rewriting large showcase PDFs alone.

## 6.1 Gowkhtmltopdf scenarios

- [x] Render fixture-55 with no font flags and record the deterministic
  bundled fallback.
- [x] Render fixture-55 with an explicit supported family directory and record
  the selected embedded family.
- [x] Render fixture-55 through the public `Document.WritePDF` path with the
  same font options.
- [x] Render fixture-55 through image mode and confirm the same resolver and
  metrics are used.
- [x] Render a wrapping-sensitive fixture with a deliberately invalid optional
  face and confirm it completes with the bundled Liberation fallback.
- [x] Render a face that fails subset/embed preflight and confirm fallback
  re-layout, page count, and text geometry are internally consistent.
- [x] Render fixture-15 and the typography-focused fixtures to detect the
  broader font issues already observed.
- [x] Render the CSS family/style matrix with page counts and text extraction.
- [x] Rasterize affected pages and inspect crops, especially fixture-55 page 3
  and wrapping-sensitive headings/body rows.

## 6.2 Controlled external comparison

- [x] Compare Chrome and WeasyPrint only when all engines receive the same
  explicit font file through `@font-face` or an equivalent controlled input.
- [x] Keep a separate environment-resolution comparison documenting that
  Chrome on Windows may use actual Georgia while Linux WeasyPrint may use a
  Fontconfig substitute.
- [x] Record engine versions, operating system, installed-font context, CSS,
  selected PDF font names, page counts, and text geometry.
- [x] Treat external tools as differential evidence, not compliance oracles
  and not mandatory runtime dependencies.
- [x] Do not use a browser pixel diff as the only acceptance criterion; combine
  screenshots with extracted text, font resources, and structural checks.

## 6.3 Golden policy

- [x] Do not overwrite committed fixture PDFs until the selected fallback or
  supplied-font behavior is approved.
- [x] If output changes intentionally, record the reason, selected font,
  expected pagination changes, and visual review artifact.
- [x] Keep a dedicated font-resolution fixture for future regressions instead
  of relying only on large showcase PDFs.

## 6.4 Closure gates

- [x] Fixture-55 page 3 has an accepted font family, weight, width, and wrap.
- [x] Fixture-15 and the selected corpus have no unexplained font changes.
- [x] Default output is stable across two runs on the same host.
- [x] Explicit-font output is stable across two runs with the same font bytes.
- [x] Comparison artifacts are stored in a temporary or documented evidence
  location until the product decision is made.

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
