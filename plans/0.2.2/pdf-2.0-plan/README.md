# gowkhtmltopdf — PDF 2.0 Plan

> **Issue:** [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32) — pdf: PDF 2.0 support
> **Parent epic:** [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)
> **Sibling:** [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31) / [../pdf-1.7-plan/](../pdf-1.7-plan/) — shared version policy
> **Highest 2.0 compliance (not this ledger):** **PDF/A-4 + PDF/UA-2** — [#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33)
> **1.7 equivalent:** [../pdf-1.7-compliance-plan/](../pdf-1.7-compliance-plan/) — PDF/A-3a + PDF/UA-1
> **Not this plan:** [#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33) — PDF/A-4 and PDF/UA-2
> **Status:** not started
> **Workflow:** [`skills/phase-wise-checklist/SKILLS.md`](../../../skills/phase-wise-checklist/SKILLS.md)

Canonical ledger: [00-canonical-pdf-20-plan.md](00-canonical-pdf-20-plan.md)

This is **not** a new PDF engine and **not** a layout rewrite. gowkhtmltopdf already converts authored HTML through `load → html → css → layout → convert → internal/pdf`. Issue #32 only adds an explicit PDF 2.0 emit path on that existing writer.

Each phase is a checklist you can execute independently. Complete them in order unless the ledger says otherwise.

| Phase | File | Goal | Gate |
|------:|------|------|------|
| 1 | [phase-01-version-policy-and-header.md](phase-01-version-policy-and-header.md) | Shared version policy + `%PDF-2.0` header | Unit: header/version, default still 1.4 |
| 2 | [phase-02-catalog-trailer-metadata.md](phase-02-catalog-trailer-metadata.md) | Catalog, trailer `/ID`, Info + non-claiming XMP | Unit: trailer/catalog structure |
| 3 | [phase-03-fonts-images-content-gates.md](phase-03-fonts-images-content-gates.md) | Version gates on existing fonts/images/content | Existing font/image tests + 2.0 fixtures |
| 4 | [phase-04-settings-cli-library.md](phase-04-settings-cli-library.md) | Settings, CLI, library select PDF 2.0 | CLI/library tests; unknown values error |
| 5 | [phase-05-convert-pipeline.md](phase-05-convert-pipeline.md) | `convert` constructs the document with the policy | End-to-end `%PDF-2.0` from HTML |
| 6 | [phase-06-validation-and-goldens.md](phase-06-validation-and-goldens.md) | Structural fixtures, semantic parse, goldens | Parser + golden needles |
| 7 | [phase-07-docs-and-honesty.md](phase-07-docs-and-honesty.md) | Docs distinguish version from conformance | `make claim-scan` |
| 8 | [phase-08-closure.md](phase-08-closure.md) | Lint, test, 1.4 default proof, no #33 claims | `make lint` + `make test` |

**Default output stays PDF 1.4** until a later, explicit compatibility transition.

**Out of scope here:** HTML/CSS/layout changes, image mode, PDF/A-4, PDF/UA-2, encryption, signatures, AcroForm, object streams, xref streams, linearization, Zerodha/JSON templates, a second writer package.
