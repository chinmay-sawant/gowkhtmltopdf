# Plans — v0.2.2 (Newer PDF versions)

| File / Folder | Role |
|---------------|------|
| [pdf-1.7-plan/](pdf-1.7-plan/) | **[#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31)** PDF 1.7 version support — **completed** |
| [pdf-1.7-compliance-plan/](pdf-1.7-compliance-plan/) | Highest **1.7** conformance: **PDF/A-3a + PDF/UA-1** — **completed** |
| [pdf-2.0-plan/](pdf-2.0-plan/) | **[#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32)** PDF 2.0 version support — **completed** |
| [pdf-a4-ua2-compliance-plan.md](pdf-2.0-plan/pdf-a4-ua2-compliance-plan.md) | **[#33](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/33)** Highest **2.0** conformance: **PDF/A-4 + PDF/UA-2** — **completed** |
| [criticality-optimization-checklist.md](criticality-optimization-checklist.md) | Post-#45/#46 criticality + optimization follow-up (no new claims) — **not started** |

Workflow: [../../skills/phase-wise-checklist/SKILLS.md](../../skills/phase-wise-checklist/SKILLS.md)

Parent epic: [GitHub #29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29) — newer PDF versions and compliance.

Predecessor: [../0.2.1/24-canonical-0.2.1-roadmap.md](../0.2.1/24-canonical-0.2.1-roadmap.md) (phases 24–30).

This release does **not** rebuild layout, fonts, or the convert pipeline. It extends the existing PDF 1.4 writer with an explicit version policy: [#31](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/31) owns `WriterPolicy` and the PDF 1.7 path; [#32](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/32) extends that policy with PDF 2.0.

**Highest compliance by version**

| Base | Archival | Accessibility | Plan |
|------|----------|---------------|------|
| PDF 1.7 | PDF/A-3a | PDF/UA-1 | [pdf-1.7-compliance-plan/](pdf-1.7-compliance-plan/) |
| PDF 2.0 | PDF/A-4 | PDF/UA-2 | [pdf-a4-ua2-compliance-plan.md](pdf-2.0-plan/pdf-a4-ua2-compliance-plan.md) |
