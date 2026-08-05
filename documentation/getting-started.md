# Getting started

## Requirements

- Go **1.26+**
- No network required to build (stdlib only)

## Build

```sh
# static binaries (recommended)
CGO_ENABLED=0 go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
CGO_ENABLED=0 go build -o bin/gowkhtmltoimage ./cmd/gowkhtmltoimage

# or Makefile
make build
```

Version stamp:

```sh
go build -ldflags "-X gowkhtmltopdf/internal/cli.Version=$(cat VERSION)" \
  -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
```

## First local PDF

Local files are **blocked by default** (security ACL). Opt in with
`--enable-local-file-access`:

```sh
./bin/gowkhtmltopdf --enable-local-file-access \
  testdata/golden/fixture-01-simple-invoice.html \
  /tmp/invoice.pdf
```

Open `/tmp/invoice.pdf` in any PDF viewer. Committed samples live in
[`output/`](../output/) - regenerate with `make samples`.

## Remote URL

```sh
./bin/gowkhtmltopdf \
  "https://example.com/report.html" \
  /tmp/remote.pdf
```

HTTP(S) fetch uses timeouts, redirect limits, and size caps. Complex public
sites (heavy CSS/JS) will not look like a browser; the product target is
server-generated HTML. Phase 21 “decent print” for arbitrary sites is a
**progressive goal**, not MVP acceptance — see
[fidelity.md](fidelity.md#arbitrary-websites-phase-21).

**If you embed this in Gin (or any API):** do **not** pass arbitrary user
`?url=` values into the converter without host allowlists and network
isolation - that is classic **SSRF** (your server fetches internal hosts).
The **preferred** pattern is to generate HTML yourself, then convert that.
The same class of issue exists for **upstream wkhtmltopdf**. See
[cli.md — Remote URL security](cli.md#remote-url-security),
[integration-security.md](integration-security.md), and
[THREAT-MODEL.md](THREAT-MODEL.md).

## Image output

```sh
./bin/gowkhtmltoimage --enable-local-file-access \
  --width 1024 \
  testdata/golden/fixture-01-simple-invoice.html \
  /tmp/invoice.png
```

## Library (minimal)

```go
package main

import (
	"context"
	"os"

	gowkhtmltopdf "gowkhtmltopdf"
)

func main() {
	c := gowkhtmltopdf.NewConverter()
	_ = c.Global().Set("size.pagesize", "A4")
	_ = c.Global().Set("enablelocalfileaccess", "true")

	obj := gowkhtmltopdf.NewObjectSettings().SetPage("invoice.html")
	_ = obj.Set("load.blocklocalfileaccess", "false")
	c.AddObject(obj)

	if err := c.Convert(context.Background()); err != nil {
		panic(err)
	}
	_ = os.WriteFile("out.pdf", c.Output(), 0o644)
}
```

More detail: [library-api.md](library-api.md), [examples/](../examples/).

## Run tests

```sh
make test
make lint
make golden   # fixture structure + feature checks
```
