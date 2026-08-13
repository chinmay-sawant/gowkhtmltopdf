# Plans

| File | Role |
|------|------|
| [00-canonical-pure-go-rewrite.md](00-canonical-pure-go-rewrite.md) | **MVP ledger** (phases 0–9) - complete for v0.1.0 |
| [10-canonical-post-mvp-roadmap.md](10-canonical-post-mvp-roadmap.md) | **Active post-MVP execution ledger** - update status here |
| [phases/](phases/) | Per-phase atomic checklists (MVP 00–09 + post-MVP 10–23) |
| [phases/subplans-tier-2/](phases/subplans-tier-2/) | Tier 2 subplans: closed pending + sticky/flex-grid/shaping/image-fontface; nested HF → v0.3.0 |
| [exploration/](exploration/) | Multi-agent analysis snapshots (read-only evidence) |
| [reviews/ponytail/](reviews/ponytail/) | Ponytail leanness audits + remediation (**5.7 → 9.5/10** weighted, phases 0–5 + partials closed) |
| [reviews/improve-codebase/](reviews/improve-codebase/) | Architecture deepening reviews (reserved) |
| [frontend-improves/](frontend-improves/) | Docs-site UI/UX ledger (2026-08-14, **7.0 / 10** baseline) |
| [PR/](PR/) | PR/issue body archives |

Workflow: [../skills/phase-wise-checklist/SKILLS.md](../skills/phase-wise-checklist/SKILLS.md)

Product framing: [../documentation/overview.md](../documentation/overview.md),
[../documentation/fidelity.md](../documentation/fidelity.md),
[../documentation/deferred.md](../documentation/deferred.md).
Performance snapshots: [../documentation/performance.md](../documentation/performance.md).

---

## Active work (post-MVP)

Canonical ledger: **[10-canonical-post-mvp-roadmap.md](10-canonical-post-mvp-roadmap.md)**

Ordered for **quick wins first**, still stdlib-only (no third-party libraries or plugins):

| Phase | Title | Tier | Status (2026-08-05) |
|------:|-------|------|---------------------|
| 10 | [HTML/CSS fidelity documentation](phases/phase-10-fidelity-docs.md) | 1 | **done** (`documentation/fidelity.md`) |
| 11 | [Library API for Go embedders](phases/phase-11-library-api-embedders.md) | 1 | **done** |
| 12 | [Typography - real bold/italic](phases/phase-12-typography-faces.md) | 1 | **done** |
| 13 | [Typography - spacing](phases/phase-13-typography-spacing.md) | 1 | **done** |
| 14 | [PDF images robust path](phases/phase-14-pdf-images.md) | 1 | **done** |
| 15 | [Image mode TTF/AA raster](phases/phase-15-image-mode-raster.md) | 1 | **done** |
| 16 | [Invoice CSS (selectors + float lite)](phases/phase-16-invoice-css.md) | 1 | **done** |
| 17 | [Broader CSS (flex / grid lite / position)](phases/phase-17-broader-css.md) | 2 | **done** (#16/#17 + doc honesty); [subplan](phases/subplans-tier-2/phase-17-pending.md) |
| 18 | [Pagination polish (thead repeat)](phases/phase-18-pagination-polish.md) | 2 | **done** (#16 + doc honesty); [subplan](phases/subplans-tier-2/phase-18-pending.md) |
| 19 | [Fonts / i18n / discovery / CJK](phases/phase-19-fonts-i18n.md) | 2 | **done** (#16/#17 + `@font-face` audit); [subplan](phases/subplans-tier-2/phase-19-pending.md) |
| 20 | [HF / links edge cases](phases/phase-20-hf-links-edges.md) | 2 | **done** (#16/#17 + HF fragment GoTo); [subplan](phases/subplans-tier-2/phase-20-pending.md) |
| 21 | [Arbitrary websites / paste-any-URL](phases/phase-21-arbitrary-websites.md) | 2→3 | next product work |
| 22 | [JavaScript (staged)](phases/phase-22-javascript.md) | 2→3 |
| 23 | [Tier 3 open-web (deferred)](phases/phase-23-tier3-deferred.md) | 3 `[~]` |

**Tier 1** (solid report engine): phases 10–16 — **closed** 2026-08-04.  
**Tier 2** (leave wkhtmltopdf for most jobs): phases 17–20 — **closed** on `feature/tier-2-pending-2` (doc honesty + `@font-face` audit + HF fragment GoTo; intentional `[~]` deferrals remain).  
**Tier 3** (compete on open web): phase 23 deferred - use Chrome/Playwright for that class of problem.

---

## MVP history (complete)

| Phase | Detail |
|------:|--------|
| 0–9 | [00-canonical-pure-go-rewrite.md](00-canonical-pure-go-rewrite.md) + [phases/phase-00](phases/phase-00-scope-foundations.md) … [phase-09](phases/phase-09-hardening-closure.md) |
