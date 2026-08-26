## Summary

Add in-process Python bindings for the pure-Go engine via an opt-in cgo `c-shared` library and ship them on PyPI. The default Go library and CLI stay `CGO_ENABLED=0` pure-Go; the Python path loads `libgowkhtmltopdf.so/.dylib/.dll` through `ctypes` with a frozen C ABI (v1).

---

## Motivation / context

- Plans: `plans/0.2.5/40-canonical-0.2.5-python-bindings.md` (phases 40-47) and `plans/0.2.5/README.md`
- Phase checklists: `plans/0.2.5/phases/phase-40-c-abi-and-c-shared-build.md` through `phase-47-closure.md`
- Issues: see **Related issues**
- Before this, Python users could only call the CLI via subprocess. This closes #35 by exporting a stable C ABI and a `pip install gowkhtmltopdf` package that calls the same `load -> parse -> cascade -> layout -> paginate -> paint -> pdf write` pipeline in-process.

---

## Changes

### C ABI and c-shared build (`bindings/c`)

- New frozen header `bindings/c/include/gowkhtmltopdf.h` as committed source of truth: `GOWKHTMLTOPDF_ABI_VERSION 1`, `GOWKHTMLTOPDF_VERSION 0.2.4`, 7 status codes (0 OK through 6 INTERNAL), borrowed-string inputs, `malloc`/`free` pairs for `out_data`/`out_err`/`version`, `abi_version`+`struct_size` gate, `NULL opts` means engine defaults.
- New package `bindings/c` (only package allowed to contain `import "C"`): `main.go` stamps `libVersion` via `BINDINGS_VERSION_LDFLAGS`, `exports_cgo.go` facades `Document.WritePDF`/`ImageDocument.WriteImage` with timeout/context and `classifyError` mapping, `exports_stub.go` `!cgo` stub returns `INTERNAL`, `options_pdf.go`/`options_image.go` map `GwkPdfOptions`/`GwkImageOptions` to Go `Document`/`ImageDocument`, `classify.go` maps `context`/`load` errors to status codes.
- `cshared_test.go` (`//go:build cgo`, 9 cases): inline hello PDF shape (`%PDF-`, `%%EOF`, `xref`, `/FontFile2`), fixture-01 invoice, option validation (`A4` vs `Bogus`, `copies 1001`), ABI/size gate, nil HTML, PNG magic, cancelled context timeout, free-pairing loop (500x), version/last_error slot.

### Python package (`bindings/python`)

- New package `bindings/python` (`pyproject.toml` `0.2.4`, `src` layout, `setup.py` shim aligning `VERSION`): zero runtime deps, `requires-python >=3.8`, `cibuildwheel` manylinux_2_28 x86_64/aarch64, vendored `libgowkhtmltopdf.*` via `package-data`.
- `src/gowkhtmltopdf/_lib.py`: search `GOWKHTMLTOPDF_LIBRARY_PATH` then package dir then `dist/`, `CDLL` `RTLD_LOCAL`, pinned `argtypes` for 8 exports, ABI check (`abi_version()==1`), `GwkPdfOptions` 144 bytes / `GwkImageOptions` 104 bytes structs, `free`/`free_string` pairing, thread locks (`_LOAD_LOCK`, `_CALL_LOCK`, GIL released during call).
- `src/gowkhtmltopdf/document.py`: snake_case parity with `document.go` (`PageSize->page_size` etc.), `Content` exact-one validation (`html`|`file`|`url`), `Margin`/`HeaderFooter`/`TOC`/`Crop`/`NetworkPolicy`/`Page`/`PDFOptions` (15 fields)/`ImageOptions` (13 fields)/`Document`/`ImageDocument` with `validate()` mirroring `document_validate.go` (page size `a0-a6`/`b0-b6`..., orientation, pdf version `1.4`/`1.7`/`2.0`, copies `0..1000`), `_serialize` keepalive for `c_char_p`, `effective_base_url()`, v1 note that cover/toc/header/footer beyond geometry/margins/title/pdf_version/copies/grayscale/ACL/network/timeout are accepted for parity but keep engine defaults.
- `src/gowkhtmltopdf/api.py`: `convert_html_to_pdf(html, options, timeout)` and `convert_file_to_pdf(source, out_path)` (sets `base_url` to file parent `file://` URI and `allow_local_files=True` by default so linked CSS like `fixture-56-architecture-diagram.css` resolves), `convert_html_to_image`, `convert_url_to_pdf` raises `NotImplementedError` until handle-based ABI.
- `src/gowkhtmltopdf/exceptions.py`: hierarchy `GowkhtmltopdfError -> ConversionError (code,message) -> InvalidArgumentError/LoadDeniedError/RenderError/ConversionTimeoutError/ResourceLimitError/InternalEngineError` plus sentinel singletons `ErrEmptyContent` etc. with substring sniffing and `error_from_status`.
- `tests/test_model.py` (36 cases, no library) and `tests/test_binding.py` (8 cases, skips gracefully if no `.so`): inline/fixture parity (normalized dates), invalid sentinel, empty doc, image PNG, version/ABI, 25x repeated stability. `examples/invoice.py` covers both `Document` and helper styles.
- `scripts/build_cshared_for_wheel.sh`: host before-build for macOS (`.dylib`) / Windows (`.dll`) / Linux (`.so`).

