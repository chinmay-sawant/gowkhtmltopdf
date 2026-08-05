# Tier 2 Pending-3 — Nested HTML headers/footers as child documents

> **Parent:** [`plans/phases/phase-20-hf-links-edges.md`](../phase-20-hf-links-edges.md)  
> **Supersedes:** [`../subplans-tier-2/nested-hf-v0.3.0.md`](../subplans-tier-2/nested-hf-v0.3.0.md) (was deferred to v0.3.0; **now in scope**)  
> **Status:** not started  
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

- [ ] Document call order: body `Paint` → TOC/internal links → `materializeCopies` → `drawHeadersFooters`
- [ ] Cite `internal/convert/hf.go`: `loadHTMLHF`, `hfHeightFor`, `drawHTMLHF`, `drawHeadersFooters`, `hfParms.substitute`
- [ ] Cite `internal/convert/links.go`: `buildBodyIDIndex`, `remapPageForCopies`
- [ ] Cite tests: `hf_links_test.go` (`TestHTMLHeaderFragmentGoTo*`, external URI)
- [ ] Proof: `go test ./internal/convert -run 'HTMLHeader|HF' -count=1` baseline green before edits

### 1.2 Architecture decision

- [ ] HF = **child layout document** with viewport = content width × reserved band height
- [ ] Reuse: load → `collectSheets` → `MergeFontFaces` → `layout.Layout` → clipped paint
- [ ] Path: extract `internal/convert/hf_doc.go` (or deepen `hf.go` without cycles)
- [ ] **Forbid** independent multi-page HF pagination (single-page clamp + clip)
- [ ] **Forbid** CSS `position: running()` / named pages in this subplan
- [ ] Keep `--header-html` / `--footer-html` as **URL only** (raw markup skip stays)

### 1.3 Non-goals (even when complete)

- [~] Running elements / GCPM named pages — permanent unless amended
- [~] HF JavaScript / wkhtmltopdf query-string `subst()` / `window.status`
- [~] Browser nested browsing context HF
- [~] PDF named actions beyond GoTo/URI

---

## Phase 2: Child document pipeline

### 2.1 Fonts & registry

- [ ] Thread body `*pdf.Registry` (or HF-local registry merged from `--font-path`) into HF `layout.Options`
- [ ] Call `MergeFontFaces` for HF stylesheets under same `FetchSub` ACL as body
- [ ] HF `@font-face` local TTF/OTF works; WOFF/remote/`data:` still skipped (parity)
- [ ] ACL deny on HF font → Liberation fallback; no panic
- [ ] Path: `hf.go` / `hf_doc.go` + `convert.MergeFontFaces`
- [ ] Proof: new `TestHTMLHeaderFontFaceLocal` in `hf_links_test.go` or `hf_doc_test.go`

### 2.2 Layout richness

- [ ] HF layout uses same modes as body: block, table, flex, grid lite, images, relative/absolute (within band)
- [ ] Spike fixture: HF HTML with flex row + `<img>` + text + `<a href="#id">`
- [ ] Clip overflow to margin band in draw (existing clip path); log warning if `res.Height > band`
- [ ] Images in HF via same `imagesFn` / ACL as body
- [ ] Path: `loadHTMLHF` Options + draw clip
- [ ] Proof: `TestHTMLHeaderFlexImage` paints without panic; image object present when ACL allows

### 2.3 Height reservation

- [ ] Keep `hfHeightFor` measuring layout **before** body pagination (`effectiveMargins`)
- [ ] Document: tall HF content is **clipped**, not allowed to steal unbounded body space
- [ ] Optional: cap measured height to a documented max fraction of page if product wants
- [ ] Proof: existing margin/auto-HF tests still green; new tall-HF clip test

### 2.4 Placeholders & copies

- [ ] Keep `[page]`/`[topage]`/`[webpage]`/… string substitute via `hfParms`
- [ ] Re-layout HF only when placeholders present (`perPage`); cache otherwise
- [ ] Draw **after** `materializeCopies` (already)
- [ ] Proof: `TestHTMLHeaderFragmentGoToCopies` + non-collate; add placeholder text assert under `--copies=2`

---

## Phase 3: Links (regression-critical)

### 3.1 Preserve shipped behavior

- [ ] External URI from HF → `AddLinkURI` gated by `ExternalLinks`
- [ ] `#id` from HF → body `AddLinkDest` via `buildBodyIDIndex`
- [ ] Missing id / `LocalLinks=false` → silent skip
- [ ] Dest page indices remapped with `remapPageForCopies` (collate + non-collate)
- [ ] `resolveRelativeLinkURIs` on HF ops with HF base URL
- [ ] Proof: full `go test ./internal/convert -run 'HTMLHeader|FragmentGoTo|HF' -count=1`

### 3.2 HF-only ids

- [ ] Document: `#id` destinations are **body** ids, not ids inside the HF tree
- [ ] Matrix / README honesty sentence

---

## Phase 4: Fixtures & tests

### 4.1 Required tests

- [ ] Extend `hf_links_test.go` for flex + image HF + fragment GoTo
- [ ] `TestHTMLHeaderFontFaceLocal` (+ ACL deny)
- [ ] `TestHTMLHeaderTallContentClipped` — no overlap into body content box
- [ ] `TestHTMLHeaderPlaceholdersCopies` — `[page]`/`[topage]` correct under copies
- [ ] Keep existing GoTo/URI/copies tests green

### 4.2 Optional golden

- [~] New golden `fixture-36-hf-nested-flex.html` (+ header/footer HTML files) — only if convert golden runner can load HF URLs cleanly
- [~] Otherwise keep asserts in convert package tests only

### 4.3 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Record outcomes beside this section

---

## Phase 5: Docs & pointer hygiene

### 5.1 Product docs

- [ ] Matrix header/footer: nested HTML HF = **Partial** (child layout; clipped band; no browser HF)
- [ ] `cli.md` / `library-api.md`: note HF supports body CSS subset + local `@font-face` under ACL
- [ ] README deferred: remove “nested HF → v0.3.0”; mark Partial / shipped when done
- [ ] `phase-20-hf-links-edges.md`: Out of scope → narrow; Pending pointer here
- [ ] Rewrite `subplans-tier-2/nested-hf-v0.3.0.md` as `[~]` superseded pointer to this file

### 5.2 Closure

- [ ] Parent Phase 20 nested-HF row → `[x]` when proven
- [ ] Next: orphans-widows-css or float-table-packing

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
