# Tier 2 Subplans (`plans/phases/subplans-tier-2/`)

> **Parent:** [`plans/10-canonical-post-mvp-roadmap.md`](../../10-canonical-post-mvp-roadmap.md)  
> **Branch:** `feature/tier-2-pending-2` (deepen work closed); leftovers → `feature/tier-2-pending-3`  
> **Skill:** [`skills/phase-wise-checklist/SKILLS.md`](../../../../skills/phase-wise-checklist/SKILLS.md)

---

## A. Closed — post-#17 leftovers (executed)

| Subplan | Status |
|---------|--------|
| [00-shared-doc-honesty.md](00-shared-doc-honesty.md) | **done** |
| [phase-17-pending.md](phase-17-pending.md) | **done** (docs + fixture-29) |
| [phase-18-pending.md](phase-18-pending.md) | **done** (docs + fixture-30) |
| [phase-19-pending.md](phase-19-pending.md) | **done** (`@font-face` PDF audit) |
| [phase-20-pending.md](phase-20-pending.md) | **done** (HF fragment GoTo) |

---

## B. Deepen work (status)

| Order | Subplan | Kind | Status |
|------:|---------|------|--------|
| 1 | [image-mode-fontface.md](image-mode-fontface.md) | Image pipeline `@font-face` parity | **done** |
| 2 | [sticky-print.md](sticky-print.md) | Full print-scoped CSS sticky | **done** (fixture-31) |
| 3 | [shaping-gotext-typesetting.md](shaping-gotext-typesetting.md) | OT shaping via `go-text/typesetting` | **done** (`go.mod` + `ShapeTextFont`) |
| 4 | [flex-grid-full.md](flex-grid-full.md) | **Separate** full Flex + Grid ledger | **done** (Stage A/B + Stage C lite; true L1/L3 + Chrome parity remain `[~]`) |

### WOFF note

Adopting `go-text/typesetting` does **not** require WOFF/WOFF2. Keep TTF/OTF via
`--font-path` / local `@font-face`. Details in shaping subplan § WOFF clarification.

### Noto CJK

No subplan — **`--font-path` policy stands**; do not bundle full Noto CJK.

---

## C. Remaining phases 17–20 leftovers → pending-3

| Ledger | Status |
|--------|--------|
| [`../tier-2-pending-3/README.md`](../tier-2-pending-3/README.md) | **canonical** execution plan on `feature/tier-2-pending-3` |
| [nested-hf-v0.3.0.md](nested-hf-v0.3.0.md) | `[~]` superseded → [`../tier-2-pending-3/nested-html-hf.md`](../tier-2-pending-3/nested-html-hf.md) |

Includes: nested HTML HF (pulled forward from v0.3.0), CSS orphans/widows parse,
float↔table edges, multicol, static transforms, `:has`/`@container`, flex/grid
hard-edge honesty, sticky overflow honesty, fonts policy.

---

## Amendments

| File | Role |
|------|------|
| [`plans/amendments/2026-08-05-gotext-typesetting.md`](../../amendments/2026-08-05-gotext-typesetting.md) | Allowlist **only** `go-text/typesetting` |
| [`plans/amendments/2026-08-04-shaping-stdlib.md`](../../amendments/2026-08-04-shaping-stdlib.md) | Interim Arabic/Hangul; partly superseded |

---

## Status legend

- `[ ]` not started / not proven
- `[x]` implemented and validated
- `[~]` deferred/partial — reason + next gate required