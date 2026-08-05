# Tier 2 Pending-3 — Close phases 17–20 leftovers

> **Parent:** [`plans/10-canonical-post-mvp-roadmap.md`](../../10-canonical-post-mvp-roadmap.md)  
> **Branch:** `feature/tier-2-pending-3`  
> **Status:** planning (execution ledger)  
> **Estimated effort:** 6–12 weeks (depends on which hard CSS edges product keeps)  
> **Skill:** [`skills/phase-wise-checklist/SKILLS.md`](../../../skills/phase-wise-checklist/SKILLS.md)  
> **Constraint:** stdlib layout + allowlisted `go-text/typesetting`; no CGO HarfBuzz; no browser engine

---

## Overview

After Tier 2 core + deepen work (sticky print, flex/grid Stage A–C lite, OT
shaping, image `@font-face`, HF fragment GoTo), a residual checklist remained
as intentional `[~]` / unfinished polish. This ledger is the **canonical
execution plan** to finish phases **17–20** leftovers — including **nested HTML
headers/footers as child documents** (pulled forward from former v0.3.0 deferral).

**Research inputs (2026-08-05):**

| Source | Role |
|--------|------|
| Web CSS research | Multicol, transforms, `:has`/`@container`, orphans/widows, flex/grid hard edges, sticky, float↔table |
| Web fonts/HF research | WOFF policy, `halt`/`palt`, Indic honesty, Noto path, wkhtmltopdf nested HF model |
| Codebase explore | Current seams: `css.go`, `layout/*`, `paint.go`, `hf.go`, `MergeFontFaces`, `ShapeTextFont` |

---

## Executive Summary

| Order | Subplan | Phase | Kind | Default disposition |
|------:|---------|-------|------|---------------------|
| 1 | [nested-html-hf.md](nested-html-hf.md) | 20 | **Must** | Implement child HF document pipeline |
| 2 | [orphans-widows-css.md](orphans-widows-css.md) | 18 | **Must** | Parse CSS props; wire into fragmentation |
| 3 | [float-table-packing.md](float-table-packing.md) | 17 | **Must** | Deterministic float↔table edge cases |
| 4 | [multicol.md](multicol.md) | 17 | **Must** | `column-count` / width / gap / span lite |
| 5 | [static-transforms.md](static-transforms.md) | 17 | **Should** | Static 2D `transform` paint + CB/stacking |
| 6 | [selectors-has-container.md](selectors-has-container.md) | 17 | **Should** | `:has()` then `@container` size lite |
| 7 | [flex-grid-remaining.md](flex-grid-remaining.md) | 17 | **Should / `[~]`** | Flex min-size polish; true subgrid/masonry L3 stay `[~]` unless amended |
| 8 | [sticky-overflow-honesty.md](sticky-overflow-honesty.md) | 17 | **Honesty + optional** | Document print sticky; overflow sticky remains non-goal |
| 9 | [fonts-remaining.md](fonts-remaining.md) | 19 | **Honesty / optional** | Confirm WOFF/Noto/halt policy; optional WOFF1 only if amended |

---

## Recommended execution order

```text
1. nested-html-hf          (product-visible; reuses body layout)
2. orphans-widows-css      (fragmentation foundation)
3. float-table-packing     (incremental float.go / cells)
4. multicol                (needs fragmentation)
5. static-transforms       (paint + CB side effects)
6. selectors-has-container (:has first; @container after widths)
7. flex-grid-remaining     (optional flex polish; keep L3 [~])
8. sticky-overflow-honesty (docs; no scrollport sticky unless amended)
9. fonts-remaining         (docs + optional WOFF1; no Brotli without amendment)
```

Closure of this ledger means: every **Must** row is `[x]` with `make lint` +
`make test` recorded; remaining items are either `[x]` or intentional `[~]`
with reason + matrix honesty — then product next is **Phase 21**.

---

## Pointer hygiene

When a row moves here from older ledgers, rewrite the source as `[~]` with an
explicit pointer to this directory (skill rule §8).

| Older source | Action |
|--------------|--------|
| `subplans-tier-2/nested-hf-v0.3.0.md` | Superseded → pointer here |
| `phase-17` / `phase-18` / `phase-19` / `phase-20` pending `[~]` rows | Point to matching subplan |
| `subplans-tier-2/README.md` §C | Remove v0.3.0 deferral; link pending-3 |

---

## Status legend

- `[ ]` not started / not proven
- `[x]` implemented and validated with current evidence
- `[~]` intentionally deferred/partial — reason + next gate required

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 17–20 core shipped | Stable seams |
| Shared doc honesty | Matrix baseline |
| This ledger | Phase 21 arbitrary websites |

---

## Out of scope (permanent unless product amends)

- Chrome layout-test pixel parity
- CGO HarfBuzz / extra Go modules beyond `go-text/typesetting` (WOFF2+Brotli needs amendment)
- Bundle full Noto CJK in the default binary
- CSS animations/transitions timelines; 3D transforms; SVG filter graphs
- CSS GCPM running elements / named pages (nested HTML HF ≠ running elements)
- Independent multi-page HF documents
- Browser-engine HF
