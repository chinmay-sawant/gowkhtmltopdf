# Getting started

This page takes you from a checkout to a PDF, a PNG, and a Go library call.

Related: [CLI reference](cli.md) · [Library API](library-api.md) ·
[Samples](samples.md) · [Security](THREAT-MODEL.md)

## Requirements

- Go **1.26+** (the module pins `toolchain go1.26.4`)
- The first build may download the two allowlisted direct modules in
  [`go.mod`](../go.mod) (`go-text/typesetting`, `tdewolff/canvas`) and their
  transitive graph. After that, builds can run offline from the module cache.
- Native CLI builds require no browser process, native converter, Qt, or cgo
  toolchain. The optional browser preview uses the WASM build in
  [wasm.md](wasm.md).

## Install with Go

Pin the tagged release (Go 1.26+). Binaries land on `GOBIN` or
`$(go env GOPATH)/bin`:

```sh
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.6
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltoimage@v0.2.6
gowkhtmltopdf --version
# Name: gowkhtmltopdf
# Version: 0.2.6
```

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
# stamps VERSION into bin/gowkhtmltopdf and bin/gowkhtmltoimage
# export CGO_ENABLED=0 first for the static release link; make build
# does not set that variable itself
make build

# equivalent, explicit:
CGO_ENABLED=0 go build -ldflags "-X github.com/chinmay-sawant/gowkhtmltopdf/internal/cli.Version=$(cat VERSION)" \
  -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
CGO_ENABLED=0 go build -ldflags "-X github.com/chinmay-sawant/gowkhtmltopdf/internal/cli.Version=$(cat VERSION)" \
  -o bin/gowkhtmltoimage ./cmd/gowkhtmltoimage
```

Release archives are built with `CGO_ENABLED=0`, which produces a static
binary with no native runtime library. `make build` stamps `VERSION` and
leaves `CGO_ENABLED` alone, so export it before `make build` when you want
that same link.

Check the stamp:

```sh
./bin/gowkhtmltopdf --version
# Name: gowkhtmltopdf
# Version: 0.2.6
```

`VERSION` (currently `0.2.6`) is the project release. The library constant
`LibraryVersion` (`0.12.7-dev`) is a **wkhtmltopdf settings-surface**
compatibility id, not the release number. See [library-api.md](library-api.md#versioning).

## Browser preview

Run `make wasm` and then build the frontend to create the inline HTML browser
converter. The `/live-demo` route supports PDF, PNG, and JPEG output in a Web
Worker. Read [Browser WASM conversion](wasm.md) for the request contract,
limits, and resource boundary.

## First local PDF

Local files are **blocked by default**. Pass `--allow-local-files` and name
the output with `-o`. There is no `page` keyword:

```sh
./bin/gowkhtmltopdf --allow-local-files -o /tmp/invoice.pdf \
  testdata/golden/fixture-01-simple-invoice.html
```

Flag names are in [cli.md](cli.md). The break from the 0.2.3 grammar is in
[MIGRATION-0.2.4.md](MIGRATION-0.2.4.md).

## PDF version and profile

These flags shipped in **0.2.2**.
Default output is **unclaimed PDF 1.4**. `--pdf-version` changes the file
version only; it is **not** a PDF/A or PDF/UA claim.

```sh
# still unclaimed PDF 1.4
./bin/gowkhtmltopdf --allow-local-files -o /tmp/invoice.pdf \
  testdata/golden/fixture-01-simple-invoice.html

# version only (header / strings / non-claiming XMP — not a profile)
./bin/gowkhtmltopdf --pdf-version 1.7 --allow-local-files \
  -o /tmp/invoice-17.pdf testdata/golden/fixture-01-simple-invoice.html
./bin/gowkhtmltopdf --pdf-version 2.0 --allow-local-files \
  -o /tmp/invoice-20.pdf testdata/golden/fixture-01-simple-invoice.html

# opt-in profiles (imply PDF 1.7 and 2.0 respectively)
./bin/gowkhtmltopdf --pdf-profile a3a-ua1 --allow-local-files \
  -o /tmp/invoice-a3a-ua1.pdf testdata/golden/fixture-01-simple-invoice.html
./bin/gowkhtmltopdf --pdf-profile a4-ua2 --allow-local-files \
  -o /tmp/invoice-a4-ua2.pdf testdata/golden/fixture-01-simple-invoice.html
```

Target library fields are `PDFVersion: "1.7"` / `"2.0"` and
`PDFProfile: "a3a-ua1"` / `"a4-ua2"`. Full target flag and API tables:
[cli.md](cli.md), [library-api.md](library-api.md).

## First image

```sh
./bin/gowkhtmltoimage --allow-local-files --width 1024 \
  -o /tmp/invoice.png \
  testdata/golden/fixture-01-simple-invoice.html
