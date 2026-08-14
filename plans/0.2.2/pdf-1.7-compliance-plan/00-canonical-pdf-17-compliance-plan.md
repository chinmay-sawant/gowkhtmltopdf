# 00 — PDF 1.7 Compliance (PDF/A-3a + PDF/UA-1)

> **Parent:** `plans/0.2.2/README.md` — epic [#29](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/29)
> **Highest 1.7 profile:** PDF 1.7 + **PDF/A-3a** + **PDF/UA-1**
> **Status:** not started
> **Estimated effort:** 4–7 weeks across phases 1–8
> **Constraint:** pure Go, no CGO, no new direct modules. veraPDF is an **optional** test binary, not a Go require.
> **Depends on:** #31 version path (`WriterPolicy`, `%PDF-1.7`, `/ID`, Info, non-claiming XMP) — **completed**
> **Not #33:** that issue is PDF/A-4 + PDF/UA-2 on PDF 2.0
> **Workflow:** [`skills/phase-wise-checklist/SKILLS.md`](../../../skills/phase-wise-checklist/SKILLS.md)
> **Spec notes:** [SPEC-NOTES.md](SPEC-NOTES.md)

---

## Overview

#31 shipped an honest **PDF 1.7 version**. Files are `%PDF-1.7` with trailer `/ID`, UTF-16BE Info, and XMP that **does not claim** PDF/A or PDF/UA. `WriterPolicy.ConformanceProfile` rejects every profile (`ErrConformanceProfilesUnsupported`, message points at #33).

That is the right split: version ≠ conformance. This ledger is the missing 1.7 **compliance** work.

The highest pair that still uses ISO 32000-1 (PDF 1.7):

```text
PDF 1.7  ── archival ──────►  PDF/A-3a   (ISO 19005-3, level A)
         └── accessibility ►  PDF/UA-1   (ISO 14289-1:2014)
```

PDF/A-4 and PDF/UA-2 need PDF 2.0. They stay on #33. PDF/A-2a is a legal 1.7 archive but **not** the highest part (A-3 adds associated files; we can claim A-3a without using attachments).

A header swap or a `pdfaid` string without OutputIntent, embedded fonts, and a real structure tree is not compliance.

---

## Where compliance lands

```text
--pdf-version 1.7  +  --pdf-profile a3a-ua1   (names bikeshed)
        │
        ▼
settings.PdfGlobal
        │
        ▼
WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/A-3a+PDF/UA-1"}
        │
        ├─ internal/pdf finalize     XMP claims, OutputIntent, ICC, StructTreeRoot
        ├─ internal/pdf content      BDC/EMC, artifacts
        └─ internal/layout paint     semantic roles from HTML (h1, table, img alt, a)
                │
                ▼
        veraPDF -f 3a  and  -f ua1
```

Layout already paginates and paints. This plan adds a **tagging seam**: paint ops that represent real content carry a structure role; header/footer bands are artifacts. Do not fork a second layout engine.

### Package ownership

| Concern | Primary code | Change |
|---------|--------------|--------|
| Profile policy | `internal/pdf/policy.go` `ConformanceProfile` | Accept A-3a / UA-1 / dual; still reject A-4, UA-2, A-1 |
| Claiming XMP | `internal/pdf` metadata emit (1.7 already has a non-claiming packet) | Add `pdfaid` + `pdfuaid` + UA extension schema |
| OutputIntent / ICC | new focused files under `internal/pdf` | sRGB (+ gray) profiles, catalog `/OutputIntents` |
| Color rewrite | `internal/pdf/images.go` | ICCBased / DefaultRGB under A-3 |
| Fonts | `fonttype0.go`, `ensureFont` | Fail conversion if a used face cannot be embedded |
| Structure + marked content | new `internal/pdf` structure types + `content.go` | StructTreeRoot, MCID, ParentTree, BDC/EMC |
| HTML → tags | `internal/layout` paint / convert HF | Map headings, p, tables, lists, figures, links; HF → Artifact |
| Settings / CLI / library | `settings`, `cli/flags.go`, `api.go` | Opt-in profile; implies PDF 1.7 |
| Validation | `internal/pdf/*_test.go`, optional script | veraPDF `3a` + `ua1`, skip if missing |
| Docs | `compatibility-matrix.md`, `deferred.md` | Publish only after phase 7 |

---

## Feature matrix (headline dual profile)

| Feature | Unclaimed 1.7 (#31) | Dual A-3a + UA-1 (this plan) | Validated by |
|---------|---------------------|------------------------------|--------------|
| Header | `%PDF-1.7` | `%PDF-1.7` | existing |
| Trailer `/ID` | yes | yes (required by A-3) | existing + A-3 fixture |
| Info | UTF-16BE | kept, consistent with XMP | unit |
| XMP without pdfaid | yes | **no** — must claim | byte test |
| `pdfaid:part=3` `conformance=A` | no | yes | XMP parse |
| `pdfuaid:part=1` + extension schema | no | yes | XMP parse |
| OutputIntent + sRGB ICC | no | yes | catalog test |
| DeviceRGB images | yes | rewritten to ICCBased | image test |
| Fonts embedded + ToUnicode | yes | required; else error | negative test |
| Encryption / JS / forms | rejected | still rejected | existing policy tests |
| StructTreeRoot / MarkInfo / Lang | no | yes | writer fixture |
| BDC/EMC + ParentTree | no | yes | writer fixture |
| HF as `/Artifact` | n/a | yes | convert fixture |
| HTML `h1`/`table`/`img`/`a` | untagged paint | standard structure types | convert fixture |
| veraPDF `-f 3a` | n/a | PASS (skip if no binary) | phase 7 |
| veraPDF `-f ua1` | n/a | PASS (skip if no binary) | phase 7 |
| veraPDF `-f 4` / `-f ua2` | n/a | **must not** be claimed | negative |

---

## In scope

1. Document the 1.7 vs 2.0 compliance split (this file + SPEC-NOTES).
2. Extend `WriterPolicy` so A-3a / UA-1 / dual are first-class and validated **before** `Write`.
3. Claiming XMP, OutputIntent, ICC, color rewrite, embed-or-fail fonts.
4. PDF/UA-1 structure tree and marked content (ISO 32000-1 tags, **not** PDF 2.0 namespaces).
5. Layout/convert bridge so authored HTML produces those tags.
6. Opt-in settings/CLI/library. Default remains unclaimed 1.4.
7. Dual fixtures and skippable veraPDF `3a` + `ua1`.
8. Docs that never call unclaimed 1.7 “PDF/A” or “accessible.”

## Out of scope

- PDF/A-4, PDF/UA-2, PDF 2.0 namespaces (#33 / #32).
- PDF/A-1 (wrong base; bad pairing with UA-1).
- Making A-3b / A-3u / UA-1-only product modes (may be added later; not the headline).
- Shipping associated-file / portfolio features just to justify A-3 over A-2.
- Encryption, signatures, AcroForm, XFA, JavaScript.
- Image mode.
- Requiring veraPDF in CI if the binary is absent.

---

## Phase map

```text
1 Profile policy + matrix
  → 2 PDF/A-3 archival objects
      → 3 Color + fonts under A-3
          → 4 PDF/UA-1 structure (writer)
              → 5 Layout tagging bridge
                  → 6 Settings / CLI / library
                      → 7 veraPDF + goldens
                          → 8 Docs + closure
```

Phases 2 and 4 may overlap after phase 1 if they do not fight over `finalize`. Phase 5 needs phase 4’s BDC/EMC API. Phase 6 may start the setting once the profile names exist, but must not ship a user path until 2–5 can emit a real dual file.

| Phase | File | Goal |
|------:|------|------|
| 1 | [phase-01-profile-policy-and-matrix.md](phase-01-profile-policy-and-matrix.md) | Policy + matrix |
| 2 | [phase-02-pdfa3-archival.md](phase-02-pdfa3-archival.md) | XMP + OutputIntent |
| 3 | [phase-03-color-fonts-under-a3.md](phase-03-color-fonts-under-a3.md) | Color + embed-or-fail |
| 4 | [phase-04-pdfua1-structure.md](phase-04-pdfua1-structure.md) | Tagged writer |
| 5 | [phase-05-layout-tagging-bridge.md](phase-05-layout-tagging-bridge.md) | HTML → tags |
| 6 | [phase-06-settings-cli-library.md](phase-06-settings-cli-library.md) | Opt-in surface |
| 7 | [phase-07-verapdf-and-goldens.md](phase-07-verapdf-and-goldens.md) | Proof |
| 8 | [phase-08-docs-and-closure.md](phase-08-docs-and-closure.md) | Honesty |

---

## Success criteria

- [ ] Highest 1.7 profile is documented as PDF/A-3a + PDF/UA-1 (not A-4 / UA-2)
- [ ] Dual-mode fixture PASSes veraPDF `-f 3a` and `-f ua1` (or skip is recorded)
- [ ] Unclaimed 1.4 and 1.7 version tests stay green
- [ ] A-4 / UA-2 profile strings still error
- [ ] Missing title, missing `alt`, or unembeddable font fails the profile **before** a claiming file is written
- [ ] Docs do not call default output PDF/A or PDF/UA

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| #31 `WriterPolicy` + 1.7 XMP/`/ID` | Objects to specialize |
| Existing layout display list + HF bands | Tagging targets |
| #33 | Nothing — sibling, later, on PDF 2.0 |
