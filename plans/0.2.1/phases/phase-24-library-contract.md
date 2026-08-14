# Phase 24 - Library Contract

> **Parent:** `plans/0.2.1/24-canonical-0.2.1-roadmap.md`
> **Status:** not started
> **Estimated effort:** 1–2 weeks
> **Depends on:** v0.2.0 Phase 11 (library API exists)
> **Unblocks:** Phase 30 library docs; embedders of `PDFRequest` / `Converter`

---

## Overview

The root package currently exposes three ways to configure a job: dotted
`Set`/`Get` (wkhtmltopdf compatibility), fluent `PdfGlobalOptions` (panic on
bad input), and typed `PDFRequest` / `ImageRequest`. Nil-receiver behavior
is not uniform. This phase picks one preferred embedder path, makes the
others adapters, and puts user-facing validation on the error path.

## Executive Summary

| Surface | Today | Target |
|---------|-------|--------|
| Preferred embedder API | Documented as typed `RunPDF` / `RunImage`; examples still teach `Converter` + `Set` | One preferred path in `doc.go`, `library-api.md`, and `examples/` |
| Dotted `Set` | Complete wkhtml surface; Policy A ignored keys | Keep; it is the compatibility product |
| Fluent `PdfGlobalOptions` | Panics on invalid page size / copies | Errors at convert/build time, or typed enums that cannot spell invalid input |
| Nil receivers | `AddObject` swallows nil; `AddHTML` panics | Same policy on every mutator |
| Local-file ACL | Two magic strings | Helper that sets the pair |

---

## Phase 24 checklist

### 24.1 Policy (write it down first)

- [ ] Write the nil / error / panic table into `documentation/library-api.md`: panic only for programmer-broken builders (`nil` fluent receiver); invalid user values return `error` at `Convert` / `RunPDF` / `RunImage` / `WithSetting`
- [ ] Name the preferred embedder entry: typed `PDFRequest` + `RunPDF` and `ImageRequest` + `RunImage`
- [ ] State that dotted `Set` remains the complete compatibility surface
- [ ] State that `Converter` remains supported; new examples do not lead with it

### 24.2 Nil-receiver consistency

- [ ] `AddHTML` nil-checks the receiver the same way `AddObject` does — `api.go` (`AddHTML`, `AddObject`)
- [ ] Audit every exported mutator on `Converter`, `ImageConverter`, `GlobalSettings`, `ObjectSettings`, `ImageSettings`, `PdfGlobalOptions` for mixed swallow / panic / error
- [ ] Test: `TestAddHTMLNilReceiver` (or equivalent) matches `AddObject` behavior
- [ ] Test: existing `TestNilRequestSentinelsAreDistinct` still passes; do not add new nil sentinels unless a caller must distinguish them

### 24.3 Fluent builder validation

- [ ] `PdfGlobalOptions.WithPageSize` does not panic on an unknown size — `api.go`
- [ ] `PdfGlobalOptions.WithCopies` does not panic on `copies < 1` — `api.go`
- [ ] Invalid values fail at `RunPDF` / `ValidatePDF` / `WithSetting` with `errors.Is` to the existing sentinels (`ErrInvalidPageSize`, `ErrInvalidPDFCopies`)
- [ ] Test: table of bad page sizes and copy counts returns errors, does not panic
- [ ] Exported `With*` methods that stay public get godoc (`WithMargins`, `WithTitle`, `WithOutline`, `WithSmartShrinking`, …)

### 24.4 Embedder helpers

- [ ] Add `EnableLocalFileAccess()` (name bikeshed OK) that sets both `enablelocalfileaccess=true` and `load.blocklocalfileaccess=false` — `api.go` + `ObjectSettings` as needed
- [ ] Test: helper is sufficient to convert a `testdata/golden` fixture from a file path without extra `Set` calls
- [ ] Update `examples/pdf/main.go` to use the helper instead of the two-string dance

### 24.5 Docs and examples

- [ ] `doc.go` quick-start uses the preferred typed path
- [ ] `documentation/library-api.md` leads with `RunPDF` / `PDFRequest`; `Converter` is “compatibility / lifecycle hooks”
- [ ] `examples/pdf` and `examples/image` compile and produce `%PDF-` / PNG/JPEG magic
- [ ] Progress / outline-dump / `Now` availability is documented per surface (what `Converter` has vs what `PDFRequest` has)

### 24.6 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Parent Phase 24 row checked
- [ ] Next: Phase 25

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 11 library surface | Phase 30 docs |
| Existing sentinels in `api.go` / `errs` | Stable `errors.Is` |

---

## Out of scope

- Changing dotted key names
- C ABI
- Publishing a new module path (record the current `gowkhtmltopdf` path honestly; do not invent `go get` URLs)
- Deleting `Converter` in this release
