# Phase 37 - Docs, Examples, and Migration

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** complete — validated 2026-08-18
> **Estimated effort:** 3–5 days
> **Depends on:** Phases 35–36
> **Unblocks:** Phase 38

---

## Overview

Make every user-facing document teach the Document model and new CLI. Ship
an explicit migration guide from the 0.2.3 wkhtml-shaped library API.

## Executive Summary

| Doc / artifact | Change |
|----------------|--------|
| `documentation/library-api.md` | Rewrite around Document / ImageDocument |
| `documentation/cli.md` | Rewrite around new grammar |
| `doc.go` | Quick start = Document |
| `examples/pdf`, `examples/image` | Already Document from Phase 35; verify copy-paste |
| `documentation/MIGRATION-0.2.4.md` | **New** old→new table |
| Comparison / overview / deferred | Drop Converter-first language |
| Frontend / docs site snippets | Update if they show library usage |
| `plans/README.md` | 0.2.4 row already added at plan authoring; keep status honest |

---

## Phase 37 checklist

### 37.1 Library docs

- [x] `documentation/library-api.md` leads with Document.WritePDF / PDF
- [x] Content helpers, AllowLocalFiles, Network, TOC/Cover examples
- [x] Nil / error / panic table matches Phase 31
- [x] Remove dotted key tables as the primary contract (optional appendix: “engine internals still use settings keys” — do not teach them as public API)
- [x] `doc.go` quick start uses Document

### 37.2 CLI docs

- [x] `documentation/cli.md` matches Phase 36 grammar and exit codes
- [x] getting-started walkthrough updated
- [x] Help text and docs agree

### 37.3 Migration guide

- [x] Add `documentation/MIGRATION-0.2.4.md` with tables:
  - `Converter` → `Document`
  - `GlobalSettings.Set("size.pagesize", "A4")` → `PageSize: "A4"`
  - `SetBody(html, base)` → `Content{HTML: html, Base: base}` / `HTML(...)`
  - `SetPage(path)` → `File(path)` / `Content{File: path}`
  - `EnableLocalFileAccess` pair → `AllowLocalFiles: true`
  - `RunPDF` / `PDFRequest` → `WritePDF` / `PDF`
  - `NewTOCObject` / `NewCoverObject` → `TOC` / `Cover`
  - Old CLI object grammar → new flags
- [x] Link migration guide from README and library-api.md

### 37.4 Examples and site

- [x] `examples/pdf` and `examples/image` compile and produce magic bytes
- [x] Update comparison doc vs SebastiaanKlippert: typed structs **without** shelling out
- [x] Update frontend/docs snippets that demonstrate library or CLI usage
- [x] Architecture docs that cite `Converter` as the public face updated

### 37.5 Plans index hygiene

- [x] `plans/README.md` 0.2.4 row accurate (planned → in progress / complete as status moves)
- [x] Any 0.2.1 Phase 24 “preferred PDFRequest” claim is superseded by this ledger

### 37.6 Closure gates

- [x] Docs-only edits: no lint/test required solely for markdown; if Go examples changed, run `make lint` and `make test`
- [x] Record example smoke commands and outcomes
- [x] Parent Phase 37 row checked
- [x] Next: Phase 38

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 35–36 final surfaces | Phase 38 release notes |

---

## Out of scope

- New fidelity claims
- Translating all historical PR bodies

## Validation record (2026-08-18)

- Updated root godoc, library/CLI/security/architecture docs, migration guide, examples, comparison docs, generated architecture fixture, and frontend content to the v0.2.4 typed API and CLI grammar.
- `go run ./examples/pdf`, `go run ./examples/image`, and the golden API example produced valid PDF/image magic. `npm --prefix frontend run lint` and `npm --prefix frontend run build` passed; the build regenerated `docs/` from frontend source.
