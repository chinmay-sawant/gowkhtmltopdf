# Phase 33 - Options Structs

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** complete — validated 2026-08-18
> **Estimated effort:** 1 week
> **Depends on:** Phase 31–32
> **Unblocks:** Phase 34 adapters

---

## Overview

Map every **honored** engine setting that library/CLI users need onto
Document / Page / ImageDocument / HeaderFooter / TOC / Crop fields. Policy A
(ignored wkhtml keys) stays inside `internal/settings` for any residual CLI
legacy paths during migration, but is **not** part of the new public API or
new CLI flag set.

## Executive Summary

| Area | Struct fields (illustrative) | Engine destination |
|------|------------------------------|--------------------|
| Geometry | `PageSize`, `WidthMM`, `HeightMM`, `Orientation`, `Margin` | `PdfGlobal` |
| PDF version/profile | `PDFVersion`, `PDFProfile` | `PdfGlobal` |
| Document meta | `Title`, `Copies`, `Collate`, outline fields | `PdfGlobal` |
| HF | `HeaderFooter` on Document and Page | `Header`/`Footer` + override bits |
| TOC | `TOC` | TOC object + `PdfGlobal.TOC` defaults |
| Load / security | `AllowLocalFiles`, `Network`, `FontPaths` | Load ACL + policy |
| Image | `Width`, `Height`, `Format`, `Quality`, `Crop`, … | `ImageGlobal` |

---

## Phase 33 checklist

### 33.1 Inventory honored keys

- [x] Walk `documentation/library-api.md` Global / Object / Image key tables and mark each: **field on Document**, **internal-only**, or **dropped (Policy A)**
- [x] Confirm Policy A keys (`load.jsdelay`, `web.javascript`, `dpi`, …) are not Document fields and not new CLI flags
- [x] Record default values that match `settings.DefaultPdfGlobal` / `DefaultImageGlobal` / `DefaultPdfObject` when fields are zero / nil

### 33.2 PDF Document options

- [x] Page size validation reuses `ErrInvalidPageSize`
- [x] PDF version / profile validation reuses existing profile/version sentinels
- [x] `AllowLocalFiles` sets both former ACL halves in the adapter (Phase 34); Document field is a single bool
- [x] `Network *NetworkPolicy` optional; nil → compatible historical loader behavior
- [x] `Now` pins `/CreationDate` and `[date]`/`[time]` placeholders
- [x] Header/footer placeholders documented unchanged (`[page]`, `[topage]`, …)

### 33.3 Page / TOC / Cover options

- [x] Page-level HF override via non-nil `Page.Header` / `Page.Footer`
- [x] Cover is `*Page` with adapter marking `IsCover` (no HF by default unless set)
- [x] TOC struct maps caption / dotted lines / font scale / indentation / link flags
- [x] Outline membership pointers on Page (`IncludeInOutline`)

### 33.4 ImageDocument options

- [x] Width / Height / Quality / Format / SmartWidth / Transparent / Crop
- [x] Shared: `AllowLocalFiles`, `Network`, background, hooks, `Now`
- [x] `web.images` / media-type honors that image mode needs become fields or documented fixed defaults

### 33.5 Tests

- [x] Defaults: zero Document after NewDocument produces same effective globals as today’s defaults (adapter-level or Validate+map golden)
- [x] Invalid page size / profile / version fail Validate or Write with `errors.Is`
- [x] Page HF override bit set when Page.Header non-nil

### 33.6 Closure gates

- [x] `make lint` → changed-surface lint passes; repository-wide result recorded in Phase 38
- [x] `make test` → passed with GOCACHE=/tmp/gowkhtmltopdf-go-cache
- [x] Parent Phase 33 row checked
- [x] Next: Phase 34

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| `internal/settings` struct fields | Phase 34 field-to-field map |
| Phase 32 Content | Complete Document tree |

---

## Out of scope

- Exposing Policy A keys “for script compatibility” on Document
- Functional-options rewrite (structs are the chosen shape)
- Cookie / custom-header / POST maps (CLI-only historically; add only if inventory proves engine support and product need)

## Validation record (2026-08-18)

- Public option inventory is documented in `documentation/library-api.md`; Policy A dotted keys remain internal. `Document` / `ImageDocument` carry the honored geometry, profile, ACL, network, font, header/footer, TOC, image, and hook fields.
- `document_test.go` validates the option surface and invalid page size/profile/version sentinels; `make test` passed. Changed Go files pass focused golangci-lint checks.
