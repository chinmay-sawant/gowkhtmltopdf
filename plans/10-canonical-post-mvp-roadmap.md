# 10 - Post-MVP Quality & Capability Roadmap (Canonical Execution Ledger)

> **Parent:** `plans/00-canonical-pure-go-rewrite.md` (MVP phases 0–9 complete, v0.1.0)  
> **Status:** active - Tier 1 partially landed (12, 13, 15 complete; 10/11/14/16 partial) as of 2026-08-04  

> **Estimated effort:** Tier 1 ~4–8 mo · Tier 2 ~6–12 mo additional · Tier 3 deferred (not planned as pure-stdlib goal)  
> **Constraint:** pure Golang, **Go standard library only** (no third-party modules, no Chrome/WebKit, no cgo, no plugins)  
> **Ordering principle:** **quick wins first** (docs/API polish → typography → image mode → invoice CSS → broader CSS → fonts/i18n → web heuristics → JS research). Dependency edges still respected within each phase.  
> **Workflow:** [`skills/phase-wise-checklist/SKILLS.md`](../skills/phase-wise-checklist/SKILLS.md)

---

## Overview

MVP ships a usable **report/invoice** pipeline (load → HTML/CSS subset → layout → paginate → PDF/PNG). This ledger is the **live execution map** for post-MVP work that makes gowkhtmltopdf:

1. **Honest** about what it supports (fidelity docs + matrix)
2. **Easy to embed** from other Go apps (library API)
3. **Visually solid** for controlled reports (real bold/italic, spacing, image-mode quality, invoice CSS)
4. **Broader** for “most of my jobs” (flex/float lite, multi-font/Unicode, pagination polish)
5. **Careful** about open-web / JS ambitions under stdlib-only (staged, evidence-gated, Tier 3 deferred)

**Product tiers (from product framing):**

| Tier | Goal | This ledger |
|------|------|-------------|
| **Tier 1** | Solid report engine - “better for our use” | Phases 10–16 (must) |
| **Tier 2** | Leave wkhtmltopdf for most jobs | Phases 17–20 |
| **Tier 3** | Compete on the open web | Phase 23 - **deferred**; Chrome/Playwright territory |

**Hard non-goals unless this ledger is amended:**

- Full WebKit/Chrome pixel parity under pure stdlib
- Shipping third-party PDF/HTML engines or browser embeds
- Claiming Wikipedia parity before matrix + fixtures prove it

---


## Recommended next work (2026-08-04 reconcile)

Tier 1 is **not closed** yet. Suggested order for the next sessions:

1. **Phase 10 remainder** (docs-only, quick): write `documentation/fidelity.md`, link from docs index, stamp matrix audit date.
2. **Phase 16 remainder** (highest product value left in Tier 1): `float` lite, real `inline-block`, `box-sizing: border-box`.
3. **Phase 14 remainder** (small): document JPEG/PNG alpha/DPI knobs; optional `web.images=false` test.
4. **Phase 11 remainder** (optional): publish/install story + optional `ConvertHTML` helper.

**Already shipped post-MVP:** phases **12** (real bold/italic faces), **13** (spacing/coalesce), **15** (image-mode TTF AA). Selector expansion inside **16** is also shipped.


## Executive Summary

| Fact (current evidence) | Location |
|-------------------------|----------|
| MVP phases 0–9 complete | `plans/00-canonical-pure-go-rewrite.md` |
| Liberation Sans R/B/I/BI embedded | `internal/pdf/assets/` + `faces.go` |
| Real bold/italic via `FaceSet.Resolve` (fake bold only if face missing) | `layout.faceFor`, `paint.go` |
| Image mode = pure-Go TTF outline AA (+ 2× supersample) | `internal/imageout/ttfraster.go` |
| Selectors: attr / nth-child / siblings shipped; float still no | matrix §4 / §2.2 |
| Flex/grid/position = no | matrix §2.2 / §5 |
| JS stripped; flags warn only | load + CLI |
| Library API exists (`Converter`, examples, integration docs) | `api.go`, `examples/`, `library-api.md` |
| Wikipedia smoke PDF exists, layout poor | `output/wiki-ana-de-armas.pdf` |
| Zero module deps | `go.mod` |

**Quick-win dependency order:**

