# Tier 2 Pending-3 — CSS `orphans` / `widows` property parsing

> **Parent:** [`plans/phases/phase-18-pagination-polish.md`](../phase-18-pagination-polish.md)  
> **Related:** [`../subplans-tier-2/phase-18-pending.md`](../subplans-tier-2/phase-18-pending.md)  
> **Status:** done  
> **Estimated effort:** 3–7 days  
> **Constraint:** stdlib-only  
> **Spec:** [CSS Fragmentation Level 3 — widows & orphans](https://www.w3.org/TR/css-break-3/#widows-orphans)

---

## Overview

Phase 18 shipped **geometric heuristics** (`paint.go` `orphansWidows`,
`keepHeadingWithNext`, fixture-30). This subplan adds author-valued CSS
`orphans` / `widows` integers and enforces Fragmentation Rule 3 against line
boxes where available, falling back to the heuristic when line counts are
unavailable.

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

- [x] Confirm zero handlers for `orphans`/`widows` under `internal/css/` + `applyRestProps` (pre-change; now handled in `applyRestProps`)
- [x] Cite `paint.go:orphansWidows` (~14–60pt straddler heuristic → `orphansWidowsHeuristic`)
- [x] Cite fixture-30 honesty: CSS props not parsed (historical); fixture-37 covers CSS props
- [x] Proof: `rg -n 'orphans|widows' internal/css internal/layout/style.go` → `Orphans`/`Widows` fields + parse

### 1.2 Spec rules to implement

- [x] `orphans` / `widows`: inherited; initial **2**; integer ≥ 1; invalid → ignore declaration
- [x] Applies to block containers that establish an IFC (have line boxes)
- [x] Class B break only if lines before ≥ orphans **and** lines after ≥ widows
- [x] Forced breaks override widow/orphan restrictions (`TestBreakBeforeAlwaysIgnoresOrphans`)
- [x] `break-inside: avoid` remains higher priority than widows (modern fragmentation text) — `avoidInside` runs before `orphansWidows`
- [x] Progress escape: if no legal break and content won't fit, may break anyway (documented in `orphansWidows` comment)

---

## Phase 2: Parse & cascade

### 2.1 Style fields

- [x] Add `Orphans`, `Widows int` on `ResolvedStyle` (default 2)
- [x] Parse in `applyRestProps` (or dedicated helper): integer ≥ 1
- [x] Inherit through cascade
- [x] Path: `internal/layout/style.go`, optionally `internal/css` if needed
- [x] Proof: unit test parse + inherit (`TestOrphansWidowsParseAndInherit`)

### 2.2 Alias hygiene

- [x] Confirm `page-break-*` → `break-*` already mapped; no change required unless gaps found
- [x] Document interaction with existing `page-break-inside: avoid`

---

## Phase 3: Fragmentation wiring

### 3.1 Line-aware path (preferred)

- [x] When fragmenting a text block, count line boxes before/after candidate break
- [x] Reject Class B breaks that violate orphans/widows
- [x] If block has fewer lines than the property → keep all lines together
- [x] Path: `paint.go` pagination helpers / text split sites
- [x] Proof: unit test with known line metrics + `orphans:4; widows:2` (`TestOrphansWidowsLineAwareKeepTogether`)

### 3.2 Heuristic fallback

- [x] Keep geometric `orphansWidows` when line boxes unavailable (`orphansWidowsHeuristic`)
- [x] When CSS props differ from initial 2, prefer line-aware path; document fallback
- [x] Optional: `TestOrphansWidowsHeuristic` under `internal/layout/` (`TestOrphansWidowsHeuristicFallback`)

### 3.3 Fixtures

- [x] New golden **or** convert test: `orphans:4; widows:2` with forced multi-page paragraph
- [x] Do **not** edit fixture-11 / fixture-23 / fixture-30 envelopes tightly
- [x] Optional: extend fixture-30 with a CSS-prop section (new file preferred: `fixture-37-orphans-css.html`)
- [x] Forced `break-before: page` mid-flow ignores orphans (assert via `break-before: always`)

---

## Phase 4: Docs & closure

### 4.1 Honesty

- [x] Matrix §2.6: Partial → note **CSS properties parsed**; heuristics remain for edge cases
- [x] `cli.md` / `library-api.md`: update “CSS properties ignored” sentence
- [x] Parent phase-18 Out of scope: remove “property parsing” or narrow to full css-break-3
- [x] Flip phase-18 `[~]` CSS orphans/widows parse row → `[x]` when proven

### 4.2 Gates

- [x] `make lint` → `go vet ./...` exit 0 (2026-08-05)
- [x] `go test ./internal/layout ./internal/convert -count=1` → both ok; `TestGoldenCorpusAllFixtures/fixture-37-orphans-css.html` PASS
- [x] `make test` → `go test ./...` exit 0 (all packages ok)
- [x] Record outcomes
- [x] Next: float-table-packing or multicol

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
