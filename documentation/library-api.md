# Library API

Module: `gowkhtmltopdf` (import path same as module name).

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

// Set page URL/path and convert — see examples/image for the exact setters
// for your version (mirrors CLI image flags).
```

Worked examples:

- [`examples/pdf`](../examples/pdf/)
- [`examples/image`](../examples/image/)

## Settings model

- Dotted keys: `size.pagesize`, `margin.top`, `header.left`, `load.*`, `web.*`, …
- Unknown names: `Set` returns an error  
- Defaults mirror upstream wkhtmltopdf where implemented  

Authoritative support matrix: [compatibility-matrix.md](compatibility-matrix.md).

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
