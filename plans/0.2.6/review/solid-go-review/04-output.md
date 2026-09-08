# Review - PDF, image output, SVG, outline, and profiles

> **Parent:** `skills/solid-go-review/SKILLS.md` - review workflow
> **Status:** Complete. All findings were implemented or closed by current-source proof.
> **Estimated effort:** Not estimated

---

## Overview

This packet covered `internal/pdf`, `internal/pdf/assets`, `internal/pdfprofile`, `internal/imageout`, `internal/svg`, and `internal/outline`.

## Executive Summary

The output packages document ownership, use typed PDF references, bound raster work, isolate non-thread-safe SVG calls, and expose a neutral outline seam. The main risks are output contracts, cancellation during finalization, unbounded font input, and a large image output file with a package-wide lint suppression.

## Files and responsibilities

- `internal/pdf/pdf.go:122-231` - PDF document ownership, policy, object allocation, and constructors.
- `internal/pdf/registry.go:17-181` - font registry and synchronization.
- `internal/pdf/fonts.go:79-95` - TrueType parsing.
- `internal/imageout/imageout.go:117-180` - raster entry points and layout setup.
- `internal/imageout/imageout.go:324-390` - raster allocation and painting.
- `internal/imageout/imageout.go:1631-1721` - image pipeline, encoding, and output writes.
- `internal/svg/raster.go:40-120` - SVG parsing, serialization, and panic recovery.
- `internal/outline/outline.go:18-29` - location reader seam and outline tree inputs.

## SOLID and Go verdicts

| Area | Verdict | Evidence |
| --- | --- | --- |
| SRP | MIXED | PDF responsibilities are grouped around document output, while image output combines layout, raster allocation, decoding, caching, painting, and encoding (`internal/imageout/imageout.go:324-390`, `internal/imageout/imageout.go:1631-1721`). |
| OCP | PASS | Output policies are selected through writer policy and format resolution, and outline consumes a neutral location reader (`internal/pdf/pdf.go:221-231`, `internal/outline/outline.go:18-29`). |
| LSP | N/A | No confirmed substitutable hierarchy with a failing contract was found in this packet. |
| ISP | PASS | Registry and outline seams are narrow. The output path mostly uses concrete types where no substitution contract is needed (`internal/pdf/registry.go:17-24`, `internal/outline/outline.go:18-29`). |
| DIP | MIXED | SVG serialization isolates a dependency behind a package function, but image finalization directly writes and encodes through the request sink (`internal/imageout/imageout.go:1684-1721`). |

## Confirmed findings

### Phase 1: Output correctness

- [x] `IMG-OUT-01` `internal/imageout/imageout.go:1712-1721` - encoded PNG and JPEG writes now reject silent short writes with `io.ErrShortWrite`. Proof: short-writer regression tests pass.
- [x] `PDF-OUT-01` `internal/pdf/pdf.go:553-575` - explicitly present empty PDF streams now serialize with stream markers and `/Length 0`. Proof: strict empty-stream parser test passes.
- [x] `PDF-OUT-02` `internal/pdf/pdf.go:669-707` - `WriteTo` now reports bytes accepted by the external sink after flush. Proof: prefix-writer flush failure test passes.

### Phase 2: Context and resource bounds

- [x] `IMG-CTX-01` `internal/imageout/imageout.go:1684-1721` - image finalization now checks cancellation before encoding and writing, while the sink contract remains synchronous. Proof: cancellation and blocking-writer tests pass.
- [x] `PDF-FONT-RES-01` `internal/pdf/registry.go:368-381` - font directory scans and direct TTF parsing now share a 32 MiB input limit. Proof: oversized scan and direct-parse tests pass.
- [x] `SVG-RES-01` `internal/svg/raster.go:50-59` - SVG input is limited to 32 MiB and detection inspects only a bounded prefix. Proof: oversized-input and bounded-probe tests pass.

### Phase 3: Structure and maintainability

- [x] `IMG-ARCH-01` `internal/imageout/imageout.go:1` - the package-wide lint suppression was removed; output and raster responsibilities now pass independent lint and package tests.

## Hypotheses

- [x] `D-H1` `internal/pdf/pdf.go:776-829` - shipped retry-safety fix: late write failure leaves the document retryable without duplicated finalization state. Proof: repeated-write regression test passes.
- [x] `D-H2` `internal/pdf/images.go:153-324` - shipped deduplication fix: repeated JPEG payloads now share the image deduplication path. Proof: repeated-image object-count test passes.
- [x] `D-H3` `internal/svg/raster.go:71-128` - shipped dimension-contract fix: raster dimensions describe logical CSS size while PNG pixels include supersampling. Proof: dimension-contract regression test passes.

Rows marked as validated non-findings or validated design decisions close a review question through source and test evidence. They do not claim that production architecture changed.

## Area score

**7.5/10.** The packages have strong ownership comments, caps, locking, and focused seams. Output, cancellation, and untrusted font input remain open risks.

## Validation

Focused output tests, `make test`, `make test-race`, `make lint`, and `make golden` passed.

## Dependencies

Fix output short writes and empty streams before refactoring image or PDF structure. Resource-bound changes require race and golden validation where pipeline behavior changes.
