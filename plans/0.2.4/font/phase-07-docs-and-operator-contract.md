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
  network policy, and unsupported formats.
- [ ] Document the distinction between an exact supplied font and a bundled
  compatibility fallback.
- [ ] Update `documentation/compatibility-matrix.md` and `documentation/fidelity.md`
  with the tested named-family and style limitations.
- [ ] Update CLI and library API docs for `FontPaths` and `UseSystemFonts`.
- [ ] Add compliance notes: selected fonts must remain embeddable and carry
  `ToUnicode` under claiming profiles.
- [ ] Update architecture docs if the resolver seam or registry ownership
  changes.

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

- [ ] CLI help, public API docs, font docs, compatibility matrix, and code agree.
- [ ] Examples use supported paths and produce a selected-family proof.
- [ ] No document claims exact Georgia when only Liberation Serif fallback was
  selected.
