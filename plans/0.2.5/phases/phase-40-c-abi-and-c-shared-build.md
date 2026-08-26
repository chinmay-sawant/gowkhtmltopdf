# Phase 40: C ABI & c-shared build (purity guard)

> **Parent:** `../40-canonical-0.2.5-python-bindings.md` Phase 40
> **Status:** not started
> **Estimated effort:** 5-7 days
> **Owner:** bindings/c

---

## Overview

Freeze a minimal versioned C ABI and prove the isolated `c-shared` build without breaking the `CGO_ENABLED=0` default for Go consumers. This is the foundation for all Python work and the only place `CGO_ENABLED=1` is allowed.

## Goals

- One-shot `html -> pdf` and `html -> image` exports with clear memory ownership and versioned header.
- No `import "C"` leaks outside `bindings/c`; `go test ./...` stays pure-Go.

## Deliverables

- `bindings/c/include/gowkhtmltopdf.h` (committed or generated with drift check)
- `bindings/c/exports_cgo.go` (`//go:build cgo` + `//export`) and `exports_stub.go` (`//go:build !cgo`)
- Makefile `c-shared` isolated target and CI `build-shared` job

## Checklist

### 40.1 ABI contract freeze (`bindings/c/include/gowkhtmltopdf.h`)
- [x] 40.1.1 Define `GOWKHTMLTOPDF_ABI_VERSION 1` and `GOWKHTMLTOPDF_VERSION "0.2.5"` plus `GOWKHTMLTOPDF_LIBRARY_VERSION "0.12.7-dev"` macros matching `VERSION:1` and `api.go:23`. Proof: `grep -n ABI_VERSION bindings/c/include/gowkhtmltopdf.h`.
- [x] 40.1.2 Define `GwkPdfOptions` struct with leading `abi_version`, `struct_size`, then `page_size`, `width_mm`, `height_mm`, `orientation`, `margin_* mm`, `title`, `pdf_version`, `pdf_profile`, `copies`, `grayscale`, `enable_local_file_access`, `allow`+`allow_len`, `network_policy` (+ schemes/hosts/private/cross-host), `base_url`, `timeout_ms`, `_reserved[8]`. Proof: header diff in PR.
- [x] 40.1.3 Define `GwkImageOptions` mirroring `ImageGlobal` (`settings.go:562` defaults: width 1024, quality 94, crop -1). Proof: header.
- [x] 40.1.4 Define exports: `int gowkhtmltopdf_html_to_pdf(const char *html, size_t html_len, const GwkPdfOptions*, unsigned char **out_pdf, size_t *out_len)`, `gowkhtmltopdf_html_to_image`, `void gowkhtmltopdf_free(void*)`, `void gowkhtmltopdf_free_string(char*)`, `const char* gowkhtmltopdf_version(void)`, `int gowkhtmltopdf_abi_version(void)`, `int gowkhtmltopdf_last_error_length(void)`, `int gowkhtmltopdf_last_error(char*,size_t)`. Proof: `nm -D dist/libgowkhtmltopdf.so | grep gowkhtmltopdf_`.
- [x] 40.1.5 Document status codes `0 OK, 1 INVALID_ARG, 2 LOAD_DENIED, 3 RENDER_ERROR, 4 TIMEOUT, 5 RESOURCE_LIMIT, 6 INTERNAL` and `*out_pdf==NULL` on failure plus free pairing `C.malloc`/`gowkhtmltopdf_free`. Proof: header comment.

