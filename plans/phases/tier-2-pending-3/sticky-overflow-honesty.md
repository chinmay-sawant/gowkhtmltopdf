# Tier 2 Pending-3 — Sticky overflow honesty (+ optional non-goal lock)

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Related:** [`../subplans-tier-2/sticky-print.md`](../subplans-tier-2/sticky-print.md)  
> **Status:** not started (docs-first)  
> **Estimated effort:** 0.5 day docs; **implementation of overflow sticky = out of scope**  
> **Constraint:** PDF has no user scroll  
> **Spec:** [CSS Position 3 — sticky](https://www.w3.org/TR/css-position-3/#stickypos-insets) · [CSS Overflow 3](https://www.w3.org/TR/css-overflow-3/)

---

## Overview

Print-scoped sticky (page content box = scrollport) is **shipped** (`sticky.go`,
fixture-31). Continuous-media sticky inside `overflow: auto|scroll` remains a
non-goal for PDF. This subplan locks honesty and optional fixture notes; it does
**not** implement nested scrollports unless product amends.

---

## Executive Summary

| Topic | Stance |
|-------|--------|
| Print sticky | **Done** — keep |
| Overflow-scroll sticky | **`[~]` non-goal** — degrade to in-flow / relative |
| Nested sticky axis conflicts | Best-effort clamp only |
| Chrome scroll sticky pixel parity | Non-goal |

---

## Phase 1: Honesty pass

### 1.1 Docs

- [ ] Matrix §2.2 sticky row: reaffirm print scrollport; overflow sticky unsupported
- [ ] `cli.md` / `library-api.md` already mention print sticky — add one overflow-unsupported sentence if missing
- [ ] Parent phase-17 / sticky-print `[~]` overflow rows — keep with pointer here
- [ ] README deferred table alignment

### 1.2 Code comment

- [ ] Ensure `sticky.go` / `applyStickyPrint` comment states overflow boxes are not scrollports in PDF
- [ ] If `overflow` is parsed later for clipping, do **not** silently enable sticky-as-scroll without amendment

---

## Phase 2: Optional tests (no new feature)

### 2.1 Regression only

- [ ] Keep `TestSticky*` + fixture-31 green
- [~] Optional: document-only HTML sample showing `overflow:auto` + sticky → treated as relative/in-flow (no golden required)

### 2.2 Explicit non-implement

- [~] Parse `overflow` + nested scrollport sticky clamp — **out of scope** for pending-3
- [~] Sticky thead-like div fixture (optional nice-to-have from sticky-print) — only if free

---

## Phase 3: Closure

- [ ] Docs-only path: skill says skip lint/test
- [ ] Mark this subplan **done** when honesty sentences land
- [ ] Next: fonts-remaining

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| sticky-print shipped | Baseline |
| Phase 21 | Real sites that misuse overflow sticky |

---

## Out of scope

- Implementing continuous-media sticky
- Equating sticky with `position: fixed` page stamps
