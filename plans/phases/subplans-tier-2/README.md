# Tier 2 Pending Subplans (phases 17–20)

> **Parent:** [`plans/10-canonical-post-mvp-roadmap.md`](../../10-canonical-post-mvp-roadmap.md)  
> **Branch:** `feature/tier-2-pending-2`  
> **Status:** planning ledger for post-#17 leftovers  
> **Skill:** [`skills/phase-wise-checklist/SKILLS.md`](../../../skills/phase-wise-checklist/SKILLS.md)

---

## Why this folder exists

Tier 2 **core** (phases 17–20) shipped on `master` via #16 / #17. Each parent phase file
now has a **Pending (after #17)** table. These subplans turn those leftovers into
evidence-backed, atomic checklists.

Evidence gathered 2026-08-05 via parallel code scans of layout/convert/pdf + docs.

## Recommended execution order

| Order | Subplan | Kind | Blocks honesty? |
|------:|---------|------|-----------------|
| 0 | [00-shared-doc-honesty.md](00-shared-doc-honesty.md) | Docs (matrix-first) | — |
| 1 | [phase-17-pending.md](phase-17-pending.md) | Mostly docs; optional float+table fixture | No |
| 2 | [phase-18-pending.md](phase-18-pending.md) | Docs + optional orphans fixture | No |
| 3 | [phase-19-pending.md](phase-19-pending.md) | `@font-face` audit + docs | Partial label needs audit |
| 4 | [phase-20-pending.md](phase-20-pending.md) | **Code:** HF `#id` GoTo + docs | Real leftover |

**Quality fixtures already added** on this branch (new files only):

- `testdata/golden/fixture-29-float-beside-table.html` (phase 17)
- `testdata/golden/fixture-30-orphans-heuristic.html` (phase 18)

**Conflict rule:** do **not** let phases 17–20 each rewrite
`documentation/compatibility-matrix.md` independently. Land shared Pass 0 first
(or as one coordinated PR), then per-phase code leftovers.

## Parent phase pointers

| Phase | Parent ledger | Subplan |
|------:|---------------|---------|
| 17 | [phase-17-broader-css.md](../phase-17-broader-css.md) | [phase-17-pending.md](phase-17-pending.md) |
| 18 | [phase-18-pagination-polish.md](../phase-18-pagination-polish.md) | [phase-18-pending.md](phase-18-pending.md) |
| 19 | [phase-19-fonts-i18n.md](../phase-19-fonts-i18n.md) | [phase-19-pending.md](phase-19-pending.md) |
| 20 | [phase-20-hf-links-edges.md](../phase-20-hf-links-edges.md) | [phase-20-pending.md](phase-20-pending.md) |

## Status legend

- `[ ]` not started / not proven
- `[x]` implemented and validated with current evidence
- `[~]` intentionally deferred/partial — reason + next gate required
