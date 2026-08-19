# Font Phase 7 — Documentation and Operator Contract

> **Status:** complete — validated 2026-08-19
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phases 1–6
> **Unblocks:** Phase 8

## Overview

Make the implementation discoverable and prevent users from interpreting a
fallback as an exact Georgia/Arial/Times implementation.

## 7.1 Documentation updates

- [x] Update `documentation/fonts.md` with the final resolution order,
  directory semantics for `--font-path`, exact-family matching, and generic
  fallback behavior.
- [x] Document that `--font-path` is not a `--font` file argument and does not
  import Fontconfig aliases.
- [x] Document `@font-face` source restrictions, weight/style behavior, ACL,
  network policy, and unsupported formats. State clearly that HTTPS
  TTF/OTF/WOFF1 already works; WOFF2/`data:`/EOT remain skipped.
- [x] Document the distinction between an exact supplied font and a bundled
  compatibility fallback.
- [x] Update `documentation/compatibility-matrix.md` and `documentation/fidelity.md`
  with the tested named-family and style limitations. Fix the matrix row that
  claims only CSS generics expand to Liberation while code still maps
  Georgia/Arial/Times/… (or mark the row as the post-Phase-1/2 contract).
- [x] Correct `--header-font-name` / `--footer-font-name` matrix wording:
  `resolveHFFont` already consults the registry before Liberation.
- [x] Update `documentation/deferred.md` so the WOFF2 row does not imply that
  remote HTTPS TTF/OTF/WOFF1 is unsupported.
- [x] Update CLI and library API docs for `FontPaths` and `UseSystemFonts`.
- [x] Add compliance notes: selected fonts must remain embeddable and carry
  `ToUnicode` under claiming profiles. Keep fidelity language honest
  (`--pdf-version` is not a PDF/A claim; profiles are opt-in).
- [x] Update architecture docs if the resolver seam or registry ownership
  changes.

## 7.1b Knowledge-base hygiene

Per `AGENTS.md` / `knowledge-base/SCHEMA.md`, wiki pages must follow
`documentation/` and `plans/` when they drift.

- [x] Retarget `knowledge-base/wiki/syntheses/fonts-and-typography.md` and
  `knowledge-base/wiki/syntheses/roadmap.md` from removed `temps/font/` to
  `plans/0.2.5/font/` (done 2026-08-19 during plan reconciliation; not
  implementation evidence for phases 1–6).
- [x] Update `knowledge-base/wiki/summaries/fonts.md` if the shipped contract
  changes.
- [x] Append `knowledge-base/wiki/log.md` for the path retarget; keep
  `wiki/index.md` current when summary text changes.
- [x] Drop or stub dead wiki links to missing `concepts/fonts-unicode.md` if
  still referenced.

## 7.2 Diagnostics and examples

- [x] Add an operator-facing example using a directory containing a supported
  font whose internal family name is visible in CSS.
- [x] Add an example using document `@font-face` with explicit regular and
  bold/italic sources if supported by the final contract.
- [x] Ensure warnings identify the source category and reason for a skipped
  font without leaking private document data.
- [x] Ensure errors distinguish invalid path, unsupported format, parse failure,
  missing family, and compliance embedding failure.
- [x] Add a short troubleshooting section for Chrome/WeasyPrint differences:
  same CSS family name does not imply same installed font.

## 7.3 Closure gates

- [x] CLI help, public API docs, font docs, compatibility matrix, knowledge-base
  font pages, and code agree.
- [x] Examples use supported paths and produce a selected-family proof.
- [x] No document claims exact Georgia when only Liberation Serif fallback was
  selected.
- [x] No remaining live cites of `temps/font/` for this track (wiki path
  retarget done 2026-08-19; re-check at closure).

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
