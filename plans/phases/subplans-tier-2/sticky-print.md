# Tier 2 Subplan - Full CSS `position: sticky` (print / paged)

> **Parent:** [`plans/phases/phase-17-broader-css.md`](../phase-17-broader-css.md) (`[~] sticky deferred`)  
> **Status:** not started  
> **Estimated effort:** 2–4 weeks (print-scoped full sticky)  
> **Constraint:** stdlib-only layout; no browser scroll APIs  
> **Spec:** [CSS Positioned Layout Level 3 — sticky](https://drafts.csswg.org/css-position/#stickypos)

---

## Overview

Replace today’s sticky≡relative-offset alias with **real sticky positioning for
paged media**: each page’s content box acts as the sticky view / scrollport;
sticky boxes clamp to `top`/`right`/`bottom`/`left` insets while remaining
inside their containing block. Goal: report/table headers and section labels
that stick within a page the way authors expect from CSS sticky — not
continuous-media viewport scrolling.

## Executive Summary

| Today | Target |
|-------|--------|
| `position: sticky` parsed; `applyRelativeOffset` only | Sticky view rectangle per page; clamp paint/layout offsets |
| No page-edge sticking | Stick until containing-block edge (CSS Position 3) |
| No dedicated tests | Fixtures + unit tests for top-sticky and containing-block limit |

---

## Phase 1: Spec & model (correctness first)

### 1.1 Print semantics

- [ ] Document in plan/code comment: nearest “scrollport” = **page content box**
      (`contentH` / content width from convert geometry), not a CSS `overflow`
      scroller (PDF has no scroll)
- [ ] Containing block for sticky = same as relative (nearest block ancestor /
      existing absolute CB rules already in layout)
- [ ] Insets: non-`auto` `top`/`right`/`bottom`/`left` define sticky view
      rectangle edges; `auto` = no constraint on that side
- [ ] Path notes: `internal/layout/layout.go`, `style.go`, `paint.go`

### 1.2 Explicit non-goals (this subplan)

- [~] Continuous-media scrollport sticky inside `overflow: auto` boxes — out of
      scope for PDF; honesty: ignored / degrade to relative
- [~] Sticky in both axes with nested sticky conflicts beyond simple clamp
- [~] Pixel parity with Chrome sticky on scrollable divs

---

## Phase 2: Layout / pagination integration

### 2.1 Identify sticky boxes

- [ ] Stop treating `sticky` identically to `relative` in the “always apply
      relative offset” path without page context
- [ ] Tag boxes / ops with sticky insets + containing-block bounds at layout
- [ ] Path: `internal/layout/layout.go` (`applyRelativeOffset` call sites),
      new helper e.g. `sticky.go`

### 2.2 Per-page clamp

- [ ] After fragmentation / during `paginateOps` (or equivalent), for each page
      interval `[pageY, pageY+contentH)`, compute sticky offset so the margin
      box stays inside the sticky view rectangle **and** inside the containing
      block (CSS Position 3 algorithm lite)
- [ ] Support `top` sticky first (primary report case); then `bottom`; then
      horizontal `left`/`right` if cheap
- [ ] Interaction with existing `position:fixed` stamps: fixed remains
      page-replicated; sticky does **not** replicate like fixed — it only
      clamps within flow on pages where the CB intersects the page
- [ ] Proof sketch: unit test with known `contentH`, CB height, `top: 0`

### 2.3 Paint

- [ ] Ensure display-list Y positions reflect clamped sticky offsets
- [ ] z-index / paint order unchanged unless sticky creates stacking issues
      (document if tree order kept)

---

## Phase 3: Fixtures & tests

### 3.1 New fixtures only (do not edit existing goldens)

- [ ] `testdata/golden/fixture-NN-sticky-top.html` — sticky bar + long body;
      assert bar geometry near page top on continuation pages where CB allows
- [ ] Optional: sticky thead-like div (not table thead repeat — that is Phase 18)
- [ ] Envelope in `fixturePageBounds`
- [ ] Proof: `go test ./internal/layout -run Sticky -count=1`
- [ ] Proof: `go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-NN-sticky' -count=1`

### 3.2 Regression

- [ ] Existing fixture-26/28 position/fixed tests still green
- [ ] `make lint` → ; `make test` → ; record outcomes

---

## Phase 4: Docs honesty

### 4.1 Matrix / README

- [ ] Matrix §2.2: sticky → Partial/Implemented (print scrollport = page);
      note overflow-scroll sticky not supported
- [ ] Phase 17 checklist: flip `[~] sticky` → `[x]` when proven
- [ ] Fidelity / deferred table refresh

---

## Phase 5: Closure gates

- [ ] Spec comment + algorithm helper landed
- [ ] `top` sticky fixture green; containing-block stop proven
- [ ] `make lint` / `make test` recorded
- [ ] Parent Phase 17 sticky rows updated
- [ ] Next: flex-grid-full or shaping-gotext as product prioritizes

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 17 position lite / pagination | Sticky clamp inputs |
| Paint display list | Correct sticky paint Y |

---

## Out of scope

- Browser scroll-container sticky
- Sticky as a substitute for `<thead>` repeat (Phase 18 remains authoritative for tables)
