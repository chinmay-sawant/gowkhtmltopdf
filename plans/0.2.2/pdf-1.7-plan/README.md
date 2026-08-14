# gowkhtmltopdf — PDF 1.7 Plan

> **Highest 1.7 compliance (not this ledger):** **PDF/A-3a + PDF/UA-1** — [../pdf-1.7-compliance-plan/](../pdf-1.7-compliance-plan/)
> **Issue:** [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31) — pdf: PDF 1.7 **version** support
> **Parent epic:** [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)
> **Sibling:** [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32) / [../pdf-2.0-plan/](../pdf-2.0-plan/) — PDF 2.0 version
> **Not this plan:** PDF/A-3a + PDF/UA-1 (compliance ledger); [#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33) PDF/A-4 + PDF/UA-2
> **Status:** completed
> **Workflow:** [`skills/phase-wise-checklist/SKILLS.md`](../../../skills/phase-wise-checklist/SKILLS.md)

This version plan only makes a legal `%PDF-1.7` file. It does **not** claim PDF/A or PDF/UA.

| Base | Highest archival | Highest accessibility |
|------|------------------|------------------------|
| **PDF 1.7** | PDF/A-3a (ISO 19005-3) | PDF/UA-1 (ISO 14289-1) |
| PDF 2.0 | PDF/A-4 (ISO 19005-4) | PDF/UA-2 (ISO 14289-2) |

Canonical ledger: [00-canonical-pdf-17-plan.md](00-canonical-pdf-17-plan.md)

Spec notes (ISO 32000-1 citations): [SPEC-NOTES.md](SPEC-NOTES.md)

This plan **introduces** the shared `WriterPolicy` type (`PDF14`, `PDF17`, reserved `PDF20`). The 2.0 plan extends it; it must not create a second policy type.

This is **not** a new PDF engine and **not** a layout rewrite. gowkhtmltopdf already converts authored HTML through `load → html → css → layout → convert → internal/pdf`. Issue #31 adds an explicit PDF 1.7 emit path on that writer.

Each phase is a checklist you can execute independently. Complete them in order unless the ledger says otherwise.

| Phase | File | Goal | Gate |
|------:|------|------|------|
| 1 | [phase-01-version-policy-and-header.md](phase-01-version-policy-and-header.md) | Shared version policy + `%PDF-1.7` header | Unit: header/version, default still 1.4 |
| 2 | [phase-02-catalog-trailer-metadata.md](phase-02-catalog-trailer-metadata.md) | Trailer `/ID`, Info + UTF-16BE, non-claiming XMP | Unit: trailer/catalog structure |
| 3 | [phase-03-fonts-images-content-gates.md](phase-03-fonts-images-content-gates.md) | Version gates on existing fonts/images/content | Existing font/image tests + 1.7 fixtures |
| 4 | [phase-04-settings-cli-library.md](phase-04-settings-cli-library.md) | Settings, CLI, library select PDF 1.7 | CLI/library tests; unknown values error |
| 5 | [phase-05-convert-pipeline.md](phase-05-convert-pipeline.md) | `convert` constructs the document with the policy | End-to-end `%PDF-1.7` from HTML |
| 6 | [phase-06-validation-and-goldens.md](phase-06-validation-and-goldens.md) | Structural fixtures, semantic parse, goldens | Parser + golden needles |
| 7 | [phase-07-docs-and-honesty.md](phase-07-docs-and-honesty.md) | Docs distinguish version from conformance | `make claim-scan` |
| 8 | [phase-08-closure.md](phase-08-closure.md) | Lint, test, 1.4 default proof, no #32/#33 claims | `make lint` + `make test` |

**Default output stays PDF 1.4** until a later, explicit compatibility transition.

**Out of scope here:** HTML/CSS/layout, image mode, PDF 2.0 emit (sibling), **PDF/A-3 / PDF/UA-1** ([compliance plan](../pdf-1.7-compliance-plan/)), PDF/A-4, PDF/UA-2, encryption, signatures, AcroForm, object streams, xref streams, linearization, JPEG 2000, CFF/OTTO, a second writer package.
