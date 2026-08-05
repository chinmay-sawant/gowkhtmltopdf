# Tier 2 Pending-3 — CSS `orphans` / `widows` property parsing

> **Parent:** [`plans/phases/phase-18-pagination-polish.md`](../phase-18-pagination-polish.md)  
> **Related:** [`../subplans-tier-2/phase-18-pending.md`](../subplans-tier-2/phase-18-pending.md)  
> **Status:** not started  
> **Estimated effort:** 3–7 days  
> **Constraint:** stdlib-only  
> **Spec:** [CSS Fragmentation Level 3 — widows & orphans](https://www.w3.org/TR/css-break-3/#widows-orphans)

---

## Overview

Phase 18 shipped **geometric heuristics** (`paint.go` `orphansWidows`,
`keepHeadingWithNext`, fixture-30). CSS properties `orphans` / `widows` are
still **not parsed**. This subplan adds author-valued integers and enforces
Fragmentation Rule 3 against line boxes where available, falling back to the
heuristic when line counts are unavailable.

---

## Executive Summary

| Today | Target |
|-------|--------|
| Heuristic short-block shift | Keep as fallback |
| No CSS parse | Parse + inherit `<integer ≥ 1>` (initial 2) |
| Block-height based | Prefer **line-box counts** when IFC has lines |

---

## Phase 1: Evidence baseline

### 1.1 Current behavior

- [ ] Confirm zero handlers for `orphans`/`widows` under `internal/css/` + `applyRestProps`
- [ ] Cite `paint.go:orphansWidows` (~14–60pt straddler heuristic)
- [ ] Cite fixture-30 honesty: CSS props not parsed
- [ ] Proof: `rg -n 'orphans|widows' internal/css internal/layout/style.go`

### 1.2 Spec rules to implement

- [ ] `orphans` / `widows`: inherited; initial **2**; integer ≥ 1; invalid → ignore declaration
- [ ] Applies to block containers that establish an IFC (have line boxes)
- [ ] Class B break only if lines before ≥ orphans **and** lines after ≥ widows
- [ ] Forced breaks override widow/orphan restrictions
- [ ] `break-inside: avoid` remains higher priority than widows (modern fragmentation text)
- [ ] Progress escape: if no legal break and content won't fit, may break anyway (document)

---

## Phase 2: Parse & cascade

### 2.1 Style fields

- [ ] Add `Orphans`, `Widows int` on `ResolvedStyle` (default 2)
- [ ] Parse in `applyRestProps` (or dedicated helper): integer ≥ 1
- [ ] Inherit through cascade
- [ ] Path: `internal/layout/style.go`, optionally `internal/css` if needed
- [ ] Proof: unit test parse + inherit

### 2.2 Alias hygiene

- [ ] Confirm `page-break-*` → `break-*` already mapped; no change required unless gaps found
- [ ] Document interaction with existing `page-break-inside: avoid`

---

## Phase 3: Fragmentation wiring

### 3.1 Line-aware path (preferred)

- [ ] When fragmenting a text block, count line boxes before/after candidate break
- [ ] Reject Class B breaks that violate orphans/widows
- [ ] If block has fewer lines than the property → keep all lines together
- [ ] Path: `paint.go` pagination helpers / text split sites
- [ ] Proof: unit test with known line metrics + `orphans:4; widows:2`

### 3.2 Heuristic fallback

- [ ] Keep geometric `orphansWidows` when line boxes unavailable
- [ ] When CSS props differ from initial 2, prefer line-aware path; document fallback
- [ ] Optional: `TestOrphansWidowsHeuristic` under `internal/layout/` (phase-18-pending open row)

### 3.3 Fixtures

- [ ] New golden **or** convert test: `orphans:4; widows:2` with forced multi-page paragraph
- [ ] Do **not** edit fixture-11 / fixture-23 / fixture-30 envelopes tightly
- [ ] Optional: extend fixture-30 with a CSS-prop section (new file preferred: `fixture-37-orphans-css.html`)
- [ ] Forced `break-before: page` mid-flow ignores orphans (assert)

---

## Phase 4: Docs & closure

### 4.1 Honesty

- [ ] Matrix §2.6: Partial → note **CSS properties parsed**; heuristics remain for edge cases
- [ ] `cli.md` / `library-api.md`: update “CSS properties ignored” sentence
- [ ] Parent phase-18 Out of scope: remove “property parsing” or narrow to full css-break-3
- [ ] Flip phase-18 `[~]` CSS orphans/widows parse row → `[x]` when proven

### 4.2 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Record outcomes
- [ ] Next: float-table-packing or multicol

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 5 / 18 fragmentation | Break candidate sites |
| Line box geometry in layout | Accurate counts |
| Multicol (later) | Same rules on column fragmentainers |

---

## Out of scope

- Full CSS Paged Media Level 3
- Applying widows to flex/grid items without line boxes
- Pixel-matching Chrome break positions
