# 40 - v0.2.5 Python cgo c-shared bindings and PyPI (Canonical Execution Ledger)

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md` (Document API + CLI rethink, complete 2026-08-18) and `https://github.com/chinmay-sawant/gowkhtmltopdf/issues/35` (python: cgo c-shared bindings and PyPI in-process)
> **Status:** draft - not started, planning complete 2026-08-26
> **Estimated effort:** 4-6 weeks across phases 40-47, single engineer + CI
> **Constraint:** pure-Go default stays `CGO_ENABLED=0`; cgo is opt-in only for `bindings/c` shared-library build. Direct Go deps stay allowlisted (`go-text/typesetting`, `tdewolff/canvas`) per `internal/pdf/shape_test.go:187` `TestDirectModuleAllowlist` and `Makefile:3-9`.
> **Ordering principle:** correctness and purity guard first, then C ABI and build, then Python API and snippet parity, then platform and wheel matrix, then tests, then docs and security, then PyPI publish, then closure gates. No phase closes on intent.
> **Workflow:** `skills/phase-wise-checklist/SKILLS.md`

---

## Overview

v0.2.4 shipped the idiomatic Go `Document` / `ImageDocument` API (`document.go:99`, `document.go:144`) over the pure-Go pipeline (`internal/convert`, `internal/pdf`, `internal/layout`), plus a redesigned CLI. The Python ecosystem can already call the CLI via subprocess, which is out of scope for this ledger and pays spawn and IPC cost on every conversion.

Issue #35 asks for the missing in-process path: a stable C ABI exported from a Go `c-shared` build (`-buildmode=c-shared`, `CGO_ENABLED=1` only for that target) and an installable Python package on PyPI that loads that library via `ctypes` or `cffi` in the same process. The default Go library and CLI path must remain buildable with `CGO_ENABLED=0` with no regression for existing Go users.

This ledger is the single canonical execution record for that work. Every row is gated on current code and test evidence, not prose. It merges the 10 parallel planning inputs gathered 2026-08-26 (C ABI, c-shared build, Python package, PyPI/wheels, CI tests, docs/security, repo layout purity, Go->Python parity, platform matrix, ABI stability).

Knowledge base entry point for this work is `knowledge-base/wiki/index.md` plus `knowledge-base/wiki/concepts/library-api.md`, `knowledge-base/wiki/architecture.md`, `knowledge-base/wiki/concepts/loader.md`, `knowledge-base/wiki/security-model.md`, and `knowledge-base/wiki/syntheses/roadmap.md`. Committed references are `documentation/library-api.md`, `documentation/deferred.md`, `documentation/THREAT-MODEL.md`, `documentation/architecture/README.md`, and `VERSION`.

---

## Executive Summary

| Fact (current evidence) | Location |
|---|---|
| Project release `VERSION` is `0.2.4` stamped via ldflags `-X .../cli.Version=$(cat VERSION)` | `VERSION:1`, `internal/cli/help.go:10`, `Makefile:43`, `.github/workflows/ci.yml:48`, `.github/workflows/release.yml:68` |
| Settings-surface id `LibraryVersion = "0.12.7-dev"` is distinct from `VERSION` | `api.go:21-28`, `doc.go:58` |
| Public Go API is `Document{Pgs, PageSize, ...}.PDF(ctx)` and `ImageDocument{Source, Width,...}.Image(ctx)` with `Content{HTML,Base,File,URL}` exact-one validation | `document.go:15-47`, `document.go:99-141`, `document.go:240`, `document_validate.go:36` |
| Build is pure-Go `CGO_ENABLED=0` on all hot paths; no `import "C"` in tree today | `Makefile:3-9`, `AGENTS.md:17`, `.github/workflows/ci.yml:37`, `go.mod:8` |
| Allowlist gates only two direct deps; any addition needs sign-off | `internal/pdf/shape_test.go:187`, `Makefile:3-9`, `AGENTS.md:269` |
| No `bindings/` or `python/` directory exists; no `pyproject.toml` exists | `bash ls bindings` miss, `glob python/**/*` empty 2026-08-26 |
| Deferred row says `C ABI (...) | CGO forbidden | Only if consumer demand` | `documentation/deferred.md:78` |
| CI today is single runner `ubuntu-latest`, no cgo, no Python wheel job | `.github/workflows/ci.yml:9-34`, `:36-65` |
| Release builds 6 static targets with `CGO_ENABLED=0` | `.github/workflows/release.yml:69-93` |
| Golden corpus is 61 fixtures, page-envelope hard-fail on missing bound | `internal/convert/golden_test.go:242-410`, `:496`, `testdata/golden/README.md:47` |

