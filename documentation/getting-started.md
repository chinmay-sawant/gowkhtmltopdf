# Getting started

This page takes you from a checkout to a PDF, a PNG, and a Go library call.

Related: [CLI reference](cli.md) · [Library API](library-api.md) ·
[Samples](samples.md) · [Security](THREAT-MODEL.md)

## Requirements

- Go **1.26+** (the module pins `toolchain go1.26.4`)
- The first build may download the two allowlisted direct modules in
  [`go.mod`](../go.mod) (`go-text/typesetting`, `tdewolff/canvas`) and their
  transitive graph. After that, builds can run offline from the module cache.
- No browser, native converter, Qt, or cgo toolchain is required.

## Install prebuilt binaries

Tagged releases publish static `gowkhtmltopdf` and `gowkhtmltoimage` builds for
Linux, Windows, and macOS (amd64 and arm64) as GitHub Release assets, with a
`SHA256SUMS` file. Download from the project's
[Releases](https://github.com/chinmay-sawant/gowkhtmltopdf/releases) page.

Assets are produced only when a `v*` tag is pushed (see
[CONTRIBUTING.md](../CONTRIBUTING.md#cutting-a-release-multi-platform-binaries));
ordinary CI on pull requests does not upload platform archives.

## Build

```sh
# recommended: static binaries, version stamped from VERSION
make build
# writes bin/gowkhtmltopdf and bin/gowkhtmltoimage

# equivalent, explicit:
CGO_ENABLED=0 go build -ldflags "-X gowkhtmltopdf/internal/cli.Version=$(cat VERSION)" \
  -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
CGO_ENABLED=0 go build -ldflags "-X gowkhtmltopdf/internal/cli.Version=$(cat VERSION)" \
  -o bin/gowkhtmltoimage ./cmd/gowkhtmltoimage
```

Always build with `CGO_ENABLED=0`. The resulting binaries are statically
linked and have no native runtime library requirements.

Check the stamp:

```sh
./bin/gowkhtmltopdf --version
# Name: gowkhtmltopdf
# Version: 0.2.0
```

`VERSION` (currently `0.2.0`) is the project release. The library constant
`LibraryVersion` (`0.12.7-dev`) is a **wkhtmltopdf settings-surface**
compatibility id, not the release number. See [library-api.md](library-api.md#versioning).

## First local PDF

Local files are **blocked by default**. Opt in with
`--enable-local-file-access`:

```sh
./bin/gowkhtmltopdf --enable-local-file-access \
  testdata/golden/fixture-01-simple-invoice.html \
  /tmp/invoice.pdf
```

Open `/tmp/invoice.pdf` in any PDF viewer. Committed samples live in
[`output/`](../output/) — regenerate with `make samples`.

The same conversion with an explicit `page` keyword (useful once you add
page-scoped flags):

```sh
./bin/gowkhtmltopdf page --enable-local-file-access \
  testdata/golden/fixture-01-simple-invoice.html \
  /tmp/invoice.pdf
```

`--enable-local-file-access` before any `page`/`cover` attaches to the
**first real page**, not to a TOC object. Later pages stay blocked unless
you repeat the flag or use `--allow`. Details: [cli.md](cli.md#local-files).

## First image

```sh
./bin/gowkhtmltoimage --enable-local-file-access \
  --width 1024 \
  testdata/golden/fixture-01-simple-invoice.html \
  /tmp/invoice.png
```

Image mode renders one canvas (default viewport 1024 CSS px). Text uses the
same TTF outline raster path as PDF, with coverage anti-aliasing.

## Remote URL

```sh
./bin/gowkhtmltopdf \
  "https://example.com/report.html" \
  /tmp/remote.pdf
```

HTTP(S) fetch uses a 30 s connect timeout, 60 s response timeout (override
with `--timeout`), at most 10 redirects, and a 100 MiB body cap. TLS
verification is on. There is no `--insecure`.

**This is not “decent print.”** Complex public sites (heavy CSS/JS) will not
look like a browser. Phase 21 “decent print” is a progressive goal — see
[fidelity.md](fidelity.md#arbitrary-websites-phase-21).

**If you embed this in a web API:** do **not** pass arbitrary user `?url=`
values into the converter without host allowlists and network isolation.
That is classic **SSRF** (your server fetches internal hosts). The preferred
pattern is to generate HTML yourself, then convert that. The same class of
issue exists for upstream wkhtmltopdf. See
[cli.md — Remote URL security](cli.md#remote-url-security),
[integration-security.md](integration-security.md), and
[THREAT-MODEL.md](THREAT-MODEL.md).

For untrusted HTML, prefer `--restrict-network` (blocks private/loopback
destinations and cross-host redirects):

```sh
./bin/gowkhtmltopdf --restrict-network \
  'https://example.com/report.html' out.pdf
```

## A slightly richer document

Headers, page numbers, outline bookmarks, and a generated TOC:

```sh
./bin/gowkhtmltopdf \
  --header-left "gowkhtmltopdf" --header-right "[title]" --header-line \
  --footer-center "[page] / [topage]" --footer-line \
  --toc-header-text "Report contents" --disable-dotted-lines \
  --outline --outline-depth 4 --title "Invoice Report" \
  --enable-local-file-access \
  toc page testdata/golden/fixture-16-invoice-with-css.html \
  /tmp/book.pdf
```

Placeholders: `[page]`, `[topage]`, `[frompage]`, `[date]`, `[time]`,
`[title]`, `[doctitle]`, `[webpage]`, `[section]`, `[subsection]`.
`[subject]` expands empty. Custom substitutions: `--replace key value`.

Outlines are **on by default** (depth 4). PDF `/Title` comes from `--title`,
not from the HTML `<title>` (`<title>` feeds `[doctitle]` only).

## Library (minimal)

Module path is `gowkhtmltopdf` (same as the module name). Until the module
is published to a reachable path, use a `replace` against a checkout.

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

Both the global enable **and** the object-level unblock are required for a
local path. That matches the CLI pair.

Prefer the typed writer-first API when embedding:

```go
var out bytes.Buffer
err := gowkhtmltopdf.RunPDF(ctx, &gowkhtmltopdf.PDFRequest{
	Global: global,
	Objects: []*gowkhtmltopdf.ObjectSettings{
		gowkhtmltopdf.NewObjectSettings().SetBody(html, ""),
	},
	Now:    func() time.Time { return time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) },
	Output: &out,
})
```

`Now` pins PDF metadata and `[date]`/`[time]`. Without it the writer uses
the wall clock, so default CLI bytes are **not** hash-stable.

Worked programs: [`examples/pdf`](../examples/pdf/) and
[`examples/image`](../examples/image/). Full surface:
[library-api.md](library-api.md).

## Run tests

```sh
make test      # go test ./...
make lint      # golangci-lint (installs a pinned version if missing)
make golden    # fixture structure + feature checks
make samples   # regenerate output/*.pdf (network only for the wiki smoke)
```

Layout / print changes should also open a regenerated PDF in a real viewer.
See [samples.md](samples.md) and [CONTRIBUTING.md](../CONTRIBUTING.md).

## Makefile targets

| Target | Action |
|--------|--------|
| `make test` | `go test ./...` |
| `make lint` | `golangci-lint run` via [`.golangci.yml`](../.golangci.yml) |
| `make build` | `bin/gowkhtmltopdf`, `bin/gowkhtmltoimage` (stamps `VERSION`) |
| `make golden` | Golden corpus tests |
| `make golden-update GOLDEN_FIXTURE=fixture-NN-name.html GOLDEN_APPROVE=1` | One reviewed PDF under ignored `testdata/golden/out/` |
| `make samples` | Refresh [`output/`](../output/) |
| `make fmt` | `gofmt -w .` |
| `make bench` | In-process Go benchmark matrix |
| `make bench-cli-compare` | Process-level CLI comparison vs installed wkhtmltopdf |
| `make claim-scan` | Fail on forbidden over-claim phrases in user-facing docs |
| `make clean` | Remove `testdata/golden/out` |
