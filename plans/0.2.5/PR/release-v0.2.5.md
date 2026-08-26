## v0.2.5

Sixth public release of **gowkhtmltopdf**: a **pure-Go**, **no-cgo**, **no Qt/WebKit**, **no browser** HTML template engine that turns structured HTML and templates into multi-page PDFs and images.

**v0.2.4** shipped the Go-native `Document` / `ImageDocument` API and CLI redesign. **v0.2.5** adds the missing in-process Python path: an opt-in cgo `c-shared` library with a frozen C ABI, plus a `pip install gowkhtmltopdf` package that loads it through `ctypes`. Default Go builds stay `CGO_ENABLED=0`. The pure-Go library and both CLIs do not change their public contract.

Default output is still **unclaimed PDF 1.4**. `--pdf-version` / `Document.PDFVersion` / Python `pdf_version` is a version header, **not** a conformance claim. The claim is `--pdf-profile` / `Document.PDFProfile` / Python `pdf_profile`.

- **License:** [MIT](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/LICENSE) - Copyright (c) 2026 Chinmay Sawant
- **Version source:** [`VERSION`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/VERSION) (`0.2.5`)
- **Site:** https://chinmay-sawant.github.io/gowkhtmltopdf/
- **Python guide:** [`documentation/python.md`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/documentation/python.md)
- **Compare:** https://github.com/chinmay-sawant/gowkhtmltopdf/compare/v0.2.4...v0.2.5
- **PR:** [#58](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/58) (Python cgo bindings and PyPI for [#35](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/35))

---

### Highlights

| Area | What you get in v0.2.5 |
|------|------------------------|
| **C ABI** | Frozen header `bindings/c/include/gowkhtmltopdf.h`: `GOWKHTMLTOPDF_ABI_VERSION 1`, size-gated option structs, status codes 0-6, `gowkhtmltopdf_html_to_pdf` / `html_to_image` + free helpers. |
| **c-shared build** | Opt-in only: `CGO_ENABLED=1 make c-shared` writes `dist/libgowkhtmltopdf.so` (or `.dylib` / `.dll`). Pure-Go `make build` / `make test` stay `CGO_ENABLED=0`. |
| **Python package** | `pip install gowkhtmltopdf`: `Document` / `ImageDocument` snake_case parity, `convert_html_to_pdf` / `convert_file_to_pdf` / `convert_html_to_image`, typed errors, zero runtime deps (`ctypes`). |
| **Wheels** | `manylinux_2_28` x86_64 + aarch64; macOS arm64 (`macos-latest`); Windows. Tag `v*` triggers Trusted Publishing. |
| **Samples** | `make samples-python` regenerates `output/python/`; `make python-api` for architecture / inline / compliance; `make python-benchmarks` on the same dirty report template as `make bench-lib`. |
| **Version stamp** | `VERSION` / `internal/cli.Version` / Python package / header macro **0.2.5**. Dated `CHANGELOG` **0.2.5 (2026-08-26)**. |

PDF 1.7 / 2.0 and PDF/A + PDF/UA profiles from earlier releases remain available on both the Go and Python surfaces.

**Highest compliance by version** (unchanged)

| Base | Archival | Accessibility | Opt-in |
|------|----------|---------------|--------|
| PDF 1.7 | PDF/A-3a | PDF/UA-1 | `--pdf-profile a3a-ua1` / `pdf_profile="a3a-ua1"` |
| PDF 2.0 | PDF/A-4 | PDF/UA-2 | `--pdf-profile a4-ua2` / `pdf_profile="a4-ua2"` |

---

### Showcase - Python path

Live gallery: https://chinmay-sawant.github.io/gowkhtmltopdf/#/showcase  
Python samples: [`output/python/`](https://github.com/chinmay-sawant/gowkhtmltopdf/tree/v0.2.5/output/python)  
Guide: [`documentation/python.md`](https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/documentation/python.md)

On the same dirty `report.html.tmpl` fixture (20 invoice rows per page), the in-process Python `Document.pdf()` path sits within a few percent of Go `Document.WritePDF` (about **3.7 ms** at 2 pages on the reference host). That is the same engine behind a thin `ctypes` hop, not a second renderer.

---

### Install / build

Cross-platform Go binaries are attached to this release (`gowkhtmltopdf` and `gowkhtmltoimage` for linux / windows / darwin x amd64 / arm64) plus `SHA256SUMS`.

Or install the CLIs with Go 1.26+:

```sh
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.5
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltoimage@v0.2.5
gowkhtmltopdf --version
```

Library pin:

```sh
go get github.com/chinmay-sawant/gowkhtmltopdf@v0.2.5
```

Python (in-process):

```sh
pip install gowkhtmltopdf
```

```python
from gowkhtmltopdf import Document, Page, Content, PDFOptions, convert_html_to_pdf

pdf_bytes = convert_html_to_pdf(
    b"<html><body><h1>Invoice #42</h1></body></html>",
    options=PDFOptions(page_size="A4"),
)

doc = Document(
    pages=[Page(source=Content(html=b"<html><body><h1>Invoice</h1></body></html>"))],
    page_size="A4",
)
pdf_bytes = doc.pdf()
```

From a source checkout (shared library for local Python):

```sh
git clone https://github.com/chinmay-sawant/gowkhtmltopdf.git
cd gowkhtmltopdf
git checkout v0.2.5
make build
CGO_ENABLED=1 make c-shared
pip install -e bindings/python
make samples-python
make python-benchmarks
```

---

### What landed in v0.2.5

#### 1. C ABI and c-shared build ([#58](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/58), [#35](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/35))

- Committed header `bindings/c/include/gowkhtmltopdf.h` as the ABI source of truth.
- `bindings/c` is the only package allowed to contain `import "C"`; stub builds keep `CGO_ENABLED=0` green.
- Exports wrap `Document.WritePDF` / `ImageDocument.WriteImage` with timeout and status mapping.
- `make c-shared` refuses to run unless `CGO_ENABLED=1`.

#### 2. Python package and PyPI ([#58](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/58))

- `bindings/python`: `Document` / `ImageDocument` parity, helpers, exceptions, `py.typed`.
- `convert_file_to_pdf` sets a `file://` base so linked CSS (fixture-56) resolves.
- CI: purity guard, `build-shared`, `python-binding` jobs; publish workflow for wheels on `v*` tags.
- `make check-versions` keeps `VERSION` and `pyproject.toml` aligned.

#### 3. Samples, benches, and docs ([#58](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/58))

- `testdata/golden/python_api/`: architecture diagram generator, inline invoice sample, compliance smokes, full `generate_samples.py`.
- `make python-api`, `make samples-python`, `make python-benchmarks`.
- `documentation/python.md` plus README / deferred / samples index updates.
- Bulk `output/python/fixture-*.pdf` gitignored; fixture-55 / fixture-56 kept as committed smoke exceptions.

---

### Documentation

| Doc | Link |
|------|------|
| **Site** | https://chinmay-sawant.github.io/gowkhtmltopdf/ |
| **Python** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/documentation/python.md |
| **Overview** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/documentation/overview.md |
| **Getting started** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/documentation/getting-started.md |
| **CLI** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/documentation/cli.md |
| **Library API** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/documentation/library-api.md |
| **Samples** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/output/README.md |
| **Deferred** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/documentation/deferred.md |
| **Changelog** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/CHANGELOG.md#025-2026-08-26 |
| **0.2.5 plans** | https://github.com/chinmay-sawant/gowkhtmltopdf/blob/v0.2.5/plans/0.2.5/README.md |

---

### Known limits (honest)

- v1 one-shot ABI is inline-HTML oriented. File sources are read in Python and passed as HTML with a `file://` base. URL sources and header/footer HTML companions are not on the one-shot path yet.
- `font_paths` / system font flags are not on `GwkPdfOptions`, so `make samples-python` soft-skips fixture-27 (CJK font-path).
- Go default builds remain pure-Go. Rebuilding the shared library from source still needs a C toolchain and `CGO_ENABLED=1`.
