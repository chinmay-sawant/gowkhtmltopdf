# Phase 08 — Go Library API

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** complete (2026-08-03)  
> **Estimated effort:** 3–6 weeks solo  
> **Depends on:** Phase 5–6 convert path working  
> **Unblocks:** embedders; optional C ABI later

---

## Overview

Expose an idiomatic Go API mirroring the lifecycle of `pdf.h` / `image.h` without requiring C. Optional C ABI is deferred.

## Checklist

### 8.1 PDF package API
- [x] `type GlobalSettings`, `ObjectSettings` with Get/Set string keys — root package `gowkhtmltopdf`: `GlobalSettings`/`ObjectSettings` wrapping settings.PdfGlobal/PdfObject; `Set` delegates to settings dotted names, `Get` via reflect (case-insensitive, canonical rendering)
- [x] `type Converter` with AddObject, Convert, Output, HttpErrorCode — `NewConverter()`, `AddObject(*ObjectSettings) *Converter`, `Convert(ctx) error`, `Output() []byte`, `HttpErrorCode() int` (placeholder 0)
- [x] Callbacks: Debug/Info/Warning/Error, PhaseChanged, ProgressChanged, Finished — `OnInfo/OnWarn/OnError`, `OnPhase`, `OnProgress` (log tee + RunPDFContext progress)
- [x] Context-aware `Convert(ctx context.Context)` with cancel — threaded through RunPDFContext (cancel test passes)
- [x] Example: `examples/pdf/main.go` parallel to `examples/pdf_c_api.c` — builds and runs (verified 1-page PDF)

### 8.2 Image package API
- [x] Single settings + Convert + Output — `ImageConverter` (Set/AddObject/Convert/Output) wrapping imageout.Run
- [x] Example image program — `examples/image/main.go` (verified PNG magic bytes)

### 8.3 Version / init
- [x] `Version() string` — `"0.12.7-dev (gowkhtmltopdf pure-go)"`
- [x] No process-global QApplication; document thread-safety (prefer one convert per call) — documented "one Converter per conversion; not safe for concurrent Convert"

### 8.4 Optional C ABI
- [ ] `[~]` cgo exports matching `wkhtmltopdf_*` — only if consumer demand; not MVP — deferred as planned

### 8.5 Closure
- [x] Examples build with `go run` — `examples/pdf`, `examples/image` verified end-to-end
- [x] `make test` / `make lint` — `go test ./...` / `go vet ./...` / `gofmt -l .` all clean (2026-08-03)

---

## Design notes (filled 2026-08-03)

- **Output capture via temp file**: the pipeline writes to `cmd.Output` or stdout; the library converts to a temp file and reads it back — no memory-buffer path in convert yet.
- **Local-file ACL documented**: library users must set BOTH `Global().Set("enablelocalfileaccess","true")` AND `obj.Set("load.blocklocalfileaccess","false")`, mirroring the CLI pair.

---

## Upstream refs

- `pdf.h`, `pdf_c_bindings.cc`
- `image.h`, `image_c_bindings.cc`
- `examples/*.c`
