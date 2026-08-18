# Public library API

The v0.2.4 root package is a data-first boundary over the existing pure-Go
engine. It exposes `Document` for PDF jobs and `ImageDocument` for one image
canvas. Internal `settings`, `convert.Request`, and `imageout.Request` types
never appear in the public signatures.

## Boundary

```text
Content{HTML|File|URL}
        │
        ▼
Page / Cover / TOC → Document.Validate → Document.toPDFRequest → convert.Run

Content{HTML|File|URL}
        │
        ▼
ImageDocument.Validate → ImageDocument.toImageRequest → imageout.RunRequest
```

`Content` accepts exactly one source kind. HTML bytes are copied by `HTML`
and again at the mapper boundary; `Base` is legal only for HTML. Empty or
ambiguous sources fail before files, URLs, or output sinks are opened.

## PDF model

```go
doc := gowkhtmltopdf.Document{
    Cover: &gowkhtmltopdf.Page{Source: gowkhtmltopdf.File("cover.html")},
    Pages: []gowkhtmltopdf.Page{{
        Source: gowkhtmltopdf.HTML([]byte("<h1>Report</h1>"), ""),
    }},
    TOC: &gowkhtmltopdf.TOC{Caption: "Contents"},
    PageSize: "A4",
    AllowLocalFiles: true,
}

pdf, err := doc.PDF(ctx)
```

The adapter emits objects in `Cover → TOC → Pages` order. A cover does not
enter the outline by default. Page-level non-nil `Header` and `Footer` values
set the internal override bits; nil inherits the document header/footer.

`Document.WritePDF(ctx, writer)` is the writer-first form. `PDF` returns an
owned byte slice. `WritePDFOutline(ctx, pdfWriter, outlineWriter)` is the only
outline-dump entrypoint and keeps PDF and XML in separate sinks.

## Image model

```go
image := gowkhtmltopdf.ImageDocument{
    Source: gowkhtmltopdf.File("report.html"),
    Width: 1024,
    Format: "png",
    AllowLocalFiles: true,
}
png, err := image.Image(ctx)
```

Image documents have exactly one source and no PDF-only concepts such as
pages, covers, TOC, outlines, copies, or headers. `Width`, `Height`, `Format`,
`Quality`, `SmartWidth`, `Transparent`, and `Crop` map to `ImageGlobal`.

## Validation and errors

| Condition | Result |
|---|---|
| nil `*Document` / `*ImageDocument` | `ErrNilDocument` / `ErrNilImageDocument` |
| no or multiple `Content` source kinds | `ErrInvalidContent` |
| empty HTML | `ErrEmptyHTML` via `errors.Is` |
| TOC-only document | `ErrNoRenderablePDFObjects` |
| invalid page size/version/profile | corresponding typed sentinel |
| missing output writer | `ErrMissingPDFOutput` / `ErrMissingImageOutput` |
| nil context | `ErrNilContext` |

User errors are returned, not panicked. The API has no public dotted setting
tables or compatibility subpackage. `NetworkPolicy` remains available as a
typed policy, and `LibraryVersion` remains the upstream settings-surface id;
the project release is `VERSION`.

## Ownership and hooks

`WritePDF` and `WriteImage` clone source bytes, string slices, maps, and
internal settings before execution. Callers can reuse or mutate the model
after the boundary without changing the engine request. `Now` supplies a
deterministic metadata clock. `OnInfo`, `OnWarn`, `OnError`, `OnPhase`, and
`OnProgress` receive engine events without exposing internal log writers.

## CLI relationship

The command parsers are separate from the root package but construct the same
conceptual model:

```text
gowkhtmltopdf --allow-local-files -o report.pdf report.html
gowkhtmltopdf -o report.pdf --html '<h1>Report</h1>'
gowkhtmltopdf --cover cover.html --toc -o book.pdf chapter-1.html chapter-2.html
gowkhtmltoimage --format png -o report.png report.html
```

The old `page`, `cover`, and `toc` positional object grammar and generic
`--set` escape hatch are rejected. Engine dotted names remain private to
`internal/settings` and the CLI-to-engine adapter.

## Verification

The public boundary is covered by `document_test.go` and
`document_render_test.go`; the complete repository gate is:

```sh
GOCACHE=/tmp/gowkhtmltopdf-go-cache go test ./...
make lint
```
