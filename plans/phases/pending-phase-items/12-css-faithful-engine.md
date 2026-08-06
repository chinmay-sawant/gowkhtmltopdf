# Pending phase items — CSS-faithful / site-agnostic engine

> **Parent:** [`README.md`](README.md) · [`../phase-21-arbitrary-websites.md`](../phase-21-arbitrary-websites.md)  
> **Branch:** `feature/pending-phase-items`  
> **Status:** **done** (2026-08-05) — cascade lies removed; operator flags only  
> **Estimated effort:** 1–2 weeks (small commits; no feature expansion)  
> **Depends on:** current print/layout work through `826527b`  
> **Constraint:** stdlib-only; **no site-specific style injection** in the default layout path

---

## Overview

Make the layout/CSS engine **website- and HTML-agnostic**: it must **honor the
cascaded styles of the page being converted**, not special-case Wikipedia,
Vector/Codex tokens, or any one skin.

Ana / Chrome PDFs remain **canaries and recipes**, never hardwired product
behavior. Operator flags (`--zoom`, `--simplify-dom`, `--simplify-dom-profile`,
`--print-link-underline`, `--use-system-fonts`, `--font-path`) are the only
intentional “policy knobs.”

### Product principle

| Do | Don't |
|----|--------|
| Parse and apply author/UA CSS (including `var()`, inheritance, print `@media`) | Override computed styles because a wiki skin “looks better” that way |
| Fall back fonts when a **named face is missing** from the registry | Rewrite `font-family: Georgia` to Liberation before trying the author’s stack order |
| Use `--zoom` / smart-shrinking for density | Bake `8pt` or `zoom 2/3` into cascade defaults |
| Opt-in chrome-strip with **generic** landmarks | Inject MediaWiki selectors into the default layout path |
| Prove behavior with generic fixtures + optional live smoke | Encode `#mw-*` / `.infobox` / `.vector-body` into production style resolution |

---

## Inventory — current non-agnostic behavior (2026-08-05 / `826527b`)

Evidence from branch audit after Liberation/CSS-var/float-margin work.

### A. Must remove or replace (cascade lies)

| ID | Location | Behavior today | Agnostic replacement |
|----|----------|----------------|----------------------|
| A1 | `internal/layout/style.go` | `<a href>` with cascaded `text-decoration: inherit` forced to `underline` | **Delete override.** Honor computed `inherit` / `none`. Link discoverability = author CSS or an **opt-in** flag (e.g. `--print-link-underline`) default **off** |
| A2 | `internal/css/css.go` `knownCSSVarDefault` | Unresolved `--font-size-medium/small` → `8pt`; line-height tokens → `1.6` | **Delete token size table.** Unresolved `var(--x)` without fallback → invalid / inherit per property. Density only via author CSS or `--zoom` |
| A3 | Makefile / cli/samples recipes | Documented as “engine density” narrative | Keep `--zoom 0.666667` as **optional smoke recipe** only; say explicitly it is operator policy, not CSS fidelity |

### B. Soften / generalize (policy OK if not CSS override)

| ID | Location | Behavior today | Agnostic replacement |
|----|----------|----------------|----------------------|
| B1 | `internal/pdf/registry.go` `fontFamilyKeys` | Named `Georgia`/`Arial`/… always map to Liberation keys | **Lookup named family first** as registered. Only expand **CSS generics** (`serif` / `sans-serif` / `monospace`) to Liberation. Missing named faces fall through the author’s comma stack, then FaceSet |
| B2 | `internal/convert/simplify.go` | Opt-in sheet includes `#mw-navigation`, `.mw-jump-link` | Split: **default `--simplify-dom`** = landmarks only (`nav`/`footer`/`aside` + ARIA roles). Optional `--simplify-dom-profile=mediawiki` (or documented extra selectors) for MW IDs/classes |
| B3 | Comments / test names | “wiki infobox”, “Vector”, “Ana” in production comments | Prefer generic wording in production code; keep wiki vocabulary in **tests/fixtures/docs** |

### C. Keep (already agnostic or correctly opt-in)

| Item | Why OK |
|------|--------|
| Float **margin-box** exclusion | CSS2 float model; wiki is motivation only |
| Table max-content / rowspan / avoid-inside | Generic table layout |
| Dropping `::before`/`::after` from host selectors | Correct selector semantics for all sites |
| Nowrap wrap beside floats | Generic inline + float |
| Link hit-box ascent/descent | Generic paint |
| Custom property inheritance + `var()` resolution | Real CSS (once A2 defaults removed) |
| `rem` from root used font-size | Real CSS |
| `--use-system-fonts` / `--font-path` | Operator font discovery |
| Vendored `testdata/web/*` + live Ana smoke | Proof, not engine policy |
| `--simplify-dom` default **off** | Reports unchanged |

---

## Executive Summary

