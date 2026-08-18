# Phase 34 - Engine Adapters

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** not started
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

- [ ] Single unexported (or internal) mapper: Document + writers → `*convert.PDFRequest` (or equivalent `NewPDFRequest` call)
- [ ] Map geometry, title, version, profile, copies, outline, background, compression, smart shrinking, resolve-relative-links
- [ ] Map `AllowLocalFiles` → enable global local access + unblock object local access
- [ ] Map `Network` via existing `ApplyNetworkPolicy`
- [ ] Map `FontPaths` / `UseSystemFonts`
- [ ] Map Document HF → global Header/Footer
- [ ] Cover `*Page` → object with `IsCover` + content source
- [ ] `TOC != nil` → TOC object + TOC defaults from struct
- [ ] Each `Pages[i]` → body object; Page HF sets override bits
- [ ] Content: HTML → inline body + base; File → page path; URL → page URL (no public `inline:` strings)

### 34.2 Image mapper

- [ ] Single mapper: ImageDocument + writer → `*imageout.Request`
- [ ] Map viewport / format / quality / crop / smart width / transparent
- [ ] Map Source Content same rules as PDF
- [ ] Map shared ACL / network / background / hooks

### 34.3 Execution + hooks

- [ ] `WritePDF` / `PDF` / `WritePDFOutline` call Validate then mapper then engine
- [ ] `WriteImage` / `Image` same pattern
- [ ] Wire OnInfo / OnWarn / OnError / OnPhase / OnProgress to existing convert hooks
- [ ] Context cancel aborts load (parity with today’s Convert)
- [ ] Nil output writer → `ErrMissingPDFOutput` / `ErrMissingImageOutput`

### 34.4 Ownership and isolation

- [ ] Clone settings / HTML bytes at mapper boundary
- [ ] Mutating Document after WritePDF returns does not change written bytes
- [ ] No export of `settings.PdfGlobal` from the root package

### 34.5 Tests

- [ ] Smoke: Document with `Content{HTML:…}` produces `%PDF-` magic
- [ ] Smoke: ImageDocument PNG magic
- [ ] Local file fixture with `AllowLocalFiles: true` only (no dotted ACL)
- [ ] Outline path requires outline writer
- [ ] Context cancel test
- [ ] Cover + TOC + body ordering reflected in object list (unit test on mapper)

### 34.6 Closure gates

- [ ] `make lint` → (record outcome)
- [ ] `make test` → (record outcome)
- [ ] Parent Phase 34 row checked
- [ ] Next: Phase 35 (and Phase 36 in parallel after examples can compile)

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