---

## Target public API (sketch - freeze in Phase 40 and Phase 42)

### Go snippet (existing, required to mirror)

```go
doc := gowkhtmltopdf.Document{
    Pages: []gowkhtmltopdf.Page{{
        Source: gowkhtmltopdf.Content{
            HTML: []byte(`<html><body><h1>Invoice</h1></body></html>`),
        },
    }},
    PageSize: "A4",
}
pdfBytes, err := doc.PDF(ctx)
```

Source: `document.go:101`, `document.go:71`, `document.go:15`, `document.go:240`.

### Python snippet - Document parity (must ship, same shape)

```python
from gowkhtmltopdf import Document, Page, Content

doc = Document(
    pages=[Page(source=Content(html=b"<html><body><h1>Invoice</h1></body></html>"))],
    page_size="A4",
)
pdf_bytes: bytes = doc.pdf()  # or doc.pdf(timeout=30)
```

Parity table `PageSize` -> `page_size`, `AllowLocalFiles` -> `allow_local_files`, `Margin{Top,Right,Bottom,Left}` -> `Margin(top,right,bottom,left)`. Covers `Content(html, base, file, url)` exact-one, `HeaderFooter`, `TOC`, `Crop` dataclasses. See `knowledge-base/wiki/concepts/library-api.md` and `document.go:27-47`.

### Python snippet - high-level helper (from issue, must also ship)

```python
from gowkhtmltopdf import convert_html_to_pdf, PDFOptions

pdf_bytes = convert_html_to_pdf(
    html=b"<html><body><h1>Invoice #42</h1><p>Total: $19.00</p></body></html>",
    options=PDFOptions(
        page_size="A4",
        orientation="portrait",
        enable_local_file_access=False,
    ),
)

with open("invoice.pdf", "wb") as f:
    f.write(pdf_bytes)
```

Also `convert_file_to_pdf("report.html", "report.pdf", page_size="A4")` file helper. Issue body target caller API is the contract; implementation is `ctypes` loads `libgowkhtmltopdf.so/.dylib/.dll` in-process (`bindings/c` via `-buildmode=c-shared`), not subprocess.

### Python image parity

```python
from gowkhtmltopdf import ImageDocument, Content

png_bytes = ImageDocument(
    source=Content(html=b"<h1>Badge</h1>"),
    width=1024,
    format="png",
).image()

# sugar
from gowkhtmltopdf import convert_html_to_image, ImageOptions
png_bytes = convert_html_to_image(b"<html>...", options=ImageOptions(width=1024, format="png"))
```

### Layers for implementers only (not the caller surface)

```text
App -> high-level Python API (convert_html_to_pdf / Document.pdf)
    -> ctypes/cffi (package internal, stdlib ctypes preferred for v1)
    -> libgowkhtmltopdf.so/.dylib/.dll (CGO_ENABLED=1 -buildmode=c-shared)
    -> Go library (RunPDF / image API via internal/convert / internal/imageout)
```

---

## Phase map

```text
40 C ABI & c-shared build (purity guard)
  -> 41 Python package & loader
    -> 42 Document parity & snippet
      -> 43 Platform & wheel matrix
        -> 44 Tests (C ABI smoke + Python integration + leak)
          -> 45 Docs & security
            -> 46 PyPI publish & versioning
              -> 47 Closure & verification gates
```

All phases depend on Phase 40 laying the isolated build seam; Python tests (44) need both the shared lib and the wrapper; docs (45) need the built API; publish (46) needs wheels and docs; closure (47) needs every prior phase green.

