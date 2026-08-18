# Phase 31 - Document Model Contract

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** not started
> **Estimated effort:** 3–5 days
> **Depends on:** v0.2.3 tree; v0.2.1 Phase 24 preferred-path policy (superseded here)
> **Unblocks:** Phases 32–38

---

## Overview

Freeze the public Document / ImageDocument type names, field meanings, nil
policies, and entry points before writing mappers or deleting the old API.
This phase is primarily a contract + initial type stubs; it must not delete
`Converter` yet.

## Executive Summary

| Surface | Today | Target |
|---------|-------|--------|
| Preferred embedder API | `PDFRequest` + dotted/`PdfGlobalOptions` settings | `Document.WritePDF` / `Document.PDF` |
| Image API | `ImageRequest` / `ImageConverter` + dotted keys | `ImageDocument.WriteImage` / `ImageDocument.Image` |
| Content | `SetPage` string / `SetBody` bytes | `Content{HTML\|File\|URL}` |
| Bool defaults | Implicit via settings defaults | `*bool` = unset → engine default |

---

## Phase 31 checklist

### 31.1 Freeze type inventory

- [ ] Lock exported names: `Content`, `Margin`, `HeaderFooter`, `Page`, `TOC`, `Crop`, `Document`, `ImageDocument`
- [ ] Lock methods: `Document.Validate`, `WritePDF`, `WritePDFOutline`, `PDF`; `ImageDocument.Validate`, `WriteImage`, `Image`
- [ ] Lock helpers: `HTML`, `File`, `URL`, `NewDocument` (exact signatures recorded in this phase file when implemented)
- [ ] Record document order: Cover → TOC → Pages

### 31.2 Nil / pointer / ownership policy

- [ ] Write the nil / error / panic table into the phase notes (and later `library-api.md` in Phase 37): panic only for programmer-broken nil fluent helpers if any remain; invalid user values return `error` from `Validate` / `Write*`
- [ ] `*bool` fields mean unset → engine default; plain `bool` means explicit value
- [ ] `Header` / `Footer` nil on `Page` means inherit `Document` HF
- [ ] `WritePDF` / `WriteImage` clone at the ownership boundary (same guarantee as today’s `RunPDF`)
- [ ] Nil `*Document` / `*ImageDocument` receiver returns a typed sentinel (reuse or add `ErrNilDocument` / `ErrNilImageDocument`)

### 31.3 Preferred entry points

- [ ] Name preferred PDF path: `Document.WritePDF` (writer-first) and `Document.PDF` (bytes)
- [ ] Name preferred image path: `ImageDocument.WriteImage` / `ImageDocument.Image`
- [ ] Outline dump only via `WritePDFOutline` (no silent attach to PDF stream)
- [ ] Hooks (`OnInfo` / `OnWarn` / `OnError` / `OnPhase` / `OnProgress`) live on Document / ImageDocument fields

### 31.4 Hard non-goals (written into godoc / ledger)

- [ ] No public dotted settings keys
- [ ] No `compat` subpackage in 0.2.4
- [ ] No C ABI, no JavaScript
- [ ] `LibraryVersion` remains the wkhtml settings-surface id; project release stays in `VERSION`

### 31.5 Stub without hard break

- [ ] Add types (and no-op or `ErrNotImplemented`-free Validate stubs only if needed for compile) without removing `Converter` yet — prefer landing real Validate in Phase 32
- [ ] Do not rewrite examples in this phase
- [ ] Inventory current exports that Phase 35 will delete (list in closure notes)

### 31.6 Closure gates

- [ ] `make lint` → (record outcome)
- [ ] `make test` → (record outcome)
- [ ] Parent Phase 31 row checked
- [ ] Next: Phase 32

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Current `api.go` / settings structs | Name freeze for 32–34 |
| Phase 24 nil/error policy lessons | Consistent receiver policy |

---

## Out of scope

- Deleting `Converter` / `PDFRequest` (Phase 35)
- CLI flag changes (Phase 36)
- Mapping every settings key (Phase 33)