```
10 Docs/matrix honesty
  → 11 Library API polish (embedder DX)
  → 12 Real bold/italic faces
  → 13 Typography spacing
  → 14 PDF images robust path
  → 15 Image-mode TTF/AA raster
  → 16 Invoice CSS (selectors + float lite)
  → 17 Partial flex / position
  → 18 Pagination polish (thead repeat)
  → 19 Fonts/i18n discovery + CJK path
  → 20 HF / links edge cases
  → 21 Arbitrary websites / “paste any URL”
  → 22 JavaScript (staged; full engine gated)
  → 23 Tier 3 open-web (explicitly deferred)
```

---

## Phase Index

| Phase | Title | Tier | Detail ledger | Effort (solo) | Status |
|------:|-------|------|---------------|---------------|--------|
| 10 | HTML/CSS fidelity documentation | 1 | [phases/phase-10-fidelity-docs.md](phases/phase-10-fidelity-docs.md) | 3–7 days | partial |
| 11 | Library API for Go embedders | 1 | [phases/phase-11-library-api-embedders.md](phases/phase-11-library-api-embedders.md) | 1–2 wk | partial |
| 12 | Typography - real bold/italic faces | 1 | [phases/phase-12-typography-faces.md](phases/phase-12-typography-faces.md) | 2–4 wk | `[x]` 2026-08-04 |
| 13 | Typography - spacing stability | 1 | [phases/phase-13-typography-spacing.md](phases/phase-13-typography-spacing.md) | 1–2 wk | `[x]` 2026-08-04 |
| 14 | Images in PDF - robust path | 1 | [phases/phase-14-pdf-images.md](phases/phase-14-pdf-images.md) | 1–2 wk | partial |
| 15 | Image mode - real TTF/AA raster | 1 | [phases/phase-15-image-mode-raster.md](phases/phase-15-image-mode-raster.md) | 3–6 wk | `[x]` 2026-08-04 |
| 16 | CSS invoices use (selectors + float lite) | 1 | [phases/phase-16-invoice-css.md](phases/phase-16-invoice-css.md) | 3–6 wk | partial (selectors done) |
| 17 | Broader CSS (position/float, partial flex) | 2 | [phases/phase-17-broader-css.md](phases/phase-17-broader-css.md) | 2–4 mo | `[ ]` |
| 18 | Pagination polish (thead repeat, breaks) | 2 | [phases/phase-18-pagination-polish.md](phases/phase-18-pagination-polish.md) | 3–6 wk | `[ ]` |
| 19 | Fonts / i18n / discovery / CJK | 2 | [phases/phase-19-fonts-i18n.md](phases/phase-19-fonts-i18n.md) | 1–3 mo | `[ ]` |
| 20 | HF / links edge cases | 2 | [phases/phase-20-hf-links-edges.md](phases/phase-20-hf-links-edges.md) | 2–4 wk | `[ ]` |
| 21 | Arbitrary websites / paste-any-URL | 2→3 | [phases/phase-21-arbitrary-websites.md](phases/phase-21-arbitrary-websites.md) | 2–4 mo | `[ ]` |
| 22 | JavaScript support (staged) | 2→3 | [phases/phase-22-javascript.md](phases/phase-22-javascript.md) | research + years for full | `[ ]` |
| 23 | Tier 3 open-web competition | 3 | [phases/phase-23-tier3-deferred.md](phases/phase-23-tier3-deferred.md) | n/a | `[~]` deferred |

---

## Goal → Phase Map

| User goal | Primary phase(s) |
|-----------|------------------|
| HTML/CSS fidelity documentation (phase-wise checklist) | **10** (+ updates in 12–17, 21) |
| Full JavaScript support | **22** (honest stages; full engine not free under stdlib) |
| Fonts / localization / installed + folder fonts | **12**, **19** |
| API library for other Go projects | **11** (MVP API exists in phase 8) |
| Arbitrary websites (Wikipedia, marketing) | **21** (+ **17**, **19**) |
| “Paste any URL and get a decent print” | **21** |
| Flex / grid / floats / position | **16** (float lite), **17** (partial flex/position); grid `[~]` |
| JS-driven pages | **22** |
| Bold/italic families + CJK | **12** (bold/italic), **19** (CJK/Unicode) |
| Image mode like a real screenshot (not 5×7) | **15** |
| Years of edge-case print CSS | **16–18**, **20**; remainder in **23** |
| Tier 1 solid report engine | **10–16** |
| Tier 2 leave wkhtmltopdf for most jobs | **17–20** |
| Tier 3 compete on open web | **23** deferred |

