# Plans - v0.2.5 (Python cgo c-shared bindings and PyPI + Font tracks)

| File / Folder | Role |
|---------------|------|
| [40-canonical-0.2.5-python-bindings.md](40-canonical-0.2.5-python-bindings.md) | **Canonical v0.2.5 execution ledger** for issue #35 python cgo `c-shared` bindings and PyPI (phases 40-47) |
| [phases/](phases) | Per-phase atomic checklists for python bindings: C ABI and c-shared build (40), Python package and loader (41), Document parity and snippet (42), platform and wheel matrix (43), tests (44), docs and security (45), PyPI publish and versioning (46), closure (47) |
| [PR/release-v0.2.5.md](PR/release-v0.2.5.md) | GitHub Release body for tag `v0.2.5` |
| [../PR/pr-python-cgo-pypi.md](../PR/pr-python-cgo-pypi.md) | PR #58 body (feature branch) |
| `font/` | **Complete** (2026-08-19) font resolution track phases 01-08 (`FontResolver`, discovery diagnostics, `@font-face` weight/style, preflight). Not in this repo worktree but cited in `knowledge-base/wiki/syntheses/roadmap.md` and `wiki/syntheses/fonts-and-typography.md`. `VERSION` stayed `0.2.4` until tag |
| `woff2-metric-aliases/` lives under `plans/0.2.6/` | **Complete** (2026-08-20) WOFF2 decode plus opt-in metric aliases |

Workflow: [`../../skills/phase-wise-checklist/SKILLS.md`](../../skills/phase-wise-checklist/SKILLS.md)

Predecessor: [`../0.2.4/31-canonical-0.2.4-roadmap.md`](../0.2.4/31-canonical-0.2.4-roadmap.md) (Document API + CLI rethink, complete 2026-08-18).
Issue: [`#35 python: cgo c-shared bindings and PyPI (in-process)`](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/35).
Body record after issue start: `../PR/issues/issue-python-native-pypi-body.md` (suggested branch `feat/python-cgo-bindings-pypi`).

## Scope in one line

Keep the `CGO_ENABLED=0` pure-Go engine and `Document` API unchanged; add an **isolated** `c-shared` build (`bindings/c` with `//go:build cgo` only) exporting a versioned C ABI, and a `ctypes`-based Python package on PyPI that loads that library in-process. CLI subprocess remains possible but is out of scope for this ticket.

## Python snippet parity (user-required)

Go and Python must look the same:

```go
doc := gowkhtmltopdf.Document{
    Pages: []gowkhtmltopdf.Page{{
        Source: gowkhtmltopdf.Content{HTML: []byte(`<html><body><h1>Invoice</h1></body></html>`)},
    }},
    PageSize: "A4",
}
pdfBytes, err := doc.PDF(ctx)
```

```python
from gowkhtmltopdf import Document, Page, Content

doc = Document(
    pages=[Page(source=Content(html=b"<html><body><h1>Invoice</h1></body></html>"))],
    page_size="A4",
)
pdf_bytes: bytes = doc.pdf()
```

Plus high-level helper:

```python
from gowkhtmltopdf import convert_html_to_pdf, PDFOptions

pdf_bytes = convert_html_to_pdf(
    html=b"<html><body><h1>Invoice #42</h1></body></html>",
    options=PDFOptions(page_size="A4", orientation="portrait"),
)
```

Both snippets are contract in `40-canonical-0.2.5-python-bindings.md` Target API and in `phases/phase-42-document-parity-and-snippet.md`.

## Module and CI locality

- Go module stays `github.com/chinmay-sawant/gowkhtmltopdf`; `go test ./...` with `CGO_ENABLED=0` remains the default CI gate (`ci.yml:24`).
- `bindings/c` is never imported by `internal/*` or root; only the opt-in `c-shared` Makefile target and a dedicated `build-shared (CGO_ENABLED=1)` CI job touch `import "C"`.
- Wheels via `cibuildwheel` manylinux_2_28 x86_64 first, then arm64, darwin, windows. sdist includes `bindings/c` for rebuild.

## Verification

- C ABI smoke (`bindings/c/cshared_test.go` with `//go:build cgo`) asserts `%PDF-`, `%%EOF`, `xref`, `/FontFile2` on `fixture-01-simple-invoice.html`
- Python integration `pytest bindings/python/tests -k test_convert_header` and leak/free loop
- `make test`, `make lint`, `make golden`, `make claim-scan` still green on `CGO_ENABLED=0` (Phase 47)

## References

- Library API: `api.go:1`, `document.go:15-101`, `knowledge-base/wiki/concepts/library-api.md`
- Architecture DAG: `knowledge-base/wiki/architecture.md:16`
- Security: `knowledge-base/wiki/concepts/loader.md`, `knowledge-base/wiki/security-model.md`, `documentation/THREAT-MODEL.md`
- Deferred C ABI row: `documentation/deferred.md:78`
