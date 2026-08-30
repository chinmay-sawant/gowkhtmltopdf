# Phase 67: Partial program closure

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 67
> **Status:** complete (honest baseline: 152 Implemented / 23 Partial / 643 Unsupported / 0 Ignored)
> **Estimated effort:** S
> **Owner:** ledger / catalog
> **Depends on:** Phases 58, 61, 63 honesty rows closed or `[~]`
> **Unblocks:** clean handoff into Ignored reopen (68+)
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

Partial program honestly closed. Baseline is **152 / 23 / 643 / 0** with all properties backed by real engine consumers and matching matrix descriptions.

## Checklist

- [x] 67.1.1 Phases 58/61/63 are `complete` with honest Implemented/Partial notes, or `[~]` with pointers. Proof: `phase-58-paint-finishes.md`, `phase-61-overflow-visibility-table.md`, `phase-63-writing-mode-vertical.md`.
- [x] 67.1.2 Recount matches `mapping.json` row Counter. Proof: `coverage-summary.json`, `property-counts.md`.
- [x] 67.1.3 Matrix does not call a name Implemented while mapping says Partial (or the reverse) for the honesty set. Proof: Matrix §2.1, §2.3, §2.4, §2.5 synced.
- [x] 67.1.4 `python3 scripts/css-catalog-map.py --check`; `make test`; `make lint`. Proof: all exit 0.

## Forbidden proofs

- Pasting historical 174/0 or 421/0 as current truth
- Closing while `writing-mode` is still glyph-rotate-only and marked Implemented

## Handoff

Ignored reopen implementation starts at Phase 69 (68 inventory already complete).
