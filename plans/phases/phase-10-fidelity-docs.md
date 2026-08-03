# Phase 10 — HTML/CSS Fidelity Documentation

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** not started  
> **Estimated effort:** 3–7 days  
> **Depends on:** MVP matrix exists (`documentation/compatibility-matrix.md`)  
> **Unblocks:** honest marketing, Tier 1 planning, all later matrix updates  
> **Tier:** 1 (must) · **Constraint:** docs only preferred; stdlib-only product

---

## Overview

Create a **detailed, phase-wise fidelity document set** so nobody expects Wikipedia or browser parity from a report engine. Treat the compatibility matrix as the contract; this phase makes the contract navigable, evidence-backed, and tiered.

## Executive Summary

| Today | Gap |
|-------|-----|
| Matrix is long and accurate for MVP | No single “fidelity story” for Tier 1/2/3 |
| Samples under `output/` | Not linked per feature pass/fail |
| README deferred table | Points at intermediate roadmap, not this post-MVP ledger |
| Image-mode 5×7 not loud enough in matrix | Users may think PNG is screenshot-quality |

---

## Phase 10 checklist

### 10.1 Fidelity guide (new doc)

- [ ] Create `documentation/fidelity.md` (or equivalent name) with:
  - Product positioning: **controlled report HTML**, not a browser
  - Tier 1 / Tier 2 / Tier 3 goals in plain language
  - What “good” means for invoices vs Wikipedia vs marketing sites
  - Explicit: full WebKit parity under pure stdlib is **not a milestone**
- [ ] Section **“How to read the matrix”**: Implemented / Partial / Not implemented / Ignored
- [ ] Section **“How we prove fidelity”**: golden fixtures, `make samples`, visual smoke, structure tests
- [ ] Section **“Failure modes”**: graceful degrade (ignored CSS, stripped script, missing font → fallback), never crash
- [ ] Link from `documentation/README.md` and `documentation/overview.md`

### 10.2 Feature fidelity map (phase-wise)

Produce a table (in fidelity.md or matrix appendix) mapping **user-facing goals → current status → target phase**:

- [ ] Typography (bold/italic/spacing) → Partial → phases 12–13
- [ ] Image mode quality → 5×7 bitmap → phase 15
- [ ] Invoice CSS (boxes/tables) → Implemented subset → phase 16 expands
- [ ] Floats / flex / position / grid → No → phases 16–17 (grid deferred)
- [ ] PDF images (logos/grids) → Implemented PNG/JPEG → phase 14 harden
- [ ] Pagination / thead repeat → Partial → phase 18
- [ ] Fonts / CJK / discovery → Single face Latin → phases 12, 19
- [ ] HF / links edges → Mostly done, known gaps → phase 20
- [ ] Arbitrary URL print → Smoke only → phase 21
- [ ] JavaScript → Stripped → phase 22 (staged)
- [ ] Open-web competition → Not planned → phase 23

### 10.3 Matrix honesty audit (evidence, not prose)

Re-walk code and tick only with file references:

- [ ] Audit §2.1 box model against `internal/layout/style.go` + `layout.go`
- [ ] Audit §2.2 display/float/position against consumers (not just parsers)
- [ ] Audit §2.3 fonts (fake bold, no italic paint, single face)
- [ ] Audit §2.5 tables (`rowspan` no, header repeat no)
- [ ] Audit §4 selectors (sibling partial, attribute/pseudo dropped)
- [ ] Audit image converter path: document 5×7 + metric mismatch in matrix § or fidelity.md
- [ ] Audit JS: strip site + flag warnings listed
- [ ] Fix any stale “Implemented” that is only “parsed”
- [ ] Record audit date and commit/sha in matrix header when done

### 10.4 Fixture ↔ feature index

- [ ] Extend `documentation/samples.md` (or fidelity appendix) with a matrix: fixture-01…20 → features exercised
- [ ] Link `output/*.pdf` / `*.png` as visual evidence for each major feature
- [ ] Document Wikipedia smoke: `output/wiki-ana-de-armas.pdf` = **smoke only**, not pass

### 10.5 README / changelog pointers

- [ ] Update README “Deferred / not planned” to point at `plans/10-canonical-post-mvp-roadmap.md` as active ledger
- [ ] Keep Tier 3 rows as “not planned” with pointer to phase 23
- [ ] Add one-line “Fidelity” link under Docs table
- [ ] Optional: CHANGELOG “Unreleased” note that post-MVP roadmap published

### 10.6 Docs-only closure gates

- [ ] Fidelity guide reviewed for over-claim language (ban “pixel perfect”, “full CSS”, “browser replacement”)
- [ ] All internal links resolve
- [ ] Documentation-only: **no** `make lint` / `make test` required for pure doc PR (per skill)
- [ ] If any code comment/doc.go strings updated for honesty, run `make lint` + `make test` and record outcomes here:

```
# record when non-doc files touched:
# make lint → 
# make test → 
```

### 10.7 Phase complete criteria

- [ ] Parent ledger Phase 10 rows checked with evidence paths
- [ ] Next phase handoff: **Phase 11** (Library API embedders)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Existing matrix + samples | All later phases (update matrix on ship) |
| Issue epic archive under `plans/PR/issues/` | Language for Tier goals |

---

## Out of scope

- Implementing CSS/fonts/JS (later phases)
- Changing default security ACL
- Expanding allowlist without code + tests
