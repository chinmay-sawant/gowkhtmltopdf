# Pending — Phase 4: `@media print` + large stylesheet application

> **Parent:** [`README.md`](README.md)  
> **Status:** Partial (2026-08-05) — media matching done; live Ana page-count still high  
> **Estimated effort:** 3–10 days  
> **Prior plan coverage:** **Yes** — Phase 21 §21.3; layout already `Media: "print"`; matrix `@media` **feature queries** weak  

---

## Overview

Chrome print applies wiki print CSS aggressively. We filter `print`/`all` links
but still miss feature queries and may under-apply rules that hide chrome /
tighten type — driving page-count inflation.

---

## Phase 4 checklist

### 4.1 Audit

- [x] Log which stylesheets apply on Ana fetch (print vs screen) — temporary debug or test harness OK
- [x] List high-value `@media print` rules we skip (e.g. `(max-width)`, `print` + features)

**Audit notes (2026-08-05 Ana HTML):**
- `link media="(min-width: 500px)"` (×2) — previously **skipped** (no print/all substring); now evaluated via `MediaMatches` against A4 content width
- Inline/skin: `@media print`, `@media screen`, `@media(min-width:640px)`, `prefers-color-scheme:dark`
- Prior bug: bare `(min-width:…)` was stored as media `"all"` → always applied; `not print` wrongly classified as print via substring

### 4.2 Media matching

- [x] Improve `@media` matching for print pipeline: at least `print` and `all`; document feature-query policy (`css.MediaMatches`, `documentation/compatibility-matrix.md`)
- [x] Unknown features → **false** (consistent); size features + orientation evaluated; tests `TestMediaMatches*`
- [x] Ensure `link rel=stylesheet media="print"` still loads (`TestLinkStylesheetMediaMatches`, `TestRunPDFScreenOnlyStylesheetExcluded`)

### 4.3 Volume / degrade

- [x] Verify graceful degrade on huge CSS (already Phase 21) — soft warn ≥25000 rules in `collectSheets`
- [x] Optional: warn when rule count exceeds soft threshold (no hard fail)

### 4.4 Gates

- [x] `make lint` → pass
- [x] `make test` → pass
- [x] Ana smoke page-count note vs Phase 2 baseline
- [x] Status → Partial

**Smoke note (2026-08-05):** raw Ana + `--use-system-fonts` still **~93 pages** (same as Phase 3 baseline artifact — earlier “~32” estimate was stale). Media matching is correct; remaining density is cascade/layout (Phases 2/6/7), not mis-typed `@media`.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 2 density work | Combined page-count improvement |
| `collectSheets` / css media | |

---

## Out of scope

- Full CSS Media Queries Level 4
- Screen-only interactive chrome fidelity
