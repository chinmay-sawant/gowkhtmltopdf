# gowkhtmltopdf — PDF 1.7 Compliance Plan

> **Highest 1.7 profile:** **PDF 1.7 + PDF/A-3a + PDF/UA-1**
> **Parent epic:** [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)
> **Depends on:** [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31) / [../pdf-1.7-plan/](../pdf-1.7-plan/) (version path, **completed**)
> **Not this plan:** [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32) PDF 2.0; [#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33) PDF/A-4 + PDF/UA-2
> **Status:** not started
> **Workflow:** [`skills/phase-wise-checklist/SKILLS.md`](../../../skills/phase-wise-checklist/SKILLS.md)

Canonical ledger: [00-canonical-pdf-17-compliance-plan.md](00-canonical-pdf-17-compliance-plan.md)

Spec notes: [SPEC-NOTES.md](SPEC-NOTES.md)

Issue #31 made a legal PDF 1.7 file. That is **not** archival or accessible. The highest standards that still use PDF 1.7 (ISO 32000-1) are:

| | Archival | Accessibility |
|--|----------|----------------|
| **1.7 (this plan)** | **PDF/A-3a** (ISO 19005-3, level A) | **PDF/UA-1** (ISO 14289-1:2014) |
| 2.0 (#33) | PDF/A-4 (ISO 19005-4) | PDF/UA-2 (ISO 14289-2:2024) |

Default compliant 1.7 mode = **both** claims on one file. Gate: veraPDF `-f 3a` **and** `-f ua1`.

This is not a new engine. It extends `WriterPolicy.ConformanceProfile` (today every profile is rejected) and teaches layout to emit tagged structure.

| Phase | File | Goal | Gate |
|------:|------|------|------|
| 1 | [phase-01-profile-policy-and-matrix.md](phase-01-profile-policy-and-matrix.md) | Profile enum + standards matrix | Unit: accept A-3a / UA-1; still reject A-4 / UA-2 |
| 2 | [phase-02-pdfa3-archival.md](phase-02-pdfa3-archival.md) | Claiming XMP, OutputIntent, ICC | Structure tests; no veraPDF yet |
| 3 | [phase-03-color-fonts-under-a3.md](phase-03-color-fonts-under-a3.md) | ICCBased images, forced embed, ToUnicode | Existing font/image tests + A-3 cases |
| 4 | [phase-04-pdfua1-structure.md](phase-04-pdfua1-structure.md) | StructTreeRoot, BDC/EMC, ParentTree | Writer-level tagged fixture |
| 5 | [phase-05-layout-tagging-bridge.md](phase-05-layout-tagging-bridge.md) | HTML → standard structure types | Convert fixture: headings/tables/figures/links |
| 6 | [phase-06-settings-cli-library.md](phase-06-settings-cli-library.md) | Opt-in profile selection | CLI/library tests; default still unclaimed 1.4 |
| 7 | [phase-07-verapdf-and-goldens.md](phase-07-verapdf-and-goldens.md) | Dual fixtures + veraPDF | `-f 3a` and `-f ua1` (skip if missing) |
| 8 | [phase-08-docs-and-closure.md](phase-08-docs-and-closure.md) | Honest docs | `make lint` + `make test` + claim-scan |

**Default output stays unclaimed PDF 1.4.** Compliance is opt-in and implies PDF 1.7.

**Out of scope:** PDF/A-4, PDF/UA-2, PDF/A-1, encryption, XFA, associated-file product features, image mode, a second writer.
