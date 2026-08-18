# 31 - v0.2.4 Idiomatic Document API and CLI Rethink (Canonical Execution Ledger)

> **Parent:** `plans/0.2.3/README.md` (released module path); contracts from `plans/0.2.1/24-canonical-0.2.1-roadmap.md`
> **Status:** not started
> **Estimated effort:** 6–10 weeks across phases 31–38
> **Constraint:** pure Go, no CGO, no browser embed. Direct modules stay on the existing allowlist (`go-text/typesetting`, `tdewolff/canvas`) unless a new amendment is filed.
> **Ordering principle:** freeze the Document contract first, then content validation, then options, then adapters, then delete the old public surface, then CLI, then docs/release.
> **Workflow:** [`skills/phase-wise-checklist/SKILLS.md`](../../skills/phase-wise-checklist/SKILLS.md)

---

## Overview

v0.2.1–v0.2.3 shipped a wkhtmltopdf-compatible settings surface for both CLI
and library: dotted `Set`/`Get`, `Converter` / `ImageConverter`, and typed
`PDFRequest` that still wraps `*GlobalSettings` / `*ObjectSettings`. That
mirrors the native tool, but it is the wrong default for Go embedders.

v0.2.4 replaces the **public** library with a Go-native **Document** model
(structs for content, pages, cover, TOC, headers/footers, and options) and
redesigns the **CLI** to the same shape. This is an intentional **hard
break**: no `compat` subpackage, no public dotted keys, no
`Converter`-shaped driver in the root package.

**In scope**

1. Freeze and implement `Content`, `Page`, `Document`, `ImageDocument`, and related option structs.
2. Validate content sources (HTML bytes / file / URL) without stringly `page` / `inline:` guessing at the public boundary.
3. Map Document → existing `internal/convert` / `internal/imageout` engines (internals may still use `settings.PdfGlobal`).
4. Delete the wkhtml-shaped public library surface.
5. Redesign `gowkhtmltopdf` / `gowkhtmltoimage` argv to Document-aligned flags.
6. Docs, examples, migration guide, VERSION / CHANGELOG.

**Hard non-goals (unless this ledger is amended)**

- JavaScript / Phase 22, Chrome parity / Phase 23
- C ABI, encryption, AcroForm, SOCKS5
- Keeping a public dotted-key or `Converter` compatibility layer
- Changing PDF 1.7 / 2.0 / profile writer semantics (already shipped in 0.2.2)
- Layout / CSS fidelity work
- Pixel-diff goldens as the default gate

**Product decisions locked**

| Decision | Choice |
|----------|--------|
| Library break | Hard break in the root package |
| Content shape | Full document model (cover, TOC, pages, HF, sink) |
| Old API refuge | None in 0.2.4 |
| CLI | Redesign in the same release |

---

## Phase map

```text
31 Document model contract
  → 32 Content and validation
  → 33 Options structs
      → 34 Engine adapters
          → 35 Library hard break
          → 36 CLI redesign
              → 37 Docs, examples, migration
                  → 38 Closure
```

Phases 35 and 36 both depend on 34. Prefer: adapters green → rewrite
in-repo consumers to Document → delete old API → finish CLI → docs →
closure.

| Phase | File | Goal |
|------:|------|------|
| 31 | [phases/phase-31-document-model-contract.md](phases/phase-31-document-model-contract.md) | Freeze Document / ImageDocument public types and policies |
| 32 | [phases/phase-32-content-and-validation.md](phases/phase-32-content-and-validation.md) | Content kinds + Validate invariants |
| 33 | [phases/phase-33-options-structs.md](phases/phase-33-options-structs.md) | Geometry, HF, TOC, image, network as structs |
| 34 | [phases/phase-34-engine-adapters.md](phases/phase-34-engine-adapters.md) | Document → convert / imageout mappers |
| 35 | [phases/phase-35-library-hard-break.md](phases/phase-35-library-hard-break.md) | Remove Converter / dotted Set public surface |
| 36 | [phases/phase-36-cli-redesign.md](phases/phase-36-cli-redesign.md) | New CLI grammar aligned with Document |
| 37 | [phases/phase-37-docs-examples-migration.md](phases/phase-37-docs-examples-migration.md) | Docs, examples, MIGRATION-0.2.4 |
| 38 | [phases/phase-38-closure.md](phases/phase-38-closure.md) | VERSION, CHANGELOG, ledger gates |

---

## Executive Summary

| Fact (current evidence) | Location |
|-------------------------|----------|
| Preferred path is still `PDFRequest` + `RunPDF`, but settings are dotted or fluent wrappers | `api.go`, `documentation/library-api.md` |
| `Converter` + `GlobalSettings.Set("size.pagesize", …)` taught in examples | `examples/pdf/main.go`, `doc.go` |
| `PdfGlobalOptions` covers only common globals; objects still use `Set` / `SetBody` | `api.go` |
| Phase 28 explicitly deferred “full typed `PdfGlobal` for every key” | `plans/0.2.1/phases/phase-28-settings-requests.md` Out of scope |
| Engine already has rich structs (`settings.PdfGlobal`, `PdfObject`, `HeaderFooter`) | `internal/settings/settings.go` |
| CLI is wkhtml multi-object (`page` / `cover` / `toc`) + dotted flags | `internal/cli/`, `documentation/cli.md` |
| Project release is `0.2.3`; `LibraryVersion` remains `0.12.7-dev` settings-surface id | `VERSION`, `api.go` |

