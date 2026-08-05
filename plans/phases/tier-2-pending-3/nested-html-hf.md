# Tier 2 Pending-3 — Nested HTML headers/footers as child documents

> **Parent:** [`plans/phases/phase-20-hf-links-edges.md`](../phase-20-hf-links-edges.md)  
> **Supersedes:** [`../subplans-tier-2/nested-hf-v0.3.0.md`](../subplans-tier-2/nested-hf-v0.3.0.md) (was deferred to v0.3.0; **now in scope**)  
> **Status:** done (registry + MergeFontFaces + tests + docs/gates)  
> **Estimated effort:** 2–4 weeks  
> **Constraint:** stdlib-only; wkhtmltopdf-compatible nested HTML HF (not browser HF, not CSS running elements)  
> **Spec / docs:** [wkhtmltopdf usage](https://wkhtmltopdf.org/usage/wkhtmltopdf.txt) · CSS GCPM (non-goal contrast)

---

## Overview

Today HTML HF is a **margin-band mini-layout** (`loadHTMLHF` → stamp ops in
`drawHTMLHF`). This subplan deepens it into a **child document pipeline** that
reuses the body convert/layout subset (CSS, flex/grid, images, fonts via
`MergeFontFaces` + `Registry`) while remaining clipped to the reserved header/
footer band. Preserve shipped URI + fragment GoTo and copies-aware remapping.

**Model choice (research):** stay on the **wkhtmltopdf separate-HTML-document**
axis — not WeasyPrint/Prince running elements.

---

## Executive Summary

| Concern | Today | Target |
|---------|-------|--------|
| Load | `loadHTMLHF` URL + sheets + `layout.Layout` | Same + `MergeFontFaces` + `Options.Registry` |
| Pipeline | Ad-hoc stamp in `drawHTMLHF` | `hf_doc.go` child layout result; clip to band |
| Fonts | Body registry **not** passed to HF | HF faces via shared registry / local `@font-face` |
| Links | URI + `#id` GoTo (copies-aware) | **Preserve** + re-test |
| Height | `hfHeightFor` → `effectiveMargins` | Keep; warn/clamp if content > band |
| Placeholders | `[page]`/`[topage]` after copies | Keep string substitute (no HF JS) |

---

## Phase 1: Design & evidence baseline

### 1.1 Current seams (do not break)

- [x] Document call order: body `Paint` → TOC/internal links → `materializeCopies` → `drawHeadersFooters`
- [x] Cite `internal/convert/hf.go`: `loadHTMLHF`, `hfHeightFor`, `drawHTMLHF`, `drawHeadersFooters`, `hfParms.substitute`
- [x] Cite `internal/convert/links.go`: `buildBodyIDIndex`, `remapPageForCopies`
- [x] Cite tests: `hf_links_test.go` (`TestHTMLHeaderFragmentGoTo*`, external URI, fontface, flex+image, tall clip)
- [x] Proof: `go test ./internal/convert -run 'HTMLHeader|RemapPage' -count=1` → PASS (2026-08-05)

### 1.2 Architecture decision

- [x] HF = **child layout document** with viewport = content width × reserved band height
- [x] Reuse: load → `collectSheets` → `MergeFontFaces` → `layout.Layout` → clipped paint
- [x] Path: deepened `hf.go` (`htmlHFLayout.registry`); no separate browser HF
- [x] **Forbid** independent multi-page HF pagination (single-page clamp + clip)
- [x] **Forbid** CSS `position: running()` / named pages in this subplan
- [x] Keep `--header-html` / `--footer-html` as **URL only** (raw markup skip stays)

### 1.3 Permanent product boundaries (documented, not deferred work)

- [x] Running elements / GCPM named pages — out of product scope (nested HTML HF ≠ GCPM)
- [x] HF JavaScript / wkhtmltopdf query-string `subst()` / `window.status` — out of product scope
- [x] Browser nested browsing context HF — out of product scope
- [x] PDF named actions beyond GoTo/URI — out of product scope

---

## Phase 2: Child document pipeline

### 2.1 Fonts & registry

- [x] Thread body `*pdf.Registry` into HF `layout.Options` (`objectState.registry`)
- [x] Call `MergeFontFaces` for HF stylesheets under same `FetchSub` ACL as body
- [x] HF `@font-face` local TTF/OTF works via shared path
- [x] ACL deny on HF font → Liberation fallback; no panic (shared MergeFontFaces)
- [x] Path: `hf.go` + `convert.MergeFontFaces`
- [x] Proof: `TestHTMLHeaderFontFaceLocal`

### 2.2 Layout richness

- [x] HF layout uses same modes as body (flex/grid/images via shared `layout.Layout`)
- [x] Spike: HF HTML with flex row + `<img>` + link (`TestHTMLHeaderFlexImage`)
- [x] Clip overflow to margin band in draw; tall content clamped
- [x] Images in HF via same `imagesFn` / ACL as body
- [x] Path: `loadHTMLHF` Options + draw clip
- [x] Proof: `TestHTMLHeaderFlexImage` + `TestHTMLHeaderTallContentClipped`

### 2.3 Height reservation

- [x] Keep `hfHeightFor` measuring layout **before** body pagination (`effectiveMargins`)
- [x] Document: tall HF content is **clipped**, not allowed to steal unbounded body space
- [x] Proof: `TestHTMLHeaderTallContentClipped`

### 2.4 Placeholders & copies

- [x] Keep `[page]`/`[topage]`/`[webpage]`/… string substitute via `hfParms`
- [x] Re-layout HF only when placeholders present (`perPage`); cache otherwise
- [x] Draw **after** `materializeCopies` (already)
- [x] Proof: existing GoTo copies tests + `TestHTMLHeaderPlaceholdersCopies`

---

## Phase 3: Links (regression-critical)

### 3.1 Preserve shipped behavior

- [x] External URI from HF → `AddLinkURI` gated by `ExternalLinks`
- [x] `#id` from HF → body `AddLinkDest` via `buildBodyIDIndex`
- [x] Missing id / `LocalLinks=false` → silent skip
- [x] Dest page indices remapped with `remapPageForCopies` (collate + non-collate)
- [x] `resolveRelativeLinkURIs` on HF ops with HF base URL
- [x] Proof: `go test ./internal/convert -run 'HTMLHeader|RemapPage' -count=1` → PASS

### 3.2 HF-only ids

- [x] Document: `#id` destinations are **body** ids, not ids inside the HF tree
- [x] Matrix / README honesty sentence

---

## Phase 4: Fixtures & tests

### 4.1 Required tests

- [x] Extend `hf_links_test.go` for flex + image HF + fragment GoTo
- [x] `TestHTMLHeaderFontFaceLocal`
- [x] `TestHTMLHeaderTallContentClipped`
- [x] `TestHTMLHeaderPlaceholdersCopies`
- [x] Keep existing GoTo/URI/copies tests green

### 4.2 Optional golden

- [ ] New golden `fixture-36-hf-nested-flex.html` (+ header/footer HTML files) when convert golden runner can load HF URLs cleanly

### 4.3 Gates

- [x] `make lint` → PASS (`go vet ./...`, 2026-08-05)
- [x] `go test ./internal/layout ./internal/convert -count=1` → PASS (layout 0.063s, convert 0.752s; 2026-08-05)
- [x] Record outcomes beside this section

---

## Phase 5: Docs & pointer hygiene

### 5.1 Product docs

- [x] Matrix header/footer: nested HTML HF = **Partial** (child layout; clipped band; no browser HF)
- [x] `cli.md` / `library-api.md`: note HF supports body CSS subset + local `@font-face` under ACL; `#id` → body only
- [x] README deferred: nested HF Partial / shipped child pipeline
- [x] `phase-20-hf-links-edges.md`: Pending pointer here → done when gates pass
- [x] Rewrite `subplans-tier-2/nested-hf-v0.3.0.md` as superseded pointer

### 5.2 Closure

- [x] Parent Phase 20 nested-HF row → `[x]` when lint/test + docs done
- [x] Next: orphans-widows-css / float-table-packing (wave 1 parallel)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 20 GoTo/URI | Link semantics to preserve |
| Flex/grid/sticky shipped | HF layout richness |
| `MergeFontFaces` | HF `@font-face` |

---

## Risks

| Risk | Mitigation |
|------|------------|
| Copies × GoTo dest wrong | Keep `remapPageForCopies`; both collate modes tested |
| Double layout cost | Cache non-placeholder HF; measure once for height |
| Font subset timing | Ensure HF glyphs join embed/subset before write |
| Scope creep to running elements | Explicit non-goal; matrix honesty |

---

## Out of scope

- Full browser HTML HF
- CSS running elements / named pages
- HF that paginates independently across many pages
- HF JavaScript
