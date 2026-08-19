# Font Phase 3 — Discovery, CLI, Library, and `@font-face`

> **Status:** complete — validated 2026-08-19
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phase 2
> **Unblocks:** Phases 4–6

## Overview

Make the existing font inputs behave predictably. The supported flag is
`--font-path` (repeatable directory discovery), not a generic `--font` flag.
The same settings already exist on `Document.FontPaths` and
`Document.UseSystemFonts`; this phase verifies that CLI, library, PDF, and
image paths share one registry contract.

**Already shipped (do not re-plan as missing):** local and `http(s)`
TTF/OTF/WOFF1 `@font-face` fetch via `FetchSub` under the existing ACL,
timeouts, and body caps. Tests such as `TestFontFaceHTTPSFetchAttempted` and
`TestFontFaceWOFFEmbed` lock this. Remaining work here is style metadata,
diagnostics, directory/file semantics, and shared-registry proofs.

## 3.1 `--font-path` and `--use-system-fonts`

- [x] Document and validate that `--font-path` accepts directories, not a
  single font file path.
- [x] Decide whether an explicitly supplied file path should be rejected with
  a clear error or accepted as a convenience; do not silently treat it as an
  empty directory.
- [x] Preserve repeatable paths and deterministic scan order.
- [x] Preserve the current depth limit and supported extension policy, or
  amend it explicitly with tests.
- [x] Report scanned paths, loaded-face count, skipped-file count, and useful
  skip reasons without exposing font file contents.
- [x] Treat a skipped or unparseable optional face as a candidate miss and
  continue through the author CSS stack or bundled Liberation fallback.
- [x] Keep `--use-system-fonts` opt-in and independent from Fontconfig alias
  rules.
- [x] Verify `Document.FontPaths` and `UseSystemFonts` reach the exact same
  resolver as the CLI.
- [x] Verify PDF and image output use the same registry and family decisions.

## 3.2 CSS `@font-face`

- [x] Extend `css.FontFace` beyond `Family` + `Src` so supported `font-weight`
  and `font-style` descriptors are retained through `parseFontFace` into
  `MergeFontFaces`.
- [x] Register each face under the author family plus its declared style
  metadata without losing fingerprint identity.
- [x] Define source-order and duplicate-face selection for multiple
  `@font-face` rules with the same family.
- [x] Preserve ACL and network policy for local and remote font sources
  (HTTPS TTF/OTF/WOFF1 remains in scope as already-shipped behavior).
- [x] Keep explicit warnings for WOFF2, EOT, `data:`, malformed, and
  unparseable sources. WOFF2 decode stays out of this track (see pending
  `09-remote-webfonts.md` + allowlist amendment).
- [x] Ensure those warnings do not become conversion errors when a valid
  fallback face exists.
- [x] Verify body, header, footer, cover, TOC, PDF, and image paths share the
  document font registry where the existing architecture promises sharing
  (including nested HTML HF `@font-face` via the shared merge path).
- [x] Add tests for regular/bold/italic/bold-italic `@font-face` rules and
  fallback when one face is missing.

## 3.3 Public and CLI tests

- [x] Extend CLI tests beyond field assignment to prove the selected font is
  visible in the emitted PDF.
- [x] Extend `document_test.go` to prove the public `Document` path honors
  `FontPaths` and `UseSystemFonts`.
- [x] Add an explicit regression test showing that supplying a directory with
  a family named Georgia selects that family for `font-family: Georgia`.
- [x] Add a regression test showing that supplying only Gelasio does not
  silently rename it to Georgia.
- [x] Add a regression test showing an invalid `@font-face` falls back to
  `sans-serif`/Liberation Sans and still produces a PDF.

## 3.4 Closure gates

- [x] CLI and library paths produce the same selected-family evidence.
- [x] Unsupported inputs produce an actionable diagnostic and do not create a
  claiming PDF.
- [x] `go test ./internal/cli ./internal/convert ./internal/layout .` passes.

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
