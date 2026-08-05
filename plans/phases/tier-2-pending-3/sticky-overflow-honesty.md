# Tier 2 Pending-3 — Sticky overflow honesty (+ overflow scrollport at offset 0)

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md)  
> **Related:** [`../subplans-tier-2/sticky-print.md`](../subplans-tier-2/sticky-print.md)  
> **Status:** done  
> **Estimated effort:** 0.5–1 day  
> **Constraint:** PDF has no user scroll  
> **Spec:** [CSS Position 3 — sticky](https://www.w3.org/TR/css-position-3/#stickypos-insets) · [CSS Overflow 3](https://www.w3.org/TR/css-overflow-3/)

---

## Overview

Print-scoped sticky (page content box = scrollport) remains **shipped**
(`sticky.go`, fixture-31). Amendment: sticky inside
`overflow: auto|scroll|hidden|clip` uses that box as the sticky scrollport and
clamps at **scroll offset 0** (PDF has no scroll) — no page-edge continuation
clones for overflow-contained sticky.

---

## Executive Summary

| Topic | Stance |
|-------|--------|
| Print sticky | **Done** — keep (fixture-31) |
| Overflow sticky | **Done (Partial)** — scrollport = overflow box @ offset 0 |
| Nested sticky axis conflicts | Best-effort clamp only |
| Chrome scroll sticky pixel parity | Non-goal |
| Continuous-media scroll animation | Non-goal (no user scroll) |

---

## Phase 1: Honesty pass + real overflow scrollport

### 1.1 Docs

- [x] Matrix §2.2 sticky / overflow rows: print scrollport + overflow @ offset 0
- [x] `cli.md` / `library-api.md`: overflow sticky sentence
- [x] Parent phase-17 overflow row → `[x]` with pointer here
- [x] README deferred table alignment

### 1.2 Code

- [x] Parse `overflow` / `overflow-x` / `overflow-y` into `ResolvedStyle.Overflow`
- [x] `applyStickyPrint` selects nearest overflow scrollport ancestor
- [x] Overflow path: clamp at offset 0; **no** page continuation clones
- [x] Path: `sticky.go`, `style.go`; comments state PDF has no scroll
- [x] Proof: `TestStickyOverflowScrollportNoPageClone`, `TestStickyOverflowClampAtOffsetZero`

---

## Phase 2: Tests

### 2.1 Regression

- [x] Keep `TestSticky*` + fixture-31 green
- [x] Overflow sticky does not page-clone (`TestStickyOverflowScrollportNoPageClone`)

### 2.2 Explicit non-implement (product boundaries, documented)

- [x] Continuous-media scroll offset > 0 — N/A in PDF (documented)
- [x] Equating sticky with `position: fixed` page stamps — out of scope

---

## Phase 3: Closure

- [x] `make lint` → PASS (`go vet ./...`, 2026-08-05)
- [x] `go test ./internal/layout ./internal/convert -count=1` → PASS (2026-08-05)
- [x] Mark this subplan **done**
- [x] Next: fonts-remaining / Phase 21 corpus

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| sticky-print shipped | Baseline |
| Phase 21 | Real sites that misuse overflow sticky |

---

## Out of scope

- User-scroll offset > 0 sticky animation
- Equating sticky with `position: fixed` page stamps
- Full overflow clipping paint (overflow parsed for sticky scrollport only)
