# Review - SOLID and Go reuse audit

> **Parent:** `skills/solid-go-review/SKILLS.md` - review workflow
> **Status:** Complete. All findings were implemented or closed by current-source proof.
> **Estimated effort:** Not estimated until findings are accepted for implementation

---

## Overview

This review examined the current Go source and tests through four disjoint packets:

1. Public API, CLI, application adapters, settings, examples, and C bindings.
2. Conversion orchestration and its preparation, render, and benchmark-island packages.
3. Layout and line packages.
4. PDF, image output, SVG, outline, and PDF profile packages.

The review skipped historical documentation and used current source, tests, package discovery, and validation commands only.

## Executive Summary

The codebase has clear pipeline stages, useful function and interface seams, explicit ownership comments, bounded raster work, and strong current test coverage. The main risks sit at boundaries where two paths implement the same policy differently or where output and cancellation contracts stop one step too early.

Overall score: **7.2/10**, medium-high confidence.

Weighted arithmetic:

```text
SRP 6.5 x 0.10 = 0.650
OCP 7.0 x 0.10 = 0.700
LSP 8.5 x 0.10 = 0.850
ISP 8.0 x 0.10 = 0.800
DIP 7.0 x 0.10 = 0.700
Package cohesion and dependency direction 7.0 x 0.15 = 1.050
API ergonomics and reusable seams 6.5 x 0.10 = 0.650
Errors, context, ownership, and resource bounds 6.0 x 0.10 = 0.600
Tests, benchmarks, and validation 8.5 x 0.10 = 0.850
Dependency simplicity and language fit 7.5 x 0.05 = 0.375
Total = 7.225, reported as 7.2/10
```

## SOLID verdicts

| Principle | Verdict | Evidence |
| --- | --- | --- |
| SRP | MIXED | `render.Pipeline` isolates lifecycle ordering (`internal/convert/render/pipeline.go:11-18`), but `internal/convert/hf.go` combines loading, layout, painting, margins, and dispatch (`internal/convert/hf.go:23-877`), and `imageout.go` combines raster stages and encoding (`internal/imageout/imageout.go:1631-1721`). |
| OCP | MIXED | The render lifecycle accepts adapters through a narrow interface (`internal/convert/render/pipeline.go:11-18`), but island and generic body paths apply different policies (`internal/convert/convert.go:564-615`). |
| LSP | PASS where applicable | The review found real substitution contracts for the render pipeline, IP resolver, and outline location reader. No confirmed precondition or postcondition violation was found (`internal/convert/render/pipeline.go:14-18`, `internal/load/load.go:376-378`, `internal/outline/outline.go:18-29`). |
| ISP | PASS | Interfaces are small and behavior-focused. The render pipeline has three lifecycle methods, while the loader and outline seams expose only the behavior their consumers need (`internal/convert/render/pipeline.go:14-18`, `internal/load/load.go:376-378`, `internal/outline/outline.go:18-29`). |
| DIP | MIXED | High-level conversion uses a lifecycle seam, but image output constructs a concrete loader and default font inside orchestration (`internal/imageout/imageout.go:1600-1622`). This is acceptable at an adapter boundary, but it limits replacement and focused testing. |

## Strong design choices

- Public validation runs before rendering and the adapters validate before opening output files (`document_validate.go:32-188`, `internal/app/pdf.go:73-100`).
- The conversion lifecycle is represented by a small interface with explicit cancellation checks between stages (`internal/convert/render/pipeline.go:11-57`).
- The loader exposes a narrow resource context for URL resolution, ACL checks, limits, and errors (`internal/load/load.go:102-110`).
- Layout workspace reuse documents sequential ownership and release lifetime (`internal/layout/layout.go:227-257`).
- Raster dimensions, decoded image data, caches, and pooled buffers have explicit limits (`internal/imageout/imageout.go:424-485`, `internal/imageout/imageout.go:513-523`).
- The PDF writer documents single-goroutine ownership and uses typed object references (`internal/pdf/pdf.go:122-126`, `internal/pdf/pdf.go:81`).
- The outline package uses a neutral location reader instead of depending on layout implementation types (`internal/outline/outline.go:18-29`).
- All current repository gates passed: `make test`, `make test-race`, `make lint`, `make claim-scan`, and `make golden`.

## Highest-value confirmed findings

1. `C-1` cloned layout results retain shared PDF structure pointers, which can cross document ownership boundaries (`internal/layout/layout.go:167-176`, `internal/layout/layout.go:404-405`, `internal/layout/tagging.go:26-54`).
2. `A-01` zero-initialized C image options turn smart width off and create a zero crop instead of preserving engine defaults (`bindings/c/options_image.go:8-16`, `bindings/c/options_image.go:56-69`, `bindings/c/exports_cgo.go:330-370`).
3. `PB-003` copy validation differs between request, page-plan, materialization, and non-collated ordering seams (`internal/convert/convert.go:155-160`, `internal/convert/page_plan.go:43-49`, `internal/convert/page_plan.go:182-188`, `internal/convert/render/plan.go:137-150`).
4. `IMG-OUT-01` image output ignores silent short writes and can report success for truncated output (`internal/imageout/imageout.go:1712-1721`).
5. `PDF-OUT-01` an empty uncompressed page receives a `/Length 0` dictionary without stream markers (`internal/pdf/pdf.go:553-575`, `internal/pdf/pdf.go:1020-1029`).

