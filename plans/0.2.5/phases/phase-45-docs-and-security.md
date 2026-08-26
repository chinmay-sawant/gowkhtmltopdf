# Phase 45: Docs & security

> **Parent:** `../40-canonical-0.2.5-python-bindings.md` Phase 45
> **Status:** not started
> **Estimated effort:** 3-4 days
> **Owner:** documentation + bindings/python

---

## Overview

Make the in-process binding discoverable and safe: how to build the c-shared lib locally, how to install and call Python in-process, what the ABI stability promise is, and what security rules match the Go engine (`knowledge-base/wiki/concepts/loader.md`, `knowledge-base/wiki/security-model.md`, `internal/load/load.go`).

## Goals

- Committed docs are the truth; `knowledge-base/` is gitignored local mirror (`AGENTS.md:116`, `.gitignore:35`)
- One snippet proves copy-paste works (`invoice.py`)
- Security notes match Go exactly so Python callers do not accidentally widen `AllowLocalFiles`

## Checklist

### 45.1 Committed docs (`documentation/`)
- [x] 45.1.1 Create `documentation/python.md` covering **build c-shared**: `CGO_ENABLED=1 go build -buildmode=c-shared -ldflags "-X github.com/chinmay-sawant/gowkhtmltopdf/internal/cli.Version=$(cat VERSION)" -o libgowkhtmltopdf.so ./bindings/c` (contrast `Makefile:43` stamp, `VERSION:1` = `0.2.5`), plus `make c-shared` short path, platform notes `linux .so`, `darwin .dylib`, `windows .dll`, and `libgowkhtmltopdf.h` output location. Proof: `grep -n "CGO_ENABLED=1.*buildmode=c-shared" documentation/python.md`.
- [x] 45.1.2 **Python install / quickstart (in-process only)** mirroring `documentation/library-api.md:20-69` pattern and both snippets from canonical Target API: high-level `convert_html_to_pdf` with `PDFOptions(page_size="A4")` and `Document(pages=[Page(...)], page_size="A4").pdf()`, plus `convert_file_to_pdf` helper and `ImageDocument` quickstart. Proof: `grep -n "convert_html_to_pdf" documentation/python.md`.
- [x] 45.1.3 **ABI versioning / stability**: two versions are `VERSION` project release (`VERSION:1`, `internal/cli/help.go:10`) vs `LibraryVersion 0.12.7-dev` (`api.go:23`), plus `GOWKHTMLTOPDF_ABI_VERSION 1` header macro; semver `MAJOR` bump = breaking export/struct removal, `MINOR` additive, `PATCH` no ABI change; additive-only exports enforced via `nm -D` snapshot; structs carry `size` + `abi_version` for size-gate. Proof: `grep -n ABI_VERSION documentation/python.md`.
- [x] 45.1.4 **Security notes (same as Go)**: `Allow []string` + `AllowLocalFiles bool` (`document.go:129-133`, `api.go:30-42`) default deny with prefix wins and `EvalSymlinks` (`load.go:248-285`), `file://` host `""`/`localhost` only (`load.go:1184-1204`), `NetworkPolicy` identical (`internal/load/load.go:108-138`, `api.go:30-42`) `CompatibleNetworkPolicy` (`load.go:123-127`) vs `RestrictedNetworkPolicy` (`load.go:132-138`) with pinned dial (`load.go:522-580`), limits `ConnectTimeout 30s`/`ResponseTimeout 60s`/`MaxBodySize 100MiB`/`MaxRedirects 10` (`load.go:38-43`), `Base` only with `HTML` (`document_validate.go:136`). Recommend `Allow: ["/var/app/templates"]` over `AllowLocalFiles:true`. Proof: citations above reachable from docs.
- [x] 45.1.5 **Contrast note in-process vs CLI**: state `ctypes` loads `libgowkhtmltopdf.so` in-process, not `subprocess.Popen(["gowkhtmltopdf",...])`; CLI flags/page grammar `documentation/cli.md` are not the Python surface; `documentation/deferred.md:78` from `Only if consumer demand` to shipped note. Proof: `grep -n "subprocess.*not" documentation/python.md`.

### 45.2 Docs index updates
- [x] 45.2.1 Add row for `documentation/python.md` in `documentation/README.md:27-40` guides table and security row `52-57` referencing `python.md`. Proof: `grep python documentation/README.md`.
- [x] 45.2.2 Add one-paragraph teaser + install snippet in root `README.md:54-81` docs index and `documentation/getting-started.md:146-153` remote-URL security box. Proof: `grep python README.md`.
- [x] 45.2.3 Update `documentation/deferred.md:78` row `C ABI (...)` to `Shipped 0.2.5: c-shared + PyPI via bindings/c` rather than `Only if consumer demand`. Proof: `grep -n "C ABI" documentation/deferred.md`.

### 45.3 Package docs
- [x] 45.3.1 Add `bindings/python/README.md` for PyPI `long_description` with short `pip install gowkhtmltopdf` + `import gowkhtmltopdf; convert_html_to_pdf(b"<h1>Invoice</h1>")` and link to `documentation/python.md` for ACL/NetworkPolicy/ABI contract; ensure `make claim-scan` (`Makefile:51-62` forbids `using only the standard library`, `Qt WebKit` etc) passes including `frontend/src/data/content` and `internal/cli/help.go` if scanned. Proof: `python -m build` METADATA contains README.
- [x] 45.3.2 Provide runnable `bindings/python/examples/invoice.py` (already Phase 42.4) and reference from docs. Proof: doc link.

### 45.4 Claims and lint
- [x] 45.4.1 `make claim-scan` clean across new docs; update scan globs if `bindings/python/README.md` needed. Proof: exit 0.
- [x] 45.4.2 `make lint` (`golangci-lint v1.64.8` + `lint-frontend`) clean; docs-only changes skip lint per `skills/phase-wise-checklist:56` but Phase 45 runs after code phases. Proof: exit 0.

## Dependencies

Depends on Phase 40 ABI header and Phase 42 quickstart snippets; `frontend/scripts/copy-to-docs.mjs` must not overwrite `documentation/python.md` (docs is generated site at `docs/` not `documentation/` per `AGENTS.md:79` trap: `docs/` is built from `frontend/`).

## Evidence

- `documentation/python.md` exists and contains all four sections above
- `grep -n python README.md` hits teaser
- `make claim-scan` exit 0

## Out of scope

Conda-forge / distro docs (follow-up); full `compatibility-matrix.md` extension beyond security subset.

## Handoff

Next is Phase 46 PyPI publish.
