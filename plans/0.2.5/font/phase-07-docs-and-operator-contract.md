# Font Phase 7 — Documentation and Operator Contract

> **Status:** planned
> **Parent:** [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
> **Depends on:** Phases 1–6
> **Unblocks:** Phase 8

## Overview

Make the implementation discoverable and prevent users from interpreting a
fallback as an exact Georgia/Arial/Times implementation.

## 7.1 Documentation updates

- [ ] Update `documentation/fonts.md` with the final resolution order,
  directory semantics for `--font-path`, exact-family matching, and generic
  fallback behavior.
- [ ] Document that `--font-path` is not a `--font` file argument and does not
  import Fontconfig aliases.
- [ ] Document `@font-face` source restrictions, weight/style behavior, ACL,
  network policy, and unsupported formats. State clearly that HTTPS
  TTF/OTF/WOFF1 already works; WOFF2/`data:`/EOT remain skipped.
- [ ] Document the distinction between an exact supplied font and a bundled
  compatibility fallback.
- [ ] Update `documentation/compatibility-matrix.md` and `documentation/fidelity.md`
  with the tested named-family and style limitations. Fix the matrix row that
  claims only CSS generics expand to Liberation while code still maps
  Georgia/Arial/Times/… (or mark the row as the post-Phase-1/2 contract).
- [ ] Correct `--header-font-name` / `--footer-font-name` matrix wording:
  `resolveHFFont` already consults the registry before Liberation.
- [ ] Update `documentation/deferred.md` so the WOFF2 row does not imply that
  remote HTTPS TTF/OTF/WOFF1 is unsupported.
- [ ] Update CLI and library API docs for `FontPaths` and `UseSystemFonts`.
- [ ] Add compliance notes: selected fonts must remain embeddable and carry
  `ToUnicode` under claiming profiles. Keep fidelity language honest
  (`--pdf-version` is not a PDF/A claim; profiles are opt-in).
- [ ] Update architecture docs if the resolver seam or registry ownership
  changes.

## 7.1b Knowledge-base hygiene

Per `AGENTS.md` / `knowledge-base/SCHEMA.md`, wiki pages must follow
`documentation/` and `plans/` when they drift.

- [x] Retarget `knowledge-base/wiki/syntheses/fonts-and-typography.md` and
  `knowledge-base/wiki/syntheses/roadmap.md` from removed `temps/font/` to
  `plans/0.2.5/font/` (done 2026-08-19 during plan reconciliation; not
  implementation evidence for phases 1–6).
- [ ] Update `knowledge-base/wiki/summaries/fonts.md` if the shipped contract
  changes.
- [x] Append `knowledge-base/wiki/log.md` for the path retarget; keep
  `wiki/index.md` current when summary text changes.
- [ ] Drop or stub dead wiki links to missing `concepts/fonts-unicode.md` if
  still referenced.

## 7.2 Diagnostics and examples

- [ ] Add an operator-facing example using a directory containing a supported
  font whose internal family name is visible in CSS.
- [ ] Add an example using document `@font-face` with explicit regular and
  bold/italic sources if supported by the final contract.
- [ ] Ensure warnings identify the source category and reason for a skipped
  font without leaking private document data.
- [ ] Ensure errors distinguish invalid path, unsupported format, parse failure,
  missing family, and compliance embedding failure.
- [ ] Add a short troubleshooting section for Chrome/WeasyPrint differences:
  same CSS family name does not imply same installed font.

## 7.3 Closure gates

- [ ] CLI help, public API docs, font docs, compatibility matrix, knowledge-base
  font pages, and code agree.
- [ ] Examples use supported paths and produce a selected-family proof.
- [ ] No document claims exact Georgia when only Liberation Serif fallback was
  selected.
- [x] No remaining live cites of `temps/font/` for this track (wiki path
  retarget done 2026-08-19; re-check at closure).
