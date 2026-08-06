# Reviews / Improve codebase architecture

Reserved for **architecture deepening** reviews (`improve-codebase-architecture` skill), separate from ponytail leanness.

## Relation to ponytail

| Folder | Focus | Skill |
|---|---|---|
| [`../ponytail/`](../ponytail/) | Over-engineering / dead code / YAGNI — what to **delete** | `skills/ponytail-audit`, `skills/ponytail-review` |
| `improve-codebase/` (this folder) | Module depth, seams, locality — what to **deepen** | `improve-codebase-architecture`, `codebase-design` |

Run architecture reviews **after** (or in parallel with) ponytail Phase 0–1 deletes so deepening does not solidify stub surfaces.

## Status

- [ ] Architecture review not yet run for gowkhtmltopdf.
- [x] Ponytail baseline: [`../ponytail/ponytail-ultra-2026-08-06.md`](../ponytail/ponytail-ultra-2026-08-06.md) — **5.7 / 10** leanness.

When ready, spawn explore agents for deep modules (convert pipeline, layout engine, pdf font registry) and write a phase-wise checklist here using `skills/phase-wise-checklist/SKILLS.md`.
