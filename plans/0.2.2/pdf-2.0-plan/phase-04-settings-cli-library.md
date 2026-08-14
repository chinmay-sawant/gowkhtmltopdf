# Phase 4 — Settings, CLI, and Library

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** not started
> **Estimated effort:** 2–3 days
> **Depends on:** Phase 1 (`PDFVersion` exists). Convert wiring is phase 5.
> **Unblocks:** phase 5 user-visible path

---

## Overview

wkhtmltopdf has no PDF version flag. This is a new, explicit setting.
Policy A still applies: do not accept a silent no-op. Unknown values
error at `Set` / `Convert` / CLI parse.

Default when the key is absent is PDF 1.4. Image mode does not get
this key.

---

## Executive Summary

| Surface | Today | Target |
|---------|-------|--------|
| `settings.PdfGlobal` | no version field | typed version field (enum or string with one parser) |
| Dotted `Set` | `1.4` / `1.7` from #31 | same key also accepts `2.0` |
| CLI | `--pdf-version` from #31 | same flag accepts `2.0` |
| `PdfGlobalOptions` | n/a | `WithPDFVersion(...)` that cannot panic on user input |
| `PDFRequest` | no field | reads the global setting; no fourth API |
| Image settings | n/a | key remains unknown |

`1.4` and `1.7` are owned by the 1.7 plan. This phase only teaches the
same key to accept `2.0`.

---

## Phase 4 checklist

### 4.1 Settings model

- [ ] Add a field on `settings.PdfGlobal` — `internal/settings/settings.go`
- [ ] `DefaultPdfGlobal` leaves it at 1.4 (zero value or explicit default)
- [ ] Dotted key registered in `internal/settings/reflect.go` (and getters if needed)
- [ ] Parser accepts `2.0` in addition to the #31 values `1.4` / `1.7`
- [ ] Invalid values return a sentinel (`ErrInvalidPDFVersion` or similar) — no panic
- [ ] Clone path copies the field — `internal/settings/clone.go`
- [ ] Test: descriptor parity still fails if the key and struct field drift
- [ ] Test: table of good/bad strings

### 4.2 CLI

- [ ] Register `--pdf-version` in `addGlobalFlags` — `internal/cli/flags.go`
- [ ] Mode is `ModePDF` only
- [ ] Value is passed through `c.Global.Set(...)`
- [ ] `--help` lists the flag
- [ ] Test: parse `--pdf-version 2.0` writes the setting
- [ ] Test: `--pdf-version 9.9` is a user error (non-zero / `error`), not Policy A ignore
- [ ] Test: image binary rejects the flag (unknown option)

### 4.3 Library

- [ ] `PdfGlobalOptions.WithPDFVersion` (name bikeshed OK) returns the builder; invalid values fail at `Build` / `RunPDF` / `WithSetting`, not via panic — `api.go`
- [ ] Preferred embedder path (`PDFRequest` + `RunPDF`) documents the setting
- [ ] Dotted `Set("pdfversion", "2.0")` remains the complete compatibility surface
- [ ] Test: invalid version on `RunPDF` / `ValidatePDF` is `errors.Is` the sentinel
- [ ] Test: omitted version on `PDFRequest` is 1.4

### 4.4 Convert request shape (types only)

- [ ] `convert.Request` does not need a parallel version field if it already carries `Global`
- [ ] Do **not** call `NewDocumentWithPolicy` yet (that is phase 5)
- [ ] Test: `go test ./internal/settings ./internal/cli` green

---

## Explicitly out of scope

- Changing `pdf.NewDocument()` inside `convert.Run` (phase 5)
- Documentation site copy (phase 7)
- Image-mode version

---

## Done when

CLI and library can name `2.0` or `1.4`, reject garbage, and default
to 1.4, without yet changing emitted files in `convert.Run`.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 `PDFVersion` (or a settings-local enum mapped in phase 5) | Phase 5 |