| Phase | File | Goal |
|------:|------|------|
| 40 | [phases/phase-40-c-abi-and-c-shared-build.md](phases/phase-40-c-abi-and-c-shared-build.md) | Freeze C ABI, isolated c-shared build, header, version macro, purity guard |
| 41 | [phases/phase-41-python-package-and-loader.md](phases/phase-41-python-package-and-loader.md) | Scaffold Python package, ctypes loader, buffer ownership, thread/GIL notes |
| 42 | [phases/phase-42-document-parity-and-snippet.md](phases/phase-42-document-parity-and-snippet.md) | Python Document/ImageDocument parity, PDFOptions, convert_* helpers, NetworkPolicy |
| 43 | [phases/phase-43-platform-and-wheel-matrix.md](phases/phase-43-platform-and-wheel-matrix.md) | manylinux/musllinux/macOS/Windows wheels via cibuildwheel |
| 44 | [phases/phase-44-tests-c-abi-and-python-integration.md](phases/phase-44-tests-c-abi-and-python-integration.md) | C ABI smoke, Python integration, leak/free smoke, CI isolation |
| 45 | [phases/phase-45-docs-and-security.md](phases/phase-45-docs-and-security.md) | Build, install, quickstart, ABI stability, security (ACL/NetworkPolicy) |
| 46 | [phases/phase-46-pypi-publish-and-versioning.md](phases/phase-46-pypi-publish-and-versioning.md) | PyPI Trusted Publishing, tag workflow, version alignment, attestations |
| 47 | [phases/phase-47-closure.md](phases/phase-47-closure.md) | Lint, test, golden, build gates green; KB and plans/README sync |

---

## Phase 40: C ABI & c-shared build (purity guard)

### 40.1 ABI contract freeze
- [ ] Define `bindings/c/include/gowkhtmltopdf.h` with `GOWKHTMLTOPDF_ABI_VERSION 1`, `GOWKHTMLTOPDF_VERSION "0.2.5"` macro, `GwkPdfOptions` and `GwkImageOptions` structs with leading `abi_version` + `struct_size` size-gate, documented `timeout_ms`, `base_url`, `allow[]`, `network_policy` fields. Proof: committed header + `grep -n GOWKHTMLTOPDF_ABI_VERSION bindings/c/include/gowkhtmltopdf.h`.
- [ ] Decide one-shot export names `gowkhtmltopdf_html_to_pdf` / `gowkhtmltopdf_html_to_image` + free helpers `gowkhtmltopdf_free` / `gowkhtmltopdf_free_string` + `gowkhtmltopdf_version` + `gowkhtmltopdf_abi_version` + `gowkhtmltopdf_last_error`. Proof: `nm -D dist/libgowkhtmltopdf.so | grep gowkhtmltopdf_`.
- [ ] Document error codes `0 OK, 1 INVALID_ARG, 2 LOAD_DENIED, 3 RENDER_ERROR, 4 TIMEOUT/CANCEL, 5 RESOURCE_LIMIT, 6 INTERNAL` and `char **out_error` vs `last_error` buffer rule. Proof: header comment.

### 40.2 Go //export facade
- [ ] Add `bindings/c/exports_cgo.go` with `//go:build cgo`, `import "C"` and `//export` funcs wrapping `Document.WritePDF` / `ImageDocument.WriteImage` via `convert.Run` / `imageout.RunRequest` (`api.go:105`, `document.go:192`, `document.go:257`). Copies HTML via `C.GoBytes` + `cloneBytes` pattern (`api.go:193`). Proof: `CGO_ENABLED=1 go vet ./bindings/c/...` passes, `CGO_ENABLED=0 go vet ./bindings/c/...` skips file.
- [ ] Add `bindings/c/exports_stub.go` with `//go:build !cgo` stub returning `ENOSYS` so `go test ./...` with `CGO_ENABLED=0` has non-empty package. Proof: `CGO_ENABLED=0 go test ./bindings/c -run ^$ -v` reports skip not build error.
- [ ] Wire `GwkPdfOptions` -> `settings.DefaultPdfGlobal()` selective overrides (`document.go:337`) without importing old dotted `Set` (`internal/cli`). Proof: unit test maps `page_size "A4"` -> `settings.ParsePageSize` and invalid -> `INVALID_ARG`.

### 40.3 Isolated Makefile target
- [ ] Add `c-shared:` target gated on `CGO_ENABLED=1` doing `CGO_ENABLED=1 go build -buildmode=c-shared -ldflags "$(CLI_VERSION_LDFLAGS) -s -w" -o dist/libgowkhtmltopdf.so ./bindings/c` and `file dist/*`. Must not be part of default `make build` / `make test`. Proof: `CGO_ENABLED=0 make build` unchanged, `make c-shared` without env fails with guidance.
- [ ] Add `bindings-clean` or extend `clean` to `rm -rf dist/*`. Proof: `make clean && ls dist`.

