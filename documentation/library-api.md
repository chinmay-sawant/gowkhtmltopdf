# Library API

Module: `gowkhtmltopdf` (import path same as module name).

## Install

```sh
# when published / tagged:
go get gowkhtmltopdf@v0.1.0

# local checkout (this repo today):
go mod edit -replace gowkhtmltopdf=/path/to/gowkhtmltopdf
go get gowkhtmltopdf@v0.0.0
```

`go.mod` has two allowlisted direct pure-Go module dependencies:
[`github.com/go-text/typesetting`](https://github.com/go-text/typesetting) for
OpenType shaping and [`github.com/tdewolff/canvas`](https://github.com/tdewolff/canvas)
for SVG rasterization. Their transitive module graph is resolved by Go; this is
not a stdlib-only module and it does not require cgo, a browser, or a native
converter process. Retain the upstream notices for these dependencies and
bundled fonts when preparing a redistribution archive.

Pin a tagged release once the module is published to a reachable path; until
then use `replace` against a checkout.

## PDF converter

```go
c := gowkhtmltopdf.NewConverter()

// Global settings (dotted names, wkhtmltopdf-style)
_ = c.Global().Set("size.pagesize", "A4")
_ = c.Global().Set("orientation", "Portrait")
_ = c.Global().Set("margin.top", "15")
_ = c.Global().Set("title", "Invoice")
_ = c.Global().Set("enablelocalfileaccess", "true")
_ = c.Global().Set("outline", "true")

obj := gowkhtmltopdf.NewObjectSettings().SetPage("invoice.html")
// Pair with global enable for local files (upstream-compatible ACL):
_ = obj.Set("load.blocklocalfileaccess", "false")
c.AddObject(obj)

// Optional callbacks: OnInfo, OnWarn, OnError, OnPhase, OnProgress
if err := c.Convert(context.Background()); err != nil {
	return err
}
pdf := c.Output() // []byte
```

### In-memory HTML (`ConvertHTML`)

```go
pdf, err := gowkhtmltopdf.ConvertHTML(ctx, []byte(`<html><body><h1>Hi</h1></body></html>`), nil)
// optional GlobalSettings for page size / margins / web.images, etc.
```

Does not hide ACL: linked local images/CSS still need the enable +
`blocklocalfileaccess=false` pair on a full `Converter` when those assets
are files.

### Multi-object

Add several objects (cover, TOC-like body pages) in order. TOC as a first-class
CLI object is primarily a CLI feature; from the library, body pages with
headings + global outline/TOC settings cover most report cases. Prefer the
CLI multi-object grammar when you need an explicit `toc` object in the page
sequence.

## Image converter

```go
ic := gowkhtmltopdf.NewImageConverter()
_ = ic.Set("format", "png")
_ = ic.Set("width", "1024")
_ = ic.Global().Set("enablelocalfileaccess", "true") // if available on image settings surface

// Set page URL/path and convert - see examples/image for the exact setters
// for your version (mirrors CLI image flags).
```

Worked examples:

- [`examples/pdf`](../examples/pdf/)
- [`examples/image`](../examples/image/)

## Settings model

- Dotted keys: `size.pagesize`, `margin.top`, `header.left`, `load.*`, `web.*`, …
- `web.images=false` on **global** settings disables `<img>` fetch/paint in PDF and image mode
- Unknown names: `Set` returns an error  
- Defaults mirror upstream wkhtmltopdf where implemented  

**Pagination / tables:** `<thead>` repeats on continuation pages; `position: sticky` clamps to the page content box (print scrollport) within its containing block without fixed-style continuation-page clones, or to a nearest `overflow:auto|scroll|hidden|clip` ancestor at scroll offset 0; `zoom` /
smart-shrinking settings re-layout when wired (same behavior as CLI `--zoom` /
`--smart-shrinking`). CSS `orphans`/`widows` are parsed with Rule 3 when line
boxes exist; geometric heuristic fallback — see matrix §2.6.

**Fonts / links:** `font-path` / `use-system-fonts` feed the font registry;
the image converter honors local `@font-face` TTF/OTF/WOFF1 under the same ACL as
PDF. Text runs use OpenType shaping via `go-text/typesetting` when the face
has GSUB (`ShapeTextFont` / `TextShow`), with presentation-form Arabic as
fallback; CJK punctuation may request OT `halt`/`palt` — see [fonts.md](fonts.md). `resolve-relative-links` / internal-links
behave as in matrix §7.5. Body `#` GoTo is shipped; HTML HF `#id` fragment
GoTo resolves to **body** destinations only (copies-aware). Nested HTML HF is a
child layout (body CSS subset + local `@font-face` under ACL), clipped to the
margin band — see matrix §7.7.

Authoritative support matrix: [compatibility-matrix.md](compatibility-matrix.md).
Fonts details: [fonts.md](fonts.md).

## Local file access (library)

Both are required for a normal local path:

1. Global: `enablelocalfileaccess=true`  
2. Object: `load.blocklocalfileaccess=false`  

This matches the CLI pair `--enable-local-file-access` / blocked-by-default
load policy.

## Errors

- Load failures (network, ACL, HTTP status) wrap underlying errors  
- HTTP 404/401 can surface exit-code helpers for CLI mapping  
- Context cancellation is honored during load/convert  

## Thread safety

A `Converter` instance is not required to be safe for concurrent `Convert`
calls. Create one converter per job or serialize use.

## Security when embedding (Gin / HTTP APIs)

The converter fetches the page and its subresources **from your process**.
That is safe for **HTML you generate**; it is dangerous if you pass
**user-controlled URLs** (SSRF) or enable **local file access** for untrusted
input.

**Preferred:** render your template → convert that file/HTML → return PDF.  
**Avoid:** `convert(c.Query("url"))` for arbitrary users without allowlists
and network isolation.

Same design risk class as **upstream wkhtmltopdf** if you pipe user URLs into
either tool. Details and Gin scenarios:
**[integration-security.md](integration-security.md)** and
**[THREAT-MODEL.md](THREAT-MODEL.md)** §7.1.