### 40.2 Go //export facade (`bindings/c/`)
- [x] 40.2.1 Add `bindings/c/exports_cgo.go` with `//go:build cgo`, `import "C"` and `//export` funcs that copy HTML via `C.GoBytes` + `cloneBytes` (`api.go:193`) then call `Document{Pages:[Page{Source:Content{HTML:..}}]}.WritePDF` via `convert.Run` (`internal/convert/convert.go:350`) or `imageout.RunRequest` (`internal/imageout/imageout.go:1514`). Proof: `CGO_ENABLED=1 go vet ./bindings/c/...` exits 0.
- [x] 40.2.2 Add `bindings/c/exports_stub.go` with `//go:build !cgo` returning `ENOSYS` so package non-empty under `CGO_ENABLED=0`. Proof: `CGO_ENABLED=0 go test ./bindings/c -run ^$ -v` reports skip no build error.
- [x] 40.2.3 Map `GwkPdfOptions` -> `settings.DefaultPdfGlobal()` selective overrides (`document.go:337`) for `page_size` via `settings.ParsePageSize` (`pagesize.go:51`), `orientation` via `ParseOrientation` (`settings.go:138`), `pdf_version`/`pdf_profile` via `ParsePDFVersion/Profile` (`settings.go:56`), `copies` 0->1 and `1..1000` (`api.go:46`, `convert.go:44`). Invalid -> `INVALID_ARG`. Proof: unit test covers invalid `page_size` and `copies=0`.
- [x] 40.2.4 Map `allow[]` + `enable_local_file_access` -> `AccessController.Allowed` with `EvalSymlinks` (`load.go:248`) and `BlockLocalFileAccess` (`settings.go:556`). Map `network_policy` presets `CompatibleNetworkPolicy` / `RestrictedNetworkPolicy` (`load.go:123-138`) via `ApplyNetworkPolicy` (`load.go:149`). Map `timeout_ms` -> `context.WithTimeout` + `LoadPage.Timeout` seconds (`load.go:1380`). Proof: deny fixture without `allow` returns `LOAD_DENIED`.

### 40.3 Makefile isolation
- [x] 40.3.1 Add `c-shared:` target gated on `$(CGO_ENABLED)==1` doing `CGO_ENABLED=1 go build -buildmode=c-shared -ldflags "$(CLI_VERSION_LDFLAGS) -s -w" -o dist/libgowkhtmltopdf.so ./bindings/c` plus `file dist/*`. Must not be in `make build` / `make test`. Proof: `CGO_ENABLED=0 make build` still `CGO_ENABLED=0`, `make c-shared` without env prints guidance and exits 2.
- [x] 40.3.2 Extend `clean` or add `bindings-clean` to `rm -rf dist/` (`Makefile:244`). Proof: `make clean && ls dist` after `c-shared`.

### 40.4 Gitignore and purity guard
- [x] 40.4.1 Update `.gitignore` for `/dist/`, `bindings/c/*.so`, `*.a`, `*.dll`, `*.dylib`, `*.h` (except committed vendored header if any). Proof: `git check-ignore dist/libgowkhtmltopdf.so`.
- [x] 40.4.2 Assert `CGO_ENABLED=0 go list -json ./... | jq '.CgoFiles'` empty for `internal/*` and root. Proof: snapshot in closure note.
- [x] 40.4.3 Allow `TestDirectModuleAllowlist` to stay green: zero new direct deps (`go list -m -f {{if and (not .Main)(not .Indirect)}}` only `go-text/typesetting`, `tdewolff/canvas` per `shape_test.go:202`). Proof: `go test ./internal/pdf -run TestDirectModuleAllowlist -v`.

### 40.5 CI c-shared linux amd64 build
- [x] 40.5.1 Add `build-shared` job `runs-on: ubuntu-latest` with `CGO_ENABLED=1` that builds `dist/libgowkhtmltopdf.so`, runs `nm -D | grep`, `file`, and version ldflags assert `gowkhtmltopdf_version()==VERSION`. Proof: GH Actions log.
- [x] 40.5.2 If header vendored, add drift check `diff <(generated) <(committed)` like `Makefile:78 GOLDEN_APPROVE` guard. Proof: CI step.

## Dependencies

Depends on `document.go:99` Document contract and `VERSION`.

## Evidence

- `CGO_ENABLED=0 make test` green
- `CGO_ENABLED=1 make c-shared && nm -D dist/libgowkhtmltopdf.so`
- Header committed

## Out of scope

Multi-page/cover/TOC/header-footer C exports beyond one-shot; full Python wrapper (Phase 41).

## Handoff

Next is Phase 41 Python package and loader.
