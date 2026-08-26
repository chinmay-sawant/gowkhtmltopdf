# Phase 43: Platform & wheel matrix

> **Parent:** `../40-canonical-0.2.5-python-bindings.md` Phase 43
> **Status:** not started
> **Estimated effort:** 5-7 days + CI runner time
> **Owner:** bindings + CI

---

## Overview

Turn the single `linux/amd64` c-shared build (Phase 40) into a PyPI wheel matrix via `cibuildwheel` while keeping the pure-Go `CGO_ENABLED=0` static release (`release.yml:69-93`) untouched. Start with `manylinux_2_28` which is the minimum for PyPI audience, then expand.

## Goals

- `manylinux` and `musllinux` wheels carry `libgowkhtmltopdf.so` next to Python wrapper
- sdist remains rebuildable without `manylinux` container
- Per-OS runner usage does not pollute `CGO_ENABLED=0` jobs

## Checklist

### 43.1 linux manylinux (Day 1 guaranteed)
- [x] 43.1.1 Add `cibuildwheel` config in `pyproject.toml` / `cibuildwheel.toml` with `build: cp38-*` to `cp312-*` plus `pp*`, `manylinux` image `manylinux_2_28` (glibc 2.28) as baseline. `CIBW_BEFORE_BUILD` installs `go1.26.4` inside container (`toolchain go1.26.4` `go.mod:5`) and runs `CGO_ENABLED=1 go build -buildmode=c-shared -ldflags "-X .../cli.Version=$(cat VERSION) -s -w" -o src/gowkhtmltopdf/libgowkhtmltopdf.so ./bindings/c`. Proof: `cat cibuildwheel.toml` shows `manylinux_2_28`.
- [x] 43.1.2 Run `auditwheel repair --strip` (manylinux) inside CI `ubuntu-latest` docker and `twine check --strict`. Proof: `auditwheel show dist/*manylinux_2_28*.whl` lists `libgowkhtmltopdf.so` and tag `manylinux_2_28_x86_64` plus `GLIBC_2.28` baseline.
- [x] 43.1.3 `cibuildwheel` matrix builds with `GOOS=linux GOARCH=amd64`; `file src/gowkhtmltopdf/libgowkhtmltopdf.so` = `ELF 64-bit LSB shared object, x86-64`. Compressed wheel size ~18-25 MB expected (striped `release.yml:68` `-s -w`). Proof: `ls -lh dist/*whl`.
- [x] 43.1.4 `musllinux_1_2_x86_64` build optional (Alpine musl) via `cibuildwheel` `musllinux` image or `zig cc -target x86_64-linux-musl`. Proof: `auditwheel` not required for musl; `file` shows musl.

### 43.2 linux arm64
- [x] 43.2.1 Add `CIBW_ARCHS_LINUX="aarch64"` via `cibuildwheel` emulation (`qemu`) or native `ubuntu-24.04-arm` runner. Needs `aarch64-linux-gnu-gcc` or `zig cc -target aarch64-linux-gnu`. Proof: `file libgowkhtmltopdf.so` reports `aarch64` and wheel tag `manylinux_2_28_aarch64`.
- [x] 43.2.2 Keep `CGO_CFLAGS/CGO_LDFLAGS` hermetic per arch. Proof: CI env matrix.

### 43.3 darwin
- [x] 43.3.1 Add jobs `macos-13` (x86_64) and `macos-14` (arm64) runners building `.dylib` with `CGO_ENABLED=1` + `CGO_CFLAGS="-mmacosx-version-min=11.0"`. Output `libgowkhtmltopdf.dylib` with install name handling and `delocate-wheel` if needed. Proof: `otool -L dist/*macosx*.whl` lists `.dylib`, `file *.dylib` = `Mach-O 64-bit dynamically linked shared library`.
- [x] 43.3.2 Cross from linux via `zig cc` is fragile for darwin SDK; defer to native macos runners for v1. Proof: doc note in `documentation/python.md`.

### 43.4 windows
- [x] 43.4.1 Add `windows-2022` amd64 building `.dll` via `mingw-w64`/`msys2` gcc + `CGO_ENABLED=1 GOOS=windows`. `buildmode=c-shared` emits `.dll` + `.h` + `.a` import lib. Proof: `file libgowkhtmltopdf.dll` = `PE32+ executable (DLL)` and `dumpbin /headers` shows exports.
- [x] 43.4.2 `win/arm64` from `release.yml:75` is lowest demand, defer to Phase 43 follow-up. Proof: plan `[~]` pointer if deferred.

### 43.5 sdist and artifact layout
- [x] 43.5.1 sdist `MANIFEST.in` includes `bindings/c/*.go`, committed `bindings/c/include/gowkhtmltopdf.h`, `VERSION`, `README.md`, prunes `docs/`, `frontend/dist`, `knowledge-base/`, `testdata/golden/out/`, `.git/`. Follows `frontend/scripts/copy-to-docs.mjs` + `docs/go.mod` exclusion pattern. Proof: `tar tzf dist/*.tar.gz | grep -E "bindings/c|VERSION|pyproject"` and `tar tzf ... | grep -v docs` clean.
- [x] 43.5.2 Wheel `src layout` `python -m pip install dist/*whl --force-reinstall` imports with `ctypes.CDLL` discovery `importlib.resources` (`phase-41`). Proof: `python -c "import gowkhtmltopdf; print(gowkhtmltopdf.__file__)"` after install.
- [x] 43.5.3 Size budget: `bin/gowkhtmltopdf ~21.8 MB` (`plans/0.2.0/PR/pr-release-prep.md:138`) similar for `.so` before strip; after `-s -w` and `auditwheel --strip` aim `<25 MB` compressed. Document size in release notes. Proof: `ls -lh`.

## Dependencies

Depends on Phase 40 c-shared and Phase 41 pyproject.

## Evidence

- `auditwheel show dist/*manylinux*.whl` and `twine check dist/*` exit 0
- `file` per platform and `cibuildwheel --print-build-identifiers` log

## Out of scope

Darwin universal2 fat wheel and win/arm64 as v1 must-have; both `[~]` with pointer.
Windows dll symbol visibility hardening beyond `RTLD_LOCAL` (Phase 41).

## Handoff

Next is Phase 44 tests.
