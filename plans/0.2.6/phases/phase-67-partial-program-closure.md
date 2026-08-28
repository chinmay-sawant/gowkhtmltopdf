# Phase 67: Partial program closure

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 67
> **Status:** reopen (catalog-fake closure; honesty baseline 166/9/643/0)
> **Estimated effort:** S
> **Owner:** ledger / catalog
> **Depends on:** Phases 58, 61, 63 honesty rows closed or `[~]`
> **Unblocks:** clean handoff into Ignored reopen (68+)
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

The old “174 Implemented / 0 Partial, program closed” claim was **tag arithmetic** after mass Partial→Implemented flips. After the 2026-08-28 audit, baseline is **166 / 9 / 643 / 0**.

This phase only closes when remaining Partial honesty holes are fixed in code or explicitly deferred.

## Work order

1. Confirm open Partial honesty rows: Phase 58 (`box-shadow`, `background*`), Phase 61 (`overflow-x/y`, `visibility`, `border-collapse`), Phase 63 (`writing-mode`).
2. Either deepen them (follow those phase work orders) or mark `[~]` with owner + reason in those phase files.
3. Recount `coverage-summary.json` + `property-counts.md` from `mapping.json` (do not hard-code stale 174/0 or 421/0).
4. Spot-check: no Implemented row with empty `code_path` for names touched in 57-66.

## Checklist

- [ ] 67.1.1 Phases 58/61/63 are `complete` with honest Implemented/Partial notes, or `[~]` with pointers.
- [ ] 67.1.2 Recount matches `mapping.json` row Counter.
- [ ] 67.1.3 Matrix does not call a name Implemented while mapping says Partial (or the reverse) for the honesty set.
- [ ] 67.1.4 `python3 scripts/css-catalog-map.py --check`; `make test`; `make lint`.

## Forbidden proofs

- Pasting historical 174/0 or 421/0 as current truth
- Closing while `writing-mode` is still glyph-rotate-only and marked Implemented

## Handoff

Ignored reopen implementation starts at Phase 69 (68 inventory already complete).