### Platform and wheels

- `pyproject.toml` `[tool.cibuildwheel]`: `cp38-*` through `cp313-*`, `archs x86_64 aarch64`, `manylinux_2_28`, `before-build` curls `go1.26.0` inside manylinux and builds `libgowkhtmltopdf.so` next to package for `auditwheel repair`.
- `scripts/build_cshared_for_wheel.sh` is the macOS/Windows counterpart used via `CIBW_BEFORE_BUILD` (linux wheels keep using `pyproject.toml` `before-build`).

### CI, build, and versioning

- `Makefile`: new `BINDINGS_VERSION_LDFLAGS` (`-X .../bindings/c.libVersion=$(cat VERSION)`), `c-shared` target guarded `[ "$(CGO_ENABLED)" = "1" ] || exit 2` (pure-Go default stays `CGO_ENABLED=0`), `bindings-clean`, `check-versions`, `python-binding-test`, `clean` now removes `dist/`.
- `.github/workflows/ci.yml`: keeps 4 pure-Go jobs, adds `build` purity guard (`CGO_ENABLED=0 go list -json | jq CgoFiles` allow only `bindings/c`), `build-shared` (`CGO_ENABLED=1 make c-shared` plus ctypes `abi_version==1` and `version==VERSION` asserts), `python-binding` (`pip install -e ./bindings/python` + `python3 -m unittest discover`).
- `.github/workflows/publish-pypi.yml` (new): triggers `push tags v*` + `workflow_dispatch`, matrix `ubuntu-22.04 x86_64+aarch64` (with `setup-qemu`), `macos-13`, `macos-14`, `windows-2019`; linux uses `pyproject.toml` `before-build`, macos/windows use `scripts/build_cshared_for_wheel.sh`; jobs `build-wheels`+`sdist` -> `check` (`check_versions.sh` + `twine check --strict`) -> `publish` (`pypa/gh-action-pypi-publish@release/v1` with `id-token: write` + attestations, `environment: pypi`).
- `scripts/check_versions.sh`: single-source gate `VERSION` (`0.2.4`) vs `pyproject.toml` version, wired as `make check-versions` and publish `check` job.
- `.golangci.yml`: `exclude-dirs` now also `bindings` (cgo glue validated via `go vet` + c-shared build + tests).
- `.gitignore`: ignores `/dist/`, `bindings/c/libgowkhtmltopdf.h`, `bindings/**/*.so/.dylib/.dll`, plus `__pycache__`, and clarifies committed `include/gowkhtmltopdf.h` stays trackable.

### Docs and repo

- New `documentation/python.md` (376 lines): install, quickstart for both `convert_html_to_pdf` helper and `Document(pages=[Page(source=Content(html=...))]).pdf()` parity (snake_case table), file/URL sources, `Document`/`ImageDocument`/`PDFOptions`/`ImageOptions` reference, errors `0-6` with sentinels, timeouts (`timeout_ms` -> `context.WithTimeout`, fixed network caps `30s`/`60s`/`100MiB`/`10` redirects), security (`allow_local_files` vs `allow=[...]`, `EvalSymlinks`, `file://` host check `load.go:1195`, `CompatibleNetworkPolicy`/`RestrictedNetworkPolicy`, GIL release), ABI semver + `abi_version`/`struct_size` gate, self-build (`CGO_ENABLED=1 make c-shared`), platforms.
- `documentation/README.md` adds `python.md` to Guides and Security tables, `documentation/getting-started.md` adds Python callout box, `documentation/deferred.md` flips C ABI row from `Only if consumer demand` to `Shipped in 0.2.5`.
- `README.md` docs index + Python teaser snippet.
- Plans: `plans/0.2.5/40-canonical-0.2.5-python-bindings.md` (57 rows), `plans/0.2.5/README.md`, `plans/README.md` 0.2.5 row, 8 phase checklists `phase-40` through `phase-47` (105 rows), all `[x]` per validation record 2026-08-26.
- Samples: `output/python/fixture-55-lantern-cooperative-report.pdf` (60K) and `output/python/fixture-56-architecture-diagram.pdf` (803K) after fixing `convert_file_to_pdf` linked-CSS base (`file://` parent URI). Fixture-56 was 415K/15p unstyled, now matches Go samples at 20p.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Python call is now in-process `ctypes` with GIL released during render (no CLI spawn/IPC); Go hot path unchanged |
| **Memory** | Borrowed inputs, `C.CBytes`/`C.CString` ownership with documented `gowkhtmltopdf_free`/`free_string` pairs; Python keepalive list prevents GC during call; no change to Go engine memory |
| **Behavior / correctness** | New Python surface only; Go `Document`/`ImageDocument` unchanged. `convert_file_to_pdf` now sets `Content.base` to file parent `file://` URI so sibling CSS resolves (fixes fixture-56 parity: 15p unstyled -> 20p styled, 415K -> 803K) |
| **API / CLI** | Additive only: new `bindings/c` C ABI (frozen v1) and `bindings/python` package `gowkhtmltopdf` (`Document`, `ImageDocument`, `PDFOptions`, `ImageOptions`, `convert_html_to_pdf`/`convert_file_to_pdf`/`convert_html_to_image`, exception hierarchy). No Go API or CLI flag removed |
| **Dependencies** | No new direct Go deps (allowlist still `go-text/typesetting` + `tdewolff/canvas`); Python runtime has zero deps (stdlib `ctypes`) |
| **Binary size / build time** | Pure-Go `make build` unchanged (`22M`/`21M` stamped `0.2.4`); opt-in `CGO_ENABLED=1 make c-shared` produces `dist/libgowkhtmltopdf.so` ~18M stripped, 16 exported `gowkhtmltopdf_*` symbols |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

