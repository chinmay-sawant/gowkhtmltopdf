# Phase 23 - Tier 3: Compete on the Open Web (Deferred Ledger)

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** `[~]` intentionally deferred  
> **Estimated effort:** multi-year or external engine; **not** a pure-stdlib milestone  
> **Depends on:** product amendment  
> **Unblocks:** nothing in active Tier 1/2  
> **Tier:** 3 · **Constraint conflict:** full open-web stack vs stdlib-only

---

## Overview

Tier 3 is the ambition to **compete with browsers** on arbitrary websites: full CSS (grid, modern flex, sticky, transforms), full JS/DOM, years of print CSS edge cases, screenshot-class image mode. Under **pure Go stdlib with no third-party libraries or plugins**, this is **usually the wrong goal**. People who need that stack use **Chrome headless / Playwright / wkhtmltopdf+WebKit**.

This file is the **sole deferred ledger** for Tier 3 so active phases do not silently expand into it.

## Executive Summary

| Goal | Why deferred |
|------|----------------|
| Real browser or large CSS/JS stack | Unbounded surface; stdlib has no engine |
| Full WebKit/Chrome parity | Upstream itself freezes old WebKit; we are not reimplementing it |
| “Years of edge-case print CSS” complete set | Infinite regression; track only high-value rows in Tier 1/2 |
| JS-driven SPA print | Needs Stage D from phase 22 |

---

## Phase 23 checklist (all deferred)

### 23.1 Explicit non-goals (keep checked as deferred)

- [~] Embed Chrome/Chromium/WebKit/Playwright for conversion  
  **Reason:** violates no-third-party / no-browser product rule  
  **Next gate:** formal constraint amendment + new security model  
  **Owner boundary:** product owner only

- [~] Full CSS Grid, multi-col, sticky, container queries, `:has()`, shadow DOM  
  **Reason:** layout surface unbounded  
  **Next gate:** after Phase 17 partial flex proven; still may stay deferred

- [~] Full ES + HTML5 DOM + network from script  
  **Reason:** Phase 22 Stage D  
  **Next gate:** plan amendment or external tool recommendation in docs

- [~] Pixel-identical screenshot mode vs browsers  
  **Reason:** image mode aims for “real print raster”, not Chrome

- [~] Complete print CSS edge-case corpus (paged media level 3, running elements, footnotes, …)  
  **Reason:** multi-year; pull individual items into Tier 2 only with evidence

### 23.2 What Tier 1/2 already covers instead

| User phrase | Active phase |
|-------------|--------------|
| Solid HTML template engine | 10–16 |
| Leave wkhtmltopdf for most jobs | 17–20 |
| Paste URL decent print | 21 |
| Bold/italic + CJK | 12, 19 |
| Image mode not 5×7 | 15 |
| Flex/float lite | 16–17 |
| JS research | 22 A–C |

### 23.3 If product amends constraints later

Do **not** silently edit this file into active work. Create a **new** canonical ledger (e.g. `plans/11-browser-backed-optional.md`) and mark this phase:

- [~] Superseded by `<new ledger path>` - date, link

Options to document if ever needed (not endorsements):

1. Optional build tag integrating an external converter (separate module)
2. Document “use Playwright for open web; use gowkhtmltopdf for reports”
3. Multi-year pure-Go browser project (new org/repo scale)

### 23.4 Closure

- [x] Tier 3 recorded as deferred with pointers (this file)
- [ ] No Tier 3 rows appear as active `[ ]` work in phases 10–22 without amendment
- [ ] README “Full WebKit parity not planned” remains true until amendment

---

## Dependencies

None for active development. **Do not** block Tier 1/2 on this phase.

---

## Out of scope

Everything in this file is out of scope for pure-stdlib gowkhtmltopdf until explicitly amended.