---

## Phase 10: HTML/CSS Fidelity Documentation

> Detail: [phases/phase-10-fidelity-docs.md](phases/phase-10-fidelity-docs.md)  
> **Why first:** pure documentation; unblocks honest product messaging and every later matrix update.

### 10.1 Fidelity contract
- [ ] Publish a **fidelity guide** that states: report HTML first; not a browser
- [ ] Cross-link matrix ↔ fixtures ↔ sample PDFs with pass/fail/partial
- [ ] Document Tier 1 / 2 / 3 expectations and this roadmap

### 10.2 Matrix honesty pass
- [ ] Re-audit every “Implemented / Partial / Not implemented” row against current code
- [x] Mark image-mode text quality limits explicitly (TTF AA shipped; residual no FreeType hinting)
- [x] Mark font-face / bold / italic / CJK limits explicitly

### 10.3 Closure
- [ ] No code change required; skip `make lint`/`make test` for pure docs (skill rule)
- [ ] Point README “Deferred” table at this ledger

---

## Phase 11: Library API for Go Embedders

> Detail: [phases/phase-11-library-api-embedders.md](phases/phase-11-library-api-embedders.md)  
> **Baseline:** Phase 8 complete (`api.go`, `examples/pdf`, `examples/image`, `documentation/library-api.md`)

### 11.1 Embedder DX
- [ ] Expand library docs: invoice-from-bytes, multi-object, image convert, ACL pair
- [ ] Add copy-paste integration examples (HTTP handler pattern, no Gin dep in module)
- [ ] Document thread-safety, deterministic output, error types

### 11.2 API surface polish
- [ ] Review public API for missing helpers (e.g. convert HTML string → PDF bytes without temp dance if still awkward)
- [ ] Ensure examples build in CI / documented `go run`
- [ ] Optional: `go doc` completeness on exported types

### 11.3 Closure
- [ ] `make test` + `make lint` green; examples verified

---

## Phase 12: Typography - Real Bold (and Italic)

> Detail: [phases/phase-12-typography-faces.md](phases/phase-12-typography-faces.md)  
> **Tier 1 #1** · Related issues: multi-font, font-spacing

### 12.1 Faces
- [x] Bundle OFL Liberation Sans **Bold** (+ Italic / BoldItalic if available under same license)
- [x] Font registry: map `font-weight` / `font-style` → face
- [x] PDF: multiple subset fonts (`F0`/`F1`/…); drop fake-bold when real bold selected

### 12.2 Layout/paint
- [x] Measure with the selected face metrics
- [x] `<b>`, `<strong>`, `font-weight:bold` use bold face end-to-end
- [x] `<i>`, `<em>`, `font-style:italic` use italic face when present

### 12.3 Closure
- [x] Fixture visual smoke + tests; matrix + docs updated

---

## Phase 13: Typography - Spacing Stability

> Detail: [phases/phase-13-typography-spacing.md](phases/phase-13-typography-spacing.md)  
> **Tier 1 #1 (spacing)** · Related: issue font-spacing · **Status:** complete 2026-08-04

### 13.1 Advances
- [x] Align layout advances with PDF paint advances
- [x] Reduce word-by-word `Tj` fragmentation where safe (`coalesceTextItems`)
- [x] Regression tests for fixture-01 / fixture-16 spacing

### 13.2 Closure
- [x] No return of double letter-spacing (1000-unit `/Widths` still correct)

---

## Phase 14: Images in PDF - Robust Path

> Detail: [phases/phase-14-pdf-images.md](phases/phase-14-pdf-images.md)  
> **Tier 1 #4** · **Status:** partial

### 14.1 Robustness
- [x] Harden PNG/JPEG embed path for logos and grids (fixtures 07, 20)
- [x] Broken-image / missing-src behavior documented + tested (skip without crash)
- [x] Intrinsic size / CSS width/height interactions covered (tests exist; docs polish remain)

### 14.2 Closure
- [ ] Golden/structure tests; matrix image notes current

---

## Phase 15: Image Mode - Real Raster (Not 5×7)

> Detail: [phases/phase-15-image-mode-raster.md](phases/phase-15-image-mode-raster.md)  
> **Tier 1 #2** · Related: issue image-mode-raster-quality · **Status:** complete 2026-08-04

