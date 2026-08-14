# Phase 6 — Settings, CLI, and Library

> **Parent:** `plans/0.2.2/pdf-1.7-compliance-plan/00-canonical-pdf-17-compliance-plan.md`
> **Status:** not started
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

- [ ] Field on `settings.PdfGlobal` — `internal/settings/settings.go`
- [ ] Default empty
- [ ] Dotted key (e.g. `pdfprofile`) in `reflect.go` + clone
- [ ] Parser accepts only the names phase 1 allowlisted
- [ ] Dual / A-3a / UA-1 forces `PdfVersion` to 1.7 if it was default 1.4; explicit `--pdf-version 1.4` + profile is an error
- [ ] Test: descriptor parity; good/bad table

### 6.2 CLI

- [ ] `--pdf-profile` (name bikeshed) in `addGlobalFlags`, `ModePDF` only
- [ ] `--help` lists accepted values
- [ ] Test: `--pdf-profile a3a-ua1` + HTML writes a claiming 1.7 file (needs phases 2–5)
- [ ] Test: image binary rejects the flag
- [ ] Test: unknown profile is a user error

### 6.3 Library

- [ ] `PdfGlobalOptions.WithPDFProfile` (or similar); invalid values error at `RunPDF` / `WithSetting`, no panic
- [ ] Documented on the preferred `PDFRequest` path
- [ ] Test: omitted profile → unclaimed; dual → 1.7 + profile

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
