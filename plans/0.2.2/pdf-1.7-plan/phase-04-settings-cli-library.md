# Phase 4 — Settings, CLI, and Library

> **Parent:** `plans/0.2.2/pdf-1.7-plan/00-canonical-pdf-17-plan.md`
> **Status:** completed
> **Estimated effort:** 2–3 days
> **Depends on:** Phase 1 (`PDFVersion` exists). Convert wiring is phase 5.
> **Unblocks:** phase 5 user-visible path; 2.0 plan later adds `2.0` to this same key

---

## Overview

wkhtmltopdf has no PDF version flag. This is a new, explicit setting.
This plan **owns** the key. Policy A still applies: do not accept a
silent no-op. Unknown values error at `Set` / `Convert` / CLI parse.

Default when the key is absent is PDF 1.4. Image mode does not get
this key. `2.0` is rejected here until #32 lands.

---

## Executive Summary

| Surface | Today | Target |
|---------|-------|--------|
| `settings.PdfGlobal` | no version field | typed version field (enum or string with one parser) |
| Dotted `Set` | n/a | one key, e.g. `pdfversion` → `1.4` / `1.7` |
| CLI | n/a | `--pdf-version 1.4\|1.7` on `ModePDF` only |
| `PdfGlobalOptions` | n/a | `WithPDFVersion(...)` that cannot panic on user input |
| `PDFRequest` | no field | reads the global setting; no fourth API |
| Image settings | n/a | key remains unknown |
| `2.0` | n/a | error naming #32 until the sibling plan implements it |

---

## Phase 4 checklist

### 4.1 Settings model

- [x] Add a field on `settings.PdfGlobal` — `internal/settings/settings.go`
- [x] `DefaultPdfGlobal` leaves it at 1.4 (zero value or explicit default)
- [x] Dotted key registered in `internal/settings/reflect.go` (and getters if needed)
- [x] Parser accepts `1.4` and `1.7`
- [x] `2.0` returns a sentinel that names the unimplemented version (or #32) — not a header bump
- [x] Invalid values return a sentinel (`ErrInvalidPDFVersion` or similar) — no panic
- [x] Clone path copies the field — `internal/settings/clone.go`
- [x] Test: descriptor parity still fails if the key and struct field drift
- [x] Test: table of good/bad strings

### 4.2 CLI

- [x] Register `--pdf-version` in `addGlobalFlags` — `internal/cli/flags.go`
- [x] Mode is `ModePDF` only
- [x] Value is passed through `c.Global.Set(...)`
- [x] `--help` lists the flag
- [x] Test: parse `--pdf-version 1.7` writes the setting
- [x] Test: `--pdf-version 9.9` is a user error, not Policy A ignore
- [x] Test: image binary rejects the flag (unknown option)

### 4.3 Library

- [x] `PdfGlobalOptions.WithPDFVersion` (name bikeshed OK) returns the builder; invalid values fail at `Build` / `RunPDF` / `WithSetting`, not via panic — `api.go`
- [x] Preferred embedder path (`PDFRequest` + `RunPDF`) documents the setting
- [x] Dotted `Set("pdfversion", "1.7")` remains the complete compatibility surface
- [x] Test: invalid version on `RunPDF` / `ValidatePDF` is `errors.Is` the sentinel
- [x] Test: omitted version on `PDFRequest` is 1.4

### 4.4 Convert request shape (types only)

- [x] `convert.Request` does not need a parallel version field if it already carries `Global`
- [x] Do **not** call `NewDocumentWithPolicy` yet (that is phase 5)
- [x] Test: `go test ./internal/settings ./internal/cli` green

---

## Explicitly out of scope

- Changing `pdf.NewDocument()` inside `convert.Run` (phase 5)
- Accepting `2.0` as a working emit path (#32)
- Documentation site copy (phase 7)
- Image-mode version

---

## Done when

CLI and library can name `1.7` or `1.4`, reject garbage and `2.0`, and
default to 1.4, without yet changing emitted files in `convert.Run`.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 `PDFVersion` | Phase 5; #32 extends the same key |