## Phase-wise findings ledger

### Phase 1: Correctness, ownership, and coupling

- [x] `C-1` `internal/layout/layout.go:167-176` - cloned results clear document-owned `StructElem` pointers before a second document paints them. Proof: `TestCloneResultDropsDocumentOwnedStructureElements` passes.
- [x] `A-01` `bindings/c/options_image.go:56-69` - zero-initialized C image options preserve engine defaults through unset markers. Proof: C adapter regression tests pass.
- [x] `PB-003` `internal/convert/page_plan.go:43-49` - request, page-plan, and ordering seams share one bounded copy validation contract. Proof: copy seam tests pass.
- [x] `IMG-OUT-01` `internal/imageout/imageout.go:1717-1721` - silent short writes now return `io.ErrShortWrite` for PNG and JPEG. Proof: output regression tests pass.
- [x] `PDF-OUT-01` `internal/pdf/pdf.go:564-575` - explicitly present empty page streams serialize as valid empty streams. Proof: strict parser test passes.

### Phase 2: API and reuse

- [x] `A-02` `internal/cli/cli.go:324-334` - terminal XSL mode rejects conflicting output arguments. Proof: parser and command tests pass.
- [x] `A-03` `internal/app/pdf.go:26-34` - app and lower-level nil-command errors share one canonical sentinel. Proof: `errors.Is` tests pass.
- [x] `A-04` `bindings/c/classify.go:43-63` - image quality and crop input errors map to invalid arguments. Proof: C classification tests pass.
- [x] `A-05` `document_validate.go:109-112` - trimmed orientation values drive both validation and mapping. Proof: end-to-end orientation test passes.
- [x] `PB-004` `internal/convert/prepare/styles.go:33-55` - stylesheet and font helpers delegate through one resource-policy owner. Proof: preparation parity tests pass.
- [x] `C-2` `internal/layout/layout.go:85-101` - image loading has a context-bearing seam that cancels a blocked resolver. Proof: blocking callback test passes.

### Phase 3: Performance, resource safety, and cleanup

- [x] `PB-001` `internal/convert/render/pipeline.go:37-55` - assembly and copy work now check cancellation before later work or output. Proof: cancellation tests pass.
- [x] `PB-002` `internal/convert/convert.go:564-615` - certified islands use the shared smart-shrink and body-policy path, with generic and island link behavior tested together. Proof: differential island tests pass.
- [x] `PB-005` `internal/convert/hf.go:1` - the file-wide lint suppression is gone and `make lint` passes with only narrow suppressions.
- [x] `C-3` `internal/layout/style.go:105-381` - benchmark evidence supports the existing pointer-access seam; a stage-view split was not justified because value copies allocate zero bytes. Proof: `BenchmarkResolvedStyleAccess` and layout race tests pass.
- [x] `C-4` `internal/layout/style.go:701-767` - dead style interning structures were removed. Proof: layout tests and lint pass.
- [x] `PDF-FONT-RES-01` `internal/pdf/registry.go:368-381` - font scans and direct parsing enforce the shared 32 MiB limit. Proof: oversized-input tests pass.
- [x] `IMG-CTX-01` `internal/imageout/imageout.go:1684-1721` - image finalization checks cancellation before encoding and writing. Proof: cancellation tests pass.
- [x] `PDF-OUT-02` `internal/pdf/pdf.go:669-707` - `WriteTo` reports bytes accepted by the external sink after flush. Proof: flush-failure test passes.
- [x] `SVG-RES-01` `internal/svg/raster.go:50-59` - SVG input and detection probes are bounded. Proof: oversized-input tests pass.
- [x] `IMG-ARCH-01` `internal/imageout/imageout.go:324-390` - the package-wide lint suppression was removed and the package passes independent tests and lint.

## Packet scores

| Packet | Scope | Score |
| --- | --- | ---: |
| A | Public API, adapters, settings, CLI, C bindings | 7.0/10 |
| B | Conversion and preparation | 7.0/10 |
| C | Layout and line | 7.0/10 |
| D | PDF, image output, SVG, outline, profiles | 7.5/10 |

## Validation record

- `make test`: passed.
- `make test-race`: passed for `internal/convert`, `internal/layout`, `internal/pdf`, `internal/imageout`, and `internal/load`.
- `make lint`: passed, including Go and frontend lint.
- `make claim-scan`: passed.
- `make golden`: passed all current fixtures.
- CLI probe: `go run ./cmd/gowkhtmltopdf --dump-default-toc-xsl --output /dev/null` exited 0 and emitted 455 bytes to stdout.

## Recommended order

1. Fix output and ownership contracts: `C-1`, `IMG-OUT-01`, `PDF-OUT-01`, and `A-01`.
2. Unify validation and policy seams: `PB-003`, `A-02`, `A-03`, `A-04`, `A-05`, and `PB-004`.
3. Thread cancellation through assembly and image finalization: `PB-001`, `C-2`, and `IMG-CTX-01`.
4. Add resource bounds for font and SVG input: `PDF-FONT-RES-01` and `SVG-RES-01`.
5. Split or remove only after measurements: `C-3`, `C-4`, `PB-005`, and `IMG-ARCH-01`.

## Dependencies

The correctness fixes should land before structural cleanup. Any change to layout, conversion, PDF, or image output requires the relevant package tests and the full `make test` and `make golden` gates. Resource and concurrency changes also require `make test-race`.