### 40.4 Gitignore and drift guard
- [ ] Update `.gitignore` for `/dist/`, `bindings/c/*.so`, `*.a`, `*.h`, `*.dll`, `*.dylib` unless committed header. Add optional committed vendored header `bindings/c/include/gowkhtmltopdf.h` plus `diff` drift check in CI. Proof: `git check-ignore dist/libgowkhtmltopdf.so && echo ignored`.
- [ ] Ensure `CGO_ENABLED=0 go list -json ./... | jq .CgoFiles` is empty for `internal/*` and root. Proof: command output snapshot in phase file closure.

### 40.5 CI c-shared linux amd64 build
- [ ] Add `build-shared` job `runs-on: ubuntu-latest` with `CGO_ENABLED=1` building c-shared and running `nm` smoke + `--version` ldflags check (`internal/cli/help.go:10`). Proof: GH Actions log shows `.so` built and `file dist/libgowkhtmltopdf.so` = ELF shared.

---

## Phase 41: Python package & loader

### 41.1 Scaffold
- [ ] Create `bindings/python/pyproject.toml` (or `python/` if PyPI tooling prefers top-level) with `[project] name = "gowkhtmltopdf"` (fallback `gowkhtmltopdf-python` if PyPI name taken), `dynamic = ["version"]` reading `VERSION`, `readme = "README.md"`, `requires-python >=3.8`, `src` layout `src/gowkhtmltopdf/`. Proof: `python -m build --sdist --wheel` exits 0, `twine check dist/*` clean.
- [ ] Add `bindings/python/src/gowkhtmltopdf/__init__.py`, `_lib.py`, `py.typed`. Proof: `pip install -e bindings/python` imports.

### 41.2 Loader
- [ ] Implement `_lib.py` stdlib `ctypes` loader: `importlib.resources` + `pathlib` candidates, platform switch `libgowkhtmltopdf.so` / `.dylib` / `.dll`, `ctypes.CDLL` with `RTLD_LOCAL`, `argtypes`/`restype` for exports. No `cffi` dep in v1. Proof: `python -c "import gowkhtmltopdf._lib; print(gowkhtmltopdf._lib.lib)"` loads on linux.
- [ ] Copy-built `.so` discovery: `python -c "from importlib.resources import files; print(files('gowkhtmltopdf'))"`. Proof: wheel `auditwheel show` lists `libgowkhtmltopdf.so` next to package.

### 41.3 Memory ownership
- [ ] Go allocates PDF bytes via `C.CBytes` / `C.malloc` and exposes `gowkhtmltopdf_free`; Python copies via `ctypes.string_at(ptr, len)` then calls `free`. Error strings via `C.CString` + `gowkhtmltopdf_free_string`. Document borrowed `html` ptr/len lifetime (caller may free after call). Proof: header comment `who frees`.

### 41.4 Thread/GIL
- [ ] Document per-handle non-thread-safety: `Document` not concurrent on same instance (`documentation/library-api.md:302`), distinct handles may run concurrently. Long renders release GIL via `ctypes` thread or doc note; Python holds `threading.Lock` per render or `Py_BEGIN_ALLOW_THREADS` rationale. Proof: doc note in `documentation/python.md`.

---

## Phase 42: Document parity & snippet (user-required)

### 42.1 Dataclasses
- [ ] Add `Content(html, base, file, url)` with exact-one validation (`ErrInvalidContent` + `ErrEmptyHTML`, `Base` only with `html` per `document_validate.go:136`). Helpers `Content.html()/file()/url()` mirror `document.go:27-47`. Proof: `pytest -k test_content_validation`.
- [ ] Add `Page(source, header, footer, include_in_outline, external_links, local_links, zoom)` (`document.go:71`), `Margin(top,right,bottom,left)`, `HeaderFooter`, `TOC`, `Crop` (`document.go:49-97`). Proof: import smoke + round-trip to Go options.
- [ ] Add `Document(pages, cover, toc, page_size, width_mm, height_mm, orientation, margin, title, pdf_version, pdf_profile, copies, collate, outline, outline_depth, background, smart_shrinking, compression, resolve_relative_links, grayscale, page_offset, exclude_from_outline, header, footer, allow, allow_local_files, font_paths, use_system_fonts, network)` (`document.go:101`). Proof: `Document(...).validate()` exercises `ErrInvalidPageSize`.
- [ ] Add `ImageDocument(source, width, height, format, quality, smart_width, transparent, crop, zoom)` (`document.go:144`, `document_validate.go:66`). Proof: `ImageDocument(...).validate()` rejects bad format.

