## v0.2.1

Third public release of **gowkhtmltopdf**: a **pure-Go**, **no-cgo**, **no Qt/WebKit**, **no browser** HTML template engine that turns structured HTML and templates into multi-page PDFs and images.

While **v0.2.0** completed the core typography, print CSS, and layout pipeline, **v0.2.1** is a stability and hardening release focused on embedder crash-safety, print layout precision, trust boundary unification, and arbitrary input resilience.

- **License:** [MIT](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/LICENSE) — Copyright (c) 2026 Chinmay Sawant
- **Version source:** [`VERSION`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/VERSION) (`0.2.1`)
- **Site:** https://chinmay-sawant.github.io/gowkhtmltopdf/
- **Compare:** https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.2.0...v0.2.1
- **PRs:** [#41](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/41) (v0.2.1 roadmap and release closure) · [#42](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/42) (fixture header alignment)

---

### Highlights

| Area | What you get in v0.2.1 |
|------|------------------------|
| **Crash safety** | Fluent builders (`WithPageSize`, `WithCopies`) no longer panic on invalid input; errors return cleanly via `ValidatePDF` / `RunPDF` sentinels (`ErrInvalidPageSize`, `ErrInvalidPDFCopies`). `AddHTML` safely guards against nil receivers. |
| **Local file access** | Added convenient `EnableLocalFileAccess()` helpers across `PDFRequest`, `ImageRequest`, `Converter`, `GlobalSettings`, and `ObjectSettings`, replacing the confusing two-key string requirement. |
| **Multi-page tables** | Multi-page `border-collapse` tables with `rowspan` emit clean closed top edges across continuation pages while preserving continuous rowspan cells. |
| **Flow layout** | Multiple same-side floats cleanly stack vertically without overlapping; block formatting context (BFC) enclosure verified; flex shrink item widths and grid row heights are deterministic. |
| **Fuzz testing** | Added continuous native Go fuzz targets (`FuzzParseHTML`, `FuzzParseCSS`, `FuzzConvertHTML`) with 400,000+ random mutations verifying parser and conversion robustness against untrusted input. |
| **Security boundary** | Canonical `NetworkPolicy` type consolidated in `internal/load` with `ApplyNetworkPolicy` helper; public `NetworkPolicy` is a clean type alias. Decoupled PDF and Image conversion request structs. |
| **Showcase & docs** | Refreshed 167 high-resolution PNG page previews and 167 WebP thumbnails across all 64 sample PDFs. Documentation site and release metadata 100% aligned to v0.2.1. |
| **CI integration** | GitHub Actions CI now triggers on both `master` and `main` branches, ensuring all future PRs and merges are continuously tested and linted. |

---

### Showcase — verified document templates

Live gallery: https://chinmay-sawant.github.io/gowkhtmltopdf/#/showcase  
Committed PDFs: [`output/`](https://github.com/chinmay-sawant/gowkhtmltopdf/tree/v0.2.1/output) · source HTML: [`testdata/golden/`](https://github.com/chinmay-sawant/gowkhtmltopdf/tree/v0.2.1/testdata/golden)

All 64 sample documents have been generated and validated with zero CGO or external browser dependencies:

| Category | What you can open |
|----------|-------------------|
| **Invoices & receipts** | Simple / CSS invoices, receipts, purchase orders, contracts, letters, shipping documents, airline boarding passes |
| **Reports & tables** | Detailed ops reports, multi-page tables with repeating `<thead>` and continuation borders, sticky print headers, colorful reports |
| **Storybooks & posters** | Asteria and Ember Harbor storybooks, night-train and observatory posters, certificates |
| **CSS & layout fixtures** | Flex, grid, float, sticky, multicol, `:has()`, `@container`, transforms, CJK + `--font-path`, nested HTML headers/footers |
| **Architecture & API** | Library architecture diagram, 20-page HTML+CSS architecture doc, font-examples (1,125 Google Fonts via `--font-path`), complex dossier |

---

### Install / build

Cross-platform binaries are attached to this release (`gowkhtmltopdf` and `gowkhtmltoimage` for linux / windows / darwin × amd64 / arm64) plus `SHA256SUMS`.

From source (Go 1.26+):

```sh
git clone https://github.com/chinmay-sawant/gowkhtmltopdf.git
cd gowkhtmltopdf
git checkout v0.2.1
make build
```

Convert any committed template:

```sh
./bin/gowkhtmltopdf --enable-local-file-access \
  testdata/golden/fixture-01-simple-invoice.html /tmp/invoice.pdf
```

Library (preferred typed API):

```go
package main

import (
    "bytes"
    "context"
    "os"

    "gowkhtmltopdf"
)

func main() {
    ctx := context.Background()
    var out bytes.Buffer

    req := gowkhtmltopdf.NewPDFRequest().EnableLocalFileAccess()
    req.Objects = []*gowkhtmltopdf.ObjectSettings{
        gowkhtmltopdf.NewObjectSettings().SetPage("invoice.html"),
    }
    req.Output = &out

    if err := gowkhtmltopdf.RunPDF(ctx, req); err != nil {
        panic(err)
    }

    _ = os.WriteFile("output.pdf", out.Bytes(), 0644)
}
```

---

### What landed in v0.2.1

#### 1. Library API & Crash-Safety
- **Panic-Free Option Validation:** `PdfGlobalOptions.WithPageSize` and `PdfGlobalOptions.WithCopies` no longer panic on malformed user strings or invalid integer ranges. Errors are deferred to validation time (`ValidatePDF` / `RunPDF`) with sentinel errors `ErrInvalidPageSize` and `ErrInvalidPDFCopies`.
- **Nil-Safe Receivers:** Added nil checks across mutators (`AddHTML`, `SetPage`, `SetBody`, `WithGlobal`).
- **Ergonomic Local File Access:** Added `EnableLocalFileAccess()` helpers on `PDFRequest`, `ImageRequest`, `Converter`, `GlobalSettings`, and `ObjectSettings`.

#### 2. Layout, Pagination & Table Rendering
- **Table Continuation Borders:** Multi-page tables with `border-collapse: collapse` now render closed top edges across continuation pages while maintaining continuous visual borders for rowspan cells spanning multiple pages.
- **Float Stacking & BFC:** Multiple same-side floats cleanly stack vertically without overlapping. Block formatting context (BFC) containment prevents floats from escaping their parent boundaries.
- **Flex & Grid Bounds:** Explicit `flex-shrink` width calculations verified against narrow viewports; single-span grid item heights calculate based on content height rather than line-height assumptions.

#### 3. Continuous Fuzzing & Input Hardening
- **Native Go Fuzz Targets:** Added continuous fuzz testing targets:
  - `FuzzParseHTML` in `internal/html` (430k+ executions)
  - `FuzzParseCSS` in `internal/css` (440k+ executions)
  - `FuzzConvertHTML` in `internal/convert` (arbitrary HTML -> PDF conversion resilience)
- Proves that malicious or deeply nested HTML/CSS inputs fail gracefully without memory out-of-bounds panics or infinite loops.

#### 4. Architecture & Security Hygiene
- **Canonical Network Policy:** Consolidated network trust boundaries into `internal/load.NetworkPolicy` with `ApplyNetworkPolicy` helper, cleanly aliased to `gowkhtmltopdf.NetworkPolicy`.
- **Pipeline Decoupling:** Cleaned `convert.Request` by removing leftover image fields from the PDF pipeline. Documented reflection-free hand-dispatch table in `internal/settings/reflect.go`.
- **Package Documentation:** Updated `internal/pdf/doc.go` to accurately reflect the pure-Go PDF 1.4 generator.

---

### Documentation

| Doc | Link |
|------|------|
| **Site** | https://chinmay-sawant.github.io/gowkhtmltopdf/ |
| **Overview** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/documentation/overview.md |
| **Getting started** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/documentation/getting-started.md |
| **Architecture** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/documentation/architecture.md |
| **Library API** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/documentation/library-api.md |
| **Fidelity** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/documentation/fidelity.md |
| **Performance** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/documentation/performance.md |
| **Deferred** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/documentation/deferred.md |
| **Contributing** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/CONTRIBUTING.md |
| **Changelog** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/CHANGELOG.md#021-2026-08-14 |
| **Roadmap** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.1/plans/0.2.1/24-canonical-0.2.1-roadmap.md |

---

### Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | Full backward compatibility with existing wkhtmltopdf-compatible dotted `Set`/`Get` and `Converter` API. |

---

### Verification Gates

```sh
make test
make lint
make claim-scan
make build
```

---

## What's Changed

* feat(release): v0.2.1 library contracts, layout fidelity, and verification by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/41
* test: align opening comment header in architecture-diagram.html by @chinmay-sawant in https://github.com/chinmay-sawant/gowkhtmltopdf/pull/42

**Full Changelog**: https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.2.0...v0.2.1
