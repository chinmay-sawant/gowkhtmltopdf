# Phase 41: Python package & loader (ctypes)

> **Parent:** `../40-canonical-0.2.5-python-bindings.md` Phase 41
> **Status:** not started
> **Estimated effort:** 4-6 days
> **Owner:** bindings/python

---

## Overview

Scaffold the installable Python package that loads the c-shared library in-process via stdlib `ctypes` and owns the buffer-free contract. No cffi or compiled extension in v1.

## Goals

- `pip install` works from sdist and wheel, version tracks `VERSION`
- Loader finds `libgowkhtmltopdf.so/.dylib/.dll` via `importlib.resources` with `RTLD_LOCAL`
- Memory ownership is explicit and tested

## Checklist

### 41.1 Scaffold (`bindings/python/`)
- [x] 41.1.1 Create `bindings/python/pyproject.toml` with `[project] name="gowkhtmltopdf"` (fallback `gowkhtmltopdf-python` if `https://pypi.org/pypi/gowkhtmltopdf/json` not 404), `dynamic = ["version"]` reading `VERSION`, `readme = "README.md"`, `requires-python >=3.8`, `packages.find where = ["src"]`, `python src layout` `src/gowkhtmltopdf/`. Proof: `python -m build --sdist --wheel` exits 0 in `bindings/python`.
- [x] 41.1.2 Create `src/gowkhtmltopdf/__init__.py`, `_lib.py`, `py.typed`, `__version__.py` generated at build from `VERSION`. Proof: `pip install -e bindings/python && python -c "import gowkhtmltopdf; print(gowkhtmltopdf.__version__)"`.
- [x] 41.1.3 `python -m build` sdist includes `bindings/c/*.go` + vendored header for rebuild without wheel. Proof: `tar tzf dist/*.tar.gz | grep bindings/c`.
- [x] 41.1.4 `twine check --strict dist/*` and `check-wheel-contents dist/*` clean. Proof: command exit 0.

### 41.2 Loader (`src/gowkhtmltopdf/_lib.py`)
- [x] 41.2.1 Implement `_load()` trying `importlib.resources.files("gowkhtmltopdf")/"libgowkhtmltopdf.so"` then `pathlib.Path(__file__).parent/"libgowkhtmltopdf.so"` then system fallback. Platform switch `.so` (linux), `.dylib` (darwin `sys.platform=="darwin"`), `.dll` (win32 `ctypes.WinDLL`). Proof: `python -c "import gowkhtmltopdf._lib; print(gowkhtmltopdf._lib.lib)"` on ubuntu prints `<CDLL '...libgowkhtmltopdf.so'>`.
- [x] 41.2.2 Set `argtypes`/`restype` for every export (`gowkhtmltopdf_html_to_pdf`, `gowkhtmltopdf_free`, `gowkhtmltopdf_version`, etc) and `use_errno=False`. Proof: `ctypes.get_errno` not needed.
- [x] 41.2.3 Loader verifies `gowkhtmltopdf_abi_version() == GOWKHTMLTOPDF_ABI_VERSION` else raise `ImportError` with version mismatch. Proof: header macro `GOWKHTMLTOPDF_ABI_VERSION` matches runtime.
- [x] 41.2.4 Loader uses `RTLD_LOCAL` (`ctypes.DEFAULT_MODE` on linux) to avoid Go runtime symbol leakage (`shape_test.go:217 harfbuzz guard` pattern). Proof: `dlopen` flags doc in `_lib.py` comment.

### 41.3 Memory ownership (`src/gowkhtmltopdf/_lib.py` + header)
- [x] 41.3.1 Go allocates `out_pdf` via `C.CBytes` / `C.malloc` (copy of `document.go:250` `append(nil, output.Bytes()...)`); Python copies via `ctypes.string_at(ptr, len)` into `bytes` then calls `gowkhtmltopdf_free(ptr)`. Proof: header `who frees` comment and loop test 1000x not leaking (`phase-44`).
- [x] 41.3.2 Borrowed inputs: `html` ptr/len and `options` strings/arrays borrowed for call only; Go copies via `C.GoString` + `cloneStrings` (`load.go:1306`). Caller may free immediately after call. Proof: doc note.
- [x] 41.3.3 Error strings: callee `C.CString(err.Error())` stored per-handle or TLS, freed via `gowkhtmltopdf_free_string`; caller must not `free` with system `free`. Proof: header free pairing table.

### 41.4 Thread/GIL notes
- [x] 41.4.1 Document `Document` not thread-safe on same handle (`documentation/library-api.md:302`), distinct handles concurrent okay (`convert.go:284 runContext`). Long renders hold `threading.Lock` or release GIL note. Proof: `documentation/python.md` note.

## Dependencies

Depends on Phase 40 `.so` and header.

## Evidence

- `pip install -e bindings/python && python -c "import gowkhtmltopdf; assert gowkhtmltopdf.__version__ == open('VERSION').read().strip()"`
- `auditwheel show` for wheel from Phase 43

## Out of scope

Full `Document` dataclasses (Phase 42); wheel matrix (Phase 43); docs (Phase 45).

## Handoff

Next is Phase 42 Document parity and snippet.