### 42.2 High-level helpers (issue contract)
- [ ] Implement `convert_html_to_pdf(html: bytes|str, options: PDFOptions|None, **kwargs) -> bytes`, `convert_file_to_pdf(path, **kwargs) -> bytes`, `convert_url_to_pdf(url, **kwargs)`. `PDFOptions` dataclass desugars to `Document(pages=[Page(...)])`. Proof: `python -c "from gowkhtmltopdf import convert_html_to_pdf; assert convert_html_to_pdf(b'<h1>Invoice</h1>').startswith(b'%PDF-')"`.
- [ ] Implement `PDFOptions(page_size="A4", orientation="portrait", enable_local_file_access=False, allow=[], base_url=None, timeout_ms=0, title=None, pdf_version=None, pdf_profile=None, ...)` and map `timeout_ms` to `context.WithTimeout` on Go side plus `LoadPage.Timeout` seconds (`internal/load/load.go:1380`). Proof: timeout test triggers code `4 TIMEOUT`.
- [ ] Implement `convert_html_to_image` / `ImageOptions` parity. Proof: `convert_html_to_image(b"<h1>Badge</h1>", options=ImageOptions(width=1024, format="png")).startswith(b"\x89PNG")`.

### 42.3 Error and policy mapping
- [ ] Map Go sentinels (`ErrInvalidContent`, `ErrNoPageObjects`, `ErrInvalidPageSize`, `ErrMissingPDFOutput`, network `ErrAccessDenied` via `internal/load/load.go:48`) to typed Python `GowkhtmltopdfError -> ValidationError / RenderError / NetworkPolicyError / FileAccessError`. Expose `exc.sentinel`. Proof: `pytest -k test_sentinels`.
- [ ] Expose `NetworkPolicy(allowed_schemes, allowed_hosts, block_private_networks, block_cross_host_redirects)` plus `compatible_network_policy()` / `restricted_network_policy()` (`api.go:34-42`, `internal/load/load.go:123-138`) and wire via `load.ApplyNetworkPolicy` (`document.go:406`). Proof: `restricted_network_policy()` blocks private IP fixture without `allow`.
- [ ] Expose `__version__` (project `VERSION`) and `library_version` (`api.go:23` `0.12.7-dev`). Proof: `python -c "import gowkhtmltopdf; assert gowkhtmltopdf.__version__ == open('VERSION').read().strip()"`.

### 42.4 Snippet gate
- [ ] Ship runnable example `bindings/python/examples/invoice.py` demonstrating both the Document-parity and the high-level helper snippets from Target API, byte-for-byte comparison. Proof: `python bindings/python/examples/invoice.py && file invoice.pdf`.

---

## Phase 43: Platform & wheel matrix

### 43.1 linux manylinux
- [ ] Configure `cibuildwheel` for `manylinux_2_28_x86_64` with `CIBW_BEFORE_BUILD` installing `go1.26` in manylinux container and running `CGO_ENABLED=1 go build -buildmode=c-shared -o src/gowkhtmltopdf/libgowkhtmltopdf.so ./bindings/c`. Run `auditwheel repair` + `twine check`. Proof: `auditwheel show dist/*manylinux*.whl` lists `libgowkhtmltopdf.so` and tag `manylinux_2_28_x86_64`.
- [ ] Also build `musllinux_1_2_x86_64` if demand. Proof: wheel tag.

### 43.2 linux arm64
- [ ] Add `linux aarch64` via `cibuildwheel` emulation or native `ubuntu-24.04-arm` runner + `aarch64-linux-gnu-gcc` or `zig cc -target aarch64-linux-gnu`. Proof: `file bindings/python/src/gowkhtmltopdf/libgowkhtmltopdf.so` reports `aarch64`.