### 15.1 Raster path
- [x] Replace 5×7 text drawing with pure-Go TTF outline raster (or greyscale AA bitmap of same face)
- [x] Image paint advances **match** layout advances for same face/size
- [x] Anti-aliased edges; more than ~6 unique colors on body text samples

### 15.2 Closure
- [x] `make samples` PNG quality gate; docs state strategy

---

## Phase 16: CSS Invoices Actually Use

> Detail: [phases/phase-16-invoice-css.md](phases/phase-16-invoice-css.md)  
> **Tier 1 #3** · Selectors + float lite before full flex · **Status:** partial (selectors shipped)

### 16.1 Selectors
- [x] Attribute selectors subset; `:first-child` / `:last-child` / simple `:nth-child`
- [x] Sibling combinators match correctly (not as descendant)

### 16.2 Layout additions
- [ ] `float: left|right` + `clear` lite (enough for two-column invoice chrome)
- [ ] `display: inline-block` real layout (not degrade-to-block only)
- [ ] Optional: simple `box-sizing: border-box`

### 16.3 Closure
- [ ] New fixtures; matrix updated; still stdlib-only

---

## Phase 17: Broader CSS (Position / Float Full / Partial Flex)

> Detail: [phases/phase-17-broader-css.md](phases/phase-17-broader-css.md)  
> **Tier 2 #6**

### 17.1 Position & float
- [ ] `position: relative` offsets; float stacking refinements
- [ ] `[~]` `absolute` / `fixed` only if report need proven

### 17.2 Partial flex
- [ ] `display: flex` row + basic `justify-content` / `align-items` / `gap` subset
- [ ] `[~]` flex-wrap, flex-grow complexity - staged
- [ ] Grid remains out of allowlist unless amended

### 17.3 Closure
- [ ] Fixtures + matrix; Wikipedia still not claimed

---

## Phase 18: Pagination Polish

> Detail: [phases/phase-18-pagination-polish.md](phases/phase-18-pagination-polish.md)  
> **Tier 2 #8**

### 18.1 Tables & breaks
- [ ] Repeat table header (`thead`) on continued pages
- [ ] Smarter `page-break-inside` / orphan-widow heuristics
- [ ] Wire `--zoom` through convert if still missing

### 18.2 Closure
- [ ] Multi-page table fixture proves header repeat

---

## Phase 19: Fonts / i18n / Discovery / CJK

> Detail: [phases/phase-19-fonts-i18n.md](phases/phase-19-fonts-i18n.md)  
> **Tier 2 #7** · Localization + installed/folder fonts

### 19.1 Discovery
- [ ] Font search paths: bundled → user folder flag → optional system dirs (documented ACL)
- [ ] Map CSS `font-family` lists to discovered faces
- [ ] `@font-face` local `src` subset (no network webfont download without policy)

### 19.2 Unicode
- [ ] Type0/CID or multi-face path for common CJK / beyond Latin-1
- [ ] Document shaping limits (no HarfBuzz): CJK OK-ish; Arabic/Indic not claimed

### 19.3 Closure
- [ ] CJK fixture; matrix Unicode section honest

---

## Phase 20: HF / Links Edge Cases

> Detail: [phases/phase-20-hf-links-edges.md](phases/phase-20-hf-links-edges.md)  
> **Tier 2 #9**

### 20.1 Gaps from README deferred list
- [ ] Inline `#anchor` link source rects where possible
- [ ] `resolveRelativeLinks`; HTML HF links on body pages
- [ ] `[topage]` with copies; dump-outline TOC offset

### 20.2 Closure
- [ ] Golden multi-chapter + HF regression

---

## Phase 21: Arbitrary Websites / “Paste Any URL”

> Detail: [phases/phase-21-arbitrary-websites.md](phases/phase-21-arbitrary-websites.md)  
> **Not full parity** - “decent print” bar

### 21.1 Product bar
- [ ] Define “decent print”: title + main article body readable; chrome reduced
- [ ] Vendored Wikipedia-class HTML fixture for CI (no live net required)
- [ ] Optional reader-mode / chrome-strip heuristics (documented, opt-in)

### 21.2 Progressive CSS for marketing/wiki
- [ ] Consume phases 16–17 features against wiki snapshot
- [ ] Media-query basics that help print (`@media print` already partial)

