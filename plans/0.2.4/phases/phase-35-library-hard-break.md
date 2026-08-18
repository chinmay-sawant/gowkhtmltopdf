# Phase 35 - Library Hard Break

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** not started
> **Estimated effort:** 1 week
> **Depends on:** Phase 34 adapters green; in-repo library tests/examples movable to Document
> **Unblocks:** Phase 37 docs honesty; Phase 38 release

---

## Overview

Remove the wkhtml-shaped public library from the root package. There is no
`compat` subpackage in 0.2.4. CLI may still use `internal/settings` dotted
tables privately until Phase 36 replaces the flag surface.

## Executive Summary

| Remove / unexport | Replacement |
|-------------------|-------------|
| `Converter`, `ImageConverter`, `ConvertHTML` | `Document`, `ImageDocument` |
| `PDFRequest`, `RunPDF`, `ImageRequest`, `RunImage` | `WritePDF` / `WriteImage` |
| `GlobalSettings`, `ObjectSettings`, `ImageSettings` | Document / Page / ImageDocument fields |
| `PdfGlobalOptions` | Document fields |
| `NewTOCObject`, `NewCoverObject`, `SetPage`, `SetBody` | `TOC`, `Cover`, `Content` helpers |
| Public `Set` / `Get` dotted APIs | gone |

---

## Phase 35 checklist

### 35.1 Rewrite in-repo consumers first

- [ ] Rewrite `api_test.go` onto Document / ImageDocument
- [ ] Rewrite `examples/pdf` and `examples/image` onto Document / ImageDocument
- [ ] Grep the module for `NewConverter`, `RunPDF`, `GlobalSettings`, `.Set(` on public types; clear library-facing hits
- [ ] Keep `internal/cli` / `internal/app` compiling (they may still use internal settings until Phase 36)

### 35.2 Delete public surface

- [ ] Remove or unexport listed types/funcs from root package (`api.go` and related)
- [ ] Keep sentinels that still apply; remap godoc to Document.Validate / Write*
- [ ] Keep `NetworkPolicy` + constructors
- [ ] Keep `LibraryVersion` / `Version()`
- [ ] Confirm `go/doc` / `go test` show no removed symbols

### 35.3 Public API inventory

- [ ] List every remaining root export in this phase file’s closure notes
- [ ] Ensure no accidental export of adapter helpers that take `settings.PdfGlobal`
- [ ] Ensure no public string key tables

### 35.4 Breaking-change honesty

- [ ] Draft CHANGELOG “Breaking” bullets (applied in Phase 38)
- [ ] Note pre-1.0 but intentional hard break for embedders

### 35.5 Tests

- [ ] `go test ./...` green with zero references to removed symbols in this module’s public tests
- [ ] Existing convert/image internal tests unchanged unless they imported public API

### 35.6 Closure gates

- [ ] `make lint` → (record outcome)
- [ ] `make test` → (record outcome)
- [ ] Parent Phase 35 row checked
- [ ] Next: finish Phase 36 if still open; else Phase 37

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 34 | Working Document path |
| Examples rewrite | Safe delete |

---

## Out of scope

- Adding `gowkhtmltopdf/compat`
- CLI argv redesign (Phase 36)
- Documentation site polish beyond what compile needs (Phase 37)
