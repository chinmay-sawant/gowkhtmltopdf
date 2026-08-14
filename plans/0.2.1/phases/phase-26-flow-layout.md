# Phase 26 - Flow Layout

> **Parent:** `plans/0.2.1/24-canonical-0.2.1-roadmap.md`
> **Status:** not started
> **Estimated effort:** 2–3 weeks
> **Depends on:** Phase 25 table/float policy decision
> **Unblocks:** Phase 27 type moves; honest matrix rows for flex/grid/float

---

## Overview

Block/inline flow, one-left-plus-one-right float edges, sibling-only
margin collapse, and flex/grid *lite* are the current print subset.
This phase closes the gaps that authored templates already hit, and
records the rest as `[~]` with a next gate. It does not become a
browser layout engine.

## Executive Summary

| Area | Today | Target |
|------|-------|--------|
| Floats | One left + one right edge; BFC enclosure is real | Multi-float packing still out; document. Optional: shrink-to-fit beside a single float for non-table blocks if Phase 25 kept tables clearing |
| Margin collapse | Sibling `max()` only | Parent/first-child collapse through empty blocks **or** documented non-support |
| Flex | Auto-min + re-shrink shipped; some tests comment shrink without asserting | Assert shrink; wrap `align-content` subset stays Partial |
| Grid | Occupancy + auto-flow shipped; row height sometimes `font-size * line-height` | Content-based row height for single-span items; subgrid/masonry stay lite |
| `break-inside: avoid-column` | Must not map to page break | Keep the wiki-refs fix; add a named test if missing |

---

## Phase 26 checklist

### 26.1 Floats

- [ ] Document the one-left-plus-one-right model in `documentation/fidelity.md` / matrix §2.2
- [ ] Test: two left floats in one block — current behavior pinned (second float below or overlapping). If overlapping, treat as a bug and stack vertically
- [ ] Test: ordinary in-flow block next to a left float is **not** squeezed; line boxes **are** shortened — `internal/layout/float.go`, `layout_flow.go`
- [ ] Test: BFC root (`overflow` not visible, or `display: flow-root` if parsed) encloses floats (CSS2.1 §10.6.7) — `establishesBFC`
- [ ] `[~]` CSS2.1 multi-float “search for a place that fits” — next gate: only if a template in `testdata/golden` requires it

### 26.2 Margin collapse

- [ ] Sibling adjoining vertical margins: keep `max()`; test two `p` + `h2` with explicit margins
- [ ] Parent / first-child collapse through an empty block: implement **or** document as unsupported in deferred.md
- [ ] If implemented: test an empty `<div>` between two paragraphs does not add its own height when all three margins adjoin
- [ ] Path: `internal/layout/layout_flow.go`

### 26.3 Flex assertions

- [ ] `internal/layout/flex_test.go` shrink case: items with `flex-shrink: 1` shrink; `flex-shrink: 0` does not. Remove the `_ = aw` placeholder
- [ ] Test: anonymous flex items from direct text runs still paint
- [ ] Test: `overflow` not visible → automatic min-size 0 (Flexbox §4.5) — `flexMinMainSize`
- [ ] `[~]` Multi-pass flex intrinsic sizing / full definite cross-size cycles — not a v0.2.1 gate

### 26.4 Grid row sizing

- [ ] Single-span grid items with wrapping text contribute content height, not only `FontSize * defaultLineHeightRatio` — `internal/layout/grid.go`
- [ ] Test: a 1-column grid item with two wrapped lines is taller than one line
- [ ] Subgrid still copy-inherits parent columns and re-resolves against its content box; matrix stays Partial
- [ ] Masonry remains shortest-column packing; no CSS Grid L3 claim

### 26.5 Column vs page breaks

- [ ] `break-inside: avoid-column` does not become `page-break-inside: avoid` — `internal/layout/style_properties.go`
- [ ] Test: multi-column article with `avoid-column` (fixture-39 or a unit) does not insert extra page breaks
- [ ] Multicol remains lite; no new column-balancing algorithm unless a fixture fails

### 26.6 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Matrix / `deferred.md` rows for float, flex, grid, multicol match the code
- [ ] Parent Phase 26 row checked
- [ ] Next: Phase 27 (seam) and Phase 28 (requests) may start

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 25 table/float policy | Phase 27 (stable flow before moving types) |
| Existing BFC / flex auto-min | Phase 29 needles on fixtures 22, 25, 28, 32–35, 39 |

---

## Out of scope

- Full CSS2.1 float placement loop
- Parent-through-self margin collapse + clearance interaction (unless 26.2 implements a stated subset)
- Joint subgrid intrinsic sizing
- `writing-mode: vertical-*` (still parsed, still horizontal)
- HTML5 adoption agency / foster parenting
