# Font Phase 3 — Discovery, CLI, Library, and `@font-face`

> **Status:** planned
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

- [ ] Document and validate that `--font-path` accepts directories, not a
  single font file path.
- [ ] Decide whether an explicitly supplied file path should be rejected with
  a clear error or accepted as a convenience; do not silently treat it as an
  empty directory.
- [ ] Preserve repeatable paths and deterministic scan order.
- [ ] Preserve the current depth limit and supported extension policy, or
  amend it explicitly with tests.
- [ ] Report scanned paths, loaded-face count, skipped-file count, and useful
  skip reasons without exposing font file contents.
- [ ] Treat a skipped or unparseable optional face as a candidate miss and
  continue through the author CSS stack or bundled Liberation fallback.
- [ ] Keep `--use-system-fonts` opt-in and independent from Fontconfig alias
  rules.
- [ ] Verify `Document.FontPaths` and `UseSystemFonts` reach the exact same
  resolver as the CLI.
- [ ] Verify PDF and image output use the same registry and family decisions.

## 3.2 CSS `@font-face`

- [ ] Extend `css.FontFace` beyond `Family` + `Src` so supported `font-weight`
  and `font-style` descriptors are retained through `parseFontFace` into
  `MergeFontFaces`.
- [ ] Register each face under the author family plus its declared style
  metadata without losing fingerprint identity.
- [ ] Define source-order and duplicate-face selection for multiple
  `@font-face` rules with the same family.
- [ ] Preserve ACL and network policy for local and remote font sources
  (HTTPS TTF/OTF/WOFF1 remains in scope as already-shipped behavior).
- [ ] Keep explicit warnings for WOFF2, EOT, `data:`, malformed, and
  unparseable sources. WOFF2 decode stays out of this track (see pending
  `09-remote-webfonts.md` + allowlist amendment).
- [ ] Ensure those warnings do not become conversion errors when a valid
  fallback face exists.
- [ ] Verify body, header, footer, cover, TOC, PDF, and image paths share the
  document font registry where the existing architecture promises sharing
  (including nested HTML HF `@font-face` via the shared merge path).
- [ ] Add tests for regular/bold/italic/bold-italic `@font-face` rules and
  fallback when one face is missing.

## 3.3 Public and CLI tests

- [ ] Extend CLI tests beyond field assignment to prove the selected font is
  visible in the emitted PDF.
- [ ] Extend `document_test.go` to prove the public `Document` path honors
  `FontPaths` and `UseSystemFonts`.
- [ ] Add an explicit regression test showing that supplying a directory with
  a family named Georgia selects that family for `font-family: Georgia`.
- [ ] Add a regression test showing that supplying only Gelasio does not
  silently rename it to Georgia.
- [ ] Add a regression test showing an invalid `@font-face` falls back to
  `sans-serif`/Liberation Sans and still produces a PDF.

## 3.4 Closure gates

- [ ] CLI and library paths produce the same selected-family evidence.
- [ ] Unsupported inputs produce an actionable diagnostic and do not create a
  claiming PDF.
- [ ] `go test ./internal/cli ./internal/convert ./internal/layout .` passes.
