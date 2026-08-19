# Go library API

Module path: `github.com/chinmay-sawant/gowkhtmltopdf`.

> **0.2.4 migration status:** this page documents the complete `Document` /
> `ImageDocument` API. The hard break is implemented in the root package;
> `api.go` no longer exports the v0.2.3 symbols. See
> [MIGRATION-0.2.4.md](MIGRATION-0.2.4.md).

The 0.2.4 library models a document as data first and conversion as one
explicit operation:

~~~text
Content source(s) → Page tree → Document / ImageDocument → WritePDF / WriteImage
~~~

There is no public `Set("dotted.key", value)` escape hatch in the v0.2.4 API.
Engine settings remain an internal implementation detail.

## PDF quick start

The v0.2.4 API uses `Content` to make the source kind explicit. A document may
contain a cover, an optional TOC, and one or more body pages. The render order
is always Cover → TOC → Pages.

~~~go
package main

import (
    "context"
    "os"

    pdf "github.com/chinmay-sawant/gowkhtmltopdf"
)

func main() {
    out, err := os.Create("report.pdf")
    if err != nil {
        panic(err)
    }
    defer out.Close()

    doc := pdf.Document{
        Pages: []pdf.Page{{
            Source: pdf.Content{
                HTML: []byte("<html><body><h1>Report</h1></body></html>"),
            },
        }},
        PageSize: "A4",
        Title:    "Report",
    }

    if err := doc.WritePDF(context.Background(), out); err != nil {
        panic(err)
    }
}
~~~

For an in-memory result, use `PDF`:

~~~go
doc := pdf.Document{
    Pages: []pdf.Page{{Source: pdf.Content{
        HTML: []byte("<h1>Invoice</h1>"),
    }}},
}

pdfBytes, err := doc.PDF(ctx)
~~~

`WritePDF` and `PDF` validate the complete document before loading content
or writing output. `WritePDFOutline(ctx, pdfWriter, outlineWriter)` is the
explicit variant that writes PDF bytes and outline XML to separate sinks.

## Content sources

`Content` has exactly one source kind:

~~~go
type Content struct {
    HTML []byte // in-memory HTML
    Base string // base URL for relative HTML subresources
    File string // local filesystem path
    URL  string // http(s) URL
}
~~~

Use `Base` only with `HTML`. Empty content, multiple source fields, and an
empty `File` or `URL` are validation errors. The convenience helpers are
`HTML(...)`, `File(...)`, and `URL(...)`. Explicit `Content` fields above
remain the stable contract when callers need to avoid helper inference.

Examples:

~~~go
inline := pdf.Page{Source: pdf.Content{
    HTML: []byte("<img src=\"logo.png\"><h1>Invoice</h1>"),
    Base: "https://assets.example.test/invoices/",
}}

local := pdf.Page{Source: pdf.Content{File: "invoice.html"}}
remote := pdf.Page{Source: pdf.Content{URL: "https://example.test/invoice"}}

doc := pdf.Document{
    Pages:           []pdf.Page{inline, local, remote},
    AllowLocalFiles: true,
}
~~~

In-memory HTML is not interpreted as a URL and does not use `inline:` or
`data:` prefixes. Relative resources need a usable `Base` URL (or a local
file base supplied by the adapter).

## Document options

The target `Document` shape is:

~~~go
type Margin struct{ Top, Right, Bottom, Left float64 } // millimetres

type HeaderFooter struct {
    Left, Center, Right string
    FontSize            float64
    FontName            string
    Line                bool
    Spacing             float64 // millimetres
    HTMLURL             string
}

