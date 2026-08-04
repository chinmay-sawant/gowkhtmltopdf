# Tier 2 Pending Subplans (phases 17–20)

> **Parent:** [`plans/10-canonical-post-mvp-roadmap.md`](../../10-canonical-post-mvp-roadmap.md)  
> **Branch:** `feature/tier-2-pending-2`  
> **Status:** **executed** on this branch (doc honesty + @font-face audit + HF fragment GoTo)  
> **Skill:** [`skills/phase-wise-checklist/SKILLS.md`](../../../skills/phase-wise-checklist/SKILLS.md)

---

## Why this folder exists

Tier 2 **core** (phases 17–20) shipped on `master` via #16 / #17. These subplans
tracked post-#17 leftovers as evidence-backed checklists. Work on
`feature/tier-2-pending-2` closed the required rows; remaining `[~]` items are
intentional deferrals (sticky, full Flex/Grid, HarfBuzz, nested HF docs, etc.).

## Execution status (2026-08-05)

| Order | Subplan | Status |
|------:|---------|--------|
| 0 | [00-shared-doc-honesty.md](00-shared-doc-honesty.md) | **done** — matrix/fidelity/overview/CLI/library |
| 1 | [phase-17-pending.md](phase-17-pending.md) | **done** — docs + fixture-29 |
| 2 | [phase-18-pending.md](phase-18-pending.md) | **done** — docs + fixture-30 |
| 3 | [phase-19-pending.md](phase-19-pending.md) | **done** — `@font-face` audit + fonts.md; image-mode `[~]` |
| 4 | [phase-20-pending.md](phase-20-pending.md) | **done** — HF `#id` GoTo + copies remap |

**Quality fixtures (new files only):**

- `testdata/golden/fixture-29-float-beside-table.html` (phase 17)
- `testdata/golden/fixture-30-orphans-heuristic.html` (phase 18)

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