### 21.3 Closure
- [ ] Smoke command + acceptance notes; **no** “Wikipedia parity” claim

---

## Phase 22: JavaScript Support (Staged)

> Detail: [phases/phase-22-javascript.md](phases/phase-22-javascript.md)  
> **Constraint note:** a full ES engine is a multi-year product under pure stdlib. Stages gate ambition.

### 22.1 Honesty & architecture
- [ ] Spec: what subset of DOM/JS report templates actually need
- [ ] Keep scripts stripped by default; enable only with explicit flag

### 22.2 Incremental capability
- [ ] Stage A: keep `--javascript-delay` as post-load wait only (document)
- [ ] Stage B: optional pure-Go **expression/template** hooks if product needs (not full JS)
- [ ] Stage C: research pure-Go ES interpreter scope (stdlib-only) - spike, go/no-go
- [ ] `[~]` Stage D: full browser JS/DOM - **not planned** under stdlib; would require plan amendment (embed runtime or abandon constraint)

### 22.3 Closure for any shipped stage
- [ ] Security review (script must not escape sandbox / FS)
- [ ] Matrix + threat model updated

---

## Phase 23: Tier 3 - Compete on Open Web (Deferred)

> Detail: [phases/phase-23-tier3-deferred.md](phases/phase-23-tier3-deferred.md)

### 23.1 Explicit deferral
- [~] Real browser or large CSS/JS stack - people use Chrome headless / Playwright; pure-stdlib reimplementation is the wrong goal for open-web parity  
  **Owner boundary:** product decision only; next gate = amend constraint or accept external engine  
  **Pointer:** this row is the sole Tier-3 ledger; do not re-open in active Tier-1/2 phases without amendment

---

## Dependencies

```mermaid
flowchart TD
  P10[10 Fidelity docs] --> P11[11 Library API]
  P10 --> P12[12 Bold/italic faces]
  P12 --> P13[13 Spacing]
  P12 --> P15[15 Image-mode raster]
  P13 --> P16[16 Invoice CSS]
  P14[14 PDF images] --> P16
  P16 --> P17[17 Broader CSS]
  P16 --> P18[18 Pagination]
  P12 --> P19[19 Fonts/i18n]
  P17 --> P21[21 Arbitrary websites]
  P19 --> P21
  P18 --> P20[20 HF/links]
  P21 --> P22[22 JavaScript staged]
  P22 --> P23[23 Tier3 deferred]
```

| Rule | Note |
|------|------|
| Docs (10) first | Every later phase updates matrix when shipping features |
| Faces (12) before image raster (15) | Share font registry / metrics |
| Invoice CSS (16) before web (21) | Report CSS is the quick win; web reuses it |
| JS (22) after layout breadth | DOM without CSS/layout is useless for “JS pages” |
| Tier 3 never blocks Tier 1 | Phase 23 stays `[~]` |

---

## Explicit Non-Goals (unless amended)

| Item | Reason |
|------|--------|
| Full WebKit parity under stdlib | Unbounded; Tier 3 |
| Third-party modules / cgo FreeType / browser embed | Project constraint |
| Network webfonts without ACL policy | Security |
| Arabic/Indic full shaping | Needs HarfBuzz-class stack |
| PDF encryption / PDF/A | Out of original scope |
| Claiming “screenshot parity” for image mode | Aim for “looks like a real print raster”, not Chrome |

---

## Completion Handoff Rules

Per [`skills/phase-wise-checklist/SKILLS.md`](../skills/phase-wise-checklist/SKILLS.md):

1. Mark a row `[x]` only after source/test evidence for that row.
2. Non-doc changes: record `make lint` + `make test` outcomes on phase closure.
3. Use `[~]` with reason + next gate for partials/deferrals.
4. Do not leave the same active work in two ledgers; point old rows here if superseded.
5. After each phase: brief result + next unchecked phase.

---

## Related Artifacts

| Artifact | Role |
|----------|------|
| `documentation/compatibility-matrix.md` | Support contract |
| `plans/PR/issues/issue-epic-rendering-quality-body.md` | GitHub epic #2 archive |
| `plans/PR/issues/issue-*-body.md` | Child issue archives |
| `output/` | Visual smoke PDFs/PNGs |
| `testdata/golden/` | Report fixtures |
| `README.md` Deferred table | Must stay synced with this ledger |