### 43.3 darwin
- [ ] Add `macos-13` (x86_64) and `macos-14` (arm64) runners building `.dylib` with `CGO_ENABLED=1`. `delocate-wheel` if needed. Proof: `otool -L` and `file *.dylib`.

### 43.4 windows
- [ ] Add `windows-2022` amd64 building `.dll` via `mingw-w64`/`msys2` gcc with `CGO_ENABLED=1`. Proof: `dumpbin /headers` or `file *.dll`.

### 43.5 sdist
- [ ] sdist includes `bindings/c/*.go` + committed header `bindings/c/include/gowkhtmltopdf.h` so install without `auditwheel` can rebuild. Exclude `docs/`, `frontend/dist`, `knowledge-base`. Proof: `tar tzf dist/*.tar.gz | grep bindings/c`.

---

## Phase 44: Tests (C ABI smoke + Python integration + leak)

### 44.1 C ABI smoke
- [ ] Add `bindings/c/cshared_test.go` with `//go:build cgo` that calls exports on `<!DOCTYPE html><h1>Hello</h1>` and on `testdata/golden/fixture-01-simple-invoice.html` (`internal/convert/golden_test.go:243`), asserting `out_len>0`, `bytes.HasPrefix(b, []byte("%PDF-"))` (`golden_test.go:175`), `%%EOF` (`golden_test.go:179`), `xref` prefix (`golden_test.go:204`), and optional `pdf.ParseSemantic` page count. Proof: `CGO_ENABLED=1 go test -tags cgo ./bindings/c -run TestCShared -count=1 -v` green.
- [ ] Also cover failure: null html -> `INVALID_ARG`, no `malloc` on error, `out_pdf==NULL, *out_len==0`. Proof: same test.

### 44.2 Python integration
- [ ] Add `bindings/python/tests/test_binding.py` (pytest) invoking `gowkhtmltopdf.convert_html_to_pdf` on inline `<h1>Invoice #42</h1>` and on `fixture-01-simple-invoice.html` file content, asserting `b"%PDF-"` header, `len>1024`, `b"%%EOF" in pdf`, `/FontFile2` present (`golden_test.go:221`). Proof: `pytest bindings/python/tests -v -k test_convert_header`.

### 44.3 Leak/free smoke
- [ ] Loop 1000x `ptr,len = Render(); assert ptr!=nil; Free(ptr); Free(nil)` no crash; optional `valgrind --leak-check=full` or `ASAN_OPTIONS` in CI. Verify `C.CString`/`C.free` paired same allocator, no double-free. Proof: CI job `leak-smoke` passes.

### 44.4 CI isolation
- [ ] Add `python-binding` job with `env: CGO_ENABLED: 1` only in that job, `setup-go 1.26` + `setup-python 3.11` + `pip install -e bindings/python`, then `go test -tags cgo ./bindings/c` + `pytest`. Keep existing `test+lint` and `static build (CGO_ENABLED=0)` jobs untouched (`ci.yml:9-65`). Proof: GH log shows two jobs, one `CGO_ENABLED=0` still green.

---

## Phase 45: Docs & security

### 45.1 Committed docs
- [ ] Add `documentation/python.md` covering: how to build c-shared locally (`CGO_ENABLED=1 go build -buildmode=c-shared -ldflags "-X .../cli.Version=$(cat VERSION)" -o libgowkhtmltopdf.so ./bindings/c`), Python install quickstart in-process (both helper and Document parity snippets), ABI versioning/stability (additive-only within major, `size`/`abi_version` gate), security notes with same ACL/local-file/network rules as Go API (`AllowLocalFiles`, `Allow`, `NetworkPolicy`), and explicit contrast that CLI subprocess is out-of-scope for this ticket (use binding for in-process). Proof: `make claim-scan` passes including `frontend/src/data/content` if present.
- [ ] Update `documentation/README.md:27-40` guides table with `documentation/python.md` row and security table `52-57`. Update `documentation/deferred.md:78` row from `Only if consumer demand` to shipped note. Proof: grep row.
- [ ] Add one-line teaser + install snippet in root `README.md:54-81` docs index and `documentation/getting-started.md:146-153` security box. Proof: `make claim-scan` clean.