Pure-Go consumers need no change (`CGO_ENABLED=0` default preserved). Python consumers add `pip install gowkhtmltopdf` and `CGO_ENABLED=1` only when rebuilding the shared library from source.

---

## Test plan

- [x] `make test` (25 packages ok per canonical validation 2026-08-26)
- [x] `make lint` (`golangci-lint v1.64.8` + `lint-frontend` eslint clean)
- [x] `make claim-scan` clean
- [x] `CGO_ENABLED=0 go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf` / `bin/gowkhtmltoimage` stamped `0.2.4`
- [x] `CGO_ENABLED=1 go build -buildmode=c-shared -o dist/libgowkhtmltopdf.so ./bindings/c` (`file` ELF + `nm -D` 16 symbols)
- [x] `CGO_ENABLED=1 go test ./bindings/c -run TestCShared -count=1` (9 cases: shape, fixture, validation, ABI gate, nil, PNG, timeout, free loop 500, version/last_error)
- [x] `python3 -m compileall bindings/python/src`
- [x] Python `36` model + `8` binding tests (`bindings/python/tests/test_model.py`, `test_binding.py`) - inline `%PDF-`/`%%EOF`/`xref`/`/FontFile2`, fixture-01 invoice, parity byte-equal after date normalize, sentinel mapping, PNG magic, version/ABI `1`, 25x repeated stability
- [x] `make golden` (61 fixtures pass, page envelopes)
- [x] `go test -race ./internal/convert ./internal/layout ./internal/pdf ./internal/imageout ./internal/load`
- [x] `bash scripts/check_versions.sh` (0.2.4 aligned)

### Commands

```sh
make test
make lint
make claim-scan
make golden
CGO_ENABLED=1 make c-shared && file dist/libgowkhtmltopdf.so && nm -D dist/libgowkhtmltopdf.so | grep gowkhtmltopdf_
CGO_ENABLED=1 go test ./bindings/c -run TestCShared -count=1 -v
python3 -m unittest discover -s bindings/python/tests -t . -v
bash scripts/check_versions.sh
CGO_ENABLED=0 go build -ldflags "-X github.com/chinmay-sawant/gowkhtmltopdf/internal/cli.Version=$(cat VERSION)" -o /tmp/gowkhtmltopdf ./cmd/gowkhtmltopdf && /tmp/gowkhtmltopdf --version
```

---

## Screenshots / sample output

```
make test              # 25 packages ok
make lint              # golangci-lint v1.64.8 clean + eslint clean
make claim-scan        # clean
make golden            # 61 fixtures PASS
CGO_ENABLED=1 make c-shared  # dist/libgowkhtmltopdf.so ELF 64-bit 18M stripped, nm 16 gowkhtmltopdf_* symbols
CGO_ENABLED=1 go test ./bindings/c -run TestCShared -v  # 9 cases PASS
python3 -m unittest discover -s bindings/python/tests -v  # 44 OK (36 model + 8 binding)
python3 bindings/python/examples/invoice.py  # writes invoice.pdf + invoice_high_level.pdf, both %PDF-
output/python/fixture-55-lantern-cooperative-report.pdf  # 60K 3p, byte-identical after date/ID normalize
output/python/fixture-56-architecture-diagram.pdf  # 803K 20p, now styled (was 415K 15p before base_url fix)
```

---

## Related issues

- Closes #35

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-python-cgo-pypi.md` when process-gated

---

## Follow-ups (out of scope)

- Conda-forge / distro packages (tracked as follow-up in #35)
- PHP / Ruby / Node reuse of the same C ABI (separate tickets)
- Handle-based ABI for cover/TOC/header/footer and `convert_url_to_pdf` URL sources (v1 one-shot keeps engine defaults for those fields)
- Full `wkhtmltopdf` flag parity (high-traffic subset only in v1)
- Pixel-golden harness inside the Python package (Go suite remains source of truth)

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
