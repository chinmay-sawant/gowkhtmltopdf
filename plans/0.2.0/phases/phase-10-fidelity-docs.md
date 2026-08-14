# Phase 10 - HTML/CSS Fidelity Documentation

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** complete (2026-08-04) - `documentation/fidelity.md` + matrix audit stamp + doc links  
> **Estimated effort:** 3–7 days  
> **Depends on:** MVP matrix exists (`documentation/compatibility-matrix.md`)  
> **Unblocks:** honest marketing, Tier 1 planning, all later matrix updates  
> **Tier:** 1 (must) · **Constraint:** docs only preferred; stdlib-only product

---

## Overview

Create a **detailed, phase-wise fidelity document set** so nobody expects Wikipedia or browser parity from a report engine. Treat the compatibility matrix as the contract; this phase makes the contract navigable, evidence-backed, and tiered.

## Executive Summary

| Today | Gap (closed) |
|-------|--------------|
| Matrix is long and evidence-cited | Fidelity guide ships product tiers + claims language |
| Samples under `output/` | Linked from samples + fidelity map |
| README deferred table | Points at post-MVP roadmap + fidelity.md |
| Image-mode 5×7 | Documented as replaced by TTF AA |

---

## Phase 10 checklist

### 10.1 Fidelity guide (new doc)

- [x] Create `documentation/fidelity.md` with:
  - Product positioning: **controlled report HTML**, not a browser
  - Tier 1 / Tier 2 / Tier 3 goals in plain language
  - What “good” means for invoices vs Wikipedia vs marketing sites
  - Explicit: full WebKit parity under pure stdlib is **not a milestone**
- [x] Section **“How to read the matrix”**: Implemented / Partial / Not implemented / Ignored
- [x] Section **“How we prove fidelity”**: golden fixtures, `make samples`, visual smoke, structure tests
- [x] Section **“Failure modes”**: graceful degrade (ignored CSS, stripped script, missing font → fallback), never crash
- [x] Link from `documentation/README.md` and `documentation/overview.md`

### 10.2 Feature fidelity map (phase-wise)

- [x] Typography (bold/italic/spacing) → map in fidelity.md (shipped phases 12–13)
- [x] Image mode quality → TTF AA shipped (phase 15)
- [x] Invoice CSS (boxes/tables) → Implemented subset → phase 16 expands
- [x] Floats / flex / position / grid → No → phases 16–17 (grid deferred)
- [x] PDF images (logos/grids) → Implemented PNG/JPEG → phase 14 harden
- [x] Pagination / thead repeat → Partial → phase 18
- [x] Fonts / CJK / discovery → Latin family → phases 12, 19
- [x] HF / links edges → Mostly done, known gaps → phase 20
- [x] Arbitrary URL print → Smoke only → phase 21
- [x] JavaScript → Stripped → phase 22 (staged)
- [x] Open-web competition → Not planned → phase 23

### 10.3 Matrix honesty audit (evidence, not prose)

- [x] Audit §2.1 box model against `internal/layout/style.go` + `layout.go` (matrix rows cite paths)
- [x] Audit §2.2 display/float/position against consumers (not just parsers) - float still Not implemented
- [x] Audit §2.3 fonts - real Bold/Italic faces; fake bold only as fallback
- [x] Audit §2.5 tables (`rowspan` no, header repeat no)
- [x] Audit §4 selectors - attribute/first-last-nth/siblings **Implemented**
- [x] Audit image converter path: TTF outline AA documented
- [x] Audit JS: strip site + flag warnings listed
- [x] Fix stale tag-row for `strong`/`em`/`b`/`i` (real faces, not upright-only italic)
- [x] Record audit date and base commit in matrix header (`Last honesty audit: 2026-08-04 · 38c82fc`)

### 10.4 Fixture ↔ feature index

- [x] Samples + golden README inventory
- [x] Link `output/*.pdf` / `*.png` as visual evidence
- [x] Document Wikipedia smoke: smoke only, not pass

### 10.5 README / changelog pointers

- [x] README “Deferred / not planned” points at `plans/10-canonical-post-mvp-roadmap.md`
- [x] Keep Tier 3 rows as “not planned” with pointer to phase 23
- [x] Fidelity link under Docs tables (README + documentation/README)
- [~] Optional: CHANGELOG “Unreleased” note that post-MVP roadmap published

### 10.6 Docs-only closure gates

- [x] Fidelity guide reviewed for over-claim language (banned claims section)
- [x] Internal links resolve (fidelity ↔ matrix ↔ samples ↔ plans)
- [x] Documentation-only: **no** `make lint` / `make test` required for pure doc PR (per skill)

### 10.7 Phase complete criteria

- [x] Parent ledger Phase 10 rows checked with evidence paths
- [x] Next phase handoff: **Phase 11** remainder / **Phase 16** float lite (product choice)

---

## Evidence (closure 2026-08-04)

- `documentation/fidelity.md` (new)
- Links: `documentation/README.md`, `overview.md`, root `README.md`
- Matrix header audit stamp + `strong`/`em`/`b`/`i` honesty fix
- Deferred table intro → post-MVP roadmap

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
