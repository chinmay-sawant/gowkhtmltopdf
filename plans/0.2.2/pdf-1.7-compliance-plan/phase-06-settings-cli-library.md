# Phase 6 — Settings, CLI, and Library

> **Parent:** `plans/0.2.2/pdf-1.7-compliance-plan/00-canonical-pdf-17-compliance-plan.md`
> **Status:** completed
> **Estimated effort:** 2–3 days
> **Depends on:** Phase 1 names; user path waits on phases 2–5
> **Unblocks:** phase 7 convert goldens from the CLI

---

## Overview

Compliance is a new, explicit setting. Default remains unclaimed
PDF 1.4. Selecting the dual profile **implies PDF 1.7** (do not let
a user ask for A-3a on 1.4).

Policy A: unknown profile names error. Image mode does not get this
key.

Suggested spellings (bikeshed in implementation):

| User value | Policy |
|------------|--------|
| (absent) | unclaimed, version from `--pdf-version` |
| `a3a-ua1` or `PDF/A-3a+PDF/UA-1` | dual headline |
| reject `a4`, `ua2`, `pdfa-1b` | #33 or unsupported |

---

## Phase 6 checklist

### 6.1 Settings

- [x] Field on `settings.PdfGlobal` — `internal/settings/settings.go`
- [x] Default empty
- [x] Dotted key (e.g. `pdfprofile`) in `reflect.go` + clone
- [x] Parser accepts only the names phase 1 allowlisted
- [x] Dual / A-3a / UA-1 forces `PdfVersion` to 1.7 if it was default 1.4; explicit `--pdf-version 1.4` + profile is an error
- [x] Test: descriptor parity; good/bad table

### 6.2 CLI

- [x] `--pdf-profile` (name bikeshed) in `addGlobalFlags`, `ModePDF` only
- [x] `--help` lists accepted values
- [x] Test: `--pdf-profile a3a-ua1` + HTML writes a claiming 1.7 file (needs phases 2–5)
- [x] Test: image binary rejects the flag
- [x] Test: unknown profile is a user error

### 6.3 Library

- [x] `PdfGlobalOptions.WithPDFProfile` (or similar); invalid values error at `RunPDF` / `WithSetting`, no panic
- [x] Documented on the preferred `PDFRequest` path
- [x] Test: omitted profile → unclaimed; dual → 1.7 + profile

---

## Explicitly out of scope

- Image-mode profile
- Accepting A-4 / UA-2 (#33)

---

## Done when

CLI and library can select the dual 1.7 profile, reject garbage, and
leave the default unclaimed.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 names; phases 2–5 emit | Phase 7 |
