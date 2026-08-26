# Phase 44: Tests (C ABI smoke + Python integration + leak)

> **Parent:** `../40-canonical-0.2.5-python-bindings.md` Phase 44
> **Status:** not started
> **Estimated effort:** 3-5 days
> **Owner:** bindings/c + bindings/python

---

## Overview

Prove the C ABI and Python binding do not silently break the Go engine path (`internal/convert` `Run` / `imageout.RunRequest`) and that buffer ownership does not leak. All tests are isolated so `make test` `CGO_ENABLED=0` remains green (`ci.yml:24`).

## Goals

- C smoke asserts `%PDF-`, `%%EOF`, `xref`, page-count via `pdf.ParseSemantic`
- Python integration asserts fixture HTML -> non-empty PDF with font embedding
- Leak smoke proves `free` pairing

## Checklist

### 44.1 C ABI smoke (`bindings/c/cshared_test.go`)
- [ ] 44.1.1 New file `bindings/c/cshared_test.go` with `//go:build cgo` that calls `gowkhtmltopdf_html_to_pdf` on `<!DOCTYPE html><h1>Hello</h1>` and on `testdata/golden/fixture-01-simple-invoice.html` (`golden_test.go:243` min 1 max 1, 52 lines, deterministic, no external `logo.png` vs `fixture-07`). Asserts `out_len>0`, `bytes.HasPrefix(C.GoBytes(ptr, len), []byte("%PDF-"))` (`golden_test.go:175`), `bytes.Contains(b, []byte("%%EOF"))` (`golden_test.go:179`), `xref` at correct offset (`golden_test.go:191-204`), `pageCount(b)>=1` (`golden_test.go:82`), and `pdf.ParseSemantic(b).PageCount>=1` (`internal/pdf/semantic.go:77`) plus `/FontFile2` present (`golden_test.go:221`). Proof: `CGO_ENABLED=1 go test -tags cgo -run TestCShared -count=1 -v ./bindings/c` green.
- [ ] 44.1.2 Failure path: null html / `html_len=0` with empty fixture returns non-zero `INVALID_ARG` and `*out_pdf==nil` / `*out_len==0` and no `C.malloc` leak (`C.GoString` not called). Also `copies=0` -> default 1, `copies=1001` -> `INVALID_ARG` (`convert.go:44`). Proof: same test binary, negative subtests `t.Run("invalid_arg")`.
- [ ] 44.1.3 `fixture-07-image-logo.html` (`golden_test.go:262` `images:true`) asserts `/Subtype /Image` present; `fixture-06-external-link.html` (`golden_test.go:258` `uris:true`) asserts `/S /URI`. Proof: semantic text needles (`golden_test.go:221-232`).

### 44.2 Python integration (`bindings/python/tests/test_binding.py`)
- [ ] 44.2.1 `pytest` `test_convert_inline_header`: `open("testdata/golden/fixture-01-simple-invoice.html").read()` -> `gowkhtmltopdf.convert_html_to_pdf(html, options=PDFOptions(page_size="A4"))` -> `assert pdf.startswith(b"%PDF-")`, `len(pdf)>1024`, `b"%%EOF" in pdf`, `pdf.count(b"/FontFile2")>=1` (`golden_test.go:221`), `b"Invoice" in pdf` semantic. Also `convert_html_to_pdf(b"<!DOCTYPE html><h1>Hello</h1>", options=PDFOptions(page_size="A4"))` fast variant without file I/O (`convert_test.go:88`). Proof: `pytest bindings/python/tests -v -k test_convert_header` green.
- [ ] 44.2.2 `test_document_parity`: `Document(pages=[Page(source=Content(html=b"<h1>Invoice</h1>"))], page_size="A4").pdf()` equals `convert_html_to_pdf(b"<h1>Invoice</h1>", options=PDFOptions(page_size="A4"))` for `%PDF-` prefix. Proof: same pytest.
- [ ] 44.2.3 `test_image`: `convert_html_to_image(b"<h1>Badge</h1>", options=ImageOptions(width=1024, format="png"))` returns `b"\x89PNG"` and `len==height*width*4` budget (`imageout.go:45`). Proof: `pytest -k test_image`.
- [ ] 44.2.4 No network required: all tests offline, no live `https://en.wikipedia.org/wiki/Ana_de_Armas` smoke (`Makefile:148`).

### 44.3 Leak/free smoke
- [ ] 44.3.1 Loop 1000x `ptr,len = gowkhtmltopdf_html_to_pdf(html, opts, &out_pdf, &out_len); assert ptr!=nil && len>0; gowkhtmltopdf_free(ptr); gowkhtmltopdf_free(nil)` no crash, no double-free, `C.CString`/`C.free` paired same allocator (`C.malloc` vs Go `C.CBytes`). Proof: `CGO_ENABLED=1 go test -run TestCSharedFree -count=1 ./bindings/c -v` green plus optional `valgrind --leak-check=full ./c_smoke` in `python-binding` CI job (`t.TempDir` isolation like `convert_test.go:46`).
- [ ] 44.3.2 Error string free: `gowkhtmltopdf_last_error(buf,len)` or `char **out_error` variant freed via `gowkhtmltopdf_free_string`; TLS buffer overwritten note tested by two consecutive failures. Proof: C test asserts second `last_error` overwrites first.

### 44.4 CI isolation
- [ ] 44.4.1 Keep existing `test+lint` job `CGO_ENABLED=0` (`ci.yml:24`) and `static build (CGO_ENABLED=0)` (`ci.yml:37`) unchanged. Add new job `python-binding` with `env: CGO_ENABLED: 1` only in that job, `runs-on: ubuntu-latest`, steps `setup-go 1.26`, `setup-python 3.11`, `pip install -e bindings/python`, then `go test -tags cgo -run TestCShared ./bindings/c` + `pytest bindings/python/tests -v`. Do not set `CGO_ENABLED=1` globally. Proof: GH log shows two jobs, `make test` still `CGO_ENABLED=0`.
- [ ] 44.4.2 `race` job stays `go test -race -count=1 ./internal/convert ./internal/layout ./internal/pdf ./internal/imageout ./internal/load` (`ci.yml:77`); optional extra `go test -race -tags cgo -count=1 ./bindings/c` under `cgo: 1` job only. Proof: CI YAML diff.

### 44.5 Fixture reuse

- [ ] 44.5.1 All smokes reuse `fixture-01-simple-invoice.html` as fastest deterministic single-page 52-line fixture. Proof: `testdata/golden/README.md:53` and `golden_test.go:243`.

## Dependencies

Depends on Phase 40 `.so` and Phase 41 loader, Phase 42 dataclasses, Phase 43 wheel for optional `auditwheel` gate but smokes run unpacked.

## Evidence

- `CGO_ENABLED=1 go test -tags cgo ./bindings/c -run TestCShared -v` green
- `pytest bindings/python/tests -v` green
- CI shows `CGO_ENABLED=0` path still green plus `CGO_ENABLED=1` job green

## Out of scope

Pixel golden harness inside Python package (`testdata/golden/README.md:139` Guard: `GOLDEN_APPROVE=1` only writes `testdata/golden/out/`); full 61-fixture Python golden (Go suite remains source of truth).

## Handoff

Next is Phase 45 docs and security.
