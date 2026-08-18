# Phase 31 - Document Model Contract

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** complete — validated 2026-08-18
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

- [x] Lock exported names: `Content`, `Margin`, `HeaderFooter`, `Page`, `TOC`, `Crop`, `Document`, `ImageDocument`
- [x] Lock methods: `Document.Validate`, `WritePDF`, `WritePDFOutline`, `PDF`; `ImageDocument.Validate`, `WriteImage`, `Image`
- [x] Lock helpers: `HTML`, `File`, `URL`, `NewDocument` (exact signatures recorded in this phase file when implemented)
- [x] Record document order: Cover → TOC → Pages

### 31.2 Nil / pointer / ownership policy

- [x] Write the nil / error / panic table into the phase notes (and later `library-api.md` in Phase 37): panic only for programmer-broken nil fluent helpers if any remain; invalid user values return `error` from `Validate` / `Write*`
- [x] `*bool` fields mean unset → engine default; plain `bool` means explicit value
- [x] `Header` / `Footer` nil on `Page` means inherit `Document` HF
- [x] `WritePDF` / `WriteImage` clone at the ownership boundary (same guarantee as today’s `RunPDF`)
- [x] Nil `*Document` / `*ImageDocument` receiver returns a typed sentinel (reuse or add `ErrNilDocument` / `ErrNilImageDocument`)

### 31.3 Preferred entry points

- [x] Name preferred PDF path: `Document.WritePDF` (writer-first) and `Document.PDF` (bytes)
- [x] Name preferred image path: `ImageDocument.WriteImage` / `ImageDocument.Image`
- [x] Outline dump only via `WritePDFOutline` (no silent attach to PDF stream)
- [x] Hooks (`OnInfo` / `OnWarn` / `OnError` / `OnPhase` / `OnProgress`) live on Document / ImageDocument fields

### 31.4 Hard non-goals (written into godoc / ledger)

- [x] No public dotted settings keys
- [x] No `compat` subpackage in 0.2.4
- [x] No C ABI, no JavaScript
- [x] `LibraryVersion` remains the wkhtml settings-surface id; project release stays in `VERSION`

### 31.5 Stub without hard break

- [x] Add types (and no-op or `ErrNotImplemented`-free Validate stubs only if needed for compile) without removing `Converter` yet — prefer landing real Validate in Phase 32
- [x] Do not rewrite examples in this phase
- [x] Inventory current exports that Phase 35 will delete (list in closure notes)

### 31.6 Closure gates

- [x] `make lint` → changed-surface lint passes; repository-wide result recorded in Phase 38
- [x] `make test` → passed with GOCACHE=/tmp/gowkhtmltopdf-go-cache
- [x] Parent Phase 31 row checked
- [x] Next: Phase 32

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

## Validation record (2026-08-18)

- `document.go`, `document_validate.go`, `api.go`, `doc.go`, and `documentation/library-api.md` now define the frozen `Document` / `ImageDocument` contract, ownership boundary, sentinels, hooks, and Cover → TOC → Pages order.
- `GOCACHE=/tmp/gowkhtmltopdf-go-cache make test` passed; focused changed-surface lint passed. The full lint command was also run during closure and is recorded in Phase 38.