```

Image mode renders one canvas (default viewport 1024 CSS px). Text uses the
same TTF outline raster path as PDF, with coverage anti-aliasing.

## Remote URL

```sh
./bin/gowkhtmltopdf -o /tmp/remote.pdf \
  --url "https://example.com/report.html"
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
[cli.md, URL mode](cli.md#url-mode-chrome-strip-simplify-dom),
[integration-security.md](integration-security.md), and
[THREAT-MODEL.md](THREAT-MODEL.md).

For untrusted HTML, prefer `--restrict-network` (blocks private/loopback
destinations and cross-host redirects):

```sh
./bin/gowkhtmltopdf --restrict-network -o out.pdf \
  --url 'https://example.com/report.html'
```

## A slightly richer document

Headers, page numbers, outline bookmarks, and a generated TOC:

```sh
./bin/gowkhtmltopdf \
  --header-left "gowkhtmltopdf" --header-right "[title]" --header-line \
  --footer-center "[page] / [topage]" --footer-line \
  --toc-header-text "Report contents" --disable-dotted-lines \
  --outline --outline-depth 4 --title "Invoice Report" \
  --allow-local-files --toc \
  -o /tmp/book.pdf testdata/golden/fixture-16-invoice-with-css.html
```

Placeholders: `[page]`, `[topage]`, `[frompage]`, `[date]`, `[time]`,
`[title]`, `[doctitle]`, `[webpage]`, `[section]`, `[subsection]`.
`[subject]` expands empty. Custom substitutions: `--replace key value`.

Outlines are **on by default** (depth 4). PDF `/Title` comes from `--title`,
not from the HTML `<title>` (`<title>` feeds `[doctitle]` only).

## Library

Module path is `github.com/chinmay-sawant/gowkhtmltopdf`. The public types
are `Document` and `ImageDocument` (shipped in 0.2.4, current at `VERSION`
0.2.6).

> **Python user?** The same engine runs in-process from Python through an
> opt-in shared library (`pip install gowkhtmltopdf`). Install steps,
> quickstart snippets, the security rules, and the ABI promise are in
> [python.md](python.md).

```go
doc := gowkhtmltopdf.Document{
	Pages: []gowkhtmltopdf.Page{{Source: gowkhtmltopdf.Content{
		HTML: html,
	}}},
	PageSize: "A4",
}
pdfBytes, err := doc.PDF(ctx)
```

Worked programs: [`examples/pdf`](../examples/pdf/) and
[`examples/image`](../examples/image/). Full surface:
[library-api.md](library-api.md). The 0.2.3 symbol list is in
[MIGRATION-0.2.4.md](MIGRATION-0.2.4.md).

## Run tests

```sh
make test      # go test ./... with -p 2 -parallel 2 (safe on ~8 GiB machines)
make test-unit # skip internal/convert; use with make golden for layout work
make lint      # golangci-lint, then frontend npm run lint (installs a pinned Go linter if missing)
make golden    # fixture structure + feature checks
make samples   # regenerate output/*.pdf (network only for the wiki smoke)
```

Layout / print changes should also open a regenerated PDF in a real viewer.
See [samples.md](samples.md) and [CONTRIBUTING.md](../CONTRIBUTING.md).

Bare `go test ./...` on a many-core host defaults to one package (and many
parallel subtests) per CPU. Convert/layout/pdf fixtures then compete for RAM
and can freeze the desktop. Prefer the Make targets above, or raise the caps
explicitly: `make test TEST_P=4 TEST_PARALLEL=4`.

## Makefile targets

| Target | Action |
|--------|--------|
| `make test` | `go test ./...` with `-p 2 -parallel 2` (override via `TEST_P` / `TEST_PARALLEL`) |
| `make test-unit` | Same caps; all packages except `internal/convert` |
| `make test-quick` | `make test` plus `-short` |
| `make test-serial` | `-p 1 -parallel 1` for very low RAM |
| `make test-race` | `-race` on convert/layout/pdf/imageout/load, same caps |
| `make lint` | `golangci-lint` (pinned v1.64.8) via [`.golangci.yml`](../.golangci.yml), then `size-check`, then `npm run lint` in `frontend/` |
| `make build` | `bin/gowkhtmltopdf`, `bin/gowkhtmltoimage` (stamps `VERSION`) |
| `make golden` | Golden corpus tests (capped parallelism) |
| `make golden-update GOLDEN_FIXTURE=fixture-NN-name.html GOLDEN_APPROVE=1` | One reviewed PDF under ignored `testdata/golden/out/` |
| `make samples` | Refresh [`output/`](../output/) |
| `make fmt` | `gofmt -w .` |
| `make bench` | External CLI comparisons via `scripts/bench-external.sh` (wkhtmltopdf, then WeasyPrint and Puppeteer on that gowk baseline) |
| `make bench-engine` | Internal engine matrix (`bench-inprocess` is the old name) |
| `make bench-lib` | Public `Document` / `ImageDocument` timings |
| `make bench-cli-compare` | Process-level comparison against installed wkhtmltopdf (`make build` first) |
| `make run` | Build the PDF CLI and convert `RUN_HTML` (default fixture-01) under `RUN_MAX_MS` (default 400) |
| `make chrome-cases-pdf` | One PDF per `test/chrome/cases/case-*.html` under `test/chrome/cases/pdf/` |
| `make wasm` / `make wasm-test` | Browser artifact in `frontend/public/wasm/`; `wasm-test` runs the adapter contract and the live-demo harness |
| `make claim-scan` | Fail on forbidden over-claim phrases in user-facing docs |
| `make clean` | Remove `testdata/golden/out` and `dist` |