### 45.2 Package docs
- [ ] Add `bindings/python/README.md` for PyPI `long_description` with short install + link to `documentation/python.md` for full ACL/NetworkPolicy/ABI contract. Proof: `python -m build` includes README in METADATA.

### 45.3 Security evidence
- [ ] Document `AccessController.Allowed` prefix wins with `EvalSymlinks` (`internal/load/load.go:248-285`), `file://` host check (`load.go:1184-1204`), default deny (`THREAT-MODEL.md:42-53`). Document `CompatibleNetworkPolicy` vs `RestrictedNetworkPolicy` (`load.go:123-138`), `MaxRedirects 10`, `MaxBodySize 100MiB` (`load.go:41-42`), pinned dial (`load.go:522-580`), wildcard boundary (`load.go:709`). Proof: cited lines reachable from docs.

---

## Phase 46: PyPI publish & versioning

### 46.1 Auth
- [ ] Configure PyPI Trusted Publishing OIDC for `gowkhtmltopdf` (add `permissions: id-token: write` + `contents: write` to workflow, `pypa/gh-action-pypi-publish@release/v1` with `attestations: true`), secret fallback `PYPI_API_TOKEN`. Proof: `https://pypi.org/manage/account/publishing` entry matches repo.
- [ ] Document name decision: try `gowkhtmltopdf` PEP-503 normalized, else fallback `gowkhtmltopdf-python` / `python-gowkhtmltopdf` with import name `gowkhtmltopdf`. Proof: `curl https://pypi.org/pypi/gowkhtmltopdf/json` 404 -> name free check in release notes.

### 46.2 Publish workflow
- [ ] Add `publish-pypi.yml` on `push tags: v*` plus `workflow_run: release` that does `python -m build` (sdist+wheel) -> `twine check --strict dist/*` -> `auditwheel show` -> `check-wheel-contents` -> `pypa/gh-action-pypi-publish`. Keep `VERSION` file as single source (`release.yml:52-57` gate). Proof: GH run uploads on `git tag v0.2.5 && git push origin v0.2.5` dry-run with ` --skip-existing`.

### 46.3 Version alignment
- [ ] Wire `pyproject.toml` `dynamic = ["version"]` via `setuptools_scm` reading `VERSION` or `hatch-vcs` tag; wheel `METADATA Version` is PEP 440 normalized `tr -d '[:space:]' < VERSION` (`ci.yml:47`). Map prerelease `v0.3.0-alpha.1` -> `0.3.0a1`. Extend `TestCLIVersionMatchesVERSIONFile` pattern (`internal/cli/cli_test.go:297`) to `python -c "import gowkhtmltopdf; assert __version__ == open('VERSION').read().strip()"`. Proof: CI version check exits 0.

---

## Phase 47: Closure & verification gates

### 47.1 Default pure-Go path still green
- [ ] `CGO_ENABLED=0 go test ./...` full suite green. Proof: `make test` exit 0 log.
- [ ] `go test -race -count=1 ./internal/convert ./internal/layout ./internal/pdf ./internal/imageout ./internal/load` green (`ci.yml:77`). Proof: command log.
- [ ] `CGO_ENABLED=0 go build -trimpath -ldflags "-X .../cli.Version=$(cat VERSION)" -o /tmp/gowkhtmltopdf ./cmd/gowkhtmltopdf && /tmp/gowkhtmltopdf --version` matches `VERSION`. Proof: `test "${got}" = "${want}"` (`ci.yml:61`).

### 47.2 Lint and scans
- [ ] `make lint` (`GOLANGCI_LINT_VERSION v1.64.8` + `lint-frontend`) clean on pure-Go tree; cgo package linted separately with `CGO_ENABLED=1 golangci-lint run ./bindings/c/...` or explicit `//nolint` with reason. Proof: `make lint` exit 0.
- [ ] `make claim-scan` clean across `README.md`, `documentation/*.md`, `documentation/python.md`, `frontend/src/data/content`, `internal/cli/help.go`. Proof: exit 0 (`Makefile:51`).

### 47.3 Golden corpus
- [ ] `make golden` (`go test ./internal/convert -run TestGoldenCorpus -v`) green on 61 fixtures (`golden_test.go:452`), `%PDF-` header, `/FontFile2`, xref/EOF, per-fixture page envelopes (`fixturePageBounds`), feature flags `images`/`uris`. Proof: `make golden` exit 0.

