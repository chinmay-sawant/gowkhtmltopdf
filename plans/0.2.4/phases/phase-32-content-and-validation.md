# Phase 32 - Content and Validation

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** not started
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

- [ ] Exactly one of `HTML`, `File`, `URL` may be set; zero or two+ → typed error (`ErrInvalidContent` or equivalent)
- [ ] `Base` allowed only with `HTML`; ignored or rejected with `File`/`URL` (pick one policy and test it)
- [ ] Empty `HTML` (len 0) → `ErrEmptyHTML` (or `ErrEmptyContent` that `errors.Is` to `ErrEmptyHTML`)
- [ ] Empty `File` / `URL` strings do not count as a source
- [ ] Helpers `HTML` / `File` / `URL` copy HTML bytes so caller mutation cannot change the stored Content

### 32.2 Document.Validate

- [ ] Nil receiver → typed nil-document sentinel
- [ ] At least one renderable body: non-empty `Pages` with valid Content, or `Cover` with valid Content
- [ ] `TOC != nil` alone is not renderable (reject TOC-only)
- [ ] Invalid nested Page Content fails Validate with a clear path in the error (`pages[i]`, `cover`)
- [ ] `Copies < 1` when explicitly set fails with `ErrInvalidPDFCopies` (default 0 means “use engine default” — document in Phase 33 if Copies zero means default 1)

### 32.3 ImageDocument.Validate

- [ ] Nil receiver → typed sentinel
- [ ] Exactly one valid `Source`
- [ ] Reject TOC-like / multi-page concepts (image mode is one document)

### 32.4 Tests

- [ ] Table test: Content combinations (none / HTML / File / URL / HTML+File / …)
- [ ] Table test: Document TOC-only, empty Pages, Cover-only, Pages-only
- [ ] Test helpers copy HTML bytes
- [ ] Test nil Document / ImageDocument Validate

### 32.5 Closure gates

- [ ] `make lint` → (record outcome)
- [ ] `make test` → (record outcome)
- [ ] Parent Phase 32 row checked
- [ ] Next: Phase 33

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
