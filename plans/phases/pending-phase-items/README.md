# Pending phase items — Chrome-print gap (open-web decent print)

> **Parent:** [`plans/phases/phase-21-arbitrary-websites.md`](../phase-21-arbitrary-websites.md) · [`plans/10-canonical-post-mvp-roadmap.md`](../../10-canonical-post-mvp-roadmap.md)  
> **Branch:** `feature/pending-phase-items`  
> **Status:** executing (Phases 1–3 done; Phase 2 partial; Phase 4 next)  
> **Estimated effort:** multi-week (quick wins first → hard/policy last)  
> **Skill:** [`skills/phase-wise-checklist/SKILLS.md`](../../../skills/phase-wise-checklist/SKILLS.md)  
> **Constraint:** stdlib layout + allowlisted `go-text/typesetting`; no CGO HarfBuzz; no browser embed  
> **Smoke (raw, no `--simplify-dom`):**  
> `./bin/gowkhtmltopdf --use-system-fonts 'https://en.wikipedia.org/wiki/Ana_de_Armas' output/wiki-ana-de-armas.pdf`  
> **Reference:** `output/chrome_ana.pdf` (Chrome print of same URL — not an acceptance golden)

---

## Overview

Close the highest-impact gaps between gowkhtmltopdf open-web/URL output and
Chrome’s print PDF **without claiming Chrome parity**. Work is ordered from
**high-impact quick wins** to **hard / policy / deferred** items. Ana de Armas
Wikipedia is the **canary**, not the only target — each phase must improve
general pages (wiki + marketing) where applicable.

Product bar remains Phase 21 **decent print** (title early, body readable,
optional chrome-strip). Full browser competition stays Phase 23.

**Execution rule:** implement phases **in order**; update this ledger after
each phase’s gates pass (`make lint` + `make test` for non-docs; smoke command
above when relevant). Do not mark `[x]` without evidence.

---

## Executive Summary

| Order | Phase | Impact | Effort | In prior plans? | Disposition |
|------:|-------|--------|--------|-----------------|-------------|
| 1 | [`01-link-pseudos.md`](01-link-pseudos.md) — `:link` / `:visited` print | Blue links (wiki Vector) | Quick | Matrix “ignored”; **no implement row** | **done** (2026-08-05) |
| 2 | [`02-openweb-css-residuals.md`](02-openweb-css-residuals.md) — Phase 17/21 CSS leftovers | Page count, infobox, density | Medium | Phase 21 §21.3; pending-3 shipped core | **partial** (attr `~=`/`*=` + wiki-like print chrome; live density open) |
| 3 | [`03-unicode-glyph-fallback.md`](03-unicode-glyph-fallback.md) — IPA / missing glyphs | Pronunciation, symbols | Quick–medium | Phase 19 fonts; WOFF2 out | **done** (2026-08-05) |
| 4 | [`04-print-media-stylesheets.md`](04-print-media-stylesheets.md) — `@media print` + sheet volume | Chrome ~10pp vs our ~32pp | Medium | Phase 21 §21.3; matrix `@media` weak | **Should** |
| 5 | [`05-url-mode-flags.md`](05-url-mode-flags.md) — URL-mode flag recipe | Usable paste-URL UX | Quick (docs) | Phase 21 §21.5 / cli.md | **Must** |
| 6 | [`06-selector-cascade-gaps.md`](06-selector-cascade-gaps.md) — skin selectors | Rules never match | Medium–hard | Selectors Partial; `:has` shipped | **Should** |
| 7 | [`07-layout-hard-edges.md`](07-layout-hard-edges.md) — flex/grid/float depth | Marketing + gadgets | Hard | flex-grid-remaining Partial | **Should / Partial** |
| 8 | [`08-svg-images.md`](08-svg-images.md) — SVG-as-`img` | Wiki logo / icons | Hard / policy | Matrix unsupported | **[x] out of scope** unless amended |
| 9 | [`09-remote-webfonts.md`](09-remote-webfonts.md) — WOFF2 / HTTPS `@font-face` | Chrome font faces | Hard + security | Phase 19 permanent non-goal | **[x] out of scope** unless amended |
| 10 | [`10-javascript.md`](10-javascript.md) — JS / hydration | SPAs | Very hard | Phase 22 | **[~] → Phase 22** |
| 11 | [`11-chrome-parity.md`](11-chrome-parity.md) — Chrome/Playwright parity | Pixel/print match | Epic | Phase 23 deferred | **[~] → Phase 23** |

---

## Recommended execution order

```text
1. :link / :visited print semantics
2. Open-web CSS residuals (float/infobox, tables, wiki density)
3. Unicode glyph fallback + URL font flags wiring
4. @media print + stylesheet application honesty
5. URL-mode recommended flags (docs; keep Ana smoke raw)
6. Selector / cascade gaps skins need
7. Layout hard edges (flex/grid/float) — Partial stop OK
8–9. SVG / remote webfonts — document out of scope (no implement unless product amends)
10–11. Point to Phase 22 / 23; do not execute here
```

---

## Smoke & proof conventions

| Artifact | Role |
|----------|------|
| `output/wiki-ana-de-armas.pdf` | Regenerated with the **raw** smoke command (no `--simplify-dom`) |
| `output/chrome_ana.pdf` | Human reference only |
| `testdata/web/wiki-like-article.html` | CI-safe wiki-like fixture |
| `testdata/web/marketing-landing.html` | CI-safe marketing fixture |
| Unit/convert tests | Prefer deterministic HTML over live Wikipedia in `make test` |

Live Wikipedia is **optional smoke** (needs network). Do not gate `make test` on it.

---

## Status legend

- `[ ]` not started / not proven
- `[x]` implemented and validated with current evidence (or permanent out of scope with reason)
- `[~]` intentionally deferred to another ledger — reason + pointer required

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 17–20 + pending-3 shipped | CSS/layout/font seams |
| Phase 21 contract + fixtures | Decent-print bar / smoke |
| This ledger | Closer open-web print; Phase 22/23 remain separate |

---

## Out of scope (unless product amends)

- Pixel parity with `chrome_ana.pdf`
- Remote WOFF2 CDN webfont fetch
- SVG-as-`img` / full SVG CSS
- CGO HarfBuzz / browser embed
- Bundling full Noto CJK in the default binary