---

## Target public API (sketch — freeze in Phase 31)

```go
type Content struct {
    HTML []byte // in-memory HTML document
    Base string // base URL for relative subresources when HTML is set
    File string // local filesystem path
    URL  string // http(s) URL
}

type Margin struct{ Top, Right, Bottom, Left float64 } // millimetres

type HeaderFooter struct {
    Left, Center, Right string
    FontSize            float64
    FontName            string
    Line                bool
    Spacing             float64 // mm
    HTMLURL             string
}

type Page struct {
    Source           Content
    Header, Footer   *HeaderFooter // nil → inherit Document
    IncludeInOutline *bool         // nil → engine default
    ExternalLinks    *bool
    LocalLinks       *bool
}

type TOC struct {
    Caption      string
    DottedLines  *bool
    FontScale    float64
    Indentation  string
    ForwardLinks *bool
    BackLinks    *bool
}

type Document struct {
    Cover *Page
    TOC   *TOC
    Pages []Page

    PageSize    string
    WidthMM     float64
    HeightMM    float64
    Orientation string
    Margin      Margin
    Title       string
    PDFVersion  string
    PDFProfile  string

    Copies          int
    Collate         bool
    Outline         *bool
    OutlineDepth    int
    Background      *bool
    SmartShrinking  *bool
    Compression     *bool
    ResolveRelLinks *bool
    Header, Footer  *HeaderFooter

    AllowLocalFiles bool
    FontPaths       []string
    UseSystemFonts  bool
    Network         *NetworkPolicy

    Now                         func() time.Time
    OnInfo, OnWarn, OnError     func(string)
    OnPhase                     func(string)
    OnProgress                  func(int)
}

func (d *Document) Validate() error
func (d *Document) WritePDF(ctx context.Context, w io.Writer) error
func (d *Document) WritePDFOutline(ctx context.Context, pdf, outline io.Writer) error
func (d *Document) PDF(ctx context.Context) ([]byte, error)

type ImageDocument struct {
    Source      Content
    Width       int
    Height      int
    Format      string // "png" | "jpg"
    Quality     int
    SmartWidth  *bool
    Transparent bool
    Crop        *Crop
    // AllowLocalFiles, Network, Background, Now, hooks — same policy as Document
}

func (d *ImageDocument) Validate() error
func (d *ImageDocument) WriteImage(ctx context.Context, w io.Writer) error
func (d *ImageDocument) Image(ctx context.Context) ([]byte, error)
```

Helpers: `HTML(...)`, `File(...)`, `URL(...)`, `NewDocument(pages ...Page)`.

**Removed in Phase 35:** `Converter`, `ImageConverter`, `ConvertHTML`,
`PDFRequest` / `RunPDF`, `ImageRequest` / `RunImage`, `GlobalSettings`,
`ObjectSettings`, `ImageSettings`, `PdfGlobalOptions`, `NewTOCObject`,
`NewCoverObject`, public dotted `Set`/`Get`.

**Kept:** typed sentinels, `NetworkPolicy` constructors, `LibraryVersion` /
`Version()` (settings-surface id vs project `VERSION` remain distinct).

**CLI target (Phase 36):**

```text
gowkhtmltopdf [global] -o out.pdf page.html
gowkhtmltopdf [global] -o out.pdf --html '<html>…</html>'
gowkhtmltopdf [global] -o out.pdf --url https://example.com/print
gowkhtmltopdf [global] -o book.pdf --cover cover.html --toc page1.html page2.html

gowkhtmltoimage [global] -o out.png page.html
```

---

## Status board

| Phase | Status |
|------:|--------|
| 31 Document model contract | [ ] |
| 32 Content and validation | [ ] |
| 33 Options structs | [ ] |
| 34 Engine adapters | [ ] |
| 35 Library hard break | [ ] |
| 36 CLI redesign | [ ] |
| 37 Docs, examples, migration | [ ] |
| 38 Closure | [ ] |

Update a row only after the phase file’s closure gates record `make lint`
and `make test` (docs-only rows in 37 may note documentation-only proof).

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| v0.2.1–v0.2.3 shipped engine + module path | Starting tree |
| Phase 31 | Stable names for 32–37 |
| Phase 32–33 | Valid Document tree for 34 |
| Phase 34 | Adapters for 35–36 |
| Phase 35–36 | Surface for 37 docs |
| Phase 37 | Evidence for 38 release |

---

## Out of scope

See Overview hard non-goals. Fidelity leftovers remain under
`plans/0.2.0/` deferred / pending ledgers; do not reopen them here.

---

## Closure (ledger)

- [ ] Every phase file 31–38 has its closure section filled
- [ ] No duplicate active work: any older row superseded here is `[~]` with a pointer to this ledger
- [ ] `VERSION` / `CHANGELOG.md` / `documentation/MIGRATION-0.2.4.md` agree
- [ ] Handoff: next unchecked work after v0.2.4 listed in phase 38
