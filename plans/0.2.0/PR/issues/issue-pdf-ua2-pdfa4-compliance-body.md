## Context

**Parent epic:** [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29) - newer PDF versions and compliance

PDF/UA-2 and PDF/A-4 are conformance profiles. They are not implied by emitting a PDF 1.7 or PDF 2.0 header. This issue defines the standards-oriented compliance work separately from the raw format support tracked by [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31) and [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32).

The current writer does not claim PDF/A or PDF/UA conformance. A credible implementation needs profile-specific metadata, document structure, embedded fonts, color management, accessibility semantics, and repeatable validation with independent tools.

## Scope (in)

1. Select the exact PDF/UA-2 and PDF/A-4 conformance profiles to support and document their requirements.
2. Build a standards matrix covering tagged PDF structure, document language and title, logical reading order, headings, alternative text, font embedding, Unicode maps, metadata, ICC output intent, color spaces, prohibited features, and deterministic output.
3. Add profile-aware writer policy and validation before serialization.
4. Add XMP metadata, output-intent handling, tagging, and accessibility structures required by the selected profiles.
5. Add representative compliance fixtures and independent validator commands to the repository workflow.
6. Define clear rejection behavior when an input or writer feature cannot satisfy the selected profile.
7. Update documentation and compatibility claims only for profiles that pass the validation gates.

## Out of scope

- Raw PDF 1.7 version support, tracked by [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31).
- Raw PDF 2.0 version support, tracked by [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32).
- Claiming compliance from a file header, metadata string, or partial validator result.
- Full encryption, digital signatures, AcroForms, or other independent feature families without separate scope.
- HTML/CSS layout-engine changes, which remain separate from this epic.

## Success criteria

- [ ] The supported PDF/UA-2 and PDF/A-4 profiles and their limitations are documented.
- [ ] A profile matrix identifies every required emitted feature and validation rule.
- [ ] Representative PDF/UA-2 and PDF/A-4 fixtures pass the selected independent validators.
- [ ] Missing or incompatible features produce explicit errors or a documented non-conformant result.
- [ ] Fonts, Unicode mapping, metadata, color output intent, tagging, and reading-order behavior have targeted tests.
- [ ] Output remains deterministic and existing PDF 1.4, PDF 1.7, and PDF 2.0 version tests are not regressed.
- [ ] User-facing documentation makes no broader compliance claim than the validator evidence supports.

## Plan

1. Select validator tools and record exact versions and commands for reproducible checks.
2. Convert PDF/UA-2 and PDF/A-4 requirements into a repository-owned standards matrix.
3. Design profile policy and validation seams in `internal/pdf` or a focused compliance package.
4. Implement metadata, ICC output intent, font, tagging, structure, and accessibility requirements in small vertical slices.
5. Add positive and negative fixtures for each supported profile.
6. Run independent validators and publish only the proven compliance boundary.

### Suggested code ownership

| Concern | Primary code | Expected change |
|---|---|---|
| Profile policy | `internal/pdf` or new compliance package | Centralize PDF/UA-2 and PDF/A-4 requirements |
| Metadata | `internal/pdf`, focused metadata files | Emit XMP and document metadata required by the profiles |
| Color and output intent | `internal/pdf/images.go`, focused color files | Validate ICC output intent and color-space restrictions |
| Fonts and Unicode | `internal/pdf/fonttype0.go`, font files | Require embedding and complete Unicode mapping |
| Tagged structure | layout and PDF object seams | Preserve logical structure, reading order, and accessibility metadata |
| Validation | compliance fixtures and external validator scripts | Prove profile conformance and rejection behavior |
| Documentation | `README.md`, `documentation/compatibility-matrix.md` | Publish only validated profile support |

## References

- Parent epic: [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)
- PDF 1.7 child: [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31)
- PDF 2.0 child: [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32)
- Current compliance boundary: `documentation/compatibility-matrix.md:255`
- Existing PDF feature gap: `plans/exploration/02-pdf-converter.md:24`
- Current project claims: `README.md:403`
