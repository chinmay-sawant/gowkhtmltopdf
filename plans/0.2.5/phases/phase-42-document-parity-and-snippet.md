# Phase 42: Document parity & snippet (user-required)

> **Parent:** `../40-canonical-0.2.5-python-bindings.md` Phase 42
> **Status:** not started
> **Estimated effort:** 6-8 days
> **Owner:** bindings/python

---

## Overview

Ship the Python `Document`/`ImageDocument` parity layer and the two snippet forms the user explicitly required: Go `doc := Document{Pages:[]Page{{Source:Content{HTML:...}}}, PageSize:"A4"}; pdfBytes,err:=doc.PDF(ctx)` mirrored in Python both as `Document(...).pdf()` and as `convert_html_to_pdf(..., PDFOptions(page_size="A4"))`. This phase maps every Go field (`document.go:101`) to snake_case Python and keeps exact-one `Content` validation.

## Goals

- Import `Document`, `Page`, `Content`, `Margin`, `HeaderFooter`, `TOC`, `Crop`, `NetworkPolicy` with `errors.Is` parity
- Both snippet forms produce the same `%PDF-` bytes via the in-process lib, no subprocess

## Checklist

### 42.1 Dataclasses (`src/gowkhtmltopdf/document.py`)
- [x] 42.1.1 `Content(html: bytes|str|None, base: str|None, file: str|None, url: str|None)` exact-one validation: zero -> `ErrEmptyHTML` (`document_validate.go:129` wraps `ErrInvalidContent`), multiple -> "exactly one of HTML, File, or URL is required", `base` only with `html` (`document_validate.go:136`). Helpers `Content.html()/file()/url()` copy bytes (`document.go:27`). Proof: `pytest -k test_content_validation` covers 4 cases.
- [x] 42.1.2 `Page(source, header, footer, include_in_outline, external_links, local_links, zoom)` (`document.go:71`); `header/footer` nil means inherit doc value, non-nil sets `HeaderSet/FooterSet` (`settings.go:366`). Proof: `Page(source=Content(html=b"<h1>ok</h1>"), header=HeaderFooter(center="Page [page]"))` round-trips.
- [x] 42.1.3 `Margin(top,right,bottom,left)` mm floats (`document.go:49`), `HeaderFooter(left,center,right,font_size,font_name,line,spacing,html_url,replace)`, `TOC(caption,dotted_lines,font_scale,indentation,forward_links,back_links)` (`document.go:81-89`), `Crop(left,top,width,height)` (`document.go:91`). Proof: import smoke + serialization to `GwkPdfOptions`.
- [x] 42.1.4 `Document(pages, cover, toc, page_size, width_mm, height_mm, orientation, margin, title, pdf_version, pdf_profile, copies, collate, outline, outline_depth, background, smart_shrinking, compression, resolve_relative_links, grayscale, page_offset, exclude_from_outline, header, footer, allow, allow_local_files, font_paths, use_system_fonts, network, now, on_info, ...)` (`document.go:101`). Zero-valued retains engine defaults via `DefaultPdfGlobal` (`settings.go:459`). Proof: `Document(pages=[Page(...)], page_size="A4").validate()` ok, `Document(pages=[]).validate()` raises `ErrNoPageObjects` (`document_validate.go:57`).
- [x] 42.1.5 `ImageDocument(source, width, height, format, quality, smart_width, transparent, crop, zoom, allow, allow_local_files, background, font_paths, use_system_fonts, network)` (`document.go:144`, `document_validate.go:66` format `png|jpg|jpeg`). Proof: `ImageDocument(source=Content(html=b"<h1>Badge</h1>"), width=1024).validate()` ok.

