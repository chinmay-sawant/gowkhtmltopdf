# Phase 32 - Content and Validation

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** complete — validated 2026-08-18
> **Estimated effort:** 3–5 days
> **Depends on:** Phase 31 type freeze
> **Unblocks:** Phases 33–34

---

## Overview

Make `Content` an exactly-one source kind and implement `Document.Validate` /
`ImageDocument.Validate` so illegal trees fail before the engine runs.
Public callers must not use `inline:` / `data:` string prefixes or
`GuessURL` at the API boundary.

## Executive Summary

| Concern | Today | Target |
|---------|-------|--------|
| In-memory HTML | `SetBody` / `ConvertHTML` | `Content{HTML, Base}` via `HTML(...)` |
| Local file | `SetPage(path)` + ACL pair | `Content{File}` + `AllowLocalFiles` |
| Remote URL | `SetPage(url)` | `Content{URL}` |
| Empty / TOC-only | `ErrNoRenderablePDFObjects` | Same semantics on `Document.Validate` |

---

## Phase 32 checklist

### 32.1 Content invariants

- [x] Exactly one of `HTML`, `File`, `URL` may be set; zero or two+ → typed error (`ErrInvalidContent` or equivalent)
- [x] `Base` allowed only with `HTML`; ignored or rejected with `File`/`URL` (pick one policy and test it)
- [x] Empty `HTML` (len 0) → `ErrEmptyHTML` (or `ErrEmptyContent` that `errors.Is` to `ErrEmptyHTML`)
- [x] Empty `File` / `URL` strings do not count as a source
- [x] Helpers `HTML` / `File` / `URL` copy HTML bytes so caller mutation cannot change the stored Content

### 32.2 Document.Validate

- [x] Nil receiver → typed nil-document sentinel
- [x] At least one renderable body: non-empty `Pages` with valid Content, or `Cover` with valid Content
- [x] `TOC != nil` alone is not renderable (reject TOC-only)
- [x] Invalid nested Page Content fails Validate with a clear path in the error (`pages[i]`, `cover`)
- [x] Negative `Copies` fails with `ErrInvalidPDFCopies`; zero means “use the engine default” and positive values are preserved.

### 32.3 ImageDocument.Validate

- [x] Nil receiver → typed sentinel
- [x] Exactly one valid `Source`
- [x] Reject TOC-like / multi-page concepts (image mode is one document)

### 32.4 Tests

- [x] Table test: Content combinations (none / HTML / File / URL / HTML+File / …)
- [x] Table test: Document TOC-only, empty Pages, Cover-only, Pages-only
- [x] Test helpers copy HTML bytes
- [x] Test nil Document / ImageDocument Validate

### 32.5 Closure gates

- [x] `make lint` → changed-surface lint passes; repository-wide result recorded in Phase 38
- [x] `make test` → passed with GOCACHE=/tmp/gowkhtmltopdf-go-cache
- [x] Parent Phase 32 row checked
- [x] Next: Phase 33

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 31 names | Phase 34 mappers |
| Existing sentinels in `api.go` / `errs` | Stable `errors.Is` |

---

## Out of scope

- Network fetch behavior (unchanged loader)
- CLI source flags (Phase 36)
- Deleting `SetBody` (Phase 35)

## Validation record (2026-08-18)

- `document_validate.go` enforces exact-one source, HTML-only Base, empty-source errors, nested path errors, renderability, image format, and typed nil receivers.
- `document_test.go` covers source combinations, TOC-only / Cover-only / Pages-only cases, sentinel matching, and byte ownership. `GOCACHE=/tmp/gowkhtmltopdf-go-cache make test` passed.
