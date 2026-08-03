# Phase 08 — Go Library API

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** not started  
> **Estimated effort:** 3–6 weeks solo  
> **Depends on:** Phase 5–6 convert path working  
> **Unblocks:** embedders; optional C ABI later

---

## Overview

Expose an idiomatic Go API mirroring the lifecycle of `pdf.h` / `image.h` without requiring C. Optional C ABI is deferred.

## Checklist

### 8.1 PDF package API
- [ ] `type GlobalSettings`, `ObjectSettings` with Get/Set string keys
- [ ] `type Converter` with AddObject, Convert, Output, HttpErrorCode
- [ ] Callbacks: Debug/Info/Warning/Error, PhaseChanged, ProgressChanged, Finished
- [ ] Context-aware `Convert(ctx context.Context)` with cancel
- [ ] Example: `examples/pdf/main.go` parallel to `examples/pdf_c_api.c`

### 8.2 Image package API
- [ ] Single settings + Convert + Output
- [ ] Example image program

### 8.3 Version / init
- [ ] `Version() string`
- [ ] No process-global QApplication; document thread-safety (prefer one convert per call)

### 8.4 Optional C ABI
- [ ] `[~]` cgo exports matching `wkhtmltopdf_*` — only if consumer demand; not MVP

### 8.5 Closure
- [ ] Examples build with `go run`
- [ ] `make test` / `make lint`

---

## Upstream refs

- `pdf.h`, `pdf_c_bindings.cc`
- `image.h`, `image_c_bindings.cc`
- `examples/*.c`
