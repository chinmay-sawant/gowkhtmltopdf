# Phase 68: Ignored inventory + policy amend

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 68
> **Status:** not started
> **Estimated effort:** S
> **Owner:** `internal/layout` / `internal/css` / catalog
> **Depends on:** Phase 67 (Partial program closed)
> **Unblocks:** Phase 69

---

## Overview

Lock all 247 Ignored names into phase ownership, amend permanent-ignore policy for browser-level print, and reclassify them onto the work list (goal:implement, engine_status:unsupported) so counts track real backlog.

Bar: **browser-level print** for the former Ignored set (247 names). Goldens stay structural unless amended. Flip mapping only with code + tests + matrix agreement.

Full name list is owned by phases 69-77 (see bucket table in 68.1.2).

## Goals

- Clear every owned name from the work list into Implemented, or mark `[~]` with an explicit reason
- Keep catalog counts honest after each promotion batch

## Checklist

### 68.1 inventory

- [ ] 68.1.1 Confirm count of Ignored properties is 247 in `catalog/mapping.json`. Proof: recount command output.
- [ ] 68.1.2 Paste bucket ownership table into this phase (vendor 70, scroll 41, svg 31, animation 28, mask-clip 24, modern 21, speech 19, pointer 7, 3d 4, filter 2). Proof: sums to 247.

### 68.2 policy amend

- [ ] 68.2.1 Amend `48-canonical-0.2.6-css-coverage.md` and catalog `policy.permanent_ignore` for browser-level print reopen of former Ignored names. Proof: ledger + `coverage-summary.json` policy text.
- [ ] 68.2.2 Set `goal: implement` on all 247 former Ignored rows. Proof: no `goal: ignore` among those names.
- [ ] 68.2.3 Move `engine_status` from `ignored` to `unsupported` for those 247 so the work list is visible (expected counts: implemented 174, partial 0, unsupported 644, ignored 0). Proof: `coverage-summary.json` + `property-counts.md`.

### 68.3 gates

- [ ] 68.3.1 `python3 scripts/css-catalog-map.py --check` exit 0 (update script allowlists if print-noop checks must change).
- [ ] 68.3.2 Docs-only except catalog: no `make lint`/`test` required unless script changes.


## Out of scope

JavaScript execution. Pixel-diff Chrome goldens as the default gate. New direct Go modules without sign-off. Growing `paint_flow.go` / `paint_pagination.go` past the soft cap without extracting.

## Handoff

Next is Phase 69.
