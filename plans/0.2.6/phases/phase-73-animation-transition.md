# Phase 73: Animation and transition

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 73
> **Status:** complete (honest: dynamic animations and transitions are unsupported for static print PDF output)
> **Estimated effort:** XL
> **Owner:** `internal/css` + `internal/layout`
> **Depends on:** Phase 72
> **Unblocks:** Phase 74
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

PDF is a static print format without a time loop or interactive frame scheduler. All 28 animation and transition properties remain **unsupported** in the catalog with honest notes.

## Checklist

- [x] 73.1.1 Ownership list locked.
- [~] 73.2.1 Dynamic time-based animation/transition properties documented as unsupported for static print output.
- [x] 73.2.2 Mapping entries kept unsupported with honest notes.
- [x] 73.2.3 All 28 names verified as unsupported in `mapping.json`.
- [x] 73.3.1 `python3 scripts/css-catalog-map.py --check`; `make test`; `make lint`. Proof: all exit 0.

## Forbidden proofs

- Tests that assert animation is ignored
- Skipping `@keyframes` while marking `animation-*` Implemented
