# Tier 2 Pending-3 — Close phases 17–20 leftovers

> **Parent:** [`plans/10-canonical-post-mvp-roadmap.md`](../../10-canonical-post-mvp-roadmap.md)  
> **Branch:** `feature/tier-2-pending-3`  
> **Status:** done (waves 1–2 shipped 2026-08-05; optional nested HF golden fixture-36 still open)  
> **Estimated effort:** 6–12 weeks  
> **Skill:** [`skills/phase-wise-checklist/SKILLS.md`](../../../skills/phase-wise-checklist/SKILLS.md)  
> **Constraint:** stdlib layout + allowlisted `go-text/typesetting`; no CGO HarfBuzz; no browser engine

---

## Overview

Last Tier 2 pending run: finish **all** phases 17–20 leftover items. Permanent
product boundaries (browser HF, CGO HarfBuzz, Noto binary bundle) are marked
`[x]` as **documented out of scope**, not deferred.

**Wave 1 (parallel, quickest wins):** orphans/widows CSS · float↔table · `:has`
· fonts WOFF/halt — plus nested HF registry wiring in parallel.

**Wave 2:** multicol · static transforms · `@container` · flex/grid remaining ·
sticky overflow behavior.

---

## Executive Summary

| Order | Subplan | Phase | Kind | Default disposition |
|------:|---------|-------|------|---------------------|
| 1 | [nested-html-hf.md](nested-html-hf.md) | 20 | **Must** | **done** — child HF layout + registry/`MergeFontFaces`; optional fixture-36 open |
| 2 | [orphans-widows-css.md](orphans-widows-css.md) | 18 | **Must** | **done** — CSS orphans/widows + fragmentation; fixture-37 |
| 3 | [float-table-packing.md](float-table-packing.md) | 17 | **Must** | **done** — clear-below tables, float-in-td, blockify; fixture-38 |
| 4 | [multicol.md](multicol.md) | 17 | **Must** | **done** — `column-count` / width / gap / span lite; fixture-39 |
| 5 | [static-transforms.md](static-transforms.md) | 17 | **Should** | **done** — static 2D transform paint + CB/stacking; fixture-40 |
| 6 | [selectors-has-container.md](selectors-has-container.md) | 17 | **Should** | **done** — `:has()` + `@container` size lite; fixtures 41–42 |
| 7 | [flex-grid-remaining.md](flex-grid-remaining.md) | 17 | **Should** | **done** — flex min-size polish; Partial subgrid/masonry expand |
| 8 | [sticky-overflow-honesty.md](sticky-overflow-honesty.md) | 17 | **Must (amended)** | **done** — overflow sticky scrollport @ offset 0 |
| 9 | [fonts-remaining.md](fonts-remaining.md) | 19 | **Honesty / optional** | **done** — WOFF1 + halt/palt; WOFF2/Noto-bundle out by design |

---

## Recommended execution order

```text
1. nested-html-hf          (product-visible; reuses body layout)
2. orphans-widows-css      (fragmentation foundation)
3. float-table-packing     (incremental float.go / cells)
4. multicol                (needs fragmentation)
5. static-transforms       (paint + CB side effects)
6. selectors-has-container (:has first; @container after widths)
7. flex-grid-remaining     (flex polish + Partial subgrid/masonry — done)
8. sticky-overflow-honesty (overflow sticky @ offset 0 — done)
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
