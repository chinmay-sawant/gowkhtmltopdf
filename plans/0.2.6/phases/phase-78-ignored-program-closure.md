# Phase 78: Ignored program closure

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 78
> **Status:** reopen (not honestly closed)
> **Estimated effort:** S
> **Owner:** ledger
> **Depends on:** Phases 69-77 delivered or `[~]`
> **Unblocks:** Ignored program complete
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

Close only after 69-77 have real code or explicit `[~]`. Do not close on catalog flips.

## Work order

1. Walk `ignored-inventory.json` names; each must be `implemented` with flip packet, or `unsupported`/`ignored`/`partial` with honest notes, or phase `[~]`.
2. Recount `coverage-summary.json` + `property-counts.md` from mapping Counter (no hard-coded 421/0).
3. Matrix agrees with mapping for delivered names.
4. Fixture-57: keep 571 harness; extend only when Ignored names actually ship (`78.2.3`).

## Checklist

- [ ] 78.1.1 Every inventory name settled (Implemented with packet, or honest non-Implemented status / `[~]`).
- [ ] 78.1.2 Recount = mapping Counter; update `property-counts.md`.
- [ ] 78.1.3 Matrix honesty for newly Implemented rows.
- [ ] 78.1.4 `python3 scripts/css-catalog-map.py --check`, `make test`, `make lint`, `make golden`, `make claim-scan`.
- [~] 78.2.1 Fixture-57 pre-68 **571** gallery + `VANGUARD-CSS-571-COVERAGE` exists; does not cover reopened Ignored set.
- [x] 78.2.2 Page envelope 13 + needles in `golden_test.go`.
- [ ] 78.2.3 Extend fixture after real Ignored Implementations.

## Forbidden proofs

- Closing because checklists were `[x]` while mapping still unsupported
- Using stale 421 Implemented counts
