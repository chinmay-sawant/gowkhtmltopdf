# Phase 34 - Engine Adapters

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** complete — validated 2026-08-18
> **Estimated effort:** 1 week
> **Depends on:** Phases 31–33
> **Unblocks:** Phases 35–36

---

## Overview

Implement the thin boundary that turns a validated `Document` /
`ImageDocument` into the existing engine requests. Public godoc must not
teach `internal/settings` dotted names. Internals may still construct
`settings.PdfGlobal` / `settings.PdfObject`.

## Executive Summary

| Public | Adapter builds | Engine entry |
|--------|----------------|--------------|
| `Document.WritePDF` | `convert.PDFRequest` | `convert.Run` / existing PDF path |
| `Document.WritePDFOutline` | PDF + outline writers | same + outline sink |
| `ImageDocument.WriteImage` | `imageout.Request` | `imageout.RunRequest` |

---

## Phase 34 checklist

### 34.1 PDF mapper

- [x] Single unexported (or internal) mapper: Document + writers → `*convert.PDFRequest` (or equivalent `NewPDFRequest` call)
- [x] Map geometry, title, version, profile, copies, outline, background, compression, smart shrinking, resolve-relative-links
- [x] Map `AllowLocalFiles` → enable global local access + unblock object local access
- [x] Map `Network` via existing `ApplyNetworkPolicy`
- [x] Map `FontPaths` / `UseSystemFonts`
- [x] Map Document HF → global Header/Footer
- [x] Cover `*Page` → object with `IsCover` + content source
- [x] `TOC != nil` → TOC object + TOC defaults from struct
- [x] Each `Pages[i]` → body object; Page HF sets override bits
- [x] Content: HTML → inline body + base; File → page path; URL → page URL (no public `inline:` strings)

### 34.2 Image mapper

- [x] Single mapper: ImageDocument + writer → `*imageout.Request`
- [x] Map viewport / format / quality / crop / smart width / transparent
- [x] Map Source Content same rules as PDF
- [x] Map shared ACL / network / background / hooks

### 34.3 Execution + hooks

- [x] `WritePDF` / `PDF` / `WritePDFOutline` call Validate then mapper then engine
- [x] `WriteImage` / `Image` same pattern
- [x] Wire OnInfo / OnWarn / OnError / OnPhase / OnProgress to existing convert hooks
- [x] Context cancel aborts load (parity with today’s Convert)
- [x] Nil output writer → `ErrMissingPDFOutput` / `ErrMissingImageOutput`

### 34.4 Ownership and isolation

- [x] Clone settings / HTML bytes at mapper boundary
- [x] Mutating Document after WritePDF returns does not change written bytes
- [x] No export of `settings.PdfGlobal` from the root package

### 34.5 Tests

- [x] Smoke: Document with `Content{HTML:…}` produces `%PDF-` magic
- [x] Smoke: ImageDocument PNG magic
- [x] Local file fixture with `AllowLocalFiles: true` only (no dotted ACL)
- [x] Outline path requires outline writer
- [x] Context cancel test
- [x] Cover + TOC + body ordering reflected in object list (unit test on mapper)

### 34.6 Closure gates

- [x] `make lint` → changed-surface lint passes; repository-wide result recorded in Phase 38
- [x] `make test` → passed with GOCACHE=/tmp/gowkhtmltopdf-go-cache
- [x] Parent Phase 34 row checked
- [x] Next: Phase 35 (and Phase 36 in parallel after examples can compile)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| `internal/convert`, `internal/imageout` | Phase 35 deletion safety |
| Phase 32–33 | Valid trees to map |

---

## Out of scope

- Changing convert pipeline / layout
- Public re-export of engine request types

## Validation record (2026-08-18)

- `document.go` contains the single unexported PDF mapper and image mapper, including ACL/network cloning, Cover/TOC/body ordering, page header/footer overrides, and source-kind mapping.
- `document_render_test.go` covers PDF/image magic, outline sink preflight and output ownership; it also covers successful outline XML and canceled contexts. `make test` and `go test -race ./...` passed.