### 42.2 High-level helpers (`src/gowkhtmltopdf/__init__.py`)
- [x] 42.2.1 `convert_html_to_pdf(html: bytes|str, options: PDFOptions|None, **kwargs) -> bytes` builds `Document(pages=[Page(source=Content(html=...))], **options.dict())` then `doc.pdf()` via ctypes one-shot `gowkhtmltopdf_html_to_pdf`. `PDFOptions` dataclass is subset (`page_size`, `orientation`, `margin`, `title`, `pdf_version`, `pdf_profile`, `copies`, `allow`, `allow_local_files`, `base_url`, `timeout_ms`, `network`). Proof: `python -c "from gowkhtmltopdf import convert_html_to_pdf; assert convert_html_to_pdf(b'<h1>Invoice</h1>').startswith(b'%PDF-')"`.
- [x] 42.2.2 `convert_file_to_pdf(path, **options)` and `convert_url_to_pdf(url, **options)` plus file helper `convert_file_to_pdf("report.html", "report.pdf", page_size="A4")` writes bytes to path. Proof: `convert_file_to_pdf("testdata/golden/fixture-01-simple-invoice.html", page_size="A4").startswith(b"%PDF-")` when `allow_local_files=True` internally.
- [x] 42.2.3 `timeout` kwarg `float|None` seconds maps to Go `context.WithTimeout` plus `LoadPage.Timeout` seconds (`load.go:1380`, `DefaultResponseTimeout 60s` `load.go:40`). Proof: `convert_html_to_pdf(b"<h1>slow</h1>", timeout=0.001)` returns code `4 TIMEOUT`.
- [x] 42.2.4 `convert_html_to_image` / `ImageOptions(width, height, format="png", quality=94, smart_width=True, transparent=False, crop=None, zoom=0)` mirrors `ImageGlobal` (`settings.go:587` defaults). Proof: `convert_html_to_image(b"<h1>Badge</h1>", options=ImageOptions(width=1024)).startswith(b"\\x89PNG")`.

### 42.3 Errors and policy
- [x] 42.3.1 Map Go sentinels (`ErrInvalidContent`, `ErrEmptyHTML`, `ErrNoPageObjects`, `ErrInvalidPageSize`, `ErrInvalidOrientation`, `ErrMissingPDFOutput`, `ErrInvalidPDFVersion/Profile` from `api.go:49` and `document_validate.go:13`) to Python `GowkhtmltopdfError -> ValidationError/RenderError/NetworkPolicyError/FileAccessError` with `exc.sentinel` attribute for `errors.Is` style `isinstance(exc, gowkhtmltopdf.ErrInvalidContent)`. Proof: `pytest -k test_sentinels` covers `ErrNoPageObjects` and `ErrInvalidPageSize`.
- [x] 42.3.2 `NetworkPolicy(allowed_schemes, allowed_hosts, block_private_networks, block_cross_host_redirects)` plus `compatible_network_policy()` / `restricted_network_policy()` (`api.go:34-42`, `load.go:123-138`) wired via `ApplyNetworkPolicy` (`document.go:406`). Proof: `restricted_network_policy()` blocks `http://127.0.0.1` fixture without allow.
- [x] 42.3.3 `__version__` from `VERSION:1` and `library_version` from `api.go:23` (`0.12.7-dev`) plus `abi_version()`. Proof: `python -c "import gowkhtmltopdf; assert __version__ == open('VERSION').read().strip()"`.

### 42.4 Snippet gate (user-required proof)
- [x] 42.4.1 Ship `bindings/python/examples/invoice.py` that runs both snippets back-to-back and writes `invoice.pdf` and `invoice_high_level.pdf`, each starting `%PDF-` and containing "Invoice". Proof: `python bindings/python/examples/invoice.py && pdfinfo invoice.pdf | grep Pages`.
- [x] 42.4.2 Include side-by-side Go/Python snippets in `documentation/python.md` mirroring Target API. Proof: doc contains both code blocks and renders via `make claim-scan`.

### 42.5 Methods
- [x] 42.5.1 `Document.validate()`, `Document.pdf(timeout=None)->bytes`, `Document.write_pdf(fileobj, timeout=None)`, `Document.write_pdf_with_outline(pdfWriter, outlineWriter)` (`document.go:192-243`), and `ImageDocument.image()/write_image` (`document.go:301`). No Go `context` param; optional `timeout` only. Proof: `doc.pdf()` returns owned `bytes` distinct on second call (copy semantics `document.go:250`).

## Dependencies

Depends on Phase 40 ABI and Phase 41 loader.

## Evidence

- `pytest bindings/python/tests -v` 8+ tests green
- `convert_html_to_pdf(b"<h1>Invoice</h1>", options=PDFOptions(page_size="A4"))` matches `Document(...).pdf()` byte prefix `%PDF-`

## Out of scope

Cover/TOC/header-footer image raster plumbing beyond dataclass (rendered by engine via `GwkPdfOptions` deferred if not in v1); streaming callbacks `OnInfo/OnPhase/OnProgress` (`api.go:74`).

## Handoff

Next is Phase 43 wheel matrix.
