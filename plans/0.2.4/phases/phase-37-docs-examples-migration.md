# Phase 37 - Docs, Examples, and Migration

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** not started
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

- [ ] `documentation/library-api.md` leads with Document.WritePDF / PDF
- [ ] Content helpers, AllowLocalFiles, Network, TOC/Cover examples
- [ ] Nil / error / panic table matches Phase 31
- [ ] Remove dotted key tables as the primary contract (optional appendix: “engine internals still use settings keys” — do not teach them as public API)
- [ ] `doc.go` quick start uses Document

### 37.2 CLI docs

- [ ] `documentation/cli.md` matches Phase 36 grammar and exit codes
- [ ] getting-started walkthrough updated
- [ ] Help text and docs agree

### 37.3 Migration guide

- [ ] Add `documentation/MIGRATION-0.2.4.md` with tables:
  - `Converter` → `Document`
  - `GlobalSettings.Set("size.pagesize", "A4")` → `PageSize: "A4"`
  - `SetBody(html, base)` → `Content{HTML: html, Base: base}` / `HTML(...)`
  - `SetPage(path)` → `File(path)` / `Content{File: path}`
  - `EnableLocalFileAccess` pair → `AllowLocalFiles: true`
  - `RunPDF` / `PDFRequest` → `WritePDF` / `PDF`
  - `NewTOCObject` / `NewCoverObject` → `TOC` / `Cover`
  - Old CLI object grammar → new flags
- [ ] Link migration guide from README and library-api.md

### 37.4 Examples and site

- [ ] `examples/pdf` and `examples/image` compile and produce magic bytes
- [ ] Update comparison doc vs SebastiaanKlippert: typed structs **without** shelling out
- [ ] Update frontend/docs snippets that demonstrate library or CLI usage
- [ ] Architecture docs that cite `Converter` as the public face updated

### 37.5 Plans index hygiene

- [ ] `plans/README.md` 0.2.4 row accurate (planned → in progress / complete as status moves)
- [ ] Any 0.2.1 Phase 24 “preferred PDFRequest” claim superseded: add `[~]` pointer to this ledger where still listed as future work

### 37.6 Closure gates

- [ ] Docs-only edits: no lint/test required solely for markdown; if Go examples changed, run `make lint` and `make test`
- [ ] Record example smoke commands and outcomes
- [ ] Parent Phase 37 row checked
- [ ] Next: Phase 38

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 35–36 final surfaces | Phase 38 release notes |

---

## Out of scope

- New fidelity claims
- Translating all historical PR bodies
