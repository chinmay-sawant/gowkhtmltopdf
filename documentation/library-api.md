# Library API

Module path: `gowkhtmltopdf`.

The root package converts HTML to PDF and to raster images. There is no cgo,
no browser process, and no native converter binary. Direct module
dependencies (see [`go.mod`](../go.mod)) are
[`github.com/go-text/typesetting`](https://github.com/go-text/typesetting)
(OpenType shaping) and
[`github.com/tdewolff/canvas`](https://github.com/tdewolff/canvas) (SVG
rasterization). Their transitive graph is resolved by Go. Retain the
upstream notices for those modules and the bundled fonts when redistributing.

This page is the user-facing contract for `api.go` / `doc.go`. It does not
invent methods. CLI flags and exit codes live in [cli.md](cli.md).

## Install

Requires Go **1.26+** (the module pins `toolchain go1.26.4`). Build with
`CGO_ENABLED=0`.

```sh
# when published / tagged:
go get gowkhtmltopdf@v0.2.1

# local checkout (this repo today):
go mod edit -replace gowkhtmltopdf=/path/to/gowkhtmltopdf
go get gowkhtmltopdf@v0.0.0
```

Pin a tagged release once the module is published to a reachable path; until
then use `replace` against a checkout.

Worked programs that call only the public API:

- [`examples/pdf`](../examples/pdf)
- [`examples/image`](../examples/image)

First-time CLI walkthrough: [getting-started.md](getting-started.md).

## Versioning

Two version strings exist on purpose. Do not treat them as the same number.

| Symbol | Value | Meaning |
|--------|-------|---------|
| `LibraryVersion` | `"0.12.7-dev"` | wkhtmltopdf **0.12.x settings-surface** compatibility id |
| `Version()` | `"0.12.7-dev (gowkhtmltopdf pure-go)"` | library banner (`LibraryVersion` plus a suffix) |
| `VERSION` file | `0.2.1` | **project release**, stamped into the CLI at build time |

`LibraryVersion` is not the project release. The CLI `--version` line comes
from `VERSION`, not from `LibraryVersion`.

```go
package main

import (
	"fmt"

	gowkhtmltopdf "gowkhtmltopdf"
)

func main() {
	fmt.Println(gowkhtmltopdf.LibraryVersion)
	fmt.Println(gowkhtmltopdf.Version())
}
```

## Which API to pick

There are two public faces. Prefer the typed writer-first pair when embedding.

| Face | Types | Output | Clock | Outline dump |
|------|-------|--------|-------|----------------|
| **Typed (preferred)** | `PDFRequest` + `RunPDF`; `ImageRequest` + `RunImage` | caller `io.Writer` | optional `Now` | `PDFRequest.OutlineOutput` required when `dumpoutline=true` |
| **Compatibility driver** | `Converter`; `ImageConverter` | `Convert` buffers for `Output()`; `ConvertTo` writes a sink | always wall clock | **not available** (`ConvertTo` always passes a nil outline sink) |

Use the typed API when you already have a writer, when you need a pinned
clock for PDF `/CreationDate` and `[date]`/`[time]`, or when you need
`--dump-outline` XML. Use `Converter` / `ImageConverter` when you want the
wkhtmltopdf-shaped `Set`/`Get` + `Output()` lifecycle, progress hooks, or
the examples under [`examples/pdf`](../examples/pdf) /
[`examples/image`](../examples/image).

`ConvertHTML` is a one-shot helper on top of `Converter`. It is convenient
for a single in-memory document with an empty base URL; it is not the
preferred embedding path.

`Now` exists **only** on `PDFRequest` and `ImageRequest`. `Converter` and
`ImageConverter` always use the wall clock.

There is **no** public `HttpErrorCode()` on any library type. HTTP 404 →
exit 2 and 401 → exit 3 are [CLI-only](cli.md#exit-codes).

## Typed PDF (`PDFRequest`, `RunPDF`)

```go
type PDFRequest struct {
    Global        *GlobalSettings
    Objects       []*ObjectSettings
    Now           func() time.Time
    Output        io.Writer
    OutlineOutput io.Writer
}
```

| Field | Role |
|-------|------|
| `Global` | optional; `nil` uses `NewGlobalSettings()` defaults when `RunPDF` clones the request |
| `Objects` | page / cover / TOC objects, in document order |
| `Now` | optional clock for PDF metadata and `[date]`/`[time]`; `nil` → `time.Now` |
| `Output` | required document sink |
| `OutlineOutput` | required **only** when `dumpoutline` is true on `Global` |

`RunPDF(ctx, req)` validates, clones settings into an internal request, and
writes the PDF to `req.Output`. Cancel `ctx` to abort. `req` must not be
`nil` (`ErrNilPDFRequest`, distinct from `ErrNilConverter`). `ctx` must not
be `nil` (`ErrNilContext`).

`ValidatePDF()` checks the same contract **without** starting the renderer.
It is safe to call before opening files. Failures:

| Condition | Sentinel |
|-----------|----------|
| `req == nil` | `ErrNilPDFRequest` |
| `Output == nil` | `ErrMissingPDFOutput` (alias of `convert.ErrMissingOutput`) |
| `dumpoutline=true` and `OutlineOutput == nil` | `ErrMissingPDFOutlineOutput` |
| `copies < 1` | `ErrInvalidPDFCopies` |
| no renderable body (empty list, empty object, or TOC-only) | `ErrNoRenderablePDFObjects` |

Engine failures after a successful preflight are wrapped as `pdf: %w`.
Preflight sentinels from `ValidatePDF` are returned **unwrapped**.

`RunPDF` clones global and object settings before the engine starts. Mutating
`req.Global` or an object after `RunPDF` returns does not change bytes that
were already written.

### Example: `RunPDF` with `SetBody` + `Now`

```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	gowkhtmltopdf "gowkhtmltopdf"
)

func main() {
	global := gowkhtmltopdf.NewGlobalSettings()
	if err := global.Set("size.pagesize", "A4"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var out bytes.Buffer
	err := gowkhtmltopdf.RunPDF(context.Background(), &gowkhtmltopdf.PDFRequest{
		Global: global,
		Objects: []*gowkhtmltopdf.ObjectSettings{
			gowkhtmltopdf.NewObjectSettings().SetBody(
				[]byte(`<html><body><h1>Invoice</h1><p>[date]</p></body></html>`),
				"",
			),
		},
		Now:    func() time.Time { return time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) },
		Output: &out,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile("invoice.pdf", out.Bytes(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

`Now` pins `/CreationDate` and the `[date]` / `[time]` header-footer
placeholders. Without it the writer uses the wall clock. `Converter` cannot
override that clock.

`SetBody(html, base)` marks an **inline document** source: no URL guessing,
and `base` resolves relative `<link>` / `<img>` / `<a>`. An empty base
leaves relative subresources unresolvable. The helper copies `html`; later
mutation of the caller's slice does not change the object.

### Dump outline

`--dump-outline` is a **typed-request** feature. Set the registered global
key and give `OutlineOutput` its own writer. The outline XML is never
appended to the PDF stream.

```go
global := gowkhtmltopdf.NewGlobalSettings()
_ = global.Set("dumpoutline", "true")

var pdfOut, outlineOut bytes.Buffer
err := gowkhtmltopdf.RunPDF(ctx, &gowkhtmltopdf.PDFRequest{
	Global:        global,
	Objects:       []*gowkhtmltopdf.ObjectSettings{body},
	Output:        &pdfOut,
	OutlineOutput: &outlineOut,
})
```

`Converter.Convert` / `ConvertTo` always pass a nil outline sink, so
`dumpoutline=true` on a converter fails with `ErrMissingPDFOutlineOutput`
wrapped as `convert: %w`. There is no converter field for an outline writer.

## Typed image (`ImageRequest`, `RunImage`)

```go
type ImageRequest struct {
    Global *GlobalSettings
    Image  *ImageSettings
    Object *ObjectSettings
    Now    func() time.Time
    Output io.Writer
}
```

Image mode renders **one** input document onto one canvas (PNG or JPEG).

| Field | Role |
|-------|------|
| `Global` | optional; loader ACL (`enablelocalfileaccess`, `allow`) and `SetNetworkPolicy` |
| `Image` | optional; `nil` uses `NewImageSettings()` defaults |
| `Object` | the single page (`SetPage` or `SetBody`) |
| `Now` | optional typed clock (same field as PDF; converter drivers have no equivalent) |
| `Output` | required encoded-image sink |

`RunImage(ctx, req)` validates, clones, and writes encoded bytes to
`req.Output`. `req` must not be `nil` (`ErrNilImageRequest`, distinct from
`ErrNilImageConverter`).

`ValidateImage()` checks the contract without rendering:

| Condition | Sentinel |
|-----------|----------|
| `req == nil` | `ErrNilImageRequest` |
| `Output == nil` | `ErrMissingImageOutput` |
| missing / empty / TOC-only object | `ErrNoInputPageAdded` |

Engine failures after preflight are wrapped as `image: %w`. Preflight
sentinels are unwrapped.

`ImageSettings` is the typed image-mode surface (`NewImageSettings`, `Set`,
`Get`). Keys include `width`, `height`, `quality`, `smartwidth`,
`transparent`, `format` (`png` / `jpg`), and `crop.left` / `crop.top` /
`crop.width` / `crop.height`. Defaults match wkhtmltoimage: viewport width
1024, quality 94, smart-width on, PNG when format is left empty.

`web.images` and the background paint switch belong on **image** settings in
this mode (`ImageSettings.Set` or `ImageConverter.Set`), not on PDF object
settings. `background` and `web.background` are aliases for the shared body
paint flag.

```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	gowkhtmltopdf "gowkhtmltopdf"
)

func main() {
	img := gowkhtmltopdf.NewImageSettings()
	if err := img.Set("format", "png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := img.Set("width", "800"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var out bytes.Buffer
	err := gowkhtmltopdf.RunImage(context.Background(), &gowkhtmltopdf.ImageRequest{
		Image:  img,
		Object: gowkhtmltopdf.NewObjectSettings().SetBody(
			[]byte(`<html><body><h1>Card</h1></body></html>`),
			"",
		),
		Output: &out,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile("card.png", out.Bytes(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

## Converter (PDF compatibility driver)

`Converter` is the wkhtmltopdf-shaped driver: configure with
`Global()` / `AddObject`, then `Convert` or `ConvertTo`.

```go
c := gowkhtmltopdf.NewConverter()
_ = c.Global().Set("size.pagesize", "A4")
c.AddObject(gowkhtmltopdf.NewObjectSettings().SetPage("report.html"))
if err := c.Convert(ctx); err != nil {
    return err
}
pdf := c.Output()
```

| Method | Behavior |
|--------|----------|
| `NewConverter()` | default global settings, no objects |
| `Global() *GlobalSettings` | live global settings; lazy-inits if nil |
| `WithGlobal(s) *Converter` | replaces globals with a **clone** of `s`; nil receiver or nil `s` is a no-op |
| `AddObject(s) *Converter` | appends a **deep copy** of `s`; nil `s` is ignored |
| `AddHTML(page, baseURL)` | `AddObject(NewObjectSettings().SetBody(page, baseURL))` |
| `Convert(ctx)` | `ConvertTo` into an internal buffer, then stores bytes for `Output()` |
| `ConvertTo(ctx, w)` | writes the PDF to `w`; **does not** populate `Output()` |
| `Output() []byte` | copy of the last successful `Convert` result, or nil |

A converter needs at least one **renderable** object (`SetPage` / `SetBody`
on a non-TOC object). Otherwise `Convert` / `ConvertTo` return
`ErrNoRenderablePDFObjects`. A nil receiver returns `ErrNilConverter`. A
nil writer to `ConvertTo` returns `ErrMissingPDFOutput`.

`AddObject` copies nested maps, slices, and `SetBody` bytes. Mutating the
original `*ObjectSettings` after `AddObject` does not change the converter.
`Convert` / `ConvertTo` snapshot global and object settings again at the
start of the run, so mutations of `Global()` after `AddObject` **do** apply
to the **next** convert, not to an in-flight one.

There is no `Now` on `Converter`. There is no outline writer. There is no
`HttpErrorCode()`.

### Hooks

```go
c.OnInfo = func(line string) { /* … */ }
c.OnWarn = func(line string) { /* … */ }
c.OnError = func(line string) { /* … */ }
c.OnPhase = func(phase string) { /* "Loading pages (1/1)", "Done", … */ }
c.OnProgress = func(percent int) { /* 0–100; last value is 100 */ }
```

Log lines are classified by marker prefix (`warning:` / `warn:` → `OnWarn`,
`error:` / `err:` → `OnError`, everything else including phase text →
`OnInfo`). Preflight and engine errors are also delivered to `OnError` when
set. Hooks may be nil.

Engine failures from `Convert` / `ConvertTo` are wrapped as `convert: %w`.

### Example: `ConvertHTML`

`ConvertHTML(ctx, html, global)` converts one in-memory document and returns
PDF bytes. `global` may be nil (defaults apply). The **base URL is always
`""`** — there is no parameter for it. Use `AddHTML` / `SetBody` when
relative subresources need a base.

```go
package main

import (
	"context"
	"fmt"
	"os"

	gowkhtmltopdf "gowkhtmltopdf"
)

func main() {
	pdf, err := gowkhtmltopdf.ConvertHTML(
		context.Background(),
		[]byte(`<html><body><h1>Hi</h1></body></html>`),
		nil,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile("hi.pdf", pdf, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

Empty `html` returns `ErrEmptyHTML`. A nil context returns `ErrNilContext`.
`ConvertHTML` does **not** relax the local-file ACL: linked local CSS/images
still need the enable + unblock pair on a full `Converter` / `PDFRequest`.

Optional globals (page size, margins, `web.images`, …) are cloned via
`WithGlobal` at the start of the helper.

### Example: Converter local-file ACL pair

Matches [`examples/pdf`](../examples/pdf): global enable **and** object
`load.blocklocalfileaccess=false`. Both are required for a normal local
path.

```go
package main

import (
	"context"
	"fmt"
	"os"

	gowkhtmltopdf "gowkhtmltopdf"
)

func main() {
	c := gowkhtmltopdf.NewConverter()
	if err := c.Global().Set("size.pagesize", "A4"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := c.Global().Set("orientation", "portrait"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := c.Global().Set("enablelocalfileaccess", "true"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	obj := gowkhtmltopdf.NewObjectSettings().SetPage("invoice.html")
	if err := obj.Set("load.blocklocalfileaccess", "false"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	c.AddObject(obj)

	if err := c.Convert(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile("invoice.pdf", c.Output(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

## ImageConverter

`ImageConverter` is the wkhtmltoimage-shaped driver. It renders **one**
page. `AddObject` takes a **string** path/URL/`inline:`/`data:` source, not
an `*ObjectSettings`. There is no `AddHTML`. There is no `OnPhase` /
`OnProgress`.

```go
ic := gowkhtmltopdf.NewImageConverter() // 1024 px smart-width, PNG
_ = ic.Set("format", "png")
_ = ic.Set("width", "1024")
ic.AddObject("page.html")
_ = ic.Object().Set("load.blocklocalfileaccess", "false")
```

| Method | Behavior |
|--------|----------|
| `NewImageConverter()` | defaults; `Object()` is valid immediately with an empty page |
| `Set(name, value)` | image-mode keys (see [Settings keys](#settings-keys)); unknown names error |
| `Global() *GlobalSettings` | loader ACL (`enablelocalfileaccess`, `allow`) and `SetNetworkPolicy` |
| `AddObject(page string)` | **replaces** the current page (most recent wins) with a fresh object |
| `Object() *ObjectSettings` | **live** pointer to the current page; set `load.*` here |
| `Convert(ctx)` | buffers encoded bytes for `Output()` |
| `ConvertTo(ctx, w)` | writes to `w`; **does not** populate `Output()` |
| `Output() []byte` | copy of the last successful `Convert` result, or nil |

`Set` routes through the image key table. `background` / `web.background`
update the shared body-paint switch on `Global`. `web.images` is set here
(or on `ImageSettings`), not as a PDF object key.

Hooks: `OnInfo`, `OnWarn`, `OnError` only. A nil receiver returns
`ErrNilImageConverter`. A convert with no page / no `SetBody` returns
`ErrNoInputPageAdded`. A nil writer to `ConvertTo` returns
`ErrMissingImageOutput`. Engine failures wrap as `image convert: %w`.

Because `AddObject` constructs a **new** `ObjectSettings`, set
`load.blocklocalfileaccess` **after** `AddObject`, via `Object()`.

A zero-value `&ImageConverter{}` lazy-inits on first use. Prefer
`NewImageConverter()`.

### Example: `ImageConverter` + `SetBody`

```go
package main

import (
	"context"
	"fmt"
	"os"

	gowkhtmltopdf "gowkhtmltopdf"
)

func main() {
	ic := gowkhtmltopdf.NewImageConverter()
	if err := ic.Set("format", "png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := ic.Set("width", "400"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ic.Object().SetBody(
		[]byte(`<html><body><div style="width:80px;height:40px;background:#112233"></div></body></html>`),
		"",
	)

	if err := ic.Convert(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile("card.png", ic.Output(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

### Example: `ImageConverter` local file

Matches [`examples/image`](../examples/image): `AddObject` takes the path
string; the ACL pair is global enable plus `Object().Set` after add.

```go
package main

import (
	"context"
	"fmt"
	"os"

	gowkhtmltopdf "gowkhtmltopdf"
)

func main() {
	c := gowkhtmltopdf.NewImageConverter()
	if err := c.Set("format", "png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := c.Set("width", "1024"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := c.Global().Set("enablelocalfileaccess", "true"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	c.AddObject("invoice.html")
	if err := c.Object().Set("load.blocklocalfileaccess", "false"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := c.Convert(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile("invoice.png", c.Output(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

## Settings keys

Global and object settings use wkhtmltopdf-style **dotted names**. `Set`
returns an error for unknown names. Known **inert** keys (Policy A: accepted
for script compatibility, no engine consumer) succeed and can be read back
with `Get`. `Get` returns `(value, false)` when the name is unknown.

Keys are normalized: trim space, lowercase. Booleans accept
`true`/`1`/`yes`/`on` (and empty → true) and `false`/`0`/`no`/`off`. `Get`
prints booleans as `"true"`/`"false"`, enums in canonical form
(`Portrait`, `print`, `abort`, …), and floats with the shortest round-trip
representation.

Public wrappers:

| Type | Construct | Write | Read |
|------|-----------|-------|------|
| `GlobalSettings` | `NewGlobalSettings()` | `Set`, `SetNetworkPolicy`, `EnableLocalFileAccess` | `Get` |
| `ObjectSettings` | `NewObjectSettings()`, `NewTOCObject()`, `NewCoverObject()` | `Set`, `SetPage`, `SetBody`, `EnableLocalFileAccess` | `Get` |
| `ImageSettings` | `NewImageSettings()` | `Set` | `Get` |
| `PdfGlobalOptions` | `NewPdfGlobalOptions()` | fluent `With*` / `WithSetting` | `Build()` → `*GlobalSettings` |

### Nil, Error, and Panic Policy

| Condition | Behavior | Reason |
|-----------|----------|--------|
| Nil fluent builder receiver (`(*PdfGlobalOptions)(nil).With*`) | **panic** | Programmer-broken builder chain |
| Nil mutator receiver (`(*GlobalSettings)(nil).Set`) | return `ErrNil*` | Typed sentinel for callers |
| Nil chained mutator receiver (`(*ObjectSettings)(nil).SetPage`, `(*Converter)(nil).AddHTML`) | return `nil` | Safe chained mutator pattern |
| Invalid configuration value (`pageSize`, `copies < 1`) | return error at `ValidatePDF` / `RunPDF` / `Convert` / `WithSetting` | Handled at validation boundaries with `errors.Is` |
| Missing output sink (`PDFRequest.Output == nil`) | return `ErrMissingPDFOutput` | Validation failure |
| Local file access denied | return `load.ErrAccessDenied` | Security ACL |

`GlobalSettings.Set("size.pagesize", …)` and `PdfGlobalOptions.WithPageSize`
accept names from the canonical page-size table (A0–A6, B0–B6, C5E, Comm10E,
DLE, Executive, Folio, Ledger, Legal, Letter, Tabloid). Invalid names fail
validation with `ErrInvalidPageSize`. An empty `size.pagesize` at the public `Set` seam
becomes `A4`. `copies < 1` fails validation with `ErrInvalidPDFCopies`. Negative margins are
valid (header/footer band reservation).

Length keys (`margin.*`, `size.width`, `size.height`) accept unit suffixes
(`mm`, `cm`, `in`, `pt`, `px`, …). A bare number is millimetres.

### `PdfGlobalOptions` fluent builder

Typed alternative to string `Set` for common PDF globals. A nil receiver
**panics** (`gowkhtmltopdf: nil PdfGlobalOptions`) — that is a programmer
error. Setting methods store values without panicking; invalid values fail at
`RunPDF` / `ValidatePDF` / `WithSetting` with typed sentinels.

`Build()` returns an independent `*GlobalSettings` snapshot. Pass it as
`PDFRequest.Global`, to `ConvertHTML`, or to `Converter.WithGlobal`. Those
entry points copy again at their ownership boundary.

```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	gowkhtmltopdf "gowkhtmltopdf"
)

func main() {
	opts := gowkhtmltopdf.NewPdfGlobalOptions().
		WithPageSize("Letter").
		WithPDFVersion("2.0"). // optional: "1.4" (default), "1.7", or "2.0" — version alone is not a PDF/A / PDF/UA claim
		WithPDFProfile("a4-ua2"). // optional: "a3a-ua1"/"a3a"/"ua1" (PDF 1.7) or "a4-ua2"/"a4"/"ua2" (PDF 2.0)
		WithMargins(15, 15, 20, 15). // top, right, bottom, left (mm)
		WithTitle("Quarterly report").
		WithCopies(1, true).
		WithOutline(true, 4).
		WithSmartShrinking(true).
		WithBackground(true).
		WithCompression(true).
		WithResolveRelativeLinks(true)
	opts, err := opts.WithSetting("header.left", "Acme")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	global := opts.Build()

	var out bytes.Buffer
	err = gowkhtmltopdf.RunPDF(context.Background(), &gowkhtmltopdf.PDFRequest{
		Global: global,
		Objects: []*gowkhtmltopdf.ObjectSettings{
			gowkhtmltopdf.NewObjectSettings().SetBody(
				[]byte(`<html><body><h1>Report</h1></body></html>`),
				"",
			),
		},
		Output: &out,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = os.WriteFile("report.pdf", out.Bytes(), 0o644)
}
```

`WithMargins(top, right, bottom, left)` takes millimetres as `float64`, not
unit strings.

### Global keys (`GlobalSettings.Set` / `Get`)

Defaults from `NewGlobalSettings()`: A4, Portrait, 10 mm margins, 1 collated
copy, outline on (depth 4), compression on, smart shrinking on, background
on, images on, resolve-relative-links on.

**Page geometry**

| Key | Notes |
|-----|--------|
| `size.pagesize` | named size; validated at this seam |
| `size.width`, `size.height` | custom mm; both `> 0` override the named size |
| `orientation` | `portrait` / `landscape` → Get `Portrait` / `Landscape` |
| `margin.top`, `margin.bottom`, `margin.left`, `margin.right` | mm (units accepted) |

**Document**

| Key | Notes |
|-----|--------|
| `title` | PDF `/Title` (HTML `<title>` feeds `[doctitle]` only) |
| `pdfversion` | PDF output version: `"1.4"` (default), `"1.7"`, or `"2.0"`. `1.7` emits `%PDF-1.7`, trailer `/ID`, Info with UTF-16BE + BOM strings, and XMP Metadata stream. `2.0` emits `%PDF-2.0`, trailer `/ID`, UTF-8 document strings, and XMP metadata stream. |
| `pdfprofile` | Conformance claim (version alone is not). Set accepts short aliases (`"a3a"`, `"ua1"`, `"a3a-ua1"`, `"a4"`, `"ua2"`, `"a4-ua2"`) and the canonical names. **`Get("pdfprofile")` always returns the canonical token:** `"PDF/A-3a"`, `"PDF/UA-1"`, `"PDF/A-3a+PDF/UA-1"`, `"PDF/A-4"`, `"PDF/UA-2"`, or `"PDF/A-4+PDF/UA-2"` (empty when unset). Implies the required base version (`1.7` for A-3a/UA-1, `2.0` for A-4/UA-2). Profile + explicit wrong version fails with `ErrConformanceRequiresPDF17` / `ErrConformanceRequiresPDF20`. Tagged lists nest `L` → `LI` → `LBody` → `Link`. |
| `copies`, `collate` | copies must be ≥ 1 |
| `pageoffset` | integer page offset |
| `grayscale`, `colormode` | both write the same grayscale bit (`color` / `grayscale`) |
| `usecompression` | PDF stream compression |
| `smartshrinking` | may re-layout when content is wider than the page |
| `background`, `web.background` | **same** body-paint switch |
| `outline`, `outlinedepth` | PDF bookmarks |
| `excludefromoutline` | append-only heading filter |
| `dumpoutline` | emit outline XML; needs `PDFRequest.OutlineOutput` |
| `dumpoutlinewithdefaulttocxsl` | dump built-in TOC XSL description |
| `quiet` | suppress progress noise |

**Load / fonts / links**

| Key | Notes |
|-----|--------|
| `enablelocalfileaccess` | global half of the local-file ACL pair |
| `allow` | append-only ACL path prefix |
| `proxy` | HTTP(S) proxy for the loader |
| `fontpath` | append-only TTF search directory; see [fonts.md](fonts.md) |
| `usesystemfonts` | opt-in OS font dirs |
| `resolverelativelinks` | resolve relative `href` against the page URL |

**Header / footer** (`header.*` and `footer.*`)

`fontsize`, `fontname`, `left`, `right`, `center`, `line`, `spacing`,
`htmlurl`.

**TOC defaults** (`toc.*`, used when a TOC object is present)

`fontscale`, `indentation`, `dottedlines`, `captiontext`, `forwardlinks`,
`backlinks`, `xslstylesheet`.

**Web** (`web.*` on **global** for PDF)

| Key | Notes |
|-----|--------|
| `web.images` | PDF `<img>` fetch/paint. **This is the PDF control.** |
| `web.printmediatype`, `web.mediatype` | media selection (`print` / `screen` / `ignore`) |
| `web.simplifydom`, `web.simplifydomprofile` | opt-in chrome-strip |
| `web.printlinkunderline` | opt-in underline `a[href]` |

On PDF, `web.images=false` is a **global** setting. On image mode set it
through `ImageSettings` / `ImageConverter.Set`.

**Accepted but inert** (stored, no engine consumer): `dpi`, `resolution`,
`imagedpi`, `imagequality`, `lowquality`, `usexserver`, `readargsfromstdin`,
`log-level`, `loglevel`, `cookiejar`, `defaultencoding`, `produceforms`,
`loaderrorhandling` (global), `web.javascript`, `web.java`, `web.plugins`,
`web.minimumfontsize`, `web.defaultencoding`, `web.userstylesheet`,
`web.loadimages`.

Network policy is **not** a dotted key. Use `SetNetworkPolicy`.

### Object keys (`ObjectSettings.Set` / `Get`)

Defaults from `NewObjectSettings()`: external/local links on, include in
outline on, use outline on, **`load.blocklocalfileaccess=true`**.

| Key | Notes |
|-----|--------|
| `page` | path, URL, `inline:…`, or `data:…` (prefer `SetPage`) |
| `externallinks`, `locallinks` | URI / internal-link gates |
| `includeinoutline`, `useoutline` | outline membership |
| `istableofcontent` | TOC object (prefer `NewTOCObject`) |
| `iscover` | cover object (prefer `NewCoverObject`) |

**Load** (`load.*`)

`zoomfactor`, `blocklocalfileaccess`, `loaderrorhandling` (`abort` / `skip`
/ `ignore`), `username`, `password`, `mediatype`, `printmediatype`,
`timeout` (seconds; 0 = default).

There is no dotted `Set` for cookie maps, custom-header maps, or POST
fields; those exist on the CLI (`--cookie`, `--custom-header`, `--post`).

**Object header / footer / toc / web** use the same dotted suffixes as
global. Setting `header.*` / `footer.*` on an object sets the override bits
so object HF replaces global HF for that page.

Object `web.images` is a registered key. The PDF pipeline reads
**`GlobalSettings` `web.images`**. Do not use the object key as the PDF
image switch.

**Accepted but inert** on objects: `pagescount`, `produceforms`,
`load.jsdelay`, `load.stopslowscripts`, `load.debugjavascript`,
`load.windowstatus`, `load.runscript`, `load.enableplugins`,
`load.defaultencoding`, `load.proxy`, `load.externallinks`,
`load.locallinks`, `load.repeatexternalheaders`,
`load.repeatexternalcookies`, `web.background`, `web.javascript`,
`web.java`, `web.plugins`, `web.minimumfontsize`, `web.defaultencoding`,
`web.userstylesheet`, `web.loadimages`.

`SetPage(page)` and `SetBody(html, base)` are the documented source
setters. `SetBody` clears `page` and copies the bytes.

### Image keys (`ImageConverter.Set`, `ImageSettings.Set` / `Get`)

| Key | Notes |
|-----|--------|
| `width`, `height` | CSS-pixel viewport; height `0` = content height |
| `quality` | JPEG quality (default 94) |
| `smartwidth` | default true |
| `transparent` | PNG transparency |
| `format` | `png` or `jpg` (empty = sniff / PNG on the converter) |
| `crop.left`, `crop.top`, `crop.width`, `crop.height` | `-1` = unset |
| `proxy` | image-mode proxy field |
| `web.images` | image-mode `<img>` fetch/paint |
| `web.printmediatype`, `web.mediatype` | media selection (image default media is `screen`) |
| `web.simplifydom`, `web.simplifydomprofile`, `web.printlinkunderline` | same meanings as PDF |
| `background`, `web.background` | aliases → shared body-paint switch |

`ImageConverter.Set` error wrap: `image set %q: %w`. The same prefix is
used by `ImageSettings.Set`. Inert global keys listed above are also
accepted on the image table.

`ImageConverter` has no `Get`. Read back through `ImageSettings` on the
typed path, or keep the values you just set.

## Local files

Local file reads are **denied by default**. A normal filesystem path needs
**both**:

1. Global: `enablelocalfileaccess=true`
2. Object: `load.blocklocalfileaccess=false`

That is the same pair as CLI `--enable-local-file-access` /
blocked-by-default load policy. One flag alone is not enough.

`allow` (repeatable) adds ACL prefixes. `ConvertHTML` and `SetBody` do
**not** relax the ACL for linked files: an inline document can still
reference local CSS/images, and those fetches still need the pair (and a
non-empty base if the references are relative).

Remote `http`/`https` pages do not need the local-file pair. They are
subject to [network policy](#network-policy) instead.

## Network policy

```go
type NetworkPolicy struct {
    AllowedSchemes          []string
    AllowedHosts            []string
    BlockPrivateNetworks    bool
    BlockCrossHostRedirects bool
}
```

| Constructor | Behavior |
|-------------|----------|
| `CompatibleNetworkPolicy()` | `http`/`https`, any host, private destinations allowed, cross-host redirects allowed. Historical default when nothing is set. |
| `RestrictedNetworkPolicy()` | `http`/`https`, **block private/loopback**, **same-origin redirects** |

`GlobalSettings.SetNetworkPolicy` copies the scheme and host slices and
marks the policy as explicit. Callers that never call it keep compatible
loader behavior.

An empty `AllowedHosts` list permits any host allowed by the scheme policy.
A non-empty list is an exact or wildcard host allowlist. Explicitly
allowlisted hosts may be private (trusted internal services).

### Example: `RestrictedNetworkPolicy`

```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	gowkhtmltopdf "gowkhtmltopdf"
)

func main() {
	global := gowkhtmltopdf.NewGlobalSettings()
	if err := global.SetNetworkPolicy(gowkhtmltopdf.RestrictedNetworkPolicy()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var out bytes.Buffer
	err := gowkhtmltopdf.RunPDF(context.Background(), &gowkhtmltopdf.PDFRequest{
		Global: global,
		Objects: []*gowkhtmltopdf.ObjectSettings{
			gowkhtmltopdf.NewObjectSettings().SetPage("https://example.com/report.html"),
		},
		Output: &out,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = os.WriteFile("report.pdf", out.Bytes(), 0o644)
}
```

Use `CompatibleNetworkPolicy()` only when you intentionally want the
historical any-host HTTP behavior. For untrusted HTML in a service, prefer
restricted policy plus host allowlists. Details:
[integration-security.md](integration-security.md) and
[THREAT-MODEL.md](THREAT-MODEL.md).

## Headers and footers

Text HF keys live on **global** settings and can be overridden per object
(`header.left`, `footer.center`, …). Object `Set("header.left", …)` marks
that object as having a header override.

Placeholders in text HF strings:

`[page]`, `[topage]`, `[frompage]`, `[date]`, `[time]`, `[title]`,
`[doctitle]`, `[webpage]`, `[section]`, `[subsection]`.

`[date]` / `[time]` follow `PDFRequest.Now` when set; `Converter` uses the
wall clock. PDF `/Title` is the global `title` key, not HTML `<title>`.

HTML HF: `header.htmlurl` / `footer.htmlurl` (URL or path values; raw markup
is rejected). The engine runs a nested child layout (body CSS subset +
local `@font-face` under the same ACL), clipped to the reserved margin
band. HTML HF `#id` fragment GoTo resolves to **body** destinations only.
See [compatibility-matrix.md](compatibility-matrix.md) §7.7.

There is no public dotted `Set` for `--replace` maps. Custom substitutions
are a CLI flag (`--replace name value`).

Cover objects (`NewCoverObject`) start with empty header/footer and are
excluded from the outline.

## TOC and cover

`NewTOCObject()` and `NewCoverObject()` **are public**. They set the
registered keys `istableofcontent=true` and `iscover=true`.

- TOC objects are metadata. They do not count as a renderable body.
- A TOC-only request is rejected: `ErrNoRenderablePDFObjects` (PDF) or
  `ErrNoInputPageAdded` (image).
- Cover objects need `SetPage` or `SetBody` like any other body page.

`ErrNoRenderablePDFObjects` is an alias of `ErrNoPageObjectsAdded`.
`errors.Is` matches either name.

Global `toc.*` keys configure the generated TOC. Outline bookmarks are
separate: `outline` / `outlinedepth` (on by default).

### Example: `NewTOCObject` + body

```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	gowkhtmltopdf "gowkhtmltopdf"
)

func main() {
	global := gowkhtmltopdf.NewGlobalSettings()
	_ = global.Set("outline", "true")
	_ = global.Set("toc.captiontext", "Contents")

	body := gowkhtmltopdf.NewObjectSettings().SetBody(
		[]byte(`<html><body>
			<h1>Chapter one</h1><p>…</p>
			<h1>Chapter two</h1><p>…</p>
		</body></html>`),
		"",
	)

	var out bytes.Buffer
	err := gowkhtmltopdf.RunPDF(context.Background(), &gowkhtmltopdf.PDFRequest{
		Global: global,
		Objects: []*gowkhtmltopdf.ObjectSettings{
			gowkhtmltopdf.NewTOCObject(),
			body,
		},
		Output: &out,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = os.WriteFile("book.pdf", out.Bytes(), 0o644)
}
```

The same object list works with `Converter.AddObject` in order. A cover
page is `NewCoverObject().SetBody(...)` or `SetPage(...)` inserted before
the TOC or body.

## Output handling

| Entry | Where bytes go | `Output()` |
|-------|----------------|------------|
| `RunPDF` / `RunImage` | `req.Output` | n/a (no converter) |
| `Converter.Convert` / `ImageConverter.Convert` | internal buffer | copy of that buffer |
| `ConvertTo` (both drivers) | the `io.Writer` argument | **unchanged** (typically nil) |
| `ConvertHTML` | returned `[]byte` | n/a |
| `dumpoutline` | `PDFRequest.OutlineOutput` only | never mixed into the PDF |

`Output()` always returns a **copy**, valid across later conversions. A
fresh converter returns nil.

Prefer `ConvertTo` or `RunPDF` / `RunImage` when the caller already has a
writer (HTTP response, file, `bytes.Buffer`). `Convert` exists so
`Output()` stays compatible with the wkhtmltopdf-shaped lifecycle.

`ConvertTo` / `Run*` do not write a temp file. The document sink is the
writer you passed.

## Errors

Match with `errors.Is`. Wrapping preserves the sentinel.

### Sentinels

| Sentinel | When |
|----------|------|
| `ErrNoPageObjectsAdded` | PDF conversion with no page objects |
| `ErrNoRenderablePDFObjects` | alias of `ErrNoPageObjectsAdded`; also TOC-only / empty body |
| `ErrEmptyHTML` | `ConvertHTML` given an empty document |
| `ErrNoInputPageAdded` | image conversion with no renderable page |
| `ErrNilImageConverter` | method on a nil `*ImageConverter` |
| `ErrNilConverter` | method on a nil `*Converter` |
| `ErrNilGlobalSettings` | `Set` / `SetNetworkPolicy` on nil globals |
| `ErrInvalidPageSize` | named page size not in the table |
| `ErrNilObjectSettings` | `Set` on nil object settings |
| `ErrNilImageSettings` | `Set` on nil image settings |
| `ErrNilPDFRequest` | `RunPDF` / `ValidatePDF` with a nil request |
| `ErrNilImageRequest` | `RunImage` / `ValidateImage` with a nil request |
| `ErrMissingPDFOutput` | typed PDF (or `ConvertTo`) without a document sink |
| `ErrInvalidPDFCopies` | non-positive copy count |
| `ErrMissingPDFOutlineOutput` | `dumpoutline` without `OutlineOutput` |
| `ErrMissingImageOutput` | typed image (or image `ConvertTo`) without a sink |
| `ErrNilContext` | nil `context.Context` on a cancellation-aware entry |
| `ErrInvalidPDFVersion` | `pdfversion` / `WithPDFVersion` not `1.4` / `1.7` / `2.0` |
| `ErrInvalidPDFProfile` | `pdfprofile` / `WithPDFProfile` not a known alias or canonical name |
| `ErrProfilePDFA1Unsupported` | PDF/A-1 requested (unsupported) |
| `ErrConformanceRequiresPDF17` | A-3a / UA-1 / `a3a-ua1` with an explicit non-1.7 version (`ErrProfileRequiresPDF17` is an alias) |
| `ErrConformanceRequiresPDF20` | A-4 / UA-2 / `a4-ua2` with an explicit non-2.0 version (`ErrProfileRequiresPDF20` is an alias) |
| `ErrTitleRequired` | PDF/UA profile with an empty document title |
| `ErrPDFUAMissingAlt` | PDF/UA profile with a figure/image that has no alt text |

`ErrProfilePDF20Unsupported` is still exported for source compatibility. It is
**never returned**; PDF 2.0 profiles are supported.

`ErrNilPDFRequest` does **not** alias `ErrNilConverter`.
`ErrNilImageRequest` does **not** alias `ErrNilImageConverter`.

There is **no** public `HttpErrorCode()`.

### Wrap prefixes

| Path | Prefix |
|------|--------|
| `Converter.Convert` / `ConvertTo` / `ConvertHTML` (engine) | `convert: %w` |
| `RunPDF` (engine, after `ValidatePDF`) | `pdf: %w` |
| `ImageConverter.Convert` / `ConvertTo` (engine) | `image convert: %w` |
| `RunImage` (engine, after `ValidateImage`) | `image: %w` |
| `GlobalSettings.Set` | `global set %q: %w` |
| `ObjectSettings.Set` | `object set %q: %w` |
| `ImageConverter.Set` / `ImageSettings.Set` | `image set %q: %w` |
| `PdfGlobalOptions.WithSetting` | `settings builder: %w` |

Preflight sentinels (`ValidatePDF`, `ValidateImage`, empty-object checks
before the engine starts) are returned without those prefixes.
`ConvertHTML` / `Convert` still wrap **engine** failures as `convert: %w`.

Load failures (network, ACL, HTTP status) wrap the underlying error.
Context cancellation is honored during load and convert.

## Thread safety

One converter per job. The same rule applies to `ImageConverter` and to
`RunPDF` / `RunImage`: do not share a `*Converter`, `*ImageConverter`, or
in-flight request across concurrent converts.

`Converter` / `ImageConverter` are **not** safe for concurrent `Convert` /
`ConvertTo`. Create one per conversion or serialize access. Settings may be
read while idle.

Ownership copies:

| Boundary | What is cloned |
|----------|----------------|
| `AddObject` | deep copy of `*ObjectSettings` |
| `WithGlobal` | clone of `*GlobalSettings` |
| `ConvertTo` (PDF and image) | snapshot of global, objects, and image settings |
| `RunPDF` / `RunImage` | clone of global, objects, and image settings |
| `SetBody` / `SetNetworkPolicy` | copy of bytes / host-scheme slices |
| `Output()` | copy of the stored buffer |
| `PdfGlobalOptions.Build` | independent snapshot |

Later mutation of the caller's settings object does not change an in-flight
request. The next `Convert` / `Run*` sees the mutated live settings.

## Security

The converter fetches the page and its subresources **from your process**.
That is the intended path for HTML **you** generate. It is dangerous if you
pass **user-controlled URLs** (SSRF) or enable **local file access** for
untrusted input.

**Preferred:** render your template → convert that file or those bytes →
return the PDF. **Avoid:** `convert(c.Query("url"))` for arbitrary users
without host allowlists and network isolation.

Defaults that help: local files denied; `RestrictedNetworkPolicy` for
untrusted HTML; context cancellation; loader timeouts and body caps.
`SetBody` / `ConvertHTML` are an explicit document source (no URL guessing),
but they do not disable subresource fetches.

This is the same design risk class as embedding upstream wkhtmltopdf if you
pipe user URLs into either tool.

Read:

- [integration-security.md](integration-security.md) — Gin / HTTP patterns
- [THREAT-MODEL.md](THREAT-MODEL.md) — ACL, network policy, limits

## Related docs

- [getting-started.md](getting-started.md) — install, first PDF/PNG, `replace`
- [cli.md](cli.md) — `gowkhtmltopdf` / `gowkhtmltoimage` grammar and flags
- [fonts.md](fonts.md) — bundled faces, `fontpath`, `@font-face`, shaping
- [compatibility-matrix.md](compatibility-matrix.md) — HTML/CSS/flag contract
- [integration-security.md](integration-security.md) — embedding without SSRF
- [THREAT-MODEL.md](THREAT-MODEL.md) — trust boundary and loader limits
- [`examples/pdf`](../examples/pdf) — Converter + local-file ACL
- [`examples/image`](../examples/image) — ImageConverter + `AddObject` path