| Step | Goal | Risk if skipped |
|------|------|-----------------|
| 1 | Remove A1 link underline override | Every site with `text-decoration: inherit` on links gets fake underlines |
| 2 | Remove A2 Codex `8pt` var defaults | Pages without skin tokens still get wiki-tuned sizes |
| 3 | Fix B1 font lookup order | Author `font-family` stacks are rewritten before missing-face fallback |
| 4 | Split B2 simplify profiles | MediaWiki IDs run for every `--simplify-dom` user |
| 5 | Docs + smoke honesty | Operators think zoom/token defaults are “correct CSS” |
| 6 | Regression fixtures | Agnostic cleanup regresses invoices or link visibility unnoticed |

---

## Checklist

### 12.1 Policy (docs first)

- [x] Record principle in `documentation/fidelity.md`: **CSS-faithful default; site names are canaries only**
- [x] Matrix / cli: clarify `--zoom` is operator scaling, not stylesheet emulation
- [x] Matrix / fonts: named faces resolve as named; generics → Liberation; no proprietary TTF requirement
- [x] Update pending-phase smoke blurb: zoom is optional recipe, not required for “correctness”

### 12.2 Remove cascade overrides (A1–A2)

- [x] Delete `<a href>` + `text-decoration: inherit` → `underline` special case in `resolveStylesCtx`
- [x] Add/adjust test: wiki-like print CSS with `text-decoration: inherit !important` → computed decoration is **not** forced underline (or only when `--print-link-underline` on)
- [x] Delete `knownCSSVarDefault` font-size/line-height hardcodes (or reduce to empty)
- [x] Keep `ResolveVar` + custom-prop inheritance; unresolved bare `var(--x)` does not invent `8pt`
- [x] Update `TestCSSVarFontSizeMediumDefault8` / related tests to expect inherit/UA size, not `8pt`

### 12.3 Font lookup honesty (B1)

- [x] `Registry.Lookup`: try exact family key first; expand generics only for `serif`/`sans-serif`/`monospace`
- [x] Optional: if exact miss, continue to next family in the CSS list (already true); do **not** rewrite `Georgia` → `liberation serif` up front
- [x] Tests: `font-family: Georgia, Liberation Serif, serif` with only Liberation on registry → Liberation Serif (second), not a forced Georgia alias
- [x] Tests: `font-family: serif` alone → Liberation Serif (generic expansion OK)

### 12.4 Simplify-dom profiles (B2)

- [x] `SimplifyChromeCSS` landmarks-only as default opt-in sheet
- [x] MediaWiki selectors moved to profile/extra constant; wired only when profile requested (flag or setting — prefer one clear CLI)
- [x] Tests: default simplify hides `nav`, does **not** require `#mw-navigation` rule for non-wiki HTML
- [x] Docs: URL recipes show `--simplify-dom` vs mediawiki profile

### 12.5 Optional discoverability flag (replaces A1 without lying)

- [x] If product still wants print link underlines when CSS says `none`/`inherit`: add **opt-in** `--print-link-underline` (default off)
- [x] When set: apply underline to `a[href]` after cascade (documented override)
- [x] Ana smoke may pass the flag in the **recipe** if desired — not in engine defaults

### 12.6 Proof / gates

- [x] `make lint` + `make test` (scoped packages green 2026-08-05)
- [x] Invoice/report fixtures unchanged in spirit (no forced wiki density)
- [x] Live Ana smoke: document flags used (`--use-system-fonts`, optional `--zoom`, optional link underline / simplify profile)
- [x] Branch audit grep: no new production matches for inventing Vector token sizes or forcing link decoration without a flag

---

## Non-goals

- Pixel parity with `chrome_ana.pdf`
- Removing Wikipedia from samples / canary PDFs
- Implementing full CSS Custom Properties Level 1 (typed OM, `@property`) — keep lite `var()` + inherited `--*`
- Bundling proprietary fonts
- Making `--simplify-dom` default on

---

## Suggested commit slices

1. docs: fidelity principle + zoom/font honesty  
2. fix(css): remove `knownCSSVarDefault` size table; fix tests  
3. fix(layout): remove link `text-decoration` force; optional flag if product wants it  
4. fix(fonts): Lookup exact name before generic/Liberation expansion  
5. fix(convert): landmarks-only simplify; mediawiki profile opt-in  
6. docs/samples: Ana recipe flags as operator policy  

Each slice leaves `make test` green.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Pending phases 1–11 print work | Real CSS surface to stay faithful to |
| This ledger | Cleaner Phase 21/23 story; fewer skin-shaped bugs |

---

## Out of scope (unless product amends)

- Auto-detecting “this is Wikipedia” to pick profiles  
- Hardcoding `.infobox` / `.vector-body` in layout  
- Changing UA initial `font-size` globally to 8pt  
