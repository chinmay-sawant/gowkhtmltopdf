# Python bindings

## What this is

`gowkhtmltopdf` on PyPI is an in-process Python binding for this repo's
HTML-to-PDF and HTML-to-image engine. Your Python process loads one shared
library (`libgowkhtmltopdf.so`, `libgowkhtmltopdf.dylib`, or
`libgowkhtmltopdf.dll`) through `ctypes` and calls it directly. No subprocess,
no browser, no JavaScript execution.

Two facts frame everything else on this page:

- The binding is an opt-in cgo build. The shared library is produced by
  `go build -buildmode=c-shared` with `CGO_ENABLED=1`, from the exports under
  `bindings/c`. Default Go builds of this repo stay `CGO_ENABLED=0`; nothing
  about the Go library or CLI changes when you install the Python package.
- The binding calls the same engine the CLI calls. HTML goes through the Go
  pipeline: load, parse, CSS cascade, layout, paginate, paint, PDF write.
  PDF structure claims stay modest: this is a print-oriented document
  engine, not Chrome visual parity.

## Install

```sh
pip install gowkhtmltopdf
```

Linux x86_64 wheels (`manylinux_2_28`) ship first. macOS and Windows wheels
follow on later releases (built by the `publish-pypi.yml` workflow). If no
wheel matches your platform yet, install the sdist after building the shared
library yourself: see [Build the shared library yourself](#build-the-shared-library-yourself).

From a repo checkout:

```sh
CGO_ENABLED=1 make c-shared
pip install ./bindings/python
```

## Quickstart

The high-level helper, one call, bytes out:

```python
from gowkhtmltopdf import convert_html_to_pdf, PDFOptions

pdf_bytes = convert_html_to_pdf(
    html=b"<html><body><h1>Invoice #42</h1></body></html>",
    options=PDFOptions(
        page_size="A4",
        orientation="portrait",
        allow_local_files=False,
    ),
)

with open("invoice.pdf", "wb") as f:
    f.write(pdf_bytes)
```

Note the flag name: it is `allow_local_files`, mirroring the Go field
`Document.AllowLocalFiles` (`document.go:130`). Local file reads are denied
by default either way; passing `False` keeps that promise visible in your
code.

The same work in document form, field-for-field next to the Go API:

```python
from gowkhtmltopdf import Document, Page, Content

doc = Document(
    pages=[Page(source=Content(html=b"<html><body><h1>Invoice</h1></body></html>"))],
    page_size="A4",
)
pdf_bytes = doc.pdf()
```

Both forms return `bytes` starting with `%PDF-`.

Images:

```python
from gowkhtmltopdf import ImageDocument, Content

png_bytes = ImageDocument(
    source=Content(html=b"<h1>Badge</h1>"),
    width=1024,
    format="png",
).image()
```

And the image sugar:

```python
from gowkhtmltopdf import convert_html_to_image, ImageOptions

png_bytes = convert_html_to_image(b"<h1>Badge</h1>", options=ImageOptions(width=1024))
```

## File and URL sources

`Content` takes exactly one source kind. Zero or two kinds raise an error;
`base` is only valid together with `html`.

| Kind | Call | Notes |
|------|------|-------|
| Inline HTML | `Content(html=b"<h1>hi</h1>")` | pass `base=` for relative links |
| Local file | `Content(file="report.html")` | subject to the local-file rules below |
| Remote URL | `Content(url="https://example.com/report.html")` | subject to the network policy below |

File helper:

```python
from gowkhtmltopdf import convert_file_to_pdf

convert_file_to_pdf("report.html", "report.pdf", page_size="A4")
```

Reading any path from disk goes through the same access controller described
under [Security](#security). If your HTML also references images or
stylesheets on disk, scope an `allow` list instead of opening the wide
switch.

## Document reference

Python mirrors the Go `Document` API (`document.go`) field-for-field in
snake_case:

| Go field | Python argument |
|----------|-----------------|
| `Pages` | `pages` |
| `Cover` | `cover` |
| `TOC` | `toc` |
| `PageSize` | `page_size` |
| `WidthMM` / `HeightMM` | `width_mm` / `height_mm` |
| `Orientation` | `orientation` |
| `Margin{Top,Right,Bottom,Left}` | `margin=Margin(top, right, bottom, left)` |
| `Title` | `title` |
| `PDFVersion` | `pdf_version` |
| `PDFProfile` | `pdf_profile` |
| `Copies` | `copies` |
| `AllowLocalFiles` | `allow_local_files` |
| `Allow` | `allow` |
| `Network` | `network` |
| `FontPaths` | `font_paths` |
| `UseSystemFonts` | `use_system_fonts` |
| `Outline` | `outline` |
| `OutlineDepth` | `outline_depth` |
| `Background` | `background` |
| `SmartShrinking` | `smart_shrinking` |
| `Compression` | `compression` |
| `ResolveRelLinks` | `resolve_relative_links` |
| `Grayscale` | `grayscale` |
| `PageOffset` | `page_offset` |
| `ExcludeFromOutline` | `exclude_from_outline` |

Methods:

| Method | Behavior |
|--------|----------|
| `Document.validate()` | Checks pages exist, sizes parse, sources are well formed. Raises before conversion starts. |
| `Document.pdf(timeout=None)` | Renders and returns PDF bytes. Each call returns fresh bytes. |
| `Document.write_pdf(fileobj, timeout=None)` | Renders into an open binary file object. |
| `ImageDocument.image()` | Renders PNG/JPEG bytes. |
| `ImageDocument.write_image(fileobj)` | Writes image bytes into an open binary file object. |

There is no Go `context` parameter here. Pass `timeout=` instead; see
[Timeouts](#timeouts).

## Options reference

`PDFOptions` drives the one-call helpers. It builds
`Document(pages=[Page(source=Content(html=...))], ...)` internally.

| Field | Type | Default | Maps to |
|-------|------|---------|---------|
| `page_size` | `str` | `"A4"` | `PageSize` |
| `orientation` | `str` | `"portrait"` | `Orientation` |
| `margin` | `Margin \| None` | `None` | `Margin` |
| `title` | `str \| None` | `None` | `Title` |
| `pdf_version` | `str \| None` | `None` | `PDFVersion` |
| `pdf_profile` | `str \| None` | `None` | `PDFProfile` |
| `copies` | `int` | `1` | `Copies` |
| `allow` | `list[str]` | `[]` | `Allow` |
| `allow_local_files` | `bool` | `False` | `AllowLocalFiles` |
| `base_url` | `str \| None` | `None` | `Content.base` |
| `timeout_ms` | `int` | `0` | conversion deadline |
| `network` | `NetworkPolicy \| None` | `None` | `Network` |

`ImageOptions` mirrors `ImageDocument` fields:

| Field | Type | Default | Maps to |
|-------|------|---------|---------|
| `width` | `int \| None` | `None` | `width` |
| `height` | `int \| None` | `None` | `height` |
| `format` | `str` | `"png"` | `format` (`png` or `jpeg`) |
| `quality` | `int` | `94` | JPEG quality |
| `smart_width` | `bool` | `True` | `smart_width` |
| `transparent` | `bool` | `False` | `transparent` |
| `crop` | `Crop \| None` | `None` | `crop` |
| `zoom` | `float` | `0` | `zoom` |

## Errors

`GowkhtmltopdfError` is the base class. Subclasses map one-to-one onto the C
ABI status codes, and every exception carries `.code`, the integer below:

| Code | Status | Exception class |
|-----:|--------|-----------------|
| 0 | OK | (no exception) |
| 1 | INVALID_ARG | `InvalidArgumentError` |
| 2 | LOAD_DENIED | `LoadDeniedError` |
| 3 | RENDER_ERROR | `RenderError` |
| 4 | TIMEOUT | `ConversionTimeoutError` |
| 5 | RESOURCE_LIMIT | `ResourceLimitError` |
| 6 | INTERNAL | `InternalEngineError` |

Go sentinel errors surface as sentinel classes you can catch with
`isinstance`, for example `ErrNoPageObjects`, `ErrInvalidPageSize`,
`ErrInvalidContent`, `ErrEmptyHTML`:

```python
import gowkhtmltopdf

try:
    gowkhtmltopdf.Document(pages=[]).validate()
except gowkhtmltopdf.ErrNoPageObjects:
    print("add at least one Page")
```

`LoadDeniedError` covers both denied local reads and denied network requests.

## Timeouts

Two layers:

- Conversion deadline: pass `timeout_ms` in `PDFOptions` / `ImageOptions`, or
  `timeout=` seconds to `pdf()` / `image()`. The value crosses the ABI as a
  millisecond budget and lands in a Go `context.WithTimeout`. When it runs
  out you get code 4, `ConversionTimeoutError`.
- Network waits have fixed caps regardless of your deadline: 30 s connect,
  60 s response, 100 MiB body, 10 redirects maximum
  (`internal/load/load.go:38-43`).

## Security

The binding calls the same loader the CLI and Go library use
(`internal/load`). The rules are identical because they are the same code,
not a re-implementation.

Local files:

- Reads are denied unless you open them.
- `allow_local_files=True` is one wide switch: any local path the HTML
  references may then be read (`document.go:130`). Avoid it for untrusted
  input.
- Prefer a scoped list: `allow=["/var/app/templates"]`. A path is readable
  when it resolves inside one of these directories.
- Prefix checks resolve symlinks on both the candidate path and the allow
  prefix before comparing, so a symlink planted inside an allowed directory
  cannot point outside it (`internal/load/load.go:248`, symlink resolution
  at `internal/load/load.go:280`).
- `file://` URLs must carry an empty host or `localhost`; any other host is
  refused (`internal/load/load.go:1195`).

Remote fetches (`api.go:30-42`, `internal/load/load.go:108-138`):

- Compatible policy (default): HTTP(S) allowed anywhere, private addresses
  included, redirects may cross hosts (`CompatibleNetworkPolicy`,
  `api.go:34`).
- Restricted policy: blocks private and link-local destinations unless the
  exact host is allowlisted, keeps redirects on the original host, and pins
  each dial to the address the original host resolved to
  (`RestrictedNetworkPolicy`, `internal/load/load.go:132`, dial enforcement
  at `internal/load/load.go:522`). Use it for untrusted HTML.
- Fixed limits apply either way: 30 s connect, 60 s response, 100 MiB body,
  10 redirects (`internal/load/load.go:38-43`).

In-process means in-process: `ctypes` maps the shared library into your
Python process and calls it. The CLI grammar in [cli.md](cli.md) is a
different interface; its flags do not exist on this surface. Wrapping the
CLI binary with `subprocess` remains possible for other projects but is out
of scope for this package.

Threading: treat each `Document` instance as owned by one thread at a time.
Distinct conversions may run concurrently; each executes inside the Go
engine independently. `ctypes` releases the GIL while a C call runs, so a
long render does not block other Python threads.

## ABI stability and versions

Three numbers identify what you are running:

| Value | Meaning | Today |
|-------|---------|-------|
| `gowkhtmltopdf.__version__` | Project release, tracks the repo `VERSION` file | `0.2.4` |
| `gowkhtmltopdf.library_version` | wkhtmltopdf settings-surface identifier the engine mirrors (`api.go:23`) | `0.12.7-dev` |
| `GOWKHTMLTOPDF_ABI_VERSION` | Raw ABI level in `bindings/c/include/gowkhtmltopdf.h` | `1` |

The loader checks the ABI version at import time and refuses a mismatched
library instead of passing structs across a boundary that changed shape.

Stability policy, semver style:

- MAJOR bump: breaking change to exported functions or struct layouts.
- MINOR bump: additive only, new exports or appended struct fields.
- PATCH bump: no ABI change.

Every option struct carries leading `abi_version` and `struct_size` fields.
The library rejects a struct whose size it does not recognize, which turns a
mixed-version mistake into a clean INVALID_ARG error rather than undefined
behavior.

## Build the shared library yourself

Needed when no wheel matches your platform, or while developing against a
checkout:

```sh
CGO_ENABLED=1 make c-shared
```

Or directly:

```sh
CGO_ENABLED=1 go build -buildmode=c-shared \
  -ldflags "-X github.com/chinmay-sawant/gowkhtmltopdf/bindings/c.libVersion=$(cat VERSION)" \
  -o dist/libgowkhtmltopdf.so ./bindings/c
```

Outputs land in `dist/libgowkhtmltopdf.so` plus a generated header beside it.
The generated header is a build artifact and stays out of git; the curated
contract header `bindings/c/include/gowkhtmltopdf.h` is the authoritative
ABI description.

Platform file names: Linux uses `.so`, macOS uses `.dylib`, Windows uses
`.dll`. To point the loader at a non-default location, set
`GOWKHTMLTOPDF_LIBRARY_PATH`.

Then install the package against your build:

```sh
pip install ./bindings/python
```

Building the sdist requires setuptools >= 61.

## Platforms

Wheels come from the repo's `publish-pypi.yml` workflow:

| Platform | Availability |
|----------|--------------|
| Linux x86_64, `manylinux_2_28` | Day 1 |
| Linux aarch64 | Follows |
| macOS 13+ / 14+, x86_64 / arm64 | Follows |
| Windows x86_64 | Follows |

The sdist carries the Go sources and the contract header, so any platform
with Go and a C toolchain can rebuild the library locally using the commands
above.

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `ImportError` mentioning an ABI mismatch | The installed wheel and the shared library disagree on the ABI level. Reinstall the matching wheel: `pip install --force-reinstall gowkhtmltopdf`. |
| Library-not-found error at import | Set `GOWKHTMLTOPDF_LIBRARY_PATH=/path/to/libgowkhtmltopdf.so` to your build output. |
| Source build fails with old packaging tools | Building from sdist needs setuptools >= 61: `pip install -U setuptools`. |

## Where the engine lives

The Python package is a thin caller. The work happens in the Go pipeline:
load, HTML parse, CSS cascade, layout, paginate, paint, PDF write, owned by
packages under `internal/`. Package map: [architecture.md](architecture.md).
Stage-by-stage notes with source citations:
[architecture/README.md](architecture/README.md).
