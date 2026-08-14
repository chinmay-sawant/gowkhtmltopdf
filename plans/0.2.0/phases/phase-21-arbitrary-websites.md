# Phase 21 - Arbitrary Websites & “Paste Any URL → Decent Print”

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** in progress (§21.1–21.7 shipped; open-web Chrome-gap work → [`pending-phase-items/`](pending-phase-items/README.md))  
> **Estimated effort:** 2–4 months (iterative)  
> **Depends on:** Phases 16–17 CSS; Phase 19 fonts strongly help  
> **Unblocks:** marketing sites / Wikipedia-class **readable** prints (not parity)  
> **Tier:** 2→3 boundary · **Constraint:** stdlib-only; no browser embed

---

## Overview

Target **Wikipedia and marketing sites** with a product bar of **“decent print”**, not pixel-perfect clone. Reuse invoice/template CSS expansions; add print heuristics (chrome reduction), vendored fixtures, and honest acceptance criteria. Full open-web competition remains Phase 23 deferred.

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

- [x] Write “decent print” criteria into `documentation/fidelity.md` — evidence: § Arbitrary websites (Phase 21), five acceptance bullets
- [x] Explicit non-claim: not Wikipedia visual parity; not marketing pixel match — evidence: fidelity non-claims table + banned claims language
- [x] CLI UX: document `gowkhtmltopdf 'https://…' out.pdf` security (SSRF, untrusted HTML) — evidence: `documentation/cli.md` § Remote URL security; README Usage URL example + Progressive goals; aligns with THREAT-MODEL §5–§7.1 / integration-security

### 21.2 Vendored fixtures (CI-safe)

- [x] Capture reduced HTML snapshot of Wikipedia article body (or synthetic wiki-like DOM) under `testdata/web/` — `wiki-like-article.html`
- [x] Capture one simple marketing landing (static HTML) under `testdata/web/` — `marketing-landing.html`
- [x] Do **not** require live network in `make test` — `web_fixtures_test.go` loads local files only
- [x] Optional live smoke job (manual / nightly) documented separately — `documentation/samples.md`

### 21.3 CSS application of prior phases

- [x] Inventory wiki/marketing CSS that still breaks after phases 16–17 — high-impact residuals (evidence):
  - **float + clear** on infobox (`table.infobox { float: right }` in `testdata/web/wiki-like-article.html`) — float lite exists; complex clear/intrusion still imperfect
  - **Large linked stylesheets** from live wiki skins — rule volume / unsupported selectors (matrix CSS gaps); `@media print` links already filtered in `collectSheets`
  - **flex/grid marketing heroes** — partial; see matrix §CSS
- [x] Prioritize remaining **high-impact** properties only (list with evidence) — inventory above; no new CSS in this pass
- [x] Media: strengthen `@media print` usage on sites that provide print CSS — verified existing: layout `Media: "print"` + `linkStylesheet` admits print/all (`internal/convert/convert.go`); `--print-media-type` stored but redundant
- [x] Large site stylesheets performance: cap rules / time — out of scope by design for Phase 21 (graceful degrade already; no hard caps unless evidence of CI pain)

### 21.4 Chrome-strip / reader heuristics (opt-in)

- [x] Design opt-in flag: `--simplify-dom` / `web.simplifydom` (default off); `--print-media-type` already exists and remains layout-print (no duplicate)
- [x] Heuristics: inject synthetic `display:none !important` sheet (`convert.SimplifyChromeCSS`) — **landmarks only** (`nav`/`footer`/`aside` + ARIA roles + `nav.site-nav`). MediaWiki IDs via `--simplify-dom-profile=mediawiki`. Proof: `TestSimplifyDOMOnHidesChrome`, `TestSimplifyDOMMediaWikiProfile`
- [x] Default **off** for authored HTML templates — `TestSimplifyDOMOffKeepsChrome`, `TestSimplifyDOMEnabled`
- [x] Security: heuristics must not fetch extra origins — synthetic CSS only; no FetchSub (`simplify.go`)
- [x] **CSS-faithful / site-agnostic cleanup** — remove skin-shaped cascade overrides; honor author CSS; operator flags only → [`pending-phase-items/12-css-faithful-engine.md`](pending-phase-items/12-css-faithful-engine.md) **done** 2026-08-05

### 21.5 “Paste any URL” path

- [x] Loader timeouts/size limits sufficient for large HTML — verified existing: `DefaultMaxBodySize` 100 MiB, connect 30s, response 60s/`--timeout`, redirects 10 (`internal/load/load.go:28-32`; tests `TestMaxBodySizeHTTP`, `TestSlowServerTimeout`)
- [x] CSS + image subresources: concurrency and failure isolation — verified + test: `collectSheets` skips failed links with warning; layout ignores failed images; `TestSubresourceFailureIsolation`
- [x] Progress logs useful for long pages — verified existing: `RunPDFContext` `report("Loading pages (i/n)", …)` to log unless `--quiet` (`convert.go`)
- [x] Graceful degrade when CSS huge: still emit PDF body text — verified existing: stylesheet parse/fetch failures warn and continue (`collectSheets`); body layout proceeds
- [x] Document recommended flags for URL mode — `documentation/cli.md` § URL mode & chrome strip; matrix §7.5 `--simplify-dom`

### 21.6 Acceptance tests

- [x] Vendored wiki fixture: PDF contains title string; page count within band; not empty — `TestWebWikiFixtureAcceptance`
- [x] Vendored marketing fixture: hero text + primary CTA text present — `TestWebMarketingFixtureAcceptance`
- [x] Live Ana de Armas command remains **manual** smoke; update `output/wiki-*.pdf` only when intentionally regenerating samples — documented in `samples.md`
- [x] Visual notes in `documentation/samples.md`

### 21.7 Docs

- [x] Fidelity section: arbitrary websites — evidence: `documentation/fidelity.md` § Arbitrary websites (Phase 21)
- [x] Matrix unchanged claims: still not full CSS — evidence: `documentation/compatibility-matrix.md` Phase 21 note + “Still not full CSS”; **no** new Implemented rows (docs-only)
- [x] README: “URL → decent print” under progressive goals, not MVP feature list until acceptance met — evidence: README § Progressive goals (post-MVP)

### 21.8 Closure gates

- [x] `make lint` → `go vet ./...` OK (2026-08-05)
- [x] `make test` / scoped → `go test ./internal/convert ./internal/cli -count=1` OK (includes fixture-36 + web fixtures + simplify-dom)
- [x] Acceptance criteria signed against vendored fixtures — `TestWebWikiFixtureAcceptance`, `TestWebMarketingFixtureAcceptance`
- [ ] Parent Phase 21 checked — leave open until product signs Phase 21 complete (CSS inventory residuals remain documented; no Chrome parity claim)
- [x] Next: **Phase 22** only if JS-driven pages are required; else stop at Tier 2 — Phase 22 remains optional/out until product amends

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
