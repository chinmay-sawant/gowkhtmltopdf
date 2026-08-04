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

`go.mod` has **zero** third-party `require` entries (stdlib only). Pin a
tagged release once the module is published to a reachable path; until then
use `replace` against a checkout.

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

**Pagination / tables:** `<thead>` repeats on continuation pages; `zoom` /
smart-shrinking settings re-layout when wired (same behavior as CLI `--zoom` /
`--smart-shrinking`). Orphan/widow control is heuristics only — see matrix §2.6.

**Fonts / links:** `font-path` / `use-system-fonts` feed the font registry;
`resolve-relative-links` / internal-links behave as in matrix §7.5. Body `#`
GoTo is shipped; HTML HF `#id` fragment GoTo resolves to body destinations (copies-aware).

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
