# Phase 11 - Library API for Go Embedders

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** not started (MVP Phase 8 API already shipped)  
> **Estimated effort:** 1–2 weeks  
> **Depends on:** Phase 8 complete (`api.go`, examples)  
> **Unblocks:** other Go projects adding gowkhtmltopdf as a library  
> **Tier:** 1 · **Constraint:** stdlib-only module (zero `require` deps)

---

## Overview

MVP already exposes an idiomatic Go API. This phase makes it **safe and obvious** for other Go applications to import and use: better docs, integration recipes, small API gaps, and verification that the public surface is embedder-ready - without third-party plugins or C ABI.

## Executive Summary

| Exists today | Gap for embedders |
|--------------|-------------------|
| `NewConverter` / `Convert` / `Output` | HTML-bytes → PDF path may force temp files / page paths |
| Dotted `Set`/`Get` | Discoverability of keys is CLI-heavy |
| `examples/pdf`, `examples/image` | Thin; no “service embed” recipe |
| `documentation/library-api.md` | Incomplete image surface; ACL pair easy to miss |
| Phase 8.4 C ABI | Deferred forever under no-cgo |

---

## Phase 11 checklist

### 11.1 Public API inventory (evidence)

- [ ] Inventory all exported types/funcs in root package (`api.go`, `doc.go`) and list in library-api.md
- [ ] Document which settings keys are honored vs accepted-but-ignored (link matrix §7 / settings)
- [ ] Document error contracts: load errors, convert errors, cancel via `context`
- [ ] Document concurrency: one `Converter` per conversion; not safe for concurrent `Convert`
- [ ] Document determinism: identical inputs → identical PDF bytes (state limits if any)

### 11.2 Embedder DX documentation

- [ ] Expand `documentation/library-api.md`:
  - Convert **local file** (ACL pair: `enablelocalfileaccess` + `load.blocklocalfileaccess=false`)
  - Convert **remote URL**
  - Convert **in-memory HTML** (document current approach: temp file / stdin / planned helper)
  - Multi-object: cover + body pages
  - Headers/footers text placeholders
  - Outline / TOC flags from library
  - Image converter full example (width, format, quality)
  - Callbacks: OnInfo/OnWarn/OnError/OnPhase/OnProgress
- [ ] Add **HTTP handler recipe** in docs (stdlib `net/http` only - no Gin dependency in this module)
- [ ] Cross-link `documentation/integration-security.md` (SSRF, local file ACL)
- [ ] Module install story: `go get` / replace / version tags when published

### 11.3 API surface polish (code, only if gap proven)

- [ ] Evaluate temp-file output path in library convert; if awkward, add in-memory capture path under stdlib
  - Path: `api.go` + `internal/convert` as needed
  - Expected: `Output() []byte` without leaving temp files on success
  - Proof: unit test + no leftover files in `os.TempDir` pattern test
- [ ] Optional helper: `ConvertHTML(ctx, html []byte, global settings…) ([]byte, error)` if it reduces boilerplate without hiding ACL
- [ ] Optional helper: `ConvertFile` / `ConvertURL` thin wrappers - only if docs alone are insufficient
- [ ] Ensure `HttpErrorCode()` behavior is documented and tested (or marked stub honestly)
- [ ] Do **not** add cgo / shared-lib ABI (phase 8.4 remains `[~]`)

### 11.4 Examples quality

- [ ] `examples/pdf/main.go`: comment every required Set for ACL + page size
- [ ] `examples/image/main.go`: complete working flags
- [ ] Optional third example under `examples/embed/` showing library-as-dependency pattern with relative replace (document only if module path not published yet)
- [ ] Verify: `go run ./examples/pdf` and `go run ./examples/image` produce valid magic bytes

### 11.5 Tests

- [ ] Library tests cover Convert cancel (`context` cancel mid-run)
- [ ] Library tests cover multi-object or multi-page minimal HTML
- [ ] Library tests cover image convert smoke
- [ ] No new third-party test deps

### 11.6 Closure gates

- [ ] `make lint` → record result
- [ ] `make test` → record result
- [ ] Docs + examples reviewed for copy-paste correctness
- [ ] Parent Phase 11 rows checked
- [ ] Next handoff: **Phase 12** (real bold/italic faces)

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