type Page struct {
    Source           Content
    Header, Footer   *HeaderFooter // nil means inherit Document values
    IncludeInOutline *bool         // nil means engine default
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
~~~

Pointer booleans mean “unset; use the engine default.” A plain `bool` is an
explicit value. A page-level non-nil header or footer overrides the document
value; nil means inheritance. `AllowLocalFiles` is one public switch that
maps to the two internal ACL decisions required for a local page and its
subresources.

Common configuration:

~~~go
enabled := true
doc := pdf.Document{
    Pages: []pdf.Page{{Source: pdf.Content{File: "invoice.html"}}},
    PageSize: "A4",
    Margin: pdf.Margin{Top: 15, Right: 12, Bottom: 15, Left: 12},
    Title: "Invoice",
    PDFVersion: "1.7",       // version only; not a conformance claim
    PDFProfile: "a3a-ua1",   // opt-in PDF/A-3a + PDF/UA-1 profile
    Outline: &enabled,
    AllowLocalFiles: true,
}
~~~

`Now` pins PDF creation metadata and `[date]` / `[time]` substitutions.
Callbacks receive informational messages, warnings, errors, phase names, or
progress percentages as specified by their field names.

## Cover, TOC, and outlines

Cover and TOC are document fields, not synthetic object settings:

~~~go
doc := pdf.Document{
    Cover: &pdf.Page{Source: pdf.Content{File: "cover.html"}},
    TOC: &pdf.TOC{
        Caption:      "Contents",
        DottedLines:  boolPtr(true),
        ForwardLinks: boolPtr(true),
        BackLinks:    boolPtr(true),
    },
    Pages: []pdf.Page{
        {Source: pdf.Content{File: "chapter-1.html"}},
        {Source: pdf.Content{File: "chapter-2.html"}},
    },
    AllowLocalFiles: true,
}
~~~

The cover does not inherit headers or footers unless configured explicitly.
The TOC is not itself a renderable body; a document with only `TOC` fails
validation. `WritePDFOutline` is the only API that emits outline XML, and it
requires a dedicated outline writer.

## ImageDocument

Image conversion has one source and one canvas:

~~~go
type ImageDocument struct {
    Source      Content
    Width       int
    Height      int
    Format      string // "png" or "jpg"
    Quality     int
    SmartWidth  *bool
    Transparent bool
    Crop        *Crop

    AllowLocalFiles bool
    Network         *NetworkPolicy
    // Background, Now, and the conversion hooks use the same policy as Document.
}
~~~

~~~go
imageDoc := pdf.ImageDocument{
    Source: pdf.Content{HTML: []byte("<h1>Badge</h1>")},
    Width:  1024,
    Format: "png",
}

pngBytes, err := imageDoc.Image(ctx)
~~~

`WriteImage` is the writer-first form. Image mode rejects empty or multi-source
content and has no pages, cover, TOC, outline, or copies.

## Validation and errors

Call `Validate` when a caller needs to report configuration errors before
starting conversion. `WritePDF`, `PDF`, `WriteImage`, and `Image` call it
themselves.

| Situation | Target behavior |
|---|---|
| Nil `*Document` receiver | Return a typed nil-document sentinel, such as `ErrNilDocument` |
| Nil `*ImageDocument` receiver | Return a typed nil-image-document sentinel, such as `ErrNilImageDocument` |
| No renderable page or cover | Return `ErrNoRenderablePDFObjects`-compatible error |
| TOC without a body or cover | Reject; a TOC is not renderable by itself |
| Zero or multiple `Content` source kinds | Return `ErrInvalidContent`-compatible error |
| Empty HTML | Return `ErrEmptyHTML`-compatible error |
| Invalid nested page | Return an error identifying `pages[i]` or `cover` |
| `Copies < 1` when explicitly set | Return `ErrInvalidPDFCopies` |
| Missing PDF writer | Return `ErrMissingPDFOutput` |
| Missing image writer | Return `ErrMissingImageOutput` |
| Invalid page size/version/profile | Return an error matching the corresponding typed sentinel |
| Nil fluent helper receiver | Panic only where a programmer-broken fluent helper requires it; user input errors return `error` |

All user-controlled values fail through `Validate` or `Write*`; they should
not be expected to panic. Use `errors.Is` for sentinels rather than matching
error strings.

## Ownership and concurrency

The write boundary owns a snapshot of the document. HTML bytes, slices, and
maps supplied by the caller are copied before the engine runs, so mutating a
`Document` after `WritePDF` or `WriteImage` starts cannot change that job.
Use one document instance per concurrent conversion unless the caller
provides its own synchronization.

## Network and local-file policy

Local files are denied by default. Set `AllowLocalFiles: true` only for
trusted input and constrain the process or working directory as appropriate.
For remote documents and subresources, `Network` optionally points to a
`NetworkPolicy`:

~~~go
doc := pdf.Document{
    Pages: []pdf.Page{{Source: pdf.Content{
        URL: "https://reports.example.test/monthly",
    }}},
    Network: &policy,
}
~~~

`CompatibleNetworkPolicy()` preserves historical permissive URL behavior;
`RestrictedNetworkPolicy()` blocks private destinations and cross-host
redirects. See [integration-security.md](integration-security.md) and
[THREAT-MODEL.md](THREAT-MODEL.md) before embedding untrusted URLs.

## Versioning

`LibraryVersion` remains the wkhtmltopdf compatibility identifier (`0.12.x`)
and is not the project release. The project release is in `VERSION`.

`PDFVersion` selects an output version (`1.4`, `1.7`, or `2.0`). A version is
not a PDF/A or PDF/UA claim. `PDFProfile` selects the opt-in profile, such as
`a3a-ua1` or `a4-ua2`, and implies the required PDF version.

## Removed public surface in 0.2.4

The hard break removes `Converter`, `ImageConverter`, `ConvertHTML`,
`PDFRequest` / `RunPDF`, `ImageRequest` / `RunImage`, `GlobalSettings`,
`ObjectSettings`, `ImageSettings`, `PdfGlobalOptions`, `NewTOCObject`,
`NewCoverObject`, `SetPage`, `SetBody`, and public dotted `Set` / `Get` APIs.
There is no `compat` package in 0.2.4. Use the
[migration guide](MIGRATION-0.2.4.md) to move each old operation.

## Internal settings appendix

The engine and the migration-period CLI may still use dotted names such as
`size.pagesize`, `margin.top`, and `load.blocklocalfileaccess` internally.
Those names are not the 0.2.4 library contract and should not appear in new
application code.
