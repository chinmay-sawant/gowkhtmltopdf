# Phase 5 — Layout Tagging Bridge

> **Parent:** `plans/0.2.2/pdf-1.7-compliance-plan/00-canonical-pdf-17-compliance-plan.md`
> **Status:** completed
> **Estimated effort:** 1–2 weeks
> **Depends on:** Phase 4 writer API
> **Unblocks:** phases 6–7 end-to-end HTML fixtures

---

## Overview

`internal/layout` already turns HTML into a display list and paints
into `*pdf.Document`. Under UA-1 / A-3a, that paint must also open
the right structure elements.

Do not change pagination or CSS. Add a tagging visitor (or paint-time
hooks) that maps **authored** HTML semantics. Decorative chrome
(repeating headers/footers, page numbers) is `/Artifact`.

If a required role cannot be assigned (image without `alt` in UA-1
mode), fail the conversion rather than emit a claiming file.

---

## Mapping (minimum)

| HTML / convert | Structure / mark |
|----------------|------------------|
| document root | `Document` |
| `h1`–`h6` | `H1`–`H6` |
| `p` and untitled blocks of body text | `P` |
| `table` / `tr` / `th` / `td` | `Table` / `TR` / `TH` / `TD` (MCID on the cell, not the row) |
| `ul`/`ol` / `li` | `L` / `LI` |
| `img` with `alt` | `Figure` + `/Alt` |
| `img` without `alt` in UA-1 / dual | error |
| `a[href]` | `Link` + existing annot + `/OBJR` |
| convert header/footer bands | `/Artifact /Pagination` |
| cover page chrome | artifact or omit from tree consistently |

---

## Phase 5 checklist

### 5.1 Seam

- [x] Layout does not import veraPDF or XMP builders
- [x] Tagging is driven by the document policy (no-op when unclaimed)
- [x] Paint order and reading order match for a single-column body (document that multi-column / float reading order is best-effort)

### 5.2 Body mapping

- [x] Headings, paragraphs, tables, lists, figures, links as in the table above
- [x] Language: document `/Lang` from HTML `lang` or a setting; default documented
- [x] Test: one HTML fixture with h1, p, table, img[alt], and a URI link produces those types

### 5.3 Artifacts

- [x] Default text/HTML header and footer bands marked as pagination artifacts
- [x] Test: HF text is not a `P` in the structure tree

### 5.4 Fail closed

- [x] UA-1 / dual + image missing `alt` → typed error, no PDF
- [x] UA-1 / dual + empty title → typed error (phase 4 catalog rule, still true through convert)
- [x] Unclaimed 1.7 HTML jobs still paint untagged (existing goldens)

---

## Explicitly out of scope

- Pixel-perfect reading order for floats/grid
- MathML / Formula (UA-2)
- Settings flag (phase 6) — tests may construct `WriterPolicy` directly

---

## Done when

`convert.Run` with a dual policy on a small HTML file emits tagged
content + artifacts, and the same HTML without a profile is untagged
1.7 or 1.4 as requested.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 4 API | Phase 7 HTML fixtures |
