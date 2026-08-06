# Pending — Phase 2: Open-web CSS residuals (Phase 17 / 21 leftovers)

> **Parent:** [`README.md`](README.md)  
> **Status:** partial (2026-08-05) — inventory + attr ops + wiki-like print chrome; live Ana page-count still open  
> **Estimated effort:** 1–3 weeks (slice by symptom)  
> **Prior plan coverage:** **Yes** — Tier-2 pending-3 shipped float/table/multicol/etc.; Phase 21 §21.3 still lists **residuals**  

---

## Overview

Core Phase 17 pending-3 work is done, but live Wikipedia still shows ~32 pages
vs Chrome ~10, IPA glyph junk (→ Phase 3), and black links until skins match
(Phase 1 done for `:link`; Vector still needs more selectors/media).

### Smoke proof

```sh
./bin/gowkhtmltopdf 'https://en.wikipedia.org/wiki/Ana_de_Armas' output/wiki-ana-de-armas.pdf
```

Reference: `output/chrome_ana.pdf`.

---

## Phase 2 checklist

### 2.1 Inventory (evidence-first)

- [x] Ana vs Chrome: ~32 vs ~10 pages; IPA mangled; links black on live skin (synthetic `:link` OK)
- [x] Vector print CSS hides `#mw-navigation,.noprint,nav,#footer,…` inside `@media print` — **parses and matches** in engine (`TestWikiPrintChromeHidden`)
- [x] Vector **screen** layout uses CSS Grid + `@media screen and (min-width:…)` — correctly classified as `screen` and filtered for print
- [x] Wiki figure CSS uses `[typeof~='mw:File/Thumb']` — was dropped (invalid attr parse) before `~=` support
- [x] Map: page-count density → Phase 4 print media + typography; IPA → Phase 3; SVG logo → Phase 8 out of scope

### 2.2 Float / infobox

- [x] Vendored `wiki-like-article.html` keeps `table.infobox { float: right }` (CI)
- [ ] Live Ana infobox density / wrap polish — still Partial (needs more evidence-driven float passes)

### 2.3 Tables / citation chrome

- [ ] Citation `[ n ]` spacing on live Ana — deferred (skin + inline spans); not blocking fixture

### 2.4 Pagination density

- [ ] Live Ana page-count reduction — **open** (pair with Phase 4); fixture stays ≤3 pages
- [x] Fixed attribute `~=` / `*=` so print/figure rules are not dropped (`TestAttrWordAndSubstring`)

### 2.5 Docs

- [x] Matrix attribute selectors → Partial (`~=` / `*=`)
- [x] This inventory recorded; Phase 21 §21.3 pointer remains valid

### 2.6 Gates

- [x] `make lint` → OK (with Phase 2 code)
- [x] `go test ./internal/css ./internal/layout ./internal/convert -count=1` → OK
- [ ] Full `make test` recorded on commit
- [x] Status → **partial**; remaining live density → Phase 4 / later slices

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 `:link` | Fixture link colors |
| Phase 3 fonts | IPA / missing glyphs |
| Phase 4 print media | Page-count |

---

## Out of scope

- Matching Chrome’s exact 10-page pagination in this slice
- Full Vector skin fidelity
