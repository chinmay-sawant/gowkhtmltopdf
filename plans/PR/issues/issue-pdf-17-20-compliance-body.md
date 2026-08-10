## Context

**Parent epic:** [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)

The current writer emits PDF 1.4 files. This issue is specifically for establishing PDF 1.7 format and feature support. PDF 2.0 support is tracked separately in [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32).

PDF 1.7 support must mean more than changing the `%PDF-1.4` header to `%PDF-1.7`. The writer needs an explicit version policy, feature compatibility rules, and validation evidence. The implementation is centered in `internal/pdf`:

```go
// internal/pdf/pdf.go
// Package pdf implements a stdlib-only PDF 1.4 writer.
```

The issue should define the supported PDF 1.7 baseline honestly. It must distinguish between files the writer emits, features it accepts, and features that have been validated by a PDF parser or validator.

## Scope (in)

1. Audit the current PDF 1.4 output and map its header, objects, xref, catalog, pages, fonts, images, links, outlines, metadata, and compression behavior to PDF 1.7 requirements.
2. Define a central PDF version policy for PDF 1.4 and PDF 1.7. Do not scatter raw version checks across the writer.
3. Define the PDF 1.7 feature baseline that Gowkhtmltopdf will support, including any required object syntax, metadata, transparency, color, font, and catalog behavior.
4. Add version-aware writer settings and reject or clearly report unsupported feature combinations before serialization.
5. Preserve PDF 1.4 as the default until the compatibility transition is explicitly approved.
6. Add structural fixtures and parser or validator checks for PDF 1.7 output.

### Proposed version policy seam

```go
type PDFVersion int

const (
    PDF14 PDFVersion = iota
    PDF17
)

type WriterPolicy struct {
    Version PDFVersion
}
```

The exact API is open for design. The important invariant is that version and feature decisions are validated before the writer emits objects, rather than inferred from a header after the fact.

## Out of scope

- PDF 2.0 serialization or feature support, tracked by [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32).
- Claiming PDF 1.7 support by changing only the file header.
- Changing the default output from PDF 1.4 before compatibility tests and migration notes exist.
- Encryption, forms, signatures, or other independent feature families without separately approved scope.
- Browser, HTML, or CSS layout changes, tracked by [#30](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/30).

## Success criteria

- [ ] Current PDF 1.4 behavior remains covered by baseline fixtures and tests.
- [ ] A documented feature matrix distinguishes PDF 1.4 and PDF 1.7 emitted, accepted, and validated behavior.
- [ ] PDF 1.7 output has an explicit version selection path and structural validation.
- [ ] Unsupported PDF 1.7 feature combinations are rejected or reported clearly.
- [ ] PDF 1.7 tests cover the header, catalog, object syntax, xref, fonts, images, links, and metadata that are in scope.
- [ ] The default PDF 1.4 output remains unchanged until a deliberate compatibility transition is approved.

## Plan

1. Audit `internal/pdf/pdf.go`, `content.go`, `images.go`, font files, annotations, outlines, and existing PDF tests.
2. Record the PDF 1.7 feature baseline under `documentation/` or `plans/`.
3. Introduce a central version policy and validate unsupported combinations early.
4. Add PDF 1.7 structural fixtures and parser or validator checks while preserving the PDF 1.4 baseline.
5. Update the compatibility matrix and user-facing documentation with only verified claims.

### Suggested code ownership

| Concern | Primary code | Expected change |
|---|---|---|
| Version policy | `internal/pdf/pdf.go`, new version policy type | Centralize PDF 1.4 and PDF 1.7 selection and feature gates |
| Header and document catalog | `internal/pdf/pdf.go` | Emit version and catalog entries from one policy |
| Objects and xref | `internal/pdf/content.go`, writer files | Add only syntax required by the approved PDF 1.7 baseline |
| Fonts and Unicode | `internal/pdf/fonts.go`, `fonttype0.go` | Verify versioned font requirements and Unicode maps |
| Images and transparency | `internal/pdf/images.go` | Validate color space, alpha, filters, and version gates |
| Validation | `internal/pdf/*_test.go`, external validator fixtures | Test structure and the claimed PDF 1.7 boundary |
| Documentation | `README.md`, `documentation/architecture.md`, `documentation/compatibility-matrix.md` | Publish only verified support claims |

## References

- Parent epic: [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)
- PDF 2.0 child: [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32)
- Related layout issue, intentionally separate: [#30](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/30)
- Current writer: `internal/pdf/pdf.go`
- Current PDF contract: `README.md:50-54`, `documentation/architecture.md:18`
- Existing gap: `plans/exploration/02-pdf-converter.md:24`
- Existing PDF test contract: `internal/pdf/pdf_test.go:45`
