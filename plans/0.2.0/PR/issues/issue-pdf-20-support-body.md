## Context

**Parent epic:** [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29) - newer PDF versions and compliance

The current writer emits PDF 1.4 files. After the PDF 1.7 baseline in [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31) is defined, this issue will establish explicit PDF 2.0 support. PDF 2.0 is a file format version and feature set, not just a newer header value.

The implementation must preserve the existing PDF 1.4 default and make PDF 2.0 selection explicit. Each claimed feature needs structural tests and, where applicable, validation with an independent PDF parser or validator.

## Scope (in)

1. Define the PDF 2.0 feature baseline relevant to Gowkhtmltopdf.
2. Extend the central version policy introduced by the PDF 1.7 work instead of adding scattered version checks.
3. Add version-aware handling for PDF 2.0 document headers, catalogs, objects, cross-reference structures, metadata, fonts, images, and other supported features.
4. Add feature gates and clear errors for combinations that the writer has not implemented.
5. Add PDF 2.0 structural fixtures, golden outputs, and parser or validator checks.
6. Document migration, compatibility, and the relationship between PDF 2.0 and standards profiles.

### Suggested policy seam

```go
type PDFVersion int

const (
    PDF14 PDFVersion = iota
    PDF17
    PDF20
)

type WriterPolicy struct {
    Version PDFVersion
}
```

The final API may differ. Version selection must remain explicit, validated before serialization, and testable without changing the default output.

## Out of scope

- PDF 1.7 baseline work, tracked by [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31).
- PDF/UA-2 and PDF/A-4 conformance, tracked by [#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33).
- Claiming PDF 2.0 support by changing only `%PDF-1.4` to `%PDF-2.0`.
- Encryption, signatures, forms, or every optional PDF 2.0 feature without separate acceptance criteria.
- HTML/CSS layout changes, which remain separate from this epic.

## Success criteria

- [ ] PDF 2.0 support has a documented feature matrix with emitted, accepted, and validated states.
- [ ] Callers can explicitly select PDF 2.0 without changing the PDF 1.4 default.
- [ ] Generated PDF 2.0 files pass structural parsing and version-specific fixture checks.
- [ ] Unsupported feature combinations fail clearly before producing misleading output.
- [ ] PDF 1.4 and PDF 1.7 regression fixtures remain green.
- [ ] Documentation distinguishes PDF 2.0 version support from PDF/A-4 and PDF/UA-2 conformance.

## Plan

1. Audit the PDF 1.7 version policy and identify PDF 2.0 deltas that affect the current writer.
2. Define the minimum supported PDF 2.0 feature set and compatibility rules.
3. Extend version-aware handling in `internal/pdf` for approved syntax and document structures.
4. Add structural, golden, negative, and compatibility tests.
5. Validate representative files with an independent parser or validator.
6. Update user-facing documentation only after the validation gates pass.

### Suggested code ownership

| Concern | Primary code | Expected change |
|---|---|---|
| Version policy | `internal/pdf/pdf.go` | Add PDF 2.0 as an explicit version with feature gates |
| Catalog and objects | `internal/pdf/pdf.go`, `internal/pdf/content.go` | Emit approved PDF 2.0 structures |
| Fonts and images | `internal/pdf/fonttype0.go`, `internal/pdf/images.go` | Preserve resource compatibility and reject unsupported combinations |
| Validation | `internal/pdf/*_test.go`, external validator fixtures | Verify version and structure independently |
| Documentation | `README.md`, `documentation/compatibility-matrix.md` | Publish the supported PDF 2.0 boundary |

## References

- Parent epic: [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)
- PDF 1.7 child: [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31)
- PDF/UA-2 and PDF/A-4 child: [#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33)
- Current writer: `internal/pdf/pdf.go`
- Current output contract: `README.md:50-54`, `documentation/architecture.md:18`
- Existing PDF feature gap: `plans/exploration/02-pdf-converter.md:24`
