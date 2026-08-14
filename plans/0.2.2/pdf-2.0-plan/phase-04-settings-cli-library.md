# Phase 4 — Settings, CLI, and Library

> **Parent:** `plans/0.2.2/pdf-2.0-plan/00-canonical-pdf-20-plan.md`
> **Status:** completed (2026-08-15) — 4.1 (settings) and 4.2 (CLI) complete;
> 4.3/4.4 closed by the phase-5/6 agent (api.go, convert.go no longer reference
> any `settings.ErrPDF20Unsupported` sentinel — the deprecated sentinel was
> removed from `internal/settings/settings.go` with zero remaining references).
> End-to-end 2.0 output proof landed in phase 5.
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

- [x] Add a field on `settings.PdfGlobal` — `internal/settings/settings.go`
  (`PdfVersion string`, ~line 464, landed with #31; doc comment updated to
  `"1.4" (default), "1.7", or "2.0"`)
- [x] `DefaultPdfGlobal` leaves it at 1.4 (zero value or explicit default)
  (empty string; the `pdfversion` getter normalizes "" → "1.4",
  `internal/settings/reflect.go:517-523`; snapshot test asserts
  `PdfVersion == ""` at `settings_test.go:21`)
- [x] Dotted key registered in `internal/settings/reflect.go` (and getters if needed)
  (`pdfversion` setter/getter, `reflect.go:506-524`)
- [x] Parser accepts `2.0` in addition to the #31 values `1.4` / `1.7`
  (`ParsePDFVersion` `settings.go:49-63`: "" → "1.4", "1.4", "1.7", "2.0")
- [x] Invalid values return a sentinel (`ErrInvalidPDFVersion` or similar) — no panic
  (garbage/`9.9`/`1.5` wrap `ErrInvalidPDFVersion`; the temporary
  `ErrPDF20Unsupported` sentinel was removed entirely during phase 5 — PDF
  2.0 is supported and no code path returns it anymore)
- [x] Clone path copies the field — `internal/settings/clone.go`
  (`ClonePdfGlobal` copies `PdfVersion` via the struct-value copy; string,
  no extra clone needed)
- [x] Test: descriptor parity still fails if the key and struct field drift
  (`TestGlobalKeyDescriptorsHaveSetAndGetSides` `reflect_parity_test.go`,
  `TestKeyTableSetGetParity` `settings_test.go:529` — both green)
- [x] Test: table of good/bad strings
  (`TestParsePDFVersion` `settings_test.go:652` — updated: `2.0`/` 2.0 `
  now succeed; `9.9`/`invalid`/`1.5` still fail; green)

### 4.2 CLI

- [x] Register `--pdf-version` in `addGlobalFlags` — `internal/cli/flags.go`
  (`flags.go:142`)
- [x] Mode is `ModePDF` only (`flags.go:142`)
- [x] Value is passed through `c.Global.Set(...)`
  (`c.Global.Set("pdfversion", vals[0])`, `flags.go:143`)
- [x] `--help` lists the flag (`TestPDFVersionFlag` help assertions,
  `cli_test.go:1070-1085` — green)
- [x] Test: parse `--pdf-version 2.0` writes the setting
  (`TestPDFVersionFlag` `cli_test.go:1050-1053` — updated, asserts
  `Global.PdfVersion == "2.0"`; green)
- [x] Test: `--pdf-version 9.9` is a user error (non-zero / `error`), not Policy A ignore
  (`cli_test.go:1040-1048`, `errors.Is(settings.ErrInvalidPDFVersion)` — green)
- [x] Test: image binary rejects the flag (unknown option)
  (`cli_test.go:1064-1068` — green)

### 4.3 Library

- [x] `PdfGlobalOptions.WithPDFVersion` (name bikeshed OK) returns the builder; invalid values fail at `Build` / `RunPDF` / `WithSetting`, not via panic — `api.go`
  (`settings/options.go:71` + `api.go:227`; builder stores the string;
  validation happens at preflight via `settings.ParsePDFVersion` —
  `api.go:679-683` ConvertPDF and `api.go:1175-1179` `ValidatePDF`; both now
  accept `2.0` and keep rejecting garbage with `ErrInvalidPDFVersion`.
  The `WithPDFVersion` doc comment now lists `"2.0"` — `api.go:224-226`)
- [x] Preferred embedder path (`PDFRequest` + `RunPDF`) documents the setting
  (`PDFRequest.Global` carries `PdfGlobalOptions`; `ValidatePDF` validates
  the version. End-to-end RunPDF 2.0 output proof landed in phase 5:
  `api_test.go` `TestPDFVersionAPI` asserts `%PDF-2.0` — green)
- [x] Dotted `Set("pdfversion", "2.0")` remains the complete compatibility surface
  (`reflect.go:506-524`; covered by `TestGlobalPdfVersionSetting`
  `settings_test.go:691` — green)
- [x] Test: invalid version on `RunPDF` / `ValidatePDF` is `errors.Is` the sentinel
  (`api_test.go` invalid-version cases exercise `ErrInvalidPDFVersion`.
  The stale `WithPDFVersion("2.0")`-expects-error cases at
  `api_test.go:1726/1740` were flipped to success by the phase-5/6 agent;
  `RunPDF` with 2.0 now asserts a `%PDF-2.0` header — green)
- [x] Test: omitted version on `PDFRequest` is 1.4
  (default `PdfVersion == ""` → `ParsePDFVersion("")` → "1.4",
  `TestDefaultPdfGlobalSnapshot` + `TestParsePDFVersion` `""` case — green;
  end-to-end `%PDF-1.4` assertion for the omitted version is in
  `api_test.go` `TestPDFVersionAPI` step 4 — green)

### 4.4 Convert request shape (types only)

- [x] `convert.Request` does not need a parallel version field if it already carries `Global`
  (verified: `Request.Global settings.PdfGlobal` — `convert.go:50`)
- [x] Do **not** call `NewDocumentWithPolicy` yet (that is phase 5)
  (closed: phase 5 landed `convert.Run` constructing the document through
  `NewDocumentWithPolicy(policy)` at `convert.go:364`; `PolicyForGlobal`
  now maps `"2.0"` → `pdf.WriterPolicy{Version: pdf.PDF20}` and keeps
  `""`/`"1.4"` → PDF14)
- [x] Test: `go test ./internal/settings ./internal/cli` green
  (ran 2026-08-15: both packages `ok`; `go vet` clean; `gofmt -l` clean;
  `go build ./...` compiles; full `go test ./...` green after phase 5/6)

---

## Explicitly out of scope

- Changing `pdf.NewDocument()` inside `convert.Run` (phase 5)
- Documentation site copy (phase 7)
- Image-mode version

---

## Done when

CLI and library can name `2.0` or `1.4`, reject garbage, and default
to 1.4, without yet changing emitted files in `convert.Run`.

(As of 2026-08-15 the "without yet changing emitted files" boundary has
been crossed by design: phase 5 now emits `%PDF-2.0` through
`convert.Run`, completing this phase's seam.)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 `PDFVersion` (or a settings-local enum mapped in phase 5) | Phase 5 |
