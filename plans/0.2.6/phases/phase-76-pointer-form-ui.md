# Phase 76: Pointer and form UI

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 76
> **Status:** complete (honest: interactive pointer and UI properties unsupported for print PDF output)
> **Estimated effort:** M
> **Owner:** `internal/layout` / catalog policy
> **Depends on:** Phase 75
> **Unblocks:** Phase 77
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

PDF is a static print format without pointer/mouse/touch interactivity. All 7 pointer and form UI properties (`appearance`, `caret-color`, `cursor`, `pointer-events`, `resize`, `touch-action`, `user-select`) remain **unsupported** in the catalog with honest notes.

## Checklist

- [x] 76.1.1 Ownership list locked.
- [x] 76.2.1 Policy A (Permanent print ignore for interactive pointer/touch UI).
- [x] 76.2.2 Mapping and matrix notes kept honest.
- [x] 76.2.3 All 7 names verified as unsupported in `mapping.json`.
- [x] 76.3.1 `python3 scripts/css-catalog-map.py --check`; `make test`; `make lint`. Proof: all exit 0.

## Forbidden proofs

- Layout variable named `cursor` / link hit-testing comments as CSS `cursor` support
- Invented test names
