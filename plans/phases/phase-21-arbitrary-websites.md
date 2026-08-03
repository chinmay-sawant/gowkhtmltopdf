# Phase 21 - Arbitrary Websites & “Paste Any URL → Decent Print”

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** not started  
> **Estimated effort:** 2–4 months (iterative)  
> **Depends on:** Phases 16–17 CSS; Phase 19 fonts strongly help  
> **Unblocks:** marketing sites / Wikipedia-class **readable** prints (not parity)  
> **Tier:** 2→3 boundary · **Constraint:** stdlib-only; no browser embed

---

## Overview

Target **Wikipedia and marketing sites** with a product bar of **“decent print”**, not pixel-perfect clone. Reuse invoice/report CSS expansions; add print heuristics (chrome reduction), vendored fixtures, and honest acceptance criteria. Full open-web competition remains Phase 23 deferred.

## Executive Summary

| Artifact | Role |
|----------|------|
| `output/wiki-ana-de-armas.pdf` | Current smoke - layout poor |
| Matrix CSS gaps | flex/float/selectors/fonts |
| No JS | SPA/marketing hydration missing (phase 22) |

**Decent print definition (acceptance):**

1. Primary article **title** visible early (not buried under 5 pages of nav)
2. Main **body text** readable, multi-page OK
3. Reduced useless chrome (search, appearance menus) when heuristics enabled
4. Non-Latin text: tofu only when fonts missing (phase 19)
5. Still may ignore JS widgets, sticky headers, complex grids

---

## Phase 21 checklist

### 21.1 Product contract

- [ ] Write “decent print” criteria into `documentation/fidelity.md`
- [ ] Explicit non-claim: not Wikipedia visual parity; not marketing pixel match
- [ ] CLI UX: document `gowkhtmltopdf 'https://…' out.pdf` security (SSRF, untrusted HTML)

### 21.2 Vendored fixtures (CI-safe)

- [ ] Capture reduced HTML snapshot of Wikipedia article body (or synthetic wiki-like DOM) under `testdata/web/`
- [ ] Capture one simple marketing landing (static HTML) under `testdata/web/`
- [ ] Do **not** require live network in `make test`
- [ ] Optional live smoke job (manual / nightly) documented separately

### 21.3 CSS application of prior phases

- [ ] Inventory wiki/marketing CSS that still breaks after phases 16–17
- [ ] Prioritize remaining **high-impact** properties only (list with evidence)
- [ ] Media: strengthen `@media print` usage on sites that provide print CSS
- [ ] `[~]` Large site stylesheets performance: cap rules / time if needed

### 21.4 Chrome-strip / reader heuristics (opt-in)

- [ ] Design opt-in flag e.g. `--print-media-type` already exists; add `--simplify-dom` or reader-mode **only if** product wants
- [ ] Heuristics examples (document each): hide `nav`/`footer` role, known wiki selectors, `display:none` on `#mw-navigation` for vendored snapshot
- [ ] Default **off** for controlled report HTML (do not break invoices)
- [ ] Security: heuristics must not fetch extra origins unexpectedly

### 21.5 “Paste any URL” path

- [ ] Loader timeouts/size limits sufficient for large HTML (verify)
- [ ] CSS + image subresources: concurrency and failure isolation
- [ ] Progress logs useful for long pages
- [ ] Graceful degrade when CSS huge: still emit PDF body text
- [ ] Document recommended flags for URL mode (disable JS already default; images on/off)

### 21.6 Acceptance tests

- [ ] Vendored wiki fixture: PDF contains title string; page count within band; not empty
- [ ] Vendored marketing fixture: hero text + primary CTA text present
- [ ] Live Ana de Armas command remains **manual** smoke; update `output/wiki-*.pdf` only when intentionally regenerating samples
- [ ] Visual notes in `documentation/samples.md`

### 21.7 Docs

- [ ] Fidelity section: arbitrary websites
- [ ] Matrix unchanged claims: still not full CSS
- [ ] README: “URL → decent print” under progressive goals, not MVP feature list until acceptance met

### 21.8 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Acceptance criteria signed against vendored fixtures
- [ ] Parent Phase 21 checked
- [ ] Next: **Phase 22** only if JS-driven pages are required; else stop at Tier 2

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| 16–17 CSS | Readable structure |
| 19 fonts | Non-Latin lists |
| Loader | URL fetch |

---

## Out of scope

- SPA hydration without JS engine
- Pixel-perfect Vector/Minerva skins
- Full “years of print CSS edge cases” (ongoing; track residuals as `[~]`)
- Tier 3 browser competition
