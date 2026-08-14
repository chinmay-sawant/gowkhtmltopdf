# Phase 11 - Library API for Go Embedders

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** complete (2026-08-04) - docs/examples + install story + `ConvertHTML`  

> **Estimated effort:** 1–2 weeks  
> **Depends on:** Phase 8 complete (`api.go`, examples)  
> **Unblocks:** other Go projects adding gowkhtmltopdf as a library  
> **Tier:** 1 · **Constraint:** stdlib-only module (zero `require` deps)

---

## Overview

MVP already exposes an idiomatic Go API. This phase makes it **safe and obvious** for other Go applications to import and use: better docs, integration recipes, small API gaps, and verification that the public surface is embedder-ready - without third-party plugins or C ABI.

## Executive Summary

| Exists today | Status |
|--------------|--------|
| `NewConverter` / `Convert` / `Output` | Shipped; `ConvertHTML` helper for in-memory HTML |
| Dotted `Set`/`Get` | Documented; matrix §7 for honored vs ignored |
| `examples/pdf`, `examples/image` | Working |
| `documentation/library-api.md` | Install + ACL + ConvertHTML |
| Phase 8.4 C ABI | Deferred forever under no-cgo |

---

## Phase 11 checklist

### 11.1 Public API inventory (evidence)

- [x] Inventory all exported types/funcs in root package (`api.go`, `doc.go`) and list in library-api.md
- [x] Document which settings keys are honored vs accepted-but-ignored (link matrix §7 / settings)
- [x] Document error contracts: load errors, convert errors, cancel via `context` (`doc.go`, library-api.md)
- [x] Document concurrency: one `Converter` per conversion; not safe for concurrent `Convert`
- [x] Document determinism: identical inputs → identical PDF bytes (overview/README)

### 11.2 Embedder DX documentation

- [x] Expand `documentation/library-api.md`:
  - Convert **local file** (ACL pair: `enablelocalfileaccess` + `load.blocklocalfileaccess=false`)
  - Convert **remote URL** (SetPage URL)
  - Convert **in-memory HTML** (inline:/data: page sources via settings + `ConvertHTML`)
  - Multi-object: cover + body pages
  - Headers/footers text placeholders (settings surface)
  - Outline / TOC flags from library
  - Image converter example (examples/image)
  - Callbacks: OnInfo/OnWarn/OnError/OnPhase/OnProgress
- [x] Add **HTTP handler recipe** in docs (`documentation/integration-security.md` - Gin/stdlib patterns)
- [x] Cross-link `documentation/integration-security.md` (SSRF, local file ACL)
- [x] Module install story: `go get` / replace / version tags (`library-api.md` Install)

### 11.3 API surface polish (code, only if gap proven)

- [x] Evaluate temp-file output path in library convert; if awkward, add in-memory capture path under stdlib
  - Path: `api.go` + `internal/convert` as needed
  - Expected: `Output() []byte` without leaving temp files on success
  - Proof: `TestConvertPDFToBytes` + related
- [x] Optional helper: `ConvertHTML(ctx, html []byte, global *GlobalSettings) ([]byte, error)` - `TestConvertHTMLHelper`
- [~] Optional helper: `ConvertFile` / `ConvertURL` thin wrappers - docs alone are sufficient
- [x] Ensure `HttpErrorCode()` behavior is documented and tested (or marked stub honestly)
- [x] Do **not** add cgo / shared-lib ABI (phase 8.4 remains `[~]`)

### 11.4 Examples quality

- [x] `examples/pdf/main.go`: ACL + page size options
- [x] `examples/image/main.go`: complete working flags
- [~] Optional third example under `examples/embed/` showing library-as-dependency pattern with relative replace (document only if module path not published yet)
- [x] Verify: `go run ./examples/pdf` and `go run ./examples/image` produce valid magic bytes

### 11.5 Tests

- [x] Library tests cover Convert cancel (`TestConvertContextCancel`)
- [x] Library tests cover multi-object or multi-page minimal HTML (`TestConvertPDFToBytes` and convert tests)
- [x] Library tests cover image convert smoke (`TestImageConverterPNG` / JPEG)
- [x] No new third-party test deps

### 11.6 Closure gates

- [x] `make lint` → green
- [x] `make test` → green
- [x] Docs + examples reviewed for copy-paste correctness
- [x] Parent Phase 11 rows checked
- [x] Next handoff: **Phase 12** (real bold/italic faces) - **shipped**

```
# closure evidence
# make lint →
# make test →
# go run ./examples/pdf →
# go run ./examples/image →
```

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 8 API | Embedders |
| Phase 10 fidelity language | Docs “what not to expect” for library users |

---

## Out of scope

- C ABI / plugins
- Third-party web frameworks as module deps
- Changing default security to “open local files”
