# Phase 2 — Document preparation, resource context, and outline ordering

> **Parent:** [`../architecture-review-2026-08-07-followup.md`](../architecture-review-2026-08-07-followup.md)
> **Status:** complete
> **Depends on:** Phase 1

## Goal

Concentrate source loading, base URL, load policy, media, stylesheets, font
faces, and resource fetch behavior behind reusable preparation/context seams.
Keep outline ordering explicit and page-local fields intact.

## Checklist

- [x] **PREP-01** — `convert.PrepareDocument` is the shared load/parse/sheet/
  simplify/font-face path consumed by PDF and image mode. `CollectSheets` and
  `MergeFontFaces` remain internal seams, and both mode tests pass.
- [x] **PREP-02** — `ResourceContext` binds loader, resolved base, and
  `LoadPage` policy. PDF body resources, image resources, and HTML
  header/footer child documents use the bound loader/policy; base/ACL/cache and
  header/image tests pass.
- [x] **PREP-03** — primary and subresource `data:` bodies plus inline HTML are
  bounded by `MaxBodySize`; empty inline bases reject unresolved relative
  references instead of falling through to local access. Focused over-limit,
  exact-limit, absolute-data, and empty-base tests pass.
- [x] **OUT-01** — `outline.PageOf`, `BuildTreeBy`, `SectionOfBy`, and
  `DumpOutlineXMLBy` provide explicit document-page ordering. PDF TOC,
  outline, and header/footer consumers now use `outline.DocumentPage` without
  copying `DocPage` into `Heading.Page`; outline and multi-object tests pass.

## Required gate

- [x] Final Phase 2 gate: `make lint`, `make test`,
  `go test ./internal/load ./internal/outline`, and affected PDF/image tests
  passed on 2026-08-07. No intended golden output changes were recorded.