### 47.4 Docs and plan sync
- [ ] `frontend production build` clean (`npm ci && npm run build` + dirty check `frontend/dist`, `docs`). Proof: CI `frontend` job green.
- [ ] Update `plans/README.md` with `0.2.5 python bindings` row pointing to this canonical ledger; archive font and woff2 tracks as `[~]` with pointer if moved; update `knowledge-base/wiki/index.md` and `wiki/log.md` (append-only entry), `wiki/syntheses/roadmap.md` milestone table; keep `knowledge-base/` gitignored but in sync per `AGENTS.md:116`. Proof: `git status --porcelain -- plans/README.md knowledge-base` diff.

### 47.5 Release readiness
- [ ] `VERSION` + `CHANGELOG.md` + `documentation/MIGRATION-0.2.5.md` (if any break is nil for Python track) agree; `git tag v0.2.5` passes `release.yml:52-57` mismatch gate; `publish-pypi.yml` dry-run passes `twine check`. Proof: `cat VERSION`, `grep -n 0.2.5 CHANGELOG.md`.

---

## Dependencies

| Depends on | Provides to |
|---|---|
| `plans/0.2.4/31-canonical-0.2.4-roadmap.md` and `document.go:101` Document contract | Stable field names for C ABI and Python parity |
| Phase 40 | ABI header for 41-46; build seam for all wheels |
| Phase 41 | Loader for 42 and test harness for 44 |
| Phase 42 | High-level API surface for docs 45 and publish 46 quickstart |
| Phase 43 | Wheels for publish 46 |
| Phase 44 | CI evidence for 47 closure |
| Phase 45 + 43 | Evidence for 46 publish notes |
| Phase 46 | PyPI artifact for 47 VERSION tag |

---

## Out of scope (unless this ledger is amended)

- Thin Python wrappers that only `subprocess` the CLI binary (no ticket needed; already doable).
- Reimplementing the layout or PDF engine in Python.
- Full wkhtmltopdf flag parity on day one (high-traffic subset only: `page_size`, `orientation`, `margin`, `title`, `pdf_version`/`pdf_profile`, `allow`/`allow_local_files`, `base_url`/`network_policy`, `timeout_ms`).
- PHP / Ruby / Node bindings (can reuse C ABI later in separate tickets).
- Making cgo required for the main module, default CI `CGO_ENABLED=0` builds, or pure-Go library import path.
- Pixel-golden harness inside the Python package (Go suite remains source of truth, `testdata/golden/README.md:112`).
- Conda-forge / distro packages (follow-up).

---

## Evidence Rules

- Prefer current code, tests, benchmarks, scanner output, and CI config over historical notes (`skills/phase-wise-checklist/SKILLS.md`).
- State negative results precisely: e.g. "no production `unsafe` beyond X found in this audit", never "no bugs exist".
- Keep risk statements separate from confirmed defects. Label hypotheses and give validation step.
- For performance items, include exact release command, dataset/path, cold/warm cache state, and metric; successful execution is not benchmark proof.
- Every row closes only after matching `make test` / `make lint` / `make golden` or `pytest` / `auditwheel` / `twine check` exit 0, recorded beside the gate.

---

## Body record and branch

- Body record: `plans/PR/issues/issue-python-native-pypi-body.md` (filled after issue number exists; for #35 already `feat/python-cgo-bindings-pypi` is suggested).
- Suggested local branch: `feat/python-cgo-bindings-pypi` (`AGENTS.md:4` branch naming `feature/` / `fix/` / `chore/` / `docs/` lowercase).

---

## Completion Handoff

Before declaring a phase complete, confirm its rows, run the smallest relevant validation (`CGO_ENABLED=0 make test` for purity, `CGO_ENABLED=1 go test ./bindings/c` for ABI, `pytest` for Python, `auditwheel show` for wheels), synchronize the checklist with `[x]` only after evidence, and provide a concise result plus the next unchecked phase. Do not create a second status document unless it is an explicit deferred ledger with a pointer from this canonical plan.

## Required Checks

- For documentation-only changes, do not run lint or test checks.
- For every non-documentation change, run `make lint` and `make test` before marking the phase complete. Record both outcomes in this ledger; leave the row unchecked if either command fails (`skills/phase-wise-checklist/SKILLS.md:56`).
