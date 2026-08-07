# Phase 2 — Document preparation, resource context, and outline ordering

> **Parent:** [`../architecture-review-2026-08-07-followup.md`](../architecture-review-2026-08-07-followup.md)
> **Status:** not started
> **Depends on:** Phase 1

## Goal

Concentrate one document's source, base URL, load policy, media, stylesheets, font
faces, and resource fetch behavior behind a small preparation interface. Make outline
ordering explicit without breaking page-local semantics.

## Checklist

- [ ] **PREP-01** — extract the shared document-preparation implementation used by
  PDF and image mode; keep `CollectSheets` and `MergeFontFaces` as internal seams.
  Proof: both modes use the same preparation path for parse/media/simplify/font
  behavior and their existing fixtures remain stable.
- [ ] **PREP-02** — bind `(loader, base, LoadPage)` into one document resource context
  used by CSS, images, fonts, and HTML headers/footers. Proof: base-resolution,
  ACL, cache, and error-prefix tests pass in both modes.
- [ ] **PREP-03** — audit `data:` body limits and empty-inline-base relative-reference
  policy; if behavior changes, classify it as a security/correctness change and add
  focused regression tests. Do not close from plan prose alone.
- [ ] **OUT-01** — replace the `Heading.Page`/`DocPage` view convention with an
  explicit ordering value or function. Proof: multi-object page-zero, TOC, PDF
  outline, and header section tests.

## Required gate

- [ ] Run `make lint` and `make test`; run affected PDF/image/golden tests separately
  and record any intended output changes.
